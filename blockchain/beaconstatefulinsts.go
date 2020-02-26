package blockchain

import (
	"encoding/base64"
	"encoding/json"
	"math/big"
	"sort"
	"strconv"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/database"
	"github.com/incognitochain/incognito-chain/database/lvdb"
	"github.com/incognitochain/incognito-chain/metadata"
)

// build instructions at beacon chain before syncing to shards
func (blockchain *BlockChain) collectStatefulActions(
	shardBlockInstructions [][]string,
) [][]string {
	// stateful instructions are dependently processed with results of instructioins before them in shards2beacon blocks
	statefulInsts := [][]string{}
	for _, inst := range shardBlockInstructions {
		if len(inst) < 2 {
			continue
		}
		if inst[0] == SetAction || inst[0] == StakeAction || inst[0] == SwapAction || inst[0] == RandomAction || inst[0] == AssignAction {
			continue
		}

		metaType, err := strconv.Atoi(inst[0])
		if err != nil {
			Logger.log.Error(err)
			continue
		}
		switch metaType {
		case metadata.IssuingRequestMeta,
			metadata.IssuingETHRequestMeta,
			metadata.PDEContributionMeta,
			metadata.PDETradeRequestMeta,
			metadata.PDEWithdrawalRequestMeta,
			metadata.PortalCustodianDepositMeta,
			metadata.PortalUserRegisterMeta,
			metadata.PortalUserRequestPTokenMeta,
			metadata.PortalExchangeRatesMeta,
			metadata.RelayingHeaderMeta:
			statefulInsts = append(statefulInsts, inst)

		default:
			continue
		}
	}
	return statefulInsts
}

func groupPDEActionsByShardID(
	pdeActionsByShardID map[byte][][]string,
	action []string,
	shardID byte,
) map[byte][][]string {
	_, found := pdeActionsByShardID[shardID]
	if !found {
		pdeActionsByShardID[shardID] = [][]string{action}
	} else {
		pdeActionsByShardID[shardID] = append(pdeActionsByShardID[shardID], action)
	}
	return pdeActionsByShardID
}

