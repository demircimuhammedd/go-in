## Usage
Get the balance for an address
```
$ go run main.go getbalance -address ADDRESS
```

Create a blockchain and send genesis reward to address
```
$ go run main.go createblockchain -address ADDRESS
```

Print the blocks in the chain
```
$ go run main.go printchain
```

Send amount of coins
```
$ go run main.go send -from FROM -to TO -amount AMOUNT
```

Create a new Wallet
```
$ go run main.go createwallet
```

List the addresses in wallet file
```
$ go run main.go listaddresses
```

Rebuild the UTXO set
```
$ go run main.go reindexutxo
```

Start a node with ID specified in NODE_ID env. var. -miner enables mining
```
$ go run main.go startnode -miner ADDRESS
```

## Wiki
- [How is the wallet address created?](https://github.com/ibrahimsn98/blockchain-in-go/wiki/How-is-the-wallet-address-created%3F)


## Requirements
- github.com/dgraph-io/badger
- github.com/mr-tron/base58
- golang.org/x/crypto
- gopkg.in/vrecan/death.v3


### Video Tutorials
[Tensor Programming](https://www.youtube.com/channel/UCYqCZOwHbnPwyjawKfE21wg)