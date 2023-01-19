from brownie import *
from variables import *
import random
import time
import json
import threading
import datetime

def custom_hook(args):
    # report the failure
    print(f'Thread failed: {args.exc_value}')
    print('taking a break for some seconds')
    time.sleep(2 * 60)
    print(f'{currentTime()} - restarting...')
    runMarketMakers()

threading.excepthook = custom_hook

def currentTime():
  now = datetime.datetime.now()
  return now.strftime("%Y-%m-%d %H:%M:%S")

def buy(triggerAddress, account, amountIn):
  me = account
  wbnb = interface.ERC20(WBNB_ADDRESS)
  balance = wbnb.balanceOf(triggerAddress)
  if amountIn > balance:
    amountIn = balance

  print(f'{currentTime()} - bnb bal {triggerAddress} is {balance/1e18}; amount is: {amountIn/1e18}')

  if amountIn == 0:
    print(f'{currentTime()} - not enough bnb in contract {triggerAddress}')
    return

  trigger = interface.ITrigger2(triggerAddress)

  amountOutMin = getDfcAmountOutMin(amountIn, 0.1)
  buyTx = trigger.swapExactETHForTokens(web3.toChecksumAddress(bunny), amountIn, amountOutMin, {"from": me, "gas_limit": 750000})
  if buyTx.status == 1:
    print(f"{currentTime()} - buy successful")
  else: 
    print(f"{currentTime()} - buy failed")

def sell(triggerAddress, account, sellOf):
  dfc = interface.ERC20(DFC_ADDRESS)
  balance = dfc.balanceOf(triggerAddress)
  print(f'{currentTime()} - dfc bal {triggerAddress} is {balance/1e8}')
  if balance/1e8 < 1000:
    return
  amountIn = random.randint(80, 100) * balance/100
  if sellOf:
    amountIn = balance
  me = account
  trigger = interface.ITrigger2(triggerAddress)
  sellTx = trigger.swapTokensForExactETH(web3.toChecksumAddress(bunny), amountIn, 0, {"from": me, "gas_limit": 750000})
  if sellTx.status == 1:
    print(f'{currentTime()} - sell successful')
  else:
    print(f'{currentTime()} - sell failed')

def getDfcAmountOutMin(amountIn, slippage):
  reserve = getReserve(DFC_PAIR_ADDRESS)
  return getAmountOutMin(amountIn, reserve[1], reserve[0], slippage)

def getBnbAmountOutMin(tokenIn, slippage):
  reserve = getReserve(DFC_PAIR_ADDRESS)
  return getAmountOutMin(tokenIn, reserve[0], reserve[1], slippage)

def getAmountOutMin(amountIn, reserveIn, reserveOut, slippage):
  router = interface.IPancakeRouter(CAKE_ROUTER_ADDRESS)
  amountOut = router.getAmountOut(amountIn, reserveIn, reserveOut)
  return amountOut - amountOut*slippage/100

def dfcPriceInBnb():
  amountOut = getBnbAmountOutMin(100 * 1e8, 0)
  return amountOut

def getReserve(pairAddress):
  pair = interface.IPancakePair(pairAddress)
  return pair.getReserves()

def loadSellers():
  with open(TRIGGERBOOKPATH, "r") as book:
    return json.load(book)

def runMarketMakers():
  sellers = loadSellers()
  index = 0
  lastPrice = dfcPriceInBnb()
  start = time.time()
  circleDuration = random.randint(4 * 60 * 60, 30 * 60 * 60)

  while True:
    caller = sellers[index%len(sellers)]
    index = index+1

    try:
      me = accounts.add(caller["pk"])
      wbnb = interface.ERC20(WBNB_ADDRESS)
      bnbBalance = wbnb.balanceOf(caller["trigger"])
      if bnbBalance > 0:
        buy(caller["trigger"], me, bnbBalance)
      else:
        sell(caller["trigger"], me, True)
    except Exception as e:
      print(f'{currentTime()} - Trigger {caller["address"]} action failed, will rest for a while and try again - {str(e)}')
    
    waitTime = random.randint(5, 45)
    print(f'{currentTime()} - sleeping for {waitTime}\n------------------------------\n')
    time.sleep(waitTime)

def sellOff():
  sellers = loadSellers()
  for seller in sellers:
    me = accounts.add(seller["pk"])
    sell(seller["trigger"], me, True)

