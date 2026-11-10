// Package index implements the structural indexing of a repository.
package index

import (
	"sort"
	"strings"

	"github.com/barceloch/codixai/internal/domain"
)

// NormalizeEvidence returns a deterministically sorted copy of evidence with
// semantically identical duplicates removed. Distinct evidence records that
// differ in meaningful provenance details are preserved.
func NormalizeEvidence(evidence []domain.ReferenceEvidence) []domain.ReferenceEvidence {
	if len(evidence) == 0 {
		return evidence
	}
	out := append([]domain.ReferenceEvidence(nil), evidence...)
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Ref != b.Ref {
			return a.Ref < b.Ref
		}
		if a.Detail != b.Detail {
			return a.Detail < b.Detail
		}
		return a.Method < b.Method
	})
	deduped := out[:0]
	for _, ev := range out {
		if len(deduped) > 0 && evidenceEqual(deduped[len(deduped)-1], ev) {
			continue
		}
		deduped = append(deduped, ev)
	}
	if len(deduped) == 0 {
		return nil
	}
	return deduped
}

func evidenceEqual(a, b domain.ReferenceEvidence) bool {
	return a.Kind == b.Kind && a.Ref == b.Ref && a.Detail == b.Detail && a.Method == b.Method
}

// NormalizeCandidates returns a deterministically sorted copy of candidates with
// identical duplicate symbol refs removed.
func NormalizeCandidates(candidates []domain.SymbolRef) []domain.SymbolRef {
	if len(candidates) == 0 {
		return candidates
	}
	out := append([]domain.SymbolRef(nil), candidates...)
	sort.Slice(out, func(i, j int) bool {
		return symbolRefLess(out[i], out[j])
	})
	deduped := out[:0]
	for _, c := range out {
		if len(deduped) > 0 && symbolRefEqual(deduped[len(deduped)-1], c) {
			continue
		}
		deduped = append(deduped, c)
	}
	if len(deduped) == 0 {
		return nil
	}
	return deduped
}

func symbolRefLess(a, b domain.SymbolRef) bool {
	if a.File != b.File {
		return a.File < b.File
	}
	if a.Line != b.Line {
		return a.Line < b.Line
	}
	if a.Column != b.Column {
		return a.Column < b.Column
	}
	if a.Name != b.Name {
		return a.Name < b.Name
	}
	if a.Kind != b.Kind {
		return a.Kind < b.Kind
	}
	if a.Language != b.Language {
		return a.Language < b.Language
	}
	if a.Container != b.Container {
		return a.Container < b.Container
	}
	if a.Signature != b.Signature {
		return a.Signature < b.Signature
	}
	return a.SignatureHash < b.SignatureHash
}

func symbolRefEqual(a, b domain.SymbolRef) bool {
	if a.Name != b.Name || a.Kind != b.Kind || a.File != b.File || a.Line != b.Line || a.Column != b.Column || a.Language != b.Language || a.Container != b.Container || a.Signature != b.Signature || a.SignatureHash != b.SignatureHash {
		return false
	}
	if a.Exported == nil && b.Exported == nil {
		return true
	}
	if a.Exported == nil || b.Exported == nil {
		return false
	}
	return *a.Exported == *b.Exported
}

func boolPtrEqual(a, b *bool) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// NormalizeReference normalizes nested evidence/candidates within a reference.
func NormalizeReference(ref domain.Reference) domain.Reference {
	ref.Evidence = NormalizeEvidence(ref.Evidence)
	ref.Candidates = NormalizeCandidates(ref.Candidates)
	return ref
}

// SortReferences returns a deterministically sorted copy of references using the
// canonical logical key for consumer-visible ordering.
func SortReferences(refs []domain.Reference) []domain.Reference {
	if len(refs) == 0 {
		return refs
	}
	out := append([]domain.Reference(nil), refs...)
	sort.Slice(out, func(i, j int) bool {
		return referenceLess(out[i], out[j])
	})
	return out
}

