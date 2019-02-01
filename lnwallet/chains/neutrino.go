package chains

import (
	"errors"
	"strings"
	"time"

	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcwallet/chain"
	"github.com/btcsuite/btcwallet/waddrmgr"
	"github.com/lightninglabs/neutrino"
	"github.com/lightningnetwork/lnd/lnwallet"
)

var (
	// errOutputSpent is returned by the GetUtxo method if the target output
	// for lookup has already been spent.
	errOutputSpent = errors.New("target output has been spent")

	// errOutputNotFound signals that the desired output could not be
	// located.
	errOutputNotFound = errors.New("target output was not found")
)

type NeutrinoChain struct {
	*chain.NeutrinoClient
}

func NewNeutrinoChain(client *chain.NeutrinoClient) *NeutrinoChain {
	return &NeutrinoChain{client}
}

// GetUtxo returns the original output referenced by the passed outpoint that
// creates the target pkScript.
//
// This method is a part of the lnwallet.BlockChainIO interface.
func (b *NeutrinoChain) GetUtxo(op *wire.OutPoint, pkScript []byte,
	heightHint uint32) (*wire.TxOut, error) {

	// TODO this is the only backend that needs pkScript passed in, is it required at all??

	spendReport, err := b.CS.GetUtxo(
		neutrino.WatchInputs(neutrino.InputWithScript{
			OutPoint: *op,
			PkScript: pkScript,
		}),
		neutrino.StartBlock(&waddrmgr.BlockStamp{
			Height: int32(heightHint),
		}),
	)
	if err != nil {
		return nil, err
	}

	// If the spend report is nil, then the output was not found in
	// the rescan.
	if spendReport == nil {
		return nil, errOutputNotFound
	}

	// If the spending transaction is populated in the spend report,
	// this signals that the output has already been spent.
	if spendReport.SpendingTx != nil {
		return nil, errOutputSpent
	}

	// Otherwise, the output is assumed to be in the UTXO.
	return spendReport.Output, nil
}

// SupportsUnconfirmedTransactions indicates if the backend supports unconfirmed
// transactions
//
// This method is a part of the lnwallet.BlockChainIO interface.
func (b *NeutrinoChain) SupportsUnconfirmedTransactions() bool {
	return false
}

// WaitForBackendToStart ensures the backend has started
// This is currently only used for tests
//
// This method is a part of the lnwallet.BlockChainIO interface.
func (b *NeutrinoChain) WaitForBackendToStart() {
	time.Sleep(time.Second)
}

func (b *NeutrinoChain) ReturnPublishTransactionError(err error) error {
	if strings.Contains(err.Error(), "already have") {
		// Transaction was already in the mempool, do
		// not treat as an error.
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

	return err
}

func (b *NeutrinoChain) GetBackend() chain.Interface {
	return b
}
