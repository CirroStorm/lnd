package sweep

import (
	"bytes"
	"github.com/btcsuite/btcd/btcjson"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
	"github.com/lightningnetwork/lnd/lnwallet"
)

// TestDetermineFeePerKw tests that given a fee preference, the
// DetermineFeePerKw will properly map it to a concrete fee in sat/kw.
func TestDetermineFeePerKw(t *testing.T) {
	t.Parallel()

	defaultFee := lnwallet.SatPerKWeight(999)
	relayFee := lnwallet.SatPerKWeight(300)

	feeEstimator := newMockFeeEstimator(defaultFee, relayFee)

	// We'll populate two items in the internal map which is used to query
	// a fee based on a confirmation target: the default conf target, and
	// an arbitrary conf target. We'll ensure below that both of these are
	// properly
	feeEstimator.blocksToFee[50] = 300
	feeEstimator.blocksToFee[defaultNumBlocksEstimate] = 1000

	testCases := []struct {
		// feePref is the target fee preference for this case.
		feePref FeePreference

		// fee is the value the DetermineFeePerKw should return given
		// the FeePreference above
		fee lnwallet.SatPerKWeight

		// fail determines if this test case should fail or not.
		fail bool
	}{
		// A fee rate below the fee rate floor should output the floor.
		{
			feePref: FeePreference{
				FeeRate: lnwallet.SatPerKWeight(99),
			},
			fee: lnwallet.FeePerKwFloor,
		},

		// A fee rate above the floor, should pass through and return
		// the target fee rate.
		{
			feePref: FeePreference{
				FeeRate: 900,
			},
			fee: 900,
		},

		// A specified confirmation target should cause the function to
		// query the estimator which will return our value specified
		// above.
		{
			feePref: FeePreference{
				ConfTarget: 50,
			},
			fee: 300,
		},

		// If the caller doesn't specify any values at all, then we
		// should query for the default conf target.
		{
			feePref: FeePreference{},
			fee:     1000,
		},

		// Both conf target and fee rate are set, we should return with
		// an error.
		{
			feePref: FeePreference{
				ConfTarget: 50,
				FeeRate:    90000,
			},
			fee:  300,
			fail: true,
		},
	}
	for i, testCase := range testCases {
		targetFee, err := DetermineFeePerKw(
			feeEstimator, testCase.feePref,
		)
		switch {
		case testCase.fail && err != nil:
			continue

		case testCase.fail && err == nil:
			t.Fatalf("expected failure for #%v", i)

		case !testCase.fail && err != nil:
			t.Fatalf("unable to estimate fee; %v", err)
		}

		if targetFee != testCase.fee {
			t.Fatalf("#%v: wrong fee: expected %v got %v", i,
				testCase.fee, targetFee)
		}
	}
}

var sweepScript = []byte{
	0x0, 0x14, 0x64, 0x3d, 0x8b, 0x15, 0x69, 0x4a, 0x54,
	0x7d, 0x57, 0x33, 0x6e, 0x51, 0xdf, 0xfd, 0x38, 0xe3,
	0xe, 0x6e, 0xf8, 0xef,
}

var deliveryAddr = func() btcutil.Address {
	_, addrs, _, err := txscript.ExtractPkScriptAddrs(
		sweepScript, &chaincfg.TestNet3Params,
	)
	if err != nil {
		panic(err)
	}

	return addrs[0]
}()

var testUtxos = []*btcjson.ListUnspentResult{
	{
		// A p2wkh output.
		TxID: "1",
		Vout: 1,
		Amount: 1000,
		ScriptPubKey: "0014643d8b15694a547d57336e51dffd38e30e6ef7ef",
	},
	{
		// A np2wkh output.
		TxID: "2",
		Vout: 2,
		Amount: 2000,
		ScriptPubKey: "a9149717f7d15f6f8b07e3584319b97ea92018c317d787",
	},
	{
		// A p2wsh output.
		TxID: "3",
		Vout: 3,
		Amount: 3000,
		ScriptPubKey: "0020701a8d401c84fb13e6baf169d59684e27abd9fa216c8bc5b9fc63d622ff8c58c",
	},
}

func assertUtxosLocked(t *testing.T, wc *lnwallet.MockWalletController,
	utxos []*btcjson.ListUnspentResult) {

	t.Helper()

	for _, utxo := range utxos {
		if _, ok := wc.LockedOutpoints[utxo.Vout]; !ok {
			t.Fatalf("utxo %v was never locked", utxo.Vout)
		}
	}

}

func assertNoUtxosUnlocked(t *testing.T, wc *lnwallet.MockWalletController,
	utxos []*btcjson.ListUnspentResult) {

	t.Helper()

	if len(wc.UnlockedOutpoints) != 0 {
		t.Fatalf("outputs have been locked, but shouldn't have been")
	}
}

func assertUtxosUnlocked(t *testing.T, wc *lnwallet.MockWalletController,
	utxos []*btcjson.ListUnspentResult) {

	t.Helper()

	for _, utxo := range utxos {
		if _, ok := wc.UnlockedOutpoints[utxo.Vout]; !ok {
			t.Fatalf("utxo %v was never unlocked", utxo.Vout)
		}
	}
}

