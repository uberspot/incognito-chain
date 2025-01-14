package blockchain

import (
	"time"

	"github.com/incognitochain/incognito-chain/common"
)

type SlashLevel struct {
	MinRange        uint8
	PunishedEpoches uint8
}
type PortalCollateral struct {
	ExternalTokenID string
	Decimal         uint8
}
type PortalParams struct {
	TimeOutCustodianReturnPubToken       time.Duration
	TimeOutWaitingPortingRequest         time.Duration
	TimeOutWaitingRedeemRequest          time.Duration
	MaxPercentLiquidatedCollateralAmount uint64
	MaxPercentCustodianRewards           uint64
	MinPercentCustodianRewards           uint64
	MinLockCollateralAmountInEpoch       uint64
	MinPercentLockedCollateral           uint64
	TP120                                uint64
	TP130                                uint64
	MinPercentPortingFee                 float64
	MinPercentRedeemFee                  float64
	SupportedCollateralTokens            []PortalCollateral
	MinPortalFee                         uint64 // nano PRV
	MinUnlockOverRateCollaterals         uint64
}

/*
Params defines a network by its component. These component may be used by Applications
to differentiate network as well as addresses and keys for one network
from those intended for use on another network
*/
type Params struct {
	Name                             string // Name defines a human-readable identifier for the network.
	Net                              uint32 // Net defines the magic bytes used to identify the network.
	DefaultPort                      string // DefaultPort defines the default peer-to-peer port for the network.
	GenesisParams                    *GenesisParams
	MaxShardCommitteeSize            int
	MinShardCommitteeSize            int
	MaxBeaconCommitteeSize           int
	MinBeaconCommitteeSize           int
	MinShardBlockInterval            time.Duration
	MaxShardBlockCreation            time.Duration
	MinBeaconBlockInterval           time.Duration
	MaxBeaconBlockCreation           time.Duration
	NumberOfFixedBlockValidators     int
	StakingAmountShard               uint64
	ActiveShards                     int
	GenesisBeaconBlock               *BeaconBlock // GenesisBlock defines the first block of the chain.
	GenesisShardBlock                *ShardBlock  // GenesisBlock defines the first block of the chain.
	BasicReward                      uint64
	Epoch                            uint64
	RandomTime                       uint64
	SlashLevels                      []SlashLevel
	EthContractAddressStr            string // smart contract of ETH for bridge
	Offset                           int    // default offset for swap policy, is used for cases that good producers length is less than max committee size
	SwapOffset                       int    // is used for case that good producers length is equal to max committee size
	IncognitoDAOAddress              string
	CentralizedWebsitePaymentAddress string //centralized website's pubkey
	CheckForce                       bool   // true on testnet and false on mainnet
	ChainVersion                     string
	AssignOffset                     int
	ConsensusV2Epoch                 uint64
	Timeslot                         uint64
	BeaconHeightBreakPointBurnAddr   uint64
	BNBRelayingHeaderChainID         string
	BTCRelayingHeaderChainID         string
	BTCDataFolderName                string
	BNBFullNodeProtocol              string
	BNBFullNodeHost                  string
	BNBFullNodePort                  string
	PortalParams                     map[uint64]PortalParams
	PortalTokens                     map[string]PortalTokenProcessor
	PortalFeederAddress              string
	EpochBreakPointSwapNewKey        []uint64
	IsBackup                         bool
	PreloadAddress                   string
	ReplaceStakingTxHeight           uint64
	ETHRemoveBridgeSigEpoch          uint64
	BCHeightBreakPointNewZKP         uint64
	PortalETHContractAddressStr      string // smart contract of ETH for portal
	BCHeightBreakPointPortalV3       uint64
}

type GenesisParams struct {
	InitialIncognito                            []string // init tx for genesis block
	FeePerTxKb                                  uint64
	PreSelectBeaconNodeSerializedPubkey         []string
	SelectBeaconNodeSerializedPubkeyV2          map[uint64][]string
	PreSelectBeaconNodeSerializedPaymentAddress []string
	SelectBeaconNodeSerializedPaymentAddressV2  map[uint64][]string
	PreSelectBeaconNode                         []string
	PreSelectShardNodeSerializedPubkey          []string
	SelectShardNodeSerializedPubkeyV2           map[uint64][]string
	PreSelectShardNodeSerializedPaymentAddress  []string
	SelectShardNodeSerializedPaymentAddressV2   map[uint64][]string
	PreSelectShardNode                          []string
	ConsensusAlgorithm                          string
}

