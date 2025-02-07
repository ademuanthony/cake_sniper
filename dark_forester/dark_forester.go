package main

import (
	"context"
	"dark_forester/contracts/erc20"
	"dark_forester/global"
	"dark_forester/services"
	"flag"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/go-redis/redis"
	"github.com/google/uuid"
)

// Entry point of dark_forester.
// Before Anything, check /global/config to correctly parametrize the bot

var wg = sync.WaitGroup{}
var TopSnipe = make(chan *big.Int)

var workerID string

// convert WEI to ETH
func formatEthWeiToEther(etherAmount *big.Int) float64 {
	var base, exponent = big.NewInt(10), big.NewInt(18)
	denominator := base.Exp(base, exponent, nil)
	tokensSentFloat := new(big.Float).SetInt(etherAmount)
	denominatorFloat := new(big.Float).SetInt(denominator)
	final, _ := new(big.Float).Quo(tokensSentFloat, denominatorFloat).Float64()
	return final
}

// fetch ERC20 token symbol
func getTokenSymbol(tokenAddress common.Address, client *ethclient.Client) string {
	tokenIntance, _ := erc20.NewErc20(tokenAddress, client)
	sym, _ := tokenIntance.Symbol(nil)
	return sym
}

// main loop of the bot
func StreamNewTxs(client *ethclient.Client, rpcClient *rpc.Client, redisClient *redis.Client) {

	// // Go channel to pipe data from client subscription
	// newTxsChannel := make(chan common.Hash)

	// // Subscribe to receive one time events for new txs
	// _, err := rpcClient.EthSubscribe(
	// 	context.Background(), newTxsChannel, "newPendingTransactions", // no additional args
	// )

	// if err != nil {
	// 	fmt.Println("error while subscribing: ", err)
	// }
	// fmt.Println("\nSubscribed to mempool txs!\n")

	fmt.Println("\n////////////// BIG TRANSFERS //////////////////\n")
	if global.BIG_BNB_TRANSFER == true {
		fmt.Println("activated\nthreshold of interest : transfers >", global.BNB[:2], " BNB")
	} else {
		fmt.Println("not activated")
	}

	fmt.Println("\n////////////// ADDRESS MONITORING //////////////////\n")
	if global.ADDRESS_MONITOR == true {
		fmt.Println("activated\nthe following addresses are monitored : \n")
		for addy, addressData := range global.AddressesWatched {
			fmt.Println("address : ", addy, "name: ", addressData.Name)
		}
	} else {
		fmt.Println("not activated")
	}

	fmt.Println("\n////////////// SANDWICHER //////////////////\n")
	if global.Sandwicher == true {
		fmt.Println("activated\n\nmax BNB amount authorised for one sandwich : ", global.Sandwicher_maxbound, "WBNB")
		fmt.Println("minimum profit expected : ", global.Sandwicher_minprofit, "WBNB")
		fmt.Println("current WBNB balance inside TRIGGER : ", formatEthWeiToEther(global.GetTriggerWBNBBalance()), "WBNB")
		fmt.Println("TRIGGER balance at which we stop execution : ", formatEthWeiToEther(global.STOPLOSSBALANCE), "WBNB")
		fmt.Println("WARNING: be sure TRIGGER WBNB balance is > SANDWICHER MAXBOUND !!")

		activeMarkets := 0
		for _, specs := range global.SANDWICH_BOOK {
			if specs.Whitelisted == true && specs.ManuallyDisabled == false {
				// fmt.Println(specs.Name, market, specs.Liquidity)
				activeMarkets += 1
			}
		}
		fmt.Println("\nNumber of active Markets: ", activeMarkets, "\n")

		fmt.Println("\nManually disabled Markets: \n")
		for market, specs := range global.SANDWICH_BOOK {
			if specs.ManuallyDisabled == true {
				fmt.Println(specs.Name, market, specs.Liquidity)
			}
		}
		fmt.Println("\nEnnemies: \n")
		for ennemy, _ := range global.ENNEMIES {
			fmt.Println(ennemy)
		}

	} else {
		fmt.Println("not activated")
	}

	fmt.Println("\n////////////// LIQUIDITY SNIPING //////////////////\n")
	if global.Sniping == true {
		name, _ := global.Snipe.Tkn.Name(&bind.CallOpts{})
		fmt.Println("token targetted: ", global.Snipe.TokenAddress, "(", name, ")")
		fmt.Println("minimum liquidity expected : ", formatEthWeiToEther(global.Snipe.MinLiq), getTokenSymbol(global.Snipe.TokenPaired, client))
		fmt.Println("current WBNB balance inside TRIGGER : ", formatEthWeiToEther(global.GetTriggerWBNBBalance()), "WBNB")

	}
	chainID, _ := client.NetworkID(context.Background())
	signer := types.NewEIP155Signer(chainID)

	// redis sub
	// NEW_TRANSACTION

	workerID = uuid.New().String()
	redisClient.Publish("REGISTER_WORKER", workerID)
	subscriber := redisClient.Subscribe("NEW_TRANSACTION_" + workerID)

	for {
		msg, err := subscriber.ReceiveMessage()
		if err != nil {
			panic(err)
		}

		txHash := msg.Payload

		go func() {
			tx, is_pending, _ := client.TransactionByHash(context.Background(), common.HexToHash(txHash))
			// If tx is valid and still unconfirmed
			if is_pending {
				_, _ = signer.Sender(tx)
				go handleTransaction(tx, client)
			}
		}()
		// TODO: should we wait for others to pick? How many tx should this process at a go
	}

	// for transactionHash := range newTxsChannel {
	// 	// msg, err := subscriber.ReceiveMessage(ctx)
	// 	// if err != nil {
	// 	// 	panic(err)
	// 	// }
	// 	// // fmt.Println(msg.Payload)

	// 	// tx, is_pending, _ := client.TransactionByHash(context.Background(), common.HexToHash(msg.Payload))
	// 	// // If tx is valid and still unconfirmed
	// 	// if is_pending {

	// 	// 	fmt.Println("fresh tx")
	// 	// 	_, _ = signer.Sender(tx)
	// 	// 	go handleTransaction(tx, client)
	// 	// } else {
	// 	// 	fmt.Println("old tx")
	// 	// }

	// 	hashCp := transactionHash

	// 	go func() {
	// 		// Get transaction object from hash by querying the client
	// 		tx, is_pending, _ := client.TransactionByHash(context.Background(), hashCp)
	// 		// If tx is valid and still unconfirmed
	// 		if is_pending {
	// 			_, _ = signer.Sender(tx)
	// 			handleTransaction(tx, client)
	// 		} else {
	// 			fmt.Println("dead")
	// 		}

	// 	}()
	// 	// select {
	// 	// // Code block is executed when a new tx hash is piped to the channel
	// 	// case transactionHash := <-newTxsChannel:
	// 	// 	// Get transaction object from hash by querying the client
	// 	// 	tx, is_pending, _ := client.TransactionByHash(context.Background(), transactionHash)
	// 	// 	// If tx is valid and still unconfirmed
	// 	// 	if is_pending {
	// 	// 		_, _ = signer.Sender(tx)
	// 	// 		handleTransaction(tx, client)
	// 	// 	}
	// 	// }
	// }
}

func handleTransaction(tx *types.Transaction, client *ethclient.Client) {
	services.TxClassifier(tx, client, TopSnipe)
}

func main() {

	// we say <place_holder> for the defval as it is anyway filtered to geth_ipc in the switch
	ClientEntered := flag.String("client", "xxx", "Gateway to the bsc protocol. Available options:\n\t-bsc_testnet\n\t-bsc\n\t-geth_http\n\t-geth_ipc")
	flag.Parse()

	rpcClient := services.InitRPCClient(ClientEntered)
	client := services.GetCurrentClient()

	global.InitDF(client)

	// init goroutine Clogg if global.Sniping == true
	if global.Sniping == true {
		wg.Add(1)
		go func() {
			services.Clogg(client, TopSnipe)
			wg.Done()
		}()
	}

	var redisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, os.Interrupt)

	// Block until a signal is received.

	go func() {
		for range shutdownSignal {
			fmt.Println("\nShut down requested. Un-registering worker")
			redisClient.Publish("REMOVE_WORKER", workerID)
			os.Exit(0)
		}
	}()

	// Launch txpool streamer
	StreamNewTxs(client, rpcClient, redisClient)

}
