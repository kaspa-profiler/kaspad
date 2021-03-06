package headersselectedtipstore

import (
	"github.com/golang/protobuf/proto"
	"github.com/kaspanet/kaspad/domain/consensus/database"
	"github.com/kaspanet/kaspad/domain/consensus/database/serialization"
	"github.com/kaspanet/kaspad/domain/consensus/model"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
)

var headerSelectedTipKey = database.MakeBucket(nil).Key([]byte("headers-selected-tip"))

type headerSelectedTipStore struct {
	staging *externalapi.DomainHash
	cache   *externalapi.DomainHash
}

// New instantiates a new HeaderSelectedTipStore
func New() model.HeaderSelectedTipStore {
	return &headerSelectedTipStore{}
}

func (hts *headerSelectedTipStore) Has(dbContext model.DBReader) (bool, error) {
	if hts.staging != nil {
		return true, nil
	}

	if hts.cache != nil {
		return true, nil
	}

	return dbContext.Has(headerSelectedTipKey)
}

func (hts *headerSelectedTipStore) Discard() {
	hts.staging = nil
}

func (hts *headerSelectedTipStore) Commit(dbTx model.DBTransaction) error {
	if hts.staging == nil {
		return nil
	}

	selectedTipBytes, err := hts.serializeHeadersSelectedTip(hts.staging)
	if err != nil {
		return err
	}
	err = dbTx.Put(headerSelectedTipKey, selectedTipBytes)
	if err != nil {
		return err
	}
	hts.cache = hts.staging

	hts.Discard()
	return nil
}

func (hts *headerSelectedTipStore) Stage(selectedTip *externalapi.DomainHash) {
	hts.staging = selectedTip
}

func (hts *headerSelectedTipStore) IsStaged() bool {
	return hts.staging != nil
}

func (hts *headerSelectedTipStore) HeadersSelectedTip(dbContext model.DBReader) (*externalapi.DomainHash, error) {
	if hts.staging != nil {
		return hts.staging, nil
	}

	if hts.cache != nil {
		return hts.cache, nil
	}

	selectedTipBytes, err := dbContext.Get(headerSelectedTipKey)
	if err != nil {
		return nil, err
	}

	selectedTip, err := hts.deserializeHeadersSelectedTip(selectedTipBytes)
	if err != nil {
		return nil, err
	}
	hts.cache = selectedTip
	return hts.cache, nil
}

func (hts *headerSelectedTipStore) serializeHeadersSelectedTip(selectedTip *externalapi.DomainHash) ([]byte, error) {
	return proto.Marshal(serialization.DomainHashToDbHash(selectedTip))
}

func (hts *headerSelectedTipStore) deserializeHeadersSelectedTip(selectedTipBytes []byte) (*externalapi.DomainHash, error) {
	dbHash := &serialization.DbHash{}
	err := proto.Unmarshal(selectedTipBytes, dbHash)
	if err != nil {
		return nil, err
	}

	return serialization.DbHashToDomainHash(dbHash)
}
