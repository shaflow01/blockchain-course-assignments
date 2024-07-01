package types

import (
	"blockchain/crypto/sha3"
)

type Address [20]byte

func PubKeyToAddress(pub []byte) Address {
	h := sha3.Keccak256(pub[1:])
	var address Address // 创建一个Address类型的实例
	copy(address[:], h[12:])
	return address
}
