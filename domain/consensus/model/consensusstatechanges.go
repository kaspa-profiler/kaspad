package model

import "github.com/kaspanet/kaspad/domain/consensus/model/externalapi"

// ConsensusStateChanges represents a set of changes that need to be made
// to transition the current consensus state to a new one
type ConsensusStateChanges struct {
	AcceptanceData  *BlockAcceptanceData
	VirtualUTXODiff *UTXODiff
	NewTips         []*externalapi.DomainHash

	NewBlockUTXODiff  *UTXODiff
	NewBlockMultiset  Multiset
	ParentDiffChanges *map[*externalapi.DomainHash]UTXODiff
}