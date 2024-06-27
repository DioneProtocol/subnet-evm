// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/DioneProtocol/subnet-evm/core"
	"github.com/DioneProtocol/subnet-evm/core/rawdb"
	"github.com/DioneProtocol/subnet-evm/core/types"
	"github.com/DioneProtocol/subnet-evm/params"
	"github.com/DioneProtocol/subnet-evm/precompile/precompileconfig"
	"github.com/DioneProtocol/subnet-evm/precompile/results"
	"github.com/DioneProtocol/subnet-evm/warp/payload"
	"github.com/DioneProtocol/subnet-evm/x/warp"

	"github.com/DioneProtocol/odysseygo/ids"
	"github.com/DioneProtocol/odysseygo/snow/choices"
	"github.com/DioneProtocol/odysseygo/snow/consensus/snowman"
	"github.com/DioneProtocol/odysseygo/snow/engine/snowman/block"
	odysseyWarp "github.com/DioneProtocol/odysseygo/vms/omegavm/warp"
)

var (
	_ snowman.Block           = (*Block)(nil)
	_ block.WithVerifyContext = (*Block)(nil)
)

// Block implements the snowman.Block interface
type Block struct {
	id       ids.ID
	ethBlock *types.Block
	vm       *VM
	status   choices.Status
}

// newBlock returns a new Block wrapping the ethBlock type and implementing the snowman.Block interface
func (vm *VM) newBlock(ethBlock *types.Block) *Block {
	return &Block{
		id:       ids.ID(ethBlock.Hash()),
		ethBlock: ethBlock,
		vm:       vm,
	}
}

// ID implements the snowman.Block interface
func (b *Block) ID() ids.ID { return b.id }

// Accept implements the snowman.Block interface
func (b *Block) Accept(context.Context) error {
	vm := b.vm

	// Although returning an error from Accept is considered fatal, it is good
	// practice to cleanup the batch we were modifying in the case of an error.
	defer vm.db.Abort()

	b.status = choices.Accepted
	log.Debug(fmt.Sprintf("Accepting block %s (%s) at height %d", b.ID().Hex(), b.ID(), b.Height()))

	// Call Accept for relevant precompile logs. Note we do this prior to
	// calling Accept on the blockChain so any side effects (eg warp signatures)
	// take place before the accepted log is emitted to subscribers. Use of the
	// sharedMemoryWriter ensures shared memory requests generated by
	// precompiles are committed atomically with the vm's lastAcceptedKey.
	rules := b.vm.chainConfig.OdysseyRules(b.ethBlock.Number(), b.ethBlock.Timestamp())
	sharedMemoryWriter := NewSharedMemoryWriter()
	if err := b.handlePrecompileAccept(&rules, sharedMemoryWriter); err != nil {
		return err
	}
	if err := vm.blockChain.Accept(b.ethBlock); err != nil {
		return fmt.Errorf("chain could not accept %s: %w", b.ID(), err)
	}

	if err := vm.acceptedBlockDB.Put(lastAcceptedKey, b.id[:]); err != nil {
		return fmt.Errorf("failed to put %s as the last accepted block: %w", b.ID(), err)
	}

	// Get pending operations on the vm's versionDB so we can apply them atomically
	// with the shared memory requests.
	vdbBatch, err := vm.db.CommitBatch()
	if err != nil {
		return fmt.Errorf("failed to get commit batch: %w", err)
	}

	// Apply any shared memory requests that accumulated from processing the logs
	// of the accepted block (generated by precompiles) atomically with other pending
	// changes to the vm's versionDB.
	return vm.ctx.SharedMemory.Apply(sharedMemoryWriter.requests, vdbBatch)
}

