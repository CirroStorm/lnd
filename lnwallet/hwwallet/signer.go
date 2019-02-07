package hwwallet

import (
	"bytes"
	"context"
	"fmt"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/wire"
	"github.com/lightningnetwork/lnd/input"
	"github.com/lightningnetwork/lnd/lnwallet"
	"runtime"
)

// FetchInputInfo queries for the WalletController's knowledge of the passed
// outpoint. If the base wallet determines this output is under its control,
// then the original txout should be returned. Otherwise, a non-nil error value
// of ErrNotMine should be returned instead.
//
// This is a part of the WalletController interface.
func (b *HwWallet) FetchInputInfo(prevOut *wire.OutPoint) (*wire.TxOut, error) {
	// TODO call rpc
	pc, _, _, _ := runtime.Caller(1)
	panic(fmt.Sprintf("%s", runtime.FuncForPC(pc).Name()))
	return nil, nil
}

// SignOutputRaw generates a signature for the passed transaction according to
// the data within the passed SignDescriptor.
//
// This is a part of the WalletController interface.
func (b *HwWallet) SignOutputRaw(tx *wire.MsgTx,
	signDesc *input.SignDescriptor) ([]byte, error) {
	// TODO call rpc
	pc, _, _, _ := runtime.Caller(1)
	panic(fmt.Sprintf("%s", runtime.FuncForPC(pc).Name()))
	// Chop off the sighash flag at the end of the signature.
	return nil, nil
}

// ComputeInputScript generates a complete InputScript for the passed
// transaction with the signature as defined within the passed SignDescriptor.
// This method is capable of generating the proper input script for both
// regular p2wkh output and p2wkh outputs nested within a regular p2sh output.
//
// This is a part of the WalletController interface.
func (b *HwWallet) ComputeInputScript(tx *wire.MsgTx,
	signDesc *input.SignDescriptor) (*input.Script, error) {

	sd := SignDescriptor{
		KeyDesc: &KeyDescriptor{
			KeyLoc: &KeyLocator{
				KeyFamily: uint32(signDesc.KeyDesc.Family),
				KeyIndex:  signDesc.KeyDesc.Index,
			},
			RawKeyBytes: signDesc.KeyDesc.PubKey.SerializeCompressed(),
		},
		SingleTweak:   signDesc.SingleTweak,
		DoubleTweak:   signDesc.DoubleTweak.Serialize(),
		WitnessScript: signDesc.WitnessScript,
		Output: &TxOut{
			Value:    signDesc.Output.Value,
			PkScript: signDesc.Output.PkScript,
		},
		Sighash:    uint32(signDesc.HashType),
		InputIndex: int32(signDesc.InputIndex),
	}

	var txBuf bytes.Buffer
	err := tx.Serialize(&txBuf)
	if err != nil {
		return nil, err
	}

	req := ComputeInputScriptReq{txBuf.Bytes(), &sd}
	resp, err := b.client.ComputeInputScript(context.Background(), &req)
	if err != nil {
		return nil, err
	}

	// fortunetly the response struct is simple enough that we can just cast it
	is := *resp.GetInputScript()
	result := input.Script{Witness: is.Witness, SigScript: is.ScriptSig}

	return &result, nil
}

// A compile time check to ensure that BtcWallet implements the Signer
// interface.
var _ input.Signer = (*HwWallet)(nil)

// SignMessage attempts to sign a target message with the private key that
// corresponds to the passed public key. If the target private key is unable to
// be found, then an error will be returned. The actual digest signed is the
// double SHA-256 of the passed message.
//
// NOTE: This is a part of the MessageSigner interface.
func (b *HwWallet) SignMessage(pubKey *btcec.PublicKey,
	msg []byte) (*btcec.Signature, error) {

	// TODO call rpc
	pc, _, _, _ := runtime.Caller(1)
	panic(fmt.Sprintf("%s", runtime.FuncForPC(pc).Name()))
	return nil, nil
}

// A compile time check to ensure that BtcWallet implements the MessageSigner
// interface.
var _ lnwallet.MessageSigner = (*HwWallet)(nil)
