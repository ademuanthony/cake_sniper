from brownie import *
from variables import *
from eth_abi import encode_abi

def main():
  me = accounts.load("press1")
  trigger = interface.ITrigger2(TRIGGER_ADDRESS_MAINNET)
  buyTx = trigger.bake(web3.toChecksumAddress(bunny), 1000000000, 0, {"from": me, "gas_limit": 750000})
  if buyTx.status == 1:
    print("buy successful")
    sellTx = trigger.serve(web3.toChecksumAddress(bunny), 0, {"from": me, "gas_limit": 750000})
    if sellTx.status == 1:
      print('sell successful')
    else:
      print('sell failed')
  else: 
    print("buy failed")
