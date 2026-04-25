package easy

import (
	easy "leetcode/src/easy"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test(t *testing.T) {
	nums := []int{2, 7, 11, 15}
	target := 9
	assert.Equal(t, []int{0, 1}, easy.TwoSum1(nums, target))
}
