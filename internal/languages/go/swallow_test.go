package golang_test

import (
	"sort"
	"testing"

	golang "github.com/barceloch/codixai/internal/languages/go"
)

// TestFindBlankErrorAssignments covers the detection of a call whose result is
// discarded to the blank identifier when the declaration of the callee states
// that it returns an error.
//
// The detection is deliberately narrow: it reports only the calls whose
// function is declared in the package directory and whose result list names
// the predeclared "error" type. For any other call the analysis has no type
// information, so it abstains instead of assuming that the value is an error.
func TestFindBlankErrorAssignments(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		siblings []golang.SiblingFile
		want     []string
	}{
		{
			name: "discarded error of a function declared in the file",
			source: `package main

func doWork() error { return nil }

func main() {
	_ = doWork()
}
`,
			want: []string{"doWork"},
		},
		{
			name: "discarded error of a function declared in a sibling file",
			source: `package main

func main() {
	_ = save()
}
`,
			siblings: []golang.SiblingFile{
				{Name: "store.go", Src: []byte("package main\n\nfunc save() error { return nil }\n")},
			},
			want: []string{"save"},
		},
		{
			name: "discarded error of the second result",
			source: `package main

func read() (string, error) { return "", nil }

func main() {
	_, _ = read()
}
`,
			want: []string{"read"},
		},
		{
			name: "value that is not an error",
			source: `package main

func count() int { return 0 }

func main() {
	_ = count()
}
`,
			want: nil,
		},
		{
			name: "error that is handled",
			source: `package main

func doWork() error { return nil }

func main() {
	if err := doWork(); err != nil {
		return
	}
}
`,
			want: nil,
		},
		{
			name: "error that is assigned and used",
			source: `package main

func doWork() error { return nil }

func main() {
	err := doWork()
	_ = err
}
`,
			want: nil,
		},
		{
			name: "call to a function of another package",
			source: `package main

import "example.com/dependency"

func main() {
	_ = dependency.Save()
}
`,
			want: nil,
		},
		{
			name: "call to a function without a declaration in the package",
			source: `package main

func main() {
	_ = unknownFunction()
}
`,
			want: nil,
		},
		{
			name: "discarded value that is not a call",
			source: `package main

func main() {
	_ = 42
}
`,
			want: nil,
		},
		{
			name: "named error result",
			source: `package main

func doWork() (err error) { return nil }

func main() {
	_ = doWork()
}
`,
			want: []string{"doWork"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := golang.FindBlankErrorAssignments("main.go", []byte(test.source), test.siblings)
			if err != nil {
				t.Fatalf("the analysis returned an error: %v", err)
			}
			names := make([]string, 0, len(got))
			lines := make([]int, 0, len(got))
			for _, assignment := range got {
				names = append(names, assignment.Callee)
				lines = append(lines, assignment.Line)
			}
			sort.Strings(names)
			want := append([]string(nil), test.want...)
			sort.Strings(want)
			if len(names) != len(want) {
				t.Fatalf("the assignments are %v, want %v", names, want)
			}
			for index := range want {
				if names[index] != want[index] {
					t.Fatalf("the assignments are %v, want %v", names, want)
				}
			}
			for _, line := range lines {
				if line <= 0 {
					t.Errorf("the assignment has a non positive line: %d", line)
				}
			}
		})
	}
}
