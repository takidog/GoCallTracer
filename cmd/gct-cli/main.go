// cmd/gct-cli/main.go
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"log"
	"os"
	"path/filepath"

	"go-call-tracer/internal/tracer"

	"golang.org/x/tools/go/packages"
)

func main() {
	// --- CLI Parameter Setup ---
	projectPath := flag.String("p", "", "Project root directory (required)")
	inputFile := flag.String("i", "", "Input file path (required)")
	targetFunc := flag.String("t", "", "Target function/method name (required)")
	outputFile := flag.String("o", "analysis_result.txt", "Output file for the result")
	deep := flag.Int("deep", 0, "Recursion depth for analysis (0 means no recursion)")
	flag.Parse()

	if *projectPath == "" || *inputFile == "" || *targetFunc == "" {
		flag.Usage()
		log.Fatal("Error: -p, -i, and -t are required arguments.")
	}

	// --- Path Handling ---
	absProjectPath, err := filepath.Abs(*projectPath)
	if err != nil {
		log.Fatalf("Error resolving project path: %v", err)
	}
	var absInputFile string
	if filepath.IsAbs(*inputFile) {
		absInputFile = filepath.Clean(*inputFile)
	} else {
		absInputFile = filepath.Join(absProjectPath, *inputFile)
	}

	// --- Load Project ---
	fmt.Printf("Loading project from: %s\n", absProjectPath)
	cfg := &packages.Config{Mode: packages.LoadSyntax | packages.LoadTypes | packages.LoadFiles, Dir: absProjectPath}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		log.Fatalf("Error loading packages: %v", err)
	}
	if packages.PrintErrors(pkgs) > 0 {
		log.Fatalf("Errors found while loading packages.")
	}

	// --- Find Initial Target ---
	var initialTarget tracer.AnalysisTarget
	for _, p := range pkgs {
		for i, file := range p.GoFiles {
			if file == absInputFile {
				for _, decl := range p.Syntax[i].Decls {
					if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == *targetFunc {
						// Note that we are creating a tracer.AnalysisTarget struct
						initialTarget = tracer.AnalysisTarget{Pkg: p, Fn: fn}
						break
					}
				}
			}
			if initialTarget.Fn != nil {
				break
			}
		}
		if initialTarget.Fn != nil {
			break
		}
	}
	if initialTarget.Fn == nil {
		log.Fatalf("Function '%s' not found in file '%s'", *targetFunc, absInputFile)
	}

	// --- Perform Analysis by calling the tracer package ---
	report, err := tracer.Analyze(initialTarget, absInputFile, *deep, pkgs)
	if err != nil {
		log.Fatalf("Analysis failed: %v", err)
	}

	// --- Write Report ---
	err = os.WriteFile(*outputFile, []byte(report), 0644)
	if err != nil {
		log.Fatalf("Error writing to output file: %v", err)
	}
	fmt.Printf("Analysis complete. Results written to %s\n", *outputFile)
}
