package chains

import (
	"fmt"
	"sync"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcwallet/chain"
	"github.com/btcsuite/btcwallet/waddrmgr"
	"github.com/lightningnetwork/lnd/lnwallet"
)

func NewMockChainIO(currentHeight uint32) *MockChainIO {
	return &MockChainIO{
		bestHeight: int32(currentHeight),
		blocks:     make(map[chainhash.Hash]*wire.MsgBlock),
		utxos:      make(map[wire.OutPoint]wire.TxOut),
		blockIndex: make(map[uint32]chainhash.Hash),
	}
}

// A compile time check to ensure MockChainIO implements the
// lnwallet.BlockChainIO interface.
var _ lnwallet.BlockChainIO = (*MockChainIO)(nil)

type MockChainIO struct {
	blocks     map[chainhash.Hash]*wire.MsgBlock
	blockIndex map[uint32]chainhash.Hash

	utxos map[wire.OutPoint]wire.TxOut

	bestHeight int32
	bestHash   *chainhash.Hash

	sync.RWMutex
}

func (m *MockChainIO) WaitForShutdown() {
	panic("implement me")
}

func (m *MockChainIO) FilterBlocks(*chain.FilterBlocksRequest) (*chain.FilterBlocksResponse, error) {
	panic("implement me")
}

func (m *MockChainIO) BlockStamp() (*waddrmgr.BlockStamp, error) {
	return &waddrmgr.BlockStamp{Height: m.bestHeight, Timestamp: time.Now()}, nil
}

func (m *MockChainIO) SendRawTransaction(*wire.MsgTx, bool) (*chainhash.Hash, error) {
	panic("implement me")
}

func (m *MockChainIO) Rescan(*chainhash.Hash, []btcutil.Address, map[wire.OutPoint]btcutil.Address) error {
	panic("implement me")
}

func (m *MockChainIO) NotifyReceived([]btcutil.Address) error {
	panic("implement me")
}

func (m *MockChainIO) NotifyBlocks() error {
	panic("implement me")
}

func (m *MockChainIO) Notifications() <-chan interface{} {
	panic("implement me")
}

func (m *MockChainIO) BackEnd() string {
	panic("implement me")
}

func (m *MockChainIO) GetBestBlock() (*chainhash.Hash, int32, error) {
	//return netparams.MainNetParams.GenesisHash, m.bestHeight, nil
	m.RLock()
	defer m.RUnlock()

	blockHash := m.blockIndex[uint32(m.bestHeight)]

	return &blockHash, m.bestHeight, nil

}

func (m *MockChainIO) GetUtxo(op *wire.OutPoint, _ []byte,
	heightHint uint32) (*wire.TxOut, error) {
	m.RLock()
	defer m.RUnlock()

	utxo, ok := m.utxos[*op]
	if !ok {
		return nil, fmt.Errorf("utxo not found")
	}

	return &utxo, nil
}

func (m *MockChainIO) GetBlockHash(blockHeight int64) (*chainhash.Hash, error) {
	m.RLock()
	defer m.RUnlock()

	hash, ok := m.blockIndex[uint32(blockHeight)]
	if !ok {
		return nil, fmt.Errorf("can't find block hash, for "+
			"height %v", blockHeight)
	}

	return &hash, nil
}

func (m *MockChainIO) GetBlock(blockHash *chainhash.Hash) (*wire.MsgBlock,
	error) {
	m.RLock()
	defer m.RUnlock()

	block, ok := m.blocks[*blockHash]
	if !ok {
		return nil, fmt.Errorf("block not found")
	}

	return block, nil
}

func (m *MockChainIO) GetBackend() chain.Interface {
	return m
}

func (m *MockChainIO) GetBlockHeader(
	blockHash *chainhash.Hash) (*wire.BlockHeader, error) {

	return &wire.BlockHeader{Timestamp: time.Now()}, nil
}

func (m *MockChainIO) ReturnPublishTransactionError(err error) error {
	return nil
}

func (m *MockChainIO) Start() error {
	return nil
}

func (m *MockChainIO) Stop() {
}

func (*MockChainIO) SupportsUnconfirmedTransactions() bool {
	return true
}

func (*MockChainIO) WaitForBackendToStart() {
}

// these functions are not part of the ChainIO interface but are used for
// testing

func (m *MockChainIO) AddUtxo(op wire.OutPoint, out *wire.TxOut) {
	m.Lock()
	m.utxos[op] = *out
	m.Unlock()
}

func (m *MockChainIO) AddBlock(block *wire.MsgBlock, height uint32,
	nonce uint32) {
	m.Lock()
	block.Header.Nonce = nonce
	hash := block.Header.BlockHash()
	m.blocks[hash] = block
	m.blockIndex[height] = hash
	m.Unlock()
}

func (m *MockChainIO) SetBestBlock(height int32) {
	m.Lock()
	defer m.Unlock()

	m.bestHeight = height
}