// handlePrecompileAccept calls Accept on any logs generated with an active precompile address that implements
// contract.Accepter
// This function assumes that the Accept function will ONLY operate on state maintained in the VM's versiondb.
// This ensures that any DB operations are performed atomically with marking the block as accepted.
func (b *Block) handlePrecompileAccept(rules *params.Rules, sharedMemoryWriter *sharedMemoryWriter) error {
	// Short circuit early if there are no precompile accepters to execute
	if len(rules.AccepterPrecompiles) == 0 {
		return nil
	}

	// Read receipts from disk
	receipts := rawdb.ReadReceipts(b.vm.chaindb, b.ethBlock.Hash(), b.ethBlock.NumberU64(), b.vm.chainConfig)
	// If there are no receipts, ReadReceipts may be nil, so we check the length and confirm the ReceiptHash
	// is empty to ensure that missing receipts results in an error on accept.
	if len(receipts) == 0 && b.ethBlock.ReceiptHash() != types.EmptyRootHash {
		return fmt.Errorf("failed to fetch receipts for accepted block with non-empty root hash (%s) (Block: %s, Height: %d)", b.ethBlock.ReceiptHash(), b.ethBlock.Hash(), b.ethBlock.NumberU64())
	}
	acceptCtx := &precompileconfig.AcceptContext{
		SnowCtx:      b.vm.ctx,
		SharedMemory: sharedMemoryWriter,
		Warp:         b.vm.warpBackend,
	}
	for _, receipt := range receipts {
		for logIdx, log := range receipt.Logs {
			accepter, ok := rules.AccepterPrecompiles[log.Address]
			if !ok {
				continue
			}
			if err := accepter.Accept(acceptCtx, log.TxHash, logIdx, log.Topics, log.Data); err != nil {
				return err
			}
		}
	}

	// If Warp is enabled, add the block hash as an unsigned message to the warp backend.
	if rules.IsPrecompileEnabled(warp.ContractAddress) {
		blockHashPayload, err := payload.NewBlockHashPayload(b.ethBlock.Hash())
		if err != nil {
			return fmt.Errorf("failed to create block hash payload: %w", err)
		}
		unsignedMessage, err := odysseyWarp.NewUnsignedMessage(b.vm.ctx.NetworkID, b.vm.ctx.ChainID, blockHashPayload.Bytes())
		if err != nil {
			return fmt.Errorf("failed to create unsigned message for block hash payload: %w", err)
		}
		if err := b.vm.warpBackend.AddMessage(unsignedMessage); err != nil {
			return fmt.Errorf("failed to add block hash payload unsigned message: %w", err)
		}
	}

	return nil
}

// Reject implements the snowman.Block interface
func (b *Block) Reject(context.Context) error {
	b.status = choices.Rejected
	log.Debug(fmt.Sprintf("Rejecting block %s (%s) at height %d", b.ID().Hex(), b.ID(), b.Height()))
	return b.vm.blockChain.Reject(b.ethBlock)
}

// SetStatus implements the InternalBlock interface allowing ChainState
// to set the status on an existing block
func (b *Block) SetStatus(status choices.Status) { b.status = status }

// Status implements the snowman.Block interface
func (b *Block) Status() choices.Status {
	return b.status
}

// Parent implements the snowman.Block interface
func (b *Block) Parent() ids.ID {
	return ids.ID(b.ethBlock.ParentHash())
}

// Height implements the snowman.Block interface
func (b *Block) Height() uint64 {
	return b.ethBlock.NumberU64()
}

// Timestamp implements the snowman.Block interface
func (b *Block) Timestamp() time.Time {
	return time.Unix(int64(b.ethBlock.Time()), 0)
}

// syntacticVerify verifies that a *Block is well-formed.
func (b *Block) syntacticVerify() error {
	if b == nil || b.ethBlock == nil {
		return errInvalidBlock
	}

	header := b.ethBlock.Header()
	rules := b.vm.chainConfig.OdysseyRules(header.Number, header.Time)
	return b.vm.syntacticBlockValidator.SyntacticVerify(b, rules)
}

// Verify implements the snowman.Block interface
func (b *Block) Verify(context.Context) error {
	return b.verify(&precompileconfig.PredicateContext{
		SnowCtx:            b.vm.ctx,
		ProposerVMBlockCtx: nil,
	}, true)
}

// ShouldVerifyWithContext implements the block.WithVerifyContext interface
func (b *Block) ShouldVerifyWithContext(context.Context) (bool, error) {
	predicates := b.vm.chainConfig.OdysseyRules(b.ethBlock.Number(), b.ethBlock.Timestamp()).Predicates
	// Short circuit early if there are no predicates to verify
	if len(predicates) == 0 {
		return false, nil
	}

	// Check if any of the transactions in the block specify a precompile that enforces a predicate, which requires
	// the ProposerVMBlockCtx.
	for _, tx := range b.ethBlock.Transactions() {
		for _, accessTuple := range tx.AccessList() {
			if _, ok := predicates[accessTuple.Address]; ok {
				log.Debug("Block verification requires proposerVM context", "block", b.ID(), "height", b.Height())
				return true, nil
			}
		}
	}

	log.Debug("Block verification does not require proposerVM context", "block", b.ID(), "height", b.Height())
	return false, nil
}

