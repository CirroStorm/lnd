package btcwallet

import (
	"bytes"
	"errors"
	"fmt"
	"sync"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcwallet/waddrmgr"
	base "github.com/btcsuite/btcwallet/wallet"
	"github.com/btcsuite/btcwallet/walletdb"
	"github.com/lightningnetwork/lnd/keychain"
	"github.com/lightningnetwork/lnd/lnwallet"
)

const (
	defaultAccount = uint32(waddrmgr.DefaultAccountNum)
)

var (
	// WaddrmgrNamespaceKey is the namespace key that the waddrmgr state is
	// stored within the top-level waleltdb buckets of btcwallet.
	WaddrmgrNamespaceKey = []byte("waddrmgr")

	// LightningAddrSchema is the scope addr schema for all keys that we
	// derive. We'll treat them all as p2wkh addresses, as atm we must
	// specify a particular type.
	LightningAddrSchema = waddrmgr.ScopeAddrSchema{
		ExternalAddrType: waddrmgr.WitnessPubKey,
		InternalAddrType: waddrmgr.WitnessPubKey,
	}
)

// BtcWallet is an implementation of the lnwallet.WalletController interface
// backed by an active instance of btcwallet. At the time of the writing of
// this documentation, this implementation requires a full btcd node to
// operate.
type BtcWallet struct {
	// wallet is an active instance of btcwallet.
	wallet *base.Wallet

	chain lnwallet.BlockChainIO

	db walletdb.DB

	cfg *Config

	netParams *chaincfg.Params

	chainKeyScope waddrmgr.KeyScope

	// lightningScope is a pointer to the scope that we'll be using as a
	// sub key manager to derive all the keys that we require.
	lightningScope *waddrmgr.ScopedKeyManager

	// utxoCache is a cache used to speed up repeated calls to
	// FetchInputInfo.
	utxoCache map[wire.OutPoint]*wire.TxOut
	cacheMtx  sync.RWMutex
}

// A compile time check to ensure that BtcWallet implements the
// WalletController interface.
var _ lnwallet.WalletController = (*BtcWallet)(nil)

// New returns a new fully initialized instance of BtcWallet given a valid
// configuration struct.
func New(cfg Config) (*BtcWallet, error) {
	// Ensure the wallet exists or create it when the create flag is set.
	netDir := NetworkDir(cfg.DataDir, cfg.NetParams)

	// Create the key scope for the coin type being managed by this wallet.
	chainKeyScope := waddrmgr.KeyScope{
		Purpose: keychain.BIP0043Purpose,
		Coin:    cfg.CoinType,
	}

	// Maybe the wallet has already been opened and unlocked by the
	// WalletUnlocker. So if we get a non-nil value from the config,
	// we assume everything is in order.
	var wallet = cfg.Wallet
	if wallet == nil {
		// No ready wallet was passed, so try to open an existing one.
		var pubPass []byte
		if cfg.PublicPass == nil {
			pubPass = defaultPubPassphrase
		} else {
			pubPass = cfg.PublicPass
		}
		loader := base.NewLoader(cfg.NetParams, netDir,
			cfg.RecoveryWindow)
		walletExists, err := loader.WalletExists()
		if err != nil {
			return nil, err
		}

		if !walletExists {
			// Wallet has never been created, perform initial
			// set up.
			wallet, err = loader.CreateNewWallet(
				pubPass, cfg.PrivatePass, cfg.HdSeed,
				cfg.Birthday,
			)
			if err != nil {
				return nil, err
			}
		} else {
			// Wallet has been created and been initialized at
			// this point, open it along with all the required DB
			// namespaces, and the DB itself.
			wallet, err = loader.OpenExistingWallet(pubPass, false)
			if err != nil {
				return nil, err
			}
		}
	}

	return &BtcWallet{
		cfg:           &cfg,
		wallet:        wallet,
		db:            wallet.Database(),
		chain:         cfg.ChainSource,
		netParams:     cfg.NetParams,
		chainKeyScope: chainKeyScope,
		utxoCache:     make(map[wire.OutPoint]*wire.TxOut),
	}, nil
}

