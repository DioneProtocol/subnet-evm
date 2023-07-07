// (c) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package warp

import (
	"testing"

	"github.com/ava-labs/avalanchego/database/memdb"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/utils/crypto/bls"
	"github.com/ava-labs/avalanchego/utils/hashing"
	avalancheWarp "github.com/ava-labs/avalanchego/vms/platformvm/warp"
	"github.com/stretchr/testify/require"
)

var (
	sourceChainID      = ids.GenerateTestID()
	destinationChainID = ids.GenerateTestID()
	payload            = []byte("test")
)

func TestClearDB(t *testing.T) {
	db := memdb.New()

	snowCtx := snow.DefaultContextTest()
	sk, err := bls.NewSecretKey()
	require.NoError(t, err)
	snowCtx.WarpSigner = avalancheWarp.NewSigner(sk, sourceChainID)
	backend := NewWarpBackend(snowCtx, db, 500)

	// use multiple messages to test that all messages get cleared
	payloads := [][]byte{[]byte("test1"), []byte("test2"), []byte("test3"), []byte("test4"), []byte("test5")}
	messageIDs := []ids.ID{}

	// add all messages
	for _, payload := range payloads {
		unsignedMsg, err := avalancheWarp.NewUnsignedMessage(sourceChainID, destinationChainID, payload)
		require.NoError(t, err)
		messageID := hashing.ComputeHash256Array(unsignedMsg.Bytes())
		messageIDs = append(messageIDs, messageID)
		err = backend.AddMessage(unsignedMsg)
		require.NoError(t, err)
		// ensure that the message was added
		_, err = backend.GetSignature(messageID)
		require.NoError(t, err)
	}

	err = backend.Clear()
	require.NoError(t, err)

	// ensure all messages have been deleted
	for _, messageID := range messageIDs {
		_, err := backend.GetSignature(messageID)
		require.ErrorContains(t, err, "failed to get warp message")
	}
}

func TestAddAndGetValidMessage(t *testing.T) {
	db := memdb.New()

	snowCtx := snow.DefaultContextTest()
	sk, err := bls.NewSecretKey()
	require.NoError(t, err)
	snowCtx.WarpSigner = avalancheWarp.NewSigner(sk, sourceChainID)
	backend := NewWarpBackend(snowCtx, db, 500)

	// Create a new unsigned message and add it to the warp backend.
	unsignedMsg, err := avalancheWarp.NewUnsignedMessage(sourceChainID, destinationChainID, payload)
	require.NoError(t, err)
	err = backend.AddMessage(unsignedMsg)
	require.NoError(t, err)

	// Verify that a signature is returned successfully, and compare to expected signature.
	messageID := unsignedMsg.ID()
	signature, err := backend.GetSignature(messageID)
	require.NoError(t, err)

	expectedSig, err := snowCtx.WarpSigner.Sign(unsignedMsg)
	require.NoError(t, err)
	require.Equal(t, expectedSig, signature[:])
}

func TestAddAndGetUnknownMessage(t *testing.T) {
	db := memdb.New()

	backend := NewWarpBackend(snow.DefaultContextTest(), db, 500)
	unsignedMsg, err := avalancheWarp.NewUnsignedMessage(sourceChainID, destinationChainID, payload)
	require.NoError(t, err)

	// Try getting a signature for a message that was not added.
	messageID := unsignedMsg.ID()
	_, err = backend.GetSignature(messageID)
	require.Error(t, err)
}

func TestZeroSizedCache(t *testing.T) {
	db := memdb.New()

	snowCtx := snow.DefaultContextTest()
	sk, err := bls.NewSecretKey()
	require.NoError(t, err)
	snowCtx.WarpSigner = avalancheWarp.NewSigner(sk, sourceChainID)

	// Verify zero sized cache works normally, because the lru cache will be initialized to size 1 for any size parameter <= 0.
	backend := NewWarpBackend(snowCtx, db, 0)

	// Create a new unsigned message and add it to the warp backend.
	unsignedMsg, err := avalancheWarp.NewUnsignedMessage(sourceChainID, destinationChainID, payload)
	require.NoError(t, err)
	err = backend.AddMessage(unsignedMsg)
	require.NoError(t, err)

	// Verify that a signature is returned successfully, and compare to expected signature.
	messageID := unsignedMsg.ID()
	signature, err := backend.GetSignature(messageID)
	require.NoError(t, err)

	expectedSig, err := snowCtx.WarpSigner.Sign(unsignedMsg)
	require.NoError(t, err)
	require.Equal(t, expectedSig, signature[:])
}
