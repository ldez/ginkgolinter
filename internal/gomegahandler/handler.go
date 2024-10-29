package gomegahandler

import (
	"go/ast"
	gotypes "go/types"
	"regexp"

	"golang.org/x/tools/go/analysis"
)

const (
	importPath = `"github.com/onsi/gomega"`
)

// Handler provide different handling, depend on the way gomega was imported, whether
// in imported with "." name, custom name or without any name.
type Handler interface {
	// GetActualFuncName returns the name of the gomega function, e.g. `Expect`
	GetActualFuncName(*ast.CallExpr) (string, bool)
	// ReplaceFunction replaces the function with another one, for fix suggestions
	ReplaceFunction(*ast.CallExpr, *ast.Ident)

	GetActualExpr(assertionFunc *ast.SelectorExpr) *ast.CallExpr

	GetActualExprClone(origFunc, funcClone *ast.SelectorExpr) *ast.CallExpr

	GetNewWrapperMatcher(name string, existing *ast.CallExpr) *ast.CallExpr
}

// GetGomegaHandler returns a gomegar handler according to the way gomega was imported in the specific file
func GetGomegaHandler(file *ast.File, pass *analysis.Pass) Handler {
	for _, imp := range file.Imports {
		if imp.Path.Value != importPath {
			continue
		}

		switch name := imp.Name.String(); {
		case name == ".":
			return &dotHandler{
				pass: pass,
			}
		case name == "<nil>": // import with no local name
			return &nameHandler{name: "gomega", pass: pass}
		default:
			return &nameHandler{name: name, pass: pass}
		}
	}

	return nil // no gomega import; this file does not use gomega
}

// dotHandler is used when importing gomega with dot; i.e.
// import . "github.com/onsi/gomega"
type dotHandler struct {
	pass *analysis.Pass
}

// GetActualFuncName returns the name of the gomega function, e.g. `Expect`
func (h dotHandler) GetActualFuncName(expr *ast.CallExpr) (string, bool) {
	switch actualFunc := expr.Fun.(type) {
	case *ast.Ident:
		return actualFunc.Name, true
	case *ast.SelectorExpr:
		if h.isGomegaVar(actualFunc.X) {
			return actualFunc.Sel.Name, true
		}

		if x, ok := actualFunc.X.(*ast.CallExpr); ok {
			return h.GetActualFuncName(x)
		}

	case *ast.CallExpr:
		return h.GetActualFuncName(actualFunc)
	}
	return "", false
}

// ReplaceFunction replaces the function with another one, for fix suggestions
func (dotHandler) ReplaceFunction(caller *ast.CallExpr, newExpr *ast.Ident) {
	switch f := caller.Fun.(type) {
	case *ast.Ident:
		caller.Fun = newExpr
	case *ast.SelectorExpr:
		f.Sel = newExpr
	}
}

func (dotHandler) GetNewWrapperMatcher(name string, existing *ast.CallExpr) *ast.CallExpr {
	return &ast.CallExpr{
		Fun:  ast.NewIdent(name),
		Args: []ast.Expr{existing},
	}
}

// nameHandler is used when importing gomega without name; i.e.
// import "github.com/onsi/gomega"
//
// or with a custom name; e.g.
// import customname "github.com/onsi/gomega"
type nameHandler struct {
	name string
	pass *analysis.Pass
}

// GetActualFuncName returns the name of the gomega function, e.g. `Expect`
func (g nameHandler) GetActualFuncName(expr *ast.CallExpr) (string, bool) {
	selector, ok := expr.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}

	switch x := selector.X.(type) {
	case *ast.Ident:
		if x.Name != g.name {
			if !g.isGomegaVar(x) {
				return "", false
			}
		}

		return selector.Sel.Name, true

	case *ast.CallExpr:
		return g.GetActualFuncName(x)
	}

	return "", false
}

// ReplaceFunction replaces the function with another one, for fix suggestions
func (nameHandler) ReplaceFunction(caller *ast.CallExpr, newExpr *ast.Ident) {
	caller.Fun.(*ast.SelectorExpr).Sel = newExpr
}

func (g nameHandler) isGomegaVar(x ast.Expr) bool {
	return isGomegaVar(x, g.pass)
}