// Start initializes the underlying rpc connection, the wallet itself, and
// begins syncing to the current available blockchain state.
//
// This is a part of the WalletController interface.
func (b *BtcWallet) Start() error {
	// We'll start by unlocking the wallet and ensuring that the KeyScope:
	// (1017, 1) exists within the internal waddrmgr. We'll need this in
	// order to properly generate the keys required for signing various
	// contracts.
	if err := b.wallet.Unlock(b.cfg.PrivatePass, nil); err != nil {
		return err
	}
	_, err := b.wallet.Manager.FetchScopedKeyManager(b.chainKeyScope)
	if err != nil {
		// If the scope hasn't yet been created (it wouldn't been
		// loaded by default if it was), then we'll manually create the
		// scope for the first time ourselves.
		err := walletdb.Update(b.db, func(tx walletdb.ReadWriteTx) error {
			addrmgrNs := tx.ReadWriteBucket(WaddrmgrNamespaceKey)

			_, err := b.wallet.Manager.NewScopedKeyManager(
				addrmgrNs, b.chainKeyScope, LightningAddrSchema,
			)
			return err
		})
		if err != nil {
			return err
		}
	}

	// Establish an RPC connection in addition to starting the goroutines
	// in the underlying wallet.
	if err := b.chain.Start(); err != nil {
		return err
	}

	// Start the underlying btcwallet core.
	b.wallet.Start()

	// Pass the rpc client into the wallet so it can sync up to the
	// current main chain.
	b.wallet.SynchronizeRPC(b.chain.GetBackend())

	return nil
}

// Stop signals the wallet for shutdown. Shutdown may entail closing
// any active sockets, database handles, stopping goroutines, etc.
//
// This is a part of the WalletController interface.
func (b *BtcWallet) Stop() error {
	b.wallet.Stop()

	b.wallet.WaitForShutdown()

	b.chain.Stop()

	return nil
}

// NewAddress returns the next external or internal address for the wallet
// dictated by the value of the `change` parameter. If change is true, then an
// internal address will be returned, otherwise an external address should be
// returned.
//
// This is a part of the WalletController interface.
func (b *BtcWallet) NewAddress(keyScope waddrmgr.KeyScope,
	change bool) (btcutil.Address, error) {
	if change {
		return b.wallet.NewChangeAddress(defaultAccount, keyScope)
	}

	return b.wallet.NewAddress(defaultAccount, keyScope)
}

// IsOurAddress checks if the passed address belongs to this wallet
//
// This is a part of the WalletController interface.
func (b *BtcWallet) IsOurAddress(a btcutil.Address) bool {
	result, err := b.wallet.HaveAddress(a)
	return result && (err == nil)
}

// SendOutputs funds, signs, and broadcasts a Bitcoin transaction paying out to
// the specified outputs. In the case the wallet has insufficient funds, or the
// outputs are non-standard, a non-nil error will be returned.
//
// This is a part of the WalletController interface.
func (b *BtcWallet) SendOutputs(outputs []*wire.TxOut,
	feeRate lnwallet.SatPerKWeight) (*wire.MsgTx, error) {

	// Convert our fee rate from sat/kw to sat/kb since it's required by
	// SendOutputs.
	feeSatPerKB := btcutil.Amount(feeRate.FeePerKVByte())

	return b.wallet.SendOutputs(outputs, defaultAccount, 1, feeSatPerKB)
}

// LockOutpoint marks an outpoint as locked meaning it will no longer be deemed
// as eligible for coin selection. Locking outputs are utilized in order to
// avoid race conditions when selecting inputs for usage when funding a
// channel.
//
// This is a part of the WalletController interface.
func (b *BtcWallet) LockOutpoint(o wire.OutPoint) {
	b.wallet.LockOutpoint(o)
}

