package syncker

import (
	"github.com/incognitochain/incognito-chain/blockchain"
	"reflect"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
)

const RUNNING_SYNC = "running_sync"
const STOP_SYNC = "stop_sync"

func isNil(v interface{}) bool {
	return v == nil || (reflect.ValueOf(v).Kind() == reflect.Ptr && reflect.ValueOf(v).IsNil())
}

func InsertBatchBlock(chain Chain, blocks []common.BlockInterface) (int, error) {
	sameCommitteeBlock := blocks

	containSwap := func(inst [][]string) bool {
		for _, inst := range inst {
			if inst[0] == blockchain.SwapAction {
				return true
			}
		}
		return false
	}

	//loop through block, to get same committee
	for i, v := range blocks {
		shouldBreak := false
		switch v.(type) {
		case *blockchain.BeaconBlock:
			// do nothing, beacon committee assume not change
			//if v.GetCurrentEpoch() == curEpoch+1 {
			//	sameCommitteeBlock = blocks[:i+1]
			//	break
			//}
		case *blockchain.ShardBlock:
			//if block contain swap inst,
			if containSwap(v.(*blockchain.ShardBlock).Body.Instructions) {
				sameCommitteeBlock = blocks[:i+1]
				shouldBreak = true
			}
		}
		if shouldBreak {
			break
		}
	}

	for i, blk := range sameCommitteeBlock {
		if i == len(sameCommitteeBlock)-1 {
			break
		}
		if blk.GetHeight() != sameCommitteeBlock[i+1].GetHeight()-1 {
			sameCommitteeBlock = blocks[:i+1]
			break
		}
	}
	//validate the last block for batching
	epochCommittee := chain.GetCommittee()
	validBlockForInsert := sameCommitteeBlock[:]
	for i := len(sameCommitteeBlock) - 1; i >= 0; i-- {
		if err := chain.ValidateBlockSignatures(sameCommitteeBlock[i], epochCommittee); err != nil {
			validBlockForInsert = sameCommitteeBlock[:i]
		} else {
			break
		}
	}

	batchingValidate := true
	//if no valid block, this could be a fork chain, or the chunks that have old committee (current best block have swap) => try to insert all with full validation
	if len(validBlockForInsert) == 0 {
		validBlockForInsert = sameCommitteeBlock[:]
		batchingValidate = false
	}

	for i, v := range validBlockForInsert {
		if !chain.CheckExistedBlk(v) {
			var err error
			if i == 0 {
				err = chain.InsertBlk(v, true)
			} else {
				err = chain.InsertBlk(v, batchingValidate == false)
			}
			if err != nil {
				committeeStr, _ := incognitokey.CommitteeKeyListToString(epochCommittee)
				Logger.Errorf("Insert block %v hash %v got error %v, Committee of epoch %v", v.GetHeight(), v.Hash(), err, committeeStr)
				return 0, err
			}
		}
	}
	return len(validBlockForInsert), nil
}

//final block
func GetFinalBlockFromBlockHash_v1(currentFinalHash string, byHash map[string]common.BlockPoolInterface, byPrevHash map[string][]string) (res []common.BlockPoolInterface) {
	var finalBlock common.BlockPoolInterface = nil
	var traverse func(currentHash string)
	traverse = func(currentHash string) {
		if byPrevHash[currentHash] == nil {
			return
		} else {
			if finalBlock == nil {
				finalBlock = byHash[currentHash]
			} else if finalBlock.GetHeight() < byHash[currentHash].GetHeight() {
				finalBlock = byHash[currentHash]
			}
			for _, nextHash := range byPrevHash[currentHash] {
				traverse(nextHash)
			}
		}
	}
	traverse(currentFinalHash)

	if finalBlock == nil {
		return nil
	}

	for {
		if currentFinalHash == finalBlock.Hash().String() {
			return
		}
		res = append([]common.BlockPoolInterface{byHash[finalBlock.Hash().String()]}, res...)
		finalBlock = byHash[finalBlock.GetPrevHash().String()]
		if finalBlock == nil || finalBlock.Hash().String() == currentFinalHash {
			break
		}
	}
	return res
}