var gomegaTypeRegex = regexp.MustCompile(`github\.com/onsi/gomega/(?:internal|types)\.Gomega`)

func isGomegaVar(x ast.Expr, pass *analysis.Pass) bool {
	if tx, ok := pass.TypesInfo.Types[x]; ok {
		return IsGomegaType(tx.Type)
	}

	return false
}

func IsGomegaType(t gotypes.Type) bool {
	var typeStr string
	switch ttx := t.(type) {
	case *gotypes.Pointer:
		tp := ttx.Elem()
		typeStr = tp.String()

	case *gotypes.Named:
		typeStr = ttx.String()

	default:
		return false
	}

	return gomegaTypeRegex.MatchString(typeStr)
}

func (h dotHandler) GetActualExpr(assertionFunc *ast.SelectorExpr) *ast.CallExpr {
	actualExpr, ok := assertionFunc.X.(*ast.CallExpr)
	if !ok {
		return nil
	}

	switch fun := actualExpr.Fun.(type) {
	case *ast.Ident:
		return actualExpr
	case *ast.SelectorExpr:
		if isHelperMethods(fun.Sel.Name) {
			return h.GetActualExpr(fun)
		}
		if h.isGomegaVar(fun.X) {
			return actualExpr
		}
	}
	return nil
}

func (h dotHandler) GetActualExprClone(origFunc, funcClone *ast.SelectorExpr) *ast.CallExpr {
	actualExpr, ok := funcClone.X.(*ast.CallExpr)
	if !ok {
		return nil
	}

	switch funClone := actualExpr.Fun.(type) {
	case *ast.Ident:
		return actualExpr
	case *ast.SelectorExpr:
		origFun := origFunc.X.(*ast.CallExpr).Fun.(*ast.SelectorExpr)
		if isHelperMethods(funClone.Sel.Name) {
			return h.GetActualExprClone(origFun, funClone)
		}
		if h.isGomegaVar(origFun.X) {
			return actualExpr
		}
	}
	return nil
}

func (h dotHandler) isGomegaVar(x ast.Expr) bool {
	return isGomegaVar(x, h.pass)
}

func (g nameHandler) GetActualExpr(assertionFunc *ast.SelectorExpr) *ast.CallExpr {
	actualExpr, ok := assertionFunc.X.(*ast.CallExpr)
	if !ok {
		return nil
	}

	switch fun := actualExpr.Fun.(type) {
	case *ast.Ident:
		return actualExpr
	case *ast.SelectorExpr:
		if x, ok := fun.X.(*ast.Ident); ok && x.Name == g.name {
			return actualExpr
		}
		if isHelperMethods(fun.Sel.Name) {
			return g.GetActualExpr(fun)
		}

		if g.isGomegaVar(fun.X) {
			return actualExpr
		}
	}
	return nil
}

func (g nameHandler) GetActualExprClone(origFunc, funcClone *ast.SelectorExpr) *ast.CallExpr {
	actualExpr, ok := funcClone.X.(*ast.CallExpr)
	if !ok {
		return nil
	}

	switch funClone := actualExpr.Fun.(type) {
	case *ast.Ident:
		return actualExpr
	case *ast.SelectorExpr:
		if x, ok := funClone.X.(*ast.Ident); ok && x.Name == g.name {
			return actualExpr
		}
		origFun := origFunc.X.(*ast.CallExpr).Fun.(*ast.SelectorExpr)
		if isHelperMethods(funClone.Sel.Name) {
			return g.GetActualExprClone(origFun, funClone)
		}

		if g.isGomegaVar(origFun.X) {
			return actualExpr
		}
	}
	return nil
}

func (g nameHandler) GetNewWrapperMatcher(name string, existing *ast.CallExpr) *ast.CallExpr {
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent(g.name),
			Sel: ast.NewIdent(name),
		},
		Args: []ast.Expr{existing},
	}
}

func isHelperMethods(funcName string) bool {
	switch funcName {
	case "WithOffset", "WithTimeout", "WithPolling", "Within", "ProbeEvery", "WithContext", "WithArguments", "MustPassRepeatedly":
		return true
	}

	return false
}
