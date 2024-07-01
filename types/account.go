package types

import (
	"blockchain/utils/hash"
	"blockchain/utils/rlp"
)

type Account struct {
	Amount   uint64
	Nonce    uint64
	CodeHash hash.Hash
	Root     hash.Hash
}

func (account Account) Bytes() []byte {
	data, _ := rlp.EncodeToBytes(account)
	return data
}

func AccountFromBytes(data []byte) *Account {
	var account Account

	err := rlp.DecodeBytes(data, &account)
	if err != nil {
		return nil
	}
	return &account
}
