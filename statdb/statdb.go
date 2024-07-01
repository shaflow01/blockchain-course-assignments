package statdb

import (
	"blockchain/types"
	"blockchain/utils/hash"
)

type StatDB interface {
	Load(address types.Address) (types.Account, error)
	Store(address types.Address, account types.Account)
	SetStatRoot(root hash.Hash)
}
