# Odyssey Warp Messaging

> **Warning**
> Odyssey Warp Messaging is currently in experimental mode to be used only on ephemeral test networks.
>
> Breaking changes to Odyssey Warp Messaging integration into Subnet-EVM may still be made.

Odyssey Warp Messaging offers a basic primitive to enable Cross-Subnet communication on the Odyssey Network.

It is intended to allow communication between arbitrary Custom Virtual Machines (including, but not limited to Subnet-EVM).

## How does Odyssey Warp Messaging Work

Odyssey Warp Messaging uses BLS Multi-Signatures with Public-Key Aggregation where every Odyssey validator registers a public key alongside its NodeID on the Odyssey O-Chain.

Every node tracking a Subnet has read access to the Odyssey O-Chain. This provides weighted sets of BLS Public Keys that correspond to the validator sets of each Subnet on the Odyssey Network. Odyssey Warp Messaging provides a basic primitive for signing and verifying messages between Subnets: the receiving network can verify whether an aggregation of signatures from a set of source Subnet validators represents a threshold of stake large enough for the receiving network to process the message.

## Integrating Odyssey Warp Messaging into the EVM

### Flow of Sending / Receiving a Warp Message within the EVM

The Odyssey Warp Precompile enables this flow to send a message from blockchain A to blockchain B:

