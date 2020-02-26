package blockchain

import (
	"encoding/base64"
	"encoding/json"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/database/lvdb"
	"github.com/incognitochain/incognito-chain/metadata"
	"strconv"
)

// beacon build new instruction from instruction received from ShardToBeaconBlock
func buildHeaderRelayingInst(
	senderAddressStr string,
	header string,
	blockHeight uint64,
	metaType int,
	shardID byte,
	txReqID common.Hash,
	status string,
) []string {
	headerRelayingContent := metadata.RelayingHeaderContent{
		IncogAddressStr: senderAddressStr,
		Header:          header,
		TxReqID:         txReqID,
		BlockHeight:     blockHeight,
	}
	headerRelayingContentBytes, _ := json.Marshal(headerRelayingContent)
	return []string{
		strconv.Itoa(metaType),
		strconv.Itoa(int(shardID)),
		status,
		string(headerRelayingContentBytes),
	}
}

// buildInstructionsForHeaderRelaying builds instruction for custodian deposit action
func (blockchain *BlockChain) buildInstructionsForHeaderRelaying(
	contentStr string,
	shardID byte,
	metaType int,
	relayingHeaderChain *RelayingHeaderChainState,
	beaconHeight uint64,
) ([][]string, error) {
	// parse instruction
	actionContentBytes, err := base64.StdEncoding.DecodeString(contentStr)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while decoding content string of portal custodian deposit action: %+v", err)
		return [][]string{}, nil
	}
	var actionData metadata.RelayingHeaderAction
	err = json.Unmarshal(actionContentBytes, &actionData)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while unmarshal portal custodian deposit action: %+v", err)
		return [][]string{}, nil
	}

	if relayingHeaderChain == nil {
		Logger.log.Warn("WARN - [buildInstructionsForHeaderRelaying]: relayingHeaderChain is null.")
		inst := buildHeaderRelayingInst(
			actionData.Meta.IncogAddressStr,
			actionData.Meta.Header,
			actionData.Meta.BlockHeight,
			actionData.Meta.Type,
			shardID,
			actionData.TxReqID,
			common.RelayingHeaderRejectedChainStatus,
		)
		return [][]string{inst}, nil
	}
	meta := actionData.Meta
	// parse and verify header chain
	headerBytes, err := base64.StdEncoding.DecodeString(meta.Header)
	if err != nil {
		Logger.log.Errorf("Error - [buildInstructionsForHeaderRelaying]: Can not decode header string.%v\n", err)
		inst := buildHeaderRelayingInst(
			actionData.Meta.IncogAddressStr,
			actionData.Meta.Header,
			actionData.Meta.BlockHeight,
			actionData.Meta.Type,
			shardID,
			actionData.TxReqID,
			common.RelayingHeaderRejectedChainStatus,
		)
		return [][]string{inst}, nil
	}

	var newHeader lvdb.BNBHeader
	err = json.Unmarshal(headerBytes, &newHeader)
	if err != nil {
		Logger.log.Errorf("Error - [buildInstructionsForHeaderRelaying]: Can not unmarshal header.%v\n", err)
		inst := buildHeaderRelayingInst(
			actionData.Meta.IncogAddressStr,
			actionData.Meta.Header,
			actionData.Meta.BlockHeight,
			actionData.Meta.Type,
			shardID,
			actionData.TxReqID,
			common.RelayingHeaderRejectedChainStatus,
		)
		return [][]string{inst}, nil
	}

	if newHeader.Header.Height != int64(actionData.Meta.BlockHeight) {
		Logger.log.Errorf("Error - [buildInstructionsForHeaderRelaying]: Block height in metadata is unmatched with block height in new header.")
		inst := buildHeaderRelayingInst(
			actionData.Meta.IncogAddressStr,
			actionData.Meta.Header,
			actionData.Meta.BlockHeight,
			actionData.Meta.Type,
			shardID,
			actionData.TxReqID,
			common.RelayingHeaderRejectedChainStatus,
		)
		return [][]string{inst}, nil
	}

	// if valid, create instruction with status accepted
	// if not, create instruction with status rejected
	latestBNBHeader := relayingHeaderChain.BNBHeaderChain.LatestHeader
	isValid, err := relayingHeaderChain.BNBHeaderChain.ReceiveNewHeader(newHeader.Header, newHeader.LastCommit)
	if err != nil || !isValid {
		Logger.log.Errorf("Error - [buildInstructionsForHeaderRelaying]: Verify new header failed. %v\n", err)
		inst := buildHeaderRelayingInst(
			actionData.Meta.IncogAddressStr,
			actionData.Meta.Header,
			actionData.Meta.BlockHeight,
			actionData.Meta.Type,
			shardID,
			actionData.TxReqID,
			common.RelayingHeaderRejectedChainStatus,
		)
		return [][]string{inst}, nil
	}

	// check newHeader is a header contain last commit for one of the header in unconfirmed header list or not
	//todo: check pointer
	newLatestBNBHeader := relayingHeaderChain.BNBHeaderChain.LatestHeader
	//newLatestBNBHeader.Last
	if newLatestBNBHeader.Height  == latestBNBHeader.Height + 1 {
		inst := buildHeaderRelayingInst(
			actionData.Meta.IncogAddressStr,
			actionData.Meta.Header,
			actionData.Meta.BlockHeight,
			actionData.Meta.Type,
			shardID,
			actionData.TxReqID,
			common.RelayingHeaderConfirmedAcceptedChainStatus,
		)
		return [][]string{inst}, nil
	}

	inst := buildHeaderRelayingInst(
		actionData.Meta.IncogAddressStr,
		actionData.Meta.Header,
		actionData.Meta.BlockHeight,
		actionData.Meta.Type,
		shardID,
		actionData.TxReqID,
		common.RelayingHeaderUnconfirmedAcceptedChainStatus,
	)
	return [][]string{inst}, nil
}