// UnlockOutpoint unlocks a previously locked output, marking it eligible for
// coin selection.
//
// This is a part of the WalletController interface.
func (b *BtcWallet) UnlockOutpoint(o wire.OutPoint) {
	b.wallet.UnlockOutpoint(o)
}

// ListUnspent returns a slice of all the unspent outputs the wallet
// controls which pay to witness programs either directly or indirectly.
//
// This is a part of the WalletController interface.
func (b *BtcWallet) ListUnspent(minConfs, maxConfs int32) ([]*btcjson.
	ListUnspentResult, error) {
	return b.wallet.ListUnspent(minConfs, maxConfs, nil)
}

// PublishTransaction performs cursory validation (dust checks, etc), then
// finally broadcasts the passed transaction to the Bitcoin network. If
// publishing the transaction fails, an error describing the reason is
// returned (currently ErrDoubleSpend). If the transaction is already
// published to the network (either in the mempool or chain) no error
// will be returned.
func (b *BtcWallet) PublishTransaction(tx *wire.MsgTx) error {
	if err := b.wallet.PublishTransaction(tx); err != nil {
		return b.chain.ReturnPublishTransactionError(err)
	}
	return nil
}

// extractBalanceDelta extracts the net balance delta from the PoV of the
// wallet given a TransactionSummary.
func extractBalanceDelta(
	txSummary base.TransactionSummary,
	tx *wire.MsgTx,
) (btcutil.Amount, error) {
	// For each input we debit the wallet's outflow for this transaction,
	// and for each output we credit the wallet's inflow for this
	// transaction.
	var balanceDelta btcutil.Amount
	for _, input := range txSummary.MyInputs {
		balanceDelta -= input.PreviousAmount
	}
	for _, output := range txSummary.MyOutputs {
		balanceDelta += btcutil.Amount(tx.TxOut[output.Index].Value)
	}

	return balanceDelta, nil
}

// minedTransactionsToDetails is a helper function which converts a summary
// information about mined transactions to a TransactionDetail.
func minedTransactionsToDetails(
	currentHeight int32,
	block base.Block,
	chainParams *chaincfg.Params,
) ([]*lnwallet.TransactionDetail, error) {

	details := make([]*lnwallet.TransactionDetail, 0, len(block.Transactions))
	for _, tx := range block.Transactions {
		wireTx := &wire.MsgTx{}
		txReader := bytes.NewReader(tx.Transaction)

		if err := wireTx.Deserialize(txReader); err != nil {
			return nil, err
		}

		var destAddresses []btcutil.Address
		for _, txOut := range wireTx.TxOut {
			_, outAddresses, _, err :=
				txscript.ExtractPkScriptAddrs(txOut.PkScript, chainParams)
			if err != nil {
				return nil, err
			}

			destAddresses = append(destAddresses, outAddresses...)
		}

		txDetail := &lnwallet.TransactionDetail{
			Hash:             *tx.Hash,
			NumConfirmations: currentHeight - block.Height + 1,
			BlockHash:        block.Hash,
			BlockHeight:      block.Height,
			Timestamp:        block.Timestamp,
			TotalFees:        int64(tx.Fee),
			DestAddresses:    destAddresses,
		}

		balanceDelta, err := extractBalanceDelta(tx, wireTx)
		if err != nil {
			return nil, err
		}
		txDetail.Value = balanceDelta

		details = append(details, txDetail)
	}

	return details, nil
}

// unminedTransactionsToDetail is a helper function which converts a summary
// for an unconfirmed transaction to a transaction detail.
func unminedTransactionsToDetail(
	summary base.TransactionSummary,
) (*lnwallet.TransactionDetail, error) {
	wireTx := &wire.MsgTx{}
	txReader := bytes.NewReader(summary.Transaction)

	if err := wireTx.Deserialize(txReader); err != nil {
		return nil, err
	}

	txDetail := &lnwallet.TransactionDetail{
		Hash:      *summary.Hash,
		TotalFees: int64(summary.Fee),
		Timestamp: summary.Timestamp,
	}

	balanceDelta, err := extractBalanceDelta(summary, wireTx)
	if err != nil {
		return nil, err
	}
	txDetail.Value = balanceDelta

	return txDetail, nil
}

