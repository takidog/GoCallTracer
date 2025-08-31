// internal/tracer/tracer.go
package tracer

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"
)

// resultCollector implements the ast.Visitor interface. It traverses a function's
// AST and collects all referenced internal functions, methods, and types.
type resultCollector struct {
	Info            *types.Info
	ProjectPackages map[string]bool // A set of package paths belonging to the user's project.
	CalledFuncs     []*types.Func   // Stores all functions/methods found.
	ReferencedTypes []types.Object  // Stores all types found.
}

// Visit is the core visitor method called for each node in the AST.
func (v *resultCollector) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}
	ident, ok := node.(*ast.Ident)
	if !ok {
		return v
	}
	obj := v.Info.ObjectOf(ident)
	if obj == nil || obj.Pkg() == nil {
		return v
	}
	if !v.ProjectPackages[obj.Pkg().Path()] {
		return v
	}
	switch obj := obj.(type) {
	case *types.Func:
		v.CalledFuncs = append(v.CalledFuncs, obj)
	case *types.TypeName:
		v.ReferencedTypes = append(v.ReferencedTypes, obj)
	}
	return v
}

// analysisResult holds the collected functions and types from a recursive analysis.
type analysisResult struct {
	CalledFuncs     map[string]*types.Func
	ReferencedTypes map[string]TypeInfo
}

// performRecursiveAnalysis contains the core logic for recursively traversing the AST.
func performRecursiveAnalysis(initialTarget AnalysisTarget, depth int, pkgs []*packages.Package) (*analysisResult, error) {
	projectPackages := make(map[string]bool)
	typePkgMap := make(map[*types.Package]*packages.Package)
	for _, p := range pkgs {
		projectPackages[p.PkgPath] = true
		if p.Types != nil {
			typePkgMap[p.Types] = p
		}
	}

	queue := []AnalysisTask{
		{Target: initialTarget, Depth: 0},
	}
	processedFuncs := make(map[string]bool)
	allCalledFuncs := make(map[string]*types.Func)
	allReferencedTypes := make(map[string]TypeInfo)

	for len(queue) > 0 {
		currentTask := queue[0]
		queue = queue[1:]
		fnObj := currentTask.Target.Pkg.TypesInfo.ObjectOf(currentTask.Target.Fn.Name)
		if fnObj == nil {
			continue
		}
		fnKey := fnObj.(*types.Func).FullName()
		if processedFuncs[fnKey] {
			continue
		}
		processedFuncs[fnKey] = true

		collector := &resultCollector{
			Info:            currentTask.Target.Pkg.TypesInfo,
			ProjectPackages: projectPackages,
		}
		ast.Walk(collector, currentTask.Target.Fn.Body)

		for _, fun := range collector.CalledFuncs {
			funKey := fun.FullName()
			if _, exists := allCalledFuncs[funKey]; !exists {
				allCalledFuncs[funKey] = fun
				if currentTask.Depth < depth && projectPackages[fun.Pkg().Path()] {
					defPkg, ok := typePkgMap[fun.Pkg()]
					if !ok {
						continue
					}
					defNode := findFuncDeclAt(defPkg, fun.Pos())
					if defNode != nil {
						queue = append(queue, AnalysisTask{
							Target: AnalysisTarget{Pkg: defPkg, Fn: defNode},
							Depth:  currentTask.Depth + 1,
						})
					}
				}
			}
		}
		for _, typeObj := range collector.ReferencedTypes {
			typeKey := fmt.Sprintf("%s.%s", typeObj.Pkg().Path(), typeObj.Name())
			if _, exists := allReferencedTypes[typeKey]; !exists {
				allReferencedTypes[typeKey] = TypeInfo{
					Name:       typeKey,
					Definition: typeObj,
				}
			}
		}
	}

	return &analysisResult{
		CalledFuncs:     allCalledFuncs,
		ReferencedTypes: allReferencedTypes,
	}, nil
}

// Analyze performs the recursive code analysis and returns a formatted report.
func Analyze(initialTarget AnalysisTarget, initialFile string, depth int, pkgs []*packages.Package) (string, error) {
	results, err := performRecursiveAnalysis(initialTarget, depth, pkgs)
	if err != nil {
		return "", err
	}

	typePkgMap := make(map[*types.Package]*packages.Package)
	for _, p := range pkgs {
		if p.Types != nil {
			typePkgMap[p.Types] = p
		}
	}

	// --- Report Generation ---
	var report strings.Builder
	report.WriteString(fmt.Sprintf("Analysis for Function: %s (depth=%d)\n", initialTarget.Fn.Name.Name, depth))
	report.WriteString(fmt.Sprintf("Defined in: %s\n", initialFile))

	report.WriteString("\n--- Target Function Source Code ---\n")
	initialSnippet, err := getFuncSourceSnippet(initialTarget.Pkg, initialTarget.Fn.Name.Pos())
	if err != nil {
		report.WriteString(fmt.Sprintf("// Error getting source: %v\n", err))
	} else {
		report.WriteString(initialSnippet + "\n")
	}

	report.WriteString("\n--- Summary of Dependencies ---\n")
	report.WriteString("Called Functions/Methods:\n")
	if len(results.CalledFuncs) > 0 {
		for name := range results.CalledFuncs {
			report.WriteString(fmt.Sprintf("- %s\n", name))
		}
	} else {
		report.WriteString("- None\n")
	}

	report.WriteString("\nReferenced Types:\n")
	if len(results.ReferencedTypes) > 0 {
		for name := range results.ReferencedTypes {
			report.WriteString(fmt.Sprintf("- %s\n", name))
		}
	} else {
		report.WriteString("- None\n")
	}

	report.WriteString("\n--- Code Snippets of Dependencies ---\n")
	if len(results.CalledFuncs) > 0 {
		for name, fun := range results.CalledFuncs {
			defPkg, ok := typePkgMap[fun.Pkg()]
			if !ok {
				continue
			}
			snippet, err := getFuncSourceSnippet(defPkg, fun.Pos())
			if err == nil {
				report.WriteString(fmt.Sprintf("\n// Source for: %s\n", name))
				report.WriteString(fmt.Sprintf("// Defined in: %s\n", defPkg.Fset.File(fun.Pos()).Name()))
				report.WriteString("// --------------------------------------------------\n")
				report.WriteString(snippet + "\n")
			}
		}
	}

	if len(results.ReferencedTypes) > 0 {
		for name, info := range results.ReferencedTypes {
			defPkg := typePkgMap[info.Definition.Pkg()]
			snippet, err := getTypeSourceSnippet(defPkg, info.Definition.Pos())
			if err == nil {
				report.WriteString(fmt.Sprintf("\n// Source for: %s\n", name))
				report.WriteString(fmt.Sprintf("// Defined in: %s\n", defPkg.Fset.File(info.Definition.Pos()).Name()))
				report.WriteString("// --------------------------------------------------\n")
				report.WriteString(snippet + "\n")
			}
		}
	}
	return report.String(), nil
}

