package statemachine

import (
	"blockchain/trie"
	"blockchain/types"
)

type IMachine interface {
	Execute(state trie.ITrie, tx types.Transaction) *types.Receiption
}

type StateMachine struct {
}

func NewStateMachine() *StateMachine {
	return &StateMachine{}
}
func (m StateMachine) Execute(state *trie.State, tx *types.Transaction) (*types.Receiption, uint64) {
	from := tx.From()
	to := tx.To()
	value := tx.Value()
	gasUsed := tx.Gas
	if gasUsed > 21000 {
		gasUsed = 21000
	}

	gasUsed = gasUsed * tx.GasPrice()
	cost := value + gasUsed
	account, err := state.Load(from)
	if err != nil {
		return nil, 0
	}
	if account.Amount < cost {
		return nil, 0
	}

	account.Nonce = account.Nonce + 1
	account.Amount = account.Amount - cost
	state.Store(from, account)

	toAccount, err := state.Load(to)
	if err != nil {
		toAccount = types.Account{}
	}

	toAccount.Amount = toAccount.Amount + value
	state.Store(to, toAccount)
	receiption := &types.Receiption{
		TxHash: tx.Hash(),
		Status: 0,
	}
	return receiption, gasUsed
}
