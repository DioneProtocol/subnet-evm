// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"fmt"

	"github.com/DioneProtocol/odysseygo/version"
	"github.com/DioneProtocol/subnet-evm/plugin/evm"
	"github.com/DioneProtocol/subnet-evm/plugin/runner"
)

func main() {
	versionString := fmt.Sprintf("Subnet-EVM/%s [OdysseyGo=%s, rpcchainvm=%d]", evm.Version, version.Current, version.RPCChainVMProtocol)
	runner.Run(versionString)
}
