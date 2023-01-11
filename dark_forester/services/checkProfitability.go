package services

import (
	"dark_forester/global"
	"math/big"
)

// func (s *sandwicher) checkProfitability(client *ethclient.Client, tkn_adddress common.Address, txValue,
// 	amountOutMinVictim, Rtkn0, Rbnb0 *big.Int) bool {

// 	priceImpact := priceImpact(txValue, Rbnb0)

// 	slippage := slippage(txValue, amountOutMinVictim, Rbnb0, Rtkn0)

// 	maxTknToBuy := new(big.Int).Add(amountOutMinVictim, big.NewInt(1))
// 	maxBuyableAmount, _ := global.ROUTER.GetAmountIn(&bind.CallOpts{}, amountOutMinVictim, Rbnb0, Rtkn0)

// 	return false
// }

// getMaxTradeAmountForSlippage returns an amount whose price impact is
// less than the provided slippage for the given reserveBnb
func getMaxTradeAmountForSlippage(reserveBnb *big.Int, slippage float64) *big.Int {
	// pm = 100 * amountInPlusFee/(reserveIn + amountInPlusFee)
	// pm(reserveIn + amountInPlusFee) = amountInPlusFee * 100
	// am = ((pm * r)/(100 - pm)) - fee

	if (100 - slippage) <= 0 {
		return big.NewInt(0)
	}

	reserveEther := formatEthWeiToEther(reserveBnb)
	amountIn := ((slippage * reserveEther / (100 - slippage))) / 1.0025

	return global.EtherToWei(big.NewFloat(amountIn))
}

func slippage(txValue, amountOutMinVictim, Rbnb0, Rtkn0 *big.Int, decimals int64) float64 {
	if amountOutMinVictim.Int64() == 0 {
		return 5
	}
	currentAmountOut := _getAmountOut(txValue, Rbnb0, Rtkn0)
	// 100 * (cAm - minAm)/minAm
	cAmountFl := _formatEthWeiToEther(currentAmountOut, decimals)
	minAmountFl := _formatEthWeiToEther(amountOutMinVictim, decimals)

	slippage := 100 * (cAmountFl - minAmountFl) / minAmountFl

	return slippage
}

// https://ethereum.stackexchange.com/questions/102063/understand-price-impact-and-liquidity-in-pancakeswap
// amountInWithFee = amount_traded * (1 - fee);
// price_impact = amountInWithFee / (reserve_a_initial + amountInWithFee);
func priceImpact(txValueInBnb, reserveBnb *big.Int) float64 {

	amountInLessFee := formatEthWeiToEther(txValueInBnb) * (1 - 0.0025)
	reservePlusAmountInLessFee := formatEthWeiToEther(reserveBnb) + amountInLessFee

	return 100 * amountInLessFee / reservePlusAmountInLessFee
}