// VerifyWithContext implements the block.WithVerifyContext interface
func (b *Block) VerifyWithContext(ctx context.Context, proposerVMBlockCtx *block.Context) error {
	return b.verify(&precompileconfig.PredicateContext{
		SnowCtx:            b.vm.ctx,
		ProposerVMBlockCtx: proposerVMBlockCtx,
	}, true)
}

// Verify the block is valid.
// Enforces that the predicates are valid within [predicateContext].
// Writes the block details to disk and the state to the trie manager iff writes=true.
func (b *Block) verify(predicateContext *precompileconfig.PredicateContext, writes bool) error {
	if predicateContext.ProposerVMBlockCtx != nil {
		log.Debug("Verifying block with context", "block", b.ID(), "height", b.Height())
	} else {
		log.Debug("Verifying block without context", "block", b.ID(), "height", b.Height())
	}
	if err := b.syntacticVerify(); err != nil {
		return fmt.Errorf("syntactic block verification failed: %w", err)
	}

	// Only enforce predicates if the chain has already bootstrapped.
	// If the chain is still bootstrapping, we can assume that all blocks we are verifying have
	// been accepted by the network (so the predicate was validated by the network when the
	// block was originally verified).
	if b.vm.bootstrapped {
		if err := b.verifyPredicates(predicateContext); err != nil {
			return fmt.Errorf("failed to verify predicates: %w", err)
		}
	}

	// The engine may call VerifyWithContext multiple times on the same block with different contexts.
	// Since the engine will only call Accept/Reject once, we should only call InsertBlockManual once.
	// Additionally, if a block is already in processing, then it has already passed verification and
	// at this point we have checked the predicates are still valid in the different context so we
	// can return nil.
	if b.vm.State.IsProcessing(b.id) {
		return nil
	}

	return b.vm.blockChain.InsertBlockManual(b.ethBlock, writes)
}

// verifyPredicates verifies the predicates in the block are valid according to predicateContext.
func (b *Block) verifyPredicates(predicateContext *precompileconfig.PredicateContext) error {
	rules := b.vm.chainConfig.OdysseyRules(b.ethBlock.Number(), b.ethBlock.Timestamp())

	switch {
	case !rules.IsDUpgrade && rules.PredicatesExist():
		return errors.New("cannot enable predicates before DUpgrade activation")
	case !rules.IsDUpgrade:
		return nil
	}

	predicateResults := results.NewPredicateResults()
	for _, tx := range b.ethBlock.Transactions() {
		results, err := core.CheckPredicates(rules, predicateContext, tx)
		if err != nil {
			return err
		}
		predicateResults.SetTxPredicateResults(tx.Hash(), results)
	}
	// TODO: document required gas constraints to ensure marshalling predicate results does not error
	predicateResultsBytes, err := predicateResults.Bytes()
	if err != nil {
		return fmt.Errorf("failed to marshal predicate results: %w", err)
	}
	extraData := b.ethBlock.Extra()
	if len(extraData) < params.DynamicFeeExtraDataSize {
		return fmt.Errorf("header extra data too short for predicate verification found: %d, required: %d", len(extraData), params.DynamicFeeExtraDataSize)
	}
	headerPredicateResultsBytes := extraData[params.DynamicFeeExtraDataSize:]
	if !bytes.Equal(headerPredicateResultsBytes, predicateResultsBytes) {
		parsedResults, err := results.ParsePredicateResults(headerPredicateResultsBytes)
		if err != nil {
			return err
		}
		return fmt.Errorf("%w (remote: %x %v local: %x %v)", errInvalidHeaderPredicateResults, headerPredicateResultsBytes, parsedResults, predicateResultsBytes, predicateResults)
	}
	return nil
}

// Bytes implements the snowman.Block interface
func (b *Block) Bytes() []byte {
	res, err := rlp.EncodeToBytes(b.ethBlock)
	if err != nil {
		panic(err)
	}
	return res
}

func (b *Block) String() string { return fmt.Sprintf("EVM block, ID = %s", b.ID()) }
