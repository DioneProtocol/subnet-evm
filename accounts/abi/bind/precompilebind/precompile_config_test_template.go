// (c) 2019-2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package precompilebind

// tmplSourcePrecompileConfigGo is the Go precompiled config source template.
const tmplSourcePrecompileConfigTestGo = `
// Code generated
// This file is a generated precompile config test with the skeleton of test functions.
// The file is generated by a template. Please inspect every code and comment in this file before use.

package {{.Package}}

import (
	"math/big"
	"testing"

	"github.com/ava-labs/subnet-evm/precompile/precompileconfig"
	"github.com/ava-labs/subnet-evm/precompile/testutils"
	{{- if .Contract.AllowList}}
	"github.com/ava-labs/subnet-evm/precompile/allowlist"

	"github.com/ethereum/go-ethereum/common"
	{{- end}}
)

// TestVerify tests the verification of Config.
func TestVerify(t *testing.T) {
	{{- if .Contract.AllowList}}
	admins := []common.Address{allowlist.TestAdminAddr}
	enableds := []common.Address{allowlist.TestEnabledAddr}
	{{- end}}
	tests := map[string]testutils.ConfigVerifyTest{
		"valid config": {
			Config: NewConfig(big.NewInt(3){{- if .Contract.AllowList}}, admins, enableds{{- end}}),
			ExpectedError: "",
		},
		// CUSTOM CODE STARTS HERE
		// Add your own Verify tests here, e.g.:
		// "your custom test name": {
		// 	Config: NewConfig(big.NewInt(3), {{- if .Contract.AllowList}} admins, enableds{{- end}}),
		// 	ExpectedError: ErrYourCustomError.Error(),
		// },
	}
	{{- if .Contract.AllowList}}
	// Verify the precompile with the allowlist.
	// This adds allowlist verify tests to your custom tests
	// and runs them all together.
	// Even if you don't add any custom tests, keep this. This will still
	// run the default allowlist verify tests.
	allowlist.VerifyPrecompileWithAllowListTests(t, Module, tests)
	{{- else}}
	// Run verify tests.
	testutils.RunVerifyTests(t, tests)
	{{- end}}
}

// TestEqual tests the equality of Config with other precompile configs.
func TestEqual(t *testing.T) {
	{{- if .Contract.AllowList}}
	admins := []common.Address{allowlist.TestAdminAddr}
	enableds := []common.Address{allowlist.TestEnabledAddr}
	{{- end}}
	tests := map[string]testutils.ConfigEqualTest{
		"non-nil config and nil other": {
			Config:   NewConfig(big.NewInt(3){{- if .Contract.AllowList}}, admins, enableds{{- end}}),
			Other:    nil,
			Expected: false,
		},
		"different type": {
			Config:   NewConfig(big.NewInt(3){{- if .Contract.AllowList}}, admins, enableds{{- end}}),
			Other:    precompileconfig.NewNoopStatefulPrecompileConfig(),
			Expected: false,
		},
		"different timestamp": {
			Config:   NewConfig(big.NewInt(3){{- if .Contract.AllowList}}, admins, enableds{{- end}}),
			Other:    NewConfig(big.NewInt(4){{- if .Contract.AllowList}}, admins, enableds{{- end}}),
			Expected: false,
		},
		"same config": {
			Config: NewConfig(big.NewInt(3){{- if .Contract.AllowList}}, admins, enableds{{- end}}),
			Other: NewConfig(big.NewInt(3){{- if .Contract.AllowList}}, admins, enableds{{- end}}),
			Expected: true,
		},
		// CUSTOM CODE STARTS HERE
		// Add your own Equal tests here
		}
		{{- if .Contract.AllowList}}
		// Run allow list equal tests.
		// This adds allowlist equal tests to your custom tests
		// and runs them all together.
		// Even if you don't add any custom tests, keep this. This will still
		// run the default allowlist equal tests.
		allowlist.EqualPrecompileWithAllowListTests(t, Module, tests)
		{{- else}}
		// Run equal tests.
		testutils.RunEqualTests(t, tests)
		{{- end}}
	}
`
