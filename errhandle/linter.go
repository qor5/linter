package errhandle

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"go/token"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
)

func init() {
	register.Plugin("errhandle", New)
}

type Settings struct {
	ProjectPath string   `json:"project-path"` // Root project path to identify internal code
	Whitelist   []string `json:"whitelist"`    // Package paths to exclude from error reporting
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
			Name: "errhandle",
			Doc:  "Check if return statements use github.com/pkg/errors",
			Run:  l.run,
		},
	}, nil
}

func (l *Linter) GetLoadMode() string {
	return register.LoadModeTypesInfo
}

const pkgErrorsPath = "github.com/pkg/errors"

// errorInterface is the error interface type for type checking
var errorInterface *types.Interface

func init() {
	// Create the error interface for type checking
	errType := types.Universe.Lookup("error").Type()
	if iface, ok := errType.Underlying().(*types.Interface); ok {
		errorInterface = iface
	}
}

func (l *Linter) run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		// Build import map for this file
		importMap := make(map[string]string)
		dotImports := make(map[string]bool)

		for _, imp := range file.Imports {
			path := strings.Trim(imp.Path.Value, "\"")
			if imp.Name != nil {
				if imp.Name.Name == "." {
					dotImports[path] = true
				} else {
					importMap[imp.Name.Name] = path
				}
			} else {
				parts := strings.Split(path, "/")
				pkgName := parts[len(parts)-1]
				importMap[pkgName] = path
			}
		}

		// Check each function separately
		ast.Inspect(file, func(n ast.Node) bool {
			if funcDecl, ok := n.(*ast.FuncDecl); ok && funcDecl.Body != nil {
				// Process this function
				l.checkFunction(pass, funcDecl, importMap, dotImports)
				return false // Don't traverse into this function's body again
			}
			return true
		})
	}
	return nil, nil
}

func (l *Linter) checkFunction(pass *analysis.Pass, funcDecl *ast.FuncDecl, importMap map[string]string, _ map[string]bool) {
	// Get named error return values for checking defer statements
	namedErrorReturns := l.getNamedErrorReturns(pass, funcDecl)

	// Find all local error variables that are returned and modified in defer
	localErrorVars := make(map[string]bool)

	// First pass: identify all local error variables that are returned
	ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
		if ret, ok := n.(*ast.ReturnStmt); ok {
			for _, result := range ret.Results {
				if l.isErrorType(pass, result) {
					if ident, ok := result.(*ast.Ident); ok {
						// This is a local variable being returned
						if !namedErrorReturns[ident.Name] {
							localErrorVars[ident.Name] = true
						}
					}
				}
			}
		}
		return true
	})

	// Check if these local error vars are modified in defer
	modifiedErrorVars := l.findErrorVarsModifiedInDefer(pass, funcDecl, importMap, localErrorVars)

	// Second pass: check return statements
	ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
		if ret, ok := n.(*ast.ReturnStmt); ok {
			// Check for direct function call returns like "return strconv.ParseInt(...)"
			// that return multiple values including an error
			if len(ret.Results) == 1 {
				if callExpr, ok := ret.Results[0].(*ast.CallExpr); ok {
					// Check if this function returns an error type
					if t := pass.TypesInfo.TypeOf(callExpr); t != nil {
						if tuple, ok := t.(*types.Tuple); ok {
							// Check if any of the return values is an error
							for i := 0; i < tuple.Len(); i++ {
								varType := tuple.At(i).Type()
								if types.Implements(varType, errorInterface) {
									// This is a function call that returns an error
									if l.shouldReportCallWithTypeInfo(pass, callExpr, importMap) {
										pass.Report(analysis.Diagnostic{
											Pos:      callExpr.Pos(),
											Category: "errhandle",
											Message:  fmt.Sprintf("error should use %s", pkgErrorsPath),
										})
									}
									break
								}
							}
						}
					}
				}
			}

			// Process individual return values
			for _, result := range ret.Results {
				// Check if this return value is an error type
				if l.isErrorType(pass, result) {
					// Handle direct function calls in return statements
					if callExpr, ok := result.(*ast.CallExpr); ok {
						// Direct function call return
						if l.shouldReportCallWithTypeInfo(pass, callExpr, importMap) {
							pass.Report(analysis.Diagnostic{
								Pos:      callExpr.Pos(),
								Category: "errhandle",
								Message:  fmt.Sprintf("error should use %s", pkgErrorsPath),
							})
						}
						continue
					}

					// Skip checking return values that are modified in defer statements
					if ident, ok := result.(*ast.Ident); ok {
						if namedErrorReturns[ident.Name] || modifiedErrorVars[ident.Name] {
							// This error return value is handled in defer, so skip it here
							continue
						}
					}

					if l.shouldReportWithTypeInfo(pass, result, funcDecl.Body, importMap, ret.Pos()) {
						pass.Report(analysis.Diagnostic{
							Pos:      result.Pos(),
							Category: "errhandle",
							Message:  fmt.Sprintf("error should use %s", pkgErrorsPath),
						})
					}
				}
			}
		}
		return true
	})

	// Check for error modifications in defer statements
	l.checkDeferErrorModifications(pass, funcDecl, importMap, namedErrorReturns, localErrorVars)
}

