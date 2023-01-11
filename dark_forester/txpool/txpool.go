package main

import (
	"context"
	"dark_forester/global"
	"dark_forester/services"
	"encoding/json"
	"flag"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/go-redis/redis"
)

var workers = []string{}
var registeredWorkers = map[string]bool{}
var workerMutex sync.Mutex

func main() {
	// we say <place_holder> for the defval as it is anyway filtered to geth_ipc in the switch
	ClientEntered := flag.String("client", "xxx", "Gateway to the bsc protocol. Available options:\n\t-bsc_testnet\n\t-bsc\n\t-geth_http\n\t-geth_ipc")
	flag.Parse()

	rpcClient := services.InitRPCClient(ClientEntered)
	// Go channel to pipe data from client subscription
	newTxsChannel := make(chan common.Hash)

	// Subscribe to receive one time events for new txs
	_, err := rpcClient.EthSubscribe(
		context.Background(), newTxsChannel, "newPendingTransactions", // no additional args
	)

	if err != nil {
		fmt.Println("error while subscribing: ", err)
	}
	fmt.Println("\nSubscribed to mempool txs!\n")

	var redisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	setupWorkerRegistry(*redisClient)

	index := 0

	for transactionHash := range newTxsChannel {
		if len(workers) == 0 {
			continue
		}
		hashCp := transactionHash
		workerIndex := index % len(workers)
		index += 1
		go func() {
			redisClient.Publish("NEW_TRANSACTION_"+workers[workerIndex], hashCp.Hex())
		}()
	}
}

func setupWorkerRegistry(redisClient redis.Client) {
	registerWorker := redisClient.Subscribe("REGISTER_WORKER")
	removeWorker := redisClient.Subscribe("REMOVE_WORKER")
	addNewMarker := redisClient.Subscribe("NEW_MARKET")
	marketTestedSubscriber := redisClient.Subscribe("MARKET_TESTED")

	go func() {
		for {
			msg, err := addNewMarker.ReceiveMessage()
			if err != nil {
				panic(err)
			}
			var newMarketContent services.NewMarketContent

			if err := json.Unmarshal([]byte(msg.Payload), &newMarketContent); err != nil {
				fmt.Println("error in decoding new market data", err.Error())
				continue
			}
			if !global.NewMarketAdded[newMarketContent.Address] {
				fmt.Println("new market to test: ")
				global.NewMarketAdded[newMarketContent.Address] = true
				_flushNewmarket(&newMarketContent)
			}
		}
	}()

	go func() {
		for {
			msg, err := registerWorker.ReceiveMessage()
			if err != nil {
				panic(err)
			}
			if registeredWorkers[msg.Payload] {
				continue
			}
			workerMutex.Lock()
			workers = append(workers, msg.Payload)
			registeredWorkers[msg.Payload] = true
			workerMutex.Unlock()

			fmt.Println("New worker registered", msg.Payload)
		}
	}()

	go func() {
		for {
			msg, err := removeWorker.ReceiveMessage()
			if err != nil {
				panic(err)
			}
			workerMutex.Lock()
			workers = append(workers, msg.Payload)
			for i, w := range workers {
				if w == msg.Payload {
					workers = removeItem(workers, i)
					registeredWorkers[w] = false
					break
				}
			}
			workerMutex.Unlock()

			fmt.Println("Worker unregistered", msg.Payload)
		}
	}()

	go func() {
		for {
			func() {
				msg, err := marketTestedSubscriber.ReceiveMessage()
				if err != nil {
					fmt.Println("Error in receiving redis msg", err)
					return
				}

				var market global.Market
				if err := json.Unmarshal([]byte(msg.Payload), &market); err != nil {
					fmt.Println("Error in marshalling new market info", err)
					return
				}
				if oldMarket, f := global.SANDWICH_BOOK[market.Address]; f && oldMarket.ManuallyDisabled {
					return
				}
				err = services.SaveMarket(market)
				if err != nil {
					fmt.Println("error in saving tested market", err)
				}
			}()
		}
	}()
}

func removeItem(slice []string, i int) []string {
	if len(slice)-1 == i {
		return slice[:i]
	}
	return append(slice[:i], slice[i+1:]...)
}

func _flushNewmarket(newMarket *services.NewMarketContent) {
	var markets []services.NewMarketContent
	filename := "../global/sandwich_book_to_test.json"
	if services.FileExist(filename) {
		if err := services.ReadFile(filename, &markets); err != nil {
			fmt.Println("Error is reading", filename, err)
		}
	}

	markets = append(markets, *newMarket)

	out, err := json.MarshalIndent(markets, "", "\t")
	if err != nil {
		fmt.Println("error in encoding markets", err)
		return
	}
	if err := services.ReplaceFileContent(filename, out); err != nil {
		fmt.Println("Error in writing", filename, err)
	}
	out, _ = json.MarshalIndent(newMarket, "", "\t")
	fmt.Println(string(out))
}
