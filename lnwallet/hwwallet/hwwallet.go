package hwwallet

import (
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcwallet/chain"
	"math"
	"runtime"
	"sync"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/lightningnetwork/lnd/lnwallet"
)

// HwWallet is an implementation of the lnwallet.WalletController interface
// backed by an active instance of btcwallet. At the time of the writing of
// this documentation, this implementation requires a full btcd node to
// operate.
type HwWallet struct {
	cfg *Config

	chain chain.Interface

	// utxoCache is a cache used to speed up repeated calls to
	// FetchInputInfo.
	utxoCache map[wire.OutPoint]*wire.TxOut
	cacheMtx  sync.RWMutex
}

// A compile time check to ensure that HwWallet implements the
// WalletController interface.
var _ lnwallet.WalletController = (*HwWallet)(nil)

// New returns a new fully initialized instance of HwWallet given a valid
// configuration struct.
func New(cfg Config) (*HwWallet, error) {
	return &HwWallet{
		cfg:       &cfg,
		utxoCache: make(map[wire.OutPoint]*wire.TxOut),
	}, nil
}

// Start initializes the underlying rpc connection, the wallet itself, and
// begins syncing to the current available blockchain state.
//
// This is a part of the WalletController interface.
func (b *HwWallet) Start() error {
	// TODO open rpc connection to wallet
	return nil
}

// Stop signals the wallet for shutdown. Shutdown may entail closing
// any active sockets, database handles, stopping goroutines, etc.
//
// This is a part of the WalletController interface.
func (b *HwWallet) Stop() error {
	// TODO close rpc connection
	pc, _, _, _ := runtime.Caller(1)
	panic(fmt.Sprintf("%s", runtime.FuncForPC(pc).Name()))

	return nil
}

// ConfirmedBalance returns the sum of all the wallet's unspent outputs that
// have at least confs confirmations. If confs is set to zero, then all unspent
// outputs, including those currently in the mempool will be included in the
// final sum.
//
// This is a part of the WalletController interface.
func (b *HwWallet) ConfirmedBalance(confs int32) (btcutil.Amount, error) {
	var balance btcutil.Amount

	witnessOutputs, err := b.ListUnspentWitness(confs, math.MaxInt32)
	if err != nil {
		return 0, err
	}

	for _, witnessOutput := range witnessOutputs {
		balance += witnessOutput.Value
	}

	return balance, nil
}

// NewAddress returns the next external or internal address for the wallet
// dictated by the value of the `change` parameter. If change is true, then an
// internal address will be returned, otherwise an external address should be
// returned.
//
// This is a part of the WalletController interface.
func (b *HwWallet) NewAddress(t lnwallet.AddressType, change bool) (btcutil.Address, error) {
	// TODO call rpc
	pc, _, _, _ := runtime.Caller(1)
	panic(fmt.Sprintf("%s", runtime.FuncForPC(pc).Name()))
	return nil, nil
}

// IsOurAddress checks if the passed address belongs to this wallet
//
// This is a part of the WalletController interface.
func (b *HwWallet) IsOurAddress(a btcutil.Address) bool {
	// TODO call rpc
	pc, _, _, _ := runtime.Caller(1)
	panic(fmt.Sprintf("%s", runtime.FuncForPC(pc).Name()))
	return false
}

// SendOutputs funds, signs, and broadcasts a Bitcoin transaction paying out to
// the specified outputs. In the case the wallet has insufficient funds, or the
// outputs are non-standard, a non-nil error will be returned.
//
// This is a part of the WalletController interface.
func (b *HwWallet) SendOutputs(outputs []*wire.TxOut,
	feeRate lnwallet.SatPerKWeight) (*wire.MsgTx, error) {

	// Convert our fee rate from sat/kw to sat/kb since it's required by
	// SendOutputs.
	//feeSatPerKB := btcutil.Amount(feeRate.FeePerKVByte())

	// TODO call rpc
	pc, _, _, _ := runtime.Caller(1)
	panic(fmt.Sprintf("%s", runtime.FuncForPC(pc).Name()))
	return nil, nil
}

// LockOutpoint marks an outpoint as locked meaning it will no longer be deemed
// as eligible for coin selection. Locking outputs are utilized in order to
// avoid race conditions when selecting inputs for usage when funding a
// channel.
//
// This is a part of the WalletController interface.
func (b *HwWallet) LockOutpoint(o wire.OutPoint) {
	// TODO call rpc
	pc, _, _, _ := runtime.Caller(1)
	panic(fmt.Sprintf("%s", runtime.FuncForPC(pc).Name()))
}

// UnlockOutpoint unlocks a previously locked output, marking it eligible for
// coin selection.
//
// This is a part of the WalletController interface.
func (b *HwWallet) UnlockOutpoint(o wire.OutPoint) {
	// TODO call rpc
	pc, _, _, _ := runtime.Caller(1)
	panic(fmt.Sprintf("%s", runtime.FuncForPC(pc).Name()))
}

