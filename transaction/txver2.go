package transaction

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/incognitochain/incognito-chain/privacy/coin"
	errhandler "github.com/incognitochain/incognito-chain/privacy/errorhandler"
	"github.com/incognitochain/incognito-chain/privacy/key"

	"github.com/incognitochain/incognito-chain/common/base58"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/privacy/operation"
	"github.com/incognitochain/incognito-chain/privacy/privacy_v2"
	"github.com/incognitochain/incognito-chain/privacy/privacy_v2/mlsag"
)

type TxVersion2 struct{}

func generateMlsagRing(inp *[]*coin.InputCoin, out *[]*coin.OutputCoin, params *TxPrivacyInitParams, pi int, shardID byte) (*mlsag.Ring, error) {
	inputCoins := *inp
	outputCoins := *out

	// loop to create list usable commitments from usableInputCoins
	listUsableCommitments := make(map[common.Hash][]byte)
	listUsableCommitmentsIndices := make([]common.Hash, len(inputCoins))
	// tick index of each usable commitment with full db commitments
	mapIndexCommitmentsInUsableTx := make(map[string]*big.Int)
	for i, in := range inputCoins {
		usableCommitment := in.CoinDetails.GetCoinCommitment().ToBytesS()
		commitmentInHash := common.HashH(usableCommitment)
		listUsableCommitments[commitmentInHash] = usableCommitment
		listUsableCommitmentsIndices[i] = commitmentInHash
		index, err := statedb.GetCommitmentIndex(params.stateDB, *params.tokenID, usableCommitment, shardID)
		if err != nil {
			Logger.Log.Error(err)
			return nil, err
		}
		commitmentInBase58Check := base58.Base58Check{}.Encode(usableCommitment, common.ZeroByte)
		mapIndexCommitmentsInUsableTx[commitmentInBase58Check] = index
	}
	lenCommitment, err := statedb.GetCommitmentLength(params.stateDB, *params.tokenID, shardID)
	if err != nil {
		Logger.Log.Error(err)
		return nil, err
	}
	if lenCommitment == nil {
		Logger.Log.Error(errors.New("Commitments is empty"))
		return nil, errors.New("Commitments is empty")
	}

	outputCommitments := new(operation.Point).Identity()
	for i := 0; i < len(outputCoins); i += 1 {
		outputCommitments.Add(outputCommitments, outputCoins[i].CoinDetails.GetCoinCommitment())
	}

	ring := make([][]*operation.Point, privacy.RingSize)
	key := params.senderSK
	for i := 0; i < privacy.RingSize; i += 1 {
		sumInputs := new(operation.Point).Identity()
		row := make([]*operation.Point, len(inputCoins))
		if i == pi {
			for j := 0; j < len(inputCoins); j += 1 {
				privKey := new(operation.Scalar).FromBytesS(*key)
				row[j] = new(operation.Point).ScalarMultBase(privKey)
				sumInputs.Add(sumInputs, inputCoins[j].CoinDetails.GetCoinCommitment())
			}
		} else {
			for j := 0; j < len(inputCoins); j += 1 {
				for {
					index, _ := common.RandBigIntMaxRange(lenCommitment)
					ok, err := statedb.HasCommitmentIndex(params.stateDB, *params.tokenID, index.Uint64(), shardID)
					if ok && err == nil {
						commitment, publicKey, _ := statedb.GetCommitmentAndPublicKeyByIndex(params.stateDB, *params.tokenID, index.Uint64(), shardID)
						if _, found := listUsableCommitments[common.HashH(commitment)]; found {
							if lenCommitment.Uint64() == 1 && len(inputCoins) == 1 {
								commitment = privacy.RandomPoint().ToBytesS()
								publicKey = privacy.RandomPoint().ToBytesS()
							} else {
								continue
							}
						}
						row[j], err = new(operation.Point).FromBytesS(publicKey)
						if err != nil {
							return nil, err
						}

						temp, err := new(operation.Point).FromBytesS(commitment)
						if err != nil {
							return nil, err
						}

						sumInputs.Add(sumInputs, temp)
						break
					} else {
						return nil, err
					}
				}
			}
		}
		row = append(row, sumInputs.Sub(sumInputs, outputCommitments))
		ring[i] = row
	}
	mlsagring := mlsag.NewRing(ring)
	return mlsagring, nil
}

