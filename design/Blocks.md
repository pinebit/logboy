# Pulling Logs

## States
* Current - current, being processed block with subscription
* Backfilling - block scheduled for backfilling
* Processed - all logs of interest have been processed

## Happy Path

Given:
[block #1003, state=Current]
[block #1002, state=Processed]
[block #1001, state=Processed]
[block #1000, state=Processed]

A new log received, if the new block number (1005) is higher, then:

+[block #1005, state=Current]
+[block #1004, state=Processed]  // added to fill gap(s), when backfilling is off
[block #1003, state=Current => Processed]
[block #1002, state=Processed]
[block #1001, state=Processed]
- [block #1000, state=Processed] // truncated due to backfill setting

## RPC Disconnect 

Given:
[block #1005, state=Current]
[block #1004, state=Processed]
[block #1003, state=Processed]
[block #1002, state=Processed]
[block #1001, state=Processed]

We don't know if #1005 is finished or not. RPC got disconnected.

First off, set Current => Backfiling:
[block #1005, state=Backfiling]

Once reconnected:
- get the latest block number (HeaderByNumber) => #1007.
- now we know we need to check 1007, 1006 and 1005.
- set all to Backfiling

[block #1007, state=Backfiling]
[block #1006, state=Backfiling]
[block #1005, state=Backfiling]
[block #1004, state=Processed]
[block #1003, state=Processed]
[block #1002, state=Processed]
[block #1001, state=Processed]

- we start backfilling loop from the highest block number:

```golang
blockNumber := getHighestBackfilingBlockNumber() // 1007
FilterLogs(blockNumber)
handleLogs()
setBlockProcessed(blockNumber)
```

[block #1007, state=Processed]
[block #1006, state=Backfiling]
[block #1005, state=Backfiling]
[block #1004, state=Processed]
[block #1003, state=Processed]
[block #1002, state=Processed]
[block #1001, state=Processed]

- restart backfill timer and repeat for blocks 1006, 1005
- repeat until `getHighestBackfilingBlockNumber` returns `!ok` (backfilling is off)

What if a new log for a new block is received during backfilling?
e.g. new block #1009

[block #1009, state=Current]
[block #1008, state=Backfiling]   // gap set to Backfiling, because backfilling is on
[block #1007, state=Processed]
[block #1006, state=Backfiling]
[block #1005, state=Backfiling]
[block #1004, state=Processed]

Then next backfilling round would process 1008, then 1006, 1005.

## Chain Reorg

Given:
[block #1005, state=Current]
[block #1004, state=Processed]
[block #1003, state=Processed]
[block #1002, state=Processed]
[block #1001, state=Processed]

In case of reorg, we shall try removing logs. Whatever is sent to us.
- Shall we just send the removed logs to outputs?
- Shall we just ignore removed logs for now?

