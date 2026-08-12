// Copyright 2026
// license that can be found in the LICENSE file.

package strings

func SortFuncAsc(i, j string) int {
	if i == j {
		return 0
	}

	if i < j {
		return -1
	}

	return 1
}

func SortFuncDesc(i, j string) int {
	if i == j {
		return 0
	}

	if i < j {
		return 1
	}

	return -1
}
