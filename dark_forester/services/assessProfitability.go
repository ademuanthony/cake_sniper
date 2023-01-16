package services

import (
	"dark_forester/contracts/uniswap"
	"dark_forester/global"
	"math/big"
	"math/rand"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Equivalent of getAmountOut function of the PCS router. Calculates z.
func getAmountOut(amountIn, reserveIn, reserveOut *big.Int) *big.Int {
	// fmt.Printf("_getAmountOut %s %s %s \n", myMaxBuy.String(), reserveBnb.String(), reserveTkn.String())
	// out, err := global.ROUTER.GetAmountOut(&bind.CallOpts{}, myMaxBuy, reserveBnb, reserveTkn)
	// if err != nil {
	// 	panic(err)
	// }
	// return out

	var myMaxBuy9975 = new(big.Int)
	var z = new(big.Int)
	num := big.NewInt(9975)
	myMaxBuy9975.Mul(num, amountIn)
	num.Mul(myMaxBuy9975, reserveOut)

	den := big.NewInt(10000)
	den.Mul(den, reserveIn)
	den.Add(den, myMaxBuy9975)
	z.Div(num, den)
	return z
}

func getAmountOutMin(amountIn, reserveIn, reserveOut *big.Int, slippage float64) *big.Int {
	amount := getAmountOut(amountIn, reserveIn, reserveOut)

	var myMaxBuySlip = new(big.Int)
	var z = new(big.Int)
	num := big.NewInt(int64(10000 - slippage*10000))
	myMaxBuySlip.Mul(num, amount)

	den := big.NewInt(10000)
	z.Div(num, den)

	return z
}

// get reserves of a PCS pair an return it
func getReservesData(client *ethclient.Client, tokenAddress common.Address) (*big.Int, *big.Int) {
	pairAddress, _ := global.FACTORY.GetPair(&bind.CallOpts{}, tokenAddress, global.WBNB_ADDRESS)
	PAIR, _ := uniswap.NewIPancakePair(pairAddress, client)
	reservesData, _ := PAIR.GetReserves(&bind.CallOpts{})
	if reservesData.Reserve0 == nil {
		return nil, nil
	}
	var Rtkn0 = new(big.Int)
	var Rbnb0 = new(big.Int)
	token0, _ := PAIR.Token0(&bind.CallOpts{})
	if token0 == global.WBNB_ADDRESS {
		Rbnb0 = reservesData.Reserve0
		Rtkn0 = reservesData.Reserve1
	} else {
		Rbnb0 = reservesData.Reserve1
		Rtkn0 = reservesData.Reserve0
	}
	return Rtkn0, Rbnb0
}

// perform the binary search to determine optimal amount of WBNB to engage on the sandwich without
// breaking victim's slippage
func (s *sandwicher) _binarySearch(amountToTest, Rtkn0, Rbnb0, txValue, amountOutMinVictim *big.Int) {

	amountTknImBuying1 := getAmountOut(amountToTest, Rbnb0, Rtkn0)
	var Rtkn1 = new(big.Int)
	var Rbnb1 = new(big.Int)
	Rtkn1.Sub(Rtkn0, amountTknImBuying1)
	Rbnb1.Add(Rbnb0, amountToTest)
	amountTknVictimWillBuy1 := getAmountOut(txValue, Rbnb1, Rtkn1)

	// check if this amountToTest is really the best we can have
	// 1) we don't break victim's slippage with amountToTest
	if amountTknVictimWillBuy1.Cmp(amountOutMinVictim) == 1 {
		// 2) engage MAXBOUND on the sandwich if MAXBOUND doesn't break slippage
		if amountToTest.Cmp(global.MAXBOUND) == 0 {
			s.BinaryResult = &BinarySearchResult{global.MAXBOUND, amountTknImBuying1, amountTknVictimWillBuy1,
				Rtkn1, Rbnb1, big.NewInt(0), s.BinaryResult.IsNewMarket}
			return
		}
		myMaxBuy := amountToTest.Add(amountToTest, global.BASE_UNIT)
		amountTknImBuying2 := getAmountOut(myMaxBuy, Rbnb0, Rtkn0)
		var Rtkn1Test = new(big.Int)
		var Rbnb1Test = new(big.Int)
		Rtkn1Test.Sub(Rtkn0, amountTknImBuying2)
		Rbnb1Test.Add(Rbnb0, myMaxBuy)
		amountTknVictimWillBuy2 := getAmountOut(txValue, Rbnb1Test, Rtkn1Test)
		// 3) if we go 1 step further on the ladder and it breaks the slippage, that means that amountToTest is really the amount of WBNB that we can engage and milk the maximum of profits from the sandwich.
		if amountTknVictimWillBuy2.Cmp(amountOutMinVictim) == -1 {
			s.BinaryResult = &BinarySearchResult{amountToTest, amountTknImBuying1,
				amountTknVictimWillBuy1, Rtkn1, Rbnb1, big.NewInt(0), s.BinaryResult.IsNewMarket}
		}
	}
}

// test if we break victim's slippage with MNBOUND WBNB engaged
func _testMinbound(Rtkn, Rbnb, txValue, amountOutMinVictim *big.Int) int {
	amountTknImBuying := getAmountOut(global.MINBOUND, Rbnb, Rtkn)
	var Rtkn1 = new(big.Int)
	var Rbnb1 = new(big.Int)
	Rtkn1.Sub(Rtkn, amountTknImBuying)
	Rbnb1.Add(Rbnb, global.MINBOUND)
	amountTknVictimWillBuy := getAmountOut(txValue, Rbnb1, Rtkn1)
	return amountTknVictimWillBuy.Cmp(amountOutMinVictim)
}

func (s *sandwicher) getMyMaxBuyAmount2(Rtkn0, Rbnb0, txValue, amountOutMinVictim *big.Int,
	arrayOfInterest []*big.Int) {
	var wg = sync.WaitGroup{}
	// test with the minimum value we consent to engage. If we break victim's slippage
	// with our MINBOUND, we don't go further.
	if _testMinbound(Rtkn0, Rbnb0, txValue, amountOutMinVictim) == 1 {
		for _, amountToTest := range arrayOfInterest {
			wg.Add(1)
			go func() {
				s._binarySearch(amountToTest, Rtkn0, Rbnb0, txValue, amountOutMinVictim)
				wg.Done()
			}()
			wg.Wait()
		}
		return
	} else {
		s.BinaryResult = &BinarySearchResult{}
	}
}

func (s *sandwicher) assessProfitability(client *ethclient.Client, tkn_adddress common.Address, txValue,
	amountOutMinVictim, Rtkn0, Rbnb0 *big.Int) bool {

	var expectedProfit = new(big.Int)

	// arrayOfInterest := global.SANDWICHER_LADDER

	// // only purpose of this function is to complete the struct BinaryResult via a binary search performed on the sandwich
	// // ladder we initialised in the config file.
	// // If we cannot even buy 1 BNB without breaking victim slippage, BinaryResult will be nil
	// s.getMyMaxBuyAmount2(Rtkn0, Rbnb0, txValue, amountOutMinVictim, arrayOfInterest)

	slippage := slippage(txValue, amountOutMinVictim, Rbnb0, Rtkn0, int64(s.swapData.Decimals))
	priceImpact := priceImpact(txValue, Rbnb0)
	amountToTest := getMaxTradeAmountForSlippage(Rbnb0, slippage)

	profitPec := (slippage + priceImpact) - 0.5

	if amountToTest.Cmp(global.MAXBOUND) >= 0 {
		percent := rand.Intn(99-75) + 75

		numerator := new(big.Int).Mul(global.MAXBOUND, big.NewInt(int64(percent)))
		amountToTest = new(big.Int).Div(numerator, big.NewInt(100))
	}

	profit := profitPec * formatEthWeiToEther(amountToTest) / 100

	amountTknImBuying1 := getAmountOut(amountToTest, Rbnb0, Rtkn0)
	var Rtkn1 = new(big.Int)
	var Rbnb1 = new(big.Int)
	Rtkn1.Sub(Rtkn0, amountTknImBuying1)
	Rbnb1.Add(Rbnb0, amountToTest)
	amountTknVictimWillBuy1 := getAmountOut(txValue, Rbnb1, Rtkn1)

	if amountTknImBuying1.Int64() == 0 {
		amountTknImBuying1 = getAmountOut(amountToTest, Rbnb0, Rtkn0)
		if amountTknImBuying1.Int64() == 0 {
			return false
		}
	}

	s.BinaryResult = &BinarySearchResult{amountToTest, amountTknImBuying1, amountTknVictimWillBuy1,
		Rtkn1, Rbnb1, big.NewInt(0), s.BinaryResult.IsNewMarket}

	var Rtkn2 = new(big.Int)
	var Rbnb2 = new(big.Int)
	Rtkn2.Sub(s.BinaryResult.Rtkn1, s.BinaryResult.AmountTknVictimWillBuy)
	Rbnb2.Add(s.BinaryResult.Rbnb1, txValue)

	// r0 --> I buy --> r1 --> victim buy --> r2 --> i sell
	// at this point of execution, we just did r2 so the "i sell" phase remains to be done
	bnbAfterSell := getAmountOut(s.BinaryResult.AmountTknIWillBuy, Rtkn2, Rbnb2)
	expectedProfit.Sub(bnbAfterSell, s.BinaryResult.MaxBNBICanBuy)

	// for new markets, buy 1 gwei for a test
	if s.BinaryResult.IsNewMarket {
		s.BinaryResult.MaxBNBICanBuy = big.NewInt(1000000000)

		amountTknVictimWillBuy1 := getAmountOut(s.BinaryResult.MaxBNBICanBuy, s.BinaryResult.Rbnb1, s.BinaryResult.Rtkn1)
		s.BinaryResult.AmountTknIWillBuy = amountTknVictimWillBuy1
	}

	if global.EtherToWei(big.NewFloat(profit)).Cmp(global.MINPROFIT) == 1 {
		s.BinaryResult.ExpectedProfits = expectedProfit

		// if slippage <= 5 {
		// 	fmt.Printf(`%s, txValue: %f, Rbnb0: %f,
		// price imp: %f, slippage: %f,  MaxBNBICanBuy: %f; profit: %f, expectedProfit: %f`+"\n\n",
		// 		tkn_adddress.Hex(),
		// 		formatEthWeiToEther(txValue),
		// 		formatEthWeiToEther(Rbnb0),
		// 		priceImpact, slippage,
		// 		formatEthWeiToEther(amountToTest), profit, formatEthWeiToEther(expectedProfit))
		// }

		return slippage <= 5
	}

	return false
}

func reinitBinaryResult(BinaryResult *BinarySearchResult) {
	BinaryResult = &BinarySearchResult{}
}
