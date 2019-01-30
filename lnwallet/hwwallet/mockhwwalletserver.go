package hwwallet

import (
	"encoding/hex"
	"github.com/tyler-smith/go-bip32"
	"golang.org/x/net/context"
	"strconv"
	"strings"
)

type MockHwWalletServer struct {
	Xpub string
}

func (b *MockHwWalletServer) DerivePublicKey(context context.Context, req *DerivePublicKeyReq) (*DerivePublicKeyResp, error) {
	var key *bip32.Key
	var err error

	for _, part := range strings.Split(req.Path, "/") {
		if part == "m" {
			hex, _ := hex.DecodeString("000102030405060708090a0b0c0d0e0f")
			key, err = bip32.NewMasterKey(hex)
			if err != nil {
				return nil, err
			}
		} else {
			var index uint64
			if part[len(part)-1:] == "'" {
				index, _ = strconv.ParseUint(strings.TrimSuffix(part, "'"), 10, 32)
				index += uint64(bip32.FirstHardenedChild)
			} else {
				index, _ = strconv.ParseUint(part, 10, 32)
			}
			key, _ = key.NewChildKey(uint32(index))
		}
	}

	bytes, _ := key.PublicKey().Serialize()

	return &DerivePublicKeyResp{bytes}, nil
}

func (MockHwWalletServer) SendOutputs(context.Context, *SendOutputsRequest) (*SendOutputsResponse, error) {
	panic("implement me")
}

func (MockHwWalletServer) SignOutputRaw(context.Context, *SignReq) (*SignResp, error) {
	panic("implement me")
}

func (MockHwWalletServer) ComputeInputScript(_ context.Context, req *ComputeInputScriptReq) (*ComputeInputScriptResp, error) {
	resp := ComputeInputScriptResp{}
	resp.InputScript.Witness = make([][]byte, 5, 5)
	return &resp, nil
}

func (MockHwWalletServer) SignMessage(context.Context, *SignMessageReq) (*SignMessageResp, error) {
	panic("implement me")
}
