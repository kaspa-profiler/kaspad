package processes

import (
	"fmt"
	"github.com/kaspanet/kaspad/domain/consensus/model"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/processes/ghostdag2"
	"github.com/kaspanet/kaspad/domain/consensus/processes/ghostdagmanager"
	"reflect"
	"testing"
)

func TestGHOSTDA(t *testing.T) {

	type implManager struct {
		function func(
			databaseContext model.DBReader,
			dagTopologyManager model.DAGTopologyManager,
			ghostdagDataStore model.GHOSTDAGDataStore,
			k model.KType) model.GHOSTDAGManager
		implName string
	}

	//NOTE: FOR ADDING/REMOVING AN IMPLEMENTATION CHANGE BELOW:
	implementationFactories := []implManager{
		{ghostdagmanager.New, "Original"},
		{ghostdag2.New, "Tal's impl"},
	}

	type testGhostdagData struct {
		hash                   *externalapi.DomainHash
		parents                []*externalapi.DomainHash
		expectedBlueScore      uint64
		expectedSelectedParent *externalapi.DomainHash
		expectedMergeSetBlues  []*externalapi.DomainHash
		expectedMergeSetReds   []*externalapi.DomainHash
	}

	type isolatedTest struct {
		k        model.KType
		subTests []testGhostdagData
	}

	genesisHash := &externalapi.DomainHash{}
	dagTopology := &DAGTopologyManagerImpl{
		parentsMap: make(map[externalapi.DomainHash][]*externalapi.DomainHash),
	}
	dagTopology.parentsMap[*genesisHash] = nil

	ghostdagDataStore := &GHOSTDAGDataStoreImpl{
		dagMap: make(map[externalapi.DomainHash]*model.BlockGHOSTDAGData),
	}
	ghostdagDataStore.dagMap[*genesisHash] = &model.BlockGHOSTDAGData{
		BlueScore:          1,
		SelectedParent:     nil,
		MergeSetBlues:      nil,
		MergeSetReds:       nil,
		BluesAnticoneSizes: nil,
	}

	// ****************************** TESTS ****************************** //
	// Test1: Graph form is a chain. K = 0
	dag1 := isolatedTest{
		k: 0,
		subTests: []testGhostdagData{
			{
				hash:                   &externalapi.DomainHash{1},
				parents:                []*externalapi.DomainHash{genesisHash},
				expectedBlueScore:      2,
				expectedSelectedParent: genesisHash,
				expectedMergeSetBlues:  []*externalapi.DomainHash{genesisHash},
				expectedMergeSetReds:   []*externalapi.DomainHash{},
			},
			{
				hash:                   &externalapi.DomainHash{2},
				parents:                []*externalapi.DomainHash{{1}},
				expectedBlueScore:      3,
				expectedSelectedParent: &externalapi.DomainHash{1},
				expectedMergeSetBlues:  []*externalapi.DomainHash{{1}},
				expectedMergeSetReds:   []*externalapi.DomainHash{},
			},
			{
				hash:                   &externalapi.DomainHash{3},
				parents:                []*externalapi.DomainHash{{2}},
				expectedBlueScore:      4,
				expectedSelectedParent: &externalapi.DomainHash{2},
				expectedMergeSetBlues:  []*externalapi.DomainHash{{2}},
				expectedMergeSetReds:   []*externalapi.DomainHash{},
			},
		},
	}

	//Test2 : The graph’s longest chain was created by malicious miners (not the “heaviest”). K = 3
	dag2 := isolatedTest{k: 3, subTests: []testGhostdagData{
		{ /* 1*/
			hash:                   &externalapi.DomainHash{1},
			parents:                []*externalapi.DomainHash{genesisHash},
			expectedBlueScore:      2,
			expectedSelectedParent: genesisHash,
			expectedMergeSetBlues:  []*externalapi.DomainHash{genesisHash},
			expectedMergeSetReds:   []*externalapi.DomainHash{},
		},
		{ /* 2 */
			hash:                   &externalapi.DomainHash{2},
			parents:                []*externalapi.DomainHash{{1}},
			expectedBlueScore:      3,
			expectedSelectedParent: &externalapi.DomainHash{1},
			expectedMergeSetBlues:  []*externalapi.DomainHash{{1}},
			expectedMergeSetReds:   []*externalapi.DomainHash{},
		},
		{ /* 3 */
			hash:                   &externalapi.DomainHash{3},
			parents:                []*externalapi.DomainHash{{1}},
			expectedBlueScore:      3,
			expectedSelectedParent: &externalapi.DomainHash{1},
			expectedMergeSetBlues:  []*externalapi.DomainHash{{1}},
			expectedMergeSetReds:   []*externalapi.DomainHash{},
		},
		{ /* 4 */
			hash:                   &externalapi.DomainHash{4},
			parents:                []*externalapi.DomainHash{{1}},
			expectedBlueScore:      3,
			expectedSelectedParent: &externalapi.DomainHash{1},
			expectedMergeSetBlues:  []*externalapi.DomainHash{{1}},
			expectedMergeSetReds:   []*externalapi.DomainHash{},
		},
		{ /* 5 */
			hash:                   &externalapi.DomainHash{5},
			parents:                []*externalapi.DomainHash{{2}, {3}, {4}},
			expectedBlueScore:      6,
			expectedSelectedParent: &externalapi.DomainHash{4},
			expectedMergeSetBlues:  []*externalapi.DomainHash{{4}, {2}, {3}},
			expectedMergeSetReds:   []*externalapi.DomainHash{},
		},
		{ /* 6 */
			hash:                   &externalapi.DomainHash{6},
			parents:                []*externalapi.DomainHash{genesisHash},
			expectedBlueScore:      2,
			expectedSelectedParent: genesisHash,
			expectedMergeSetBlues:  []*externalapi.DomainHash{genesisHash},
			expectedMergeSetReds:   []*externalapi.DomainHash{},
		},
		{ /* 7 */
			hash:                   &externalapi.DomainHash{7},
			parents:                []*externalapi.DomainHash{{6}},
			expectedBlueScore:      3,
			expectedSelectedParent: &externalapi.DomainHash{6},
			expectedMergeSetBlues:  []*externalapi.DomainHash{{6}},
			expectedMergeSetReds:   []*externalapi.DomainHash{},
		},
		{ /* 8 */
			hash:                   &externalapi.DomainHash{8},
			parents:                []*externalapi.DomainHash{{7}},
			expectedBlueScore:      4,
			expectedSelectedParent: &externalapi.DomainHash{7},
			expectedMergeSetBlues:  []*externalapi.DomainHash{{7}},
			expectedMergeSetReds:   []*externalapi.DomainHash{},
		},
		{ /* 9 */
			hash:                   &externalapi.DomainHash{9},
			parents:                []*externalapi.DomainHash{{8}},
			expectedBlueScore:      5,
			expectedSelectedParent: &externalapi.DomainHash{8},
			expectedMergeSetBlues:  []*externalapi.DomainHash{{8}},
			expectedMergeSetReds:   []*externalapi.DomainHash{},
		},
		{ /* 10 */
			hash:                   &externalapi.DomainHash{10},
			parents:                []*externalapi.DomainHash{{5}, {9}},
			expectedBlueScore:      7,
			expectedSelectedParent: &externalapi.DomainHash{5},
			expectedMergeSetBlues:  []*externalapi.DomainHash{{5}},
			expectedMergeSetReds:   []*externalapi.DomainHash{{6}, {7}, {8}, {9}},
		},
	}}

	// Test3: Selected Parent choice: same score – decide by hashes. K = 3
	dag3 := isolatedTest{k: 3, subTests: []testGhostdagData{
		{ /* 1*/
			hash:                   &externalapi.DomainHash{1},
			parents:                []*externalapi.DomainHash{genesisHash},
			expectedBlueScore:      2,
			expectedSelectedParent: genesisHash,
			expectedMergeSetBlues:  []*externalapi.DomainHash{genesisHash},
			expectedMergeSetReds:   []*externalapi.DomainHash{},
		},
		{ /* 2 */
			hash:                   &externalapi.DomainHash{2},
			parents:                []*externalapi.DomainHash{{1}},
			expectedBlueScore:      3,
			expectedSelectedParent: &externalapi.DomainHash{1},
			expectedMergeSetBlues:  []*externalapi.DomainHash{{1}},
			expectedMergeSetReds:   []*externalapi.DomainHash{},
		},
		{ /* 3 */
			hash:                   &externalapi.DomainHash{3},
			parents:                []*externalapi.DomainHash{{1}},
			expectedBlueScore:      3,
			expectedSelectedParent: &externalapi.DomainHash{1},
			expectedMergeSetBlues:  []*externalapi.DomainHash{{1}},
			expectedMergeSetReds:   []*externalapi.DomainHash{},
		},
		{ /* 4 */
			hash:                   &externalapi.DomainHash{4},
			parents:                []*externalapi.DomainHash{{1}},
			expectedBlueScore:      3,
			expectedSelectedParent: &externalapi.DomainHash{1},
			expectedMergeSetBlues:  []*externalapi.DomainHash{{1}},
			expectedMergeSetReds:   []*externalapi.DomainHash{},
		},
		{ /* 5 */
			hash:                   &externalapi.DomainHash{5},
			parents:                []*externalapi.DomainHash{{2}, {3}, {4}},
			expectedBlueScore:      6,
			expectedSelectedParent: &externalapi.DomainHash{4},
			expectedMergeSetBlues:  []*externalapi.DomainHash{{4}, {2}, {3}},
			expectedMergeSetReds:   []*externalapi.DomainHash{},
		},
		{ /* 6 */
			hash:                   &externalapi.DomainHash{6},
			parents:                []*externalapi.DomainHash{{5}},
			expectedBlueScore:      7,
			expectedSelectedParent: &externalapi.DomainHash{5},
			expectedMergeSetBlues:  []*externalapi.DomainHash{{5}},
			expectedMergeSetReds:   []*externalapi.DomainHash{},
		},
	}}

	//Test 4: mergeSetReds is not empty, one of the block in the mergeSet is not connected to more than k . K = 1
	dag4 := isolatedTest{k: 1, subTests: []testGhostdagData{
		{
			hash:                   &externalapi.DomainHash{1},
			parents:                []*externalapi.DomainHash{genesisHash},
			expectedBlueScore:      2,
			expectedSelectedParent: genesisHash,
			expectedMergeSetBlues:  []*externalapi.DomainHash{genesisHash},
			expectedMergeSetReds:   []*externalapi.DomainHash{},
		},
		{
			hash:                   &externalapi.DomainHash{2},
			parents:                []*externalapi.DomainHash{{1}},
			expectedBlueScore:      3,
			expectedSelectedParent: &externalapi.DomainHash{1},
			expectedMergeSetBlues:  []*externalapi.DomainHash{{1}},
			expectedMergeSetReds:   []*externalapi.DomainHash{},
		},
		{
			hash:                   &externalapi.DomainHash{3},
			parents:                []*externalapi.DomainHash{genesisHash},
			expectedBlueScore:      2,
			expectedSelectedParent: genesisHash,
			expectedMergeSetBlues:  []*externalapi.DomainHash{genesisHash},
			expectedMergeSetReds:   []*externalapi.DomainHash{},
		},
		{
			hash:                   &externalapi.DomainHash{4},
			parents:                []*externalapi.DomainHash{{2}, {3}},
			expectedBlueScore:      4,
			expectedSelectedParent: &externalapi.DomainHash{2},
			expectedMergeSetBlues:  []*externalapi.DomainHash{{2}},
			expectedMergeSetReds:   []*externalapi.DomainHash{{3}},
		},
	}}

	//Test 5: Adding a block to the mergeSet will destroy one of the blue block K-cluster(not in the selected chain).(the block is keeping K-cluster )
	dag5 := isolatedTest{k: 2, subTests: []testGhostdagData{
		{
			hash:                   &externalapi.DomainHash{1},
			parents:                []*externalapi.DomainHash{genesisHash},
			expectedBlueScore:      2,
			expectedSelectedParent: genesisHash,
			expectedMergeSetBlues:  []*externalapi.DomainHash{genesisHash},
			expectedMergeSetReds:   []*externalapi.DomainHash{},
		},
		{
			hash:                   &externalapi.DomainHash{2},
			parents:                []*externalapi.DomainHash{genesisHash},
			expectedBlueScore:      2,
			expectedSelectedParent: genesisHash,
			expectedMergeSetBlues:  []*externalapi.DomainHash{genesisHash},
			expectedMergeSetReds:   []*externalapi.DomainHash{},
		},
		{
			hash:                   &externalapi.DomainHash{5},
			parents:                []*externalapi.DomainHash{{1}},
			expectedBlueScore:      3,
			expectedSelectedParent: &externalapi.DomainHash{1},
			expectedMergeSetBlues:  []*externalapi.DomainHash{{1}},
			expectedMergeSetReds:   []*externalapi.DomainHash{},
		},
		{
			hash:                   &externalapi.DomainHash{4},
			parents:                []*externalapi.DomainHash{{2}},
			expectedBlueScore:      3,
			expectedSelectedParent: &externalapi.DomainHash{2},
			expectedMergeSetBlues:  []*externalapi.DomainHash{{2}},
			expectedMergeSetReds:   []*externalapi.DomainHash{},
		},
		{
			hash:                   &externalapi.DomainHash{3},
			parents:                []*externalapi.DomainHash{{2}},
			expectedBlueScore:      3,
			expectedSelectedParent: &externalapi.DomainHash{2},
			expectedMergeSetBlues:  []*externalapi.DomainHash{{2}},
			expectedMergeSetReds:   []*externalapi.DomainHash{},
		},
		{
			hash:                   &externalapi.DomainHash{7},
			parents:                []*externalapi.DomainHash{{3}, {5}},
			expectedBlueScore:      6,
			expectedSelectedParent: &externalapi.DomainHash{5},
			expectedMergeSetBlues:  []*externalapi.DomainHash{{5}, {2}, {3}},
			expectedMergeSetReds:   []*externalapi.DomainHash{},
		},
		{
			hash:                   &externalapi.DomainHash{6},
			parents:                []*externalapi.DomainHash{{5}, {4}},
			expectedBlueScore:      6,
			expectedSelectedParent: &externalapi.DomainHash{5},
			expectedMergeSetBlues:  []*externalapi.DomainHash{{5}, {2}, {4}},
			expectedMergeSetReds:   []*externalapi.DomainHash{},
		},
		{
			hash:                   &externalapi.DomainHash{8},
			parents:                []*externalapi.DomainHash{{3}},
			expectedBlueScore:      4,
			expectedSelectedParent: &externalapi.DomainHash{3},
			expectedMergeSetBlues:  []*externalapi.DomainHash{{3}},
			expectedMergeSetReds:   []*externalapi.DomainHash{},
		},
		{
			hash:                   &externalapi.DomainHash{9},
			parents:                []*externalapi.DomainHash{{6}, {7}, {8}},
			expectedBlueScore:      7,
			expectedSelectedParent: &externalapi.DomainHash{7},
			expectedMergeSetBlues:  []*externalapi.DomainHash{{7}},
			expectedMergeSetReds:   []*externalapi.DomainHash{{4}, {8}, {6}},
		},
	}}

	//Test 6: Adding a block to the mergeSet will destroy one of the blue block K-cluster in the selected chain.(the block is keeping K-cluster )
	dag6 := isolatedTest{k: 3, subTests: []testGhostdagData{
		{
			hash:                   &externalapi.DomainHash{1},
			parents:                []*externalapi.DomainHash{genesisHash},
			expectedBlueScore:      2,
			expectedSelectedParent: genesisHash,
			expectedMergeSetBlues:  []*externalapi.DomainHash{genesisHash},
			expectedMergeSetReds:   []*externalapi.DomainHash{},
		},
		{
			hash:                   &externalapi.DomainHash{2},
			parents:                []*externalapi.DomainHash{genesisHash},
			expectedBlueScore:      2,
			expectedSelectedParent: genesisHash,
			expectedMergeSetBlues:  []*externalapi.DomainHash{genesisHash},
			expectedMergeSetReds:   []*externalapi.DomainHash{},
		},
		{
			hash:                   &externalapi.DomainHash{3},
			parents:                []*externalapi.DomainHash{genesisHash},
			expectedBlueScore:      2,
			expectedSelectedParent: genesisHash,
			expectedMergeSetBlues:  []*externalapi.DomainHash{genesisHash},
			expectedMergeSetReds:   []*externalapi.DomainHash{},
		},
		{
			hash:                   &externalapi.DomainHash{4},
			parents:                []*externalapi.DomainHash{genesisHash},
			expectedBlueScore:      2,
			expectedSelectedParent: genesisHash,
			expectedMergeSetBlues:  []*externalapi.DomainHash{genesisHash},
			expectedMergeSetReds:   []*externalapi.DomainHash{},
		},
		{
			hash:                   &externalapi.DomainHash{6},
			parents:                []*externalapi.DomainHash{{1}, {2}, {3}, {4}},
			expectedBlueScore:      6,
			expectedSelectedParent: &externalapi.DomainHash{4},
			expectedMergeSetBlues:  []*externalapi.DomainHash{{4}, {1}, {2}, {3}},
			expectedMergeSetReds:   []*externalapi.DomainHash{},
		},
		{
			hash:                   &externalapi.DomainHash{5},
			parents:                []*externalapi.DomainHash{{1}, {2}, {3}},
			expectedBlueScore:      5,
			expectedSelectedParent: &externalapi.DomainHash{3},
			expectedMergeSetBlues:  []*externalapi.DomainHash{{3}, {1}, {2}},
			expectedMergeSetReds:   []*externalapi.DomainHash{},
		},
		{
			hash:                   &externalapi.DomainHash{7},
			parents:                []*externalapi.DomainHash{{5}, {6}},
			expectedBlueScore:      7,
			expectedSelectedParent: &externalapi.DomainHash{6},
			expectedMergeSetBlues:  []*externalapi.DomainHash{{6}},
			expectedMergeSetReds:   []*externalapi.DomainHash{{5}},
		},
	}}

	testsArr := []*isolatedTest{&dag1, &dag2, &dag3, &dag4, &dag5, &dag6}
	for _, factory := range implementationFactories {
		fmt.Printf("____________________________\n")
		for testNum, test := range testsArr {
			g := factory.function(nil, dagTopology, ghostdagDataStore, test.k)
			fmt.Printf("Impl:%s,  TestNum:%d\n", factory.implName, testNum+1)
			for _, testBlockData := range test.subTests {
				dagTopology.parentsMap[*testBlockData.hash] = testBlockData.parents
				err := g.GHOSTDAG(testBlockData.hash)
				if err != nil {
					t.Fatalf("Test #%d failed:on  GHOSTDAG error: %s.", testNum+1, err)
				}
				ghostdagData, err := ghostdagDataStore.Get(nil, testBlockData.hash)
				if err != nil {
					t.Fatalf("Test #%d failed: ghostdagDataStore error: %s.", testNum+1, err)
				}
				if testBlockData.expectedBlueScore != (ghostdagData.BlueScore) {
					t.Fatalf("Test #%d failed: expected blue score %d but got %d.", testNum+1, testBlockData.expectedBlueScore, ghostdagData.BlueScore)
				}

				if *testBlockData.expectedSelectedParent != *ghostdagData.SelectedParent {
					t.Fatalf("Test #%d failed: expected selected parent %v but got %v.", testNum+1, testBlockData.expectedSelectedParent, ghostdagData.SelectedParent)
				}

				if !reflect.DeepEqual(testBlockData.expectedMergeSetBlues, ghostdagData.MergeSetBlues) {
					t.Fatalf("Test #%d failed: expected merge set blues %v but got %v.", testNum+1, testBlockData.expectedMergeSetBlues, ghostdagData.MergeSetBlues)
				}

				if !reflect.DeepEqual(testBlockData.expectedMergeSetReds, ghostdagData.MergeSetReds) {
					t.Fatalf("Test #%d failed: expected merge set reds %v but got %v.", testNum+1, testBlockData.expectedMergeSetReds, ghostdagData.MergeSetReds)
				}

			}
			fmt.Printf("    Test success!\n\n")

			dagTopology.parentsMap = make(map[externalapi.DomainHash][]*externalapi.DomainHash)
			dagTopology.parentsMap[*genesisHash] = nil
			blockGHOSTDAGDataGenesis := ghostdagDataStore.dagMap[*genesisHash]
			ghostdagDataStore.dagMap = make(map[externalapi.DomainHash]*model.BlockGHOSTDAGData)
			ghostdagDataStore.dagMap[*genesisHash] = blockGHOSTDAGDataGenesis
		}
	}

}

