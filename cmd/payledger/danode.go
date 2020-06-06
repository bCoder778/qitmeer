/*
 * Copyright (c) 2020.
 * Project:qitmeer
 * File:danode.go
 * Date:6/6/20 9:28 AM
 * Author:Jin
 * Email:lochjin@gmail.com
 */

package main

import (
	"fmt"
	"github.com/Qitmeer/qitmeer/common/hash"
	"github.com/Qitmeer/qitmeer/core/blockchain"
	"github.com/Qitmeer/qitmeer/core/blockdag"
	"github.com/Qitmeer/qitmeer/core/dbnamespace"
	"github.com/Qitmeer/qitmeer/core/types"
	"github.com/Qitmeer/qitmeer/database"
	"github.com/Qitmeer/qitmeer/engine/txscript"
	"github.com/Qitmeer/qitmeer/log"
	"github.com/Qitmeer/qitmeer/params"
	"github.com/Qitmeer/qitmeer/services/index"
	"github.com/Qitmeer/qitmeer/services/mining"
	"path"
)

type DebugAddressNode struct {
	name     string
	bc       *blockchain.BlockChain
	db       database.DB
	cfg      *Config
	endPoint blockdag.IBlock
}

func (node *DebugAddressNode) init(cfg *Config) error {
	node.cfg = cfg

	// Load the block database.
	db, err := LoadBlockDB(cfg.DbType, cfg.DataDir, true)
	if err != nil {
		log.Error("load block database", "error", err)
		return err
	}

	node.db = db
	//
	var indexes []index.Indexer
	txIndex := index.NewTxIndex(db)
	indexes = append(indexes, txIndex)
	// index-manager
	indexManager := index.NewManager(db, indexes, params.ActiveNetParams.Params)

	bc, err := blockchain.New(&blockchain.Config{
		DB:           db,
		ChainParams:  params.ActiveNetParams.Params,
		TimeSource:   blockchain.NewMedianTime(),
		DAGType:      cfg.DAGType,
		BlockVersion: mining.BlockVersion(params.ActiveNetParams.Params.Net),
		IndexManager: indexManager,
	})
	if err != nil {
		log.Error(err.Error())
		return err
	}
	node.bc = bc
	node.name = path.Base(cfg.DataDir)

	log.Info(fmt.Sprintf("Load Data:%s", cfg.DataDir))

	return node.processAddress()
}

func (node *DebugAddressNode) exit() {
	if node.db != nil {
		log.Info(fmt.Sprintf("Gracefully shutting down the database:%s", node.name))
		node.db.Close()
	}
}

func (node *DebugAddressNode) BlockChain() *blockchain.BlockChain {
	return node.bc
}

func (node *DebugAddressNode) DB() database.DB {
	return node.db
}

