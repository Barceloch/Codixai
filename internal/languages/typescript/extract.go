package typescript

import "github.com/barceloch/codixai/internal/languages/javascript"

// Import is an import or dependency declaration extracted from TypeScript source.
type Import = javascript.Import

// Symbol is a named declaration extracted from TypeScript source.
type Symbol = javascript.Symbol

// File is the extracted structure of a TypeScript source file.
type File = javascript.File

// Extract parses TypeScript source conservatively and returns imports/exports and symbols.
// It uses lexical segmentation to avoid matching inside strings and comments.
// It does not execute analyzed code.
func Extract(filename string, src []byte) (*File, error) {
	return javascript.ExtractWithProfile(filename, src, Profile())
}
