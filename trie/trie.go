package trie

import (
	"blockchain/crypto/sha3"
	"blockchain/kvstore"
	"blockchain/types"
	"blockchain/utils/hash"
	"blockchain/utils/hexutil"
	"blockchain/utils/rlp"
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
)

var EmptyHash = hash.BigToHash(big.NewInt(0))

type ITrie interface {
	Store(key []byte, account types.Account) error //key用来存地址，value是账户信息，序列化之后也是byte
	Root() hash.Hash                               //返回默克尔根hash
	Load(key []byte) (types.Account, error)        //查询功能
}

type State struct { //世界状态
	root *TrieNode
	db   kvstore.KVDatabase
}

type TrieNode struct {
	Path     string //路径
	Leaf     bool
	Value    hash.Hash
	Children Children
}

type Children []Child

type Child struct {
	Path string    //压缩的前缀
	Hash hash.Hash //用于指向下个节点
}

func NewChild(path string, hash hash.Hash) Child {
	return Child{
		Path: path,
		Hash: hash,
	}

}
func (childern Children) Len() int {

	return len(childern)
}

func (childern Children) Less(i, j int) bool {
	return strings.Compare(childern[i].Path, childern[j].Path) < 0 //= 0 <-1 >1
}

func (children Children) Swap(i, j int) {
	children[i], children[j] = children[j], children[i]
}

func NewState(db kvstore.KVDatabase, root hash.Hash) *State {
	if bytes.Equal(root[:], EmptyHash[:]) {

		state := State{
			db:   db,
			root: NewTrieNode(),
		}
		state.SaveTrieNode(*NewTrieNode())
		return &state

	} else {
		value, err := db.Get(root[:])
		if err != nil {
			panic(err)
		}
		node, err := TrieNodeFromBytes(value)
		if err != nil {
			panic(err)
		}
		return &State{
			db:   db,
			root: node,
		}
	}
}

func NewTrieNode() *TrieNode {
	return &TrieNode{
		Path: "",
	}
}

func TrieNodeFromBytes(data []byte) (*TrieNode, error) {
	var node TrieNode
	err := rlp.DecodeBytes(data, &node)

	return &node, err
}
func (node *TrieNode) Sort() {
	sort.Sort(node.Children)
}

func (node *TrieNode) Bytes() []byte {
	data, _ := rlp.EncodeToBytes(node)
	return data
}

func (node *TrieNode) Hash() hash.Hash {
	data := node.Bytes()
	return sha3.Keccak256(data)
}

func (state *State) Root() hash.Hash {
	return state.root.Hash()
}

func (state *State) Pri() {

	fmt.Println("Path1:", state.root.Path)
	fmt.Println("Children2:", state.root.Children)
	fmt.Println("ROOT3:", state.Root())
}

func (state *State) Load(key types.Address) (types.Account, error) {
	path := hexutil.Encode(key[:])
	path = path[2:]
	paths, hashs := state.FindAncestors(path)
	matched := strings.Join(paths, "")
	var account types.Account
	if strings.EqualFold(path, matched) {
		lastHash := hashs[len(hashs)-1]
		leafNode, err := state.LoadTrieNodeByHash(lastHash)

		if err != nil {
			return account, err
		}
		if !leafNode.Leaf {
			return account, errors.New("not found")
		}

		data, err := state.db.Get(leafNode.Value[:])
		_ = rlp.DecodeBytes(data, &account)
		return account, err
	} else {
		return account, errors.New("not found")
	}
}

func (state *State) LoadTrieNodeByHash(hash hash.Hash) (*TrieNode, error) {
	data, err := state.db.Get(hash[:])
	if err != nil {
		return nil, err
	}
	return TrieNodeFromBytes(data)
}
func (state *State) SaveTrieNode(node TrieNode) {
	h := node.Hash()
	state.db.Put(h[:], node.Bytes())
}

