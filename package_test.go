package have

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestCompilePackageSimple(t *testing.T) {
	f1 := NewFile(
		"hello.hav",
		`package main
func main():
	pass`,
		nil,
	)

	pkg := NewPackage("main", f1)

	fmt.Printf("ZZZ errs %#v\n---\n%s\n---\n", pkg.ParseAndCheck(), f1.GenerateCode())
}

func TestCompilePackageUnorderedBinding(t *testing.T) {
	f1 := NewFile(
		"hello.hav",
		`package main
func main():
	var x = y
var y = 10`,
		nil,
	)

	pkg := NewPackage("main", f1)

	fmt.Printf("ZZZ errs %s\n---\n%s\n---\n", spew.Sdump(pkg.ParseAndCheck()), f1.GenerateCode())
}

func TestCompilePackageDependentFiles(t *testing.T) {
	f1 := NewFile(
		"hello.hav",
		`package main
func main():
	var x = y`,
		nil,
	)

	f2 := NewFile(
		"world.hav",
		`package main
var y = 10`,
		nil,
	)

	pkg := NewPackage("main", f1, f2)

	fmt.Printf("ZZZ errs %s\n---\n%s\n---\n%s\n---\n",
		spew.Sdump(pkg.ParseAndCheck()), f1.GenerateCode(), f2.GenerateCode())
}

type testStmt struct {
	name  string
	decls []string
}

func (t testStmt) Pos() int        { return 0 }
func (t testStmt) Label() *Object  { return nil }
func (t testStmt) Decls() []string { return t.decls }

func TestStmtsSort(t *testing.T) {
	type node struct {
		name        string
		decls, deps []string
	}

	var cases = []struct {
		src        []node
		goals      [][]string
		shouldFail bool
	}{
		{
			src: []node{
				{name: "1",
					decls: []string{"a"},
					deps:  []string{"b"},
				},
				{name: "2",
					decls: []string{"b"},
					deps:  []string{"c"},
				},
				{name: "3",
					decls: []string{"c"},
					deps:  []string{},
				},
			},
			goals:      [][]string{{"3", "2", "1"}},
			shouldFail: false,
		},
		{
			src: []node{
				{name: "1",
					decls: []string{"a"},
					deps:  []string{},
				},
				{name: "2",
					decls: []string{"b"},
					deps:  []string{"a"},
				},
				{name: "3",
					decls: []string{"c"},
					deps:  []string{"b"},
				},
			},
			goals:      [][]string{{"1", "2", "3"}},
			shouldFail: false,
		},
		{
			src: []node{
				{name: "1",
					decls: []string{"a"},
					deps:  []string{},
				},
				{name: "2",
					decls: []string{"b"},
					deps:  []string{"a"},
				},
				{name: "3",
					decls: []string{"c"},
					deps:  []string{"a"},
				},
			},
			goals:      [][]string{{"1", "2", "3"}, {"1", "3", "2"}},
			shouldFail: false,
		},
		{
			src: []node{
				{name: "1",
					decls: []string{"a", "aa"},
					deps:  []string{},
				},
				{name: "2",
					decls: []string{"b"},
					deps:  []string{"a"},
				},
				{name: "3",
					decls: []string{"c"},
					deps:  []string{"aa"},
				},
			},
			goals:      [][]string{{"1", "2", "3"}, {"1", "3", "2"}},
			shouldFail: false,
		},
		{
			src: []node{
				{name: "1",
					decls: []string{"a"},
					deps:  []string{"b"},
				},
				{name: "2",
					decls: []string{"b"},
					deps:  []string{"c"},
				},
				{name: "3",
					decls: []string{"c"},
					deps:  []string{"a"},
				},
			},
			goals:      nil,
			shouldFail: true,
		},
		{
			src: []node{
				{name: "1",
					decls: []string{"a", "aa"},
					deps:  []string{"b"},
				},
				{name: "2",
					decls: []string{"b"},
					deps:  []string{"a"},
				},
				{name: "3",
					decls: []string{"c"},
					deps:  []string{"aa"},
				},
			},
			goals:      nil,
			shouldFail: true,
		},
	}

	for i, c := range cases {
		if *justCase >= 0 && i != *justCase {
			continue
		}
		input := []*TopLevelStmt{}
		for _, node := range c.src {
			stmt := testStmt{name: node.name}
			stmt.decls = node.decls

			tls := &TopLevelStmt{Stmt: stmt, unboundIdents: map[string]*Ident{}}
			for _, dep := range node.deps {
				tls.unboundIdents[dep] = nil
			}
			tls.loadDeps()

			input = append(input, tls)
		}

		l, err := topoSort(input)
		if c.shouldFail {
			if err == nil {
				t.Fail()
				fmt.Printf("Case should have failed")
			}
		} else {
			if len(l) != len(input) {
				t.Fail()
				fmt.Printf("Different length: %d and %d", len(l), len(input))
			}
			ok := false
		goalsLoop:
			for _, goal := range c.goals {
				for i := range goal {
					//fmt.Printf("%s -- %s\n", goal[i], l[i].Stmt.(testStmt).name)
					if goal[i] != l[i].Stmt.(testStmt).name {
						//fmt.Printf("Difference on pos %d\n", i)
						continue goalsLoop
					}
				}
				ok = true
				break
			}

			if !ok {
				fmt.Printf("Wrong order, not found in the possible orders (case %d)", i)
				t.Fail()
			}
		}
	}
}

var justCase = flag.Int("case", -1, "Run only selected test case")

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}