func assertUtxosLockedAndUnlocked(t *testing.T, wc *lnwallet.MockWalletController,
	utxos []*btcjson.ListUnspentResult) {

	t.Helper()

	for _, utxo := range utxos {
		if _, ok := wc.LockedOutpoints[utxo.Vout]; !ok {
			t.Fatalf("utxo %v was never locked", utxo.Vout)
		}

		if _, ok := wc.UnlockedOutpoints[utxo.Vout]; !ok {
			t.Fatalf("utxo %v was never unlocked", utxo.Vout)
		}
	}
}

// TestCraftSweepAllTxCoinSelectFail tests that if coin selection fails, then
// we unlock any outputs we may have locked in the passed closure.
func TestCraftSweepAllTxCoinSelectFail(t *testing.T) {
	t.Parallel()

	wc := lnwallet.NewMockWalletController(testUtxos)
	cfg := lnwallet.Config{}
	cfg.WalletController = wc
	wallet, _ := lnwallet.NewLightningWallet(cfg)
	wallet.CoinSelectLockFail = true

	_, err := CraftSweepAllTx(
		0, 100, nil, wallet, nil, nil,
	)

	// Since we instructed the coin select locker to fail above, we should
	// get an error.
	if err == nil {
		t.Fatalf("sweep tx should have failed: %v", err)
	}

	// At this point, we'll now verify that all outputs were initially
	// locked, and then also unlocked due to the failure.
	assertUtxosLockedAndUnlocked(t, wc, testUtxos)
}

// TestCraftSweepAllTxUnknownWitnessType tests that if one of the inputs we
// encounter is of an unknown witness type, then we fail and unlock any prior
// locked outputs.
func TestCraftSweepAllTxUnknownWitnessType(t *testing.T) {
	t.Parallel()

	wc := lnwallet.NewMockWalletController(testUtxos)
	cfg := lnwallet.Config{}
	cfg.WalletController = wc
	wallet, _ := lnwallet.NewLightningWallet(cfg)

	_, err := CraftSweepAllTx(
		0, 100, nil, wallet, nil, nil,
	)

	// Since passed in a p2wsh output, which is unknown, we should fail to
	// map the output to a witness type.
	if err == nil {
		t.Fatalf("sweep tx should have failed: %v", err)
	}

	// At this point, we'll now verify that all outputs were initially
	// locked, and then also unlocked since we weren't able to find a
	// witness type for the last output.
	assertUtxosLockedAndUnlocked(t, wc, testUtxos)
}

// TestCraftSweepAllTx tests that we'll properly lock all available outputs
// within the wallet, and craft a single sweep transaction that pays to the
// target output.
func TestCraftSweepAllTx(t *testing.T) {
	t.Parallel()

	// First, we'll make a mock signer along with a fee estimator, We'll
	// use zero fees to we can assert a precise output value.
	signer := &mockSigner{}
	feeEstimator := newMockFeeEstimator(0, 0)

	// For our UTXO source, we'll pass in all the UTXOs that we know of,
	// other than the final one which is of an unknown witness type.
	targetUTXOs := testUtxos[:2]

	wc := lnwallet.NewMockWalletController(targetUTXOs)
	cfg := lnwallet.Config{}
	cfg.WalletController = wc
	wallet, _ := lnwallet.NewLightningWallet(cfg)

	sweepPkg, err := CraftSweepAllTx(
		0, 100, deliveryAddr, wallet,
		feeEstimator, signer,
	)
	if err != nil {
		t.Fatalf("unable to make sweep tx: %v", err)
	}

	// At this point, all of the UTXOs that we made above should be locked
	// and none of them unlocked.
	assertUtxosLocked(t, wc, testUtxos[:2])
	assertNoUtxosUnlocked(t, wc, testUtxos[:2])

	// Now that we have our sweep transaction, we should find that we have
	// a UTXO for each input, and also that our final output value is the
	// sum of all our inputs.
	sweepTx := sweepPkg.SweepTx
	if len(sweepTx.TxIn) != len(targetUTXOs) {
		t.Fatalf("expected %v utxo, got %v", len(targetUTXOs),
			len(sweepTx.TxIn))
	}

	// We should have a single output that pays to our sweep script
	// generated above.
	expectedSweepValue := int64(3000)
	if len(sweepTx.TxOut) != 1 {
		t.Fatalf("should have %v outputs, instead have %v", 1,
			len(sweepTx.TxOut))
	}
	output := sweepTx.TxOut[0]
	switch {
	case output.Value != expectedSweepValue:
		t.Fatalf("expected %v sweep value, instead got %v",
			expectedSweepValue, output.Value)

	case !bytes.Equal(sweepScript, output.PkScript):
		t.Fatalf("expected %x sweep script, instead got %x", sweepScript,
			output.PkScript)
	}

	// If we cancel the sweep attempt, then we should find that all the
	// UTXOs within the sweep transaction are now unlocked.
	sweepPkg.CancelSweepAttempt()
	assertUtxosUnlocked(t, wc, testUtxos[:2])
}