// 自下向上更新trie
func (state *State) UpdateTrie(node *TrieNode, hashes []hash.Hash) {
	childHash := node.Hash()
	childPath := node.Path
	depth := len(hashes)
	if depth == 1 {
		state.root = node
	}

	for i := depth - 2; i >= 0; i-- { //倒数第二个去找
		current, _ := state.LoadTrieNodeByHash(hashes[i])
		for key, _ := range current.Children {
			if strings.Contains(current.Children[key].Path, childPath) {
				current.Children[key].Hash = childHash
				current.Children[key].Path = childPath
				state.SaveTrieNode(*current)
				childHash = current.Hash()
				childPath = current.Path
				break
			}
		}
		if i == 0 {
			state.root = current
		}
	}
}

func (state *State) Store(key types.Address, account types.Account) error {
	value := account.Bytes()
	valueHash := sha3.Keccak256(value)
	state.db.Put(valueHash[:], value)
	//step1 find all ancients
	path := hexutil.Encode(key[:])
	path = path[2:]

	paths, hashes := state.FindAncestors(path)
	prefix := strings.Join(paths, "")
	depth := len(hashes)
	node, _ := state.LoadTrieNodeByHash(hashes[depth-1])
	if strings.EqualFold(prefix, path) {

		//如果已经存在节点
		node.Value = valueHash

		state.SaveTrieNode(*node)

		//自下向上更新trie
		state.UpdateTrie(node, hashes)
	} else {

		if strings.EqualFold(node.Path, paths[depth-1]) {
			prefix := strings.Join(paths, "")
			leafPath := path[len(prefix):] // error
			leafNode := NewTrieNode()
			leafNode.Leaf = true
			leafNode.Path = leafPath
			leafNode.Value = valueHash
			state.SaveTrieNode(*leafNode)
			leafHash := leafNode.Hash()
			node.Children = append(node.Children, NewChild(leafPath, leafHash))
			node.Sort()               //完成当前节点的更新
			state.SaveTrieNode(*node) //最后匹配的节点存下来

			//自下向上更新trie
			state.UpdateTrie(node, hashes)
		} else {
			//不存在节点，但是是分叉

			//第一个孩子
			lastMatched := paths[len(paths)-1]
			node.Path = node.Path[len(lastMatched):]
			state.SaveTrieNode(*node)

			prefix := strings.Join(paths, "")
			leafPath := path[len(prefix):]

			//第二个孩子
			leafNode := NewTrieNode()
			leafNode.Leaf = true
			leafNode.Path = leafPath
			leafNode.Value = valueHash
			state.SaveTrieNode(*leafNode)

			//孩子的父亲
			newNode := NewTrieNode()
			newNode.Path = lastMatched
			newNode.Children = make(Children, 0)
			newNode.Children = append(newNode.Children, NewChild(node.Path, node.Hash()), NewChild(leafNode.Path, leafNode.Hash()))
			newNode.Sort()
			state.SaveTrieNode(*newNode)
			//自下向上更新trie
			state.UpdateTrie(newNode, hashes)
		}
	}
	return nil
}

func (state *State) FindAncestors(path string) ([]string, []hash.Hash) { //返回所有的路径值，所有的hash值
	current := state.root
	paths, hashes := make([]string, 0), make([]hash.Hash, 0)
	paths = append(paths, "")
	hashes = append(hashes, state.Root())
	prefix := ""
	for {
		flag := false
		for _, child := range current.Children {
			tmp := prefix + child.Path
			length := prefixLength(path, tmp)
			if length == len(tmp) { //完全匹配
				prefix = prefix + child.Path
				paths = append(paths, child.Path)
				hashes = append(hashes, child.Hash)
				flag = true
				data, _ := state.db.Get(child.Hash[:])
				current, _ = TrieNodeFromBytes(data) //当前的current指过去
				break
			} else if length > len(prefix) {
				//部分不匹配
				l := length - len(prefix)
				str := child.Path[:l]
				paths = append(paths, str)
				hashes = append(hashes, child.Hash)
				return paths, hashes
			}

		}
		if !flag {
			break
		}
	}
	return paths, hashes
}

func prefixLength(s1, s2 string) int {
	length := len(s1)
	if length > len(s2) {
		length = len(s2) //先找到短的
	}
	for i := 0; i < length; i++ {
		if s1[i] != s2[i] {
			return i
		} //退出条件是找到第i个节点，当第几个不相等的时候，返回的是前面的数量 最长的前缀数量等于最短的字符串长度
	}
	return length
}