var ChainTestParam = Params{}
var ChainTest2Param = Params{}
var ChainMainParam = Params{}

var genesisParamsTestnetNew *GenesisParams
var genesisParamsTestnet2New *GenesisParams
var genesisParamsMainnetNew *GenesisParams
var GenesisParam *GenesisParams

func initPortalTokensForTestNet() map[string]PortalTokenProcessor {
	return map[string]PortalTokenProcessor{
		common.PortalBTCIDStr: &PortalBTCTokenProcessor{
			&PortalToken{
				ChainID: TestnetBTCChainID,
			},
		},
		common.PortalBNBIDStr: &PortalBNBTokenProcessor{
			&PortalToken{
				ChainID: TestnetBNBChainID,
			},
		},
	}
}

func initPortalTokensForMainNet() map[string]PortalTokenProcessor {
	return map[string]PortalTokenProcessor{
		common.PortalBTCIDStr: &PortalBTCTokenProcessor{
			&PortalToken{
				ChainID: MainnetBTCChainID,
			},
		},
		common.PortalBNBIDStr: &PortalBNBTokenProcessor{
			&PortalToken{
				ChainID: MainnetBNBChainID,
			},
		},
	}
}

// external tokenID there is no 0x prefix, in lower case
// @@Note: need to update before deploying
func getSupportedPortalCollateralsMainnet() []PortalCollateral {
	return []PortalCollateral{
		{"0000000000000000000000000000000000000000", 9}, // eth
		{"dac17f958d2ee523a2206206994597c13d831ec7", 6}, // usdt
		{"a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", 6}, // usdc
	}
}

// external tokenID there is no 0x prefix, in lower case
// @@Note: need to update before deploying
func getSupportedPortalCollateralsTestnet() []PortalCollateral {
	return []PortalCollateral{
		{"0000000000000000000000000000000000000000", 9}, // eth
		{"3a829f4b97660d970428cd370c4e41cbad62092b", 6}, // usdt, kovan testnet
		{"75b0622cec14130172eae9cf166b92e5c112faff", 6}, // usdc, kovan testnet
	}
}

// external tokenID there is no 0x prefix, in lower case
// @@Note: need to update before deploying
func getSupportedPortalCollateralsTestnet2() []PortalCollateral {
	return []PortalCollateral{
		{"0000000000000000000000000000000000000000", 9}, // eth
		{"3a829f4b97660d970428cd370c4e41cbad62092b", 6}, // usdt, kovan testnet
		{"75b0622cec14130172eae9cf166b92e5c112faff", 6}, // usdc, kovan testnet
	}
}