// ListTransactionDetails returns a list of all transactions which are
// relevant to the wallet.
//
// This is a part of the WalletController interface.
func (b *BtcWallet) ListTransactionDetails() ([]*lnwallet.TransactionDetail,
	error) {
	// Grab the best block the wallet knows of, we'll use this to calculate
	// # of confirmations shortly below.
	bestBlock := b.wallet.Manager.SyncedTo()
	currentHeight := bestBlock.Height

	// We'll attempt to find all unconfirmed transactions (height of -1),
	// as well as all transactions that are known to have confirmed at this
	// height.
	start := base.NewBlockIdentifierFromHeight(0)
	stop := base.NewBlockIdentifierFromHeight(-1)
	txns, err := b.wallet.GetTransactions(start, stop, nil)
	if err != nil {
		return nil, err
	}

	txDetails := make([]*lnwallet.TransactionDetail, 0,
		len(txns.MinedTransactions)+len(txns.UnminedTransactions))

	// For both confirmed and unconfirmed transactions, create a
	// TransactionDetail which re-packages the data returned by the base
	// wallet.
	for _, blockPackage := range txns.MinedTransactions {
		details, err := minedTransactionsToDetails(currentHeight, blockPackage, b.netParams)
		if err != nil {
			return nil, err
		}

		txDetails = append(txDetails, details...)
	}
	for _, tx := range txns.UnminedTransactions {
		detail, err := unminedTransactionsToDetail(tx)
		if err != nil {
			return nil, err
		}

		txDetails = append(txDetails, detail)
	}

	return txDetails, nil
}

// txSubscriptionClient encapsulates the transaction notification client from
// the base wallet. Notifications received from the client will be proxied over
// two distinct channels.
type txSubscriptionClient struct {
	txClient base.TransactionNotificationsClient

	confirmed   chan *lnwallet.TransactionDetail
	unconfirmed chan *lnwallet.TransactionDetail

	w *base.Wallet

	wg   sync.WaitGroup
	quit chan struct{}
}

// ConfirmedTransactions returns a channel which will be sent on as new
// relevant transactions are confirmed.
//
// This is part of the TransactionSubscription interface.
func (t *txSubscriptionClient) ConfirmedTransactions() chan *lnwallet.
	TransactionDetail {
	return t.confirmed
}

// UnconfirmedTransactions returns a channel which will be sent on as
// new relevant transactions are seen within the network.
//
// This is part of the TransactionSubscription interface.
func (t *txSubscriptionClient) UnconfirmedTransactions() chan *lnwallet.
	TransactionDetail {
	return t.unconfirmed
}

// Cancel finalizes the subscription, cleaning up any resources allocated.
//
// This is part of the TransactionSubscription interface.
func (t *txSubscriptionClient) Cancel() {
	close(t.quit)
	t.wg.Wait()

	t.txClient.Done()
}

// notificationProxier proxies the notifications received by the underlying
// wallet's notification client to a higher-level TransactionSubscription
// client.
func (t *txSubscriptionClient) notificationProxier() {
out:
	for {
		select {
		case txNtfn := <-t.txClient.C:
			// TODO(roasbeef): handle detached blocks
			currentHeight := t.w.Manager.SyncedTo().Height

			// Launch a goroutine to re-package and send
			// notifications for any newly confirmed transactions.
			go func() {
				for _, block := range txNtfn.AttachedBlocks {
					details, err := minedTransactionsToDetails(
						currentHeight, block, t.w.ChainParams())
					if err != nil {
						continue
					}

					for _, d := range details {
						select {
						case t.confirmed <- d:
						case <-t.quit:
							return
						}
					}
				}

			}()

			// Launch a goroutine to re-package and send
			// notifications for any newly unconfirmed transactions.
			go func() {
				for _, tx := range txNtfn.UnminedTransactions {
					detail, err := unminedTransactionsToDetail(tx)
					if err != nil {
						continue
					}

					select {
					case t.unconfirmed <- detail:
					case <-t.quit:
						return
					}
				}
			}()
		case <-t.quit:
			break out
		}
	}

	t.wg.Done()
}

