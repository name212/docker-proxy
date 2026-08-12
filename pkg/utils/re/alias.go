// Copyright 2026
// license that can be found in the LICENSE file.

package re

import (
	"fmt"
	"regexp"
)

type Regexp struct {
	*regexp.Regexp
}

func (r *Regexp) UnmarshalText(text []byte) error {
	expr := string(text)
	re, err := regexp.Compile(expr)
	if err != nil {
		return fmt.Errorf("invalid regex pattern '%s': %w", expr, err)
	}
	r.Regexp = re
	return nil
}

func (r *Regexp) Clone() *Regexp {
	if r == nil {
		panic("cannot clone nil Regexp")
	}

	rr := regexp.MustCompile(r.String())

	return &Regexp{
		Regexp: rr,
	}
}

func MustCompile(str string) *Regexp {
	r := regexp.MustCompile(str)
	return &Regexp{
		Regexp: r,
	}
}
