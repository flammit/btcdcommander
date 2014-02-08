btcdcommander
=============


Quick example:
----------------------------
```
package main

import (
	"github.com/conformal/btcjson"
	"github.com/flammit/btcdcommander"
	"log"
	"time"
)

type Listener struct{}

func (l *Listener) BtcdConnected() {
	log.Printf("BTCD Connected")
}

func (l *Listener) BtcdDisconnected() {
	log.Printf("BTCD Disconnected")
}

func (l *Listener) BlockConnected(blockHash string) {
	log.Printf("Block Connected: hash=%v", blockHash)
}

func (l *Listener) BlockDisconnected(blockHash string) {
	log.Printf("Block Disconnected: hash=%v", blockHash)
}

func (l *Listener) AddedTransaction(txId string, amount int64) {
	log.Printf("New Transaction: txId=%v, amount%d", txId, amount)
}

func (l *Listener) AddedTransactionVerbose(rawTx *btcjson.TxRawResult) {
	log.Printf("New Verbose Transaction: rawTx=%#v", rawTx)
}

func main() {
	cfg := &btcdcommander.Config{
		CAFileName: "yourcert.pem",
		Connect:    "yourserver:yourport",
		Username:   "yourusername",
		Password:   "yourpassword",
		MainNet:    true,
	}
	listener := &Listener{}
	c := btcdcommander.NewCommander(cfg, listener)
	c.Start()

	err := c.NotifyAllNewTxs(true)
	if err != nil {
		log.Printf("Received Error on NotifyAllNewTxs: err=%v", err)
	}

	blockHash := "0000000000000000323cf200defd71ffc41dba822e19a544e8c32ce19479fe21"
	log.Printf("Getting block")
	block, err := c.GetVerboseBlock(blockHash, true)
	if err != nil {
		log.Printf("Received Error on Get Verbose Block: err=%v", err)
	}
	log.Printf("Get Verbose Block: height=%v, numTxns=%v", block.Height, len(block.RawTx))

	rawBlock, err := c.GetRawBlock(blockHash)
	if err != nil {
		log.Printf("Received Error on Get Raw Block: err=%v", err)
	}
	log.Printf("Get Raw Block: blockHex=%s", rawBlock)

	log.Printf("Getting Raw Tx")
	tx, err := c.GetRawTransaction("bc4c9e3465c5583e8cba7e858476dad13b195755bffd14185cfee7fb5e3bb43e", 1)
	if err != nil {
		log.Printf("Received Error on Get Raw Transaction: err=%v", err)
	}
	log.Printf("Got Transaction: %#v", tx)

	time.Sleep(5 * time.Minute)

	c.Stop()
}
```
