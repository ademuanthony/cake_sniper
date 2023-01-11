package services

import (
	"dark_forester/global"
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis"
)


var RedisClient *redis.Client

func init() {
	RedisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	//sub to whitelisting

	go func() {
		subscriber := RedisClient.Subscribe("MARKET_TESTED")
		for {
			func() {
				msg, err := subscriber.ReceiveMessage()
				if err != nil {
					fmt.Println("Error in receiving redis msg", err)
					return
				}

				var market global.Market
				if err := json.Unmarshal([]byte(msg.Payload), &market); err != nil {
					fmt.Println("Error in marshalling new market info", err)
					return
				}
				if oldMarket, f := global.SANDWICH_BOOK[market.Address]; f && oldMarket.ManuallyDisabled {
					return
				}
				fmt.Println("New market tested", market)
				global.IN_SANDWICH_BOOK[market.Address] = true
				global.SANDWICH_BOOK[market.Address] = market
			}()
		}
	}()
}

func GetMarket(address string) (*global.Market, error) {
	marketJson, err := RedisClient.Get(address).Result()
	if err != nil {
		return nil, err
	}

	var market global.Market
	err = json.Unmarshal([]byte(marketJson), &market)
	return &market, err
}

func SaveMarket(market global.Market) error {
	marketJson, err := json.Marshal(market)
	if err != nil {
		return err
	}
	return RedisClient.Set(market.Address.Hex(), marketJson, 0).Err()
}