type GHOSTDAGDataStoreImpl struct {
	dagMap map[externalapi.DomainHash]*model.BlockGHOSTDAGData
}

func (ds *GHOSTDAGDataStoreImpl) Stage(blockHash *externalapi.DomainHash, blockGHOSTDAGData *model.BlockGHOSTDAGData) error {
	ds.dagMap[*blockHash] = blockGHOSTDAGData
	return nil
}

func (ds *GHOSTDAGDataStoreImpl) IsStaged() bool {
	panic("implement me")
}

func (ds *GHOSTDAGDataStoreImpl) Discard() {
	panic("implement me")
}

func (ds *GHOSTDAGDataStoreImpl) Commit(dbTx model.DBTransaction) error {
	panic("implement me")
}

//func (ds *GHOSTDAGDataStoreImpl) Insert(dbTx model.DBTxProxy, blockHash *externalapi.DomainHash, blockGHOSTDAGData *model.BlockGHOSTDAGData) error {
//	ds.dagMap[*blockHash] = blockGHOSTDAGData
//	return nil
//}
func (ds *GHOSTDAGDataStoreImpl) Get(dbContext model.DBReader, blockHash *externalapi.DomainHash) (*model.BlockGHOSTDAGData, error) {
	blockData, isExist := ds.dagMap[*blockHash]
	if isExist {
		return blockData, nil
	}
	return nil, nil
}

