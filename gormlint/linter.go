package gormlint

import (
	"go/ast"
	"go/token"
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

type Settings struct {
	// Providers maps fully-qualified parameter type to extractor path for context.
	// Examples:
	//   "context.Context": ""           // parameter itself is context
	//   "*net/http.Request": "Context" // req.Context()
	Providers map[string]string
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
	providers := l.providersOrDefault()
	for _, file := range pass.Files {
		// rule: enforce first-call WithContext/Session when provider params exist
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if ok && fn.Body != nil && fn.Type != nil {
				l.analyzeFunction(pass, providers, fn.Type.Params, fn.Body)
			}
		}
		ast.Inspect(file, func(n ast.Node) bool {
			if lit, ok := n.(*ast.FuncLit); ok && lit.Body != nil && lit.Type != nil {
				l.analyzeFunction(pass, providers, lit.Type.Params, lit.Body)
			}
			return true
		})

		// rule: (*gorm.DB).First must be called with exactly 1 argument (dest)
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if sel.Sel == nil || sel.Sel.Name != "First" {
				return true
			}
			if !isGormDBReceiver(pass, sel.X) {
				return true
			}
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

type funcContext struct {
	enforce        bool
	allowedCtxObjs map[types.Object]struct{}
	providers      map[*ast.Ident]string
	prepared       map[string]struct{}
}

func (l *Linter) analyzeFunction(pass *analysis.Pass, providers map[string]string, params *ast.FieldList, body *ast.BlockStmt) {
	if body == nil {
		return
	}
	ctx := &funcContext{
		allowedCtxObjs: make(map[types.Object]struct{}),
		providers:      make(map[*ast.Ident]string),
		prepared:       make(map[string]struct{}),
	}
	// Collect provider params and allowed ctx params
	if params != nil {
		for _, f := range params.List {
			for _, name := range f.Names {
				if name == nil {
					continue
				}
				obj := pass.TypesInfo.Defs[name]
				if obj == nil {
					continue
				}
				fqn, _ := typeFQNWithPtrFlag(underlyingNamedOrPtrNamed(obj.Type()))
				if extractor, ok := providers[fqn]; ok {
					ctx.enforce = true
					ctx.providers[name] = extractor
					if fqn == "context.Context" {
						ctx.allowedCtxObjs[obj] = struct{}{}
					}
				}
			}
		}
	}
	if !ctx.enforce {
		return
	}
	ast.Inspect(body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncLit:
			// Do not descend into nested functions here; they will be analyzed separately
			return false
		case *ast.AssignStmt:
			l.processAssign(pass, ctx, node)
		case *ast.CallExpr:
			l.processCall(pass, ctx, node)
		}
		return true
	})
}