func (l *Linter) isErrorType(pass *analysis.Pass, expr ast.Expr) bool {
	if t := pass.TypesInfo.TypeOf(expr); t != nil {
		return types.Implements(t, errorInterface)
	}
	return false
}

func (l *Linter) shouldReportWithTypeInfo(pass *analysis.Pass, expr ast.Expr, funcBody *ast.BlockStmt, importMap map[string]string, returnPos token.Pos) bool {
	switch e := expr.(type) {
	case *ast.CallExpr:
		return l.shouldReportCallWithTypeInfo(pass, e, importMap)
	case *ast.Ident:
		return l.shouldReportVarWithTypeInfo(pass, e, funcBody, importMap, returnPos)
	}
	return true
}

func (l *Linter) shouldReportCallWithTypeInfo(pass *analysis.Pass, call *ast.CallExpr, importMap map[string]string) bool {
	if selExpr, ok := call.Fun.(*ast.SelectorExpr); ok {
		return l.handleSelectorCall(pass, selExpr, importMap)
	}

	// Handle direct function calls (could be from dot imports or same package)
	if ident, ok := call.Fun.(*ast.Ident); ok {
		return l.handleDirectCall(pass, ident, importMap)
	}

	return false // Can't determine, assume should ignore it
}

func (l *Linter) handleSelectorCall(pass *analysis.Pass, selExpr *ast.SelectorExpr, importMap map[string]string) bool {
	if pkgIdent, ok := selExpr.X.(*ast.Ident); ok {
		pkgName := pkgIdent.Name
		if pkgPath, exists := importMap[pkgName]; exists {
			// This is a package.function() call
			if strings.HasPrefix(pkgPath, pkgErrorsPath) {
				return false // Don't report pkg/errors
			}
			if l.shouldIgnorePackage(pkgPath) {
				return false // Don't report it
			}
			return true // Report external packages
		}
	}

	// For object.method() calls, check if the method belongs to an external package
	if t := pass.TypesInfo.TypeOf(selExpr.X); t != nil {
		var methodPkgPath string

		// Check the receiver type's package
		if named, ok := t.(*types.Named); ok {
			if named.Obj().Pkg() != nil {
				methodPkgPath = named.Obj().Pkg().Path()
			}
		}
		// Handle pointer types
		if ptr, ok := t.(*types.Pointer); ok {
			if named, ok := ptr.Elem().(*types.Named); ok {
				if named.Obj().Pkg() != nil {
					methodPkgPath = named.Obj().Pkg().Path()
				}
			}
		}

		// If we found the method's package, check if it's external package
		if methodPkgPath != "" {
			if l.shouldIgnorePackage(methodPkgPath) {
				return false // Don't report it
			}
			return true // Report it
		}
	}

	return false // Can't determine, assume should ignore it
}

func (l *Linter) handleDirectCall(pass *analysis.Pass, ident *ast.Ident, _ map[string]string) bool {
	// Check if this function is from the same package or dot imports
	if obj := pass.TypesInfo.ObjectOf(ident); obj != nil {
		if obj.Pkg() != nil {
			pkgPath := obj.Pkg().Path()
			// Check if it's from pkgErrorsPath
			if strings.HasPrefix(pkgPath, pkgErrorsPath) {
				return false // Don't report pkgErrorsPath
			}
			// Check if it's from an internal package
			if l.shouldIgnorePackage(pkgPath) {
				return false // Don't report it
			}
			// It's from an external package (like stdlib via dot import)
			return true // Report external packages
		}
	}

	return false // Same package or can't determine - don't report it
}

