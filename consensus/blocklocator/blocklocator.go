package blocklocator

import (
	"github.com/kaspanet/kaspad/consensus/blocknode"
	"github.com/kaspanet/kaspad/dagconfig"
	"github.com/kaspanet/kaspad/util/daghash"
	"github.com/pkg/errors"
)

// BlockLocator is used to help locate a specific block. The algorithm for
// building the block locator is to add block hashes in reverse order on the
// block's selected parent chain until the desired stop block is reached.
// In order to keep the list of locator hashes to a reasonable number of entries,
// the step between each entry is doubled each loop iteration to exponentially
// decrease the number of hashes as a function of the distance from the block
// being located.
//
// For example, assume a selected parent chain with IDs as depicted below, and the
// stop block is genesis:
// 	genesis -> 1 -> 2 -> ... -> 15 -> 16  -> 17  -> 18
//
// The block locator for block 17 would be the hashes of blocks:
//  [17 16 14 11 7 2 genesis]
type BlockLocator []*daghash.Hash

type BlockLocatorFactory struct {
	blockNodeStore *blocknode.BlockNodeStore
	params         *dagconfig.Params
}

func NewBlockLocatorFactory(blockNodeStore *blocknode.BlockNodeStore, params *dagconfig.Params) *BlockLocatorFactory {
	return &BlockLocatorFactory{
		blockNodeStore: blockNodeStore,
		params:         params,
	}
}

// BlockLocatorFromHashes returns a block locator from high and low hash.
// See BlockLocator for details on the algorithm used to create a block locator.
func (blf *BlockLocatorFactory) BlockLocatorFromHashes(highHash, lowHash *daghash.Hash) (BlockLocator, error) {
	highNode, ok := blf.blockNodeStore.LookupNode(highHash)
	if !ok {
		return nil, errors.Errorf("block %s is unknown", highHash)
	}

	lowNode, ok := blf.blockNodeStore.LookupNode(lowHash)
	if !ok {
		return nil, errors.Errorf("block %s is unknown", lowHash)
	}

	return blf.blockLocator(highNode, lowNode)
}

// blockLocator returns a block locator for the passed high and low nodes.
// See the BlockLocator type comments for more details.
func (blf *BlockLocatorFactory) blockLocator(highNode, lowNode *blocknode.BlockNode) (BlockLocator, error) {
	// We use the selected parent of the high node, so the
	// block locator won't contain the high node.
	highNode = highNode.SelectedParent()

	node := highNode
	step := uint64(1)
	locator := make(BlockLocator, 0)
	for node != nil {
		locator = append(locator, node.Hash())

		// Nothing more to add once the low node has been added.
		if node.BlueScore() <= lowNode.BlueScore() {
			if node != lowNode {
				return nil, errors.Errorf("highNode and lowNode are " +
					"not in the same selected parent chain.")
			}
			break
		}

		// Calculate blueScore of previous node to include ensuring the
		// final node is lowNode.
		nextBlueScore := node.BlueScore() - step
		if nextBlueScore < lowNode.BlueScore() {
			nextBlueScore = lowNode.BlueScore()
		}

		// walk backwards through the nodes to the correct ancestor.
		node = node.SelectedAncestor(nextBlueScore)

		// Double the distance between included hashes.
		step *= 2
	}

	return locator, nil
}

// FindNextLocatorBoundaries returns the lowest unknown block locator, hash
// and the highest known block locator hash. This is used to create the
// next block locator to find the highest shared known chain block with the
// sync peer.
func (blf *BlockLocatorFactory) FindNextLocatorBoundaries(locator BlockLocator) (highHash, lowHash *daghash.Hash) {
	// Find the most recent locator block hash in the DAG. In the case none of
	// the hashes in the locator are in the DAG, fall back to the genesis block.
	lowNode, _ := blf.blockNodeStore.LookupNode(blf.params.GenesisHash)
	nextBlockLocatorIndex := int64(len(locator) - 1)
	for i, hash := range locator {
		node, ok := blf.blockNodeStore.LookupNode(hash)
		if ok {
			lowNode = node
			nextBlockLocatorIndex = int64(i) - 1
			break
		}
	}
	if nextBlockLocatorIndex < 0 {
		return nil, lowNode.Hash()
	}
	return locator[nextBlockLocatorIndex], lowNode.Hash()
}