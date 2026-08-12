// Copyright 2026
// license that can be found in the LICENSE file.

package strings

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSortAsc(t *testing.T) {
	t.Run("Sorted", func(t *testing.T) {
		s := []string{"a", "b", "c", "d", "e"}

		expected := make([]string, len(s))
		copy(expected, s)

		slices.SortFunc(s, SortFuncAsc)

		require.EqualValues(t, expected, s)
	})

	t.Run("Not sorted reverse", func(t *testing.T) {
		s := []string{"e", "d", "c", "b", "a"}

		expected := []string{"a", "b", "c", "d", "e"}

		slices.SortFunc(s, SortFuncAsc)

		require.EqualValues(t, expected, s)
	})

	t.Run("Not sorted random", func(t *testing.T) {
		s := []string{"a", "b", "e", "c", "d"}

		expected := []string{"a", "b", "c", "d", "e"}

		slices.SortFunc(s, SortFuncAsc)

		require.EqualValues(t, expected, s)
	})
}

func TestSortDesc(t *testing.T) {
	t.Run("Sorted", func(t *testing.T) {
		s := []string{"e", "d", "c", "b", "a"}

		expected := make([]string, len(s))
		copy(expected, s)

		slices.SortFunc(s, SortFuncDesc)

		require.EqualValues(t, expected, s)
	})

	t.Run("Not sorted reverse", func(t *testing.T) {
		s := []string{"a", "b", "c", "d", "e"}

		expected := []string{"e", "d", "c", "b", "a"}

		slices.SortFunc(s, SortFuncDesc)

		require.EqualValues(t, expected, s)
	})

	t.Run("Not sorted random", func(t *testing.T) {
		s := []string{"e", "d", "a", "c", "b"}

		expected := []string{"e", "d", "c", "b", "a"}

		slices.SortFunc(s, SortFuncDesc)

		require.EqualValues(t, expected, s)
	})
}
