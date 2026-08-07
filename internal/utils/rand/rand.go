package rand

import (
	mathrand "math/rand"
	"time"
)

var lettersRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func String(n int) string {
	randomizer := getRand()

	b := make([]rune, n)
	for i := range b {
		b[i] = lettersRunes[randomizer.Intn(len(lettersRunes))]
	}

	return string(b)
}

func getRand() *mathrand.Rand {
	return mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
}
