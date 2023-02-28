package app_test

import (
	"math/big"
	"testing"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/pinebit/lognite/app"
	"github.com/stretchr/testify/assert"
)

func TestHeaders(t *testing.T) {
	t.Parallel()

	hs := app.NewHeaders()

	h := hs.FindHeaderByNumber(0)
	assert.Nil(t, h)

	disorder := []int{500, 100, 300, 200, 700}
	for _, n := range disorder {
		header := &ethtypes.Header{
			Number: big.NewInt(int64(n)),
			Time:   uint64(n),
		}
		hs.AddHeader(header)
	}

	for _, n := range disorder {
		h := hs.FindHeaderByNumber(uint64(n))
		assert.NotNil(t, h)
		assert.Equal(t, int64(n), h.Number.Int64())
	}

	assert.Nil(t, hs.FindHeaderByNumber(400))

	hs.TrimByTimestamp(300)

	for _, n := range disorder {
		h := hs.FindHeaderByNumber(uint64(n))
		if n <= 300 {
			assert.Nil(t, h)
		} else {
			assert.NotNil(t, h)
			assert.Equal(t, int64(n), h.Number.Int64())
		}
	}

	hs.TrimByTimestamp(1000)

	h = hs.FindHeaderByNumber(500)
	assert.Nil(t, h)
}
