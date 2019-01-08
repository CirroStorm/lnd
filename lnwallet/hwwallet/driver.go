package hwwallet

import (
	"fmt"
	"github.com/lightningnetwork/lnd/lnwallet"
)

const (
	walletType = "hwwallet"
)

// createNewWallet creates a new instance of HwWallet given the proper list of
// initialization parameters. This function is the factory function required to
// properly create an instance of the lnwallet.WalletDriver struct for
// HwWallet.
func createNewWallet(args ...interface{}) (lnwallet.WalletController, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("incorrect number of arguments to .New(...), "+
			"expected 1, instead passed %v", len(args))
	}

	config, ok := args[0].(*Config)
	if !ok {
		return nil, fmt.Errorf("first argument to btcdnotifier.New is " +
			"incorrect, expected a *rpcclient.ConnConfig")
	}

	return New(*config)
}

func BackEnds() []string {
	return []string{
		"bitcoind",
		"btcd",
		"neutrino",
	}
}

// init registers a driver for the HwWallet concrete implementation of the
// lnwallet.WalletController interface.
func init() {
	// Register the driver.
	driver := &lnwallet.WalletDriver{
		WalletType: walletType,
		New:        createNewWallet,
		BackEnds:   BackEnds,
	}

	if err := lnwallet.RegisterWallet(driver); err != nil {
		panic(fmt.Sprintf("failed to register wallet driver '%s': %v",
			walletType, err))
	}
}
