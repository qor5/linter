package mustreceive

import (
	"go/ast"
	"strings"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
)

func init() {
	register.Plugin("mustreceive", New)
}

type FuncConfig struct {
	Package string `json:"package"`
	Func    string `json:"func"`
}

type Settings struct {
	MustReceiveFuncs []FuncConfig `json:"must-receive-funcs"`
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
			Name: "mustreceive",
			Doc:  "Check if function return values are properly received",
			Run:  l.run,
		},
	}, nil
}

func (l *Linter) GetLoadMode() string {
	return register.LoadModeSyntax
}

func (l *Linter) run(pass *analysis.Pass) (any, error) {
	// Create a map for faster lookup
	mustReceiveFuncs := make(map[string]map[string]bool)
	for _, f := range l.settings.MustReceiveFuncs {
		if _, ok := mustReceiveFuncs[f.Package]; !ok {
			mustReceiveFuncs[f.Package] = make(map[string]bool)
		}
		mustReceiveFuncs[f.Package][f.Func] = true
	}

	// Inspect each file
	for _, file := range pass.Files {
		// Build import path to package name mapping
		importMap := make(map[string]string)
		dotImports := make(map[string]bool) // Track packages imported with dot import
		for _, imp := range file.Imports {
			path := strings.Trim(imp.Path.Value, "\"")
			if imp.Name != nil {
				if imp.Name.Name == "." {
					// Handle dot imports like: import . "github.com/theplant/appkit/logtracing"
					dotImports[path] = true
				} else {
					// Handle named imports like: import log "github.com/theplant/appkit/logtracing"
					importMap[imp.Name.Name] = path
				}
			} else {
				// Handle regular imports like: import "github.com/theplant/appkit/logtracing"
				parts := strings.Split(path, "/")
				importMap[parts[len(parts)-1]] = path
			}
		}

		// Stack to keep track of parent nodes
		var stack []ast.Node

		ast.Inspect(file, func(n ast.Node) bool {
			if n == nil {
				// Pop the stack when we leave a node
				if len(stack) > 0 {
					stack = stack[:len(stack)-1]
				}
				return true
			}

			// Push the current node onto the stack
			stack = append(stack, n)

			// Check if it's a function call
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			// Get package and function name
			var pkgName, funcName string
			switch fun := call.Fun.(type) {
			case *ast.Ident:
				funcName = fun.Name
			case *ast.SelectorExpr:
				if pkg, ok := fun.X.(*ast.Ident); ok {
					pkgName = pkg.Name
					funcName = fun.Sel.Name
				}
			}

			// Check if this function requires return value reception
			if pkgName != "" && funcName != "" {
				// Handle normal imports with package qualifier
				if fullPkgPath, ok := importMap[pkgName]; ok {
					// Check if this package and function are in our list
					if pkgFuncs, ok := mustReceiveFuncs[fullPkgPath]; ok && pkgFuncs[funcName] {
						// Check if the call is part of an assignment or variable declaration
						if !hasReturnValueReceiver(stack) {
							pass.Report(analysis.Diagnostic{
								Pos:      call.Pos(),
								Category: "mustreceive",
								Message:  fullPkgPath + "." + funcName + " must receive its return value",
							})
						}
					}
				}
			} else if funcName != "" {
				// Handle dot imports (functions called without package qualifier)
				for dotImportPath := range dotImports {
					if pkgFuncs, ok := mustReceiveFuncs[dotImportPath]; ok && pkgFuncs[funcName] {
						// Check if the call is part of an assignment or variable declaration
						if !hasReturnValueReceiver(stack) {
							pass.Report(analysis.Diagnostic{
								Pos:      call.Pos(),
								Category: "mustreceive",
								Message:  dotImportPath + "." + funcName + " must receive its return value",
							})
						}
					}
				}
			}
			return true
		})
	}
	return nil, nil
}

// hasReturnValueReceiver checks if the function call is properly receiving its return value
func hasReturnValueReceiver(stack []ast.Node) bool {
	// Look for assignment or declaration in the stack
	for i := len(stack) - 1; i >= 0; i-- {
		switch n := stack[i].(type) {
		case *ast.AssignStmt:
			// Check if the call is on the right side of the assignment
			for _, rhs := range n.Rhs {
				if rhs == stack[len(stack)-1] {
					return true
				}
			}
		case *ast.DeclStmt:
			if genDecl, ok := n.Decl.(*ast.GenDecl); ok {
				for _, spec := range genDecl.Specs {
					if valueSpec, ok := spec.(*ast.ValueSpec); ok {
						for _, value := range valueSpec.Values {
							if value == stack[len(stack)-1] {
								return true
							}
						}
					}
				}
			}
		}
	}
	return false
}
