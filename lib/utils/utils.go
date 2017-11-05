package utils

import (
	"sort"
	"strings"
)

func StrKeys(strMap map[string]bool) (keys []string) {
	for k := range strMap {
		keys = append(keys, k)
	}
	return
}

func StrMap(strs []string) map[string]bool {
	m := map[string]bool{}
	for _, str := range strs {
		m[str] = true
	}
	return m
}

func ShortestPrefixes(strs []string) (prefixes []string) {
	sorted := make([]string, len(strs))
	copy(sorted, strs)
	sort.Strings(sorted)

	var lastPrefix string
	for _, p := range sorted {
		if lastPrefix != "" && strings.HasPrefix(p, lastPrefix) {
			continue
		}
		lastPrefix = p
		prefixes = append(prefixes, p)
	}
	return
}

func SortStrings(strs []string) []string {
	sorted := make([]string, len(strs))
	copy(sorted, strs)
	sort.Strings(sorted)
	return sorted
}
