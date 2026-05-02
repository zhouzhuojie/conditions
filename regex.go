package conditions

import (
	"regexp"
	"sync"
)

// regexCache caches compiled regex patterns to avoid recompilation on every call.
var regexCache = struct {
	sync.RWMutex
	m map[string]*regexp.Regexp
}{m: make(map[string]*regexp.Regexp)}

func getCompiledRegexp(pattern string) (*regexp.Regexp, error) {
	regexCache.RLock()
	re, ok := regexCache.m[pattern]
	regexCache.RUnlock()
	if ok {
		return re, nil
	}

	regexCache.Lock()
	defer regexCache.Unlock()
	// Double-check after acquiring write lock
	if re, ok := regexCache.m[pattern]; ok {
		return re, nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	regexCache.m[pattern] = re
	return re, nil
}
