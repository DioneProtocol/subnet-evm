// (c) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package aggregator

import (
	"context"

	"github.com/DioneProtocol/odysseygo/ids"
	"github.com/DioneProtocol/odysseygo/snow/validators"
	odysseyWarp "github.com/DioneProtocol/odysseygo/vms/omegavm/warp"
	"github.com/DioneProtocol/subnet-evm/params"
)

// Aggregator fulfills requests to aggregate signatures of a subnet's validator set for Odyssey Warp Messages.
type Aggregator struct {
	subnetID ids.ID
	client   SignatureBackend
	state    validators.State
}

// NewAggregator returns a signature aggregator, which will aggregate Warp Signatures for the given [
func NewAggregator(subnetID ids.ID, state validators.State, client SignatureBackend) *Aggregator {
	return &Aggregator{
		subnetID: subnetID,
		client:   client,
		state:    state,
	}
}

func (a *Aggregator) AggregateSignatures(ctx context.Context, unsignedMessage *odysseyWarp.UnsignedMessage, quorumNum uint64) (*AggregateSignatureResult, error) {
	// Note: we use the current height as a best guess of the canonical validator set when the aggregated signature will be verified
	// by the recipient chain. If the validator set changes from [oChainHeight] to the O-Chain height that is actually specified by the
	// ProposerVM header when this message is verified, then the aggregate signature could become outdated and require re-aggregation.
	oChainHeight, err := a.state.GetCurrentHeight(ctx)
	if err != nil {
		return nil, err
	}
	job := newSignatureAggregationJob(
		a.client,
		oChainHeight,
		a.subnetID,
		quorumNum,
		quorumNum,
		params.WarpQuorumDenominator,
		a.state,
		unsignedMessage,
	)

	return job.Execute(ctx)
}
