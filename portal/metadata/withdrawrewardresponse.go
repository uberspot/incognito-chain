package metadata

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/basemeta"
	pCommon "github.com/incognitochain/incognito-chain/portal/common"
	"github.com/incognitochain/incognito-chain/wallet"
	"strconv"
)

// PortalRequestUnlockCollateral - portal custodian requests unlock collateral (after returning pubToken to user)
// metadata - custodian requests unlock collateral - create normal tx with this metadata
type PortalWithdrawRewardResponse struct {
	basemeta.MetadataBase
	CustodianAddressStr string
	TokenID             common.Hash
	RewardAmount        uint64
	TxReqID             common.Hash
}

func NewPortalWithdrawRewardResponse(
	reqTxID common.Hash,
	custodianAddressStr string,
	tokenID common.Hash,
	rewardAmount uint64,
	metaType int,
) *PortalWithdrawRewardResponse {
	metadataBase := basemeta.MetadataBase{
		Type: metaType,
	}
	return &PortalWithdrawRewardResponse{
		MetadataBase:        metadataBase,
		CustodianAddressStr: custodianAddressStr,
		TokenID:             tokenID,
		RewardAmount:        rewardAmount,
		TxReqID:             reqTxID,
	}
}

func (iRes PortalWithdrawRewardResponse) CheckTransactionFee(tr basemeta.Transaction, minFee uint64, beaconHeight int64, db *statedb.StateDB) bool {
	// no need to have fee for this tx
	return true
}

func (iRes PortalWithdrawRewardResponse) ValidateTxWithBlockChain(txr basemeta.Transaction, chainRetriever basemeta.ChainRetriever, shardViewRetriever basemeta.ShardViewRetriever, beaconViewRetriever basemeta.BeaconViewRetriever, shardID byte, db *statedb.StateDB) (bool, error) {
	// no need to validate tx with blockchain, just need to validate with requested tx (via RequestedTxID)
	return false, nil
}

func (iRes PortalWithdrawRewardResponse) ValidateSanityData(chainRetriever basemeta.ChainRetriever, shardViewRetriever basemeta.ShardViewRetriever, beaconViewRetriever basemeta.BeaconViewRetriever, beaconHeight uint64, txr basemeta.Transaction) (bool, bool, error) {
	return false, true, nil
}

func (iRes PortalWithdrawRewardResponse) ValidateMetadataByItself() bool {
	// The validation just need to check at tx level, so returning true here
	return iRes.Type == basemeta.PortalRequestWithdrawRewardResponseMeta
}

func (iRes PortalWithdrawRewardResponse) Hash() *common.Hash {
	record := iRes.MetadataBase.Hash().String()
	record += iRes.TxReqID.String()
	record += iRes.CustodianAddressStr
	record += iRes.TokenID.String()
	record += strconv.FormatUint(iRes.RewardAmount, 10)
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (iRes *PortalWithdrawRewardResponse) CalculateSize() uint64 {
	return basemeta.CalculateSize(iRes)
}

func (iRes PortalWithdrawRewardResponse) VerifyMinerCreatedTxBeforeGettingInBlock(
	txsInBlock []basemeta.Transaction,
	txsUsed []int,
	insts [][]string,
	instUsed []int,
	shardID byte,
	tx basemeta.Transaction,
	chainRetriever basemeta.ChainRetriever,
	ac *basemeta.AccumulatedValues,
	shardViewRetriever basemeta.ShardViewRetriever,
	beaconViewRetriever basemeta.BeaconViewRetriever,
) (bool, error) {
	idx := -1
	for i, inst := range insts {
		if len(inst) < 4 { // this is not PortalWithdrawReward instruction
			continue
		}
		instMetaType := inst[0]
		if instUsed[i] > 0 ||
			instMetaType != strconv.Itoa(basemeta.PortalRequestWithdrawRewardMeta) {
			continue
		}
		instDepositStatus := inst[2]
		if instDepositStatus != pCommon.PortalRequestAcceptedChainStatus {
			continue
		}

		var shardIDFromInst byte
		var txReqIDFromInst common.Hash
		var custodianAddrStrFromInst string
		var rewardAmountFromInst uint64
		var tokenIDFromInst common.Hash

		contentBytes := []byte(inst[3])
		var reqWithdrawRewardContent PortalRequestWithdrawRewardContent
		err := json.Unmarshal(contentBytes, &reqWithdrawRewardContent)
		if err != nil {
			Logger.log.Error("WARNING - VALIDATION: an error occured while parsing portal request withdraw reward content: ", err)
			continue
		}
		shardIDFromInst = reqWithdrawRewardContent.ShardID
		txReqIDFromInst = reqWithdrawRewardContent.TxReqID
		custodianAddrStrFromInst = reqWithdrawRewardContent.CustodianAddressStr
		rewardAmountFromInst = reqWithdrawRewardContent.RewardAmount
		tokenIDFromInst = reqWithdrawRewardContent.TokenID

		if !bytes.Equal(iRes.TxReqID[:], txReqIDFromInst[:]) ||
			shardID != shardIDFromInst {
			continue
		}
		key, err := wallet.Base58CheckDeserialize(custodianAddrStrFromInst)
		if err != nil {
			Logger.log.Info("WARNING - VALIDATION: an error occured while deserializing custodian address string: ", err)
			continue
		}

		_, pk, paidAmount, assetID := tx.GetTransferData()
		if !bytes.Equal(key.KeySet.PaymentAddress.Pk[:], pk[:]) ||
			rewardAmountFromInst != paidAmount ||
			tokenIDFromInst.String() != assetID.String() {
			continue
		}
		idx = i
		break
	}
	if idx == -1 { // not found the issuance request tx for this response
		return false, fmt.Errorf(fmt.Sprintf("no PortalWithdrawReward instruction found for PortalWithdrawRewardResponse tx %s", tx.Hash().String()))
	}
	instUsed[idx] = 1
	return true, nil
}