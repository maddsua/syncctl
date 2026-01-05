package main

import (
	"os"
	"strconv"
)

func EnvIntOr(key string, orVal int) int {
	if val, err := strconv.Atoi(os.Getenv(key)); err == nil {
		return val
	}
	return orVal
}
