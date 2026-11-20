package javascript_test

import (
	"testing"

	"github.com/barceloch/codixai/internal/languages/javascript"
)

func findSymbol(f *javascript.File, name, kind string) *javascript.Symbol {
	for i := range f.Symbols {
		if f.Symbols[i].Name == name && (kind == "" || f.Symbols[i].Kind == kind) {
			return &f.Symbols[i]
		}
	}
	return nil
}

func findImport(f *javascript.File, module, kind string) *javascript.Import {
	for i := range f.Imports {
		if f.Imports[i].Module == module && (kind == "" || f.Imports[i].Kind == kind) {
			return &f.Imports[i]
		}
	}
	return nil
}

func TestExtractFunctionsClassesVars(t *testing.T) {
	src := []byte(`
export function hello(name) { return name; }
export default function main() {}
const add = (a,b) => a+b;
export class User {
  constructor(name) { this.name = name; }
  save() {}
}
let count = 1;
`)
	f, err := javascript.Extract("a.js", src)
	if err != nil {
		t.Fatal(err)
	}
	if s := findSymbol(f, "hello", "func"); s == nil || !s.Exported {
		t.Fatalf("expected exported func hello, got %#v", f.Symbols)
	}
	if s := findSymbol(f, "main", "func"); s == nil || !s.Exported {
		t.Fatalf("expected exported default func main, got %#v", f.Symbols)
	}
	if s := findSymbol(f, "add", "func"); s == nil {
		t.Fatalf("expected arrow func add, got %#v", f.Symbols)
	}
	if s := findSymbol(f, "User", "type"); s == nil || !s.Exported {
		t.Fatalf("expected exported class User, got %#v", f.Symbols)
	}
	if s := findSymbol(f, "save", "method"); s == nil || s.Container != "User" {
		t.Fatalf("expected method save in User, got %#v", f.Symbols)
	}
	if s := findSymbol(f, "count", "var"); s == nil {
		t.Fatalf("expected var count, got %#v", f.Symbols)
	}
}

func TestExtractImportsExports(t *testing.T) {
	src := []byte(`
import value from "./module.js";
import { foo, bar as baz } from "./mod2.js";
import * as ns from "pkg";
import "./setup.js";
export { foo } from "./reexport.js";
export * from "./all.js";
const mod = require("./cjs.js");
module.exports = {};
exports.util = util;
`)
	f, err := javascript.Extract("a.js", src)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		mod  string
		kind string
	}{
		{"./module.js", "esm_import"},
		{"./mod2.js", "esm_import"},
		{"pkg", "esm_import"},
		{"./setup.js", "esm_import"},
		{"./reexport.js", "esm_export"},
		{"./all.js", "esm_export"},
		{"./cjs.js", "require"},
	}
	for _, c := range cases {
		if imp := findImport(f, c.mod, c.kind); imp == nil {
			t.Fatalf("missing import/export %q kind %q: %#v", c.mod, c.kind, f.Imports)
		}
	}
}

func TestExtractDeterministicAndPositions(t *testing.T) {
	src := []byte("export function a() {}\nexport function b() {}\n")
	f1, err := javascript.Extract("a.js", src)
	if err != nil {
		t.Fatal(err)
	}
	f2, err := javascript.Extract("a.js", src)
	if err != nil {
		t.Fatal(err)
	}
	if len(f1.Symbols) != len(f2.Symbols) || len(f1.Imports) != len(f2.Imports) {
		t.Fatalf("non-deterministic lengths")
	}
	for i := range f1.Symbols {
		if f1.Symbols[i] != f2.Symbols[i] {
			t.Fatalf("non-deterministic symbol %d", i)
		}
	}
	if s := findSymbol(f1, "a", "func"); s == nil || s.Line != 1 {
		t.Fatalf("expected a on line 1, got %#v", s)
	}
}

func TestExtractIgnoresStringsComments(t *testing.T) {
	src := []byte(`
const x = "import {fake} from 'nope'; function notReal(){}";
 // export function commented(){}
/* import z from "nope2"; */
export function real() {}
`)
	f, err := javascript.Extract("a.js", src)
	if err != nil {
		t.Fatal(err)
	}
	if findSymbol(f, "notReal", "") != nil || findSymbol(f, "commented", "") != nil {
		t.Fatalf("extracted symbol from string/comment: %#v", f.Symbols)
	}
	if findImport(f, "nope", "") != nil || findImport(f, "nope2", "") != nil {
		t.Fatalf("extracted import from string/comment: %#v", f.Imports)
	}
	if findSymbol(f, "real", "func") == nil {
		t.Fatalf("missing real symbol: %#v", f.Symbols)
	}
}