func SetupParam() {
	// FOR TESTNET
	genesisParamsTestnetNew = &GenesisParams{
		PreSelectBeaconNodeSerializedPubkey:         PreSelectBeaconNodeTestnetSerializedPubkey,
		PreSelectBeaconNodeSerializedPaymentAddress: PreSelectBeaconNodeTestnetSerializedPaymentAddress,
		PreSelectShardNodeSerializedPubkey:          PreSelectShardNodeTestnetSerializedPubkey,
		PreSelectShardNodeSerializedPaymentAddress:  PreSelectShardNodeTestnetSerializedPaymentAddress,
		SelectBeaconNodeSerializedPubkeyV2:          SelectBeaconNodeTestnetSerializedPubkeyV2,
		SelectBeaconNodeSerializedPaymentAddressV2:  SelectBeaconNodeTestnetSerializedPaymentAddressV2,
		SelectShardNodeSerializedPubkeyV2:           SelectShardNodeTestnetSerializedPubkeyV2,
		SelectShardNodeSerializedPaymentAddressV2:   SelectShardNodeTestnetSerializedPaymentAddressV2,
		//@Notice: InitTxsForBenchmark is for testing and testparams only
		//InitialIncognito: IntegrationTestInitPRV,
		InitialIncognito:   TestnetInitPRV,
		ConsensusAlgorithm: common.BlsConsensus,
	}
	ChainTestParam = Params{
		Name:                   TestnetName,
		Net:                    Testnet,
		DefaultPort:            TestnetDefaultPort,
		GenesisParams:          genesisParamsTestnetNew,
		MaxShardCommitteeSize:  TestNetShardCommitteeSize,     //TestNetShardCommitteeSize,
		MinShardCommitteeSize:  TestNetMinShardCommitteeSize,  //TestNetShardCommitteeSize,
		MaxBeaconCommitteeSize: TestNetBeaconCommitteeSize,    //TestNetBeaconCommitteeSize,
		MinBeaconCommitteeSize: TestNetMinBeaconCommitteeSize, //TestNetBeaconCommitteeSize,
		StakingAmountShard:     TestNetStakingAmountShard,
		ActiveShards:           TestNetActiveShards,
		// blockChain parameters
		// GenesisBeaconBlock:               CreateGenesisBeaconBlock(1, Testnet, TestnetGenesisBlockTime, genesisParamsTestnetNew),
		// GenesisShardBlock:                CreateGenesisShardBlock(1, Testnet, TestnetGenesisBlockTime, genesisParamsTestnetNew),
		MinShardBlockInterval:            TestNetMinShardBlkInterval,
		MaxShardBlockCreation:            TestNetMaxShardBlkCreation,
		MinBeaconBlockInterval:           TestNetMinBeaconBlkInterval,
		MaxBeaconBlockCreation:           TestNetMaxBeaconBlkCreation,
		NumberOfFixedBlockValidators:     4,
		BasicReward:                      TestnetBasicReward,
		Epoch:                            TestnetEpoch,
		RandomTime:                       TestnetRandomTime,
		Offset:                           TestnetOffset,
		AssignOffset:                     TestnetAssignOffset,
		SwapOffset:                       TestnetSwapOffset,
		EthContractAddressStr:            TestnetETHContractAddressStr,
		IncognitoDAOAddress:              TestnetIncognitoDAOAddress,
		CentralizedWebsitePaymentAddress: TestnetCentralizedWebsitePaymentAddress,
		SlashLevels:                      []SlashLevel{
			//SlashLevel{MinRange: 20, PunishedEpoches: 1},
			//SlashLevel{MinRange: 50, PunishedEpoches: 2},
			//SlashLevel{MinRange: 75, PunishedEpoches: 3},
		},
		CheckForce:                     false,
		ChainVersion:                   "version-chain-test.json",
		ConsensusV2Epoch:               16930,
		Timeslot:                       10,
		BeaconHeightBreakPointBurnAddr: 250000,
		BNBRelayingHeaderChainID:       TestnetBNBChainID,
		BTCRelayingHeaderChainID:       TestnetBTCChainID,
		BTCDataFolderName:              TestnetBTCDataFolderName,
		BNBFullNodeProtocol:            TestnetBNBFullNodeProtocol,
		BNBFullNodeHost:                TestnetBNBFullNodeHost,
		BNBFullNodePort:                TestnetBNBFullNodePort,
		PortalFeederAddress:            TestnetPortalFeeder,
		PortalParams: map[uint64]PortalParams{
			0: {
				TimeOutCustodianReturnPubToken:       15 * time.Minute,
				TimeOutWaitingPortingRequest:         15 * time.Minute,
				TimeOutWaitingRedeemRequest:          10 * time.Minute,
				MaxPercentLiquidatedCollateralAmount: 105,
				MaxPercentCustodianRewards:           10, // todo: need to be updated before deploying
				MinPercentCustodianRewards:           1,
				MinLockCollateralAmountInEpoch:       10000 * 1e9, // 10000 usd
				MinPercentLockedCollateral:           150,
				TP120:                                120,
				TP130:                                130,
				MinPercentPortingFee:                 0.01,
				MinPercentRedeemFee:                  0.01,
				SupportedCollateralTokens:            getSupportedPortalCollateralsTestnet(), // todo: need to be updated before deploying
				MinPortalFee:                         100,
				MinUnlockOverRateCollaterals:         25,
			},
		},
		PortalTokens:                initPortalTokensForTestNet(),
		EpochBreakPointSwapNewKey:   TestnetReplaceCommitteeEpoch,
		ReplaceStakingTxHeight:      1,
		IsBackup:                    false,
		PreloadAddress:              "",
		BCHeightBreakPointNewZKP:    2300000, //TODO: change this value when deployed testnet
		ETHRemoveBridgeSigEpoch:     21920,

		PortalETHContractAddressStr: "0x6D53de7aFa363F779B5e125876319695dC97171E", // todo: update sc address
		BCHeightBreakPointPortalV3:  30158,
	}
	// END TESTNET

	// FOR TESTNET-2
	genesisParamsTestnet2New = &GenesisParams{
		PreSelectBeaconNodeSerializedPubkey:         PreSelectBeaconNodeTestnetSerializedPubkey,
		PreSelectBeaconNodeSerializedPaymentAddress: PreSelectBeaconNodeTestnetSerializedPaymentAddress,
		PreSelectShardNodeSerializedPubkey:          PreSelectShardNodeTestnetSerializedPubkey,
		PreSelectShardNodeSerializedPaymentAddress:  PreSelectShardNodeTestnetSerializedPaymentAddress,
		SelectBeaconNodeSerializedPubkeyV2:          SelectBeaconNodeTestnetSerializedPubkeyV2,
		SelectBeaconNodeSerializedPaymentAddressV2:  SelectBeaconNodeTestnetSerializedPaymentAddressV2,
		SelectShardNodeSerializedPubkeyV2:           SelectShardNodeTestnetSerializedPubkeyV2,
		SelectShardNodeSerializedPaymentAddressV2:   SelectShardNodeTestnetSerializedPaymentAddressV2,
		//@Notice: InitTxsForBenchmark is for testing and testparams only
		//InitialIncognito: IntegrationTestInitPRV,
		InitialIncognito:   TestnetInitPRV,
		ConsensusAlgorithm: common.BlsConsensus,
	}
	ChainTest2Param = Params{
		Name:                   Testnet2Name,
		Net:                    Testnet2,
		DefaultPort:            Testnet2DefaultPort,
		GenesisParams:          genesisParamsTestnet2New,
		MaxShardCommitteeSize:  TestNet2ShardCommitteeSize,     //TestNetShardCommitteeSize,
		MinShardCommitteeSize:  TestNet2MinShardCommitteeSize,  //TestNetShardCommitteeSize,
		MaxBeaconCommitteeSize: TestNet2BeaconCommitteeSize,    //TestNetBeaconCommitteeSize,
		MinBeaconCommitteeSize: TestNet2MinBeaconCommitteeSize, //TestNetBeaconCommitteeSize,
		StakingAmountShard:     TestNet2StakingAmountShard,
		ActiveShards:           TestNet2ActiveShards,
		// blockChain parameters
		// GenesisBeaconBlock:               CreateGenesisBeaconBlock(1, Testnet2, Testnet2GenesisBlockTime, genesisParamsTestnet2New),
		// GenesisShardBlock:                CreateGenesisShardBlock(1, Testnet2, Testnet2GenesisBlockTime, genesisParamsTestnet2New),
		MinShardBlockInterval:            TestNet2MinShardBlkInterval,
		MaxShardBlockCreation:            TestNet2MaxShardBlkCreation,
		MinBeaconBlockInterval:           TestNet2MinBeaconBlkInterval,
		MaxBeaconBlockCreation:           TestNet2MaxBeaconBlkCreation,
		NumberOfFixedBlockValidators:     4,
		BasicReward:                      Testnet2BasicReward,
		Epoch:                            Testnet2Epoch,
		RandomTime:                       Testnet2RandomTime,
		Offset:                           Testnet2Offset,
		AssignOffset:                     Testnet2AssignOffset,
		SwapOffset:                       Testnet2SwapOffset,
		EthContractAddressStr:            Testnet2ETHContractAddressStr,
		IncognitoDAOAddress:              Testnet2IncognitoDAOAddress,
		CentralizedWebsitePaymentAddress: Testnet2CentralizedWebsitePaymentAddress,
		SlashLevels:                      []SlashLevel{
			//SlashLevel{MinRange: 20, PunishedEpoches: 1},
			//SlashLevel{MinRange: 50, PunishedEpoches: 2},
			//SlashLevel{MinRange: 75, PunishedEpoches: 3},
		},
		CheckForce:                     false,
		ChainVersion:                   "version-chain-test-2.json",
		ConsensusV2Epoch:               15290,
		Timeslot:                       10,
		BeaconHeightBreakPointBurnAddr: 1,
		BNBRelayingHeaderChainID:       Testnet2BNBChainID,
		BTCRelayingHeaderChainID:       Testnet2BTCChainID,
		BTCDataFolderName:              Testnet2BTCDataFolderName,
		BNBFullNodeProtocol:            Testnet2BNBFullNodeProtocol,
		BNBFullNodeHost:                Testnet2BNBFullNodeHost,
		BNBFullNodePort:                Testnet2BNBFullNodePort,
		PortalFeederAddress:            Testnet2PortalFeeder,
		PortalParams: map[uint64]PortalParams{
			0: {
				TimeOutCustodianReturnPubToken:       15 * time.Minute,
				TimeOutWaitingPortingRequest:         15 * time.Minute,
				TimeOutWaitingRedeemRequest:          10 * time.Minute,
				MaxPercentLiquidatedCollateralAmount: 105,
				MaxPercentCustodianRewards:           10, // todo: need to be updated before deploying
				MinPercentCustodianRewards:           1,
				MinLockCollateralAmountInEpoch:       10000 * 1e9, // 10000 usd = 100 * 100
				MinPercentLockedCollateral:           150,
				TP120:                                120,
				TP130:                                130,
				MinPercentPortingFee:                 0.01,
				MinPercentRedeemFee:                  0.01,
				SupportedCollateralTokens:            getSupportedPortalCollateralsTestnet2(),
				MinPortalFee:                         100,
			},
		},
		PortalTokens:                initPortalTokensForTestNet(),
		EpochBreakPointSwapNewKey:   TestnetReplaceCommitteeEpoch,
		ReplaceStakingTxHeight:      1,
		IsBackup:                    false,
		PreloadAddress:              "",
		BCHeightBreakPointNewZKP:    1148608, //TODO: change this value when deployed testnet2
		ETHRemoveBridgeSigEpoch:     2085,
		PortalETHContractAddressStr: "0xF7befD2806afD96D3aF76471cbCa1cD874AA1F46",   // todo: update sc address
		BCHeightBreakPointPortalV3:  1328816,
	}
	// END TESTNET-2

	// FOR MAINNET
	genesisParamsMainnetNew = &GenesisParams{
		PreSelectBeaconNodeSerializedPubkey:         PreSelectBeaconNodeMainnetSerializedPubkey,
		PreSelectBeaconNodeSerializedPaymentAddress: PreSelectBeaconNodeMainnetSerializedPaymentAddress,
		PreSelectShardNodeSerializedPubkey:          PreSelectShardNodeMainnetSerializedPubkey,
		PreSelectShardNodeSerializedPaymentAddress:  PreSelectShardNodeMainnetSerializedPaymentAddress,
		SelectBeaconNodeSerializedPubkeyV2:          SelectBeaconNodeMainnetSerializedPubkeyV2,
		SelectBeaconNodeSerializedPaymentAddressV2:  SelectBeaconNodeMainnetSerializedPaymentAddressV2,
		SelectShardNodeSerializedPubkeyV2:           SelectShardNodeMainnetSerializedPubkeyV2,
		SelectShardNodeSerializedPaymentAddressV2:   SelectShardNodeMainnetSerializedPaymentAddressV2,
		InitialIncognito:                            MainnetInitPRV,
		ConsensusAlgorithm:                          common.BlsConsensus,
	}
	ChainMainParam = Params{
		Name:                   MainetName,
		Net:                    Mainnet,
		DefaultPort:            MainnetDefaultPort,
		GenesisParams:          genesisParamsMainnetNew,
		MaxShardCommitteeSize:  MainNetShardCommitteeSize, //MainNetShardCommitteeSize,
		MinShardCommitteeSize:  MainNetMinShardCommitteeSize,
		MaxBeaconCommitteeSize: MainNetBeaconCommitteeSize, //MainNetBeaconCommitteeSize,
		MinBeaconCommitteeSize: MainNetMinBeaconCommitteeSize,
		StakingAmountShard:     MainNetStakingAmountShard,
		ActiveShards:           MainNetActiveShards,
		// blockChain parameters
		// GenesisBeaconBlock:               CreateGenesisBeaconBlock(1, Mainnet, MainnetGenesisBlockTime, genesisParamsMainnetNew),
		// GenesisShardBlock:                CreateGenesisShardBlock(1, Mainnet, MainnetGenesisBlockTime, genesisParamsMainnetNew),
		MinShardBlockInterval:            MainnetMinShardBlkInterval,
		MaxShardBlockCreation:            MainnetMaxShardBlkCreation,
		MinBeaconBlockInterval:           MainnetMinBeaconBlkInterval,
		MaxBeaconBlockCreation:           MainnetMaxBeaconBlkCreation,
		NumberOfFixedBlockValidators:     22,
		BasicReward:                      MainnetBasicReward,
		Epoch:                            MainnetEpoch,
		RandomTime:                       MainnetRandomTime,
		Offset:                           MainnetOffset,
		SwapOffset:                       MainnetSwapOffset,
		AssignOffset:                     MainnetAssignOffset,
		EthContractAddressStr:            MainETHContractAddressStr,
		IncognitoDAOAddress:              MainnetIncognitoDAOAddress,
		CentralizedWebsitePaymentAddress: MainnetCentralizedWebsitePaymentAddress,
		SlashLevels:                      []SlashLevel{
			//SlashLevel{MinRange: 20, PunishedEpoches: 1},
			//SlashLevel{MinRange: 50, PunishedEpoches: 2},
			//SlashLevel{MinRange: 75, PunishedEpoches: 3},
		},
		CheckForce:                     false,
		ChainVersion:                   "version-chain-main.json",
		ConsensusV2Epoch:               3071,
		Timeslot:                       40,
		BeaconHeightBreakPointBurnAddr: 150500,
		BNBRelayingHeaderChainID:       MainnetBNBChainID,
		BTCRelayingHeaderChainID:       MainnetBTCChainID,
		BTCDataFolderName:              MainnetBTCDataFolderName,
		BNBFullNodeProtocol:            MainnetBNBFullNodeProtocol,
		BNBFullNodeHost:                MainnetBNBFullNodeHost,
		BNBFullNodePort:                MainnetBNBFullNodePort,
		PortalFeederAddress:            MainnetPortalFeeder,
		PortalParams: map[uint64]PortalParams{
			0: {
				TimeOutCustodianReturnPubToken:       24 * time.Hour,
				TimeOutWaitingPortingRequest:         24 * time.Hour,
				TimeOutWaitingRedeemRequest:          15 * time.Minute,
				MaxPercentLiquidatedCollateralAmount: 120,
				MaxPercentCustodianRewards:           20,
				MinPercentCustodianRewards:           1,
				MinPercentLockedCollateral:           200,
				MinLockCollateralAmountInEpoch:       35000 * 1e9, // 35000 usd = 350 * 100
				TP120:                                120,
				TP130:                                130,
				MinPercentPortingFee:                 0.01,
				MinPercentRedeemFee:                  0.01,
				SupportedCollateralTokens:            getSupportedPortalCollateralsMainnet(),
				MinPortalFee:                         100,
			},
		},
		PortalTokens:                initPortalTokensForMainNet(),
		EpochBreakPointSwapNewKey:   MainnetReplaceCommitteeEpoch,
		ReplaceStakingTxHeight:      559380,
		IsBackup:                    false,
		PreloadAddress:              "",
		BCHeightBreakPointNewZKP:    934858,
		ETHRemoveBridgeSigEpoch:     1973,
		PortalETHContractAddressStr: "", // todo: update sc address
		BCHeightBreakPointPortalV3:  40, // todo: should update before deploying
	}
	if IsTestNet {
		if !IsTestNet2 {
			GenesisParam = genesisParamsTestnetNew
		} else {
			GenesisParam = genesisParamsTestnet2New
		}
	} else {
		GenesisParam = genesisParamsMainnetNew
	}
}

func (p *Params) CreateGenesisBlocks() {
	blockTime := ""
	switch p.Net {
	case Mainnet:
		blockTime = MainnetGenesisBlockTime
	case Testnet:
		blockTime = TestnetGenesisBlockTime
	case Testnet2:
		blockTime = Testnet2GenesisBlockTime
	}
	p.GenesisBeaconBlock = CreateGenesisBeaconBlock(1, uint16(p.Net), blockTime, p.GenesisParams)
	p.GenesisShardBlock = CreateGenesisShardBlock(1, uint16(p.Net), blockTime, p.GenesisParams)
	return
}
