from brownie import *
from variables import *
from eth_abi import encode_abi

def testBuySell():
  me = accounts.load("press1")
  trigger = interface.ITrigger2(TRIGGER_ADDRESS_MAINNET)
  #trigger.approveRouter(WBNB_ADDRESS, 1000000000, {"from": me, "gas_limit": 750000})
  buyTx = trigger.swapExactETHForTokens(web3.toChecksumAddress(bunny), 1000000000, 0, {"from": me, "gas_limit": 750000})
  if buyTx.status == 1:
    print("buy successful")
    #trigger.approveRouter(bunny, 0, {"from": me, "gas_limit": 750000})
    sellTx = trigger.swapTokensForExactETH(web3.toChecksumAddress(bunny), 0, 0, {"from": me, "gas_limit": 750000})
    if sellTx.status == 1:
      print('sell successful')
    else:
      print('sell failed')
  else: 
    print("buy failed")


def testPriceImpact():
  reserve_a_initial = input("enter reserve A: ")
  amount_traded = input("enter amount_traded: ")
  amountInWithFee = float(amount_traded) * (1 - 0.0025);
  price_impact = amountInWithFee / (float(reserve_a_initial) + float(amountInWithFee));
  print(price_impact)

def main():
  testBuySell()