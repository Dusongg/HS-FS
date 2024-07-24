package main

import (
	"sort"
	"strings"
)

type trieNode struct {
	children map[string]*trieNode
}

type Trie struct {
	allnode map[string]*trieNode
}

func (t *Trie) Search(callChain string) bool {
	funcs := strings.Split(callChain, "->")
	node, exists := t.allnode[funcs[0]]
	if !exists {
		return false
	}
	for _, f := range funcs[1:] {
		if nextnode, exists := node.children[f]; exists {
			node = nextnode
		} else {
			return false
		}
	}
	return true
}
func (t *Trie) Insert(callChain string) {
	funcs := strings.Split(callChain, "->")
	if _, exists := t.allnode[funcs[0]]; !exists {
		t.allnode[funcs[0]] = &trieNode{make(map[string]*trieNode)}
	}
	curNode := t.allnode[funcs[0]]
	for _, f := range funcs[1:] {
		if _, exists := curNode.children[f]; !exists {
			curNode.children[f] = &trieNode{make(map[string]*trieNode)}
		}
		curNode = curNode.children[f]
		t.allnode[f] = curNode
	}
}

func duplicatesByTrie(input *SearchResultInfo) *SearchResultInfo {
	trie := &Trie{make(map[string]*trieNode)}

	sort.Slice(input.CallChain, func(i, j int) bool {
		return len(input.CallChain[i]) > len(input.CallChain[j])
	})
	sort.Slice(input.TargetRowNums, func(i, j int) bool {
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
