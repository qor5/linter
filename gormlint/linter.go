package gormlint

import (
	"go/ast"
	"go/types"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
)

// This linter checks GORM-specific usages.
// First rule: (*gorm.DB).First must be called with exactly 1 argument (dest),
// disallowing usage of its variadic conds parameter.

func init() {
	register.Plugin("gormlint", New)
}

type Settings struct{}

type Linter struct {
	settings Settings
}

func New(settings any) (register.LinterPlugin, error) {
	s, err := register.DecodeSettings[Settings](settings)
	if err != nil {
		return nil, err
	}
	return &Linter{settings: s}, nil
}

func (l *Linter) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{
		{
			Name: "gormlint",
			Doc:  "Check GORM usages (First must have exactly one argument)",
			Run:  l.run,
		},
	}, nil
}

func (l *Linter) GetLoadMode() string {
	// Need type info to confirm receiver type is *gorm.DB
	return register.LoadModeTypesInfo
}

func (l *Linter) run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			// Only care about First method
			if sel.Sel == nil || sel.Sel.Name != "First" {
				return true
			}

			// Confirm receiver is *gorm.DB (or DB/pointer) via type info
			if !isGormDBReceiver(pass, sel.X) {
				return true
			}

			// Enforce exactly 1 argument (dest)
			if len(call.Args) != 1 {
				pass.Report(analysis.Diagnostic{
					Pos:      call.Pos(),
					Category: "gormlint",
					Message:  "gorm DB.First must be called with exactly 1 argument (dest); do not use variadic conds",
				})
			}
			return true
		})
	}
	return nil, nil
}

// isGormDBReceiver returns true if the expression's type is gorm.io/gorm.DB (pointer or not).
func isGormDBReceiver(pass *analysis.Pass, expr ast.Expr) bool {
	t := pass.TypesInfo.TypeOf(expr)
	if t == nil {
		return false
	}

	// Unwrap pointer types
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}

	named, ok := t.(*types.Named)
	if !ok || named.Obj() == nil || named.Obj().Pkg() == nil {
		return false
	}

	return named.Obj().Pkg().Path() == "gorm.io/gorm" && named.Obj().Name() == "DB"
}
