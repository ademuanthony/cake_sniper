from brownie import *
from variables import *
from eth_abi import encode_abi

def main():
  requestDate = encode_abi(['address', 'uint', 'uint'], [bunny, 1000000000, 0],)
  print(requestDate)
