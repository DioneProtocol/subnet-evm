// (c) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/DioneProtocol/odysseygo/database/memdb"
	"github.com/DioneProtocol/odysseygo/ids"
	"github.com/DioneProtocol/odysseygo/snow"
	"github.com/DioneProtocol/odysseygo/utils/crypto/bls"
	odysseyWarp "github.com/DioneProtocol/odysseygo/vms/omegavm/warp"
	"github.com/DioneProtocol/subnet-evm/plugin/evm/message"
	"github.com/DioneProtocol/subnet-evm/warp"
	"github.com/DioneProtocol/subnet-evm/warp/handlers/stats"
	"github.com/stretchr/testify/require"
)

func TestSignatureHandler(t *testing.T) {
	database := memdb.New()
	snowCtx := snow.DefaultContextTest()
	blsSecretKey, err := bls.NewSecretKey()
	require.NoError(t, err)

	warpSigner := odysseyWarp.NewSigner(blsSecretKey, snowCtx.NetworkID, snowCtx.ChainID)
	backend := warp.NewBackend(warpSigner, database, 100)

	msg, err := odysseyWarp.NewUnsignedMessage(snowCtx.NetworkID, snowCtx.ChainID, []byte("test"))
	require.NoError(t, err)

	messageID := msg.ID()
	require.NoError(t, backend.AddMessage(msg))
	signature, err := backend.GetSignature(messageID)
	require.NoError(t, err)
	unknownMessageID := ids.GenerateTestID()

	emptySignature := [bls.SignatureLen]byte{}
	mockHandlerStats := &stats.MockSignatureRequestHandlerStats{}
	signatureRequestHandler := NewSignatureRequestHandler(backend, message.Codec, mockHandlerStats)

	tests := map[string]struct {
		setup       func() (request message.SignatureRequest, expectedResponse []byte)
		verifyStats func(t *testing.T, stats *stats.MockSignatureRequestHandlerStats)
	}{
		"normal": {
			setup: func() (request message.SignatureRequest, expectedResponse []byte) {
				return message.SignatureRequest{
					MessageID: messageID,
				}, signature[:]
			},
			verifyStats: func(t *testing.T, stats *stats.MockSignatureRequestHandlerStats) {
				require.EqualValues(t, 1, mockHandlerStats.SignatureRequestCount)
				require.EqualValues(t, 1, mockHandlerStats.SignatureRequestHit)
				require.EqualValues(t, 0, mockHandlerStats.SignatureRequestMiss)
				require.Greater(t, mockHandlerStats.SignatureRequestDuration, time.Duration(0))
			},
		},
		"unknown": {
			setup: func() (request message.SignatureRequest, expectedResponse []byte) {
				return message.SignatureRequest{
					MessageID: unknownMessageID,
				}, emptySignature[:]
			},
			verifyStats: func(t *testing.T, stats *stats.MockSignatureRequestHandlerStats) {
				require.EqualValues(t, 1, mockHandlerStats.SignatureRequestCount)
				require.EqualValues(t, 1, mockHandlerStats.SignatureRequestMiss)
				require.EqualValues(t, 0, mockHandlerStats.SignatureRequestHit)
				require.Greater(t, mockHandlerStats.SignatureRequestDuration, time.Duration(0))
			},
		},
	}

	for name, test := range tests {
		// Reset stats before each test
		mockHandlerStats.Reset()

		t.Run(name, func(t *testing.T) {
			request, expectedResponse := test.setup()
			responseBytes, err := signatureRequestHandler.OnSignatureRequest(context.Background(), ids.GenerateTestNodeID(), 1, request)
			require.NoError(t, err)

			// If the expected response is empty, assert that the handler returns an empty response and return early.
			if len(expectedResponse) == 0 {
				test.verifyStats(t, mockHandlerStats)
				require.Len(t, responseBytes, 0, "expected response to be empty")
				return
			}
			var response message.SignatureResponse
			_, err = message.Codec.Unmarshal(responseBytes, &response)
			require.NoError(t, err, "error unmarshalling SignatureResponse")

			require.Equal(t, expectedResponse, response.Signature[:])
			test.verifyStats(t, mockHandlerStats)
		})
	}
}
