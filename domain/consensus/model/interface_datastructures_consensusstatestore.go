package model

import "github.com/kaspanet/kaspad/domain/consensus/model/externalapi"

// ConsensusStateStore represents a store for the current consensus state
type ConsensusStateStore interface {
	Store
	IsStaged() bool

	StageVirtualUTXODiff(virtualUTXODiff externalapi.UTXODiff)
	UTXOByOutpoint(dbContext DBReader, outpoint *externalapi.DomainOutpoint) (externalapi.UTXOEntry, error)
	HasUTXOByOutpoint(dbContext DBReader, outpoint *externalapi.DomainOutpoint) (bool, error)
	VirtualUTXOSetIterator(dbContext DBReader) (externalapi.ReadOnlyUTXOSetIterator, error)
	VirtualUTXOs(dbContext DBReader,
		fromOutpoint *externalapi.DomainOutpoint, limit int) ([]*externalapi.OutpointAndUTXOEntryPair, error)

	StageTips(tipHashes []*externalapi.DomainHash)
	Tips(dbContext DBReader) ([]*externalapi.DomainHash, error)

	StartImportingPruningPointUTXOSet(dbContext DBWriter) error
	HadStartedImportingPruningPointUTXOSet(dbContext DBWriter) (bool, error)
	ImportPruningPointUTXOSetIntoVirtualUTXOSet(dbContext DBWriter, pruningPointUTXOSetIterator externalapi.ReadOnlyUTXOSetIterator) error
	FinishImportingPruningPointUTXOSet(dbContext DBWriter) error
}
