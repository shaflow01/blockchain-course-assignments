package types

import (
	"blockchain/crypto/secp256k1"
	"blockchain/crypto/sha3"
	"blockchain/utils/hash"
	"blockchain/utils/rlp"
	"fmt"
	"math/big"
)

type Receiption struct {
	TxHash hash.Hash
	Status int
}

type Transaction struct {
	Txdata
	signature
}

type Txdata struct {
	Sender   Address //测试使用，当发送签名交易的时候需要删除
	To       Address
	Nonce    uint64
	Value    uint64
	Gas      uint64
	GasPrice uint64
	//Input    []byte
}

type signature struct {
	R, S *big.Int
	V    uint8
}

func NewTransaction(nonce uint64, to Address, sender Address, value uint64, gas uint64, gasPrice uint64, input []byte) *Transaction {
	return &Transaction{
		Txdata: Txdata{
			Nonce:    nonce,
			To:       to,
			Sender:   sender,
			Value:    value,
			Gas:      gas,
			GasPrice: gasPrice,
			//Input:    input,
		},

		signature: signature{
			R: big.NewInt(0),
			S: big.NewInt(0),
			V: 0,
		},
	}
}

func (tx Transaction) From() Address {
	return tx.Txdata.Sender
}
func (tx Transaction) To() Address {
	return tx.Txdata.To
}

func (tx Transaction) Value() uint64 {
	return tx.Txdata.Value
}
func (tx Transaction) Nonce() uint64 {
	return tx.Txdata.Nonce
}
func (tx Transaction) GasPrice() uint64 {
	return tx.Txdata.GasPrice
}
func (tx Transaction) Hash() hash.Hash {
	data, _ := rlp.EncodeToBytes(tx)
	return sha3.Keccak256(data)
}

func (tx Transaction) Verify() bool {
	txdata := tx.Txdata
	toSign, err := rlp.EncodeToBytes(txdata)
	fmt.Println(toSign, err)
	msg := sha3.Keccak256(toSign)
	sig := make([]byte, 65)
	sig = append(sig, tx.signature.R.Bytes()...)
	sig = append(sig, tx.signature.S.Bytes()...)
	sig = append(sig, tx.signature.V)

	pubKey, err := secp256k1.RecoverPubkey(msg[:], sig)
	if err != nil {
		return false
	}
	recoverAddress := PubKeyToAddress(pubKey)
	return recoverAddress == tx.From()
}
