package blockchain

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"sync"

	lru "github.com/hashicorp/golang-lru"
	"github.com/incognitochain/incognito-chain/blockchain/btc"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/rawdbv2"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/incdb"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/memcache"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/multiview"
	"github.com/incognitochain/incognito-chain/privacy"
	bnbrelaying "github.com/incognitochain/incognito-chain/relaying/bnb"
	btcrelaying "github.com/incognitochain/incognito-chain/relaying/btc"
	"github.com/incognitochain/incognito-chain/transaction"
	"github.com/pkg/errors"
)

type BlockChain struct {
	BeaconChain *BeaconChain
	ShardChain  []*ShardChain
	config      Config
	cQuitSync   chan struct{}

	IsTest bool

	beaconViewCache *lru.Cache
}

// Config is a descriptor which specifies the blockchain instance configuration.
type Config struct {
	BTCChain      *btcrelaying.BlockChain
	BNBChainState *bnbrelaying.BNBChainState
	DataBase      map[int]incdb.Database
	MemCache      *memcache.MemoryCache
	Interrupt     <-chan struct{}
	ChainParams   *Params
	GenesisParams *GenesisParams
	RelayShards   []byte
	// NodeMode          string
	BlockGen          *BlockGenerator
	TxPool            TxPool
	TempTxPool        TxPool
	CRemovedTxs       chan metadata.Transaction
	FeeEstimator      map[byte]FeeEstimator
	IsBlockGenStarted bool
	PubSubManager     Pubsub
	RandomClient      btc.RandomClient
	Syncker           Syncker
	Server            Server
	ConsensusEngine   ConsensusEngine
	Highway           Highway

	relayShardLck sync.Mutex
}

func NewBlockChain(config *Config, isTest bool) *BlockChain {
	bc := &BlockChain{}
	bc.config = *config
	bc.config.IsBlockGenStarted = false
	bc.IsTest = isTest
	bc.beaconViewCache, _ = lru.New(100)
	bc.cQuitSync = make(chan struct{})
	bc.GetBeaconBestState().Params = make(map[string]string)
	bc.GetBeaconBestState().ShardCommittee = make(map[byte][]incognitokey.CommitteePublicKey)
	bc.GetBeaconBestState().ShardPendingValidator = make(map[byte][]incognitokey.CommitteePublicKey)
	return bc
}

/*
Init - init a blockchain view from config
*/
func (blockchain *BlockChain) Init(config *Config) error {
	// Enforce required config fields.
	if config.DataBase == nil {
		return NewBlockChainError(UnExpectedError, errors.New("Database is not config"))
	}
	if config.ChainParams == nil {
		return NewBlockChainError(UnExpectedError, errors.New("Chain parameters is not config"))
	}
	blockchain.config = *config
	blockchain.config.IsBlockGenStarted = false
	blockchain.IsTest = false
	blockchain.beaconViewCache, _ = lru.New(100)
	// Initialize the chain state from the passed database.  When the db
	// does not yet contain any chain state, both it and the chain state
	// will be initialized to contain only the genesis block.
	if err := blockchain.InitChainState(); err != nil {
		return err
	}
	blockchain.cQuitSync = make(chan struct{})
	return nil
}

// InitChainState attempts to load and initialize the chain state from the
// database.  When the db does not yet contain any chain state, both it and the
// chain state are initialized to the genesis block.
func (blockchain *BlockChain) InitChainState() error {
	// Determine the state of the chain database. We may need to initialize
	// everything from scratch or upgrade certain buckets.
	blockchain.BeaconChain = NewBeaconChain(multiview.NewMultiView(), blockchain.config.BlockGen, blockchain, common.BeaconChainKey)
	var err error
	blockchain.BeaconChain.hashHistory, err = lru.New(1000)
	if err != nil {
		return err
	}
	if err := blockchain.RestoreBeaconViews(); err != nil {
		Logger.log.Error("debug restore beacon fail, init", err)
		err := blockchain.initBeaconState()
		if err != nil {
			Logger.log.Error("debug beacon state init error")
			return err
		}
	}
	Logger.log.Infof("Init Beacon View height %+v", blockchain.BeaconChain.GetBestView().GetHeight())

	//beaconHash, err := statedb.GetBeaconBlockHashByIndex(blockchain.GetBeaconBestState().GetBeaconConsensusStateDB(), 1)
	//panic(beaconHash.String())

	blockchain.ShardChain = make([]*ShardChain, blockchain.GetBeaconBestState().ActiveShards)
	for shard := 1; shard <= blockchain.GetBeaconBestState().ActiveShards; shard++ {
		shardID := byte(shard - 1)
		blockchain.ShardChain[shardID] = NewShardChain(shard-1, multiview.NewMultiView(), blockchain.config.BlockGen, blockchain, common.GetShardChainKey(shardID))
		blockchain.ShardChain[shardID].hashHistory, err = lru.New(1000)
		if err != nil {
			return err
		}
		if err := blockchain.RestoreShardViews(shardID); err != nil {
			Logger.log.Error("debug restore shard fail, init")
			err := blockchain.InitShardState(shardID)
			if err != nil {
				Logger.log.Error("debug shard state init error")
				return err
			}
		}
		Logger.log.Infof("Init Shard View shardID %+v, height %+v", shardID, blockchain.ShardChain[shardID].GetFinalViewHeight())
	}

	return nil
}

