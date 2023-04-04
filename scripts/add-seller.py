from brownie import *
from variables import *


def main():
    me = accounts.load("dex-owner")
    trigger = interface.ITrigger2(TRIGGER_ADDRESS_MAINNET)
    sellerAddress = input("enter the seller address: ")
    tx = trigger.authenticateSeller(
        sellerAddress, {"from": me, "gas_limit": 750000})
    if tx.status == 1:
        print("success")
    else:
        print("failed")
