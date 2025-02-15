package services

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"unsafe"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	// // public bsc endpoint. You can't stream pending tx with those.
	// bsc_testnet = "https://data-seed-prebsc-2-s1.binance.org:8545/"
	// bsc         = "https://bsc-dataseed.binance.org/"
	// // geth AWS server
	// geth_http  = "http://x.xxx.xxx.xxx:8545"
	// geth_ipc   = "/home/ubuntu/bsc/node/geth.ipc"
)

func GetCurrentClient() *ethclient.Client {

	var clientType string = os.Getenv("BSC_NODE_ADDRESS")

	// switch *ClientEntered {
	// case "bsc_testnet":
	// 	clientType = bsc_testnet
	// case "bsc":
	// 	clientType = bsc
	// case "geth_http":
	// 	clientType = geth_http
	// case "geth_ipc":
	// 	clientType = geth_ipc
	// default:
	// 	clientType = cloud_http
	// }

	client, err := ethclient.Dial(clientType)

	if err != nil {
		fmt.Println("Error connecting to client", clientType)
		log.Fatalln(err)
	} else {
		fmt.Println("Successffully connected to ", clientType)
	}

	return client
}

func InitRPCClient() *rpc.Client {
	var clientValue reflect.Value = reflect.ValueOf(GetCurrentClient()).Elem()
	fieldStruct := clientValue.FieldByName("c")
	clientPointer := reflect.NewAt(fieldStruct.Type(), unsafe.Pointer(fieldStruct.UnsafeAddr())).Elem()
	finalClient, _ := clientPointer.Interface().(*rpc.Client)
	return finalClient
}
