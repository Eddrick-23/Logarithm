package pool

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPool_GetCreatesNewWhenEmpty(t *testing.T) {
	count := 0
	testPool := New(1, func() int {
		count++
		return 7
	})

	val := testPool.Get()

	assert.Equal(t, 7, val, "wrong value on Get")
	assert.Equal(t, 1, count, "expected newFunc to be called once")
}

func TestPool_GetThenPut(t *testing.T) {
	testPool := New(1, func() int { return 7 })

	val := testPool.Get()
	ok := testPool.Put(val)

	assert.True(t, ok, "expected successful put")
}

type testItem struct {
	id int
}

func TestPool_PutThenGetReusesItem(t *testing.T) {
	count := 0
	testPool := New(1, func() *testItem {
		count++
		return &testItem{
			id: count,
		}
	})

	first := testPool.Get()
	ok := testPool.Put(first)
	require.True(t, ok, "expected successful put")

	second := testPool.Get()

	assert.Equal(t, first, second)
	assert.Equal(t, 1, count, "newFunc should only be called once")
}

func TestPool_PutDropsWhenChannelFull(t *testing.T) {
	testPool := New(1, func() int { return 7 })

	val := testPool.Get()

	put1 := testPool.Put(val)
	put2 := testPool.Put(val)

	assert.True(t, put1, "first put should be successful")
	assert.False(t, put2, "second put should fail as capacity reached")
}

func TestPool_ReuseFunction(t *testing.T) {
	newFunc := func() int {
		return 7
	}

	tests := []struct {
		name           string
		reuseFunc      func(int) bool
		expectedResult bool
	}{
		{
			name:           "reuse not provided",
			reuseFunc:      nil,
			expectedResult: true,
		},
		{
			name:           "reuse returns true",
			reuseFunc:      func(i int) bool { return true },
			expectedResult: true,
		},
		{
			name:           "reuse returns false",
			reuseFunc:      func(i int) bool { return false },
			expectedResult: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testPool := New(1, newFunc, WithReuseCheck(tc.reuseFunc))

			val := testPool.Get()
			ok := testPool.Put(val)

			assert.Equal(t, tc.expectedResult, ok)
		})
	}
}
