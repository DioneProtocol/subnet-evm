// (c) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package warp

import (
	"context"
	"fmt"

	"github.com/DioneProtocol/odysseygo/ids"
	"github.com/DioneProtocol/odysseygo/utils/crypto/bls"
	odysseyWarp "github.com/DioneProtocol/odysseygo/vms/omegavm/warp"
)

type warpAPIFetcher struct {
	clients map[ids.NodeID]Client
}

func NewWarpAPIFetcher(clients map[ids.NodeID]Client) *warpAPIFetcher {
	return &warpAPIFetcher{
		clients: clients,
	}
}

func (f *warpAPIFetcher) FetchWarpSignature(ctx context.Context, nodeID ids.NodeID, unsignedWarpMessage *odysseyWarp.UnsignedMessage) (*bls.Signature, error) {
	client, ok := f.clients[nodeID]
	if !ok {
		return nil, fmt.Errorf("no warp client for nodeID: %s", nodeID)
	}

	signatureBytes, err := client.GetSignature(ctx, unsignedWarpMessage.ID())
	if err != nil {
		return nil, err
	}

	signature, err := bls.SignatureFromBytes(signatureBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse signature from client %s: %w", nodeID, err)
	}
	return signature, nil
}
