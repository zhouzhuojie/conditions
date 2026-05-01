package conditions

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegexError(t *testing.T) {
	t.Run("invalid regex pattern", func(t *testing.T) {
		expr, err := Parse(`{status} =~ /[/`)
		if err == nil {
			_, err = Evaluate(expr, map[string]interface{}{"status": "test"})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid regex")
		}
	})
}

func TestRegexCacheDoubleCheckLocking(t *testing.T) {
	// Clear cache
	regexCache.Lock()
	regexCache.m = make(map[string]*regexp.Regexp)
	regexCache.Unlock()

	t.Run("concurrent access", func(t *testing.T) {
		done := make(chan struct{})
		for i := 0; i < 10; i++ {
			go func() {
				re, err := getCompiledRegexp(`^test\d+`)
				assert.NoError(t, err)
				assert.NotNil(t, re)
				done <- struct{}{}
			}()
		}
		for i := 0; i < 10; i++ {
			<-done
		}
		// Verify it's cached
		regexCache.RLock()
		assert.Equal(t, 1, len(regexCache.m))
		regexCache.RUnlock()
	})
}

func TestGetCompiledRegexpDoubleCheck(t *testing.T) {
	// Clear the cache
	regexCache.Lock()
	regexCache.m = make(map[string]*regexp.Regexp)
	regexCache.Unlock()

	// First call compiles and caches
	re1, err := getCompiledRegexp(`^test$`)
	assert.NoError(t, err)
	assert.True(t, re1.MatchString("test"))

	// Second call hits cache
	re2, err := getCompiledRegexp(`^test$`)
	assert.NoError(t, err)
	assert.Same(t, re1, re2)

	// Third call with different pattern
	re3, err := getCompiledRegexp(`^other$`)
	assert.NoError(t, err)
	assert.NotSame(t, re1, re3)
	assert.True(t, re3.MatchString("other"))
}
