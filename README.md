# Subnet EVM

Odyssey is a network composed of multiple blockchains.
Each blockchain is an instance of a Virtual Machine (VM), much like an object in an object-oriented language is an instance of a class.
That is, the VM defines the behavior of the blockchain.

Subnet EVM is the Virtual Machine (VM) that defines the Subnet Contract Chains. Subnet EVM is a simplified version of [Coreth VM (D-Chain)](https://github.com/DioneProtocol/coreth).

This chain implements the Ethereum Virtual Machine and supports Solidity smart contracts as well as most other Ethereum client functionality.

## Building

The Subnet EVM runs in a separate process from the main OdysseyGo process and communicates with it over a local gRPC connection.

### OdysseyGo Compatibility

```text
[v0.0.1] OdysseyGo@v0.0.1 (Protocol Version: 28)
```

## API

The Subnet EVM supports the following API namespaces:

- `eth`
- `personal`
- `txpool`
- `debug`

Only the `eth` namespace is enabled by default.

## Compatibility

The Subnet EVM is compatible with almost all Ethereum tooling, including Remix, Metamask and Truffle.

## Differences Between Subnet EVM and Coreth

- Added configurable fees and gas limits in genesis
- Merged Odyssey hardforks into the single "Subnet EVM" hardfork
- Removed Atomic Txs and Shared Memory
- Removed Multicoin Contract and State

## Block Format

To support these changes, there have been a number of changes to the SubnetEVM block format compared to what exists on the D-Chain and Ethereum. Here we list the changes to the block format as compared to Ethereum.

### Block Header

- `BaseFee`: Added by EIP-1559 to represent the base fee of the block (present in Ethereum as of EIP-1559)
- `BlockGasCost`: surcharge for producing a block faster than the target rate

## Create an EVM Subnet on a Local Network

### Clone Subnet-evm

First install Go 1.20.8 or later. Follow the instructions [here](https://golang.org/doc/install). You can verify by running `go version`.

Set `$GOPATH` environment variable properly for Go to look for Go Workspaces. Please read [this](https://go.dev/doc/gopath_code) for details. You can verify by running `echo $GOPATH`.

As a few software will be installed into `$GOPATH/bin`, please make sure that `$GOPATH/bin` is in your `$PATH`, otherwise, you may get error running the commands below.

Download the `subnet-evm` repository into your `$GOPATH`:

```sh
cd $GOPATH
mkdir -p src/github.com/DioneProtocol
cd src/github.com/DioneProtocol
git clone git@github.com:DioneProtocol/subnet-evm.git
cd subnet-evm
```

This will clone and checkout to `master` branch.

### Run Local Network

To run a local network, it is recommended to use the [odyssey-cli](https://github.com/DioneProtocol/odyssey-cli#odyssey-cli) to set up an instance of Subnet-EVM on an local Odyssey Network.

There are two options when using the Odyssey-CLI:

1. Use an official Subnet-EVM release.
2. Build and deploy a locally built (and optionally modified) version of Subnet-EVM.
