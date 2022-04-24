# cosmos-ab

Cosmos AB (Application Blockchain) without Cosmos SDK. This mini project is for understanding how Tendermint and AB interacts with ABCI.

Application in this project saves data as KV format. You can request to save the key-value data and query them. 

## Standalone

Standalone version of AB is just running Application server without consensus mechanism. You can request to update state (deliver transaction, commit) with the CLI manually.

Install the CLI with the command:

```bash
make install
```

Run the Application server with the command:

```bash
ab-cli kv # listening on the 0.0.0.0:26658
```

Open up the other terminal window, then send the request to save:

```bash
ab-cli tx "\"abc=def\""
```
The response should be similar to as follows
```text
{"level":"info","module":"ab-client","impl":"grpcClient","service":"grpcClient","time":"2022-04-24T19:03:32+09:00","message":"starting service"}
{"level":"info","module":"ab-client","addr":"tcp://0.0.0.0:26658","time":"2022-04-24T19:03:32+09:00","message":"Dialed server. Waiting for echo."}
-> code: OK
```

Then if you want to query the data you've saved, then run the command:
```bash
ab-cli query "\"abc\""
```
You can see the log with the value of `exists`
```text
{"level":"info","module":"ab-client","impl":"grpcClient","service":"grpcClient","time":"2022-04-24T19:06:26+09:00","message":"starting service"}
{"level":"info","module":"ab-client","addr":"tcp://0.0.0.0:26658","time":"2022-04-24T19:06:26+09:00","message":"Dialed server. Waiting for echo."}
-> code: OK
-> log: exists
-> height: 0
-> key: abc
-> key.hex: 616263
-> value: def
-> value.hex: 646566
```

If you query with the key which is not stored, then server returns the log as `does not exist`

```bash
ab-cli query "\"fff\""
```
```text
{"level":"info","module":"ab-client","impl":"grpcClient","service":"grpcClient","time":"2022-04-24T19:11:01+09:00","message":"starting service"}
{"level":"info","module":"ab-client","addr":"tcp://0.0.0.0:26658","time":"2022-04-24T19:11:01+09:00","message":"Dialed server. Waiting for echo."}
-> code: OK
-> log: does not exist
-> height: 0
-> key: fff
-> key.hex: 666666
```

## Integrate with Tendermint (TBD)

Each peer running AB based on Tendermint can broadcast its transaction and synced its state by consensus. To integrate with Tendermint you need to install Tendermint first.

