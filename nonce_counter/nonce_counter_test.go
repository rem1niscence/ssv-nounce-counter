package noncecounter

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestNonceCounterIncrementNonce(t *testing.T) {
	tests := []struct {
		name          string
		initialNonces map[string]uint64
		vcAddresses   []string
		event         ValidatorAddedEvent
		wantUpdated   bool
		wantNonce     uint64
	}{
		{
			name: "valid address, increment nonce",
			initialNonces: map[string]uint64{
				"0xabCDEF1234567890ABcDEF1234567890aBCDeF12": 5,
			},
			vcAddresses: []string{"0xabCDEF1234567890ABcDEF1234567890aBCDeF12"},
			event: ValidatorAddedEvent{
				Owner: common.HexToAddress("0xabCDEF1234567890ABcDEF1234567890aBCDeF12"),
			},
			wantUpdated: true,
			wantNonce:   6,
		},
		{
			name: "address not in validator list, no increment",
			initialNonces: map[string]uint64{
				"0X1234567890ABCDEF1234567890ABCDEF12345678": 3,
			},
			vcAddresses: []string{"0X1234567890ABCDEF1234567890ABCDEF12345678"},
			event: ValidatorAddedEvent{
				Owner: common.HexToAddress("0X1234567890ABCDEF1234567890ABCDEF12345678"),
			},
			wantUpdated: false,
			wantNonce:   3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup NonceCounter
			nc := &NonceCounter{
				addressToNonce: map[string]uint64{},
				addresses:      tt.vcAddresses,
			}

			// Initialize the NonceCounter state
			for addr, nonce := range tt.initialNonces {
				nc.addressToNonce[addr] = nonce
			}

			// Execute IncrementNonce
			got := nc.incrementNonce(tt.event)

			// Validate result
			if got != tt.wantUpdated {
				t.Errorf("IncrementNonce() = %v, want %v", got, tt.wantUpdated)
			}
			if nonce, exists := nc.addressToNonce[tt.event.Owner.Hex()]; exists {
				if nonce != tt.wantNonce {
					t.Errorf("nonce for address %s = %d, want %d", tt.event.Owner.Hex(), nonce, tt.wantNonce)
				}
			} else if tt.wantUpdated {
				t.Errorf("expected address %s to exist in map", tt.event.Owner.Hex())
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Concurrency:     10,
				ContractAddress: "0x1234567890abcdef1234567890abcdef12345678",
				ContractABI:     `[]`,
				StartBlock:      0,
				EventName:       "Transfer",
				Addresses:       []string{"0xabcdef1234567890abcdef1234567890abcdef12"},
				BlockBatchSize:  100,
			},
			wantErr: false,
		},
		{
			name: "invalid concurrency",
			config: Config{
				Concurrency:     0,
				ContractAddress: "0x1234567890abcdef1234567890abcdef12345678",
				ContractABI:     `[]`,
				StartBlock:      0,
				EventName:       "Transfer",
				Addresses:       []string{"0xabcdef1234567890abcdef1234567890abcdef12"},
				BlockBatchSize:  100,
			},
			wantErr: true,
		},
		{
			name: "missing contract address",
			config: Config{
				Concurrency:     10,
				ContractAddress: "",
				ContractABI:     `[]`,
				StartBlock:      0,
				EventName:       "Transfer",
				Addresses:       []string{"0xabcdef1234567890abcdef1234567890abcdef12"},
				BlockBatchSize:  100,
			},
			wantErr: true,
		},
		{
			name: "missing contract ABI",
			config: Config{
				Concurrency:     10,
				ContractAddress: "0x1234567890abcdef1234567890abcdef12345678",
				ContractABI:     "",
				StartBlock:      0,
				EventName:       "Transfer",
				Addresses:       []string{"0xabcdef1234567890abcdef1234567890abcdef12"},
				BlockBatchSize:  100,
			},
			wantErr: true,
		},
		{
			name: "invalid start block",
			config: Config{
				Concurrency:     10,
				ContractAddress: "0x1234567890abcdef1234567890abcdef12345678",
				ContractABI:     `[]`,
				StartBlock:      -1,
				EventName:       "Transfer",
				Addresses:       []string{"0xabcdef1234567890abcdef1234567890abcdef12"},
				BlockBatchSize:  100,
			},
			wantErr: true,
		},
		{
			name: "missing event name",
			config: Config{
				Concurrency:     10,
				ContractAddress: "0x1234567890abcdef1234567890abcdef12345678",
				ContractABI:     `[]`,
				StartBlock:      0,
				EventName:       "",
				Addresses:       []string{"0xabcdef1234567890abcdef1234567890abcdef12"},
				BlockBatchSize:  100,
			},
			wantErr: true,
		},
		{
			name: "empty addresses",
			config: Config{
				Concurrency:     10,
				ContractAddress: "0x1234567890abcdef1234567890abcdef12345678",
				ContractABI:     `[]`,
				StartBlock:      0,
				EventName:       "Transfer",
				Addresses:       []string{},
				BlockBatchSize:  100,
			},
			wantErr: true,
		},
		{
			name: "invalid block batch size",
			config: Config{
				Concurrency:     10,
				ContractAddress: "0x1234567890abcdef1234567890abcdef12345678",
				ContractABI:     `[]`,
				StartBlock:      0,
				EventName:       "Transfer",
				Addresses:       []string{"0xabcdef1234567890abcdef1234567890abcdef12"},
				BlockBatchSize:  0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
