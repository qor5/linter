package useany

import (
	"go/ast"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
)

func init() {
	register.Plugin("userany", New)
}

type Settings struct {
	// Potential configuration options for future expansion:
	// CheckComments bool `json:"check-comments"` // Whether to check interface{} in comments
	// ExcludePackages []string `json:"exclude-packages"` // Package paths to exclude
}

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
			Name: "useany",
			Doc:  "Check for interface{} that can be replaced with any",
			Run:  l.run,
		},
	}, nil
}

func (l *Linter) GetLoadMode() string {
	return register.LoadModeSyntax
}

func (l *Linter) run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			if iface, ok := n.(*ast.InterfaceType); ok {
				if l.isEmptyInterface(iface) {
					pass.Report(analysis.Diagnostic{
						Pos:      iface.Pos(),
						Category: "useany",
						Message:  "use 'any' instead of 'interface{}'",
						SuggestedFixes: []analysis.SuggestedFix{{
							Message: "Replace 'interface{}' with 'any'",
							TextEdits: []analysis.TextEdit{{
								Pos:     iface.Pos(),
								End:     iface.End(),
								NewText: []byte("any"),
							}},
						}},
					})
				}
			}
			return true
		})
	}
	return nil, nil
}

// isEmptyInterface checks if the interface is an empty interface{}
func (l *Linter) isEmptyInterface(iface *ast.InterfaceType) bool {
	return iface.Methods == nil || len(iface.Methods.List) == 0
}
