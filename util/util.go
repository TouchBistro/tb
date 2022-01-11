package util

import (
	"runtime"
)

const IsMacOS = runtime.GOOS == "darwin"

func UniqueStrings(s []string) []string {
	set := make(map[string]bool)
	var us []string
	for _, v := range s {
		if ok := set[v]; ok {
			continue
		}
		set[v] = true
		us = append(us, v)
	}
	return us
}
