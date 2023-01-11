package services

import (
	"context"
	"crypto/ecdsa"
	"dark_forester/global"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Seller struct {
	Idx          int
	Address      common.Address
	Pk           string
	RawPk        *ecdsa.PrivateKey
	Balance      float64
	PendingNonce uint64
}

var loadSellerMutex sync.Mutex
var loadSellersCalled bool

func loadSellers(client *ethclient.Client, ctx context.Context) {
	if loadSellersCalled {
		return
	}
	loadSellerMutex.Lock()
	defer loadSellerMutex.Unlock()
	loadSellersCalled = true
	var guard sync.Mutex
	var swarm []Seller
	data, err := ioutil.ReadFile("./global/seller_book.json")
	if err != nil {
		log.Fatalln("loadSellers: cannot load seller_book.json ", err)
	}
	err = json.Unmarshal(data, &swarm)
	if err != nil {
		log.Fatalln("loadSellers: cannot unmarshall data into swarm ", err)
	}
	for _, sel := range swarm {

		guard.Lock()
		sel.PendingNonce, err = client.PendingNonceAt(ctx, sel.Address)
		guard.Unlock()
		if err != nil {
			fmt.Printf("couldn't fetch pending nonce for sel%v: %v", sel.Idx, err)
		}
		rawPk, err := crypto.HexToECDSA(sel.Pk[2:])
		sel.RawPk = rawPk
		if err != nil {
			log.Printf("error decrypting sel%v pk: %v", sel.Idx, err)
		}
		Sellers = append(Sellers, sel)
	}
	fmt.Println("Sellers fully loaded. ", len(Sellers), " sellers")
}

var (
	bakeSelector  = []byte{0x0d, 0xf4, 0xd8, 0x36}
	serveSelector = []byte{0xd8, 0x3f, 0x2b, 0x39}
)

// prepare frontrunning tx:
func (s *sandwicher) _prepareFrontrun(nonce uint64) (*types.Transaction, *big.Int) {

	to := global.TRIGGER_ADDRESS // trigger2 on mainnet
	gasLimit := uint64(700000)
	value := big.NewInt(0)

	txGasPrice := s.tx.GasPrice()
	if txGasPrice.Cmp(global.STANDARD_GAS_PRICE) == -1 { // if victim's tx gas Price < 5 GWEI, abort.
		return nil, nil
	}
	var gasPriceFront *big.Int
	if s.BinaryResult.IsNewMarket {
		gasPriceFront = global.STANDARD_GAS_PRICE
	} else {
		gasPriceFront = big.NewInt(global.SANDWICHIN_GASPRICE_MULTIPLIER)
		gasPriceFront.Mul(gasPriceFront, txGasPrice)
		gasPriceFront.Div(gasPriceFront, big.NewInt(1000000))
	}

	// 0x6c c8 28 89
	// []byte{0x6d, 0xb7, 0xb0, 0x60}
	// sandwichInselector := []byte{0x6c, 0xc8, 0x28, 0x89}
	var dataIn []byte
	tokenOut := common.LeftPadBytes(s.swapData.Token.Bytes(), 32)
	amIn := s.BinaryResult.MaxBNBICanBuy
	if !s.BinaryResult.IsNewMarket {
		amIn.Sub(amIn, global.AMINMARGIN)
	}

	amountIn := common.LeftPadBytes(amIn.Bytes(), 32)
	worstAmountOutTkn := big.NewInt(global.SANDWICHIN_MAXSLIPPAGE)
	worstAmountOutTkn.Mul(s.BinaryResult.AmountTknIWillBuy, worstAmountOutTkn)
	worstAmountOutTkn.Div(worstAmountOutTkn, big.NewInt(100000000))
	fmt.Println("max : ", s.BinaryResult.AmountTknIWillBuy, "worst :", worstAmountOutTkn)
	amountOutMinIn := common.LeftPadBytes(worstAmountOutTkn.Bytes(), 32)
	dataIn = append(dataIn, bakeSelector...)
	dataIn = append(dataIn, tokenOut...)
	dataIn = append(dataIn, amountIn...)
	dataIn = append(dataIn, amountOutMinIn...)

	fmt.Println("Tx data: token:", s.swapData.Token, "amount in:", formatEthWeiToEther(amIn),
		"min out:", formatEthWeiToEther(s.BinaryResult.AmountTknIWillBuy))

	frontrunningTx := types.NewTransaction(nonce, to, value, gasLimit, gasPriceFront, dataIn)
	signedFrontrunningTx, err := types.SignTx(frontrunningTx, types.NewEIP155Signer(global.CHAINID), global.DARK_FORESTER_ACCOUNT.RawPk)
	if err != nil {
		fmt.Println("Problem signing the frontrunning tx: ", err)
	}
	return signedFrontrunningTx, gasPriceFront
}

// prepare backrunning tx:
func (s *sandwicher) _prepareBackrun(nonce uint64, gasPrice *big.Int, tokenAddress common.Address, frontSucceeded bool) *types.Transaction {
	to := global.TRIGGER_ADDRESS
	gasLimit := uint64(700000)
	value := big.NewInt(0)

	sellerPK := global.DARK_FORESTER_ACCOUNT.RawPk
	if frontSucceeded {
		sellerPK1, err := crypto.HexToECDSA(os.Getenv("DARK_FORESTER_SELLER_PK"))
		if err == nil {
			sellerPK = sellerPK1
			sNou, err := s.client.PendingNonceAt(context.Background(),
				common.HexToAddress(os.Getenv("DARK_FORESTER_SELLER_ADDRESS")))
			if err == nil {
				nonce = sNou
			}
		}

		gasPrice = global.STANDARD_GAS_PRICE
	}

	// 0xe7 77 48 ae
	// []byte{0xd6, 0x4f, 0x65, 0x0d}
	// sandwichOutselector := []byte{0xe7, 0x77, 0x48, 0xae}
	var dataOut []byte
	amountOutMinOut := common.LeftPadBytes(big.NewInt(0).Bytes(), 32)
	tokenOut := common.LeftPadBytes(tokenAddress.Bytes(), 32)
	dataOut = append(dataOut, serveSelector...)
	dataOut = append(dataOut, tokenOut...)
	dataOut = append(dataOut, amountOutMinOut...)
	backrunningTx := types.NewTransaction(nonce+1, to, value, gasLimit, gasPrice, dataOut)
	signedBackrunningTx, err := types.SignTx(backrunningTx, types.NewEIP155Signer(global.CHAINID),
		sellerPK)
	if err != nil {
		fmt.Println("Problem signing the backrunning tx: ", err)
	}
	return signedBackrunningTx
}

func (s *sandwicher) _prepareSellerBackrun(seller *Seller, sellGasPrice *big.Int,
	confirmedOutTx chan *SandwichResult, tokenAddress common.Address) {

	sellerNonce := seller.PendingNonce
	to := global.TRIGGER_ADDRESS
	gasLimit := uint64(700000)
	value := big.NewInt(0)

	// sandwichOutselector := []byte{0xd6, 0x4f, 0x65, 0x0d}
	// sandwichOutselector := []byte{0xe7, 0x77, 0x48, 0xae}
	var dataOut []byte

	amountOutMinOut := common.LeftPadBytes(big.NewInt(0).Bytes(), 32)

	tokenOut := common.LeftPadBytes(tokenAddress.Bytes(), 32)
	dataOut = append(dataOut, serveSelector...)
	dataOut = append(dataOut, tokenOut...)
	dataOut = append(dataOut, amountOutMinOut...)
	backrunningTx := types.NewTransaction(sellerNonce, to, value, gasLimit, sellGasPrice, dataOut)
	signedBackrunningTx, err := types.SignTx(backrunningTx, types.NewEIP155Signer(global.CHAINID), seller.RawPk)
	if err != nil {
		fmt.Println("Problem signing the backrunning tx: ", err)
	}
	go s.WaitRoom(signedBackrunningTx.Hash(), confirmedOutTx, "backrun")
	err = s.client.SendTransaction(context.Background(), signedBackrunningTx)
	if err != nil {
		log.Println("SEND BACKRUNS: problem with sending backrunning tx: ", err)
	}
	fmt.Printf("\nBACKRUN hash: %v gasPrice: %v\n", signedBackrunningTx.Hash(), sellGasPrice)
}

// prepare cancel tx:
func (s *sandwicher) _prepareCancel(nonce uint64, gasPriceFront *big.Int) *types.Transaction {
	cancelTx := types.NewTransaction(nonce, global.DARK_FORESTER_ACCOUNT.Address, big.NewInt(0), 500000, gasPriceFront.Mul(gasPriceFront, big.NewInt(2)), nil)
	signedCancelTx, err2 := types.SignTx(cancelTx, types.NewEIP155Signer(global.CHAINID), global.DARK_FORESTER_ACCOUNT.RawPk)
	if err2 != nil {
		fmt.Println("Problem signing the cancel tx: ", err2)
	}
	return signedCancelTx
}

func (s *sandwicher) WaitRoom(txHash common.Hash, statusResults chan *SandwichResult, txType string) {
	defer _handleSendOnClosedChan()
	result := s._waitForPendingState(txHash, context.Background(), txType)
	statusResults <- result
}

func (s *sandwicher) _waitForPendingState(txHash common.Hash, ctx context.Context, txType string) *SandwichResult {
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

func _handleSendOnClosedChan() {
	if err := recover(); err != nil {
		// fmt.Println("recovering from: ", err)
	}
}

func (s *sandwicher) _buildCancelAnalytics(victimHash, cancelHash common.Hash, oldBalanceTrigger,
	gasPriceCancel *big.Int, tokenAddress common.Address) {
	s._sharedAnalytics(victimHash, oldBalanceTrigger, tokenAddress)
	gasPrice := formatEthWeiToEther(gasPriceCancel) * 1000000000
	cancelResult := CancelResultStruct{
		SharedAnalyticStruct:   SharedAnalytic,
		CancelHash:             cancelHash,
		GasPriceCancel:         gasPrice,
		InitialExpectedProfits: formatEthWeiToEther(s.BinaryResult.ExpectedProfits),
	}
	_flushAnalyticFile(reflect.ValueOf(cancelResult).Interface())
}

func (s *sandwicher) _buildFrontrunAnalytics(victimHash, frontrunHash, backrunHash common.Hash,
	revertedFront, revertedBack bool, oldBalanceTrigger, gasPriceFront *big.Int, tokenAddress common.Address) {
	s._sharedAnalytics(victimHash, oldBalanceTrigger, tokenAddress)
	realisedProfits := new(big.Int)
	newBalanceTrigger := global.GetTriggerWBNBBalance()
	realisedProfits.Sub(newBalanceTrigger, oldBalanceTrigger)
	gasPrice := formatEthWeiToEther(gasPriceFront) * 1000000000
	var bnbSent float64
	if revertedFront == true {
		bnbSent = 0.0
	} else {
		bnbSent = formatEthWeiToEther(s.BinaryResult.MaxBNBICanBuy)
	}

	frontrunResult := FrontrunResultStruct{
		SharedAnalyticStruct: SharedAnalytic,
		FrontrunHash:         frontrunHash,
		BackrunHash:          backrunHash,
		RevertedFront:        revertedFront,
		RevertedBack:         revertedBack,
		BNBSent:              bnbSent,
		GasPriceFrontRun:     gasPrice,
		ExpectedProfits:      formatEthWeiToEther(s.BinaryResult.ExpectedProfits),
		RealisedProfits:      formatEthWeiToEther(realisedProfits),
	}
	_flushAnalyticFile(reflect.ValueOf(frontrunResult).Interface())

}

func (s *sandwicher) _sharedAnalytics(victimHash common.Hash, oldBalanceTrigger *big.Int, tokenAddress common.Address) {

	pairAddress, _ := global.FACTORY.GetPair(&bind.CallOpts{}, tokenAddress, global.WBNB_ADDRESS)
	SharedAnalytic.TokenName = getTokenName(tokenAddress, s.client)
	SharedAnalytic.PairAddr = pairAddress
	SharedAnalytic.TokenAddr = tokenAddress
	SharedAnalytic.VictimHash = victimHash
	SharedAnalytic.BalanceTriggerBefore = formatEthWeiToEther(oldBalanceTrigger)
	SharedAnalytic.ExecTime = time.Since(START) / time.Millisecond
	SharedAnalytic.Consolidated = false
	newBalanceTrigger := global.GetTriggerWBNBBalance()
	SharedAnalytic.BalanceTriggerAfter = formatEthWeiToEther(newBalanceTrigger)
}

func _reinitAnalytics() {
	SharedAnalytic = SharedAnalyticStruct{}
}

func _flushAnalyticFile(structToWrite interface{}) {
	// no analytics for now
	var data []map[string]interface{}
	filename := "./global/analytics.json"
	if FileExist(filename) {
		if err := ReadFile(filename, &data); err != nil {
			fmt.Println("Error is reading", filename, err)
		}
	}

	jStr, _ := json.Marshal(structToWrite)
	var newData map[string]interface{}
	json.Unmarshal(jStr, &newData)

	//normalize market for python
	if info, f := newData["infos"]; f {
		infoMap := info.(map[string]interface{})
		delete(newData, "infos")
		for key, val := range infoMap {
			newData[key] = val
		}
	}

	data = append(data, newData)

	newContent, _ := json.MarshalIndent(data, "", "\t")

	if err := ReplaceFileContent(filename, newContent); err != nil {
		fmt.Println("Error in writing", filename, err)
	}
}

func _flushNewmarket(newMarket *NewMarketContent) {
	var markets []NewMarketContent
	filename := "./global/sandwich_book_to_test.json"
	if FileExist(filename) {
		if err := ReadFile(filename, &markets); err != nil {
			fmt.Println("Error is reading", filename, err)
		}
	}

	markets = append(markets, *newMarket)

	out, err := json.MarshalIndent(markets, "", "\t")
	if err != nil {
		fmt.Println("error in encoding markets", err)
		return
	}
	if err := ReplaceFileContent(filename, out); err != nil {
		fmt.Println("Error in writing", filename, err)
	}
	out, _ = json.MarshalIndent(newMarket, "", "\t")
	fmt.Println(string(out))
}

func addNewMarket(swapData UniswapExactETHToTokenInput, name string, lp float64, whitelist bool) {
	global.IN_SANDWICH_BOOK[swapData.Token] = true
	// save to json file
	var sandwichBook = map[common.Address]global.Market{}
	filename := "./global/sandwich_book.json"
	if err := ReadFile(filename, &sandwichBook); err != nil {
		fmt.Println("error in reading book from file", filename, err)
	}

	market := global.Market{
		Tested:      true,
		Whitelisted: whitelist,
		Address:     swapData.Token,
		Name:        name,
		Liquidity:   lp,
	}

	sandwichBook[swapData.Token] = market

	out, err := json.MarshalIndent(sandwichBook, "", "\t")
	if err != nil {
		fmt.Println("error in encoding markets", err)
		return
	}
	if err := ReplaceFileContent(filename, out); err != nil {
		fmt.Println("Error in writing", filename, err)
	}
}

func showPairAddress(tokenAddress common.Address) common.Address {
	pairAddress, _ := global.FACTORY.GetPair(&bind.CallOpts{}, tokenAddress, global.WBNB_ADDRESS)
	return pairAddress
}
