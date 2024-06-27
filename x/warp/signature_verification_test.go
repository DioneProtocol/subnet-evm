// (c) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package warp

import (
	"context"
	"math"
	"testing"

	"github.com/DioneProtocol/odysseygo/ids"
	"github.com/DioneProtocol/odysseygo/snow/validators"
	"github.com/DioneProtocol/odysseygo/utils/crypto/bls"
	"github.com/DioneProtocol/odysseygo/utils/set"
	odysseyWarp "github.com/DioneProtocol/odysseygo/vms/omegavm/warp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// This test copies the test coverage from https://github.com/DioneProtocol/odysseygo/blob/v0.0.1/vms/omegavm/warp/signature_test.go#L138.
// These tests are only expected to fail if there is a breaking change in OdysseyGo that unexpectedly changes behavior.
func TestSignatureVerification(t *testing.T) {
	tests = []signatureTest{
		{
			name: "can't get subnetID",
			stateF: func(ctrl *gomock.Controller) validators.State {
				state := validators.NewMockState(ctrl)
				state.EXPECT().GetSubnetID(gomock.Any(), sourceChainID).Return(sourceSubnetID, errTest)
				return state
			},
			quorumNum: 1,
			quorumDen: 2,
			msgF: func(require *require.Assertions) *odysseyWarp.Message {
				unsignedMsg, err := odysseyWarp.NewUnsignedMessage(
					networkID,
					sourceChainID,
					addressedPayloadBytes,
				)
				require.NoError(err)

				msg, err := odysseyWarp.NewMessage(
					unsignedMsg,
					&odysseyWarp.BitSetSignature{},
				)
				require.NoError(err)
				return msg
			},
			err: errTest,
		},
		{
			name: "can't get validator set",
			stateF: func(ctrl *gomock.Controller) validators.State {
				state := validators.NewMockState(ctrl)
				state.EXPECT().GetSubnetID(gomock.Any(), sourceChainID).Return(sourceSubnetID, nil)
				state.EXPECT().GetValidatorSet(gomock.Any(), oChainHeight, sourceSubnetID).Return(nil, errTest)
				return state
			},
			quorumNum: 1,
			quorumDen: 2,
			msgF: func(require *require.Assertions) *odysseyWarp.Message {
				unsignedMsg, err := odysseyWarp.NewUnsignedMessage(
					networkID,
					sourceChainID,
					addressedPayloadBytes,
				)
				require.NoError(err)

				msg, err := odysseyWarp.NewMessage(
					unsignedMsg,
					&odysseyWarp.BitSetSignature{},
				)
				require.NoError(err)
				return msg
			},
			err: errTest,
		},
		{
			name: "weight overflow",
			stateF: func(ctrl *gomock.Controller) validators.State {
				state := validators.NewMockState(ctrl)
				state.EXPECT().GetSubnetID(gomock.Any(), sourceChainID).Return(sourceSubnetID, nil)
				state.EXPECT().GetValidatorSet(gomock.Any(), oChainHeight, sourceSubnetID).Return(map[ids.NodeID]*validators.GetValidatorOutput{
					testVdrs[0].nodeID: {
						NodeID:    testVdrs[0].nodeID,
						PublicKey: testVdrs[0].vdr.PublicKey,
						Weight:    math.MaxUint64,
					},
					testVdrs[1].nodeID: {
						NodeID:    testVdrs[1].nodeID,
						PublicKey: testVdrs[1].vdr.PublicKey,
						Weight:    math.MaxUint64,
					},
				}, nil)
				return state
			},
			quorumNum: 1,
			quorumDen: 2,
			msgF: func(require *require.Assertions) *odysseyWarp.Message {
				unsignedMsg, err := odysseyWarp.NewUnsignedMessage(
					networkID,
					sourceChainID,
					addressedPayloadBytes,
				)
				require.NoError(err)

				msg, err := odysseyWarp.NewMessage(
					unsignedMsg,
					&odysseyWarp.BitSetSignature{
						Signers: make([]byte, 8),
					},
				)
				require.NoError(err)
				return msg
			},
			err: odysseyWarp.ErrWeightOverflow,
		},
		{
			name: "invalid bit set index",
			stateF: func(ctrl *gomock.Controller) validators.State {
				state := validators.NewMockState(ctrl)
				state.EXPECT().GetSubnetID(gomock.Any(), sourceChainID).Return(sourceSubnetID, nil)
				state.EXPECT().GetValidatorSet(gomock.Any(), oChainHeight, sourceSubnetID).Return(vdrs, nil)
				return state
			},
			quorumNum: 1,
			quorumDen: 2,
			msgF: func(require *require.Assertions) *odysseyWarp.Message {
				unsignedMsg, err := odysseyWarp.NewUnsignedMessage(
					networkID,
					sourceChainID,
					addressedPayloadBytes,
				)
				require.NoError(err)

				msg, err := odysseyWarp.NewMessage(
					unsignedMsg,
					&odysseyWarp.BitSetSignature{
						Signers:   make([]byte, 1),
						Signature: [bls.SignatureLen]byte{},
					},
				)
				require.NoError(err)
				return msg
			},
			err: odysseyWarp.ErrInvalidBitSet,
		},
		{
			name: "unknown index",
			stateF: func(ctrl *gomock.Controller) validators.State {
				state := validators.NewMockState(ctrl)
				state.EXPECT().GetSubnetID(gomock.Any(), sourceChainID).Return(sourceSubnetID, nil)
				state.EXPECT().GetValidatorSet(gomock.Any(), oChainHeight, sourceSubnetID).Return(vdrs, nil)
				return state
			},
			quorumNum: 1,
			quorumDen: 2,
			msgF: func(require *require.Assertions) *odysseyWarp.Message {
				unsignedMsg, err := odysseyWarp.NewUnsignedMessage(
					networkID,
					sourceChainID,
					addressedPayloadBytes,
				)
				require.NoError(err)

				signers := set.NewBits()
				signers.Add(3) // vdr oob

				msg, err := odysseyWarp.NewMessage(
					unsignedMsg,
					&odysseyWarp.BitSetSignature{
						Signers:   signers.Bytes(),
						Signature: [bls.SignatureLen]byte{},
					},
				)
				require.NoError(err)
				return msg
			},
			err: odysseyWarp.ErrUnknownValidator,
		},
		{
			name: "insufficient weight",
			stateF: func(ctrl *gomock.Controller) validators.State {
				state := validators.NewMockState(ctrl)
				state.EXPECT().GetSubnetID(gomock.Any(), sourceChainID).Return(sourceSubnetID, nil)
				state.EXPECT().GetValidatorSet(gomock.Any(), oChainHeight, sourceSubnetID).Return(vdrs, nil)
				return state
			},
			quorumNum: 1,
			quorumDen: 1,
			msgF: func(require *require.Assertions) *odysseyWarp.Message {
				unsignedMsg, err := odysseyWarp.NewUnsignedMessage(
					networkID,
					sourceChainID,
					addressedPayloadBytes,
				)
				require.NoError(err)

				// [signers] has weight from [vdr[0], vdr[1]],
				// which is 6, which is less than 9
				signers := set.NewBits()
				signers.Add(0)
				signers.Add(1)

				unsignedBytes := unsignedMsg.Bytes()
				vdr0Sig := bls.Sign(testVdrs[0].sk, unsignedBytes)
				vdr1Sig := bls.Sign(testVdrs[1].sk, unsignedBytes)
				aggSig, err := bls.AggregateSignatures([]*bls.Signature{vdr0Sig, vdr1Sig})
				require.NoError(err)
				aggSigBytes := [bls.SignatureLen]byte{}
				copy(aggSigBytes[:], bls.SignatureToBytes(aggSig))

				msg, err := odysseyWarp.NewMessage(
					unsignedMsg,
					&odysseyWarp.BitSetSignature{
						Signers:   signers.Bytes(),
						Signature: aggSigBytes,
					},
				)
				require.NoError(err)
				return msg
			},
			err: odysseyWarp.ErrInsufficientWeight,
		},
		{
			name: "can't parse sig",
			stateF: func(ctrl *gomock.Controller) validators.State {
				state := validators.NewMockState(ctrl)
				state.EXPECT().GetSubnetID(gomock.Any(), sourceChainID).Return(sourceSubnetID, nil)
				state.EXPECT().GetValidatorSet(gomock.Any(), oChainHeight, sourceSubnetID).Return(vdrs, nil)
				return state
			},
			quorumNum: 1,
			quorumDen: 2,
			msgF: func(require *require.Assertions) *odysseyWarp.Message {
				unsignedMsg, err := odysseyWarp.NewUnsignedMessage(
					networkID,
					sourceChainID,
					addressedPayloadBytes,
				)
				require.NoError(err)

				signers := set.NewBits()
				signers.Add(0)
				signers.Add(1)

				msg, err := odysseyWarp.NewMessage(
					unsignedMsg,
					&odysseyWarp.BitSetSignature{
						Signers:   signers.Bytes(),
						Signature: [bls.SignatureLen]byte{},
					},
				)
				require.NoError(err)
				return msg
			},
			err: odysseyWarp.ErrParseSignature,
		},
		{
			name: "no validators",
			stateF: func(ctrl *gomock.Controller) validators.State {
				state := validators.NewMockState(ctrl)
				state.EXPECT().GetSubnetID(gomock.Any(), sourceChainID).Return(sourceSubnetID, nil)
				state.EXPECT().GetValidatorSet(gomock.Any(), oChainHeight, sourceSubnetID).Return(nil, nil)
				return state
			},
			quorumNum: 1,
			quorumDen: 2,
			msgF: func(require *require.Assertions) *odysseyWarp.Message {
				unsignedMsg, err := odysseyWarp.NewUnsignedMessage(
					networkID,
					sourceChainID,
					addressedPayloadBytes,
				)
				require.NoError(err)

				unsignedBytes := unsignedMsg.Bytes()
				vdr0Sig := bls.Sign(testVdrs[0].sk, unsignedBytes)
				aggSigBytes := [bls.SignatureLen]byte{}
				copy(aggSigBytes[:], bls.SignatureToBytes(vdr0Sig))

				msg, err := odysseyWarp.NewMessage(
					unsignedMsg,
					&odysseyWarp.BitSetSignature{
						Signers:   nil,
						Signature: aggSigBytes,
					},
				)
				require.NoError(err)
				return msg
			},
			err: bls.ErrNoPublicKeys,
		},
		{
			name: "invalid signature (substitute)",
			stateF: func(ctrl *gomock.Controller) validators.State {
				state := validators.NewMockState(ctrl)
				state.EXPECT().GetSubnetID(gomock.Any(), sourceChainID).Return(sourceSubnetID, nil)
				state.EXPECT().GetValidatorSet(gomock.Any(), oChainHeight, sourceSubnetID).Return(vdrs, nil)
				return state
			},
			quorumNum: 3,
			quorumDen: 5,
			msgF: func(require *require.Assertions) *odysseyWarp.Message {
				unsignedMsg, err := odysseyWarp.NewUnsignedMessage(
					networkID,
					sourceChainID,
					addressedPayloadBytes,
				)
				require.NoError(err)

				signers := set.NewBits()
				signers.Add(0)
				signers.Add(1)

				unsignedBytes := unsignedMsg.Bytes()
				vdr0Sig := bls.Sign(testVdrs[0].sk, unsignedBytes)
				// Give sig from vdr[2] even though the bit vector says it
				// should be from vdr[1]
				vdr2Sig := bls.Sign(testVdrs[2].sk, unsignedBytes)
				aggSig, err := bls.AggregateSignatures([]*bls.Signature{vdr0Sig, vdr2Sig})
				require.NoError(err)
				aggSigBytes := [bls.SignatureLen]byte{}
				copy(aggSigBytes[:], bls.SignatureToBytes(aggSig))

				msg, err := odysseyWarp.NewMessage(
					unsignedMsg,
					&odysseyWarp.BitSetSignature{
						Signers:   signers.Bytes(),
						Signature: aggSigBytes,
					},
				)
				require.NoError(err)
				return msg
			},
			err: odysseyWarp.ErrInvalidSignature,
		},
		{
			name: "invalid signature (missing one)",
			stateF: func(ctrl *gomock.Controller) validators.State {
				state := validators.NewMockState(ctrl)
				state.EXPECT().GetSubnetID(gomock.Any(), sourceChainID).Return(sourceSubnetID, nil)
				state.EXPECT().GetValidatorSet(gomock.Any(), oChainHeight, sourceSubnetID).Return(vdrs, nil)
				return state
			},
			quorumNum: 3,
			quorumDen: 5,
			msgF: func(require *require.Assertions) *odysseyWarp.Message {
				unsignedMsg, err := odysseyWarp.NewUnsignedMessage(
					networkID,
					sourceChainID,
					addressedPayloadBytes,
				)
				require.NoError(err)

				signers := set.NewBits()
				signers.Add(0)
				signers.Add(1)

				unsignedBytes := unsignedMsg.Bytes()
				vdr0Sig := bls.Sign(testVdrs[0].sk, unsignedBytes)
				// Don't give the sig from vdr[1]
				aggSigBytes := [bls.SignatureLen]byte{}
				copy(aggSigBytes[:], bls.SignatureToBytes(vdr0Sig))

				msg, err := odysseyWarp.NewMessage(
					unsignedMsg,
					&odysseyWarp.BitSetSignature{
						Signers:   signers.Bytes(),
						Signature: aggSigBytes,
					},
				)
				require.NoError(err)
				return msg
			},
			err: odysseyWarp.ErrInvalidSignature,
		},
		{
			name: "invalid signature (extra one)",
			stateF: func(ctrl *gomock.Controller) validators.State {
				state := validators.NewMockState(ctrl)
				state.EXPECT().GetSubnetID(gomock.Any(), sourceChainID).Return(sourceSubnetID, nil)
				state.EXPECT().GetValidatorSet(gomock.Any(), oChainHeight, sourceSubnetID).Return(vdrs, nil)
				return state
			},
			quorumNum: 3,
			quorumDen: 5,
			msgF: func(require *require.Assertions) *odysseyWarp.Message {
				unsignedMsg, err := odysseyWarp.NewUnsignedMessage(
					networkID,
					sourceChainID,
					addressedPayloadBytes,
				)
				require.NoError(err)

				signers := set.NewBits()
				signers.Add(0)
				signers.Add(1)

				unsignedBytes := unsignedMsg.Bytes()
				vdr0Sig := bls.Sign(testVdrs[0].sk, unsignedBytes)
				vdr1Sig := bls.Sign(testVdrs[1].sk, unsignedBytes)
				// Give sig from vdr[2] even though the bit vector doesn't have
				// it
				vdr2Sig := bls.Sign(testVdrs[2].sk, unsignedBytes)
				aggSig, err := bls.AggregateSignatures([]*bls.Signature{vdr0Sig, vdr1Sig, vdr2Sig})
				require.NoError(err)
				aggSigBytes := [bls.SignatureLen]byte{}
				copy(aggSigBytes[:], bls.SignatureToBytes(aggSig))

				msg, err := odysseyWarp.NewMessage(
					unsignedMsg,
					&odysseyWarp.BitSetSignature{
						Signers:   signers.Bytes(),
						Signature: aggSigBytes,
					},
				)
				require.NoError(err)
				return msg
			},
			err: odysseyWarp.ErrInvalidSignature,
		},
		{
			name: "valid signature",
			stateF: func(ctrl *gomock.Controller) validators.State {
				state := validators.NewMockState(ctrl)
				state.EXPECT().GetSubnetID(gomock.Any(), sourceChainID).Return(sourceSubnetID, nil)
				state.EXPECT().GetValidatorSet(gomock.Any(), oChainHeight, sourceSubnetID).Return(vdrs, nil)
				return state
			},
			quorumNum: 1,
			quorumDen: 2,
			msgF: func(require *require.Assertions) *odysseyWarp.Message {
				unsignedMsg, err := odysseyWarp.NewUnsignedMessage(
					networkID,
					sourceChainID,
					addressedPayloadBytes,
				)
				require.NoError(err)

				// [signers] has weight from [vdr[1], vdr[2]],
				// which is 6, which is greater than 4.5
				signers := set.NewBits()
				signers.Add(1)
				signers.Add(2)

				unsignedBytes := unsignedMsg.Bytes()
				vdr1Sig := bls.Sign(testVdrs[1].sk, unsignedBytes)
				vdr2Sig := bls.Sign(testVdrs[2].sk, unsignedBytes)
				aggSig, err := bls.AggregateSignatures([]*bls.Signature{vdr1Sig, vdr2Sig})
				require.NoError(err)
				aggSigBytes := [bls.SignatureLen]byte{}
				copy(aggSigBytes[:], bls.SignatureToBytes(aggSig))

				msg, err := odysseyWarp.NewMessage(
					unsignedMsg,
					&odysseyWarp.BitSetSignature{
						Signers:   signers.Bytes(),
						Signature: aggSigBytes,
					},
				)
				require.NoError(err)
				return msg
			},
			err: nil,
		},
		{
			name: "valid signature (boundary)",
			stateF: func(ctrl *gomock.Controller) validators.State {
				state := validators.NewMockState(ctrl)
				state.EXPECT().GetSubnetID(gomock.Any(), sourceChainID).Return(sourceSubnetID, nil)
				state.EXPECT().GetValidatorSet(gomock.Any(), oChainHeight, sourceSubnetID).Return(vdrs, nil)
				return state
			},
			quorumNum: 2,
			quorumDen: 3,
			msgF: func(require *require.Assertions) *odysseyWarp.Message {
				unsignedMsg, err := odysseyWarp.NewUnsignedMessage(
					networkID,
					sourceChainID,
					addressedPayloadBytes,
				)
				require.NoError(err)

				// [signers] has weight from [vdr[1], vdr[2]],
				// which is 6, which meets the minimum 6
				signers := set.NewBits()
				signers.Add(1)
				signers.Add(2)

				unsignedBytes := unsignedMsg.Bytes()
				vdr1Sig := bls.Sign(testVdrs[1].sk, unsignedBytes)
				vdr2Sig := bls.Sign(testVdrs[2].sk, unsignedBytes)
				aggSig, err := bls.AggregateSignatures([]*bls.Signature{vdr1Sig, vdr2Sig})
				require.NoError(err)
				aggSigBytes := [bls.SignatureLen]byte{}
				copy(aggSigBytes[:], bls.SignatureToBytes(aggSig))

				msg, err := odysseyWarp.NewMessage(
					unsignedMsg,
					&odysseyWarp.BitSetSignature{
						Signers:   signers.Bytes(),
						Signature: aggSigBytes,
					},
				)
				require.NoError(err)
				return msg
			},
			err: nil,
		},
		{
			name: "valid signature (missing key)",
			stateF: func(ctrl *gomock.Controller) validators.State {
				state := validators.NewMockState(ctrl)
				state.EXPECT().GetSubnetID(gomock.Any(), sourceChainID).Return(sourceSubnetID, nil)
				state.EXPECT().GetValidatorSet(gomock.Any(), oChainHeight, sourceSubnetID).Return(map[ids.NodeID]*validators.GetValidatorOutput{
					testVdrs[0].nodeID: {
						NodeID:    testVdrs[0].nodeID,
						PublicKey: nil,
						Weight:    testVdrs[0].vdr.Weight,
					},
					testVdrs[1].nodeID: {
						NodeID:    testVdrs[1].nodeID,
						PublicKey: testVdrs[1].vdr.PublicKey,
						Weight:    testVdrs[1].vdr.Weight,
					},
					testVdrs[2].nodeID: {
						NodeID:    testVdrs[2].nodeID,
						PublicKey: testVdrs[2].vdr.PublicKey,
						Weight:    testVdrs[2].vdr.Weight,
					},
				}, nil)
				return state
			},
			quorumNum: 1,
			quorumDen: 3,
			msgF: func(require *require.Assertions) *odysseyWarp.Message {
				unsignedMsg, err := odysseyWarp.NewUnsignedMessage(
					networkID,
					sourceChainID,
					addressedPayloadBytes,
				)
				require.NoError(err)

				// [signers] has weight from [vdr2, vdr3],
				// which is 6, which is greater than 3
				signers := set.NewBits()
				// Note: the bits are shifted because vdr[0]'s key was zeroed
				signers.Add(0) // vdr[1]
				signers.Add(1) // vdr[2]

				unsignedBytes := unsignedMsg.Bytes()
				vdr1Sig := bls.Sign(testVdrs[1].sk, unsignedBytes)
				vdr2Sig := bls.Sign(testVdrs[2].sk, unsignedBytes)
				aggSig, err := bls.AggregateSignatures([]*bls.Signature{vdr1Sig, vdr2Sig})
				require.NoError(err)
				aggSigBytes := [bls.SignatureLen]byte{}
				copy(aggSigBytes[:], bls.SignatureToBytes(aggSig))

				msg, err := odysseyWarp.NewMessage(
					unsignedMsg,
					&odysseyWarp.BitSetSignature{
						Signers:   signers.Bytes(),
						Signature: aggSigBytes,
					},
				)
				require.NoError(err)
				return msg
			},
			err: nil,
		},
		{
			name: "valid signature (duplicate key)",
			stateF: func(ctrl *gomock.Controller) validators.State {
				state := validators.NewMockState(ctrl)
				state.EXPECT().GetSubnetID(gomock.Any(), sourceChainID).Return(sourceSubnetID, nil)
				state.EXPECT().GetValidatorSet(gomock.Any(), oChainHeight, sourceSubnetID).Return(map[ids.NodeID]*validators.GetValidatorOutput{
					testVdrs[0].nodeID: {
						NodeID:    testVdrs[0].nodeID,
						PublicKey: nil,
						Weight:    testVdrs[0].vdr.Weight,
					},
					testVdrs[1].nodeID: {
						NodeID:    testVdrs[1].nodeID,
						PublicKey: testVdrs[2].vdr.PublicKey,
						Weight:    testVdrs[1].vdr.Weight,
					},
					testVdrs[2].nodeID: {
						NodeID:    testVdrs[2].nodeID,
						PublicKey: testVdrs[2].vdr.PublicKey,
						Weight:    testVdrs[2].vdr.Weight,
					},
				}, nil)
				return state
			},
			quorumNum: 2,
			quorumDen: 3,
			msgF: func(require *require.Assertions) *odysseyWarp.Message {
				unsignedMsg, err := odysseyWarp.NewUnsignedMessage(
					networkID,
					sourceChainID,
					addressedPayloadBytes,
				)
				require.NoError(err)

				// [signers] has weight from [vdr2, vdr3],
				// which is 6, which meets the minimum 6
				signers := set.NewBits()
				// Note: the bits are shifted because vdr[0]'s key was zeroed
				// Note: vdr[1] and vdr[2] were combined because of a shared pk
				signers.Add(0) // vdr[1] + vdr[2]

				unsignedBytes := unsignedMsg.Bytes()
				// Because vdr[1] and vdr[2] share a key, only one of them sign.
				vdr2Sig := bls.Sign(testVdrs[2].sk, unsignedBytes)
				aggSigBytes := [bls.SignatureLen]byte{}
				copy(aggSigBytes[:], bls.SignatureToBytes(vdr2Sig))

				msg, err := odysseyWarp.NewMessage(
					unsignedMsg,
					&odysseyWarp.BitSetSignature{
						Signers:   signers.Bytes(),
						Signature: aggSigBytes,
					},
				)
				require.NoError(err)
				return msg
			},
			err: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			msg := tt.msgF(require)
			oChainState := tt.stateF(ctrl)

			err := msg.Signature.Verify(
				context.Background(),
				&msg.UnsignedMessage,
				networkID,
				oChainState,
				oChainHeight,
				tt.quorumNum,
				tt.quorumDen,
			)
			require.ErrorIs(err, tt.err)
		})
	}
}