func (blockchain *BlockChain) buildStatefulInstructions(
	statefulActionsByShardID map[byte][][]string,
	beaconHeight uint64,
	db database.DatabaseInterface,
) [][]string {
	currentPDEState, err := InitCurrentPDEStateFromDB(db, beaconHeight-1)
	if err != nil {
		Logger.log.Error(err)
	}

	currentPortalState, err := InitCurrentPortalStateFromDB(db, beaconHeight-1)
	if err != nil {
		Logger.log.Error(err)
	}

	relayingHeaderState, err := InitRelayingHeaderChainStateFromDB(db, beaconHeight-1)
	if err != nil {
		Logger.log.Error(err)
	}

	accumulatedValues := &metadata.AccumulatedValues{
		UniqETHTxsUsed:   [][]byte{},
		DBridgeTokenPair: map[string][]byte{},
		CBridgeTokens:    []*common.Hash{},
	}
	instructions := [][]string{}

	// pde instructions
	pdeContributionActionsByShardID := map[byte][][]string{}
	pdeTradeActionsByShardID := map[byte][][]string{}
	pdeWithdrawalActionsByShardID := map[byte][][]string{}

	// portal instructions
	portalCustodianDepositActionsByShardID := map[byte][][]string{}
	portalUserReqPortingActionsByShardID := map[byte][][]string{}
	portalUserReqPTokenActionsByShardID := map[byte][][]string{}
	portalExchangeRatesActionsByShardID := map[byte][][]string{}

	// relaying instructions
	// don't need to be grouped by shardID
	relayingActions := [][]string{}

	var keys []int
	for k := range statefulActionsByShardID {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	for _, value := range keys {
		shardID := byte(value)
		actions := statefulActionsByShardID[shardID]
		for _, action := range actions {
			metaType, err := strconv.Atoi(action[0])
			if err != nil {
				continue
			}
			contentStr := action[1]
			newInst := [][]string{}
			switch metaType {
			case metadata.IssuingRequestMeta:
				newInst, err = blockchain.buildInstructionsForIssuingReq(contentStr, shardID, metaType, accumulatedValues)

			case metadata.IssuingETHRequestMeta:
				newInst, err = blockchain.buildInstructionsForIssuingETHReq(contentStr, shardID, metaType, accumulatedValues)

			case metadata.PDEContributionMeta:
				pdeContributionActionsByShardID = groupPDEActionsByShardID(
					pdeContributionActionsByShardID,
					action,
					shardID,
				)
			case metadata.PDETradeRequestMeta:
				pdeTradeActionsByShardID = groupPDEActionsByShardID(
					pdeTradeActionsByShardID,
					action,
					shardID,
				)
			case metadata.PDEWithdrawalRequestMeta:
				pdeWithdrawalActionsByShardID = groupPDEActionsByShardID(
					pdeWithdrawalActionsByShardID,
					action,
					shardID,
				)
			case metadata.PortalCustodianDepositMeta:
				portalCustodianDepositActionsByShardID = groupPortalActionsByShardID(
					portalCustodianDepositActionsByShardID,
					action,
					shardID,
				)
			case metadata.PortalUserRegisterMeta:
				portalUserReqPortingActionsByShardID = groupPortalActionsByShardID(
					portalUserReqPortingActionsByShardID,
					action,
					shardID,
				)
			case metadata.PortalUserRequestPTokenMeta:
				portalUserReqPTokenActionsByShardID = groupPortalActionsByShardID(
					portalUserReqPTokenActionsByShardID,
					action,
					shardID,
				)
			case metadata.PortalExchangeRatesMeta:
				portalExchangeRatesActionsByShardID = groupPortalActionsByShardID(
					portalExchangeRatesActionsByShardID,
					action,
					shardID,
				)
			case metadata.RelayingHeaderMeta:
				relayingActions = append(relayingActions, action)
			default:
				continue
			}
			if err != nil {
				Logger.log.Error(err)
				continue
			}
			if len(newInst) > 0 {
				instructions = append(instructions, newInst...)
			}
		}
	}

	pdeInsts, err := blockchain.handlePDEInsts(
		beaconHeight-1, currentPDEState,
		pdeContributionActionsByShardID,
		pdeTradeActionsByShardID,
		pdeWithdrawalActionsByShardID,
	)

	if err != nil {
		Logger.log.Error(err)
		return instructions
	}
	if len(pdeInsts) > 0 {
		instructions = append(instructions, pdeInsts...)
	}

	portalInsts, err := blockchain.handlePortalInsts(
		beaconHeight-1,
		currentPortalState,
		portalCustodianDepositActionsByShardID,
		portalUserReqPortingActionsByShardID,
		portalUserReqPTokenActionsByShardID,
		portalExchangeRatesActionsByShardID,
		)

	if err != nil {
		Logger.log.Error(err)
		return instructions
	}
	if len(portalInsts) > 0 {
		instructions = append(instructions, portalInsts...)
	}

	relayingInsts, err := blockchain.handleRelayingInsts(
		beaconHeight-1,
		relayingHeaderState,
		relayingActions,
	)

	if err != nil {
		Logger.log.Error(err)
		return instructions
	}
	if len(relayingInsts) > 0 {
		instructions = append(instructions, relayingInsts...)
	}

	return instructions
}

func sortPDETradeInstsByFee(
	beaconHeight uint64,
	currentPDEState *CurrentPDEState,
	pdeTradeActionsByShardID map[byte][][]string,
) []metadata.PDETradeRequestAction {
	tradesByPairs := make(map[string][]metadata.PDETradeRequestAction)

	var keys []int
	for k := range pdeTradeActionsByShardID {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	for _, value := range keys {
		shardID := byte(value)
		actions := pdeTradeActionsByShardID[shardID]
		for _, action := range actions {
			contentStr := action[1]
			contentBytes, err := base64.StdEncoding.DecodeString(contentStr)
			if err != nil {
				Logger.log.Errorf("ERROR: an error occured while decoding content string of pde trade action: %+v", err)
				continue
			}
			var pdeTradeReqAction metadata.PDETradeRequestAction
			err = json.Unmarshal(contentBytes, &pdeTradeReqAction)
			if err != nil {
				Logger.log.Errorf("ERROR: an error occured while unmarshaling pde trade action: %+v", err)
				continue
			}
			tradeMeta := pdeTradeReqAction.Meta
			poolPairKey := string(lvdb.BuildPDEPoolForPairKey(beaconHeight, tradeMeta.TokenIDToBuyStr, tradeMeta.TokenIDToSellStr))
			tradesByPair, found := tradesByPairs[poolPairKey]
			if !found {
				tradesByPairs[poolPairKey] = []metadata.PDETradeRequestAction{pdeTradeReqAction}
			} else {
				tradesByPairs[poolPairKey] = append(tradesByPair, pdeTradeReqAction)
			}
		}
	}

	notExistingPairTradeActions := []metadata.PDETradeRequestAction{}
	sortedExistingPairTradeActions := []metadata.PDETradeRequestAction{}

	var ppKeys []string
	for k := range tradesByPairs {
		ppKeys = append(ppKeys, k)
	}
	sort.Strings(ppKeys)
	for _, poolPairKey := range ppKeys {
		tradeActions := tradesByPairs[poolPairKey]
		poolPair, found := currentPDEState.PDEPoolPairs[poolPairKey]
		if !found || poolPair == nil {
			notExistingPairTradeActions = append(notExistingPairTradeActions, tradeActions...)
			continue
		}
		if poolPair.Token1PoolValue == 0 || poolPair.Token2PoolValue == 0 {
			notExistingPairTradeActions = append(notExistingPairTradeActions, tradeActions...)
			continue
		}

		// sort trade actions by trading fee
		sort.Slice(tradeActions, func(i, j int) bool {
			// comparing a/b to c/d is equivalent with comparing a*d to c*b
			firstItemProportion := big.NewInt(0)
			firstItemProportion.Mul(
				big.NewInt(int64(tradeActions[i].Meta.TradingFee)),
				big.NewInt(int64(tradeActions[j].Meta.SellAmount)),
			)
			secondItemProportion := big.NewInt(0)
			secondItemProportion.Mul(
				big.NewInt(int64(tradeActions[j].Meta.TradingFee)),
				big.NewInt(int64(tradeActions[i].Meta.SellAmount)),
			)
			return firstItemProportion.Cmp(secondItemProportion) == 1
		})
		sortedExistingPairTradeActions = append(sortedExistingPairTradeActions, tradeActions...)
	}
	return append(sortedExistingPairTradeActions, notExistingPairTradeActions...)
}

func (blockchain *BlockChain) handlePDEInsts(
	beaconHeight uint64,
	currentPDEState *CurrentPDEState,
	pdeContributionActionsByShardID map[byte][][]string,
	pdeTradeActionsByShardID map[byte][][]string,
	pdeWithdrawalActionsByShardID map[byte][][]string,
) ([][]string, error) {
	instructions := [][]string{}
	sortedTradesActions := sortPDETradeInstsByFee(
		beaconHeight,
		currentPDEState,
		pdeTradeActionsByShardID,
	)
	for _, tradeAction := range sortedTradesActions {
		actionContentBytes, _ := json.Marshal(tradeAction)
		actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
		newInst, err := blockchain.buildInstructionsForPDETrade(actionContentBase64Str, tradeAction.ShardID, metadata.PDETradeRequestMeta, currentPDEState, beaconHeight)
		if err != nil {
			Logger.log.Error(err)
			continue
		}
		if len(newInst) > 0 {
			instructions = append(instructions, newInst...)
		}
	}

	// handle withdrawal
	var wrKeys []int
	for k := range pdeWithdrawalActionsByShardID {
		wrKeys = append(wrKeys, int(k))
	}
	sort.Ints(wrKeys)
	for _, value := range wrKeys {
		shardID := byte(value)
		actions := pdeWithdrawalActionsByShardID[shardID]
		for _, action := range actions {
			contentStr := action[1]
			newInst, err := blockchain.buildInstructionsForPDEWithdrawal(contentStr, shardID, metadata.PDEWithdrawalRequestMeta, currentPDEState, beaconHeight)
			if err != nil {
				Logger.log.Error(err)
				continue
			}
			if len(newInst) > 0 {
				instructions = append(instructions, newInst...)
			}
		}
	}

	// handle contribution
	var ctKeys []int
	for k := range pdeContributionActionsByShardID {
		ctKeys = append(ctKeys, int(k))
	}
	sort.Ints(ctKeys)
	for _, value := range ctKeys {
		shardID := byte(value)
		actions := pdeContributionActionsByShardID[shardID]
		for _, action := range actions {
			contentStr := action[1]
			newInst, err := blockchain.buildInstructionsForPDEContribution(contentStr, shardID, metadata.PDEContributionMeta, currentPDEState, beaconHeight)
			if err != nil {
				Logger.log.Error(err)
				continue
			}
			if len(newInst) > 0 {
				instructions = append(instructions, newInst...)
			}
		}
	}
	return instructions, nil
}

// Portal
func groupPortalActionsByShardID(
	portalActionsByShardID map[byte][][]string,
	action []string,
	shardID byte,
) map[byte][][]string {
	_, found := portalActionsByShardID[shardID]
	if !found {
		portalActionsByShardID[shardID] = [][]string{action}
	} else {
		portalActionsByShardID[shardID] = append(portalActionsByShardID[shardID], action)
	}
	return portalActionsByShardID
}

func (blockchain *BlockChain) handlePortalInsts(
	beaconHeight uint64,
	currentPortalState *CurrentPortalState,
	portalCustodianDepositActionsByShardID map[byte][][]string,
	portalUserRequestPortingActionsByShardID map[byte][][]string,
	portalUserRequestPTokenActionsByShardID map[byte][][]string,
	portalExchangeRatesActionsByShardID map[byte][][]string,
) ([][]string, error) {
	instructions := [][]string{}

	// handle portal custodian deposit inst
	var custodianShardIDKeys []int
	for k := range portalCustodianDepositActionsByShardID {
		custodianShardIDKeys = append(custodianShardIDKeys, int(k))
	}

	sort.Ints(custodianShardIDKeys)
	for _, value := range custodianShardIDKeys {
		shardID := byte(value)
		actions := portalCustodianDepositActionsByShardID[shardID]
		for _, action := range actions {
			contentStr := action[1]
			newInst, err := blockchain.buildInstructionsForCustodianDeposit(
				contentStr,
				shardID,
				metadata.PortalCustodianDepositMeta,
				currentPortalState,
				beaconHeight,
				)

			if err != nil {
				Logger.log.Error(err)
				continue
			}
			if len(newInst) > 0 {
				instructions = append(instructions, newInst...)
			}
		}
	}

	// handle portal user request porting inst
	var requestPortingShardIDKeys []int
	for k := range portalUserRequestPortingActionsByShardID {
		requestPortingShardIDKeys = append(requestPortingShardIDKeys, int(k))
	}

	sort.Ints(requestPortingShardIDKeys)
	for _, value := range requestPortingShardIDKeys {
		shardID := byte(value)
		actions := portalUserRequestPortingActionsByShardID[shardID]
		for _, action := range actions {
			contentStr := action[1]
			newInst, err := blockchain.buildInstructionsForPortingRequest(
				contentStr,
				shardID,
				metadata.PortalUserRegisterMeta,
				currentPortalState,
				beaconHeight,
			)

			if err != nil {
				Logger.log.Error(err)
				continue
			}
			if len(newInst) > 0 {
				instructions = append(instructions, newInst...)
			}
		}
	}
	// handle portal user request ptoken inst
	var reqPTokenShardIDKeys []int
	for k := range portalUserRequestPTokenActionsByShardID {
		reqPTokenShardIDKeys = append(reqPTokenShardIDKeys, int(k))
	}

	sort.Ints(reqPTokenShardIDKeys)
	for _, value := range reqPTokenShardIDKeys {
		shardID := byte(value)
		actions := portalUserRequestPTokenActionsByShardID[shardID]
		for _, action := range actions {
			contentStr := action[1]
			newInst, err := blockchain.buildInstructionsForReqPTokens(
				contentStr,
				shardID,
				metadata.PortalUserRequestPTokenMeta,
				currentPortalState,
				beaconHeight,
			)

			if err != nil {
				Logger.log.Error(err)
				continue
			}
			if len(newInst) > 0 {
				instructions = append(instructions, newInst...)
			}
		}
	}

	//handle portal exchange rates
	var exchangeRatesShardIDKeys []int
	for k := range portalExchangeRatesActionsByShardID {
		exchangeRatesShardIDKeys = append(exchangeRatesShardIDKeys, int(k))
	}

	sort.Ints(exchangeRatesShardIDKeys)
	for _, value := range exchangeRatesShardIDKeys {
		shardID := byte(value)
		actions := portalExchangeRatesActionsByShardID[shardID]
		for _, action := range actions {
			contentStr := action[1]
			newInst, err := blockchain.buildInstructionsForExchangeRates(
				contentStr,
				shardID,
				metadata.PortalExchangeRatesMeta,
				currentPortalState,
				beaconHeight,
			)

			if err != nil {
				Logger.log.Error(err)
				continue
			}
			if len(newInst) > 0 {
				instructions = append(instructions, newInst...)
			}
		}
	}

	return instructions, nil
}


// Header relaying
func groupRelayingActionsByShardID(
	relayingActionsByShardID map[byte][][]string,
	action []string,
	shardID byte,
) map[byte][][]string {
	_, found := relayingActionsByShardID[shardID]
	if !found {
		relayingActionsByShardID[shardID] = [][]string{action}
	} else {
		relayingActionsByShardID[shardID] = append(relayingActionsByShardID[shardID], action)
	}
	return relayingActionsByShardID
}

func sortRelayingInstsByBlockHeight(portalPushHeaderRelayingActions [][]string)(
	map[uint64][]metadata.RelayingHeaderAction, []uint64, error){
	// sort push header relaying inst
	actionsGroupByBlockHeight := make(map[uint64][]metadata.RelayingHeaderAction)

	var blockHeightArr []uint64

	for _, inst := range portalPushHeaderRelayingActions {
		// parse inst
		var action metadata.RelayingHeaderAction
		actionBytes, err := base64.StdEncoding.DecodeString(inst[1])
		if err != nil {
			continue
		}
		err = json.Unmarshal(actionBytes, &action)
		if err != nil {
			continue
		}

		// get blockHeight in action
		blockHeight := action.Meta.BlockHeight

		// add to blockHeightArr
		if isExist, _ := common.SliceExists(blockHeightArr, blockHeight); !isExist {
			blockHeightArr = append(blockHeightArr, blockHeight)
		}

		// add to actionsGroupByBlockHeight
		if actionsGroupByBlockHeight[blockHeight] != nil {
			actionsGroupByBlockHeight[blockHeight] = append(actionsGroupByBlockHeight[blockHeight], action)
		} else{
			actionsGroupByBlockHeight[blockHeight] = []metadata.RelayingHeaderAction{action}
		}
	}

	// sort blockHeightArr
	sort.Slice(blockHeightArr, func(i, j int) bool {
		return blockHeightArr[i] < blockHeightArr[j]
	})

	return actionsGroupByBlockHeight, blockHeightArr, nil
}

func (blockchain *BlockChain) handleRelayingInsts(
	beaconHeight uint64,
	headerChain *RelayingHeaderChainState,
	relayingHeaderActions [][]string,
) ([][]string, error) {
	instructions := [][]string{}

	actionsGroupByBlockHeight, sortedBlockHeights, _ := sortRelayingInstsByBlockHeight(relayingHeaderActions)

	for _, value := range sortedBlockHeights {
		blockHeight := uint64(value)
		actions := actionsGroupByBlockHeight[blockHeight]
		for _, action := range actions {
			actionBytes, _ := json.Marshal(action)
			contentStr := base64.StdEncoding.EncodeToString(actionBytes)
			newInst, err := blockchain.buildInstructionsForHeaderRelaying(
				contentStr,
				action.ShardID,
				metadata.RelayingHeaderMeta,
				headerChain,
				beaconHeight,
			)

			if err != nil {
				Logger.log.Error(err)
				continue
			}
			if len(newInst) > 0 {
				instructions = append(instructions, newInst...)
			}
		}
	}

	return instructions, nil
}