func (node *DebugAddressNode) processAddress() error {
	db := node.db
	par := params.ActiveNetParams.Params
	// 检测给定地址的账本记录
	checkAddress := node.cfg.DebugAddress
	tradeRecord := []*TradeRecord{}
	tradeRecordMap := map[types.TxOutPoint]*TradeRecord{}
	blueMap := map[uint]bool{}
	mainTip := node.bc.BlockDAG().GetMainChainTip()
	fmt.Printf("开始分析:%s  mainTip:%s mainOrder:%d total:%d \n", checkAddress, mainTip.GetHash(), mainTip.GetOrder(), node.bc.BlockDAG().GetBlockTotal())
	for i := uint(0); i < node.bc.BlockDAG().GetBlockTotal(); i++ {
		ib := node.bc.BlockDAG().GetBlockById(i)
		if ib == nil {
			return fmt.Errorf("出错了：%d", i)
		}
		block, err := node.bc.FetchBlockByHash(ib.GetHash())
		if err != nil {
			return fmt.Errorf("找不到块：%s", err)
		}
		confims := node.bc.BlockDAG().GetConfirmations(ib.GetID())

		for _, tx := range block.Transactions() {
			txHash := tx.Hash()
			txFullHash := tx.Tx.TxHashFull()

			txValid := node.isTxValid(db, txHash, &txFullHash, ib.GetHash())
			if !tx.Tx.IsCoinBase() {
				for txInIndex, txIn := range tx.Tx.TxIn {
					pretr, ok := tradeRecordMap[txIn.PreviousOut]
					if ok {
						tr := &TradeRecord{}
						tr.blockHash = ib.GetHash()
						tr.blockId = ib.GetID()
						tr.blockOrder = ib.GetOrder()
						tr.blockConfirm = confims
						tr.blockStatus = byte(ib.GetStatus())
						tr.blockBlue = 2
						tr.blockHeight = ib.GetHeight()
						tr.txHash = txHash
						tr.txFullHash = &txFullHash
						tr.txUIndex = txInIndex
						tr.txIsIn = true
						tr.txValid = txValid
						tr.isCoinbase = false
						tr.amount = pretr.amount

						if !knownInvalid(tr.blockStatus) && tr.txValid {

							isblue, ok := blueMap[ib.GetID()]
							if !ok {
								isblue = node.bc.BlockDAG().IsBlue(ib.GetID())
								blueMap[ib.GetID()] = isblue
							}
							if isblue {
								tr.blockBlue = 1
							} else {
								tr.blockBlue = 0
							}
						}
						tradeRecord = append(tradeRecord, tr)
					}
				}

			}
			for txOutIndex, txOut := range tx.Tx.TxOut {
				_, addr, _, err := txscript.ExtractPkScriptAddrs(txOut.GetPkScript(), par)
				if err != nil {
					return err
				}
				if len(addr) != 1 {
					fmt.Printf("忽略多地址的情况：%d\n", len(addr))
					continue
				}
				addrStr := addr[0].String()
				if addrStr != checkAddress {
					continue
				}

				tr := &TradeRecord{}
				tr.blockHash = ib.GetHash()
				tr.blockId = ib.GetID()
				tr.blockOrder = ib.GetOrder()
				tr.blockConfirm = confims
				tr.blockStatus = byte(ib.GetStatus())
				tr.blockBlue = 2
				tr.blockHeight = ib.GetHeight()
				tr.txHash = txHash
				tr.txFullHash = &txFullHash
				tr.txUIndex = txOutIndex
				tr.txIsIn = false
				tr.txValid = txValid
				tr.amount = txOut.Amount
				tr.isCoinbase = tx.Tx.IsCoinBase()

				if !knownInvalid(tr.blockStatus) && tr.txValid {

					isblue, ok := blueMap[ib.GetID()]
					if !ok {
						isblue = node.bc.BlockDAG().IsBlue(ib.GetID())
						blueMap[ib.GetID()] = isblue
					}
					if isblue {
						tr.blockBlue = 1
					} else {
						tr.blockBlue = 0
					}
				}

				tradeRecord = append(tradeRecord, tr)
				txOutPoint := types.TxOutPoint{*txHash, uint32(txOutIndex)}
				tradeRecordMap[txOutPoint] = tr
			}
		}
	}
	acc := int64(0)
	for i, tr := range tradeRecord {
		isValid := true
		if tr.isCoinbase && !tr.txIsIn && tr.blockBlue == 0 {
			isValid = false
		}
		if isValid {
			if tr.txValid {
				if tr.txIsIn {
					acc -= int64(tr.amount)
				} else {
					acc += int64(tr.amount)
				}
			}
		}

		fmt.Printf("%d Block Hash:%s Id:%d Order:%d Confirm:%d Valid:%v Blue:%s Height:%d ; Tx Hash:%s FullHash:%s UIndex:%d IsIn:%v Valid:%v Amount:%d Coinbase:%v  当前余额:%d\n",
			i, tr.blockHash, tr.blockId, tr.blockOrder, tr.blockConfirm, !knownInvalid(tr.blockStatus), blueState(tr.blockBlue), tr.blockHeight, tr.txHash, tr.txFullHash, tr.txUIndex, tr.txIsIn, tr.txValid,
			tr.amount, tr.isCoinbase, acc)

	}

	fmt.Printf("结论：%s 账本记录数:%d 总余额:%d\n", checkAddress, len(tradeRecord), acc)

	return node.checkUTXO(db, checkAddress, par, blueMap)
}

