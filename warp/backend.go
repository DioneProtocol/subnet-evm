// (c) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package warp

import (
	"fmt"

	"github.com/DioneProtocol/odysseygo/cache"
	"github.com/DioneProtocol/odysseygo/database"
	"github.com/DioneProtocol/odysseygo/ids"
	"github.com/DioneProtocol/odysseygo/utils/crypto/bls"
	odysseyWarp "github.com/DioneProtocol/odysseygo/vms/omegavm/warp"
	"github.com/DioneProtocol/subnet-evm/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

var _ Backend = &backend{}

const batchSize = ethdb.IdealBatchSize

// Backend tracks signature-eligible warp messages and provides an interface to fetch them.
// The backend is also used to query for warp message signatures by the signature request handler.
type Backend interface {
	// AddMessage signs [unsignedMessage] and adds it to the warp backend database
	AddMessage(unsignedMessage *odysseyWarp.UnsignedMessage) error

	// GetSignature returns the signature of the requested message hash.
	GetSignature(messageHash ids.ID) ([bls.SignatureLen]byte, error)

	// GetMessage retrieves the [unsignedMessage] from the warp backend database if available
	GetMessage(messageHash ids.ID) (*odysseyWarp.UnsignedMessage, error)

	// Clear clears the entire db
	Clear() error
}

// backend implements Backend, keeps track of warp messages, and generates message signatures.
type backend struct {
	db             database.Database
	warpSigner     odysseyWarp.Signer
	signatureCache *cache.LRU[ids.ID, [bls.SignatureLen]byte]
	messageCache   *cache.LRU[ids.ID, *odysseyWarp.UnsignedMessage]
}

// NewBackend creates a new Backend, and initializes the signature cache and message tracking database.
func NewBackend(warpSigner odysseyWarp.Signer, db database.Database, cacheSize int) Backend {
	return &backend{
		db:             db,
		warpSigner:     warpSigner,
		signatureCache: &cache.LRU[ids.ID, [bls.SignatureLen]byte]{Size: cacheSize},
		messageCache:   &cache.LRU[ids.ID, *odysseyWarp.UnsignedMessage]{Size: cacheSize},
	}
}

func (b *backend) Clear() error {
	b.signatureCache.Flush()
	return database.Clear(b.db, batchSize)
}

func (b *backend) AddMessage(unsignedMessage *odysseyWarp.UnsignedMessage) error {
	messageID := unsignedMessage.ID()

	// In the case when a node restarts, and possibly changes its bls key, the cache gets emptied but the database does not.
	// So to avoid having incorrect signatures saved in the database after a bls key change, we save the full message in the database.
	// Whereas for the cache, after the node restart, the cache would be emptied so we can directly save the signatures.
	if err := b.db.Put(messageID[:], unsignedMessage.Bytes()); err != nil {
		return fmt.Errorf("failed to put warp signature in db: %w", err)
	}

	var signature [bls.SignatureLen]byte
	sig, err := b.warpSigner.Sign(unsignedMessage)
	if err != nil {
		return fmt.Errorf("failed to sign warp message: %w", err)
	}

	copy(signature[:], sig)
	b.signatureCache.Put(messageID, signature)
	log.Debug("Adding warp message to backend", "messageID", messageID)
	return nil
}

func (b *backend) GetSignature(messageID ids.ID) ([bls.SignatureLen]byte, error) {
	log.Debug("Getting warp message from backend", "messageID", messageID)
	if sig, ok := b.signatureCache.Get(messageID); ok {
		return sig, nil
	}

	unsignedMessage, err := b.GetMessage(messageID)
	if err != nil {
		return [bls.SignatureLen]byte{}, fmt.Errorf("failed to get warp message %s from db: %w", messageID.String(), err)
	}

	var signature [bls.SignatureLen]byte
	sig, err := b.warpSigner.Sign(unsignedMessage)
	if err != nil {
		return [bls.SignatureLen]byte{}, fmt.Errorf("failed to sign warp message: %w", err)
	}

	copy(signature[:], sig)
	b.signatureCache.Put(messageID, signature)
	return signature, nil
}

func (b *backend) GetMessage(messageID ids.ID) (*odysseyWarp.UnsignedMessage, error) {
	if message, ok := b.messageCache.Get(messageID); ok {
		return message, nil
	}

	unsignedMessageBytes, err := b.db.Get(messageID[:])
	if err != nil {
		return nil, fmt.Errorf("failed to get warp message %s from db: %w", messageID.String(), err)
	}

	unsignedMessage, err := odysseyWarp.ParseUnsignedMessage(unsignedMessageBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse unsigned message %s: %w", messageID.String(), err)
	}
	b.messageCache.Put(messageID, unsignedMessage)

	return unsignedMessage, nil
}
