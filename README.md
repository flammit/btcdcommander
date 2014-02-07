btcdcommander
=============


Quick example:
----------------------------
```
package main

import (
	"github.com/flammit/btcdcommander"
	"log"
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

	log.Printf("Getting block")
	block, err := c.GetBlock("0000000000000000323cf200defd71ffc41dba822e19a544e8c32ce19479fe21", true, true)
	if err != nil {
		log.Printf("Received Error on Get Block: err=%v", err)
	}
	log.Printf("Get Block: height=%v, numTxns=%v", block.Height, len(block.RawTx))

	log.Printf("Getting Raw Tx")
	tx, err := c.GetRawTransaction("bc4c9e3465c5583e8cba7e858476dad13b195755bffd14185cfee7fb5e3bb43e", 1)
	if err != nil {
		log.Printf("Received Error on Get Raw Transaction: err=%v", err)
	}
	log.Printf("Got Transaction: %#v", tx)
	c.Stop()
}
```