func (node *DebugAddressNode) isTxValid(db database.DB, txHash *hash.Hash, txFullHash *hash.Hash, blockHash *hash.Hash) bool {
	var preTx *types.Transaction
	var preBlockH *hash.Hash
	err := db.View(func(dbTx database.Tx) error {
		dtx, blockH, erro := index.DBFetchTxAndBlock(dbTx, txHash)
		if erro != nil {
			return erro
		}
		preTx = dtx
		preBlockH = blockH
		return nil
	})

	if err != nil {
		//fmt.Printf("txFullHash:%s txHash:%s   error:%s\n", txFullHash.String(), txHash.String(), err.Error())
		return false
	}
	ptxFullHash := preTx.TxHashFull()

	if !preBlockH.IsEqual(blockHash) || !txFullHash.IsEqual(&ptxFullHash) {
		//fmt.Printf("txFullHash:%s txHash:%s   error: 可能是重复交易\n", txFullHash.String(), txHash.String())
		return false
	}
	return true
}

func (node *DebugAddressNode) checkUTXO(db database.DB, checkAddress string, par *params.Params, blueMap map[uint]bool) error {

	fmt.Printf("分析UTXO:%s\n", checkAddress)

	var totalAmount uint64
	var count int

	err := db.View(func(dbTx database.Tx) error {
		meta := dbTx.Metadata()
		utxoBucket := meta.Bucket(dbnamespace.UtxoSetBucketName)
		cursor := utxoBucket.Cursor()
		for ok := cursor.First(); ok; ok = cursor.Next() {
			serializedUtxo := utxoBucket.Get(cursor.Key())

			// Deserialize the utxo entry and return it.
			entry, err := blockchain.DeserializeUtxoEntry(serializedUtxo)
			if err != nil {
				return err
			}
			if entry.IsSpent() {
				continue
			}
			ib := node.bc.BlockDAG().GetBlock(entry.BlockHash())
			if ib.GetOrder() == blockdag.MaxBlockOrder {
				continue
			}
			_, addr, _, err := txscript.ExtractPkScriptAddrs(entry.PkScript(), par)
			if err != nil {
				return err
			}
			addrStr := addr[0].String()
			if addrStr != checkAddress {
				continue
			}
			isValid := true
			blockBlue := 2
			if entry.IsCoinBase() {
				isblue, ok := blueMap[ib.GetID()]
				if !ok {
					isblue = node.bc.BlockDAG().IsBlue(ib.GetID())
				}
				if !isblue {
					isValid = false
					blockBlue = 0
				} else {
					blockBlue = 1
				}

			}

			fmt.Printf("BlockHash:%s Amount:%d Valid:%v Blue:%s\n", ib.GetHash(), entry.Amount(), isValid, blueState(blockBlue))

			if isValid {
				totalAmount += entry.Amount()
			}

			count++
		}
		return nil
	})
	if err != nil {
		return err
	}
	fmt.Printf("UTXO结论：总记录数：%d  总余额:%d\n", count, totalAmount)
	return nil
}

func knownInvalid(status byte) bool {
	var statusInvalid byte
	statusInvalid = 1 << 2
	return status&statusInvalid != 0
}

func blueState(blockBlue int) string {
	if blockBlue == 0 {
		return "否"
	} else if blockBlue == 1 {
		return "是"
	}
	return "?"
}

type TradeRecord struct {
	blockHash    *hash.Hash
	blockId      uint
	blockOrder   uint
	blockConfirm uint
	blockStatus  byte
	blockBlue    int // 0:not blue;  1：blue  2：Cannot confirm
	blockHeight  uint
	txHash       *hash.Hash
	txFullHash   *hash.Hash
	txUIndex     int
	txValid      bool
	txIsIn       bool
	amount       uint64
	isCoinbase   bool
}
