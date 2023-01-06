from brownie import *
from variables import *

def main():
  me = accounts.load("press1")
  trigger = interface.ITrigger2(TRIGGER_ADDRESS_MAINNET)
  tokenAddress = input("enter the token address\n")
  triggerBalance = interface.ERC20(tokenAddress).balanceOf(trigger)
  tx = trigger.emmergencyWithdrawTkn(tokenAddress, triggerBalance, {"from": me, "gas_limit": 750000})
  if tx.status == 1:
    print("success")
  else:
    print("failed")
