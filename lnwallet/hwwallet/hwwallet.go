package hwwallet

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/juju/errors"
	"github.com/tyler-smith/go-bip32"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/btcsuite/btcwallet/waddrmgr"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcwallet/walletdb"
	"github.com/lightningnetwork/lnd/keychain"
	"github.com/lightningnetwork/lnd/lnwallet"
	"google.golang.org/grpc"
)

var (
	nextInternalAddressIndexKey = []byte("nextInternalAddressIndex")
	nextExternalAddressIndexKey = []byte("nextExternalAddressIndex")
	addressBucketKey            = []byte("addressdb")
	addressesBucketKey          = []byte("addresses")
)

// HwWallet is an implementation of the lnwallet.WalletController interface
// backed by an active instance of btcwallet. At the time of the writing of
// this documentation, this implementation requires a full btcd node to
// operate.
type HwWallet struct {
	cfg *Config

	chain lnwallet.BlockChainIO

	// utxoCache is a cache used to speed up repeated calls to
	// FetchInputInfo.
	utxoCache map[wire.OutPoint]*wire.TxOut
	cacheMtx  sync.RWMutex

	client     HwWalletClient
	clientConn *grpc.ClientConn

	db     walletdb.DB
	dbPath string

	keyScope waddrmgr.KeyScope
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
	b.dbPath = filepath.Join(b.cfg.DataDir, "hwwallet.db")
	db, err := walletdb.Create("bdb", b.dbPath)
	if err != nil {
		return err
	}
	b.db = db

	b.clientConn, err = grpc.Dial(b.cfg.RemoteAddress, grpc.WithInsecure())
	if err != nil {
		return err
	}
	b.client = NewHwWalletClient(b.clientConn)
	return nil
}

// Stop signals the wallet for shutdown. Shutdown may entail closing
// any active sockets, database handles, stopping goroutines, etc.
//
// This is a part of the WalletController interface.
func (b *HwWallet) Stop() error {
	if b.db != nil {
		b.db.Close()
		os.Remove(b.dbPath)
	}

	return b.clientConn.Close()
}

