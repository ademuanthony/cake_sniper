CREATE TABLE IF NOT EXISTS pair (
  pairAddress character varying(126) not null primary key,
  token0 character varying(126) not null,
  token1 character varying(126) not null,
  router character varying(126) not null,
  reserve0 character varying(126) not null default '0',
  reserve1 character varying(126) not null default '0',
  lastUpdate int8 not null default 0,
  tvl int8 not null default 0
);

CREATE TABLE IF NOT EXISTS tokenSecurity (
  address character varying(126) not null primary key,
  data text not null
);

CREATE TABLE IF NOT EXISTS tradePosition (
  id character varying(64) primary key,
  accountId character varying(128) not null,
  token character varying(128) not null,
  openingAmount character varying(128) not null,
  openingMarket character varying(128) not null,
  openingDate int8 not null,
  closingAmount character varying(128) not null,
  closingMarket character varying(128) not null,
  closingDate int8 not null
);

CREATE TABLE IF NOT EXISTS tradeTransaction(
  id serial not null primary key,
  positionId character varying(64) not null references tradePosition(id),
  tokenIn character varying(128) not null, 
  tokenOut character varying(128) not null, 
  amountIn character varying(128) not null, 
  amountOut character varying(128) not null, 
  fee character varying(128) not null, 
  price float not null,
  hash character varying(128) not null
);

CREATE TABLE IF NOT EXISTS bnbRoute(
  id serial not null primary key,
  token character varying(128) not null,
  tokens character varying(1000) not null,
  pairs character varying(1000) not null,
  routeLength int not null,
  baseTvl int not null,
  unique(token, pairs)
);
