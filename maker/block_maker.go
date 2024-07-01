package maker

import (
	"blockchain/blockchain"
	"blockchain/statemachine"
	"blockchain/trie"
	"blockchain/txpool"
	"blockchain/types"
	"blockchain/utils/hexutil"
	"blockchain/utils/xtime"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	Red    = "\033[31m"
	Yellow = "\033[33m"
	Green  = "\033[32m"
	Reset  = "\033[0m"
)

var mutex sync.Mutex
var sharedData int

type ChainConfig struct {
	Duration   time.Duration
	Coinbase   types.Address
	Difficulty uint64
}

type BlockMaker struct {
	txpool *txpool.DefaultPool
	state  *trie.State
	exec   *statemachine.StateMachine

	config ChainConfig
	chain  *blockchain.Blockchain

	nextHeader *blockchain.Header
	nextBody   *blockchain.Body

	interupt chan bool
}

func (maker *BlockMaker) InitMakerConfig() {
	maker.config = ChainConfig{
		Duration:   1 * time.Second,
		Coinbase:   types.Address{},
		Difficulty: 2,
	}

}

func NewBlockMaker(state *trie.State, exec *statemachine.StateMachine, chain *blockchain.Blockchain) *BlockMaker {
	return &BlockMaker{
		txpool: chain.Txpool,
		state:  state,
		exec:   exec,
		chain:  chain,
	}
}

func (maker *BlockMaker) NewBlock() {
	maker.nextBody = blockchain.NewBlockBody()
	maker.nextHeader = blockchain.NewHeader(maker.chain.CurrentHeader)
	maker.InitMakerConfig()
	maker.nextHeader.Coinbase = maker.config.Coinbase
}

func (maker *BlockMaker) Pack() uint64 {
	end := time.After(maker.config.Duration)
	var totalgas uint64 = 0
Loop:
	for {
		select {
		case <-maker.interupt:

			break Loop
		case <-end:
			break Loop
		default:
			totalgas += maker.pack()
		}
	}
	//更新区块链状态树
	maker.nextHeader.Root = maker.state.Root()
	return uint64(totalgas)
}
func (maker *BlockMaker) pack() uint64 {
	mutex.Lock()
	defer mutex.Unlock()
	tx := maker.txpool.Pop()
	if tx != nil {
		receiption, gasUsed := maker.exec.Execute(maker.state, tx)
		if receiption == nil {
			fmt.Println(Red + "Tx execute failed.")
			fmt.Printf(Reset)
			return 0
		}
		fmt.Println(Green + "The transaction has been executed successfully!")
		fmt.Printf(Reset)
		maker.nextBody.Transactions = append(maker.nextBody.Transactions, *tx)
		maker.nextBody.Receiptions = append(maker.nextBody.Receiptions, *receiption)
		if len(maker.nextBody.Transactions) >= 10 {
			maker.Interupt()
		}
		return gasUsed
	} else {
		//fmt.Println(Yellow + "Txpool is empty, waiting for transactions.")
		fmt.Printf(Reset)
		return 0
	}
}

func (maker *BlockMaker) Interupt() {
	maker.interupt <- true
}

func (maker *BlockMaker) Mint() (*blockchain.Header, *blockchain.Body) {
	maker.nextHeader.Timestamp = xtime.Now()
	maker.nextHeader.Nonce = 0
	diff := maker.config.Difficulty
	zeroPrefix := strings.Repeat("0", int(diff))
	fmt.Println("Mining difficulty:", zeroPrefix)
	for n := 0; ; n++ {
		maker.nextHeader.Nonce = uint64(n)
		hash := maker.nextHeader.Hash().String()[2 : diff+2]
		if hash == zeroPrefix {
			fmt.Println(Green+"Mining successful:", maker.nextHeader.Hash().String())
			fmt.Printf(Reset)
			break
		}
	}
	return maker.nextHeader, maker.nextBody
}

func (maker *BlockMaker) PackAndMint(minter types.Address) bool {
	maker.NewBlock()
	fmt.Println("Packing...")
	minterReward := maker.Pack()
	maker.addMinterTx(minter, minterReward)
	fmt.Printf(Reset)
	header, body := maker.Mint()
	fmt.Println("|--------------------------------------------------------------------------------------------------|")
	fmt.Println("|block data:                                                                                       |")
	fmt.Println("|--------------------------------------------------------------------------------------------------|")
	fmt.Println("|block hash:", header.Hash().String())
	fmt.Println("|ParentHash:", header.ParentHash.String())
	fmt.Println("|Height:", header.Height)
	fmt.Println("|Timestamp:", header.Timestamp)
	fmt.Println("|Transaction data:")
	for i, tx := range body.Transactions {
		fmt.Printf("|	Transaction %d:%s\n", i, tx.Hash().String())
	}
	fmt.Println("|--------------------------------------------------------------------------------------------------")
	maker.chain.CurrentHeader = *header
	return true

}

func (maker *BlockMaker) addMinterTx(minter types.Address, minterReward uint64) bool {
	var inputString string = "minter reward"
	input, _ := hexutil.Decode(inputString)
	var addr types.Address
	tx := types.NewTransaction(0, addr, minter, 50, 0, 0, input)
	toAccount, err := maker.state.Load(minter)
	if err != nil {
		toAccount = types.Account{}
	}

	toAccount.Amount = toAccount.Amount + 50 + minterReward
	maker.state.Store(minter, toAccount)
	receiption := &types.Receiption{
		TxHash: tx.Hash(),
		Status: 0,
	}
	maker.nextBody.Transactions = append(maker.nextBody.Transactions, *tx)
	maker.nextBody.Receiptions = append(maker.nextBody.Receiptions, *receiption)
	fmt.Println(Green + "minter reward has been added")
	fmt.Printf(Reset)
	return true
}
