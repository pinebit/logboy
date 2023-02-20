# Automatic Prometheus metrics

Input:
- Contract name: `FooBar`
- Contract address: `0xd9145CCE52D386f254917e481eB44e9943F39138`
- Contract ABI: `FooBar.abi`

`event SomeEvent(address indexed user, uint256 foo, string bar);`

Common metrics:
- CounterVec: `logs_processed`, Labels: `chainID`, `contractAddress`
- CounterVec: `rpc_errors`, Labels: `chainID`, `rpc`
- CounterVec: `postgres_errors`, Labels: ?

Per contract:
- CounterVec: `FooBar`, Labels: `chainID`, `contractAddress`, `eventName`

Config:
- Gauges:
    - FooBar.SomeEvent.foo
- Histograms:
    - FooBar.SomeEvent.foo
- Labels:
    - FooBar.SomeEvent.user

Upon re-orgs:
- decrement counters,
- subtract gauges, etc?