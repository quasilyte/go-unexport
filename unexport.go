package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"log"
	"os/exec"
	"strings"
	"unicode"

	"github.com/go-toolsmith/pkgload"
	"golang.org/x/tools/go/packages"
)

func main() {
	var l linter

	steps := []struct {
		name string
		fn   func() error
	}{
		{"init linter", l.init},
		{"parse flags", l.parseFlags},
		{"load targets", l.loadTargets},
		{"collect symbols", l.collectSymbols},
		{"unexport symbols", l.unexportSymbols},
		{"print results", l.printResults},
	}

	for _, step := range steps {
		if err := step.fn(); err != nil {
			log.Fatalf("%s: %v", step.name, err)
		}
	}
}

type linter struct {
	fset *token.FileSet
	pkgs []*packages.Package

	flags struct {
		targets  []string
		verbose  bool
		unexport string
		skip     string
	}

	unexport map[string]bool
	skip     map[string]bool

	symbols []*ast.Ident
	success map[string]string
}

func (l *linter) parseFlags() error {
	flag.BoolVar(&l.flags.verbose, "v", false,
		`print more information than usually`)
	flag.StringVar(&l.flags.unexport, "unexport", "",
		`comma-separated list of symbols to unexport; if empty, reads as 'all'`)
	flag.StringVar(&l.flags.skip, "skip", "",
		`comma-separated list of symbols not to unexport`)

	flag.Parse()

	l.flags.targets = flag.Args()

	for _, sym := range strings.Split(l.flags.unexport, ",") {
		l.unexport[sym] = true
	}
	for _, sym := range strings.Split(l.flags.skip, ",") {
		l.skip[sym] = true
	}

	return nil
}

func (l *linter) init() error {
	l.unexport = make(map[string]bool)
	l.skip = make(map[string]bool)
	l.success = make(map[string]string)
	return nil
}

func (l *linter) loadTargets() error {
	l.fset = token.NewFileSet()
	cfg := &packages.Config{
		Mode:  packages.LoadSyntax,
		Tests: true,
		Fset:  l.fset,
	}

	pkgs, err := packages.Load(cfg, l.flags.targets...)
	if err != nil {
		return err
	}

	pkgload.VisitUnits(pkgs, func(u *pkgload.Unit) {
		if u.Test != nil {
			l.pkgs = append(l.pkgs, u.Test)
		} else {
			l.pkgs = append(l.pkgs, u.Base)
		}
	})

	return nil
}

func (l *linter) collectSymbols() error {
	for _, pkg := range l.pkgs {
		for _, f := range pkg.Syntax {
			if l.fset.Position(f.Pos()).Filename == "" {
				continue
			}
			l.collectFileSymbols(f)
		}
	}

	return nil
}

func (l *linter) collectFileSymbols(f *ast.File) {
	for _, decl := range f.Decls {
		switch decl := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				switch spec := spec.(type) {
				case *ast.ValueSpec:
					for _, id := range spec.Names {
						l.collectSym(id)
					}
				case *ast.TypeSpec:
					l.collectSym(spec.Name)
				}
			}
		case *ast.FuncDecl:
			l.collectSym(decl.Name)
		}
	}

}

func (l *linter) collectSym(sym *ast.Ident) {
	if l.unexport != nil || l.unexport[sym.Name] {
		if !l.skip[sym.Name] {
			l.symbols = append(l.symbols, sym)
		}
	}
}

func (l *linter) unexportSymbols() error {
	for _, sym := range l.symbols {
		if ast.IsExported(sym.Name) {
			fmt.Printf("trying to unexport %s... ", sym.Name)
			status := l.tryUnexport(sym.Pos(), sym.Name)
			fmt.Println("(" + status + ")")
		}
	}

	return nil
}

func (l *linter) tryUnexport(pos token.Pos, exported string) string {
	posn := l.fset.Position(pos)
	offset := fmt.Sprintf("%s:#%d", posn.Filename, posn.Offset)
	unexported := toLowerFirst(exported)
	out, err := exec.Command("gorename", "-offset", offset, "-to", unexported).CombinedOutput()
	key := fmt.Sprintf("%s/%s", posn, exported)

	if err != nil {
		return "impossible: " + prettyError(string(out))
	}
	l.success[key] = fmt.Sprintf("%s -> %s", exported, unexported)
	return "success"
}

func (l *linter) printResults() error {
	if !l.flags.verbose {
		return nil
	}

	if len(l.success) != 0 {
		fmt.Println("unexported:")
		for key, renamed := range l.success {
			fmt.Printf("\t%s: %s\n", key, renamed)
		}
	}
	return nil
}

func prettyError(s string) string {
	switch {
	case strings.Contains(s, "breaking references"):
		return "would break package clients"
	case strings.Contains(s, "no identifier at this position"):
		return "internal error: invalid position"
	case strings.Contains(s, "not a valid identifier"):
		return "internal error: invalid identifier"
	case strings.Contains(s, "would conflict with this method"):
		return "symbols with unexported name form already exists"
	case strings.Contains(s, "no longer assignable to interface"):
		return "would breaks interface assignability"
	default:
		fmt.Println("unknown error: ", s)
		return "unknown error"
	}
}

func toLowerFirst(s string) string {
	for i, v := range s {
		return string(unicode.ToLower(v)) + s[i+1:]
	}
	return ""
}