func referenceLess(a, b domain.Reference) bool {
	if a.Location.Path != b.Location.Path {
		return a.Location.Path < b.Location.Path
	}
	if a.Location.StartLine != b.Location.StartLine {
		return a.Location.StartLine < b.Location.StartLine
	}
	if a.Location.StartColumn != b.Location.StartColumn {
		return a.Location.StartColumn < b.Location.StartColumn
	}
	if a.Source.Name != b.Source.Name {
		return a.Source.Name < b.Source.Name
	}
	if a.Kind != b.Kind {
		return a.Kind < b.Kind
	}
	if a.Language != b.Language {
		return a.Language < b.Language
	}
	if a.Resolution != b.Resolution {
		return a.Resolution < b.Resolution
	}
	if a.Source.File != b.Source.File {
		return a.Source.File < b.Source.File
	}
	if a.Source.Container != b.Source.Container {
		return a.Source.Container < b.Source.Container
	}
	if a.ID != b.ID {
		return a.ID < b.ID
	}
	// Final stable tie-breaker over semantic payload signatures.
	return referenceSemanticSignature(a) < referenceSemanticSignature(b)
}

// NormalizeReferences normalizes each reference, removes semantically identical
// duplicate references conservatively, and returns deterministic logical order.
func NormalizeReferences(refs []domain.Reference) []domain.Reference {
	if len(refs) == 0 {
		return refs
	}
	normalized := make([]domain.Reference, 0, len(refs))
	for _, ref := range refs {
		normalized = append(normalized, NormalizeReference(ref))
	}
	sort.Slice(normalized, func(i, j int) bool {
		return referenceLess(normalized[i], normalized[j])
	})
	deduped := normalized[:0]
	seen := make(map[string]bool, len(normalized))
	for _, ref := range normalized {
		key := referenceSemanticIdentity(ref)
		if seen[key] {
			continue
		}
		seen[key] = true
		deduped = append(deduped, ref)
	}
	if len(deduped) == 0 {
		return nil
	}
	return deduped
}

func referenceSemanticIdentity(ref domain.Reference) string {
	var b strings.Builder
	writeField := func(v string) {
		b.WriteString(v)
		b.WriteByte(0)
	}
	writeInt := func(v int) {
		b.WriteString(strings.TrimSpace(fmtInt(v)))
		b.WriteByte(0)
	}
	writeField(ref.Location.Path)
	writeInt(ref.Location.StartLine)
	writeInt(ref.Location.StartColumn)
	writeInt(ref.Location.EndLine)
	writeInt(ref.Location.EndColumn)
	writeField(ref.Source.Name)
	writeField(ref.Source.File)
	writeInt(ref.Source.Line)
	writeInt(ref.Source.Column)
	writeField(ref.Source.Container)
	writeField(ref.Source.Kind)
	writeField(ref.Source.Language)
	writeField(ref.Source.Signature)
	writeField(ref.Source.SignatureHash)
	writeField(boolPtrString(ref.Source.Exported))
	writeField(string(ref.Kind))
	writeField(ref.Language)
	writeField(string(ref.Resolution))
	if ref.Target == nil {
		writeField("<nil-target>")
	} else {
		writeField("target")
		writeField(ref.Target.Name)
		writeField(ref.Target.File)
		writeInt(ref.Target.Line)
		writeInt(ref.Target.Column)
		writeField(ref.Target.Container)
		writeField(ref.Target.Kind)
		writeField(ref.Target.Language)
		writeField(ref.Target.Signature)
		writeField(ref.Target.SignatureHash)
		writeField(boolPtrString(ref.Target.Exported))
	}
	writeField(fmtInt(len(ref.Candidates)))
	for _, c := range ref.Candidates {
		writeField(c.Name)
		writeField(c.File)
		writeInt(c.Line)
		writeInt(c.Column)
		writeField(c.Container)
		writeField(c.Kind)
		writeField(c.Language)
		writeField(c.Signature)
		writeField(c.SignatureHash)
		writeField(boolPtrString(c.Exported))
	}
	writeField(fmtInt(len(ref.Evidence)))
	for _, ev := range ref.Evidence {
		writeField(string(ev.Kind))
		writeField(ev.Ref)
		writeField(ev.Detail)
		writeField(ev.Method)
	}
	return b.String()
}

func referenceSemanticSignature(ref domain.Reference) string {
	return referenceSemanticIdentity(ref)
}

func boolPtrString(v *bool) string {
	if v == nil {
		return "<nil>"
	}
	if *v {
		return "true"
	}
	return "false"
}

func fmtInt(v int) string {
	// small helper to avoid strconv import churn in multiple places
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	buf := [20]byte{}
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
