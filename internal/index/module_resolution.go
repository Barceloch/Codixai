package index

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/barceloch/codixai/internal/domain"
	"github.com/barceloch/codixai/internal/languages/javascript"
)

// ModuleResolutionStatus classifies the outcome of static module resolution.
type ModuleResolutionStatus string

const (
	// ModuleResolved means a unique local destination was demonstrated.
	ModuleResolved ModuleResolutionStatus = ModuleResolutionStatus(domain.ResolutionResolved)
	// ModuleExternal means the dependency targets a builtin or external package.
	ModuleExternal ModuleResolutionStatus = ModuleResolutionStatus(domain.ResolutionExternal)
	// ModuleUnresolved means no reliable local destination was found.
	ModuleUnresolved ModuleResolutionStatus = ModuleResolutionStatus(domain.ResolutionUnresolved)
	// ModuleAmbiguous means multiple plausible local destinations exist.
	ModuleAmbiguous ModuleResolutionStatus = ModuleResolutionStatus(domain.ResolutionAmbiguous)
	// ModuleDynamic means the destination depends on a dynamic/non-literal expression.
	ModuleDynamic ModuleResolutionStatus = ModuleResolutionStatus(domain.ResolutionDynamic)
)

// ModuleDependency is the conservative resolution result for one indexed dependency.
type ModuleDependency struct {
	SourceFile       string                 `json:"source_file"`
	SourceLanguage   string                 `json:"source_language"`
	Specifier        string                 `json:"specifier"`
	Normalized       string                 `json:"normalized,omitempty"`
	Kind             string                 `json:"kind,omitempty"`
	Status           ModuleResolutionStatus `json:"status"`
	ResolvedPath     string                 `json:"resolved_path,omitempty"`
	ResolvedLanguage string                 `json:"resolved_language,omitempty"`
	Candidates       []string               `json:"candidates,omitempty"`
	Evidence         string                 `json:"evidence,omitempty"`
	Line             int                    `json:"line,omitempty"`
}

// ModuleNode represents a module/file in the local module graph.
type ModuleNode struct {
	Path     string `json:"path"`
	Language string `json:"language,omitempty"`
}

// ModuleEdge represents a confirmed dependency edge between local modules.
type ModuleEdge struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Specifier string `json:"specifier"`
	Line      int    `json:"line,omitempty"`
}

// ModuleGraph is a deterministic local dependency graph for JS/TS modules.
type ModuleGraph struct {
	Root         string             `json:"root"`
	Nodes        []ModuleNode       `json:"nodes,omitempty"`
	Edges        []ModuleEdge       `json:"edges,omitempty"`
	Dependencies []ModuleDependency `json:"dependencies,omitempty"`
}

// ModuleResolver resolves JS/TS module dependencies from an index.
type ModuleResolver interface {
	ResolveModules(ctx context.Context, idx *Index, root string) (*ModuleGraph, error)
}

// JSModuleResolver resolves JavaScript and TypeScript module dependencies conservatively.
type JSModuleResolver struct{}

// NewJSModuleResolver creates a new JS/TS module resolver.
func NewJSModuleResolver() *JSModuleResolver { return &JSModuleResolver{} }

