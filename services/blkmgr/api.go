// Copyright (c) 2017-2018 The nox developers

package blkmgr

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/noxproject/nox/common/hash"
	"github.com/noxproject/nox/core/blockchain"
	"github.com/noxproject/nox/core/json"
	"github.com/noxproject/nox/rpc"
	"github.com/noxproject/nox/services/common/error"
	"github.com/noxproject/nox/services/common/marshal"
)

func (b *BlockManager) GetChain() *blockchain.BlockChain{
	return b.chain
}
func (b *BlockManager) API() rpc.API {
	return rpc.API{
		NameSpace: rpc.DefaultServiceNameSpace,
		Service:   NewPublicBlockAPI(b),
	}
}

type PublicBlockAPI struct{
	bm *BlockManager
}

func NewPublicBlockAPI(bm *BlockManager) *PublicBlockAPI {
	return &PublicBlockAPI{bm}
}

//TODO, refactor BlkMgr API
func (api *PublicBlockAPI) GetBlockhash(height uint) (string, error){
 	block,err := api.bm.chain.BlockByHeight(uint64(height))
 	if err!=nil {
 		return "",err
	}
	return block.Hash().String(),nil
}

//TODO, refactor BlkMgr API
func (api *PublicBlockAPI) GetBlockByHeight(height uint64, fullTx bool) (json.OrderedResult, error){
	block,err := api.bm.chain.BlockByHeight(height)
 	if err!=nil {
 		return nil,err
	}

	best := api.bm.chain.BestSnapshot()

	// See if this block is an orphan and adjust Confirmations accordingly.
	onMainChain, _ := api.bm.chain.MainChainHasBlock(block.Hash())

	// Get next block hash unless there are none.
	var nextHashString string
	confirmations := int64(-1)

	if onMainChain {
		if height < best.Height {
			nextHash, err := api.bm.chain.BlockHashByHeight(height + 1)
			if err != nil {
				return nil, err
			}
			nextHashString = nextHash.String()
		}
		confirmations = 1 + int64(best.Height) - int64(height)
	}
	//TODO, refactor marshal api
	fields, err := marshal.MarshalJsonBlock(block, true, fullTx, api.bm.params, confirmations, nextHashString)
	if err != nil {
		return nil, err
	}
	return fields,nil
}


func (api *PublicBlockAPI) GetBlock(hash hash.Hash, verbose bool) (interface{}, error){

	// Load the raw block bytes from the database.
	// Note :
	// FetchBlockByHash differs from BlockByHash in that this one also returns blocks
	// that are not part of the main chain (if they are known).
	blk, err := api.bm.chain.FetchBlockByHash(&hash)
	if err != nil {
		return nil,err
	}

	// When the verbose flag isn't set, simply return the
	// network-serialized block as a hex-encoded string.
	if !verbose {
		blkBytes, err := blk.Bytes()
		if err != nil {
			return nil, er.RpcInternalError(err.Error(),
				"Could not serialize block")
		}
		return hex.EncodeToString(blkBytes), nil
	}
	best := api.bm.chain.BestSnapshot()

	// See if this block is an orphan and adjust Confirmations accordingly.
	onMainChain, _ := api.bm.chain.MainChainHasBlock(&hash)

	// Get next block hash unless there are none.
	var nextHashString string
	blockHeader := &blk.Block().Header
	height := blockHeader.Height
	confirmations := int64(-1)

	if onMainChain {
		if height < best.Height {
			nextHash, err := api.bm.chain.BlockHashByHeight(height + 1)
			if err != nil {
				return nil, err
			}
			nextHashString = nextHash.String()
		}
		confirmations = 1 + int64(best.Height) - int64(height)
	}
	//TODO, refactor marshal api
	fields, err := marshal.MarshalJsonBlock(blk, true, verbose, api.bm.params, confirmations, nextHashString)
	if err != nil {
		return nil, err
	}
	return fields,nil

}

func (api *PublicBlockAPI) GetBestBlockHash() (interface{}, error){
	best := api.bm.chain.BestSnapshot()
	return best.Hash.String(), nil
}

func (api *PublicBlockAPI) GetBlockCount() (interface{}, error){
	best := api.bm.chain.BestSnapshot()
	return best.Height, nil
}

// GetBlockHeader implements the getblockheader command.
func (api *PublicBlockAPI) GetBlockHeader(hash hash.Hash, verbose bool) (interface{}, error) {

	// Fetch the header from chain.
	blockHeader, err := api.bm.chain.HeaderByHash(&hash)
	if err != nil {
		return nil, er.RpcInternalError(err.Error(), fmt.Sprintf("Block not found: %v", hash))
	}

	// When the verbose flag isn't set, simply return the serialized block
	// header as a hex-encoded string.
	if !verbose {
		var headerBuf bytes.Buffer
		err := blockHeader.Serialize(&headerBuf)
		if err != nil {
			context := "Failed to serialize block header"
			return nil, er.RpcInternalError(err.Error(), context)
		}
		return hex.EncodeToString(headerBuf.Bytes()), nil
	}

	best := api.bm.chain.BestSnapshot()

	// Get next block hash unless there are none.
	var nextHashString string
	confirmations := int64(-1)
	height := blockHeader.Height
	onMainChain, err := api.bm.chain.MainChainHasBlock(&hash)
	if onMainChain {
		if height < best.Height {
			nextHash, err := api.bm.chain.BlockHashByHeight(height + 1)
			if err != nil {
				context := "No next block"
				return nil, er.RpcInternalError(err.Error(),
					context)
			}
			nextHashString = nextHash.String()
		}
		confirmations = 1 + int64(best.Height - height)
	}

	blockHeaderReply := json.GetBlockHeaderVerboseResult{
		Hash:          hash.String(),
		Confirmations: confirmations,
		Version:       int32(blockHeader.Version),
		PreviousHash:  blockHeader.ParentRoot.String(),
		TxRoot:        blockHeader.TxRoot.String(),
		StateRoot:     blockHeader.StateRoot.String(),
		Difficulty:    blockHeader.Difficulty,
		Height:        uint32(height),
		Time:          blockHeader.Timestamp.Unix(),
		Nonce:         blockHeader.Nonce,
		NextHash:      nextHashString,
	}

	return blockHeaderReply, nil

}




