package txpool

import (
	"blockchain/types"
	"blockchain/utils/hash"
)

type TxPool interface {
	NewTx(tx *types.Transaction)
	PoP() *types.Transaction
	SetStatRoot(root hash.Hash)
	NotifyTxEvent(txs []*types.Transaction)
}
