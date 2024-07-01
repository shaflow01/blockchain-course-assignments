package blockchain

import (
	"blockchain/crypto/sha3"
	"blockchain/trie"
	"blockchain/txpool"
	"blockchain/types"
	"blockchain/utils/hash"
	"blockchain/utils/rlp"
)

type Header struct {
	Root       hash.Hash
	ParentHash hash.Hash
	Height     uint64
	Coinbase   types.Address
	Timestamp  uint64
	Nonce      uint64
	//TODO: Add difficulty
}

type Body struct {
	Transactions []types.Transaction
	Receiptions  []types.Receiption
}

func (header Header) Hash() hash.Hash {
	data, _ := rlp.EncodeToBytes(header)
	return sha3.Keccak256(data)
}

func NewHeader(parent Header) *Header {
	return &Header{
		Root:       parent.Root,
		ParentHash: parent.Hash(),
		Height:     parent.Height + 1,
	}
}

func NewBlockBody() *Body {
	return &Body{
		Transactions: make([]types.Transaction, 0),
		Receiptions:  make([]types.Receiption, 0),
	}
}

type Blockchain struct {
	CurrentHeader Header
	Statedb       *trie.State
	Txpool        *txpool.DefaultPool
}

func NewBlockchain(statedb *trie.State, txpool *txpool.DefaultPool) *Blockchain {
	return &Blockchain{
		CurrentHeader: Header{
			Root:       statedb.Root(),
			ParentHash: hash.Hash{},
			Height:     0,
			Coinbase:   types.Address{},
			Timestamp:  0,
			Nonce:      0,
		},
		Statedb: statedb,
		Txpool:  txpool,
	}
}
