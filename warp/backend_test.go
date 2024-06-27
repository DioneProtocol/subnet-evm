// (c) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package warp

import (
	"testing"

	"github.com/DioneProtocol/odysseygo/database/memdb"
	"github.com/DioneProtocol/odysseygo/ids"
	"github.com/DioneProtocol/odysseygo/utils/crypto/bls"
	"github.com/DioneProtocol/odysseygo/utils/hashing"
	odysseyWarp "github.com/DioneProtocol/odysseygo/vms/omegavm/warp"
	"github.com/stretchr/testify/require"
)

var (
	networkID     uint32 = 54321
	sourceChainID        = ids.GenerateTestID()
	payload              = []byte("test")
)

func TestClearDB(t *testing.T) {
	db := memdb.New()

	sk, err := bls.NewSecretKey()
	require.NoError(t, err)
	warpSigner := odysseyWarp.NewSigner(sk, networkID, sourceChainID)
	backend := NewBackend(warpSigner, db, 500)

	// use multiple messages to test that all messages get cleared
	payloads := [][]byte{[]byte("test1"), []byte("test2"), []byte("test3"), []byte("test4"), []byte("test5")}
	messageIDs := []ids.ID{}

	// add all messages
	for _, payload := range payloads {
		unsignedMsg, err := odysseyWarp.NewUnsignedMessage(networkID, sourceChainID, payload)
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

	sk, err := bls.NewSecretKey()
	require.NoError(t, err)
	warpSigner := odysseyWarp.NewSigner(sk, networkID, sourceChainID)
	backend := NewBackend(warpSigner, db, 500)

	// Create a new unsigned message and add it to the warp backend.
	unsignedMsg, err := odysseyWarp.NewUnsignedMessage(networkID, sourceChainID, payload)
	require.NoError(t, err)
	err = backend.AddMessage(unsignedMsg)
	require.NoError(t, err)

	// Verify that a signature is returned successfully, and compare to expected signature.
	messageID := unsignedMsg.ID()
	signature, err := backend.GetSignature(messageID)
	require.NoError(t, err)

	expectedSig, err := warpSigner.Sign(unsignedMsg)
	require.NoError(t, err)
	require.Equal(t, expectedSig, signature[:])
}

func TestAddAndGetUnknownMessage(t *testing.T) {
	db := memdb.New()

	sk, err := bls.NewSecretKey()
	require.NoError(t, err)
	warpSigner := odysseyWarp.NewSigner(sk, networkID, sourceChainID)
	backend := NewBackend(warpSigner, db, 500)
	unsignedMsg, err := odysseyWarp.NewUnsignedMessage(networkID, sourceChainID, payload)
	require.NoError(t, err)

	// Try getting a signature for a message that was not added.
	messageID := unsignedMsg.ID()
	_, err = backend.GetSignature(messageID)
	require.Error(t, err)
}

func TestZeroSizedCache(t *testing.T) {
	db := memdb.New()

	sk, err := bls.NewSecretKey()
	require.NoError(t, err)
	warpSigner := odysseyWarp.NewSigner(sk, networkID, sourceChainID)

	// Verify zero sized cache works normally, because the lru cache will be initialized to size 1 for any size parameter <= 0.
	backend := NewBackend(warpSigner, db, 0)

	// Create a new unsigned message and add it to the warp backend.
	unsignedMsg, err := odysseyWarp.NewUnsignedMessage(networkID, sourceChainID, payload)
	require.NoError(t, err)
	err = backend.AddMessage(unsignedMsg)
	require.NoError(t, err)

	// Verify that a signature is returned successfully, and compare to expected signature.
	messageID := unsignedMsg.ID()
	signature, err := backend.GetSignature(messageID)
	require.NoError(t, err)

	expectedSig, err := warpSigner.Sign(unsignedMsg)
	require.NoError(t, err)
	require.Equal(t, expectedSig, signature[:])
}
