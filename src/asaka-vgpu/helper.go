package main

import (
	"hash/fnv"
	"io"
	"sort"
	"strings"
)

func hash(val string) int {
	hasher := fnv.New32a()
	io.WriteString(hasher, val)
	return int(hasher.Sum32())
}

func StringsToHash(s []string) int {
	c := make([]string, len(s))
	copy(c, s)
	sort.Strings(c)
	return hash(strings.Join(c, ","))
}
