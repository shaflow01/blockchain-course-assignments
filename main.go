package main

import (
	"blockchain/blockchain"
	"blockchain/kvstore"
	"blockchain/maker"
	"blockchain/statemachine"
	"blockchain/trie"
	"blockchain/txpool"
	"blockchain/types"
	"blockchain/utils/hexutil"
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"
)

const (
	Red    = "\033[31m"
	Yellow = "\033[33m"
	Green  = "\033[32m"
	Reset  = "\033[0m"
)

type Node interface {
	startNode() error
}

type node struct {
	blockchain *blockchain.Blockchain
	minter     string
}

type TransactionData struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Nonce    uint64 `json:"nonce"`
	Value    uint64 `json:"value"`
	Gas      uint64 `json:"gas"`
	GasPrice uint64 `json:"gasPrice"`
	Input    string `json:"input"`
	R        string `json:"r"`
	S        string `json:"s"`
	V        uint8  `json:"v"`
}

type AccountStatusResponse struct {
	Balance uint64 `json:"balance"`
	Nonce   uint64 `json:"nonce"`
}

func main() {
	node := initNode()
	node.startNode()
}

func NewNode(statedb *trie.State, txpool *txpool.DefaultPool) *node {
	return &node{
		blockchain.NewBlockchain(statedb, txpool),
		"0xbE4bf446e2Bdd6ebaD529A4df21911c87E48E535",
	}
}

func (n *node) startNode() error {
	fmt.Println("The node has started and is now listening for transactions...")
	go n.listenForTransactions()
	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-ticker.C:
			n.createBlock()
		}
	}
}

func initNode() *node {
	fmt.Println("Initialing...")
	db := kvstore.NewLevelDB("./leveldb")
	state := trie.NewState(db, trie.EmptyHash)

	// Initialize accounts for testing
	initAccount(state, "0x9B682e9770C315f43954e37D8880a6Be815A3E53", 300, 0)

	txpool := txpool.NewDefaultPool(state)
	node := NewNode(state, txpool)
	fmt.Println(Green + "Node initialization successful!")
	fmt.Printf(Reset)
	return node
}

func initAccount(state *trie.State, address string, amount uint64, nonce uint64) {
	account := types.Account{
		Amount: amount,
		Nonce:  nonce,
	}
	add, _ := hexutil.Decode(address)
	var addr types.Address
	copy(addr[:], add[:20])
	fmt.Println("Init A Account")
	state.Store(addr, account)
}

func (n *node) listenForTransactions() {
	listen, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println(Red+"Error setting up listener:", err)
		return
	}
	defer listen.Close()

	fmt.Println("Listening on port 8080...")
	fmt.Println("================================================================")
	for {
		conn, err := listen.Accept()
		if err != nil {
			fmt.Println(Yellow+"Error accepting connection:", err)
			fmt.Printf(Reset)
			continue
		}

		go n.handleConnection(conn)
	}
}

func (n *node) handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for {
		request, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println(Red+"Error reading data:", err)
			fmt.Printf(Reset)
			return
		}

		request = strings.TrimSpace(request)
		if strings.HasPrefix(request, "GET_ACCOUNT_STATUS") {
			address := strings.TrimPrefix(request, "GET_ACCOUNT_STATUS ")
			n.handleAccountStatusRequest(conn, address)
		} else {
			n.handleTransactionRequest(conn, request)
		}
	}
}

func (n *node) handleTransactionRequest(conn net.Conn, request string) {
	var txData TransactionData
	err := json.Unmarshal([]byte(request), &txData)
	if err != nil {
		fmt.Println("Error unmarshalling data:", err)
		fmt.Printf(Reset)
		return
	}

	fromAddress := txData.From
	fromAdd, _ := hexutil.Decode(fromAddress)
	var fromAddr types.Address
	copy(fromAddr[:], fromAdd[:20])

	toAddress := txData.To
	toAdd, _ := hexutil.Decode(toAddress)
	var toAddr types.Address
	copy(toAddr[:], toAdd[:20])
	tx := types.NewTransaction(txData.Nonce, toAddr, fromAddr, txData.Value, txData.Gas, txData.GasPrice, []byte(txData.Input))
	success := tx.Verify()
	if success {
		n.blockchain.Txpool.NewTx(tx)
	} else {
		fmt.Println("Transaction verification failed!")
	}

}

func (n *node) handleAccountStatusRequest(conn net.Conn, address string) {
	address1, _ := hexutil.Decode(address)
	var Address1 types.Address
	copy(Address1[:], address1[:20])
	account, _ := n.blockchain.Statedb.Load(Address1)
	response := AccountStatusResponse{
		Balance: account.Amount,
		Nonce:   account.Nonce,
	}

	respJSON, err := json.Marshal(response)
	if err != nil {
		fmt.Println("Error marshalling response:", err)
		return
	}

	respJSON = append(respJSON, '\n')
	_, err = conn.Write(respJSON)
	if err != nil {
		fmt.Println("Error sending response:", err)
	}
}

func (n *node) createBlock() {

	fmt.Println("start make block...")
	machine := statemachine.NewStateMachine()
	blockMaker := maker.NewBlockMaker(n.blockchain.Statedb, machine, n.blockchain)
	address := "0xbE4bf446e2Bdd6ebaD529A4df21911c87E48E535"
	add, _ := hexutil.Decode(address)
	var addr types.Address
	copy(addr[:], add[:20])
	if blockMaker.PackAndMint(addr) {
		fmt.Println(Green + "block make success.")
		fmt.Printf(Reset)
	} else {
		fmt.Println(Reset + "stop make.")
		fmt.Printf(Reset)
	}
	fmt.Println("================================================================")

}
