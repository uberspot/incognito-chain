package metadata

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/basemeta"
	"github.com/incognitochain/incognito-chain/wallet"
	"reflect"
	"strconv"
)

type PortalRedeemLiquidateExchangeRates struct {
	basemeta.MetadataBase
	TokenID               string // pTokenID in incognito chain
	RedeemAmount          uint64
	RedeemerIncAddressStr string
}

type PortalRedeemLiquidateExchangeRatesAction struct {
	Meta    PortalRedeemLiquidateExchangeRates
	TxReqID common.Hash
	ShardID byte
}

type PortalRedeemLiquidateExchangeRatesContent struct {
	TokenID               string // pTokenID in incognito chain
	RedeemAmount          uint64
	RedeemerIncAddressStr string
	TxReqID               common.Hash
	ShardID               byte
	TotalPTokenReceived   uint64
}

type RedeemLiquidateExchangeRatesStatus struct {
	TxReqID             common.Hash
	TokenID             string
	RedeemerAddress     string
	RedeemAmount        uint64
	Status              byte
	TotalPTokenReceived uint64
}

func NewRedeemLiquidateExchangeRatesStatus(txReqID common.Hash, tokenID string, redeemerAddress string, redeemAmount uint64, status byte, totalPTokenReceived uint64) *RedeemLiquidateExchangeRatesStatus {
	return &RedeemLiquidateExchangeRatesStatus{TxReqID: txReqID, TokenID: tokenID, RedeemerAddress: redeemerAddress, RedeemAmount: redeemAmount, Status: status, TotalPTokenReceived: totalPTokenReceived}
}

func NewPortalRedeemLiquidateExchangeRates(
	metaType int,
	tokenID string,
	redeemAmount uint64,
	incAddressStr string,
) (*PortalRedeemLiquidateExchangeRates, error) {
	metadataBase := basemeta.MetadataBase{Type: metaType}

	portalRedeemLiquidateExchangeRates := &PortalRedeemLiquidateExchangeRates{
		TokenID:               tokenID,
		RedeemAmount:          redeemAmount,
		RedeemerIncAddressStr: incAddressStr,
	}

	portalRedeemLiquidateExchangeRates.MetadataBase = metadataBase

	return portalRedeemLiquidateExchangeRates, nil
}

func (redeemReq PortalRedeemLiquidateExchangeRates) ValidateTxWithBlockChain(
	txr basemeta.Transaction,
	chainRetriever basemeta.ChainRetriever, shardViewRetriever basemeta.ShardViewRetriever, beaconViewRetriever basemeta.BeaconViewRetriever,
	shardID byte,
	db *statedb.StateDB,
) (bool, error) {
	return true, nil
}

func (redeemReq PortalRedeemLiquidateExchangeRates) ValidateSanityData(chainRetriever basemeta.ChainRetriever, shardViewRetriever basemeta.ShardViewRetriever, beaconViewRetriever basemeta.BeaconViewRetriever, beaconHeight uint64, txr basemeta.Transaction) (bool, bool, error) {
	if txr.GetType() == common.TxCustomTokenPrivacyType && reflect.TypeOf(txr).String() == "*transaction.Tx" {
		return true, true, nil
	}
	// validate RedeemerIncAddressStr
	keyWallet, err := wallet.Base58CheckDeserialize(redeemReq.RedeemerIncAddressStr)
	if err != nil {
		return false, false, basemeta.NewMetadataTxError(basemeta.PortalRedeemLiquidateExchangeRatesParamError, errors.New("Address incognito redeem is invalid"))
	}

	incAddr := keyWallet.KeySet.PaymentAddress
	if len(incAddr.Pk) == 0 {
		return false, false, basemeta.NewMetadataTxError(basemeta.PortalRedeemLiquidateExchangeRatesParamError, errors.New("Payment incognito address is invalid"))
	}
	if !bytes.Equal(txr.GetSigPubKey()[:], incAddr.Pk[:]) {
		return false, false, basemeta.NewMetadataTxError(basemeta.PortalRedeemLiquidateExchangeRatesParamError, errors.New("Address incognito redeem is not signer"))
	}

	// check tx type
	if txr.GetType() != common.TxCustomTokenPrivacyType {
		return false, false, errors.New("tx redeem request must be TxCustomTokenPrivacyType")
	}

	if !txr.IsCoinsBurning(chainRetriever, shardViewRetriever, beaconViewRetriever, beaconHeight) {
		return false, false, errors.New("txprivacytoken in tx redeem request must be coin burning tx")
	}

	// validate redeem amount
	minAmount := common.MinAmountPortalPToken[redeemReq.TokenID]
	if redeemReq.RedeemAmount < minAmount {
		return false, false, fmt.Errorf("redeem amount should be larger or equal to %v", minAmount)
	}

	// validate value transfer of tx for redeem amount in ptoken
	if redeemReq.RedeemAmount != txr.CalculateTxValue() {
		return false, false, errors.New("redeem amount should be equal to the tx value")
	}

	// validate tokenID
	if redeemReq.TokenID != txr.GetTokenID().String() {
		return false, false, basemeta.NewMetadataTxError(basemeta.PortalRedeemLiquidateExchangeRatesParamError, errors.New("TokenID in metadata is not matched to tokenID in tx"))
	}
	// check tokenId is portal token or not
	if !IsPortalToken(redeemReq.TokenID) {
		return false, false, basemeta.NewMetadataTxError(basemeta.PortalRedeemLiquidateExchangeRatesParamError, errors.New("TokenID is not in portal tokens list"))
	}

	// reject Redeem Request from Liquidation pool from BCHeightBreakPointPortalV3
	if beaconHeight >= chainRetriever.GetBCHeightBreakPointPortalV3() {
		return false, false, basemeta.NewMetadataTxError(basemeta.PortalRedeemLiquidateExchangeRatesParamError, fmt.Errorf("Should create redeem request from liquidation pool v3 after epoch %v", chainRetriever.GetBCHeightBreakPointPortalV3()))
	}
	return true, true, nil
}

func (redeemReq PortalRedeemLiquidateExchangeRates) ValidateMetadataByItself() bool {
	return redeemReq.Type == basemeta.PortalRedeemFromLiquidationPoolMeta
}

func (redeemReq PortalRedeemLiquidateExchangeRates) Hash() *common.Hash {
	record := redeemReq.MetadataBase.Hash().String()
	record += redeemReq.TokenID
	record += strconv.FormatUint(redeemReq.RedeemAmount, 10)
	record += redeemReq.RedeemerIncAddressStr
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (redeemReq *PortalRedeemLiquidateExchangeRates) BuildReqActions(tx basemeta.Transaction, chainRetriever basemeta.ChainRetriever, shardViewRetriever basemeta.ShardViewRetriever, beaconViewRetriever basemeta.BeaconViewRetriever, shardID byte, shardHeight uint64) ([][]string, error) {
	actionContent := PortalRedeemLiquidateExchangeRatesAction{
		Meta:    *redeemReq,
		TxReqID: *tx.Hash(),
		ShardID: shardID,
	}
	actionContentBytes, err := json.Marshal(actionContent)
	if err != nil {
		return [][]string{}, err
	}
	actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
	action := []string{strconv.Itoa(basemeta.PortalRedeemFromLiquidationPoolMeta), actionContentBase64Str}
	return [][]string{action}, nil
}

func (redeemReq *PortalRedeemLiquidateExchangeRates) CalculateSize() uint64 {
	return basemeta.CalculateSize(redeemReq)
}