/*
// createChainState initializes both the database and the chain state to the
// genesis block.  This includes creating the necessary buckets and inserting
// the genesis block, so it must only be called on an uninitialized database.
*/
func (blockchain *BlockChain) InitShardState(shardID byte) error {
	initShardState := NewBestStateShardWithConfig(shardID, blockchain.config.ChainParams)
	// Create a new block from genesis block and set it as best block of chain
	initShardBlock := ShardBlock{}
	initShardBlock = *blockchain.config.ChainParams.GenesisShardBlock
	initShardBlock.Header.ShardID = shardID
	initShardBlockHeight := initShardBlock.Header.Height
	_, newShardCandidate := GetStakingCandidate(*blockchain.config.ChainParams.GenesisBeaconBlock)
	newShardCandidateStructs := []incognitokey.CommitteePublicKey{}
	for _, candidate := range newShardCandidate {
		key := incognitokey.CommitteePublicKey{}
		err := key.FromBase58(candidate)
		if err != nil {
			return err
		}
		newShardCandidateStructs = append(newShardCandidateStructs, key)
	}
	initShardState.ShardCommittee = append(initShardState.ShardCommittee, newShardCandidateStructs[int(shardID)*blockchain.config.ChainParams.MinShardCommitteeSize:(int(shardID)*blockchain.config.ChainParams.MinShardCommitteeSize)+blockchain.config.ChainParams.MinShardCommitteeSize]...)
	beaconBlocks, err := blockchain.GetBeaconBlockByHeight(initShardBlockHeight)
	if err != nil {
		return NewBlockChainError(FetchBeaconBlockError, err)
	}
	genesisBeaconBlock := beaconBlocks[0]

	err = initShardState.initShardBestState(blockchain, blockchain.GetShardChainDatabase(shardID), &initShardBlock, genesisBeaconBlock)
	if err != nil {
		return err
	}
	committeeChange := newCommitteeChange()
	committeeChange.shardCommitteeAdded[shardID] = initShardState.GetShardCommittee()

	err = blockchain.processStoreShardBlock(initShardState, &initShardBlock, committeeChange, []*BeaconBlock{genesisBeaconBlock})
	if err != nil {
		return err
	}

	return nil
}

