package lnwallet

import (
	"fmt"
	"github.com/btcsuite/btcd/btcec"
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
	Fail                  bool
}

// FetchInputInfo will be called to get info about the inputs to the funding
// transaction.
func (*MockWalletController) FetchInputInfo(
	prevOut *wire.OutPoint) (*wire.TxOut, error) {
	txOut := &wire.TxOut{
		Value:    int64(10 * btcutil.SatoshiPerBitcoin),
		PkScript: []byte("dummy"),
	}
	return txOut, nil
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

// ListUnspentWitness is called by the wallet when doing coin selection. We just
// need one unspent for the funding transaction.
func (m *MockWalletController) ListUnspentWitness(minconfirms,
	maxconfirms int32) ([]*Utxo, error) {
	utxo := &Utxo{
		KeyScope: waddrmgr.KeyScopeBIP0084,
		Value:    btcutil.Amount(10 * btcutil.SatoshiPerBitcoin),
		PkScript: make([]byte, 22),
		OutPoint: wire.OutPoint{
			Hash:  chainhash.Hash{},
			Index: m.Index,
		},
	}
	atomic.AddUint32(&m.Index, 1)
	var ret []*Utxo
	ret = append(ret, utxo)
	return ret, nil
}
func (*MockWalletController) ListTransactionDetails() ([]*TransactionDetail, error) {
	return nil, nil
}
func (*MockWalletController) LockOutpoint(o wire.OutPoint)   {}
func (*MockWalletController) UnlockOutpoint(o wire.OutPoint) {}
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
	if m.Fail {
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
