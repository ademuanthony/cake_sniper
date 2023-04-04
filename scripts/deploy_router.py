from brownie import *


def main():
    me = accounts.load("dex-owner")
    trigger = SandwichRouter.deploy({"from": me})
    print(trigger.address)