// (Helper functions findFuncDeclAt, getFuncSourceSnippet, getTypeSourceSnippet are now un-exported)
func findFuncDeclAt(pkg *packages.Package, pos token.Pos) *ast.FuncDecl {
	// ... (implementation is identical to original)
	for _, fileAST := range pkg.Syntax {
		if fileAST.Pos() <= pos && pos < fileAST.End() {
			var foundNode *ast.FuncDecl
			ast.Inspect(fileAST, func(n ast.Node) bool {
				if fn, ok := n.(*ast.FuncDecl); ok && fn.Name.Pos() == pos {
					foundNode = fn
					return false
				}
				return true
			})
			if foundNode != nil {
				return foundNode
			}
		}
	}
	return nil
}

func getFuncSourceSnippet(pkg *packages.Package, pos token.Pos) (string, error) {
	// ... (implementation is identical to original)
	node := findFuncDeclAt(pkg, pos)
	if node == nil {
		return "", fmt.Errorf("could not find FuncDecl node at position %d", pos)
	}
	var buf bytes.Buffer
	err := format.Node(&buf, pkg.Fset, node)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func getTypeSourceSnippet(pkg *packages.Package, pos token.Pos) (string, error) {
	// ... (implementation is identical to original)
	for _, fileAST := range pkg.Syntax {
		if fileAST.Pos() <= pos && pos < fileAST.End() {
			var foundNode ast.Node
			ast.Inspect(fileAST, func(n ast.Node) bool {
				if ts, ok := n.(*ast.TypeSpec); ok && ts.Name.Pos() == pos {
					foundNode = ts
					return false
				}
				return true
			})
			if foundNode != nil {
				var buf bytes.Buffer
				var parentNode ast.Node = foundNode
				path, _ := astutil.PathEnclosingInterval(fileAST, foundNode.Pos(), foundNode.End())

				for _, pnode := range path {
					if gd, ok := pnode.(*ast.GenDecl); ok && gd.Tok == token.TYPE {
						parentNode = gd
						break
					}
				}

				err := format.Node(&buf, pkg.Fset, parentNode)
				if err != nil {
					return "", err
				}
				return buf.String(), nil
			}
		}
	}
	return "", fmt.Errorf("could not find TypeSpec node at position %d", pos)
}


// FindTarget locates the target function declaration within the loaded packages.
func FindTarget(pkgs []*packages.Package, filePath, funcName string) (AnalysisTarget, error) {
	var target AnalysisTarget
	for _, p := range pkgs {
		for i, file := range p.GoFiles {
			if file == filePath {
				for _, decl := range p.Syntax[i].Decls {
					if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == funcName {
						target = AnalysisTarget{Pkg: p, Fn: fn}
						return target, nil
					}
				}
			}
		}
	}
	return target, fmt.Errorf("function '%s' not found in file '%s'", funcName, filePath)
}

// GetFuncCode returns the source code of a specific function.
func GetFuncCode(target AnalysisTarget) (string, error) {
	return getFuncSourceSnippet(target.Pkg, target.Fn.Name.Pos())
}

// ExtractTypes finds all referenced types within a function, with optional recursion.
func ExtractTypes(target AnalysisTarget, depth int, pkgs []*packages.Package) ([]string, error) {
	results, err := performRecursiveAnalysis(target, depth, pkgs)
	if err != nil {
		return nil, err
	}

	var typeNames []string
	for name := range results.ReferencedTypes {
		typeNames = append(typeNames, name)
	}
	return typeNames, nil
}

// ExtractCalledFuncs finds all functions and methods called by a function, with optional recursion.
func ExtractCalledFuncs(target AnalysisTarget, depth int, pkgs []*packages.Package) ([]string, error) {
	results, err := performRecursiveAnalysis(target, depth, pkgs)
	if err != nil {
		return nil, err
	}

	var funcNames []string
	for name := range results.CalledFuncs {
		funcNames = append(funcNames, name)
	}
	return funcNames, nil
}