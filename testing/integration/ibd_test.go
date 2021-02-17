package integration

import (
	"fmt"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"sync"
	"testing"
	"time"

	"github.com/kaspanet/kaspad/domain/dagconfig"

	"github.com/kaspanet/kaspad/app/appmessage"
)

func TestIBD(t *testing.T) {
	const numBlocks = 100

	syncer, syncee, _, teardown := standardSetup(t)
	defer teardown()

	for i := 0; i < numBlocks; i++ {
		mineNextBlock(t, syncer)
	}

	blockAddedWG := sync.WaitGroup{}
	blockAddedWG.Add(numBlocks)
	receivedBlocks := 0
	setOnBlockAddedHandler(t, syncee, func(_ *appmessage.BlockAddedNotificationMessage) {
		receivedBlocks++
		blockAddedWG.Done()
	})

	// We expect this to trigger IBD
	connect(t, syncer, syncee)

	select {
	case <-time.After(defaultTimeout):
		t.Fatalf("Timeout waiting for IBD to finish. Received %d blocks out of %d", receivedBlocks, numBlocks)
	case <-ReceiveFromChanWhenDone(func() { blockAddedWG.Wait() }):
	}

	tip1Hash, err := syncer.rpcClient.GetSelectedTipHash()
	if err != nil {
		t.Fatalf("Error getting tip for syncer")
	}
	tip2Hash, err := syncee.rpcClient.GetSelectedTipHash()
	if err != nil {
		t.Fatalf("Error getting tip for syncee")
	}

	if tip1Hash.SelectedTipHash != tip2Hash.SelectedTipHash {
		t.Errorf("Tips of syncer: '%s' and syncee '%s' are not equal", tip1Hash.SelectedTipHash, tip2Hash.SelectedTipHash)
	}
}