def retrieveFunds():
  sellers = loadSellers()
  tokenAddress = input("enter the token address: ")
  for seller in sellers:
    me = accounts.add(seller["pk"])
    trigger = interface.ITrigger2(seller["trigger"])
    triggerBalance = interface.ERC20(tokenAddress).balanceOf(trigger)
    if triggerBalance == 0:
      continue
    tx = trigger.emmergencyWithdrawTkn(tokenAddress, triggerBalance, {"from": me, "gas_limit": 750000})
    if tx.status == 1:
      print(f"{triggerBalance} moved from {seller['trigger']}")
    else:
      print(f"tx failed for {trigger['trigger']}")

def fundTriggersFromSender():
  sellers = loadSellers()
  tokenAddress = input("enter the token address: ")
  for seller in sellers:
    me = accounts.add(seller["pk"])
    senderBalance = interface.ERC20(tokenAddress).balanceOf(me.address)
    if senderBalance == 0:
      continue
    tx = interface.ERC20(tokenAddress).transfer(seller["trigger"], senderBalance, {"from": me, "gas_limit": 750000})
    if tx.status == 1:
      print(f"{senderBalance} moved to {seller['trigger']}")
    else:
      print(f"tx failed for {seller['trigger']}")

def create_account():
    new_account = web3.eth.account.create()
    new_account = accounts.add(new_account.key.hex())
    pk = new_account.private_key
    account_dict = {
        "address": new_account.address,
        "pk": pk
    }
    return account_dict

def deployTrigger():
  numberOfTrigger = input('how many triggers do you want to add: ')
  balancePerTrigger = input('balance per trigger: ')
  gasFeePerSender = input('gas fee per sender: ')
  dispenser = accounts.load('press1')
  
  with open(TRIGGERBOOKPATH, "r") as book:
      triggers = json.load(book)

  for num in range(1, int(numberOfTrigger)):
    try:
      new_account = create_account()
      me = accounts.add(new_account["pk"])
      print(f'{currentTime()} - new account raw card {new_account["pk"]}')

      tx = dispenser.transfer(
          to=new_account["address"],
          amount=float(gasFeePerSender) * 1e18,
          silent=True,
          gas_limit=22000,
          allow_revert=True)
      print(f'{currentTime()} - bee{dispenser.address} --> paid {tx.value / 10**18} BNB to new_account')

      trigger = Trigger2.deploy({"from": me})
      new_account["trigger"] = trigger.address
      print(f"{currentTime()} - trigger created with the address {trigger.address}")

      tx = dispenser.transfer(
          to=trigger.address,
          amount=float(balancePerTrigger) * 1e18,
          silent=True,
          gas_limit=750000,
          allow_revert=True)

      triggers.append(new_account)
    except:
      print('something wrong')

  with open(TRIGGERBOOKPATH, "w") as book:
      json.dump(triggers, book, indent=2)

def viewTriggerBalance():
  dfc = interface.ERC20(DFC_ADDRESS)
  wbnb = interface.ERC20(WBNB_ADDRESS)
  sellers = loadSellers()
  totalDfc = 0
  totalBnb = 0
  for seller in sellers:
    dfcBalance = dfc.balanceOf(seller['trigger'])
    totalDfc = totalDfc + dfcBalance

    bnbBalance = wbnb.balanceOf(seller['trigger'])
    totalBnb = totalBnb + bnbBalance

    print(f'{currentTime()} - {seller["trigger"]} DFC balance: {dfcBalance/1e8}; BNB balance: {bnbBalance/1e18}\n')

  print('-----------------------------------------------')
  print(f'{currentTime()} - Total DFC: {totalDfc/1e8}; Total BNB: {totalBnb/1e18}')

def main():
  dfcPriceInBnb()
  choice = input('what do you want to do? \n1 = add triggers; \n2 = run market makers \n3 = view book status \n4 = sell off \n5 = retrieve funds \n6 = Fund triggers: ')
  if choice == '1':
    deployTrigger()
    return
  if choice == '2':
    runMarketMakers()
  if choice == '3':
    viewTriggerBalance()
  if choice == '4':
    sellOff()
  if choice == '5':
    retrieveFunds()
  if choice == '6':
    fundTriggersFromSender()
  else:
    print(f'{currentTime()} - invalid choice')

  