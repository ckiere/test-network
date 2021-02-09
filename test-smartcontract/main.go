package main

import (
    "github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func main() {
    dataContract := new(DataContract)

    cc, err := contractapi.NewChaincode(dataContract)

    if err != nil {
        panic(err.Error())
    }

    if err := cc.Start(); err != nil {
        panic(err.Error())
    }
}