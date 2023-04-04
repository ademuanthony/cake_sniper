from brownie import *


def main():
    me = accounts.load("dex-owner")
    trigger = Trigger2.deploy("0x067932279861a95228725aA019615a6a92A86a3D", {"from": me})
    print(trigger.address)