func (l *Linter) processAssign(pass *analysis.Pass, ctx *funcContext, as *ast.AssignStmt) {
	if len(as.Rhs) != len(as.Lhs) {
		return
	}
	for i := range as.Rhs {
		rhs := as.Rhs[i]
		lhs := as.Lhs[i]
		// Track ctx aliases: x := provider.Context()
		if id, ok := lhs.(*ast.Ident); ok {
			if l.isAllowedCtxExpr(pass, ctx, rhs) {
				if obj := pass.TypesInfo.Defs[id]; obj != nil && isContextType(obj.Type()) {
					ctx.allowedCtxObjs[obj] = struct{}{}
				}
			}
		}
		// Track prepared DBs via assignment from WithContext/Session
		call, ok := rhs.(*ast.CallExpr)
		if !ok {
			continue
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil {
			continue
		}
		if !isGormDBReceiver(pass, sel.X) {
			continue
		}
		method := sel.Sel.Name
		if method == "WithContext" {
			if len(call.Args) == 1 && l.isAllowedCtxExpr(pass, ctx, call.Args[0]) && !isBackgroundOrTODO(pass, call.Args[0]) {
				if isDBExpr(pass, lhs) {
					ctx.prepared[exprKey(lhs)] = struct{}{}
				}
			}
			continue
		}
		if method == "Session" {
			if len(call.Args) == 1 && l.isSessionWithAllowedContext(pass, ctx, call.Args[0]) {
				if isDBExpr(pass, lhs) {
					ctx.prepared[exprKey(lhs)] = struct{}{}
				}
			}
			continue
		}
	}
}

func (l *Linter) processCall(pass *analysis.Pass, ctx *funcContext, call *ast.CallExpr) {
	firstCall, firstSel := firstDBCallInChain(pass, call)
	if firstCall == nil || firstSel == nil {
		return
	}
	// Only handle once per chain: trigger from the first call node itself
	if firstCall != call {
		return
	}
	// If base was prepared earlier via assignment, skip enforcement
	base := firstSel.X
	if _, ok := ctx.prepared[exprKey(base)]; ok {
		return
	}
	method := firstSel.Sel.Name
	switch method {
	case "WithContext":
		if len(firstCall.Args) == 1 {
			if isBackgroundOrTODO(pass, firstCall.Args[0]) {
				pass.Report(analysis.Diagnostic{Pos: firstCall.Pos(), Category: "gormlint", Message: "do not use context.Background/TODO when provider parameters exist; use request/context param"})
				return
			}
			if l.isAllowedCtxExpr(pass, ctx, firstCall.Args[0]) {
				return
			}
		}
		pass.Report(analysis.Diagnostic{Pos: firstCall.Pos(), Category: "gormlint", Message: "first call on gorm DB must be WithContext(valid context from parameters) or Session with Context"})
		return
	case "Session":
		if len(firstCall.Args) == 1 && l.isSessionWithAllowedContext(pass, ctx, firstCall.Args[0]) {
			return
		}
		pass.Report(analysis.Diagnostic{Pos: firstCall.Pos(), Category: "gormlint", Message: "first call on gorm DB must be WithContext(valid context from parameters) or Session with Context"})
		return
	default:
		pass.Report(analysis.Diagnostic{Pos: firstCall.Pos(), Category: "gormlint", Message: "first call on gorm DB must be WithContext(valid context from parameters) or Session with Context"})
	}
}

// firstDBCallInChain finds the earliest method call in a chain on a *gorm.DB value.
// It returns the CallExpr and its SelectorExpr.
func firstDBCallInChain(pass *analysis.Pass, expr ast.Expr) (*ast.CallExpr, *ast.SelectorExpr) {
	cur, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil, nil
	}
	for {
		sel, ok := cur.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil {
			return nil, nil
		}
		if !isGormDBReceiver(pass, sel.X) {
			return nil, nil
		}
		if inner, ok := sel.X.(*ast.CallExpr); ok {
			// Descend only if the inner is also a method call on DB
			if innerSel, ok := inner.Fun.(*ast.SelectorExpr); ok && innerSel.Sel != nil && isGormDBReceiver(pass, innerSel.X) {
				cur = inner
				continue
			}
			// Otherwise, current sel is the first DB method after a function that returns DB
			return cur, sel
		}
		return cur, sel
	}
}

func (l *Linter) providersOrDefault() map[string]string {
	if len(l.settings.Providers) > 0 {
		return l.settings.Providers
	}
	return map[string]string{
		"context.Context":   "",
		"*net/http.Request": "Context",
	}
}

func isGormDBReceiver(pass *analysis.Pass, expr ast.Expr) bool {
	t := pass.TypesInfo.TypeOf(expr)
	if t == nil {
		return false
	}
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	named, ok := t.(*types.Named)
	if !ok || named.Obj() == nil || named.Obj().Pkg() == nil {
		return false
	}
	return named.Obj().Pkg().Path() == "gorm.io/gorm" && named.Obj().Name() == "DB"
}

func isDBExpr(pass *analysis.Pass, expr ast.Expr) bool {
	t := pass.TypesInfo.TypeOf(expr)
	if t == nil {
		return false
	}
	if _, ok := t.(*types.Pointer); ok {
		return isGormDBReceiver(pass, expr)
	}
	return isGormDBReceiver(pass, expr)
}

