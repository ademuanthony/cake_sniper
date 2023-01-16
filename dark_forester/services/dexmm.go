package services

import (
	"context"
	"dark_forester/global"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

func LaunchDexmm(client  *ethclient.Client) {
	if len(Sellers) == 0 {
		loadSellers(client, context.Background())
	}
	d := NewDexmm(client, Sellers[0], global.TRIGGER_ADDRESS)
	d.Run(context.Background())
}

type dexmm struct {
	client  *ethclient.Client
	seller  *Seller
	trigger common.Address
}

func NewDexmm(client *ethclient.Client, seller *Seller, trigger common.Address) *dexmm {
	return &dexmm{client: client, seller: seller, trigger: trigger}
}

func (s *dexmm) Run(ctx context.Context) {
	var firstConfirmed = make(chan *SandwichResult, 100)
	go s.Buy(s.seller, global.EtherToWei(big.NewFloat(25090)), global.STANDARD_GAS_PRICE, firstConfirmed, global.DFC)

	select {
	case <- ctx.Done():
		// do house keeping
		return
	case result := <-firstConfirmed:
		if result.Status != 0 {
			fmt.Println("buy tx successful...", result.Hash.Hex())
		} else {
			fmt.Println("Buy tx failed")
		}
	}

	fmt.Println("Action completed")
}

func (s *dexmm) Buy(sender *Seller, amountIn, sellGasPrice *big.Int,
	confirmedOutTx chan *SandwichResult, tokenAddress common.Address) {

	rbnb, rtnk := getReservesData(s.client, tokenAddress)
	amountOutMin := getAmountOutMin(amountIn, rbnb, rtnk, 0.1)

	sendersNonce := sender.PendingNonce(context.Background(), s.client)
	to := s.trigger
	gasLimit := uint64(700000)
	value := big.NewInt(0)

	var dataOut []byte

	amountOutMinPadded := common.LeftPadBytes(amountOutMin.Bytes(), 32)
	amountInPadded := common.LeftPadBytes(amountIn.Bytes(), 32)

	tokenOut := common.LeftPadBytes(tokenAddress.Bytes(), 32)
	dataOut = append(dataOut, bakeSelector...)
	dataOut = append(dataOut, tokenOut...)
	dataOut = append(dataOut, amountInPadded...)
	dataOut = append(dataOut, amountOutMinPadded...)
	buyTx := types.NewTransaction(sendersNonce, to, value, gasLimit, sellGasPrice, dataOut)
	signedBuyTx, err := types.SignTx(buyTx, types.NewEIP155Signer(global.CHAINID), sender.RawPk)
	if err != nil {
		fmt.Println("Problem signing the buy tx tx: ", err)
	}
	go s.WaitRoom(signedBuyTx.Hash(), confirmedOutTx, "backrun")
	err = s.client.SendTransaction(context.Background(), signedBuyTx)
	if err != nil {
		log.Println("SEND BACKRUNS: problem with sending buy tx: ", err)
	}
	fmt.Printf("\nBACKRUN hash: %v gasPrice: %v\n", signedBuyTx.Hash(), sellGasPrice)
}

func (s *dexmm) Sell(seller *Seller, amountIn, sellGasPrice *big.Int,
	confirmedOutTx chan *SandwichResult, tokenAddress common.Address) {

	rbnb, rtnk := getReservesData(s.client, tokenAddress)
	amountOutMin := getAmountOutMin(amountIn, rtnk, rbnb, 0.1)

	sellerNonce := seller.PendingNonce(context.Background(), s.client)
	to := s.trigger
	gasLimit := uint64(700000)
	value := big.NewInt(0)

	var dataOut []byte

	amountOutMinPadded := common.LeftPadBytes(amountOutMin.Bytes(), 32)
	amountInPadded := common.LeftPadBytes(amountIn.Bytes(), 32)

	tokenOut := common.LeftPadBytes(tokenAddress.Bytes(), 32)
	dataOut = append(dataOut, serveSelector...)
	dataOut = append(dataOut, tokenOut...)
	dataOut = append(dataOut, amountInPadded...)
	dataOut = append(dataOut, amountOutMinPadded...)
	sellTx := types.NewTransaction(sellerNonce, to, value, gasLimit, sellGasPrice, dataOut)
	signedSellTx, err := types.SignTx(sellTx, types.NewEIP155Signer(global.CHAINID), seller.RawPk)
	if err != nil {
		fmt.Println("Problem signing the sell tx tx: ", err)
	}
	go s.WaitRoom(signedSellTx.Hash(), confirmedOutTx, "backrun")
	err = s.client.SendTransaction(context.Background(), signedSellTx)
	if err != nil {
		log.Println("SEND BACKRUNS: problem with sending sell tx: ", err)
	}
	fmt.Printf("\nBACKRUN hash: %v gasPrice: %v\n", signedSellTx.Hash(), sellGasPrice)
}

func (s *dexmm) WaitRoom(txHash common.Hash, statusResults chan *SandwichResult, txType string) {
	defer _handleSendOnClosedChan()
	result := s._waitForPendingState(txHash, context.Background(), txType)
	statusResults <- result
}

func (s *dexmm) _waitForPendingState(txHash common.Hash, ctx context.Context, txType string) *SandwichResult {
	isPending := true
	for isPending {
		_, pending, _ := s.client.TransactionByHash(ctx, txHash)
		isPending = pending
	}
	timeCounter := 0

	for {

		receipt, err := s.client.TransactionReceipt(context.Background(), txHash)
		if err == nil {
			return &SandwichResult{txHash, receipt.Status, txType}

		} else if timeCounter < 60 {
			timeCounter += 1
			time.Sleep(500 * time.Millisecond)
		} else {
			return nil
		}
	}
}
