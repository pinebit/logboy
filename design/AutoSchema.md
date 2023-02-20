# Automatic DB schema migration from ABI

Input:
- Contract name: `FooBar`
- Contract address: `0xd9145CCE52D386f254917e481eB44e9943F39138`
- Contract ABI: `FooBar.abi`

Runtime fetching:
- Using Etherscan.
- RPC can report chainID / network name?

Assumptions:
- There maybe many different contracts having the same name.
- The same contract may have many versions, having different addresses.
- The same contract may be deployed many times.
- Take into account multiple networks.

But, for most users the desired experience is this:
1. For the first time, OBRY creates inital tables per each event:
```sql
CREATE TABLE NETWORK.TestContract_SomeEvent(
    _id BIGSERIAL PRIMARY KEY,
    _block_timestamp TIMESTAMP NOT NULL,
    _tx_hash TEXT NOT NULL,
    _contract_address TEXT NOT NULL,
    user TEXT NOT NULL,
    foo NUMERIC(78,0) NOT NULL,
    bar TEXT NOT NULL
);

CREATE INDEX TestContract_SomeEvent_Index ON TestContract_SomeEvent(_block_timestamp, _tx_hash, _contract_address);
CREATE INDEX TestContract_SomeEvent_User ON TestContract_SomeEvent(user);
```

2. OBRY ingests SomeEvent data into the table from the designated contract address.
2.1. OBRY backfills events if requested.
3. User can query the table.

What happens if a user deploys the same contract multiple times?
In this case the table schema does not change, but the ingestor populates different `_contract_address`.
And, of course, the listener must be configured to listen to multiple addresses.
For different chains, OBRY creates different PG schema (namespace).

What happens if a user changes ABI of a known event?
e.g.
`event SomeEvent(address indexed user, uint256 foo, string bar);`
a) added a new param at the end: `ALTER TABLE xxx ADD COLUMN... SET DEFAULT?`
b) a param is removed: `ALTER TABLE xxx DROP COLUMN...`
c) event renamed => this is a new table
d) a param type changed: `ALTER COLUMN xxx TYPE ...`
OBRY should try converting existing data, if possible;
if conversion is not possible:
- create a new column with suffix `_newtype`, e.g. `bar_bytes32`
- drop existing column and add new with default values (default as per config?)

Another great solution is to use JSONB for event parameters:
```sql
CREATE TABLE NETWORK.TestContract_SomeEvent(
    id BIGSERIAL PRIMARY KEY,
    block_timestamp TIMESTAMP NOT NULL,
    tx_hash TEXT NOT NULL,
    contract_address TEXT NOT NULL,
    params JSONB NOT NULL
);
```

TODO: handle reorgs...
OBRY should remove removed txes from the tables...


