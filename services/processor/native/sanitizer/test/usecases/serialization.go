// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package usecases

const Serialization = `package main

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
	"encoding/json"
	"encoding/hex"
)

var PUBLIC = sdk.Export(add, getJSON, getHex)
var SYSTEM = sdk.Export(_init)

var COUNTER_KEY = []byte("count")

func _init() {
	state.WriteUint64(COUNTER_KEY, 0)
}

func add(amount uint64) {
	count := state.ReadUint64(COUNTER_KEY)
	count += amount
	state.WriteUint64(COUNTER_KEY, count)
}

func getJSON() []byte {
	data, _ := json.Marshal(state.ReadUint64(COUNTER_KEY))
	return data
}

func getHex() string {
	return hex.EncodeToString(state.ReadUint64(COUNTER_KEY))
}
`
