package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ademuanthony/goarbitrage/services"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func main() {
	//monitorMempool()

	fmt.Println("end")

	client := services.GetCurrentClient()

	headers := make(chan *types.Header)

	sub, err := client.SubscribeNewHead(context.Background(), headers)
	if err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case err := <-sub.Err():
			log.Fatal(err)
		case header := <-headers:
			receiveTime := time.Now().UTC()
			fmt.Println(header.Hash().Hex()) // 0xbc10defa8dda384c96a17640d84de5578804945d347072e091b4e5f390ddea7f

			block, err := client.BlockByHash(context.Background(), header.Hash())
			if err != nil {
				log.Fatal(err)
			}

			blockTime := time.Unix(int64(block.Time()), 0)

			fmt.Printf("NO %d; Block time %v; receive time %v; diff %v \n-----------------\n", block.NumberU64(),
				blockTime, receiveTime, receiveTime.Sub(blockTime))

			//  biRouter, err := biswap.NewBiswap(common.HexToAddress("0x3a6d8ca21d1cf76f653a67577fa0d27453350dd8"), client)
			//  if err != nil {
			// 	panic(err)
			//  }

			//  pcsRouter, err := pancakeswap.NewPancakeswap(common.HexToAddress(""), client)
			//  if err != nil {
			// 	panic(err)
			//  }
			//  pcsRouter.GetAmountsOut()

		}
	}
}

func monitorMempool() {
	client := services.GetCurrentClient()
	rpcClient := services.InitRPCClient()
	newTxsChannel := make(chan common.Hash)
	_, err := rpcClient.EthSubscribe(context.Background(), newTxsChannel, "newPendingTransactions")
	if err != nil {
		fmt.Println("error while subscribing: ", err)
		return
	}
	fmt.Println("\nSubscribed to mempool txs!")

	for {
		select {
		// Code block is executed when a new tx hash is piped to the channel
		case transactionHash := <-newTxsChannel:
			// Get transaction object from hash by querying the client
			tx, is_pending, _ := client.TransactionByHash(context.Background(), transactionHash)
			// If tx is valid and still unconfirmed
			if is_pending {
				fmt.Println("New pending transaction", transactionHash.Hex(), tx.To())
			}
		}
	}

}
