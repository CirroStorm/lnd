package lnwallet

import (
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcwallet/waddrmgr"
	"github.com/lightningnetwork/lnd/keychain"
	"sync/atomic"
)

// MockWalletController is used by the LightningWallet, and let us mock the
// interaction with the bitcoin network.
type MockWalletController struct {
	RootKey               *btcec.PrivateKey
	PrevAddres            btcutil.Address
	PublishedTransactions chan *wire.MsgTx
	Index                 uint32
	DeriveKeyFail                  bool
	LockedOutpoints map[uint32]struct{}
	UnlockedOutpoints map[uint32]struct{}
	TestUtxos []*Utxo
	Outpoints map[uint32]*wire.TxOut
	Outputs []*btcjson.ListUnspentResult
}

func NewMockWalletController(utxos []*btcjson.ListUnspentResult) *MockWalletController {
	m := &MockWalletController{
		LockedOutpoints: make(map[uint32]struct{}),
		UnlockedOutpoints: make(map[uint32]struct{}),
		Outpoints: make(map[uint32]*wire.TxOut),
		Outputs: utxos,
	}

	if utxos != nil {
		for _, utxo := range utxos {
			pkscript, _ := hex.DecodeString(utxo.ScriptPubKey)
			m.Outpoints[utxo.Vout] = &wire.TxOut{
				Value:    int64(utxo.Amount),
				PkScript: pkscript,
			}
		}
	}

	return m
}

// FetchInputInfo will be called to get info about the inputs to the funding
// transaction.
func (m *MockWalletController) FetchInputInfo(
	prevOut *wire.OutPoint) (*wire.TxOut, error) {
	if len(m.Outpoints) != 0 {
		txOut, ok := m.Outpoints[prevOut.Index]
		if !ok {
			return nil, fmt.Errorf("no output found")
		} else {
			return txOut, nil
		}

	} else {
		txOut := &wire.TxOut{
			Value:    int64(10 * btcutil.SatoshiPerBitcoin),
			PkScript: []byte("dummy"),
		}
		return txOut, nil
	}
}

func (*MockWalletController) ConfirmedBalance(confs int32) (btcutil.Amount, error) {
	return 0, nil
}

// NewAddress is called to get new addresses for delivery, change etc.
func (m *MockWalletController) NewAddress(keyScope waddrmgr.KeyScope,
	change bool) (btcutil.Address, error) {
	addr, _ := btcutil.NewAddressPubKey(
		m.RootKey.PubKey().SerializeCompressed(), &chaincfg.MainNetParams)
	return addr, nil
}

func (*MockWalletController) IsOurAddress(a btcutil.Address) bool {
	return false
}

func (*MockWalletController) SendOutputs(outputs []*wire.TxOut,
	_ SatPerKWeight) (*wire.MsgTx, error) {

	return nil, nil
}

func (m *MockWalletController) ListUnspent(minConfs, maxConfs int32) ([]*btcjson.ListUnspentResult, error) {
	if m.Outputs != nil {
		return m.Outputs, nil
	} else {
		utxo := &btcjson.ListUnspentResult{
			Vout: m.Index,
			ScriptPubKey: "0014643d8b15694a547d57336e51dffd38e30e6ef7ef",
			Amount: btcutil.Amount(10 * btcutil.SatoshiPerBitcoin).ToBTC(),
		}
		atomic.AddUint32(&m.Index, 1)
		var ret []*btcjson.ListUnspentResult
		ret = append(ret, utxo)
		return ret, nil
	}
}

func (*MockWalletController) ListTransactionDetails() ([]*TransactionDetail, error) {
	return nil, nil
}

func (m *MockWalletController) LockOutpoint(o wire.OutPoint)   {
	m.LockedOutpoints[o.Index] = struct{}{}
}

func (m *MockWalletController) UnlockOutpoint(o wire.OutPoint) {
	m.UnlockedOutpoints[o.Index] = struct{}{}
}

func (m *MockWalletController) PublishTransaction(tx *wire.MsgTx) error {
	m.PublishedTransactions <- tx
	return nil
}

func (*MockWalletController) SubscribeTransactions() (TransactionSubscription, error) {
	return nil, nil
}

func (*MockWalletController) IsSynced() (bool, int64, error) {
	return true, int64(0), nil
}

func (*MockWalletController) Start() error {
	return nil
}

func (*MockWalletController) Stop() error {
	return nil
}

func (m *MockWalletController) GetRevocationRoot(nextRevocationKeyDesc keychain.KeyDescriptor) (*chainhash.Hash, error) {
	return chainhash.NewHash(m.RootKey.Serialize())
}

func (b *MockWalletController) GetNodeKey() (*btcec.PrivateKey, error) {
	return nil, nil
}

func (m *MockWalletController) DeriveNextKey(keyFam keychain.KeyFamily) (keychain.KeyDescriptor, error) {
	return keychain.KeyDescriptor{
		PubKey: m.RootKey.PubKey(),
	}, nil
}

func (m *MockWalletController) DeriveKey(keyLoc keychain.KeyLocator) (keychain.KeyDescriptor, error) {
	if m.DeriveKeyFail {
		return keychain.KeyDescriptor{}, fmt.Errorf("fail")
	}

	_, pub := btcec.PrivKeyFromBytes(btcec.S256(), testWalletPrivKey)
	return keychain.KeyDescriptor{
		PubKey: pub,
	}, nil
}

func (m *MockWalletController) DerivePrivKey(keyDesc keychain.KeyDescriptor) (*btcec.PrivateKey, error) {
	return m.RootKey, nil
}
