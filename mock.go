package main

import (
	"crypto/sha256"
	"fmt"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"

	"github.com/lightningnetwork/lnd/chainntnfs"
	"github.com/lightningnetwork/lnd/lnwallet"
	"sync"
)

// The block height returned by the mock BlockChainIO's GetBestBlock.
const fundingBroadcastHeight = 123

type mockSigner struct {
	key *btcec.PrivateKey
}

func (m *mockSigner) SignOutputRaw(tx *wire.MsgTx,
	signDesc *input.SignDescriptor) ([]byte, error) {
	amt := signDesc.Output.Value
	witnessScript := signDesc.WitnessScript
	privKey := m.key

	if !privKey.PubKey().IsEqual(signDesc.KeyDesc.PubKey) {
		return nil, fmt.Errorf("incorrect key passed")
	}

	switch {
	case signDesc.SingleTweak != nil:
		privKey = input.TweakPrivKey(privKey,
			signDesc.SingleTweak)
	case signDesc.DoubleTweak != nil:
		privKey = input.DeriveRevocationPrivKey(privKey,
			signDesc.DoubleTweak)
	}

	sig, err := txscript.RawTxInWitnessSignature(tx, signDesc.SigHashes,
		signDesc.InputIndex, amt, witnessScript, signDesc.HashType,
		privKey)
	if err != nil {
		return nil, err
	}

	return sig[:len(sig)-1], nil
}

func (m *mockSigner) ComputeInputScript(tx *wire.MsgTx,
	signDesc *input.SignDescriptor) (*input.Script, error) {

	// TODO(roasbeef): expose tweaked signer from lnwallet so don't need to
	// duplicate this code?

	privKey := m.key

	switch {
	case signDesc.SingleTweak != nil:
		privKey = input.TweakPrivKey(privKey,
			signDesc.SingleTweak)
	case signDesc.DoubleTweak != nil:
		privKey = input.DeriveRevocationPrivKey(privKey,
			signDesc.DoubleTweak)
	}

	witnessScript, err := txscript.WitnessSignature(tx, signDesc.SigHashes,
		signDesc.InputIndex, signDesc.Output.Value, signDesc.Output.PkScript,
		signDesc.HashType, privKey, true)
	if err != nil {
		return nil, err
	}

	return &input.Script{
		Witness: witnessScript,
	}, nil
}

type mockNotfier struct {
	confChannel chan *chainntnfs.TxConfirmation
}

func (m *mockNotfier) RegisterConfirmationsNtfn(txid *chainhash.Hash,
	_ []byte, numConfs, heightHint uint32) (*chainntnfs.ConfirmationEvent, error) {
	return &chainntnfs.ConfirmationEvent{
		Confirmed: m.confChannel,
	}, nil
}
func (m *mockNotfier) RegisterBlockEpochNtfn(
	bestBlock *chainntnfs.BlockEpoch) (*chainntnfs.BlockEpochEvent, error) {
	return &chainntnfs.BlockEpochEvent{
		Epochs: make(chan *chainntnfs.BlockEpoch),
		Cancel: func() {},
	}, nil
}

func (m *mockNotfier) Start() error {
	return nil
}

func (m *mockNotfier) Stop() error {
	return nil
}
func (m *mockNotfier) RegisterSpendNtfn(outpoint *wire.OutPoint, _ []byte,
	heightHint uint32) (*chainntnfs.SpendEvent, error) {
	return &chainntnfs.SpendEvent{
		Spend:  make(chan *chainntnfs.SpendDetail),
		Cancel: func() {},
	}, nil
}

// mockSpendNotifier extends the mockNotifier so that spend notifications can be
// triggered and delivered to subscribers.
type mockSpendNotifier struct {
	*mockNotfier
	spendMap map[wire.OutPoint][]chan *chainntnfs.SpendDetail
	mtx      sync.Mutex
}

func makeMockSpendNotifier() *mockSpendNotifier {
	return &mockSpendNotifier{
		mockNotfier: &mockNotfier{
			confChannel: make(chan *chainntnfs.TxConfirmation),
		},
		spendMap: make(map[wire.OutPoint][]chan *chainntnfs.SpendDetail),
	}
}

func (m *mockSpendNotifier) RegisterSpendNtfn(outpoint *wire.OutPoint,
	_ []byte, heightHint uint32) (*chainntnfs.SpendEvent, error) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	spendChan := make(chan *chainntnfs.SpendDetail)
	m.spendMap[*outpoint] = append(m.spendMap[*outpoint], spendChan)
	return &chainntnfs.SpendEvent{
		Spend: spendChan,
		Cancel: func() {
		},
	}, nil
}

// Spend dispatches SpendDetails to all subscribers of the outpoint. The details
// will include the transaction and height provided by the caller.
func (m *mockSpendNotifier) Spend(outpoint *wire.OutPoint, height int32,
	txn *wire.MsgTx) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	if spendChans, ok := m.spendMap[*outpoint]; ok {
		delete(m.spendMap, *outpoint)
		for _, spendChan := range spendChans {
			txnHash := txn.TxHash()
			spendChan <- &chainntnfs.SpendDetail{
				SpentOutPoint:     outpoint,
				SpendingHeight:    height,
				SpendingTx:        txn,
				SpenderTxHash:     &txnHash,
				SpenderInputIndex: outpoint.Index,
			}
		}
	}
}

type mockPreimageCache struct {
	sync.Mutex
	preimageMap map[[32]byte][]byte
}

func (m *mockPreimageCache) LookupPreimage(hash []byte) ([]byte, bool) {
	m.Lock()
	defer m.Unlock()

	var h [32]byte
	copy(h[:], hash)

	p, ok := m.preimageMap[h]
	return p, ok
}

func (m *mockPreimageCache) AddPreimage(preimage []byte) error {
	m.Lock()
	defer m.Unlock()

	m.preimageMap[sha256.Sum256(preimage[:])] = preimage

	return nil
}