func (l *Linter) shouldReportVarWithTypeInfo(pass *analysis.Pass, ident *ast.Ident, funcBody *ast.BlockStmt, importMap map[string]string, returnPos token.Pos) bool {
	varName := ident.Name
	shouldReport := true
	var lastAssignPos token.Pos

	// Only search within the current function body, and only consider assignments before the return statement
	ast.Inspect(funcBody, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.AssignStmt:
			// Skip assignments that come after the return statement
			if node.Pos() >= returnPos {
				return true
			}

			for i, lhs := range node.Lhs {
				if lhsIdent, ok := lhs.(*ast.Ident); ok && lhsIdent.Name == varName {
					// Only consider this assignment if it's the latest one before the return
					if node.Pos() > lastAssignPos {
						lastAssignPos = node.Pos()

						// For multiple assignment, we need to check if there's a single call on the right
						// that returns multiple values (like _, err := someFunc())
						if len(node.Rhs) == 1 {
							// Single expression on right, multiple variables on left
							if callExpr, ok := node.Rhs[0].(*ast.CallExpr); ok {
								shouldReport = l.shouldReportCallWithTypeInfo(pass, callExpr, importMap)
								return true
							}
							if rhsIdent, ok := node.Rhs[0].(*ast.Ident); ok {
								shouldReport = l.shouldReportVarWithTypeInfo(pass, rhsIdent, funcBody, importMap, returnPos)
								return true
							}
						} else if i < len(node.Rhs) {
							// Normal assignment, one-to-one mapping
							if callExpr, ok := node.Rhs[i].(*ast.CallExpr); ok {
								shouldReport = l.shouldReportCallWithTypeInfo(pass, callExpr, importMap)
								return true
							}
							if rhsIdent, ok := node.Rhs[i].(*ast.Ident); ok {
								shouldReport = l.shouldReportVarWithTypeInfo(pass, rhsIdent, funcBody, importMap, returnPos)
								return true
							}
						}
					}
				}
			}
		case *ast.GenDecl:
			// Skip declarations that come after the return statement
			if node.Pos() >= returnPos {
				return true
			}

			for _, spec := range node.Specs {
				if valueSpec, ok := spec.(*ast.ValueSpec); ok {
					for i, name := range valueSpec.Names {
						if name.Name == varName && i < len(valueSpec.Values) {
							// Only consider this declaration if it's the latest one before the return
							if node.Pos() > lastAssignPos {
								lastAssignPos = node.Pos()

								if callExpr, ok := valueSpec.Values[i].(*ast.CallExpr); ok {
									shouldReport = l.shouldReportCallWithTypeInfo(pass, callExpr, importMap)
									return true
								}
							}
						}
					}
				}
			}
		}
		return true
	})

	return shouldReport
}

func (l *Linter) shouldIgnorePackage(pkgPath string) bool {
	// Check if it's an internal package
	if l.settings.ProjectPath != "" && strings.HasPrefix(pkgPath, l.settings.ProjectPath) {
		return true
	}

	// Check if it's in the whitelist
	for _, whitelistedPath := range l.settings.Whitelist {
		if strings.HasPrefix(pkgPath, whitelistedPath) {
			return true
		}
	}

	return false
}

// getNamedErrorReturns identifies named error return values in a function
func (l *Linter) getNamedErrorReturns(pass *analysis.Pass, funcDecl *ast.FuncDecl) map[string]bool {
	namedErrorReturns := make(map[string]bool)
	if funcDecl.Type.Results != nil {
		for _, field := range funcDecl.Type.Results.List {
			// Check if this field is of type error
			if l.isErrorType(pass, field.Type) {
				// Add all names to our map
				for _, name := range field.Names {
					namedErrorReturns[name.Name] = true
				}
			}
		}
	}
	return namedErrorReturns
}

