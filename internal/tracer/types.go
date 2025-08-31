// internal/tracer/types.go
package tracer

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/packages"
)

// AnalysisTarget represents a function or method to be analyzed.
type AnalysisTarget struct {
	Pkg *packages.Package
	Fn  *ast.FuncDecl
}

// AnalysisTask represents a task in the analysis work queue.
type AnalysisTask struct {
	Target AnalysisTarget
	Depth  int
}

// TypeInfo stores information about a discovered type definition.
type TypeInfo struct {
	Name       string
	Definition types.Object
	Snippet    string
}