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

var START time.Time

type sandwicher struct {
	BinaryResult *BinarySearchResult
	tx           *types.Transaction
	client       *ethclient.Client
	swapData     UniswapExactETHToTokenInput
}

func NewSandwicher(tx *types.Transaction, client *ethclient.Client, swapData UniswapExactETHToTokenInput,
	BinaryResult *BinarySearchResult) *sandwicher {
	return &sandwicher{BinaryResult, tx, client, swapData}
}

func (s *sandwicher) Run() {
	defer _reinitAnalytics()
	defer func ()  {
		fmt.Println("++++++==========END===========++++++")
	}()
	
	START = time.Now()
	oldBalanceTrigger := global.GetTriggerWBNBBalance()
	var FirstConfirmed = make(chan *SandwichResult, 100)

	////////// SEND FRONTRUNNING TX ///////////////////

	nonce, err := s.client.PendingNonceAt(context.Background(), global.DARK_FORESTER_ACCOUNT.Address)
	if err != nil {
		fmt.Printf("couldn't fetch pending nonce for DARK_FORESTER_ACCOUNT", err)
	}
	signedFrontrunningTx, gasPriceFront := s._prepareFrontrun(nonce)
	if signedFrontrunningTx == nil {
		return
	}

	SANDWICHWATCHDOG = true
	fmt.Println("Watchdog activated")
	//we  wait for vitim tx to confirm before sending backrunning tx
	go s.WaitRoom(s.tx.Hash(), FirstConfirmed, "frontrun")
	err = s.client.SendTransaction(context.Background(), signedFrontrunningTx)
	if err != nil {
		log.Fatalln("handleWatchedAddressTx: problem with frontrunning tx : ", err)
	}
	fmt.Println("Frontrunning tx hash: ", signedFrontrunningTx.Hash())
	fmt.Println("Targetted token : ", s.swapData.Token)
	fmt.Println("Name : ", getTokenName(s.swapData.Token, s.client))
	fmt.Println("pair : ", showPairAddress(s.swapData.Token))

	select {
	case <-SomeoneTryToFuckMe:
		//try to cancel the tx
		s.emmmergencyCancel(nonce, gasPriceFront, oldBalanceTrigger, signedFrontrunningTx.Hash(),
			s.tx.Hash(), FirstConfirmed)

	case result := <-FirstConfirmed:
		if result.Status == 0 {

			fmt.Println("frontrunning tx reverted")
			s._buildFrontrunAnalytics(s.tx.Hash(), signedFrontrunningTx.Hash(), global.Nullhash, true, true, oldBalanceTrigger,
				gasPriceFront, s.swapData.Token)

		} else {
			// check target token balance on trigger to ensure that the token was bought
			tokenBalance, err := global.GetTriggerTokenBalance(s.swapData.Token)
			if err != nil {
				fmt.Println("Error in getting trigger token balance", err)
			} else {
				if tokenBalance.Int64() == 0 {
					fmt.Println("frontrunning tx failed")
					s._buildFrontrunAnalytics(s.tx.Hash(), signedFrontrunningTx.Hash(), global.Nullhash,
						true, true, oldBalanceTrigger, gasPriceFront, s.swapData.Token)
				} else {

					fmt.Println("frontrunning tx successful. Sending backrunning..")
					s.sendBackRunningTx(nonce, common.Big1.Mul(global.STANDARD_GAS_PRICE, big.NewInt(2)), oldBalanceTrigger,
						signedFrontrunningTx.Hash(), s.tx.Hash())
				}
			}
		}
	}

	SANDWICHWATCHDOG = false
	FRONTRUNNINGWATCHDOGBLOCK = false
	close(FirstConfirmed)
	select {
	case <-SomeoneTryToFuckMe:
		fmt.Println("cleaning SomeoneTryToFuckMe channel")
	default:
	}

	fmt.Println("sandwiching last line")
	return
}