// TestIBDWithPruning checks the IBD from a node with
// already pruned blocks.
func TestIBDWithPruning(t *testing.T) {
	const numBlocks = 100

	overrideDAGParams := dagconfig.SimnetParams

	// This is done to make a pruning depth of 6 blocks
	overrideDAGParams.FinalityDuration = 2 * overrideDAGParams.TargetTimePerBlock
	overrideDAGParams.K = 0
	harnesses, teardown := setupHarnesses(t, []*harnessParams{
		{
			p2pAddress:              p2pAddress1,
			rpcAddress:              rpcAddress1,
			miningAddress:           miningAddress1,
			miningAddressPrivateKey: miningAddress1PrivateKey,
			overrideDAGParams:       &overrideDAGParams,
		},
		{
			p2pAddress:              p2pAddress2,
			rpcAddress:              rpcAddress2,
			miningAddress:           miningAddress2,
			miningAddressPrivateKey: miningAddress2PrivateKey,
			overrideDAGParams:       &overrideDAGParams,
		},
	})
	defer teardown()

	syncer, syncee := harnesses[0], harnesses[1]

	// Let the syncee have two blocks that the syncer
	// doesn't have to test a situation where
	// the block locator will need more than one
	// iteration to find the highest shared chain
	// block.
	const synceeOnlyBlocks = 2
	for i := 0; i < synceeOnlyBlocks; i++ {
		mineNextBlock(t, syncee)
	}

	for i := 0; i < numBlocks-1; i++ {
		mineNextBlock(t, syncer)
	}

	// We expect this to trigger IBD
	connect(t, syncer, syncee)

	syncerBlockCountResponse, err := syncer.rpcClient.GetBlockCount()
	if err != nil {
		t.Fatalf("GetBlockCount: %+v", err)
	}

	if syncerBlockCountResponse.BlockCount == syncerBlockCountResponse.HeaderCount {
		t.Fatalf("Expected some pruned blocks but found none")
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	start := time.Now()
	for range ticker.C {
		if time.Since(start) > defaultTimeout {
			t.Fatalf("Timeout waiting for IBD to finish.")
		}

		tip1Hash, err := syncer.rpcClient.GetSelectedTipHash()
		if err != nil {
			t.Fatalf("Error getting tip for syncer")
		}
		tip2Hash, err := syncee.rpcClient.GetSelectedTipHash()
		if err != nil {
			t.Fatalf("Error getting tip for syncee")
		}

		if tip1Hash.SelectedTipHash == tip2Hash.SelectedTipHash {
			break
		}
	}

	synceeBlockCountResponse, err := syncee.rpcClient.GetBlockCount()
	if err != nil {
		t.Fatalf("GetBlockCount: %+v", err)
	}

	if synceeBlockCountResponse.BlockCount != syncerBlockCountResponse.BlockCount+synceeOnlyBlocks+1 {
		t.Fatalf("Because the syncee haven't pruned any of its old blocks, its expected "+
			"block count is expected to be greater than the syncer by synceeOnlyBlocks(%d)+genesis, but instead "+
			"we got syncer block count of %d and syncee block count of %d", synceeOnlyBlocks,
			syncerBlockCountResponse.BlockCount,
			synceeBlockCountResponse.BlockCount)
	}

	if synceeBlockCountResponse.HeaderCount != syncerBlockCountResponse.HeaderCount+synceeOnlyBlocks {
		t.Fatalf("Because the syncer haven't synced from the syncee, its expected "+
			"block count is expected to be smaller by synceeOnlyBlocks(%d), but instead "+
			"we got syncer headers count of %d and syncee headers count of %d", synceeOnlyBlocks,
			syncerBlockCountResponse.HeaderCount,
			synceeBlockCountResponse.HeaderCount)
	}
}

func TestIBDTwoChains(t *testing.T) {
	shortKaspad, longKaspad, syncee, teardown := standardSetup(t)
	defer teardown()

	// Build many short chains on shortKaspad
	shortKaspadBlocks := []*externalapi.DomainBlock{}
	for i := 0; i < 100; i++ {
		blockTemplate, err := shortKaspad.rpcClient.GetBlockTemplate(shortKaspad.miningAddress)
		if err != nil {
			t.Fatalf("Error getting block template: %+v", err)
		}
		block := appmessage.MsgBlockToDomainBlock(blockTemplate.MsgBlock)
		solveBlock(block)
		shortKaspadBlocks = append(shortKaspadBlocks, block)
	}
	for _, shortKaspadBlock := range shortKaspadBlocks {
		_, err := shortKaspad.rpcClient.SubmitBlock(shortKaspadBlock)
		if err != nil {
			t.Fatalf("Error submitting block: %s", err)
		}
	}

	// Build a long chain on longKaspad
	for i := 0; i < 10_000; i++ {
		mineNextBlock(t, longKaspad)
	}

	// Connect shortKaspad and longKaspad. This should trigger IBD between them
	connect(t, shortKaspad, longKaspad)

	waitForIBDToFinish := func(t *testing.T, harness1 *appHarness, harness2 *appHarness) {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for range ticker.C {
			tip1Hash, err := harness1.rpcClient.GetSelectedTipHash()
			if err != nil {
				t.Fatalf("Error getting tip for syncer: %s", err)
			}
			tip2Hash, err := harness2.rpcClient.GetSelectedTipHash()
			if err != nil {
				t.Fatalf("Error getting tip for syncee: %s", err)
			}
			if tip1Hash.SelectedTipHash == tip2Hash.SelectedTipHash {
				break
			}
		}
	}

	// Wait for IBD to finish
	waitForIBDToFinish(t, shortKaspad, longKaspad)

	// Merge all the chains into one
	for {
		block := mineNextBlock(t, longKaspad)
		if len(block.Header.ParentHashes()) == 1 {
			break
		}
	}

	// Connect longKaspad and syncee. This should trigger IBD between them
	connect(t, longKaspad, syncee)

	// Wait for IBD to finish
	start := time.Now()
	waitForIBDToFinish(t, longKaspad, syncee)
	fmt.Printf("IBD finished. Took: %s\n", time.Since(start))
}
