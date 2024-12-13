package noncecounter

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"golang.org/x/exp/slices"
)

type NonceCounter struct {
	contractAddress string
	eventName       string
	addresses       []string
	contractAbi     abi.ABI
	addressToNonce  map[string]uint64
	blockBatchSize  int64
	mu              sync.Mutex
}

func NewNonceCounter(contractAddress, rawABI, eventName string, startBlock uint64, blockBatchSize int64, addresses []string) (*NonceCounter, error) {
	contractAbi, err := abi.JSON(strings.NewReader(rawABI))
	if err != nil {
		log.Fatalf("failed to parse contract ABI: %v", err)
	}

	addressToNonce := make(map[string]uint64, len(addresses))
	for _, address := range addresses {
		addressToNonce[address] = 0
	}

	return &NonceCounter{
		contractAddress: contractAddress,
		eventName:       eventName,
		contractAbi:     contractAbi,
		addresses:       addresses,
		blockBatchSize:  blockBatchSize,
		addressToNonce:  addressToNonce,
		mu:              sync.Mutex{},
	}, nil
}

func (nc *NonceCounter) Start(ctx context.Context, startBlock uint64, rpcURL string) error {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return err
	}
	defer client.Close()

	currentBlock := new(big.Int).Set(big.NewInt(int64(startBlock)))

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			// Query the latest block number
			header, err := client.HeaderByNumber(context.Background(), nil)
			if err != nil {
				log.Printf("failed to fetch block header: %v\n", err)
				// On production code, the error should be handled properly and the retry and a exponential backoff should be implemented
				time.Sleep(5 * time.Second)
				break
			}

			query := nc.prepareQuery(header, currentBlock)
			fmt.Printf("Block Range %d-%d\n", query.FromBlock.Int64(), query.ToBlock.Int64())
			logs, err := client.FilterLogs(context.Background(), query)
			if err != nil {
				log.Printf("error fetching logs for block %d: %v", currentBlock.Int64(), err)
				// On production code, the error should be handled properly and the retry and a exponential backoff should be implemented
				time.Sleep(5 * time.Second)
				break
			}

			foundAddress := false
			for _, vLog := range logs {
				event := &ValidatorAddedEvent{}
				if err := event.Parse(nc.eventName, nc.contractAbi, vLog); err != nil {
					// log.Printf("failed to parse log: %v", err)
					continue
				}

				// Process the event
				if incremented := nc.IncrementNonce(*event); !incremented {
					continue
				}

				if !foundAddress {
					foundAddress = true
				}
			}

			if foundAddress {
				nc.PrintNonces()
			}

			// Move to the next block range
			currentBlock.Add(query.ToBlock, big.NewInt(1))
		}
	}
}

func (nc *NonceCounter) prepareQuery(header *types.Header, currentBlock *big.Int) ethereum.FilterQuery {
	latestBlock := header.Number

	endBlock := new(big.Int).Add(currentBlock, big.NewInt(nc.blockBatchSize))
	if endBlock.Cmp(latestBlock) >= 0 {
		// Avoid going past the latest block
		endBlock = latestBlock
	}

	// Prevent invalid ranges where FromBlock > ToBlock
	if currentBlock.Cmp(endBlock) > 0 {
		// If currentBlock is past latestBlock, adjust to latestBlock
		currentBlock = endBlock
	}

	return ethereum.FilterQuery{
		FromBlock: currentBlock,
		ToBlock:   endBlock,
		Addresses: []common.Address{
			common.HexToAddress(nc.contractAddress),
		},
	}
}

func (nc *NonceCounter) IncrementNonce(vae ValidatorAddedEvent) bool {
	if contains := slices.Contains(nc.addresses, vae.Owner.Hex()); !contains {
		return false
	}

	nc.mu.Lock()
	defer nc.mu.Unlock()

	nc.addressToNonce[vae.Owner.Hex()]++
	return true
}

func (nc *NonceCounter) PrintNonces() {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	fmt.Println("-----------------------------------------")
	fmt.Println("Nonce for address modified, current state:")
	for address, nonce := range nc.addressToNonce {
		fmt.Printf("Address: %s, Nonce: %d\n", address, nonce)
	}
	fmt.Println("-----------------------------------------")
}
