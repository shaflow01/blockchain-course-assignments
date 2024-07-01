package txpool

import (
	"blockchain/trie"
	"blockchain/types"
	"blockchain/utils/hash"
	"fmt"
	"sort"
	"sync"
)

const (
	Red    = "\033[31m"
	Yellow = "\033[33m"
	Green  = "\033[32m"
	Reset  = "\033[0m"
)

var mutex sync.Mutex
var sharedData int

type SortedTxs interface {
	GasPrice() uint64
	Push(tx *types.Transaction)
	Replace(tx *types.Transaction)
	Pop() *types.Transaction
	Nonce() uint64
}

type DefaultPool struct {
	Stat *trie.State

	all      map[hash.Hash]bool
	txs      pendingTxs
	pendings map[types.Address]pendingTxs
	queue    map[types.Address]QueueSortedTxs
}

// 打印信息，测试用的
func (pool *DefaultPool) PrintfPool() {
	for i, txs := range pool.txs {
		fmt.Println(i)
		for j, tx := range *txs {
			fmt.Printf("%d", j)
			fmt.Println(" - tx:", tx)

		}
	}
	for _, txs := range pool.pendings {
		for _, tx := range txs {
			for _, t := range *tx {
				fmt.Println("pending:", t)
			}
		}
	}
	fmt.Println("QUEUE:", pool.queue)
}

type QueueSortedTxs []*types.Transaction

func NewDefaultPool(state *trie.State) *DefaultPool {
	return &DefaultPool{
		Stat:     state,
		all:      make(map[hash.Hash]bool),
		pendings: make(map[types.Address]pendingTxs),
		queue:    make(map[types.Address]QueueSortedTxs),
	}
}

func (q QueueSortedTxs) Len() int {
	return len(q)
}

func (q QueueSortedTxs) Less(i, j int) bool {
	return q[i].Nonce() < q[j].Nonce()
}
func (q QueueSortedTxs) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
}

type DefaultSortedTxs []*types.Transaction

func (sorted *DefaultSortedTxs) Push(tx *types.Transaction) {
	*sorted = append(*sorted, tx)
}
func (sorted DefaultSortedTxs) Replace(tx *types.Transaction) {

	for key, value := range sorted {
		if value.Nonce() == tx.Nonce() && tx.GasPrice() > value.GasPrice() {
			sorted[key] = tx
		}
	}
}
func (sorted *DefaultSortedTxs) Pop() *types.Transaction {
	if len(*sorted) > 1 {
		// 当数组长度大于1时的处理逻辑不变

		tx := (*sorted)[0]
		*sorted = (*sorted)[1:]
		return tx
	} else if len(*sorted) == 1 {

		// 当数组长度等于1时，先保存交易，然后将数组清空
		tx := (*sorted)[0]
		*sorted = (*sorted)[1:] // 由于len(*sorted) == 1，等价于*sorted = (*sorted)[1:1]，这会将数组清空
		return tx
	} else {

		// 当数组为空时，返回nil
		return nil
	}
}
func (sorted DefaultSortedTxs) Nonce() uint64 {
	if len(sorted) != 0 {
		return sorted[len(sorted)-1].Nonce()
	}
	return 0
}
func (sorted DefaultSortedTxs) GasPrice() uint64 {
	if len(sorted) != 0 {
		return sorted[0].GasPrice()
	}

	return 0
}

type pendingTxs []*DefaultSortedTxs

func (p pendingTxs) Len() int {
	return len(p)
}

func (p pendingTxs) Less(i, j int) bool {
	return p[i].GasPrice() < p[j].GasPrice()
}
func (p pendingTxs) Swap(i, j int) {
	mutex.Lock()
	defer mutex.Unlock()
	p[i], p[j] = p[j], p[i]
}

func (pool DefaultPool) SetStatRoot(root hash.Hash) {
	// pool.Stat.SetStatRoot(root)
}

func (pool *DefaultPool) NewTx(tx *types.Transaction) {
	mutex.Lock()
	defer mutex.Unlock()
	account, _ := pool.Stat.Load(tx.From())

	if account.Nonce >= tx.Nonce() {
		fmt.Println(Red + "Invalid nonce, transaction discarded")
		fmt.Printf(Reset)
		return
	}

	nonce := account.Nonce

	pools := pool.pendings[tx.From()]

	if len(pools) > 0 {
		//fmt.Printf("lenth: %d", len(pools))
		last := pools[len(pools)-1]
		nonce = last.Nonce()
		//fmt.Println("last Nonce", last.Nonce())
	}
	if tx.Nonce() > nonce+1 {
		// 加到queue
		fmt.Println(Yellow + "Transaction add Queue")
		fmt.Printf(Reset)
		pool.addQueueTx(tx)

	} else if tx.Nonce() == nonce+1 {
		// 加到pending，判断是否有queue的交易可以pop
		pool.pushPendingTx(tx)
		fmt.Println(Yellow + "Received and added new transaction to the pool")
		fmt.Printf(Reset)
	} else {
		// replace
		pool.replacePendingTx(tx)
		fmt.Println(Yellow + "Replace transaction")
		fmt.Printf(Reset)
	}

}

func (pool *DefaultPool) replacePendingTx(tx *types.Transaction) {

	blks := pool.pendings[tx.From()]
	for _, blk := range blks {
		if blk.Nonce() >= tx.Nonce() {
			if blk.GasPrice() > tx.GasPrice() {
				blk.Replace(tx)
				sort.Sort(blks)
				break
			}

		}
	}
}

func (pool *DefaultPool) pushPendingTx(tx *types.Transaction) {
	blks := pool.pendings[tx.From()]
	if len(blks) == 0 {
		blk := &DefaultSortedTxs{tx}
		blks = append(blks, blk)
		pool.pendings[tx.From()] = blks
		pool.txs = append(pool.txs, blk)
		sort.Sort(pool.txs)
	} else {
		blk := &DefaultSortedTxs{tx}
		blks = append(blks, blk)
		pool.pendings[tx.From()] = blks
		pool.txs = append(pool.txs, blk)
		sort.Sort(pool.txs)
	}
	//TODO 更新queue中可以pop到pending中的交易
	queueTxs := pool.queue[tx.From()]
	nonece := tx.Nonce()
	for key, queueTx := range queueTxs {
		if queueTx.Nonce() == nonece+1 {
			nonece++
			queueTxs = append(queueTxs[:key], queueTxs[key+1:]...)
			pool.queue[tx.From()] = queueTxs
			pool.pushPendingTx(queueTx)
		}
	}
}

func (pool DefaultPool) addQueueTx(tx *types.Transaction) {
	txs := pool.queue[tx.From()]
	txs = append(txs, tx)
	sort.Sort(txs)
	pool.queue[tx.From()] = txs
}

func (pool *DefaultPool) Pop() *types.Transaction {
	mutex.Lock()
	defer mutex.Unlock()
	if len(pool.txs) == 0 {
		return nil
	}

	tx := pool.txs[0].Pop()
	pools := pool.pendings[tx.From()]

	if pools != nil {

		if len(pools) > 0 {
			//fmt.Printf("pools length: %d\n", len(pools))

			pool.pendings[tx.From()] = pool.pendings[tx.From()][1:]
		}
	}
	if tx != nil {
		pool.txs = pool.txs[1:]
	}

	return tx
}

func (pool DefaultPool) NotifyTxEvent(txs []*types.Transaction) {

}