type DAGTopologyManagerImpl struct {
	parentsMap map[externalapi.DomainHash][]*externalapi.DomainHash
}

func (dt *DAGTopologyManagerImpl) Tips() ([]*externalapi.DomainHash, error) {
	panic("implement me")
}

func (dt *DAGTopologyManagerImpl) AddTip(tipHash *externalapi.DomainHash) error {
	panic("implement me")
}

//Implemented//
func (dt *DAGTopologyManagerImpl) Parents(blockHash *externalapi.DomainHash) ([]*externalapi.DomainHash, error) {
	blockData, isExist := dt.parentsMap[*blockHash]
	if !isExist {
		return []*externalapi.DomainHash{}, nil
	}
	return blockData, nil
}

func (dt *DAGTopologyManagerImpl) Children(blockHash *externalapi.DomainHash) ([]*externalapi.DomainHash, error) {
	panic("unimplemented")
}

func (dt *DAGTopologyManagerImpl) IsParentOf(blockHashA *externalapi.DomainHash, blockHashB *externalapi.DomainHash) (bool, error) {
	panic("unimplemented")
}

func (dt *DAGTopologyManagerImpl) IsChildOf(blockHashA *externalapi.DomainHash, blockHashB *externalapi.DomainHash) (bool, error) {
	panic("unimplemented")
}

