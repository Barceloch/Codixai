package policies

import (
	"github.com/barceloch/codixai/internal/glob"
)

// MatchGlob reports whether a repository relative path matches a glob
// pattern. It is the helper used to decide whether a file is excluded from
// the analysis.
//
// The matcher lives in the glob package so that the claims of a claim file
// share the same syntax without depending on the configuration of the
// repository.
func MatchGlob(pattern, path string) (bool, error) {
	return glob.Match(pattern, path)
}
