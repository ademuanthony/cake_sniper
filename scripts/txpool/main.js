var Web3 = require('web3');

const main = async () => {
  //
  const options = {
    reconnect: {
      auto: true,
      delay: 2000,
      maxAttempts: 3,
      onTimeout: false,
    },
  };
  const web3 = new Web3(
    new Web3.providers.WebsocketProvider(
      process.env.BSC_NODE_ADDRESS,
      options
    )
  );
  var redis = require('redis');
  var publisher = redis.createClient();
  publisher.connect()

  // subscribe to pendingTransactions events
  web3.eth
    .subscribe('pendingTransactions', async (error, result) => {
      if (error) console.log('error', error);
    })
    .on('data', async (trxId) => {
      // receives the transaction id
      // console.log('🧮 TRX ID >> ', trxId);
      publisher.publish('NEW_TRANSACTION', trxId);
    });
};

main();
