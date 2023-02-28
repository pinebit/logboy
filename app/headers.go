package app

import (
	"math/big"
	"sort"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

type Headers interface {
	AddHeader(header *ethtypes.Header)
	FindHeaderByNumber(number uint64) *ethtypes.Header
	TrimByTimestamp(timestamp uint64)
}

// Headers are ordered by block number.
// First header has the highest block number.
type headers struct {
	list []*ethtypes.Header
}

func NewHeaders() Headers {
	return &headers{
		list: []*ethtypes.Header{},
	}
}

func (h *headers) AddHeader(header *ethtypes.Header) {
	// Shall be O(1) in average, given we usually receive a higher block.
	// Worst case O(N) in case of a deep re-org.
	for i := 0; i < len(h.list); i++ {
		r := h.list[i].Number.Cmp(header.Number)
		if r > 0 {
			continue
		}
		if r < 0 {
			last := len(h.list) - 1
			h.list = append(h.list, h.list[last])
			copy(h.list[i+1:], h.list[i:last])
			h.list[i] = header
		}
		return
	}
	h.list = append(h.list, header)
}

func (h headers) FindHeaderByNumber(number uint64) *ethtypes.Header {
	hlen := len(h.list)
	if hlen == 0 {
		return nil
	}
	bn := big.NewInt(int64(number))
	// Most requests will take the most recent block, hence O(1)
	if hlen > 0 && h.list[0].Number.Cmp(bn) == 0 {
		return h.list[0]
	}
	// In case of re-orgs or reconnects, search will be O(logN)
	i := sort.Search(hlen, func(i int) bool {
		return h.list[i].Number.Cmp(bn) <= 0
	})
	if i < hlen && h.list[i].Number.Cmp(bn) == 0 {
		return h.list[i]
	}
	return nil
}

func (h *headers) TrimByTimestamp(timestamp uint64) {
	i := sort.Search(len(h.list), func(i int) bool {
		return h.list[i].Time <= timestamp
	})
	if i < len(h.list) {
		h.list = h.list[0:i]
	}
}
