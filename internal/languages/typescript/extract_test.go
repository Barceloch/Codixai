package typescript_test

import (
	"testing"

	"github.com/barceloch/codixai/internal/languages/typescript"
)

func findSymbol(f *typescript.File, name, kind string) *typescript.Symbol {
	for i := range f.Symbols {
		if f.Symbols[i].Name == name && (kind == "" || f.Symbols[i].Kind == kind) {
			return &f.Symbols[i]
		}
	}
	return nil
}

func findImport(f *typescript.File, module string) *typescript.Import {
	for i := range f.Imports {
		if f.Imports[i].Module == module {
			return &f.Imports[i]
		}
	}
	return nil
}

func TestExtractTSDeclarations(t *testing.T) {
	src := []byte(`
export interface User { id: number; }
export type UserID = string | number;
export enum Status { Active, Disabled }
export class Admin implements User { id = 1; save(): void {} }
export function f(x: string): number { return 1; }
export const V: number = 1;
`)
	f, err := typescript.Extract("a.ts", src)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"User", "UserID", "Status", "Admin"} {
		if findSymbol(f, name, "type") == nil {
			t.Fatalf("missing type symbol %s: %#v", name, f.Symbols)
		}
	}
	if s := findSymbol(f, "save", "method"); s == nil || s.Container != "Admin" {
		t.Fatalf("expected method save in Admin, got %#v", f.Symbols)
	}
	if findSymbol(f, "f", "func") == nil || findSymbol(f, "V", "var") == nil {
		t.Fatalf("missing function/var: %#v", f.Symbols)
	}
}

func TestExtractTSTypeImports(t *testing.T) {
	src := []byte(`
import type { User } from "./models";
import { User as Account } from "./models";
export type { User } from "./models";
`)
	f, err := typescript.Extract("a.ts", src)
	if err != nil {
		t.Fatal(err)
	}
	imp := findImport(f, "./models")
	if imp == nil {
		t.Fatalf("missing import ./models: %#v", f.Imports)
	}
	foundTypeOnly := false
	for i := range f.Imports {
		if f.Imports[i].Module == "./models" && f.Imports[i].TypeOnly {
			foundTypeOnly = true
		}
	}
	if !foundTypeOnly {
		t.Fatalf("expected type-only import for ./models: %#v", f.Imports)
	}
}
