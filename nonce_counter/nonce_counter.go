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
	"golang.org/x/sync/semaphore"
)

// NonceCounter manages nonces for specific blockchain addresses by tracking contract events in a thread-safe manner.
type NonceCounter struct {
	contractAddress string
	eventName       string
	addresses       []string
	contractAbi     abi.ABI
	addressToNonce  map[string]uint64
	blockBatchSize  int64
	mu              sync.Mutex
	concurrency     int64
}

// Config represents the configuration required for initializing and managing a nonce counter.
type Config struct {
	Concurrency     int64
	ContractAddress string
	ContractABI     string
	StartBlock      int64
	EventName       string
	Addresses       []string
	BlockBatchSize  int64
}

// Validate checks the Config fields for validity and returns an error if any required field is invalid or missing.
func (ncc Config) Validate() error {
	if ncc.Concurrency <= 0 {
		return fmt.Errorf("concurrency must be greater than 0")
	}
	if ncc.ContractAddress == "" {
		return fmt.Errorf("contract address must be provided")
	}
	if ncc.ContractABI == "" {
		return fmt.Errorf("contract ABI must be provided")
	}
	if ncc.StartBlock < 0 {
		return fmt.Errorf("start block must be greater than or equal to 0")
	}
	if ncc.EventName == "" {
		return fmt.Errorf("event name must be provided")
	}
	if len(ncc.Addresses) == 0 {
		return fmt.Errorf("addresses must be provided")
	}
	if ncc.BlockBatchSize <= 0 {
		return fmt.Errorf("block batch size must be greater than 0")
	}

	return nil
}

// NewNonceCounter initializes a NonceCounter instance using the provided configuration.
// It validates the configuration and sets up the necessary internal state for nonce management.
func NewNonceCounter(config Config) (*NonceCounter, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	contractAbi, err := abi.JSON(strings.NewReader(config.ContractABI))
	if err != nil {
		log.Fatalf("failed to parse contract ABI: %v", err)
	}

	addressToNonce := make(map[string]uint64, len(config.Addresses))
	for _, address := range config.Addresses {
		addressToNonce[address] = 0
	}

	return &NonceCounter{
		contractAddress: config.ContractAddress,
		eventName:       config.EventName,
		contractAbi:     contractAbi,
		addresses:       config.Addresses,
		blockBatchSize:  config.BlockBatchSize,
		addressToNonce:  addressToNonce,
		concurrency:     config.Concurrency,
		mu:              sync.Mutex{},
	}, nil
}

// Start begins tracking and processing blockchain events from a specified start block using the provided RPC URL and context.
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
				// On production code, the error should be handled properly and the retry and an exponential backoff should be implemented
				time.Sleep(5 * time.Second)
				break
			}

			query := nc.prepareQuery(header, currentBlock)
			fmt.Printf("Block Range %d-%d\n", query.FromBlock.Int64(), query.ToBlock.Int64())
			logs, err := client.FilterLogs(context.Background(), query)
			if err != nil {
				log.Printf("error fetching logs for block %d: %v", currentBlock.Int64(), err)
				// On production code, the error should be handled properly and the retry and an exponential backoff should be implemented
				time.Sleep(5 * time.Second)
				break
			}

			if foundAddress := nc.FindNonces(ctx, logs); foundAddress {
				nc.printNonces()
			}

			// Move to the next block range
			currentBlock.Add(query.ToBlock, big.NewInt(1))
		}
	}
}

// FindNonces processes blockchain logs to identify relevant events, increment
// nonces for tracked addresses, and returns success.
func (nc *NonceCounter) FindNonces(ctx context.Context, logs []types.Log) bool {
	foundAddress := false

	sem := semaphore.NewWeighted(nc.concurrency)
	var wg sync.WaitGroup

	for _, vLog := range logs {
		wg.Add(1)

		if err := sem.Acquire(ctx, 1); err != nil {
			log.Printf("failed to acquire semaphore: %v\n", err)
			wg.Done()
			continue
		}

		go func(vLog types.Log) {
			defer wg.Done()
			defer sem.Release(1)

			event := &ValidatorAddedEvent{}
			if err := event.Parse(nc.eventName, nc.contractAbi, vLog); err != nil {
				// This should be handled properly in production code, for now just ignore it and move on
				return
			}

			// Process the event
			if incremented := nc.incrementNonce(*event); !incremented {
				return
			}

			if !foundAddress {
				foundAddress = true
			}
		}(vLog)
	}
	wg.Wait()

	return foundAddress
}

// prepareQuery constructs and returns an Ethereum FilterQuery to fetch logs within a specific block range and address list.
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

// incrementNonce increments the nonce for a specific address if it exists and returns whether a change was made.
func (nc *NonceCounter) incrementNonce(vae ValidatorAddedEvent) bool {
	if contains := slices.Contains(nc.addresses, vae.Owner.Hex()); !contains {
		return false
	}

	nc.mu.Lock()
	defer nc.mu.Unlock()

	nc.addressToNonce[vae.Owner.Hex()]++
	return true
}

// printNonces prints the current state of address-to-nonce mappings to the console.
func (nc *NonceCounter) printNonces() {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	fmt.Println("-----------------------------------------")
	fmt.Println("Nonce for address modified, current state:")
	for address, nonce := range nc.addressToNonce {
		fmt.Printf("Address: %s, Nonce: %d\n", address, nonce)
	}
	fmt.Println("-----------------------------------------")
}