func createPrivKeyMlsag(inp *[]*coin.InputCoin, out *[]*coin.OutputCoin, senderSK *key.PrivateKey) *[]*operation.Scalar {
	inputCoins := *inp
	outputCoins := *out

	sumRand := new(operation.Scalar).FromUint64(0)
	for _, in := range inputCoins {
		sumRand.Add(sumRand, in.CoinDetails.GetRandomness())
	}
	for _, out := range outputCoins {
		sumRand.Add(sumRand, out.CoinDetails.GetRandomness())
	}

	sk := new(operation.Scalar).FromBytesS(*senderSK)
	privKeyMlsag := make([]*operation.Scalar, len(inputCoins)+1)
	for i := 0; i < len(inputCoins); i += 1 {
		privKeyMlsag[i] = sk
	}
	privKeyMlsag[len(inputCoins)] = sumRand
	return &privKeyMlsag
}

// signTx - signs tx
func signTxVer2(inp *[]*coin.InputCoin, out *[]*coin.OutputCoin, tx *Tx, params *TxPrivacyInitParams) error {
	if tx.Sig != nil {
		return NewTransactionErr(UnexpectedError, errors.New("input transaction must be an unsigned one"))
	}

	var pi int = common.RandIntInterval(0, privacy.RingSize-1)
	shardID := common.GetShardIDFromLastByte(tx.PubKeyLastByteSender)
	ring, err := generateMlsagRing(inp, out, params, pi, shardID)
	if err != nil {
		return err
	}
	privKeysMlsag := *createPrivKeyMlsag(inp, out, params.senderSK)

	sag := mlsag.NewMlsag(privKeysMlsag, ring, pi)

	tx.sigPrivKey, err = privacy.ArrayScalarToBytes(&privKeysMlsag)
	if err != nil {
		return err
	}

	tx.SigPubKey, err = ring.ToBytes()
	if err != nil {
		return err
	}

	message := tx.Proof.Bytes()
	mlsagSignature, err := sag.Sign(message)
	if err != nil {
		return err
	}

	tx.Sig, err = mlsagSignature.ToBytes()
	check, err := mlsag.Verify(mlsagSignature, ring, message)

	fmt.Println("After proving")
	fmt.Println("After proving")
	fmt.Println("After proving")
	fmt.Println(check)
	fmt.Println(check)
	fmt.Println(err)
	fmt.Println(err)
	return err
}

func (*TxVersion2) Prove(tx *Tx, params *TxPrivacyInitParams) error {
	outputCoins, err := parseOutputCoins(params)
	if err != nil {
		return err
	}
	for i := 0; i < len(*outputCoins); i += 1 {
		(*outputCoins)[i].CoinDetails.SetRandomness(operation.RandomScalar())
		err := (*outputCoins)[i].CoinDetails.CommitValueRandomness()
		if err != nil {
			return err
		}
	}
	inputCoins := &params.inputCoins

	tx.Proof, err = privacy_v2.Prove(inputCoins, outputCoins, params.hasPrivacy, &params.paymentInfo)
	if err != nil {
		return err
	}

	err = signTxVer2(inputCoins, outputCoins, tx, params)
	return err
}

func (txVer2 *TxVersion2) ProveASM(tx *Tx, params *TxPrivacyInitParamsForASM) error {
	return txVer2.Prove(tx, &params.txParam)
}