// SubscribeTransactions returns a TransactionSubscription client which
// is capable of receiving async notifications as new transactions
// related to the wallet are seen within the network, or found in
// blocks.
//
// This is a part of the WalletController interface.
func (b *BtcWallet) SubscribeTransactions() (lnwallet.
	TransactionSubscription, error) {
	walletClient := b.wallet.NtfnServer.TransactionNotifications()

	txClient := &txSubscriptionClient{
		txClient:    walletClient,
		confirmed:   make(chan *lnwallet.TransactionDetail),
		unconfirmed: make(chan *lnwallet.TransactionDetail),
		w:           b.wallet,
		quit:        make(chan struct{}),
	}
	txClient.wg.Add(1)
	go txClient.notificationProxier()

	return txClient, nil
}

// keyScope attempts to return the key scope that we'll use to derive all of
// our keys. If the scope has already been fetched from the database, then a
// cached version will be returned. Otherwise, we'll fetch it from the database
// and cache it for subsequent accesses.
func (b *BtcWallet) keyScope() (*waddrmgr.ScopedKeyManager, error) {
	// If the scope has already been populated, then we'll return it
	// directly.
	if b.lightningScope != nil {
		return b.lightningScope, nil
	}

	// Otherwise, we'll first do a check to ensure that the root manager
	// isn't locked, as otherwise we won't be able to *use* the scope.
	if b.wallet.Manager.IsLocked() {
		return nil, fmt.Errorf("cannot create BtcWalletKeyRing with " +
			"locked waddrmgr.Manager")
	}

	// If the manager is indeed unlocked, then we'll fetch the scope, cache
	// it, and return to the caller.
	lnScope, err := b.wallet.Manager.FetchScopedKeyManager(b.chainKeyScope)
	if err != nil {
		return nil, err
	}

	b.lightningScope = lnScope

	return lnScope, nil
}

// createAccountIfNotExists will create the corresponding account for a key
// family if it doesn't already exist in the database.
func (b *BtcWallet) createAccountIfNotExists(
	addrmgrNs walletdb.ReadWriteBucket, keyFam keychain.KeyFamily,
	scope *waddrmgr.ScopedKeyManager) error {

	// If this is the multi-sig key family, then we can return early as
	// this is the default account that's created.
	if keyFam == keychain.KeyFamilyMultiSig {
		return nil
	}

	// Otherwise, we'll check if the account already exists, if so, we can
	// once again bail early.
	_, err := scope.AccountName(addrmgrNs, uint32(keyFam))
	if err == nil {
		return nil
	}

	// If we reach this point, then the account hasn't yet been created, so
	// we'll need to create it before we can proceed.
	return scope.NewRawAccount(addrmgrNs, uint32(keyFam))
}

