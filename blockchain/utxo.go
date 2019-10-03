package blockchain

import (
	"bytes"
	"encoding/hex"
	"github.com/dgraph-io/badger"
	"log"
)

var (
	utxoPrefix = []byte("utxo-")
)

type UTXOSet struct {
	BlockChain *BlockChain
}

func (u UTXOSet) FindSpendableOutputs(pubKeyHash []byte, amount int) (int, map[string][]int) {
	unspentOuts := make(map[string][]int)
	accumulated := 0

	err := u.BlockChain.Database.DB.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		for it.Seek(utxoPrefix); it.ValidForPrefix(utxoPrefix); it.Next() {
			item := it.Item()
			k := item.Key()

			k = bytes.TrimPrefix(k, utxoPrefix)
			txID := hex.EncodeToString(k)

			err := item.Value(func(val []byte) error {
				outs := DeserializeOutputs(val)

				for outIdx, out := range outs.Outputs {
					if out.IsLockedWithKey(pubKeyHash) && accumulated < amount {
						accumulated += out.Value
						unspentOuts[txID] = append(unspentOuts[txID], outIdx)
					}
				}

				return nil
			})

			Handle(err)
		}
		return nil
	})
	Handle(err)
	return accumulated, unspentOuts
}

func (u UTXOSet) FindUnspentTransactions(pubKeyHash []byte) []TxOutput {
	var UTXOs []TxOutput

	err := u.BlockChain.Database.DB.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(utxoPrefix); it.ValidForPrefix(utxoPrefix); it.Next() {
			item := it.Item()
			var v []byte

			err := item.Value(func(val []byte) error {
				v = val
				return nil
			})
			Handle(err)

			outs := DeserializeOutputs(v)
			for _, out := range outs.Outputs {
				if out.IsLockedWithKey(pubKeyHash) {
					UTXOs = append(UTXOs, out)
				}
			}

		}
		return nil
	})
	Handle(err)

	return UTXOs
}

func (u UTXOSet) FindUTXO(pubKeyHash []byte) []TxOutput {
	var UTXOs []TxOutput

	db := u.BlockChain.Database

	err := db.DB.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(utxoPrefix); it.ValidForPrefix(utxoPrefix); it.Next() {
			item := it.Item()

			err := item.Value(func(val []byte) error {
				outs := DeserializeOutputs(val)

				for _, out := range outs.Outputs {
					if out.IsLockedWithKey(pubKeyHash) {
						UTXOs = append(UTXOs, out)
					}
				}

				return nil
			})

			Handle(err)
		}

		return nil
	})
	Handle(err)

	return UTXOs
}

func (u UTXOSet) CountTransactions() int {
	db := u.BlockChain.Database
	counter := 0

	err := db.Iterator(true, func(it *badger.Iterator) error {
		for it.Seek(utxoPrefix); it.ValidForPrefix(utxoPrefix); it.Next() {
			counter++
		}

		return nil
	})

	Handle(err)

	return counter
}

func (u UTXOSet) Reindex() {
	db := u.BlockChain.Database

	u.DeleteByPrefix(utxoPrefix)

	UTXO := u.BlockChain.FindUTXO()

	err := db.DB.Update(func(txn *badger.Txn) error {
		for txId, outs := range UTXO {
			key, err := hex.DecodeString(txId)
			if err != nil {
				return err
			}
			key = append(utxoPrefix, key...)

			err = txn.Set(key, outs.Serialize())
			Handle(err)
		}

		return nil
	})
	Handle(err)
}

func (u *UTXOSet) Update(block *Block) {
	db := u.BlockChain.Database

	err := db.DB.Update(func(txn *badger.Txn) error {
		for _, tx := range block.Transactions {
			if tx.IsCoinbase() == false {
				for _, in := range tx.Inputs {
					updatedOuts := TxOutputs{}
					inID := append(utxoPrefix, in.ID...)
					item, err := txn.Get(inID)
					Handle(err)

					err = item.Value(func(val []byte) error {
						outs := DeserializeOutputs(val)

						for outIdx, out := range outs.Outputs {
							if outIdx != in.Out {
								updatedOuts.Outputs = append(updatedOuts.Outputs, out)
							}
						}

						if len(updatedOuts.Outputs) == 0 {
							if err := txn.Delete(inID); err != nil {
								log.Panic(err)
							}

						} else {
							if err := txn.Set(inID, updatedOuts.Serialize()); err != nil {
								log.Panic(err)
							}
						}

						return nil
					})

					Handle(err)
				}
			}

			newOutputs := TxOutputs{}
			for _, out := range tx.Outputs {
				newOutputs.Outputs = append(newOutputs.Outputs, out)
			}

			txID := append(utxoPrefix, tx.ID...)
			if err := txn.Set(txID, newOutputs.Serialize()); err != nil {
				log.Panic(err)
			}
		}

		return nil
	})
	Handle(err)
}

func (u *UTXOSet) DeleteByPrefix(prefix []byte) {
	deleteKeys := func(keysForDelete [][]byte) error {
		if err := u.BlockChain.Database.DB.Update(func(txn *badger.Txn) error {
			for _, key := range keysForDelete {
				if err := txn.Delete(key); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}

		return nil
	}

	collectSize := 100000

	err := u.BlockChain.Database.Iterator(false, func(it *badger.Iterator) error {
		keysForDelete := make([][]byte, 0, collectSize)
		keysCollected := 0

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			key := it.Item().KeyCopy(nil)
			keysForDelete = append(keysForDelete, key)
			keysCollected++
			if keysCollected == collectSize {
				if err := deleteKeys(keysForDelete); err != nil {
					log.Panic(err)
				}
				keysForDelete = make([][]byte, 0, collectSize)
				keysCollected = 0
			}
		}

		if keysCollected > 0 {
			if err := deleteKeys(keysForDelete); err != nil {
				log.Panic(err)
			}
		}

		return nil
	})

	Handle(err)
}
