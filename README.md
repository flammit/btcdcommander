btcdcommander
=============


Quick example:
----------------------------
```
package main

import (
	"github.com/conformal/btcws"
	"github.com/flammit/btcdcommander"
	"log"
	"time"
)

func main() {
	cfg := &btcdcommander.Config{
		CAFileName: "yourcert.pem",
		Connect:    "yourserver:yourport",
		Username:   "yourusername",
		Password:   "yourpassword",
		MainNet:    true,
	}
	
	c := btcdcommander.NewCommander(cfg)

	go func() {
		ntfnChan := c.NtfnChan()
		for {
			cmd, ok := <-ntfnChan
			if !ok {
				return
			}

			switch cmd.Method() {
			case btcdcommander.BtcdConnectedNtfnMethod:
				log.Printf("BTCD CONNECTED")
			case btcdcommander.BtcdDisconnectedNtfnMethod:
				log.Printf("BTCD DISCONNECTED")
			case btcws.BlockConnectedNtfnMethod:
				bcn, ok := cmd.(*btcws.BlockConnectedNtfn)
				if !ok {
					return
				}
				log.Printf("BLOCK CONNECTED %v", bcn.Hash)
				go func() {
					log.Printf("REQUESTING VERBOSE BLOCK")
					block, err := c.GetVerboseBlock(bcn.Hash, true)
					if err != nil {
						log.Printf("Received Error on Get Block: err=%v", err)
					}
					log.Printf("BLOCK CONNECTED VERBOSE: height=%v, numTxns=%v", block.Height, len(block.RawTx))
				}()
			case btcws.BlockDisconnectedNtfnMethod:
				bdn, ok := cmd.(*btcws.BlockDisconnectedNtfn)
				if !ok {
					return
				}
				log.Printf("BLOCK DISCONNECTED %v", bdn.Hash)
			case btcws.AllTxNtfnMethod:
				tx, ok := cmd.(*btcws.AllTxNtfn)
				if !ok {
					return
				}
				log.Printf("NEW TRANSACTION: txId=%v, amount=%d", tx.TxID, tx.Amount)
			case btcws.AllVerboseTxNtfnMethod:
				vtx, ok := cmd.(*btcws.AllVerboseTxNtfn)
				if !ok {
					return
				}
				log.Printf("NEW VERBOSE TRANSACTION: rawTx=%#v", vtx.RawTx)
			}
		}
	}()

	c.Start()
	log.Printf("Finished Start")

	err := c.NotifyAllNewTxs(false)
	if err != nil {
		log.Printf("Received Error on NotifyAllNewTxs: err=%v", err)
	}

	log.Printf("Getting block")
	block, err := c.GetVerboseBlock("0000000000000000323cf200defd71ffc41dba822e19a544e8c32ce19479fe21", true)
	if err != nil {
		log.Printf("Received Error on Get Block: err=%v", err)
	}
	log.Printf("Get Block: height=%v, numTxns=%v", block.Height, len(block.RawTx))

	log.Printf("Getting Raw TX")
	tx, err := c.GetRawTransaction("bc4c9e3465c5583e8cba7e858476dad13b195755bffd14185cfee7fb5e3bb43e", 1)
	if err != nil {
		log.Printf("Received Error on Get Raw Transaction: err=%v", err)
	}
	log.Printf("Got Transaction: %#v", tx)

	log.Printf("Sleeping for 60 minutes")
	time.Sleep(60 * time.Minute)
	log.Printf("Shutting down")

	c.Stop()
}
```
