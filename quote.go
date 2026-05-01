package conditions

import (
	"regexp"
	"strings"
)

// quoteIdentRe is pre-compiled to avoid recompilation on every call.
var quoteIdentRe = regexp.MustCompile(`[^a-zA-Z_.]`)

// quoteReplacer is pre-allocated to avoid creating a new Replacer on every Quote call.
var quoteReplacer = strings.NewReplacer("\n", `\n`, `\`, `\\`, `"`, `\"`)

// Quote returns a quoted string.
func Quote(s string) string {
	return `"` + quoteReplacer.Replace(s) + `"`
}

// QuoteIdent returns a quoted identifier if the identifier requires quoting.
// Otherwise returns the original string passed in.
func QuoteIdent(s string) string {
	if s == "" || quoteIdentRe.MatchString(s) {
		return Quote(s)
	}
	return s
}
