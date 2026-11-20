package python

import (
	"testing"
)

func TestExtractSymbols(t *testing.T) {
	src := []byte(`CONFIG = {}

class User:
    name = ""
    age: int = 0

    def __init__(self, email):
        self.email = email
        self.name = ""

    def save(self):
        pass

def run():
    pass
`)
	f, err := Extract("app/models.py", src)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]string{}
	for _, sym := range f.Symbols {
		found[sym.Name] = sym.Kind + "|" + sym.Container
	}
	cases := map[string]string{
		"CONFIG": "var|",
		"User":   "type|",
		"name":   "field|User",
		"age":    "field|User",
		"email":  "field|User",
		"save":   "method|User",
		"run":    "func|",
	}
	for name, want := range cases {
		got, ok := found[name]
		if !ok {
			t.Fatalf("symbol %q not found in %#v", name, found)
		}
		if got != want {
			t.Fatalf("symbol %q = %q, want %q", name, got, want)
		}
	}
}

func TestExtractImports(t *testing.T) {
	src := []byte(`import os
import app.models as models
from . import util
from ..core import db as database
from app.models import User, Repo as R
from app.models import *
`)
	f, err := Extract("app/services/x.py", src)
	if err != nil {
		t.Fatal(err)
	}
	type key struct {
		module string
		name   string
		alias  string
		level  int
	}
	got := map[key]bool{}
	for _, imp := range f.Imports {
		got[key{imp.Module, imp.Name, imp.Alias, imp.Level}] = true
	}
	want := []key{
		{"os", "", "", 0},
		{"app.models", "", "models", 0},
		{"", "util", "", 1},
		{"core", "db", "database", 2},
		{"app.models", "User", "", 0},
		{"app.models", "Repo", "R", 0},
		{"app.models", "*", "", 0},
	}
	for _, w := range want {
		if !got[w] {
			t.Fatalf("import %#v not found in %#v", w, got)
		}
	}
}
