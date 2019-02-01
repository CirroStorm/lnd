package chains

import (
	"encoding/hex"
	"github.com/btcsuite/btcwallet/chain"
	"strings"

	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/lightningnetwork/lnd/lnwallet"
)

type BitcoindChain struct {
	*chain.BitcoindClient
}

func NewBitcoindChain(client *chain.BitcoindClient) *BitcoindChain {
	return &BitcoindChain{client}
}

// GetUtxo returns the original output referenced by the passed outpoint that
// creates the target pkScript.
//
// This method is a part of the lnwallet.BlockChainIO interface.
func (b *BitcoindChain) GetUtxo(op *wire.OutPoint, pkScript []byte,
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

	// Sadly, gettxout returns the output value in BTC instead of
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
func (b *BitcoindChain) SupportsUnconfirmedTransactions() bool {
	return true
}

// WaitForBackendToStart ensures the backend has started
// This is currently only used for tests
//
// This method is a part of the lnwallet.BlockChainIO interface.
func (b *BitcoindChain) WaitForBackendToStart() {
}

func (b *BitcoindChain) ReturnPublishTransactionError(err error) error {
	if strings.Contains(err.Error(), "txn-already-in-mempool") {
		// Transaction in mempool, treat as non-error.
		return nil
	}
	if strings.Contains(err.Error(), "txn-already-known") {
		// Transaction in mempool, treat as non-error.
		return nil
	}
	if strings.Contains(err.Error(), "already in block") {
		// Transaction was already mined, we don't
		// consider this an error.
		return nil
	}
	if strings.Contains(err.Error(), "txn-mempool-conflict") {
		// Output was spent by other transaction
		// already in the mempool.
		return lnwallet.ErrDoubleSpend
	}
	if strings.Contains(err.Error(), "insufficient fee") {
		// RBF enabled transaction did not have enough fee.
		return lnwallet.ErrDoubleSpend
	}
	if strings.Contains(err.Error(), "Missing inputs") {
		// Transaction is spending either output that
		// is missing or already spent.
		return lnwallet.ErrDoubleSpend
	}

	return err
}

func (b *BitcoindChain) GetBackend() chain.Interface {
	return b
}
