package noncecounter

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type ValidatorAddedEvent struct {
	Owner       common.Address
	OperatorIds []uint64
	PublicKey   []byte
	Shares      []byte
	Cluster     struct {
		ValidatorCount  uint32
		NetworkFeeIndex uint64
		Index           uint64
		Active          bool
		Balance         *big.Int
	}
}

func (vae *ValidatorAddedEvent) Parse(eventName string, contractABI abi.ABI, vLog types.Log) error {
	// Decode event data
	err := contractABI.UnpackIntoInterface(vae, eventName, vLog.Data)
	if err != nil {
		return fmt.Errorf("failed to decode log: %v", err)
	}
	vae.Owner = common.HexToAddress(vLog.Topics[1].Hex())
	return nil
}
