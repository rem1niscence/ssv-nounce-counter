package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	noncecounter "github.com/rem1niscence/ssv-nounce-counter/nonce_counter"
)

const (
	rpcURL          = "https://ethereum-holesky-rpc.publicnode.com"
	contractAddress = "0x38A4794cCEd47d3baf7370CcC43B560D3a1beEFA"
	eventName       = "ValidatorAdded"
	startBlock      = 181612
	blockBatchSize  = 50000
)

var (
	addresses = []string{
		"0xfc4b7d410Aa23bab793Ea7694D182f5c93f32aB2",
		"0x9a8e8762CE71B669250e964d5262C390416aB3BA",
		"0x350e4F967A62714492Ce180f4035036Dd193B733",
		"0x83110aa1EC834f93f779Fb89e93550140f5397A7",
		"0xAcc3139dd26197669012930C9DAAcECbe260c856",
	}
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ncCounter, err := noncecounter.NewNonceCounter(contractAddress, contractABIJSON, eventName, startBlock, blockBatchSize, addresses)
	if err != nil {
		panic(fmt.Sprintf("failed to create nonce counter: %v", err))
	}

	ncCounter.Start(ctx, startBlock, rpcURL)
}
