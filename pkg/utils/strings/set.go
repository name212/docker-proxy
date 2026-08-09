package strings

import (
	"fmt"
	"slices"
)

type Set map[string]struct{}

func NewSet(cpy ...Set) Set {
	r := make(Set)

	for _, s := range cpy {
		for k := range s {
			r.Add(k)
		}
	}

	return r
}

func NewSetFromSlice(s []string) Set {
	r := make(Set, len(s))

	for _, k := range s {
		r.Add(k)
	}

	return r
}

func NewSetFromSliceStructs[T any](s []T, consumer func(T) string) Set {
	r := make(Set, len(s))

	for _, o := range s {
		k := consumer(o)
		r.Add(k)
	}

	return r
}

func NewSetFromMap[T any](s map[string]T) Set {
	r := make(Set, len(s))

	for k := range s {
		r.Add(k)
	}

	return r
}

func (s Set) Add(k string) {
	s[k] = struct{}{}
}

func (s Set) AddAll(toAdd ...Set) {
	for _, a := range toAdd {
		for k := range a {
			s.Add(k)
		}
	}
}

func (s Set) Remove(k string) {
	delete(s, k)
}

func (s Set) Clean() {
	for k := range s {
		s.Remove(k)
	}
}

func (s Set) Has(k string) bool {
	_, ok := s[k]

	return ok
}

func (s Set) Len() int {
	return len(s)
}

func (s Set) Empty() bool {
	return s.Len() == 0
}

func (s Set) Keys(sorted ...bool) []string {
	keys := make([]string, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}

	if len(sorted) > 0 {
		sortFunc := SortFuncAsc
		if sorted[0] {
			sortFunc = SortFuncDesc
		}

		slices.SortStableFunc(keys, sortFunc)
	}

	return keys
}

func (s Set) String() string {
	return fmt.Sprintf("%v", s.Keys(false))
}
