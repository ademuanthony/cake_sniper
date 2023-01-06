from brownie import *

def main():
  me = accounts.load("press1")
  trigger = SandwichRouter.deploy({"from": me})
  print(trigger.address)
  