// NewAddress returns the next external or internal address for the wallet
// dictated by the value of the `change` parameter. If change is true, then an
// internal address will be returned, otherwise an external address should be
// returned.
//
// This is a part of the WalletController interface.
func (b *HwWallet) NewAddress(keyScope waddrmgr.KeyScope, change bool) (btcutil.Address, error) {
	// TODO handle keyscope

	keyDescriptor, err := b.deriveNextKey(keyScope, keychain.KeyFamilyMultiSig, change)

	result, err := btcutil.NewAddressPubKey(keyDescriptor.PubKey.SerializeCompressed(), b.cfg.NetParams)
	if err != nil {
		return nil, err
	}

	return result, nil
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

// ListUnspent returns a slice of all the unspent outputs the wallet
// controls which pay to witness programs either directly or indirectly.
//
// This is a part of the WalletController interface.
func (b *HwWallet) ListUnspent(minConfs, maxConfs int32) ([]*btcjson.ListUnspentResult, error) {
	// TODO call rpc
	pc, _, _, _ := runtime.Caller(1)
	panic(fmt.Sprintf("%s", runtime.FuncForPC(pc).Name()))
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
	// TODO call RP
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

func (b *HwWallet) deriveNextKey(keyScope waddrmgr.KeyScope, keyFam keychain.KeyFamily, change bool) (keychain.KeyDescriptor, error) {
	var result keychain.KeyDescriptor

	accountKey, err := b.getAccountKey(keyFam)
	if err != nil {
		return result, err
	}

	addressIndex, err := b.getNextAddressIndex(keyFam, change)
	if err != nil {
		return result, err
	}

	var chainID uint32

	if change {
		chainID = 1
	} else {
		chainID = 0
	}

	child, err := accountKey.NewChildKey(chainID)
	if err != nil {
		return result, err
	}

	child, err = child.NewChildKey(addressIndex)
	if err != nil {
		return result, err
	}

	result.PubKey, err = btcec.ParsePubKey(child.Key, btcec.S256())
	if err != nil {
		return result, err
	}
	result.Index = addressIndex
	result.Family = keyFam
	result.KeyLocator = keychain.KeyLocator{
		Family: keyFam,
		Index:  addressIndex,
	}

	return result, nil
}

// DeriveNextKey attempts to derive the *next* key within the key family
// (account in BIP43) specified. This method should return the next external
// child within this branch.
//
// NOTE: This is part of the keychain.KeyRing interface.
func (b *HwWallet) DeriveNextKey(keyFam keychain.KeyFamily) (keychain.KeyDescriptor, error) {
	return b.deriveNextKey(b.keyScope, keyFam, false)
}

// DeriveKey attempts to derive an arbitrary key specified by the passed
// KeyLocator. This may be used in several recovery scenarios, or when manually
// rotating something like our current default node key.
//
// NOTE: This is part of the keychain.KeyRing interface.
func (b *HwWallet) DeriveKey(keyLoc keychain.KeyLocator) (keychain.KeyDescriptor, error) {
	var result keychain.KeyDescriptor

	// m / purpose' / coin_type' / account' / change / address_index
	req := DerivePublicKeyReq{fmt.Sprintf("m/%d'/%d/%d'/0/%d", b.keyScope.Purpose, b.keyScope.Coin, keyLoc.Family, keyLoc.Index)}
	resp, err := b.client.DerivePublicKey(context.Background(), &req)
	if err != nil {
		return result, err
	}

	key, err := bip32.Deserialize(resp.PublicKeyBytes)
	if err != nil {
		return result, err
	}

	result.PubKey, err = btcec.ParsePubKey(key.Key, btcec.S256())
	if err != nil {
		return result, err
	}

	result.KeyLocator.Index = keyLoc.Index
	result.KeyLocator.Family = keyLoc.Family

	return result, nil
}

func (b *HwWallet) DerivePrivKey(keyLoc keychain.KeyLocator) (*btcec.PrivateKey, error) {
	// private keys for KeyFamilyMultiSig are stored on a separate device and so are not
	// available directly
	// but we can return private keys for other families
	if keyLoc.Family == keychain.KeyFamilyMultiSig {
		return nil, errors.New("Getting private keys for KeyFamilyMultiSig is not allowed")
	}

	var result *btcec.PrivateKey

	err := walletdb.Update(b.db, func(tx walletdb.ReadWriteTx) error {
		familyBucket, err := b.getFamilyBucket(keyLoc.Family, tx)
		if err != nil {
			return err
		}

		nodeKeyKey := []byte{byte(keyLoc.Family)}
		nodeKeyBytes := familyBucket.Get(nodeKeyKey)
		if nodeKeyBytes == nil {
			privateKey, err := btcec.NewPrivateKey(btcec.S256())
			if err != nil {
				return err
			}
			nodeKeyBytes = privateKey.Serialize()
			familyBucket.Put(nodeKeyKey, nodeKeyBytes)
		}

		result, _ = btcec.PrivKeyFromBytes(btcec.S256(), nodeKeyBytes)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

//{
//  "<scope>": {
//    "<family/account>": {
//      "publicKey": [],
//      "nextInternalAddressIndex": 0,
//      "nextExternalAddressIndex": 0,
//      "addresses": [
//        "1BvBMSEYstWetqTFn5Au4m4GFg7xJaNVN2",
//        "bc1qar0srrr7xfkvy5l643lydnw9re59gtzzwf5mdq"
//      ]
//    }
//  }
//}

func (b *HwWallet) getFamilyBucket(family keychain.KeyFamily, tx walletdb.ReadWriteTx) (walletdb.ReadWriteBucket, error) {
	var err error

	scopeBucketKey := []byte(b.keyScope.String())
	scopeBucket := tx.ReadWriteBucket(scopeBucketKey)
	if scopeBucket == nil {
		scopeBucket, err = tx.CreateTopLevelBucket(scopeBucketKey)
		if err != nil {
			return nil, err
		}
	}

	familyBucketKey := []byte(fmt.Sprintf("%d", family))
	familyBucket := scopeBucket.NestedReadWriteBucket(familyBucketKey)
	if familyBucket == nil {
		familyBucket, err = scopeBucket.CreateBucket(familyBucketKey)
		if err != nil {
			return nil, err
		}
	}

	return familyBucket, nil
}

func (b *HwWallet) getAccountKey(family keychain.KeyFamily) (*bip32.Key, error) {
	var result *bip32.Key

	err := walletdb.Update(b.db, func(tx walletdb.ReadWriteTx) error {
		familyBucket, err := b.getFamilyBucket(family, tx)
		if err != nil {
			return err
		}

		publicKeyKey := []byte("publicKey")
		publicKeyBytes := familyBucket.Get(publicKeyKey)
		if publicKeyBytes == nil {
			// TODO call rpc to get public key for account
			req := DerivePublicKeyReq{fmt.Sprintf("m/%d'/%d/%d'", b.keyScope.Purpose, b.keyScope.Coin, family)}
			resp, err := b.client.DerivePublicKey(context.Background(), &req)
			if err != nil {
				return err
			}
			publicKeyBytes = resp.PublicKeyBytes
			familyBucket.Put(publicKeyKey, publicKeyBytes)
		}

		result, err = bip32.Deserialize(publicKeyBytes)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (b *HwWallet) getNextAddressIndex(family keychain.KeyFamily, change bool) (uint32, error) {
	var result uint32

	err := walletdb.Update(b.db, func(tx walletdb.ReadWriteTx) error {
		familyBucket, err := b.getFamilyBucket(family, tx)
		if err != nil {
			return err
		}

		addressIndexKey := []byte("nextExternalAddressIndex")
		if change {
			addressIndexKey = []byte("nextInternalAddressIndex")
		}

		addressIndexBytes := familyBucket.Get(addressIndexKey)
		if addressIndexBytes != nil {
			result = binary.BigEndian.Uint32(addressIndexBytes)
		}

		// now increment the index and store it in the db
		addressIndexBytes = make([]byte, 4)
		binary.BigEndian.PutUint32(addressIndexBytes, result+1)
		err = familyBucket.Put(addressIndexKey, addressIndexBytes)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	return result, nil
}