//Implemented//
func (dt *DAGTopologyManagerImpl) IsAncestorOf(hashBlockA *externalapi.DomainHash, hashBlockB *externalapi.DomainHash) (bool, error) {
	blockBParents, isOk := dt.parentsMap[*hashBlockB]
	if !isOk {
		return false, nil
	}

	for _, parentOfB := range blockBParents {
		if *parentOfB == *hashBlockA {
			return true, nil
		}
	}

	for _, parentOfB := range blockBParents {
		isAncestorOf, err := dt.IsAncestorOf(hashBlockA, parentOfB)
		if err != nil {
			return false, err
		}
		if isAncestorOf {
			return true, nil
		}
	}
	return false, nil

}

func (dt *DAGTopologyManagerImpl) IsDescendantOf(blockHashA *externalapi.DomainHash, blockHashB *externalapi.DomainHash) (bool, error) {
	panic("unimplemented")
}

func (dt *DAGTopologyManagerImpl) IsAncestorOfAny(blockHash *externalapi.DomainHash, potentialDescendants []*externalapi.DomainHash) (bool, error) {
	panic("unimplemented")
}
func (dt *DAGTopologyManagerImpl) IsInSelectedParentChainOf(blockHashA *externalapi.DomainHash, blockHashB *externalapi.DomainHash) (bool, error) {
	panic("unimplemented")
}

func (dt *DAGTopologyManagerImpl) SetParents(blockHash *externalapi.DomainHash, parentHashes []*externalapi.DomainHash) error {
	panic("unimplemented")
}