func isBackgroundOrTODO(pass *analysis.Pass, expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil {
		return false
	}
	if id, ok := sel.X.(*ast.Ident); ok {
		obj := pass.TypesInfo.Uses[id]
		if pkgName, ok := obj.(*types.PkgName); ok && pkgName.Imported().Path() == "context" {
			return sel.Sel.Name == "Background" || sel.Sel.Name == "TODO"
		}
	}
	return false
}

func (l *Linter) isSessionWithAllowedContext(pass *analysis.Pass, ctx *funcContext, arg ast.Expr) bool {
	un, ok := arg.(*ast.UnaryExpr)
	if !ok || un.Op != token.AND {
		return false
	}
	cl, ok := un.X.(*ast.CompositeLit)
	if !ok {
		return false
	}
	t := pass.TypesInfo.TypeOf(cl)
	if t == nil {
		return false
	}
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	named, ok := t.(*types.Named)
	if !ok || named.Obj() == nil || named.Obj().Pkg() == nil {
		return false
	}
	if !(named.Obj().Pkg().Path() == "gorm.io/gorm" && named.Obj().Name() == "Session") {
		return false
	}
	for _, elt := range cl.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if keyIdent, ok := kv.Key.(*ast.Ident); ok && keyIdent.Name == "Context" {
				return l.isAllowedCtxExpr(pass, ctx, kv.Value) && !isBackgroundOrTODO(pass, kv.Value)
			}
		}
	}
	return false
}

func (l *Linter) isAllowedCtxExpr(pass *analysis.Pass, ctx *funcContext, expr ast.Expr) bool {
	if id, ok := expr.(*ast.Ident); ok {
		if obj := pass.TypesInfo.Uses[id]; obj != nil {
			if _, ok := ctx.allowedCtxObjs[obj]; ok {
				return true
			}
		}
	}
	if call, ok := expr.(*ast.CallExpr); ok {
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel != nil {
			if recvIdent, ok := sel.X.(*ast.Ident); ok {
				if extractor, ok := ctx.providers[recvIdent]; ok {
					if extractor == sel.Sel.Name {
						retT := pass.TypesInfo.TypeOf(expr)
						return isContextType(retT)
					}
				}
			}
		}
	}
	if isContextType(pass.TypesInfo.TypeOf(expr)) && !isBackgroundOrTODO(pass, expr) {
		return true
	}
	return false
}

func isContextType(t types.Type) bool {
	if t == nil {
		return false
	}
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	if named, ok := t.(*types.Named); ok && named.Obj() != nil && named.Obj().Pkg() != nil {
		return named.Obj().Pkg().Path() == "context" && named.Obj().Name() == "Context"
	}
	return false
}

func exprKey(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.Ident:
		return "id:" + v.Name
	case *ast.SelectorExpr:
		return "sel:" + exprKey(v.X) + "." + v.Sel.Name
	default:
		return "?"
	}
}

func underlyingNamedOrPtrNamed(t types.Type) types.Type {
	if t == nil {
		return nil
	}
	if p, ok := t.(*types.Pointer); ok {
		return p.Elem()
	}
	return t
}

func typeFQNWithPtrFlag(t types.Type) (string, bool) {
	ptr := false
	if t == nil {
		return "", false
	}
	orig := t
	if p, ok := orig.(*types.Pointer); ok {
		ptr = true
		t = p.Elem()
	}
	if named, ok := t.(*types.Named); ok && named.Obj() != nil && named.Obj().Pkg() != nil {
		name := named.Obj().Pkg().Path() + "." + named.Obj().Name()
		if ptr {
			name = "*" + name
		}
		return name, ptr
	}
	return "", ptr
}

// isGormDBReceiver returns true if the expression's type is gorm.io/gorm.DB (pointer or not).
// duplicate removed
