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
	"github.com/sirupsen/logrus"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"time"
)

// SimpleAsset implements a simple chaincode to manage an asset
type SimpleAsset struct {
	token map[string]uint64
}

// Init is called during chaincode instantiation to initialize any
// data. Note that chaincode upgrade also calls this function to reset
// or to migrate data.
func (t *SimpleAsset) Init(shim.ChaincodeStubInterface) peer.Response {
	t.token = map[string]uint64{}

	return shim.Success(nil)
}

// Invoke is called per transaction on the chaincode. Each transaction is
// either a 'get' or a 'set' on the asset created by Init function. The Set
// method may create a new asset by specifying a new key-value pair.
func (t *SimpleAsset) Invoke(stub shim.ChaincodeStubInterface) peer.Response {

	if t.token == nil {
		t.token = map[string]uint64{}
	}
	// Extract the function and args from the transaction proposal
	fn, args := stub.GetFunctionAndParameters()

	var result string
	var err error
	v, ok := t.token[args[0]]

	if !ok {
		vv, err := stub.GetState(args[0])
		if err != nil {
			return shim.Error(fmt.Sprintf("user is not found, %v", err))
		}
		if len(vv) == 0 {
			t.token[args[0]] = 0
		} else {
			v, err = strconv.ParseUint(string(vv), 10, 64)
			if err != nil {
				logger.Error(err)
				return shim.Error(err.Error())
			}
			t.token[args[0]] = v
		}

	}

	if fn == "add" {
		result, err = add(stub, args[0], v)
		if err != nil {
			return shim.Error(err.Error())
		}
	} else if fn == "random" {
		result, err = random(stub, args[0], v)
		if err != nil {
			return shim.Error(err.Error())
		}
	}
	// Return the result as success payload
	return shim.Success([]byte(result))
}

func add(stub shim.ChaincodeStubInterface, userID string, token uint64) (string, error) {

	res, err := addRand(stub, userID, token)
	if err != nil {
		return "", err
	}
	err = stub.PutState(userID, []byte(res))
	if err != nil {
		return "", fmt.Errorf("failed to set asset %d for %s", token, userID)
	}

	return fmt.Sprintf("%v", token), nil
}

func random(stub shim.ChaincodeStubInterface, userID string, token uint64) (string, error) {

	rand.Seed(time.Now().Unix())

	return fmt.Sprintf("%v", rand.Uint64()), nil
}

func addRand(stub shim.ChaincodeStubInterface, userID string, token uint64) (string, error) {
	var res []uint64
	for i := 1; i < 3; i++ {
		cc := fmt.Sprintf("addc%d", i)
		resp := stub.InvokeChaincode(cc, [][]byte{[]byte("random"), []byte(userID)}, "mychannel")
		if resp.Status != http.StatusOK {
			logger.Error("failed to invoke chaincode [%d], %v", cc, resp.Message)
			continue
		}
		v, err := strconv.ParseUint(string(resp.Payload), 10, 64)
		if err != nil {
			logger.Error("failed to convert from ", cc, err)
			continue
		}
		res = append(res, v)
	}

	sort.Slice(res, func(i, j int) bool {
		return i > j
	})

	return fmt.Sprintf("%v", res[0]), nil
}

var logger = logrus.New()

func toUint(str string) uint64 {
	if str == "" {
		return 0
	}

	parseUint, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		logger.Error(err)
		return 0
	}
	return parseUint
}

// main function starts up the chaincode in the container during instantiate
func main() {
	if err := shim.Start(new(SimpleAsset)); err != nil {
		fmt.Printf("Error starting SimpleAsset chaincode: %s", err)
	}
}
