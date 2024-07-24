package main

import (
	"sort"
	"strings"
)

type trieNode struct {
	children map[string]*trieNode
}

type Trie struct {
	root *trieNode
}

func reverse(slice []string) []string {
	for i, j := 0, len(slice)-1; i < j; i, j = i+1, j-1 {
		slice[i], slice[j] = slice[j], slice[i]
	}
	return slice
}
func (t *Trie) Search(callChain string) bool {
	funcs := reverse(strings.Split(callChain, "->"))
	node := t.root
	for _, f := range funcs {
		if nextnode, exists := node.children[f]; exists {
			node = nextnode
		} else {
			return false
		}
	}
	return true
}
func (t *Trie) Insert(callChain string) {
	funcs := reverse(strings.Split(callChain, "->"))
	node := t.root
	for _, f := range funcs {
		if _, exists := node.children[f]; !exists {
			node.children[f] = &trieNode{make(map[string]*trieNode)}
		}
		node = node.children[f]
	}
}

func duplicatesByTrie(input *SearchResultInfo) *SearchResultInfo {
	trie := &Trie{&trieNode{make(map[string]*trieNode)}}

	sort.SliceStable(input.CallChain, func(i, j int) bool {
		return len(input.CallChain[i]) > len(input.CallChain[j])
	})
	sort.SliceStable(input.TargetRowNums, func(i, j int) bool {
		return len(input.CallChain[i]) > len(input.CallChain[j])
	})
	output := &SearchResultInfo{Errs: input.Errs}

	for i, callLine := range input.CallChain {
		if !trie.Search(callLine) {
			output.CallChain = append(output.CallChain, callLine)
			output.TargetRowNums = append(output.TargetRowNums, input.TargetRowNums[i])
			trie.Insert(callLine)
		}
	}
	return output
}
