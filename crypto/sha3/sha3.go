package sha3

import (
	"golang.org/x/crypto/sha3"

	"blockchain/utils/hash"
)

func Keccak256(value []byte) hash.Hash {
	sha := sha3.NewLegacyKeccak256()
	sha.Write(value)
	return hash.BytesToHash(sha.Sum(nil))
}
