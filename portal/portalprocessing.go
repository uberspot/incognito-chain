package portal

import (
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/portal/portalrelaying"
	portalprocessv3 "github.com/incognitochain/incognito-chain/portal/portalv3/portalprocess"
	portalprocessv4 "github.com/incognitochain/incognito-chain/portal/portalv4/portalprocess"
)

func CollectPortalInstructions(pm *PortalManager, metaType int, action []string, shardID byte) bool {
	isCollected := true
	if metadata.IsPortalRelayingMetaType(metaType) {
		portalrelaying.CollectPortalRelayingInsts(pm.RelayingChainsProcessors, metaType, action, shardID)
	} else if metadata.IsPortalMetaTypeV3(metaType) {
		portalprocessv3.CollectPortalV3Insts(pm.PortalInstProcessorsV3, metaType, action, shardID)
	} else if metadata.IsPortalMetaTypeV4(metaType) {
		portalprocessv4.CollectPortalV4Insts(pm.PortalInstProcessorsV4, metaType, action, shardID)
	} else {
		isCollected = false
	}

	return isCollected
}

func HandlePortalInsts(
	bc metadata.ChainRetriever,
	stateDB *statedb.StateDB,
	beaconHeight uint64,
	shardHeight map[byte]uint64,
	currentPortalState *portalprocessv3.CurrentPortalState,
	currentPortalStateV4 *portalprocessv4.CurrentPortalStateV4,
	relayingState *portalrelaying.RelayingHeaderChainState,
	rewardForCustodianByEpoch map[common.Hash]uint64,
	portalParams PortalParams,
	pm *PortalManager,
	epochBlocks uint64,
) ([][]string, error) {
	currentEpoch := common.GetEpochFromBeaconHeight(beaconHeight, epochBlocks)
	instructions := [][]string{}
	// handle portal instructions v3
	if bc.IsEnableFeature(common.PortalV3Flag, currentEpoch) {
		portalInstsV3, err := portalprocessv3.HandlePortalInstsV3(
			bc, stateDB, beaconHeight, shardHeight, currentPortalState, rewardForCustodianByEpoch,
			portalParams.GetPortalParamsV3(beaconHeight), pm.PortalInstProcessorsV3)
		if err != nil {
			Logger.log.Error(err)
		}
		if len(portalInstsV3) > 0 {
			instructions = append(instructions, portalInstsV3...)
		}
	}

	// Handle relaying instruction
	if bc.IsEnableFeature(common.PortalRelayingFlag, currentEpoch) {
		relayingInsts := portalrelaying.HandleRelayingInsts(bc, relayingState, pm.RelayingChainsProcessors)
		if len(relayingInsts) > 0 {
			instructions = append(instructions, relayingInsts...)
		}
	}

	// handle portal instructions v4
	if bc.IsEnableFeature(common.PortalV4Flag, currentEpoch) {
		portalInstsV4, err := portalprocessv4.HandlePortalInstsV4(
			bc, stateDB, beaconHeight, shardHeight, currentPortalStateV4,
			portalParams.GetPortalParamsV4(beaconHeight), pm.PortalInstProcessorsV4)
		if err != nil {
			Logger.log.Error(err)
		}
		if len(portalInstsV4) > 0 {
			instructions = append(instructions, portalInstsV4...)
		}
	}

	// Handle next things ...

	return instructions, nil
}

func ProcessPortalInsts(
	bc metadata.ChainRetriever,
	portalStateDB *statedb.StateDB,
	relayingState *portalrelaying.RelayingHeaderChainState,
	portalParams PortalParams,
	beaconHeight uint64,
	instructions [][]string,
	pm *PortalManager,
	epoch uint64,
	isSkipPortalV3Ints bool) error {
	// process portal instructions v3
	if !isSkipPortalV3Ints {
		err := portalprocessv3.ProcessPortalInstsV3(
			portalStateDB, portalParams.GetPortalParamsV3(beaconHeight),
			beaconHeight, instructions, pm.PortalInstProcessorsV3, epoch)
		if err != nil {
			Logger.log.Error(err)
			return err
		}
	}

	// process relaying instructions
	err := portalrelaying.ProcessRelayingInstructions(instructions, relayingState)
	if err != nil {
		Logger.log.Error(err)
		return err
	}

	// process portal instructions v4
	err = portalprocessv4.ProcessPortalInstsV4(
		portalStateDB, portalParams.GetPortalParamsV4(beaconHeight),
		beaconHeight, instructions, pm.PortalInstProcessorsV4, epoch)
	if err != nil {
		Logger.log.Error(err)
		return err
	}

	// Handle next things ...

	return nil
}