func (blockchain *BlockChain) initBeaconState() error {
	initBeaconBestState := NewBeaconBestStateWithConfig(blockchain.config.ChainParams)
	initBlock := blockchain.config.ChainParams.GenesisBeaconBlock
	err := initBeaconBestState.initBeaconBestState(initBlock, blockchain, blockchain.GetBeaconChainDatabase())
	if err != nil {
		return err
	}
	initBlockHash := initBeaconBestState.BestBlock.Header.Hash()
	initBlockHeight := initBeaconBestState.BestBlock.Header.Height
	// Insert new block into beacon chain
	if err := statedb.StoreAllShardCommittee(initBeaconBestState.consensusStateDB, initBeaconBestState.ShardCommittee); err != nil {
		return err
	}
	if err := statedb.StoreBeaconCommittee(initBeaconBestState.consensusStateDB, initBeaconBestState.BeaconCommittee); err != nil {
		return err
	}

	if err := statedb.StoreStakerInfo(
		initBeaconBestState.consensusStateDB,
		initBeaconBestState.BeaconCommittee,
		initBeaconBestState.RewardReceiver,
		initBeaconBestState.AutoStaking.data,
		initBeaconBestState.StakingTx,
	); err != nil {
		return err
	}
	for _, committee := range initBeaconBestState.ShardCommittee {
		if err := statedb.StoreStakerInfo(
			initBeaconBestState.consensusStateDB,
			committee,
			initBeaconBestState.RewardReceiver,
			initBeaconBestState.AutoStaking.data,
			initBeaconBestState.StakingTx,
		); err != nil {
			return err
		}
	}
	consensusRootHash, err := initBeaconBestState.consensusStateDB.Commit(true)
	err = initBeaconBestState.consensusStateDB.Database().TrieDB().Commit(consensusRootHash, false)
	if err != nil {
		return err
	}
	initBeaconBestState.consensusStateDB.ClearObjects()
	if err := rawdbv2.StoreBeaconBlockByHash(blockchain.GetBeaconChainDatabase(), initBlockHash, &initBeaconBestState.BestBlock); err != nil {
		Logger.log.Error("Error store beacon block", initBeaconBestState.BestBlockHash, "in beacon chain")
		return err
	}
	rawdbv2.StoreFinalizedBeaconBlockHashByIndex(blockchain.GetBeaconChainDatabase(), initBlockHeight, initBlockHash)

	// State Root Hash
	bRH := BeaconRootHash{
		ConsensusStateDBRootHash: consensusRootHash,
		FeatureStateDBRootHash:   common.EmptyRoot,
		RewardStateDBRootHash:    common.EmptyRoot,
		SlashStateDBRootHash:     common.EmptyRoot,
	}
	initBeaconBestState.ConsensusStateDBRootHash = consensusRootHash
	if err := rawdbv2.StoreBeaconRootsHash(blockchain.GetBeaconChainDatabase(), initBlockHash, bRH); err != nil {
		return NewBlockChainError(StoreShardBlockError, err)
	}

	// Insert new block into beacon chain
	blockchain.BeaconChain.multiView.AddView(initBeaconBestState)
	if err := blockchain.BackupBeaconViews(blockchain.GetBeaconChainDatabase()); err != nil {
		Logger.log.Error("Error Store best state for block", blockchain.GetBeaconBestState().BestBlockHash, "in beacon chain")
		return NewBlockChainError(UnExpectedError, err)
	}

	return nil
}