// findErrorVarsModifiedInDefer identifies local error variables that are modified in defer statements
func (l *Linter) findErrorVarsModifiedInDefer(pass *analysis.Pass, funcDecl *ast.FuncDecl, importMap map[string]string, localErrorVars map[string]bool) map[string]bool {
	modifiedVars := make(map[string]bool)

	// If no local error vars, nothing to check
	if len(localErrorVars) == 0 {
		return modifiedVars
	}

	// Find all defer statements in the function body
	ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
		if deferStmt, ok := n.(*ast.DeferStmt); ok {
			// Check if this defer statement contains a function literal (anonymous function)
			if funcLit, ok := deferStmt.Call.Fun.(*ast.FuncLit); ok && funcLit.Body != nil {
				// Inspect the function body for assignments to local error vars
				ast.Inspect(funcLit.Body, func(innerNode ast.Node) bool {
					if assignStmt, ok := innerNode.(*ast.AssignStmt); ok {
						for i, lhs := range assignStmt.Lhs {
							if lhsIdent, ok := lhs.(*ast.Ident); ok {
								// Check if this is assigning to a local error var
								if localErrorVars[lhsIdent.Name] {
									// We found an assignment to a local error var in a defer
									var rhsExpr ast.Expr
									if i < len(assignStmt.Rhs) {
										rhsExpr = assignStmt.Rhs[i]
									} else if len(assignStmt.Rhs) == 1 {
										// Multiple assignment with one value on right
										rhsExpr = assignStmt.Rhs[0]
									}

									if rhsExpr != nil {
										// Check if this is a call expression
										if callExpr, ok := rhsExpr.(*ast.CallExpr); ok {
											// Check if the call is using pkg/errors
											if !l.shouldReportCallWithTypeInfo(pass, callExpr, importMap) {
												// This error is properly handled with pkg/errors
												modifiedVars[lhsIdent.Name] = true
											}
										}
									}
								}
							}
						}
					}
					return true
				})
			}
		}
		return true
	})

	return modifiedVars
}

// checkDeferErrorModifications checks if error return values are modified in defer statements
// and reports if the modifications don't use github.com/pkg/errors
func (l *Linter) checkDeferErrorModifications(pass *analysis.Pass, funcDecl *ast.FuncDecl, importMap map[string]string, namedErrorReturns map[string]bool, localErrorVars map[string]bool) {
	// Combine named error returns and local error vars that are returned
	allErrorVars := make(map[string]bool)
	for name := range namedErrorReturns {
		allErrorVars[name] = true
	}
	for name := range localErrorVars {
		allErrorVars[name] = true
	}

	// If no error vars to check, return
	if len(allErrorVars) == 0 {
		return
	}

	// Find all defer statements in the function body
	ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
		if deferStmt, ok := n.(*ast.DeferStmt); ok {
			// Check if this defer statement contains a function literal (anonymous function)
			if funcLit, ok := deferStmt.Call.Fun.(*ast.FuncLit); ok && funcLit.Body != nil {
				// Inspect the function body for assignments to error vars
				ast.Inspect(funcLit.Body, func(innerNode ast.Node) bool {
					if assignStmt, ok := innerNode.(*ast.AssignStmt); ok {
						for i, lhs := range assignStmt.Lhs {
							if lhsIdent, ok := lhs.(*ast.Ident); ok {
								// Check if this is assigning to an error var
								if allErrorVars[lhsIdent.Name] {
									// We found an assignment to an error var in a defer
									var rhsExpr ast.Expr
									if i < len(assignStmt.Rhs) {
										rhsExpr = assignStmt.Rhs[i]
									} else if len(assignStmt.Rhs) == 1 {
										// Multiple assignment with one value on right
										rhsExpr = assignStmt.Rhs[0]
									}

									if rhsExpr != nil {
										// Check if this is a call expression
										if callExpr, ok := rhsExpr.(*ast.CallExpr); ok {
											// Check if the call is using pkg/errors
											if l.shouldReportCallWithTypeInfo(pass, callExpr, importMap) {
												// This error is not properly handled
												pass.Report(analysis.Diagnostic{
													Pos:      callExpr.Pos(),
													Category: "errhandle",
													Message:  fmt.Sprintf("error in defer should use %s", pkgErrorsPath),
												})
											}
										}
									}
								}
							}
						}
					}
					return true
				})
			}
		}
		return true
	})
}
