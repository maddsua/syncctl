package utils

import (
	"os"
	"strconv"
)

func SelectValue[T any](filter func(val T) bool, opts ...T) (val T) {
	for _, val := range opts {
		if filter(val) {
			return val
		}
	}
	return
}

func EnvInt(key string) int {
	val, _ := strconv.Atoi(os.Getenv(key))
	return val
}