1. Call the Warp Precompile `sendWarpMessage` function with the arguments for the `UnsignedMessage`
2. Warp Precompile emits an event / log containing the `UnsignedMessage` specified by the caller of `sendWarpMessage`
3. Network accepts the block containing the `UnsignedMessage` in the log, so that validators are willing to sign the message
4. An off-chain relayer queries the validators for their signatures of the message and aggregate the signatures to create a `SignedMessage`
5. The off-chain relayer encodes the `SignedMessage` as the [predicate](#predicate-encoding) in the AccessList of a transaction to deliver on blockchain B
6. The transaction is delivered on blockchain B, the signature is verified prior to executing the block, and the message is accessible via the Warp Precompile's `getVerifiedWarpMessage` during the execution of that transaction

### Warp Precompile

The Warp Precompile is broken down into three functions defined in the Solidity interface file [here](../../../contracts/contracts/interfaces/IWarpMessenger.sol).

#### sendWarpMessage

`sendWarpMessage` is used to send a verifiable message. Calling this function results in sending a message with the following contents:

- `SourceChainID` - blockchainID of the sourceChain on the Odyssey O-Chain
- `SourceAddress` - `msg.sender` encoded as a 32 byte value that calls `sendWarpMessage`
- `DestinationChainID` - `bytes32` argument specifies the blockchainID on the Odyssey O-Chain that should receive the message
- `DestinationAddress` - 32 byte value that represents the destination address that should receive the message (on the EVM this is the 20 byte address left zero extended)
- `Payload` - `payload` argument specified in the call to `sendWarpMessage` emitted as the unindexed data of the resulting log

Calling this function will issue a `SendWarpMessage` event from the Warp Precompile. Since the EVM limits the number of topics to 4 including the EventID, this message includes only the topics that would be expected to help filter messages emitted from the Warp Precompile the most.

Specifically, the `payload` is not emitted as a topic because each topic must be encoded as a hash. It could include the warp `messageID` as a topic, but that would not add more information. Therefore, we opt to take advantage of each possible topic to maximize the possible filtering for emitted Warp Messages.

Additionally, the `SourceChainID` is excluded because anyone parsing the chain can be expected to already know the blockchainID. Therefore, the `SendWarpMessage` event includes the indexable attributes:

- `destinationChainID`
- `destinationAddress`
- `sender`

The actual `message` is the entire [Odyssey Warp Unsigned Message](https://github.com/DioneProtocol/odysseygo/blob/develop/vms/omegavm/warp/unsigned_message.go#L14) including the Subnet-EVM [Addressed Payload](../../../warp/payload/payload.go).


#### getVerifiedMessage

`getVerifiedMessage` is used to read the contents of the delivered Odyssey Warp Message into the expected format.

It returns the message if present and a boolean indicating if a message is present.

To use this function, the transaction must include the signed Odyssey Warp Message encoded in the [predicate](#predicate-encoding) of the transaction. Prior to executing a block, the VM iterates through transactions and pre-verifies all predicates. If a transaction's predicate is invalid, then it is considered invalid to include in the block and dropped.

This leads to the following advantages:

1. The EVM execution does not need to verify the Warp Message at runtime (no signature verification or external calls to the O-Chain)
2. The EVM can deterministically re-execute and re-verify blocks assuming the predicate was verified by the network (eg., in bootstrapping)

This pre-verification is performed using the ProposerVM Block header during [block verification](../../../plugin/evm/block.go#L220) and [block building](../../../miner/worker.go#L200).

Note: in order to support the notion of an `AnycastID` for the `DestinationChainID`, `getVerifiedMessage` and the predicate DO NOT require that the `DestinationChainID` matches the `blockchainID` currently running. Instead, callers of `getVerifiedMessage` should use `getBlockchainID()` to decide how they should interpret the message. In other words, does the `destinationChainID` match either the local `blockchainID` or the `AnycastID`.

#### getBlockchainID

`getBlockchainID` returns the blockchainID of the blockchain that Subnet-EVM is running on.

This is different from the conventional Ethereum ChainID registered to https://chainlist.org/.

The `blockchainID` in Odyssey refers to the txID that created the blockchain on the Odyssey O-Chain.


### Predicate Encoding

Odyssey Warp Messages are encoded as a signed Odyssey [Warp Message](https://github.com/DioneProtocol/odysseygo/blob/develop/vms/omegavm/warp/message.go#L7) where the [UnsignedMessage](https://github.com/DioneProtocol/odysseygo/blob/develop/vms/omegavm/warp/unsigned_message.go#L14)'s payload includes an [AddressedPayload](../../../warp/payload/payload.go).

Since the predicate is encoded into the [Transaction Access List](https://eips.ethereum.org/EIPS/eip-2930), it is packed into 32 byte hashes intended to declare storage slots that should be pre-warmed into the cache prior to transaction execution.

Therefore, we use the [Predicate Utils](../../../utils/predicate/README.md) package to encode the actual byte slice of size N into the access list.

### Performance Optimization: D-Chain to Subnet

To support D-Chain to Subnet communication, or more generally Primary Network to Subnet communication, we special case the D-Chain for two reasons:

1. Every Subnet validator validates the D-Chain
2. The Primary Network has the largest possible number of validators

Since the Primary Network has the largest possible number of validators for any Subnet on Odyssey, it would also be the most expensive Subnet to receive and verify Odyssey Warp Messages from as it reaching a threshold of stake on the primary network would require many signatures. Luckily, we can do something much smarter.

When a Subnet receives a message from a blockchain on the Primary Network, we use the validator set of the receiving Subnet instead of the entire network when validating the message. This means that the D-Chain sending a message can be the exact same as Subnet to Subnet communication.

However, when Subnet B receives a message from the D-Chain, it changes the semantics to the following:

1. Read the SourceChainID of the signed message (D-Chain)
2. Look up the SubnetID that validates D-Chain: Primary Network
3. Look up the validator set of Subnet B (instead of the Primary Network) and the registered BLS Public Keys of Subnet B at the O-Chain height specified by the ProposerVM header
4. Continue Warp Message verification using the validator set of Subnet B instead of the Primary Network

This means that D-Chain to Subnet communication only requires a threshold of stake on the receiving subnet to sign the message instead of a threshold of stake for the entire Primary Network.

This assumes that the security of Subnet B already depends on the validators of Subnet B to behave virtuously. Therefore, requiring a threshold of stake from the receiving Subnet's validator set instead of the whole Primary Network does not meaningfully change security of the receiving Subnet.

Note: this special case is ONLY applied during Warp Message verification. The message sent by the Primary Network will still contain the Odyssey D-Chain's blockchainID as the sourceChainID and signatures will be served by querying the D-Chain directly.

## Design Considerations

### Re-Processing Historical Blocks

Odyssey Warp Messaging depends on the Odyssey O-Chain state at the O-Chain height specified by the ProposerVM block header.

Verifying a message requires looking up the validator set of the source subnet on the O-Chain. To support this, Odyssey Warp Messaging uses the ProposerVM header, which includes the O-Chain height it was issued at as the canonical point to lookup the source subnet's validator set.

This means verifying the Warp Message and therefore the state transition on a block depends on state that is external to the blockchain itself: the O-Chain.

The Odyssey O-Chain tracks only its current state and reverse diff layers (reversing the changes from past blocks) in order to re-calculate the validator set at a historical height. This means calculating a very old validator set that is used to verify a Warp Message in an old block may become prohibitively expensive.

Therefore, we need a heuristic to ensure that the network can correctly re-process old blocks (note: re-processing old blocks is a requirement to perform bootstrapping and is used in some VMs including Subnet-EVM to serve or verify historical data).

As a result, we require that the block itself provides a deterministic hint which determines which Odyssey Warp Messages were considered valid/invalid during the block's execution. This ensures that we can always re-process blocks and use the hint to decide whether an Odyssey Warp Message should be treated as valid/invalid even after the O-Chain state that was used at the original execution time may no longer support fast lookups.

To provide that hint, we've explored two designs:

1. Include a predicate in the transaction to ensure any referenced message is valid
2. Append the results of checking whether a Warp Message is valid/invalid to the block data itself

The current implementation uses option (1).

The original reason for this was that the notion of predicates for precompiles was designed with Shared Memory in mind. In the case of shared memory, there is no canonical "O-Chain height" in the block which determines whether or not Odyssey Warp Messages are valid.

Instead, the VM interprets a shared memory import operation as valid as soon as the UTXO is available in shared memory. This means that if it were up to the block producer to staple the valid/invalid results of whether or not an attempted atomic operation should be treated as valid, a byzantine block producer could arbitrarily report that such atomic operations were invalid and cause a griefing attack to burn the gas of users that attempted to perform an import.

Therefore, a transaction specified predicate is required to implement the shared memory precompile to prevent such a griefing attack.

In contrast, Odyssey Warp Messages are validated within the context of an exact O-Chain height. Therefore, if a block producer attempted to lie about the validity of such a message, the network would interpret that block as invalid.

### Guarantees Offered by Warp Precompile vs. Built on Top

#### Guarantees Offered by Warp Precompile

The Warp Precompile was designed with the intention of minimizing the trusted computing base for Subnet-EVM. Therefore, it makes several tradeoffs which encourage users to use protocols built ON TOP of the Warp Precompile itself as opposed to directly using the Warp Precompile.

The Warp Precompile itself provides ONLY the following ability:

- Emit a verifiable message from (Address A, Blockchain A) to (Address B, Blockchain B) that can be verified by the destination chain

#### Explicitly Not Provided / Built on Top

The Warp Precompile itself does not provide any guarantees of:

- Eventual message delivery (may require re-send on blockchain A and additional assumptions about off-chain relayers and chain progress)
- Ordering of messages (requires ordering provided a layer above)
- Replay protection (requires replay protection provided a layer above)
