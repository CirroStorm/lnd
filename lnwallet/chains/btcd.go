package chains

import (
	"encoding/hex"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcwallet/chain"
	"github.com/lightningnetwork/lnd/lnwallet"
	"strings"
)

type BtcdChain struct {
	*chain.RPCClient
}

func NewBtcdChain(client *chain.RPCClient) *BtcdChain {
	return &BtcdChain{client}
}

// GetUtxo returns the original output referenced by the passed outpoint that
// creates the target pkScript.
//
// This method is a part of the lnwallet.BlockChainIO interface.
func (b *BtcdChain) GetUtxo(op *wire.OutPoint, pkScript []byte,
	heightHint uint32) (*wire.TxOut, error) {

	txout, err := b.GetTxOut(&op.Hash, op.Index, false)
	if err != nil {
		return nil, err
	} else if txout == nil {
		return nil, errOutputSpent
	}

	pkScript, err = hex.DecodeString(txout.ScriptPubKey.Hex)
	if err != nil {
		return nil, err
	}

	// We'll ensure we properly convert the amount given in BTC to
	// satoshis.
	amt, err := btcutil.NewAmount(txout.Value)
	if err != nil {
		return nil, err
	}

	return &wire.TxOut{
		Value:    int64(amt),
		PkScript: pkScript,
	}, nil
}

// SupportsUnconfirmedTransactions indicates if the backend supports unconfirmed
// transactions
//
// This method is a part of the lnwallet.BlockChainIO interface.
func (b *BtcdChain) SupportsUnconfirmedTransactions() bool {
	return true
}

// Backend specific code to ensure the backend has started
// This is currently only used for tests
//
// This method is a part of the lnwallet.BlockChainIO interface.
func (b *BtcdChain) WaitForBackendToStart() {
}

func (b *BtcdChain) ReturnPublishTransactionError(err error) error {
	if strings.Contains(err.Error(), "already have") {
		// Transaction was already in the mempool, do
		// not treat as an error. We do this to mimic
		// the behaviour of bitcoind, which will not
		// return an error if a transaction in the
		// mempool is sent again using the
		// sendrawtransaction RPC call.
		return nil
	}
	if strings.Contains(err.Error(), "already exists") {
		// Transaction was already mined, we don't
		// consider this an error.
		return nil
	}
	if strings.Contains(err.Error(), "already spent") {
		// Output was already spent.
		return lnwallet.ErrDoubleSpend
	}
	if strings.Contains(err.Error(), "already been spent") {
		// Output was already spent.
		return lnwallet.ErrDoubleSpend
	}
	if strings.Contains(err.Error(), "orphan transaction") {
		// Transaction is spending either output that
		// is missing or already spent.
		return lnwallet.ErrDoubleSpend
	}

	return err
}

func (b *BtcdChain) GetBackend() chain.Interface {
	return b
}
