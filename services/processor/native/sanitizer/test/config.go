// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import "github.com/orbs-network/orbs-network-go/services/processor/native/sanitizer"

func SanitizerConfigForTests() *sanitizer.SanitizerConfig {
	return &sanitizer.SanitizerConfig{
		ImportWhitelist: map[string]string{
			`"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"`:       "SDK",
			`"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"`: "SDK",
			`"encoding/json"`: "Encoding",
			`"time"`:          "test",
		},
		FunctionBlacklist: map[string][]string{
			`time`: {"Sleep", "After", "AfterFunc"},
		},
	}
}

func SanitizerConfigWithWildcardForTests() *sanitizer.SanitizerConfig {
	return &sanitizer.SanitizerConfig{
		ImportWhitelist: map[string]string{
			`"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/*"`: "SDK",
			`"encoding/json"`: "Encoding",
			`"time"`:          "test",
		},
		FunctionBlacklist: map[string][]string{
			`time`: {"Sleep", "After", "AfterFunc"},
		},
	}
}