func (blockchain *BlockChain) GetClonedBeaconBestState() (*BeaconBestState, error) {
	result := NewBeaconBestState()
	err := result.cloneBeaconBestStateFrom(blockchain.GetBeaconBestState())
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetReadOnlyShard - return a copy of Shard of BestState
func (blockchain *BlockChain) GetClonedAllShardBestState() map[byte]*ShardBestState {
	result := make(map[byte]*ShardBestState)
	for _, v := range blockchain.ShardChain {
		sidState := NewShardBestState()
		err := sidState.cloneShardBestStateFrom(blockchain.ShardChain[v.GetShardID()].GetBestState())
		if err != nil {
			return nil
		}
		result[byte(v.GetShardID())] = sidState
	}
	return result
}

// GetReadOnlyShard - return a copy of Shard of BestState
func (blockchain *BlockChain) GetClonedAShardBestState(shardID byte) (*ShardBestState, error) {
	result := NewShardBestState()
	err := result.cloneShardBestStateFrom(blockchain.ShardChain[int(shardID)].GetBestState())
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (blockchain *BlockChain) GetCurrentBeaconBlockHeight(shardID byte) uint64 {
	return blockchain.GetBeaconBestState().BestBlock.Header.Height
}

func (blockchain BlockChain) RandomCommitmentsProcess(usableInputCoins []*privacy.InputCoin, randNum int, shardID byte, tokenID *common.Hash) (commitmentIndexs []uint64, myCommitmentIndexs []uint64, commitments [][]byte) {
	param := transaction.NewRandomCommitmentsProcessParam(usableInputCoins, randNum, blockchain.GetBestStateShard(shardID).GetCopiedTransactionStateDB(), shardID, tokenID)
	return transaction.RandomCommitmentsProcess(param)
}

func (blockchain *BlockChain) GetActiveShardNumber() int {
	return blockchain.GetBeaconBestState().ActiveShards
}

func (blockchain *BlockChain) GetShardIDs() []int {
	shardIDs := []int{}
	for i := 0; i < blockchain.GetActiveShardNumber(); i++ {
		shardIDs = append(shardIDs, i)
	}
	return shardIDs
}

// -------------- Start of Blockchain retriever's implementation --------------
func (blockchain *BlockChain) SetIsBlockGenStarted(value bool) {
	blockchain.config.IsBlockGenStarted = value
}

func (blockchain *BlockChain) AddTxPool(txpool TxPool) {
	blockchain.config.TxPool = txpool
}

func (blockchain *BlockChain) AddTempTxPool(temptxpool TxPool) {
	blockchain.config.TempTxPool = temptxpool
}

func (blockchain *BlockChain) SetFeeEstimator(feeEstimator FeeEstimator, shardID byte) {
	if len(blockchain.config.FeeEstimator) == 0 {
		blockchain.config.FeeEstimator = make(map[byte]FeeEstimator)
	}
	blockchain.config.FeeEstimator[shardID] = feeEstimator
}

func (blockchain *BlockChain) InitChannelBlockchain(cRemovedTxs chan metadata.Transaction) {
	blockchain.config.CRemovedTxs = cRemovedTxs
}

// -------------- End of Blockchain retriever's implementation --------------

// -------------- Start of Blockchain BackUp And Restore --------------
func CalculateNumberOfByteToRead(amountBytes int) []byte {
	var result = make([]byte, 8)
	binary.LittleEndian.PutUint32(result, uint32(amountBytes))
	return result
}

func GetNumberOfByteToRead(value []byte) (int, error) {
	var result uint32
	err := binary.Read(bytes.NewBuffer(value), binary.LittleEndian, &result)
	if err != nil {
		return -1, err
	}
	return int(result), nil
}

func (blockchain *BlockChain) BackupShardChain(writer io.Writer, shardID byte) error {
	bestStateBytes, err := rawdbv2.GetShardBestState(blockchain.GetShardChainDatabase(shardID), shardID)
	if err != nil {
		return err
	}
	shardBestState := &ShardBestState{}
	err = json.Unmarshal(bestStateBytes, shardBestState)
	bestShardHeight := shardBestState.ShardHeight
	var i uint64
	for i = 1; i < bestShardHeight; i++ {
		shardBlocks, err := blockchain.GetShardBlockByHeight(i, shardID)
		if err != nil {
			return err
		}
		var shardBlock *ShardBlock
		for _, v := range shardBlocks {
			shardBlock = v
		}
		data, err := json.Marshal(shardBlocks)
		if err != nil {
			return err
		}
		_, err = writer.Write(CalculateNumberOfByteToRead(len(data)))
		if err != nil {
			return err
		}
		_, err = writer.Write(data)
		if err != nil {
			return err
		}
		if i%100 == 0 {
			Logger.log.Infof("Backup Shard %+v Block %+v", shardBlock.Header.ShardID, i)
		}
		if i == bestShardHeight-1 {
			Logger.log.Infof("Finish Backup Shard %+v with Block %+v", shardBlock.Header.ShardID, i)
		}
	}
	return nil
}

func (blockchain *BlockChain) BackupBeaconChain(writer io.Writer) error {
	bestStateBytes, err := rawdbv2.GetBeaconViews(blockchain.GetBeaconChainDatabase())
	if err != nil {
		return err
	}
	beaconBestState := &BeaconBestState{}
	err = json.Unmarshal(bestStateBytes, beaconBestState)
	bestBeaconHeight := beaconBestState.BeaconHeight
	var i uint64
	for i = 1; i < bestBeaconHeight; i++ {
		beaconBlocks, err := blockchain.GetBeaconBlockByHeight(i)
		if err != nil {
			return err
		}
		beaconBlock := beaconBlocks[0]
		data, err := json.Marshal(beaconBlock)
		if err != nil {
			return err
		}
		numOfByteToRead := CalculateNumberOfByteToRead(len(data))
		_, err = writer.Write(numOfByteToRead)
		if err != nil {
			return err
		}
		_, err = writer.Write(data)
		if err != nil {
			return err
		}
		if i%100 == 0 {
			Logger.log.Infof("Backup Beacon Block %+v", i)
		}
		if i == bestBeaconHeight-1 {
			Logger.log.Infof("Finish Backup Beacon with Block %+v", i)
		}
	}
	return nil
}

//TODO:
// current implement: backup all view data
// Optimize: backup view -> backup view hash instead of view
// restore: get view from hash and create new view, then insert into multiview
/*
Backup all BeaconView into Database
*/
func (blockchain *BlockChain) BackupBeaconViews(db incdb.KeyValueWriter) error {
	allViews := []*BeaconBestState{}
	for _, v := range blockchain.BeaconChain.multiView.GetAllViewsWithBFS() {
		allViews = append(allViews, v.(*BeaconBestState))
	}
	b, _ := json.Marshal(allViews)
	return rawdbv2.StoreBeaconViews(db, b)
}

/*
Restart all BeaconView from Database
*/
func (blockchain *BlockChain) RestoreBeaconViews() error {
	allViews := []*BeaconBestState{}
	bcDB := blockchain.GetBeaconChainDatabase()
	b, err := rawdbv2.GetBeaconViews(bcDB)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, &allViews)
	if err != nil {
		return err
	}
	sID := []int{}
	for i := 0; i < blockchain.config.ChainParams.ActiveShards; i++ {
		sID = append(sID, i)
	}

	blockchain.BeaconChain.multiView.Reset()
	for _, v := range allViews {
		v.RestoreBeaconViewStateFromHash(blockchain)
		beaconConsensusStateDB, err := statedb.NewWithPrefixTrie(v.ConsensusStateDBRootHash, statedb.NewDatabaseAccessWarper(bcDB))
		if err != nil {
			return NewBlockChainError(BeaconError, err)
		}
		v.AutoStaking = NewMapStringBool()
		v.AutoStaking.data = statedb.GetMapAutoStaking(beaconConsensusStateDB, sID)
		// finish reproduce
		if !blockchain.BeaconChain.multiView.AddView(v) {
			panic("Restart beacon views fail")
		}
	}
	return nil
}

/*
Backup shard views
*/
func (blockchain *BlockChain) BackupShardViews(db incdb.KeyValueWriter, shardID byte) error {
	allViews := []*ShardBestState{}
	for _, v := range blockchain.ShardChain[shardID].multiView.GetAllViewsWithBFS() {
		allViews = append(allViews, v.(*ShardBestState))
	}
	// fmt.Println("debug BackupShardViews", len(allViews))
	return rawdbv2.StoreShardBestState(db, shardID, allViews)
}

/*
Restart all BeaconView from Database
*/
func (blockchain *BlockChain) RestoreShardViews(shardID byte) error {
	allViews := []*ShardBestState{}
	b, err := rawdbv2.GetShardBestState(blockchain.GetShardChainDatabase(shardID), shardID)
	if err != nil {
		fmt.Println("debug Cannot see shard best state")
		return err
	}
	err = json.Unmarshal(b, &allViews)
	if err != nil {
		fmt.Println("debug Cannot unmarshall shard best state", string(b))
		return err
	}
	// fmt.Println("debug RestoreShardViews", len(allViews))
	blockchain.ShardChain[shardID].multiView.Reset()

	for _, v := range allViews {
		block, _, err := blockchain.GetShardBlockByHash(v.BestBlockHash)
		if err != nil || block == nil {
			fmt.Println("block ", block)
			panic(err)
		}
		v.BestBlock = block

		err = v.InitStateRootHash(blockchain.GetShardChainDatabase(shardID), blockchain)
		if err != nil {
			panic(err)
		}

		err = v.RestoreCommittee(shardID, blockchain)
		if err != nil {
			panic(err)
		}

		v.StakingTx = NewMapStringString()
		v.StakingTx.data, err = blockchain.GetShardStakingTx(v)
		if err != nil {
			panic(err)
		}

		err = v.RestorePendingValidators(shardID, blockchain)
		if err != nil {
			panic(err)
		}

		if !blockchain.ShardChain[shardID].multiView.AddView(v) {
			panic("Restart shard views fail")
		}
	}
	return nil
}

func (blockchain *BlockChain) GetShardStakingTx(shardView *ShardBestState) (map[string]string, error) {
	shardID := shardView.ShardID
	//build staking tx
	beaconConsensusRootHash, err := blockchain.GetBeaconConsensusRootHash(blockchain.GetBeaconBestState(), shardView.BeaconHeight)
	if err != nil {
		Logger.log.Error("Cannot restore shard, beacon not ready!")
		return nil, err
	}

	beaconConsensusStateDB, err := statedb.NewWithPrefixTrie(beaconConsensusRootHash, statedb.NewDatabaseAccessWarper(blockchain.GetBeaconChainDatabase()))
	mapStakingTx, err := beaconConsensusStateDB.GetAllStakingTX(blockchain.GetShardIDs())

	if err != nil {
		fmt.Println(err)
		panic("Something wrong when retrieve mapStakingTx")
	}

	sdb := blockchain.GetShardChainDatabase(byte(shardID))
	shardStakingTx := map[string]string{}
	for _, stakingtx := range mapStakingTx {
		if stakingtx != common.HashH([]byte{0}).String() {
			stakingTxHash, _ := common.Hash{}.NewHashFromStr(stakingtx)
			blockHash, txindex, err := rawdbv2.GetTransactionByHash(sdb, *stakingTxHash)
			if err != nil { //no transaction in this node
				continue
			}
			shardBlockBytes, err := rawdbv2.GetShardBlockByHash(sdb, blockHash)
			if err != nil { //no transaction in this node
				panic("Have transaction but cannot found block")
			}
			shardBlock := NewShardBlock()
			err = json.Unmarshal(shardBlockBytes, shardBlock)
			if err != nil {
				panic("Cannot unmarshal shardblock")
			}
			if shardBlock.GetShardID() != int(shardID) {
				continue
			}
			txData := shardBlock.Body.Transactions[txindex]
			committeePk := txData.GetMetadata().(*metadata.StakingMetadata).CommitteePublicKey
			shardStakingTx[committeePk] = stakingtx
		}
	}
	return shardStakingTx, nil
}

// -------------- End of Blockchain BackUp And Restore --------------

// func (blockchain *BlockChain) GetNodeMode() string {
// 	return blockchain.config.NodeMode
// }

func (blockchain *BlockChain) GetWantedShard(isBeaconCommittee bool) map[byte]struct{} {
	res := map[byte]struct{}{}
	if isBeaconCommittee {
		for sID := byte(0); sID < byte(blockchain.config.ChainParams.ActiveShards); sID++ {
			res[sID] = struct{}{}
		}
	} else {
		blockchain.config.relayShardLck.Lock()
		for _, sID := range blockchain.config.RelayShards {
			res[sID] = struct{}{}
		}
		blockchain.config.relayShardLck.Unlock()
	}
	return res
}

// GetConfig returns blockchain's config
func (blockchain *BlockChain) GetConfig() *Config {
	return &blockchain.config
}

// GetPortalParams returns portal params in beaconheight
func (blockchain *BlockChain) GetPortalParams(beaconHeight uint64) PortalParams {
	portalParamMap := blockchain.GetConfig().ChainParams.PortalParams
	// only has one value - default value
	if len(portalParamMap) == 1 {
		return portalParamMap[0]
	}

	bchs := []uint64{}
	for bch := range portalParamMap {
		bchs = append(bchs, bch)
	}
	sort.Slice(bchs, func(i, j int) bool {
		return bchs[i] < bchs[j]
	})

	bchKey := bchs[len(bchs)-1]
	for i := len(bchs) - 1; i >= 0; i-- {
		if beaconHeight < bchs[i] {
			continue
		}
		bchKey = bchs[i]
		break
	}

	return portalParamMap[bchKey]
}

func (blockchain *BlockChain) GetSupportedCollateralTokenIDs(beaconHeight uint64) []string {
	portalParams := blockchain.GetPortalParams(beaconHeight)
	tokenIDs := []string{}
	for _, col := range portalParams.SupportedCollateralTokens {
		tokenIDs = append(tokenIDs, col.ExternalTokenID)
	}
	return tokenIDs
}

func (blockchain *BlockChain) GetSupportedCollateralInfo(beaconHeight uint64) []PortalCollateral {
	portalParams := blockchain.GetPortalParams(beaconHeight)
	return portalParams.SupportedCollateralTokens
}

func (blockchain *BlockChain) GetPortalETHContractAddrStr() string {
	return blockchain.GetConfig().ChainParams.PortalETHContractAddressStr
}

func (blockchain *BlockChain) GetBeaconChainDatabase() incdb.Database {
	return blockchain.config.DataBase[common.BeaconChainDataBaseID]
}

func (blockchain *BlockChain) GetShardChainDatabase(shardID byte) incdb.Database {
	return blockchain.config.DataBase[int(shardID)]
}

func (blockchain *BlockChain) GetBeaconViewStateDataFromBlockHash(blockHash common.Hash) (*BeaconBestState, error) {
	v, ok := blockchain.beaconViewCache.Get(blockHash)
	if ok {
		return v.(*BeaconBestState), nil
	}
	bcDB := blockchain.GetBeaconChainDatabase()
	rootHash, err := rawdbv2.GetBeaconRootsHash(bcDB, blockHash)
	if err != nil {
		return nil, err
	}
	bRH := &BeaconRootHash{}
	err = json.Unmarshal(rootHash, bRH)
	if err != nil {
		return nil, err
	}

	beaconView := &BeaconBestState{
		BestBlockHash:            blockHash,
		ActiveShards:             blockchain.config.ChainParams.ActiveShards, //we assume active shard not change (if not, we must store active shard in db)
		ConsensusStateDBRootHash: bRH.ConsensusStateDBRootHash,
		FeatureStateDBRootHash:   bRH.FeatureStateDBRootHash,
		RewardStateDBRootHash:    bRH.RewardStateDBRootHash,
		SlashStateDBRootHash:     bRH.SlashStateDBRootHash,
	}

	err = beaconView.RestoreBeaconViewStateFromHash(blockchain)
	if err != nil {
		Logger.log.Error(err)
	}
	sID := []int{}
	for i := 0; i < blockchain.config.ChainParams.ActiveShards; i++ {
		sID = append(sID, i)
	}
	beaconConsensusStateDB, err := statedb.NewWithPrefixTrie(beaconView.ConsensusStateDBRootHash, statedb.NewDatabaseAccessWarper(bcDB))
	if err != nil {
		return nil, NewBlockChainError(BeaconError, err)
	}
	beaconView.AutoStaking = NewMapStringBool()
	beaconView.AutoStaking.data = statedb.GetMapAutoStaking(beaconConsensusStateDB, sID)
	blockchain.beaconViewCache.Add(blockHash, beaconView)
	return beaconView, err
}

// GetFixedRandomForShardIDCommitment returns the fixed randomness for shardID commitments
// if bc height is greater than or equal to BCHeightBreakPointFixRandShardCM
// otherwise, return nil
func (blockchain *BlockChain) GetFixedRandomForShardIDCommitment(beaconHeight uint64) *privacy.Scalar {
	if beaconHeight == 0 {
		beaconHeight = blockchain.GetBeaconBestState().GetHeight()
	}
	if beaconHeight >= blockchain.GetConfig().ChainParams.BCHeightBreakPointNewZKP {
		return privacy.FixedRandomnessShardID
	}

	return nil
}

func (blockchain *BlockChain) IsAfterNewZKPCheckPoint(beaconHeight uint64) bool {
	if beaconHeight == 0 {
		beaconHeight = blockchain.GetBeaconBestState().GetHeight()
	}

	return beaconHeight >= blockchain.GetConfig().ChainParams.BCHeightBreakPointNewZKP
}

func (s *BlockChain) GetChainParams() *Params {
	return s.config.ChainParams
}

func (s *BlockChain) AddRelayShard(sid int) error {
	s.config.relayShardLck.Lock()
	for _, shard := range s.config.RelayShards {
		if shard == byte(sid) {
			s.config.relayShardLck.Unlock()
			return errors.New("already relay this shard" + strconv.Itoa(sid))
		}
	}
	s.config.RelayShards = append(s.config.RelayShards, byte(sid))
	s.config.relayShardLck.Unlock()
	return nil
}

func (s *BlockChain) RemoveRelayShard(sid int) {
	s.config.relayShardLck.Lock()
	for idx, shard := range s.config.RelayShards {
		if shard == byte(sid) {
			s.config.RelayShards = append(s.config.RelayShards[:idx], s.config.RelayShards[idx+1:]...)
			break
		}
	}
	s.config.relayShardLck.Unlock()
	return
}