// DeriveNextKey attempts to derive the *next* key within the key family
// (account in BIP43) specified. This method should return the next external
// child within this branch.
//
// NOTE: This is part of the keychain.KeyRing interface.
func (b *BtcWallet) DeriveNextKey(keyFam keychain.KeyFamily) (keychain.
	KeyDescriptor, error) {
	var (
		pubKey *btcec.PublicKey
		keyLoc keychain.KeyLocator
	)

	db := b.wallet.Database()
	err := walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		addrmgrNs := tx.ReadWriteBucket(WaddrmgrNamespaceKey)

		scope, err := b.keyScope()
		if err != nil {
			return err
		}

		// If the account doesn't exist, then we may need to create it
		// for the first time in order to derive the keys that we
		// require.
		err = b.createAccountIfNotExists(addrmgrNs, keyFam, scope)
		if err != nil {
			return err
		}

		addrs, err := scope.NextExternalAddresses(
			addrmgrNs, uint32(keyFam), 1,
		)
		if err != nil {
			return err
		}

		// Extract the first address, ensuring that it is of the proper
		// interface type, otherwise we can't manipulate it below.
		addr, ok := addrs[0].(waddrmgr.ManagedPubKeyAddress)
		if !ok {
			return fmt.Errorf("address is not a managed pubkey " +
				"addr")
		}

		pubKey = addr.PubKey()

		_, pathInfo, _ := addr.DerivationInfo()
		keyLoc = keychain.KeyLocator{
			Family: keyFam,
			Index:  pathInfo.Index,
		}

		return nil
	})
	if err != nil {
		return keychain.KeyDescriptor{}, err
	}

	return keychain.KeyDescriptor{
		PubKey:     pubKey,
		KeyLocator: keyLoc,
	}, nil
}

// DeriveKey attempts to derive an arbitrary key specified by the passed
// KeyLocator. This may be used in several recovery scenarios, or when manually
// rotating something like our current default node key.
//
// NOTE: This is part of the keychain.KeyRing interface.
func (b *BtcWallet) DeriveKey(keyLoc keychain.KeyLocator) (keychain.
	KeyDescriptor, error) {
	var keyDesc keychain.KeyDescriptor

	db := b.wallet.Database()
	err := walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		addrmgrNs := tx.ReadWriteBucket(WaddrmgrNamespaceKey)

		scope, err := b.keyScope()
		if err != nil {
			return err
		}

		// If the account doesn't exist, then we may need to create it
		// for the first time in order to derive the keys that we
		// require.
		err = b.createAccountIfNotExists(addrmgrNs, keyLoc.Family, scope)
		if err != nil {
			return err
		}

		path := waddrmgr.DerivationPath{
			Account: uint32(keyLoc.Family),
			Branch:  0,
			Index:   uint32(keyLoc.Index),
		}
		addr, err := scope.DeriveFromKeyPath(addrmgrNs, path)
		if err != nil {
			return err
		}

		keyDesc.KeyLocator = keyLoc
		keyDesc.PubKey = addr.(waddrmgr.ManagedPubKeyAddress).PubKey()

		return nil
	})
	if err != nil {
		return keyDesc, err
	}

	return keyDesc, nil
}

// DerivePrivKey will attempt to derive the private key that corresponds to the
// passed key descriptor.
//
// NOTE: This is part of the keychain.SecretKeyRing interface.
func (b *BtcWallet) DerivePrivKey(keyLoc keychain.KeyLocator) (*btcec.
	PrivateKey, error) {
	// BtcWallet technically could allow this,
	// but since this is a restriction caused by HwWallet we will enforce
	// the restriction as well
	if keyLoc.Family == keychain.KeyFamilyMultiSig {
		return nil, errors.New("Getting private keys for KeyFamilyMultiSig" +
			" is not allowed")
	}

	var key *btcec.PrivateKey

	db := b.wallet.Database()
	err := walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		addrmgrNs := tx.ReadWriteBucket(WaddrmgrNamespaceKey)

		scope, err := b.keyScope()
		if err != nil {
			return err
		}

		// If the account doesn't exist, then we may need to create it
		// for the first time in order to derive the keys that we
		// require.
		err = b.createAccountIfNotExists(
			addrmgrNs, keyLoc.Family, scope,
		)
		if err != nil {
			return err
		}

		// Now that we know the account exists, we can safely
		// derive the full private key from the given path.
		path := waddrmgr.DerivationPath{
			Account: uint32(keyLoc.Family),
			Branch:  0,
			Index:   uint32(keyLoc.Index),
		}
		addr, err := scope.DeriveFromKeyPath(addrmgrNs, path)
		if err != nil {
			return err
		}

		key, err = addr.(waddrmgr.ManagedPubKeyAddress).PrivKey()
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return key, nil
}
