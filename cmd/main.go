package main

import (
	"log"

	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	rpcURL          = "https://ethereum-holesky-rpc.publicnode.com"
	contractAddress = "0x38A4794cCEd47d3baf7370CcC43B560D3a1beEFA"
)

func main() {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		log.Fatalf("failed to connect to Ethereum RPC: %v", err)
	}
	defer client.Close()
}
