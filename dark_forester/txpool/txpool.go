package main

import (
	"bufio"
	"context"
	"dark_forester/global"
	"dark_forester/services"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
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
}

func removeItem(slice []string, i int) []string {
	if len(slice)-1 == i {
		return slice[:i]
	}
	return append(slice[:i], slice[i+1:]...)
}

func _flushNewmarket(newMarket *services.NewMarketContent) {
	out, _ := json.MarshalIndent(newMarket, "", "\t")
	file, err := os.OpenFile("../global/sandwich_book_to_test.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	writer := bufio.NewWriter(file)
	_, err = writer.WriteString(string(out))
	if err != nil {
		log.Fatalf("Got error while writing to a file. Err: %s", err.Error())
	}
	_, err = writer.WriteString(",\n")
	if err != nil {
		log.Fatalf("Got error while writing to a file. Err: %s", err.Error())
	}
	writer.Flush()
	fmt.Println(string(out))
}
