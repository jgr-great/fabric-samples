/*
 * Copyright IBM Corp All Rights Reserved
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"fmt"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
	"strconv"
	"sync"
)

// SimpleAsset implements a simple chaincode to manage an asset
type SimpleAsset struct {
	lock  sync.Mutex
	nonce uint64
}

// Init is called during chaincode instantiation to initialize any
// data. Note that chaincode upgrade also calls this function to reset
// or to migrate data.
func (t *SimpleAsset) Init(stub shim.ChaincodeStubInterface) peer.Response {
	// Get the args from the transaction proposal
	args := stub.GetStringArgs()
	if len(args) != 2 {
		return shim.Error("Incorrect arguments. Expecting a key and a value")
	}
	err := stub.PutState("nonce", []byte(fmt.Sprintf("%d", 0)))
	if err != nil {
		return shim.Success([]byte(fmt.Sprintf("failed to init nonce")))
	}
	// Set up any variables or assets here by calling stub.PutState()
	t.nonce = 0
	// We store the key and the value on the ledger
	err = stub.PutState("nonce", []byte(fmt.Sprintf("%d", t.nonce)))
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to create asset: %s", args[0]))
	}
	return shim.Success(nil)
}

// Invoke is called per transaction on the chaincode. Each transaction is
// either a 'get' or a 'set' on the asset created by Init function. The Set
// method may create a new asset by specifying a new key-value pair.
func (t *SimpleAsset) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	// Extract the function and args from the transaction proposal
	fn, args := stub.GetFunctionAndParameters()

	var result string
	var err error
	if fn == "set" {
		nonceBytes, err := stub.GetState("nonce")
		if err != nil {
			return shim.Error(err.Error())
		}
		nonceStr := "0"
		if len(nonceBytes) != 0 {
			nonceStr = string(nonceBytes)
		}

		nonce, err := strconv.ParseUint(string(nonceStr), 10, 64)
		if err != nil {
			return shim.Error(err.Error())
		}
		t.lock.Lock()
		defer t.lock.Unlock()
		if t.nonce < nonce {
			return shim.Error("double spent")
		}
		t.nonce++
		result, err = set(stub, args, nonce)
	} else { // assume 'get' even if fn is nil

		result, err = get(stub, args)

	}
	if err != nil {
		return shim.Error(err.Error())
	}

	// Return the result as success payload
	return shim.Success([]byte(result))
}

// Set stores the asset (both key and value) on the ledger. If the key exists,
// it will override the value with the new one
func set(stub shim.ChaincodeStubInterface, args []string, nonce uint64) (string, error) {
	if len(args) != 2 {
		return "", fmt.Errorf("Incorrect arguments. Expecting a key and a value")
	}

	err := stub.PutState(args[0], []byte(args[1]))
	if err != nil {
		return "", fmt.Errorf("Failed to set asset: %s", args[0])
	}
	err = stub.PutState("nonce", []byte(fmt.Sprintf("%d", nonce)))
	if err != nil {
		return "", fmt.Errorf("Failed to set nonce: %s", args[0])
	}
	return args[1], nil
}

// Get returns the value of the specified asset key
func get(stub shim.ChaincodeStubInterface, args []string) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("Incorrect arguments. Expecting a key")
	}

	value, err := stub.GetState(args[0])
	if err != nil {
		return "", fmt.Errorf("Failed to get asset: %s with error: %s", args[0], err)
	}
	if value == nil {
		return "", fmt.Errorf("Asset not found: %s", args[0])
	}
	return string(value), nil
}

// main function starts up the chaincode in the container during instantiate
func main() {
	if err := shim.Start(new(SimpleAsset)); err != nil {
		fmt.Printf("Error starting SimpleAsset chaincode: %s", err)
	}
}