func GetLongestChain(currentFinalHash string, byHash map[string]common.BlockPoolInterface, byPrevHash map[string][]string) (res []common.BlockPoolInterface) {
	var finalBlock common.BlockPoolInterface = nil
	var traverse func(currentHash string)
	traverse = func(currentHash string) {
		if byPrevHash[currentHash] == nil {
			if finalBlock == nil {
				finalBlock = byHash[currentHash]
			} else if finalBlock.GetHeight() < byHash[currentHash].GetHeight() {
				finalBlock = byHash[currentHash]
			}
			return
		} else {

			for _, nextHash := range byPrevHash[currentHash] {
				traverse(nextHash)
			}
		}
	}
	traverse(currentFinalHash)

	if finalBlock == nil {
		return nil
	}

	for {
		res = append([]common.BlockPoolInterface{byHash[finalBlock.Hash().String()]}, res...)
		finalBlock = byHash[finalBlock.GetPrevHash().String()]
		if finalBlock == nil {
			break
		}
	}
	return res
}

func GetPoolInfo(byHash map[string]common.BlockPoolInterface) (res []common.BlockPoolInterface) {
	for _, v := range byHash {
		res = append(res, v)
	}
	return res
}

func compareLists(poolList map[byte][]interface{}, hashList map[byte][]common.Hash) (diffHashes map[byte][]common.Hash) {
	diffHashes = make(map[byte][]common.Hash)
	poolListsHash := make(map[byte][]common.Hash)
	for shardID, blkList := range poolList {
		for _, blk := range blkList {
			blkHash := blk.(common.BlockPoolInterface).Hash()
			poolListsHash[shardID] = append(poolListsHash[shardID], *blkHash)
		}
	}

	for shardID, blockHashes := range hashList {
		if blockList, ok := poolListsHash[shardID]; ok {
			for _, blockHash := range blockHashes {
				if exist, _ := common.SliceExists(blockList, blockHash); !exist {
					diffHashes[shardID] = append(diffHashes[shardID], blockHash)
				}
			}
		} else {
			diffHashes[shardID] = blockHashes
		}
	}
	return diffHashes
}

func compareListsByHeight(poolList map[byte][]interface{}, heightList map[byte][]uint64) (diffHeights map[byte][]uint64) {
	diffHeights = make(map[byte][]uint64)
	poolListsHeight := make(map[byte][]uint64)
	for shardID, blkList := range poolList {
		for _, blk := range blkList {
			blkHeight := blk.(common.BlockPoolInterface).GetHeight()
			poolListsHeight[shardID] = append(poolListsHeight[shardID], blkHeight)
		}
	}

	for shardID, blockHeights := range heightList {
		if blockList, ok := poolListsHeight[shardID]; ok {
			for _, height := range blockHeights {
				if exist, _ := common.SliceExists(blockList, height); !exist {
					diffHeights[shardID] = append(diffHeights[shardID], height)
				}
			}
		} else {
			diffHeights[shardID] = blockHeights
		}
	}
	return diffHeights
}

func GetBlksByPrevHash(
	prevHash string,
	byHash map[string]common.BlockPoolInterface,
	byPrevHash map[string][]string,
) (
	res []common.BlockPoolInterface,
) {
	if hashes, ok := byPrevHash[prevHash]; ok {
		for _, hash := range hashes {
			if blk, exist := byHash[hash]; exist {
				res = append(res, blk)
			}
		}
	}
	return res
}

func GetAllViewFromHash(
	rHash string,
	byHash map[string]common.BlockPoolInterface,
	byPrevHash map[string][]string,
) (
	res []common.BlockPoolInterface,
) {
	hashes := []string{rHash}
	for {
		if len(hashes) == 0 {
			return res
		}
		hash := hashes[0]
		hashes = hashes[1:]
		for h, blk := range byHash {
			if blk.GetPrevHash().String() == hash {
				hashes = append(hashes, h)
				res = append(res, blk)
			}
		}
	}
}
