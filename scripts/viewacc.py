from brownie import *

def main():
  me = accounts.load(input("Enter account name: "))
  print("Account info")
  print(me.address)
  print(me.private_key)
  