// ListUnspentWitness returns a slice of all the unspent outputs the wallet
// controls which pay to witness programs either directly or indirectly.
//
// This is a part of the WalletController interface.
func (b *HwWallet) ListUnspentWitness(minConfs, maxConfs int32) (
	[]*lnwallet.Utxo, error) {

	var unspentOutputs []*btcjson.ListUnspentResult

	// TODO call rpc to get unspent outputs

	pc, _, _, _ := runtime.Caller(1)
	panic(fmt.Sprintf("%s", runtime.FuncForPC(pc).Name()))

	// Next, we'll run through all the regular outputs, only saving those
	// which are p2wkh outputs or a p2wsh output nested within a p2sh output.
	witnessOutputs := make([]*lnwallet.Utxo, 0, len(unspentOutputs))
	for _, output := range unspentOutputs {
		pkScript, err := hex.DecodeString(output.ScriptPubKey)
		if err != nil {
			return nil, err
		}

		var addressType lnwallet.AddressType
		if txscript.IsPayToWitnessPubKeyHash(pkScript) {
			addressType = lnwallet.WitnessPubKey
		} else if txscript.IsPayToScriptHash(pkScript) {
			// TODO(roasbeef): This assumes all p2sh outputs returned by the
			// wallet are nested p2pkh. We can't check the redeem script because
			// the btcwallet service does not include it.
			addressType = lnwallet.NestedWitnessPubKey
		}

		if addressType == lnwallet.WitnessPubKey ||
			addressType == lnwallet.NestedWitnessPubKey {

			txid, err := chainhash.NewHashFromStr(output.TxID)
			if err != nil {
				return nil, err
			}

			// We'll ensure we properly convert the amount given in
			// BTC to satoshis.
			amt, err := btcutil.NewAmount(output.Amount)
			if err != nil {
				return nil, err
			}

			utxo := &lnwallet.Utxo{
				AddressType: addressType,
				Value:       amt,
				PkScript:    pkScript,
				OutPoint: wire.OutPoint{
					Hash:  *txid,
					Index: output.Vout,
				},
				Confirmations: output.Confirmations,
			}
			witnessOutputs = append(witnessOutputs, utxo)
		}

	}

	return witnessOutputs, nil
}

// PublishTransaction performs cursory validation (dust checks, etc), then
// finally broadcasts the passed transaction to the Bitcoin network. If
// publishing the transaction fails, an error describing the reason is
// returned (currently ErrDoubleSpend). If the transaction is already
// published to the network (either in the mempool or chain) no error
// will be returned.
func (b *HwWallet) PublishTransaction(tx *wire.MsgTx) error {
	// TODO call RPC
	pc, _, _, _ := runtime.Caller(1)
	panic(fmt.Sprintf("%s", runtime.FuncForPC(pc).Name()))
	//var err error
	//return b.chain.CheckPublishTransactionResult(err);
}

// ListTransactionDetails returns a list of all transactions which are
// relevant to the wallet.
//
// This is a part of the WalletController interface.
func (b *HwWallet) ListTransactionDetails() ([]*lnwallet.TransactionDetail, error) {
	// TODO call rcp
	pc, _, _, _ := runtime.Caller(1)
	panic(fmt.Sprintf("%s", runtime.FuncForPC(pc).Name()))
	return nil, nil
}

// txSubscriptionClient encapsulates the transaction notification client from
// the base wallet. Notifications received from the client will be proxied over
// two distinct channels.
type txSubscriptionClient struct {
	confirmed   chan *lnwallet.TransactionDetail
	unconfirmed chan *lnwallet.TransactionDetail
}

// ConfirmedTransactions returns a channel which will be sent on as new
// relevant transactions are confirmed.
//
// This is part of the TransactionSubscription interface.
func (t *txSubscriptionClient) ConfirmedTransactions() chan *lnwallet.TransactionDetail {
	return t.confirmed
}

// UnconfirmedTransactions returns a channel which will be sent on as
// new relevant transactions are seen within the network.
//
// This is part of the TransactionSubscription interface.
func (t *txSubscriptionClient) UnconfirmedTransactions() chan *lnwallet.TransactionDetail {
	return t.unconfirmed
}

// Cancel finalizes the subscription, cleaning up any resources allocated.
//
// This is part of the TransactionSubscription interface.
func (t *txSubscriptionClient) Cancel() {
}

// SubscribeTransactions returns a TransactionSubscription client which
// is capable of receiving async notifications as new transactions
// related to the wallet are seen within the network, or found in
// blocks.
//
// This is a part of the WalletController interface.
func (b *HwWallet) SubscribeTransactions() (lnwallet.TransactionSubscription, error) {
	txClient := &txSubscriptionClient{
		confirmed:   make(chan *lnwallet.TransactionDetail),
		unconfirmed: make(chan *lnwallet.TransactionDetail),
	}

	return txClient, nil
}

// IsSynced returns a boolean indicating if from the PoV of the wallet,
// it has fully synced to the current best block in the main chain.
//
// This is a part of the WalletController interface.
func (b *HwWallet) IsSynced() (bool, int64, error) {
	// TODO call rpc
	pc, _, _, _ := runtime.Caller(1)
	panic(fmt.Sprintf("%s", runtime.FuncForPC(pc).Name()))
	return false, 0, nil
}
