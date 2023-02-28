package app_test

import (
	"testing"

	"github.com/pinebit/lognite/app"
	"github.com/stretchr/testify/assert"
)

func TestBlocks_Basics(t *testing.T) {
	t.Parallel()

	const depth = 5
	blocks := app.NewBlocks(depth)
	assert.True(t, blocks.IsEmpty())

	block := blocks.GetBlockByNumber(1)
	assert.Nil(t, block)

	err := blocks.AddNewBlock(1, 100)
	assert.NoError(t, err)

	block = blocks.GetBlockByNumber(1)
	assert.Equal(t, uint64(1), block.Number)
	assert.Equal(t, uint64(100), block.Timestamp)
	assert.Equal(t, app.NewBlockState, block.State)

	err = blocks.AddNewBlock(1, 100)
	assert.ErrorIs(t, err, app.ErrTooLowBlockNumber)

	for i := 2; i < 2+depth; i++ {
		err := blocks.AddNewBlock(uint64(i), uint64(i*100))
		assert.NoError(t, err)
	}

	block = blocks.GetBlockByNumber(6)
	assert.Equal(t, uint64(6), block.Number)
	assert.Equal(t, uint64(600), block.Timestamp)
	assert.Equal(t, app.NewBlockState, block.State)

	for i := 5; i > 1; i-- {
		block = blocks.GetBlockByNumber(uint64(i))
		assert.Equal(t, uint64(i), block.Number)
		assert.Equal(t, uint64(i*100), block.Timestamp)
		assert.Equal(t, app.ProcessedBlockState, block.State)
	}
}

func TestBlocks_Backfilling(t *testing.T) {
	t.Parallel()

	const depth = 5
	blocks := app.NewBlocks(depth)
	err := blocks.StartBackfilling(1)
	assert.ErrorIs(t, err, app.ErrNoBlocksToBackfill)

	_, ok := blocks.GetNextBackfillingBlockNumber()
	assert.False(t, ok)

	for i := 1; i <= depth; i++ {
		err := blocks.AddNewBlock(uint64(i), uint64(i*100))
		assert.NoError(t, err)
	}

	_, ok = blocks.GetNextBackfillingBlockNumber()
	assert.False(t, ok)

	err = blocks.StartBackfilling(1)
	assert.ErrorIs(t, err, app.ErrTooLowBlockNumber)

	err = blocks.StartBackfilling(depth)
	assert.NoError(t, err)

	number, ok := blocks.GetNextBackfillingBlockNumber()
	assert.True(t, ok)
	assert.Equal(t, uint64(depth), number)

	err = blocks.StartBackfilling(100)
	assert.NoError(t, err)

	backfilledCount := 0
	for {
		number, ok := blocks.GetNextBackfillingBlockNumber()
		if !ok {
			break
		}
		block := blocks.GetBlockByNumber(number)
		block.State = app.ProcessedBlockState
		backfilledCount++
	}
	assert.Equal(t, depth, backfilledCount)
}