// ResolveModules implements ModuleResolver.
func (r *JSModuleResolver) ResolveModules(ctx context.Context, idx *Index, root string) (*ModuleGraph, error) {
	if idx == nil {
		return &ModuleGraph{Root: filepath.ToSlash(root)}, nil
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		absRoot = root
	}
	absRoot = filepath.Clean(absRoot)
	g := &ModuleGraph{Root: filepath.ToSlash(absRoot)}

	fileLang := map[string]string{}
	for _, s := range idx.Symbols() {
		if s.File == "" {
			continue
		}
		rel := relToRoot(absRoot, s.File)
		if rel == "" {
			rel = filepath.ToSlash(s.File)
		}
		if _, ok := fileLang[rel]; !ok && s.Language != "" {
			fileLang[rel] = s.Language
		}
	}
	for _, d := range idx.Dependencies() {
		if d.File == "" {
			continue
		}
		rel := relToRoot(absRoot, d.File)
		if rel == "" {
			rel = filepath.ToSlash(d.File)
		}
		if _, ok := fileLang[rel]; !ok && d.Language != "" {
			fileLang[rel] = d.Language
		}
	}

	for _, d := range idx.Dependencies() {
		if d.Language != domain.LangJavaScript && d.Language != domain.LangTypeScript {
			continue
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		src := relToRoot(absRoot, d.File)
		if src == "" {
			src = filepath.ToSlash(d.File)
		}
		md := r.resolveDependency(absRoot, src, d.Language, d)
		g.Dependencies = append(g.Dependencies, md)
		if md.Status == ModuleResolved && md.ResolvedPath != "" {
			g.Edges = append(g.Edges, ModuleEdge{From: src, To: md.ResolvedPath, Specifier: d.Path, Line: d.Line})
		}
	}

	nodeSet := map[string]string{}
	for p, l := range fileLang {
		if l == domain.LangJavaScript || l == domain.LangTypeScript {
			nodeSet[p] = l
		}
	}
	for _, e := range g.Edges {
		if _, ok := nodeSet[e.From]; !ok {
			nodeSet[e.From] = fileLang[e.From]
		}
		if _, ok := nodeSet[e.To]; !ok {
			nodeSet[e.To] = fileLang[e.To]
		}
	}
	for p, l := range nodeSet {
		g.Nodes = append(g.Nodes, ModuleNode{Path: p, Language: l})
	}

	sort.Slice(g.Nodes, func(i, j int) bool { return g.Nodes[i].Path < g.Nodes[j].Path })
	sort.Slice(g.Edges, func(i, j int) bool {
		if g.Edges[i].From != g.Edges[j].From {
			return g.Edges[i].From < g.Edges[j].From
		}
		if g.Edges[i].To != g.Edges[j].To {
			return g.Edges[i].To < g.Edges[j].To
		}
		if g.Edges[i].Specifier != g.Edges[j].Specifier {
			return g.Edges[i].Specifier < g.Edges[j].Specifier
		}
		return g.Edges[i].Line < g.Edges[j].Line
	})
	sort.Slice(g.Dependencies, func(i, j int) bool {
		a, b := g.Dependencies[i], g.Dependencies[j]
		if a.SourceFile != b.SourceFile {
			return a.SourceFile < b.SourceFile
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.Specifier != b.Specifier {
			return a.Specifier < b.Specifier
		}
		return string(a.Status) < string(b.Status)
	})
	// dedupe identical edges
	if len(g.Edges) > 1 {
		out := g.Edges[:1]
		for i := 1; i < len(g.Edges); i++ {
			p, c := out[len(out)-1], g.Edges[i]
			if p.From == c.From && p.To == c.To && p.Specifier == c.Specifier && p.Line == c.Line {
				continue
			}
			out = append(out, c)
		}
		g.Edges = out
	}
	return g, nil
}

func (r *JSModuleResolver) resolveDependency(absRoot, srcFile, srcLang string, d Dependency) ModuleDependency {
	md := ModuleDependency{
		SourceFile:     srcFile,
		SourceLanguage: srcLang,
		Specifier:      d.Path,
		Kind:           "",
		Line:           d.Line,
	}
	spec := strings.TrimSpace(d.Path)
	md.Normalized = spec
	if spec == "" {
		md.Status = ModuleUnresolved
		md.Evidence = "empty specifier"
		return md
	}
	if spec == "module.exports" {
		md.Status = ModuleExternal
		md.Evidence = "CommonJS module export marker"
		return md
	}
	if javascript.IsBuiltin(spec) {
		md.Status = ModuleExternal
		md.Evidence = "Node.js builtin module"
		return md
	}
	if strings.HasPrefix(spec, ".") {
		return r.resolveRelative(absRoot, srcFile, srcLang, md)
	}
	// Absolute/project-root paths are treated conservatively.
	if strings.HasPrefix(spec, "/") {
		md.Status = ModuleUnresolved
		md.Evidence = "absolute specifier not resolved conservatively"
		return md
	}
	md.Status = ModuleExternal
	md.Evidence = "non-relative specifier classified as external package"
	return md
}

var jsTSResolveExts = []string{".js", ".jsx", ".mjs", ".cjs", ".ts", ".tsx", ".mts", ".cts"}
var jsTSIndexFiles = []string{"index.js", "index.jsx", "index.mjs", "index.cjs", "index.ts", "index.tsx", "index.mts", "index.cts"}

func (r *JSModuleResolver) resolveRelative(absRoot, srcFile, srcLang string, md ModuleDependency) ModuleDependency {
	baseDir := filepath.Dir(filepath.FromSlash(srcFile))
	candidateBase := filepath.Clean(filepath.Join(baseDir, filepath.FromSlash(md.Specifier)))
	// ensure inside root
	absBase := filepath.Clean(filepath.Join(absRoot, candidateBase))
	relBase, err := filepath.Rel(absRoot, absBase)
	if err != nil || relBase == ".." || strings.HasPrefix(relBase, ".."+string(filepath.Separator)) {
		md.Status = ModuleUnresolved
		md.Evidence = "relative specifier escapes project root"
		return md
	}
	md.Normalized = filepath.ToSlash(relBase)

	var candidates []string
	hasExt := filepath.Ext(md.Specifier) != ""
	if hasExt {
		candidates = append(candidates, relBase)
	} else {
		candidates = append(candidates, r.expandCandidates(relBase)...)
	}

	existing := make([]string, 0, len(candidates))
	seen := map[string]bool{}
	for _, c := range candidates {
		rel := filepath.ToSlash(filepath.Clean(c))
		if rel == "." || strings.HasPrefix(rel, "../") || rel == ".." || seen[rel] {
			continue
		}
		seen[rel] = true
		abs := filepath.Join(absRoot, filepath.FromSlash(rel))
		info, err := os.Lstat(abs)
		if err != nil {
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		if info.IsDir() {
			// try index files only when specifier omitted extension / points to dir
			for _, idx := range jsTSIndexFiles {
				irel := filepath.ToSlash(filepath.Join(rel, idx))
				iabs := filepath.Join(absRoot, filepath.FromSlash(irel))
				iinfo, ierr := os.Lstat(iabs)
				if ierr != nil || iinfo.IsDir() || iinfo.Mode()&os.ModeSymlink != 0 {
					continue
				}
				if !seen[irel] {
					seen[irel] = true
					existing = append(existing, irel)
				}
			}
			continue
		}
		existing = append(existing, rel)
	}
	// If explicit extension candidate was a file, existing may contain it; if it was a dir, we handled index above.
	// Also handle case where omitted extension points directly to an existing file with exact path? no, omitted means no ext.
	if len(existing) == 0 && !hasExt {
		// try as directory with index files when relBase exists as dir
		abs := filepath.Join(absRoot, filepath.FromSlash(relBase))
		if info, err := os.Lstat(abs); err == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
			for _, idx := range jsTSIndexFiles {
				irel := filepath.ToSlash(filepath.Join(relBase, idx))
				iabs := filepath.Join(absRoot, filepath.FromSlash(irel))
				iinfo, ierr := os.Lstat(iabs)
				if ierr == nil && !iinfo.IsDir() && iinfo.Mode()&os.ModeSymlink == 0 {
					existing = append(existing, irel)
				}
			}
		}
	}

	sort.Strings(existing)
	existing = uniqueStrings(existing)
	switch len(existing) {
	case 0:
		md.Status = ModuleUnresolved
		md.Evidence = "no local file matched normalized specifier"
	case 1:
		md.Status = ModuleResolved
		md.ResolvedPath = existing[0]
		md.ResolvedLanguage = langFromPath(existing[0])
		md.Evidence = "unique local file matched"
	default:
		md.Status = ModuleAmbiguous
		md.Candidates = existing
		md.Evidence = "multiple plausible local files matched"
	}
	return md
}

func (r *JSModuleResolver) expandCandidates(relBase string) []string {
	out := make([]string, 0, len(jsTSResolveExts)+len(jsTSIndexFiles)+1)
	for _, ext := range jsTSResolveExts {
		out = append(out, relBase+ext)
	}
	// directory index candidates
	for _, idx := range jsTSIndexFiles {
		out = append(out, filepath.ToSlash(filepath.Join(relBase, idx)))
	}
	return out
}

func langFromPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".ts", ".tsx", ".mts", ".cts":
		return domain.LangTypeScript
	case ".js", ".jsx", ".mjs", ".cjs":
		return domain.LangJavaScript
	default:
		return ""
	}
}

func uniqueStrings(in []string) []string {
	if len(in) < 2 {
		return in
	}
	out := in[:1]
	for i := 1; i < len(in); i++ {
		if in[i] != out[len(out)-1] {
			out = append(out, in[i])
		}
	}
	return out
}
