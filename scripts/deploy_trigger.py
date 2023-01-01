from brownie import *

def main():
  me = accounts.load("press1")
  trigger = Trigger2.deploy({"from": me})
  print(trigger.address)
  