// verifySigTx - verify signature on tx
func verifySigTxVer2(tx *Tx) (bool, error) {
	// check input transaction
	if tx.Sig == nil || tx.SigPubKey == nil {
		return false, NewTransactionErr(UnexpectedError, errors.New("input transaction must be an signed one"))
	}
	var err error

	ring, err := new(mlsag.Ring).FromBytes(tx.SigPubKey)
	if err != nil {
		return false, err
	}

	txSig, err := new(mlsag.MlsagSig).FromBytes(tx.Sig)
	if err != nil {
		return false, err
	}

	message := tx.Proof.Bytes()

	fmt.Println("Verifying")
	fmt.Println("Verifying")
	fmt.Println("Verifying")
	fmt.Println(txSig)
	fmt.Println(tx.SigPubKey)
	fmt.Println(message)
	return mlsag.Verify(txSig, ring, message)
}

// TODO privacy
func (*TxVersion2) Verify(tx *Tx, hasPrivacy bool, transactionStateDB *statedb.StateDB, bridgeStateDB *statedb.StateDB, shardID byte, tokenID *common.Hash, isBatch bool, isNewTransaction bool) (bool, error) {
	var valid bool
	var err error

	if valid, err := verifySigTxVer2(tx); !valid {
		if err != nil {
			Logger.Log.Errorf("Error verifying signature ver2 with tx hash %s: %+v \n", tx.Hash().String(), err)
			return false, NewTransactionErr(VerifyTxSigFailError, err)
		}
		Logger.Log.Errorf("FAILED VERIFICATION SIGNATURE ver2 with tx hash %s", tx.Hash().String())
		return false, NewTransactionErr(VerifyTxSigFailError, fmt.Errorf("FAILED VERIFICATION SIGNATURE ver2 with tx hash %s", tx.Hash().String()))
	}

	if tx.Proof == nil {
		return true, nil
	}

	tokenID, err = parseTokenID(tokenID)
	if err != nil {
		return false, err
	}

	// Wonder if ver 2 needs this
	// if isNewTransaction {
	// 	for i := 0; i < len(outputCoins); i++ {
	// 		// Check output coins' SND is not exists in SND list (Database)
	// 		if ok, err := CheckSNDerivatorExistence(tokenID, outputCoins[i].CoinDetails.GetSNDerivator(), transactionStateDB); ok || err != nil {
	// 			if err != nil {
	// 				Logger.Log.Error(err)
	// 			}
	// 			Logger.Log.Errorf("snd existed: %d\n", i)
	// 			return false, NewTransactionErr(SndExistedError, err, fmt.Sprintf("snd existed: %d\n", i))
	// 		}
	// 	}
	// }

	// Verify the payment proof
	var txProofV2 *privacy.ProofV2 = tx.Proof.(*privacy.ProofV2)
	valid, err = txProofV2.Verify(hasPrivacy, tx.SigPubKey, tx.Fee, shardID, tokenID, isBatch, nil)

	if !valid {
		if err != nil {
			Logger.Log.Error(err)
		}
		Logger.Log.Error("FAILED VERIFICATION PAYMENT PROOF VER 2")
		err1, ok := err.(*privacy.PrivacyError)
		if ok {
			// parse error detail
			if err1.Code == privacy.ErrCodeMessage[errhandler.VerifyOneOutOfManyProofFailedErr].Code {
				if isNewTransaction {
					return false, NewTransactionErr(VerifyOneOutOfManyProofFailedErr, err1, tx.Hash().String())
				} else {
					// for old txs which be get from sync block or validate new block
					if tx.LockTime <= ValidateTimeForOneoutOfManyProof {
						// only verify by sign on block because of issue #504(that mean we should pass old tx, which happen before this issue)
						return true, nil
					} else {
						return false, NewTransactionErr(VerifyOneOutOfManyProofFailedErr, err1, tx.Hash().String())
					}
				}
			}
		}
		return false, NewTransactionErr(TxProofVerifyFailError, err, tx.Hash().String())
	}
	Logger.Log.Debugf("SUCCESSED VERIFICATION PAYMENT PROOF ")
	return true, nil
}