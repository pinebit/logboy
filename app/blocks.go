package app

import (
	"errors"
)

type BlockState int

const (
	NewBlockState BlockState = iota
	BackfillingBlockState
	ProcessedBlockState
)

var (
	ErrTooLowBlockNumber  = errors.New("too old block number")
	ErrNoBlocksToBackfill = errors.New("no blocks to backfill")
)

type Block struct {
	Number    uint64
	Timestamp uint64
	State     BlockState
}

type Blocks interface {
	IsEmpty() bool
	GetBlockByNumber(number uint64) *Block
	AddNewBlock(number uint64, timestamp uint64) error
	StartBackfilling(number uint64) error
	GetNextBackfillingBlockNumber() (uint64, bool)
}

type blocks struct {
	depth int
	list  []*Block
}

func NewBlocks(depth int) Blocks {
	return &blocks{
		depth: depth,
		list:  []*Block{},
	}
}

func (b blocks) IsEmpty() bool {
	return len(b.list) == 0
}

func (b blocks) GetBlockByNumber(number uint64) *Block {
	lastIndex := len(b.list) - 1
	if lastIndex < 0 {
		return nil
	}
	index := int64(b.list[0].Number) - int64(number)
	if index < 0 || index > int64(lastIndex) {
		return nil
	}
	return b.list[index]
}

func (b *blocks) AddNewBlock(number uint64, timestamp uint64) error {
	newBlock := &Block{
		Number:    number,
		Timestamp: timestamp,
		State:     NewBlockState,
	}
	if len(b.list) == 0 {
		b.list = []*Block{newBlock}
	} else {
		if number <= b.list[0].Number {
			return ErrTooLowBlockNumber
		}
		if b.list[0].State == NewBlockState {
			b.list[0].State = ProcessedBlockState
		}
		gaps := int64(number) - int64(b.list[0].Number) - 1
		prepend := make([]*Block, gaps+1)
		prepend[0] = newBlock
		for i := 1; i <= int(gaps); i++ {
			prepend[i] = &Block{
				Number: number - uint64(i),
				State:  ProcessedBlockState,
			}
		}
		b.list = append(prepend, b.list...)
		b.truncate()
	}
	return nil
}

func (b *blocks) StartBackfilling(number uint64) error {
	firstBlock := &Block{
		Number: number,
		State:  BackfillingBlockState,
	}
	if len(b.list) == 0 {
		return ErrNoBlocksToBackfill
	} else {
		if number < b.list[0].Number {
			return ErrTooLowBlockNumber
		}
		if number == b.list[0].Number {
			b.list[0].State = BackfillingBlockState
			return nil
		}
		if b.list[0].State == NewBlockState {
			b.list[0].State = BackfillingBlockState
		}
		gaps := int64(number) - int64(b.list[0].Number) - 1
		prepend := make([]*Block, gaps+1)
		prepend[0] = firstBlock
		for i := 1; i <= int(gaps); i++ {
			prepend[i] = &Block{
				Number: number - uint64(i),
				State:  BackfillingBlockState,
			}
		}
		b.list = append(prepend, b.list...)
		b.truncate()
	}
	return nil
}

func (b *blocks) GetNextBackfillingBlockNumber() (uint64, bool) {
	for _, block := range b.list {
		if block.State == BackfillingBlockState {
			return block.Number, true
		}
	}
	return 0, false
}

func (b *blocks) truncate() {
	if len(b.list) > b.depth {
		b.list = b.list[:b.depth]
	}
}
