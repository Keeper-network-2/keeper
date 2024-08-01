package types

import (
    "github.com/ethereum/go-ethereum/common"
)

type TaskResponse struct {
    JobID        uint32
    TaskID       uint32
    ChainID      uint
    ContractAddr common.Address
    Result       []byte
}