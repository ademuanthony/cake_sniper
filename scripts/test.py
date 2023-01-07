from brownie import *
from variables import *
from eth_abi import encode_abi

def main():
  me = accounts.load("press1")
  trigger = interface.ITrigger2(TRIGGER_ADDRESS_MAINNET)
  requestDate = encode_abi(['address', 'uint', 'uint'], [bunny, 1000000000, 0],)
  buyTx = trigger.bake(requestDate, {"from": me, "gas_limit": 750000})
  if buyTx.status == 1:
    print("buy successful")
    requestDate = encode_abi(['address', 'uint'], [bunny, 0])
    sellTx = trigger.serve(requestDate, {"from": me, "gas_limit": 750000})
    if sellTx.status == 1:
      print('sell successful')
    else:
      print('sell failed')
  else: 
    print("buy failed")