func (s *sandwicher) emmmergencyCancel(nonce uint64, gasPriceFront,
	oldBalanceTrigger *big.Int, frontrunHash, victimHash common.Hash,
	FirstConfirmed chan *SandwichResult) {

	fmt.Println("launching emmergency cancel")
	signedCancelTx := s._prepareCancel(nonce, gasPriceFront)
	err := s.client.SendTransaction(context.Background(), signedCancelTx)
	if err != nil {
		log.Fatalln("handleWatchedAddressTx: problem with Cancel tx : ", err)
	}
	fmt.Println("Cancel tx hash: ", signedCancelTx.Hash())

	go s.WaitRoom(signedCancelTx.Hash(), FirstConfirmed, "cancel")

	var firstTxConfirmed common.Hash
	for result := range FirstConfirmed {
		if result.Status == 0 {
			fmt.Println(result.Hash, "reverted")
		} else if result.Status == 9 {
			fmt.Println(result.Hash, "couldn't fetch receipt")
		} else if result.Status == 1 {
			fmt.Println(result.Hash, "confirmed !")
			firstTxConfirmed = result.Hash
			break
		} else {
			fmt.Println(result.Hash, "unknow status:", result.Status)
		}
	}

	if firstTxConfirmed == signedCancelTx.Hash() {
		fmt.Println("Cancel tx confirmed successfully before frontrunning tx")
		s._buildCancelAnalytics(victimHash, signedCancelTx.Hash(), oldBalanceTrigger, signedCancelTx.GasPrice(),
			s.swapData.Token)
	} else {
		fmt.Println("Frontrunning tx confirmed before cancel tx... launching backrunning tx")
		s.sendBackRunningTx(nonce, gasPriceFront, oldBalanceTrigger, victimHash, frontrunHash)
	}
}

// we send backrunning tx only if frontruning succeeded and wasn't cancelled.
func (s *sandwicher) sendBackRunningTx(nonce uint64, gasPriceFront, oldBalanceTrigger *big.Int,
	frontrunHash, victimHash common.Hash) {
	if s.BinaryResult.IsNewMarket {
		gasPriceFront = global.STANDARD_GAS_PRICE
	}
	signedBackrunningTx := s._prepareBackrun(nonce, gasPriceFront, s.swapData.Token)
	err := s.client.SendTransaction(context.Background(), signedBackrunningTx)
	if err != nil {
		log.Fatalln("sendBackRunningTx: problem with backrunning tx : ", err)
	}
	fmt.Println("Backrunning tx hash: ", signedBackrunningTx.Hash())

	// check if backrunning tx succeeded:
	result := s._waitForPendingState(signedBackrunningTx.Hash(), context.Background(), "backrun")

	if result.Status == 0 {
		// a failed backrunning tx is worrying if front succeeded. It means the stinky tokens are locked in TRIGGER and couldn't be sold back.
		// at this point, we need to shut down dark forested and rescue the tokens manually.
		fmt.Printf("\nbackrunning tx reverted. Need to manually rescue funds:\ntoken name involved : %v\nBEP20 address:%v\n", SharedAnalytic.TokenName, SharedAnalytic.TokenAddr)
		s._buildFrontrunAnalytics(victimHash, frontrunHash, signedBackrunningTx.Hash(),
			false, true, oldBalanceTrigger, signedBackrunningTx.GasPrice(), s.swapData.Token)
		log.Fatalln()
	} else {
		// backrunning tx succeeded. Calculates realised profits
		fmt.Println("backrunning tx sucessful")
		s._buildFrontrunAnalytics(victimHash, frontrunHash, signedBackrunningTx.Hash(), false, false,
			oldBalanceTrigger, signedBackrunningTx.GasPrice(), s.swapData.Token)

		if s.BinaryResult.IsNewMarket {
			name := getTokenName(s.swapData.Token, s.client)
			addNewMarket(s.swapData, name, formatEthWeiToEther(s.BinaryResult.Rbnb1))
			fmt.Println("Whitelisted a new market")
		}
	}
}
