package blockchain

import (
	libp2p "github.com/libp2p/go-libp2p-peer"
)

func (self *BlockChain) OnBlockShardReceived(block *ShardBlock) {
	if self.newShardBlkCh[block.Header.ShardID] != nil {
		*self.newShardBlkCh[block.Header.ShardID] <- block
	}
}
func (self *BlockChain) OnBlockBeaconReceived(block *BeaconBlock) {
	if self.syncStatus.Beacon {
		self.newBeaconBlkCh <- block
	}
}

func (self *BlockChain) GetBeaconState() (*BeaconChainState, error) {
	state := &BeaconChainState{
		Height:    self.BestState.Beacon.BeaconHeight,
		BlockHash: self.BestState.Beacon.BestBlockHash,
	}
	return state, nil
}

func (self *BlockChain) OnBeaconStateReceived(state *BeaconChainState, shardsPoolState map[byte][]uint64, peerID libp2p.ID) {
	if self.syncStatus.Beacon {
		self.BeaconStateCh <- &PeerBeaconChainState{
			state, shardsPoolState, peerID,
		}
	}
}

func (self *BlockChain) GetShardState(shardID byte) *ShardChainState {
	state := &ShardChainState{
		Height:    self.BestState.Shard[shardID].ShardHeight,
		ShardID:   shardID,
		BlockHash: self.BestState.Shard[shardID].BestShardBlockHash,
	}
	return state
}

func (self *BlockChain) OnShardStateReceived(state *ShardChainState, crossShardsPoolState map[byte][]uint64, peerID libp2p.ID) {
	if self.newShardBlkCh[state.ShardID] != nil {
		self.ShardStateCh[state.ShardID] <- &PeerShardChainState{
			state, crossShardsPoolState, peerID,
		}
	}
}

func (self *BlockChain) OnShardToBeaconBlockReceived(block ShardToBeaconBlock) {
	self.config.ShardToBeaconPool.AddShardBeaconBlock(block)
}

func (self *BlockChain) OnCrossShardBlockReceived(block CrossShardBlock) {
	self.config.CrossShardPool.AddCrossShardBlock(block)

}
