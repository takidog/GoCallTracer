package server

import (
	"go-call-tracer/internal/tracer"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"context"

	"golang.org/x/tools/go/packages"
)

// loadProject is a shared helper that loads Go packages from a project path.
func loadProject(projectPath string) ([]*packages.Package, error) {
	cfg := &packages.Config{Mode: packages.LoadSyntax | packages.LoadTypes | packages.LoadFiles, Dir: projectPath}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, err
	}
	if packages.PrintErrors(pkgs) > 0 {
		log.Printf("Errors found while loading packages for project: %s", projectPath)
	}
	return pkgs, nil
}

// fullReportHandler handles requests for the 'full_report' tool.
func fullReportHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	project, err := request.RequireString("project")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	file, err := request.RequireString("file")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	funcName, err := request.RequireString("func")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	depth, err := request.RequireInt("depth")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	pkgs, err := loadProject(project)
	if err != nil {
		return mcp.NewToolResultError("Failed to load project: " + err.Error()), nil
	}

	target, err := tracer.FindTarget(pkgs, file, funcName)
	if err != nil {
		return mcp.NewToolResultError("Failed to find target: " + err.Error()), nil
	}

	report, err := tracer.Analyze(target, file, int(depth), pkgs)
	if err != nil {
		return mcp.NewToolResultError("Failed to analyze dependencies: " + err.Error()), nil
	}

	return mcp.NewToolResultText(report), nil
}

// funcCodeHandler handles requests for the 'func_code' tool.
func funcCodeHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	project, err := request.RequireString("project")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	file, err := request.RequireString("file")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	funcName, err := request.RequireString("func")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	pkgs, err := loadProject(project)
	if err != nil {
		return mcp.NewToolResultError("Failed to load project: " + err.Error()), nil
	}
	target, err := tracer.FindTarget(pkgs, file, funcName)
	if err != nil {
		return mcp.NewToolResultError("Failed to find target: " + err.Error()), nil
	}
	code, err := tracer.GetFuncCode(target)
	if err != nil {
		return mcp.NewToolResultError("Failed to get function code: " + err.Error()), nil
	}

	return mcp.NewToolResultText(code), nil
}

// refTypesHandler handles requests for the 'ref_types' tool.
func refTypesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	project, err := request.RequireString("project")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	file, err := request.RequireString("file")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	funcName, err := request.RequireString("func")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	// Use 3 as default if depth is not provided
	depth, err := request.RequireInt("depth")
	if err != nil {
		depth = 3
	}

	pkgs, err := loadProject(project)
	if err != nil {
		return mcp.NewToolResultError("Failed to load project: " + err.Error()), nil
	}
	target, err := tracer.FindTarget(pkgs, file, funcName)
	if err != nil {
		return mcp.NewToolResultError("Failed to find target: " + err.Error()), nil
	}
	types, err := tracer.ExtractTypes(target, int(depth), pkgs)
	if err != nil {
		return mcp.NewToolResultError("Failed to extract types: " + err.Error()), nil
	}

	return mcp.NewToolResultStructured(types, "ref_types"), nil
}

// calledFuncsHandler handles requests for the 'called_funcs' tool.
func calledFuncsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	project, err := request.RequireString("project")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	file, err := request.RequireString("file")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	funcName, err := request.RequireString("func")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	// Use 0 as default if depth is not provided
	depth, err := request.RequireInt("depth")
	if err != nil {
		depth = 3
	}

	pkgs, err := loadProject(project)
	if err != nil {
		return mcp.NewToolResultError("Failed to load project: " + err.Error()), nil
	}
	target, err := tracer.FindTarget(pkgs, file, funcName)
	if err != nil {
		return mcp.NewToolResultError("Failed to find target: " + err.Error()), nil
	}
	funcs, err := tracer.ExtractCalledFuncs(target, int(depth), pkgs)
	if err != nil {
		return mcp.NewToolResultError("Failed to extract called functions: " + err.Error()), nil
	}

	return mcp.NewToolResultStructured(funcs, "called_funcs"), nil
}

// getSnippetHandler handles requests for the 'get_snippet' tool; it reuses funcCodeHandler.
var getSnippetHandler = funcCodeHandler

// RegisterTools defines all tools on the server and registers their handlers.
func RegisterTools(s *server.MCPServer) {
	// Tool 1: generate a full recursive dependency report.
	fullReportTool := mcp.NewTool("full_report",
		mcp.WithDescription("Generate a comprehensive dependency analysis report for a Go function. This tool traces all functions, methods, and types that your target function depends on, recursively exploring the call chain to the specified depth. Perfect for understanding the complete scope and impact of code changes. Note: This generates extensive output and may consume significant tokens. For focused exploration, start with 'ref_types' and 'called_funcs' tools first."),
		mcp.WithString("project", mcp.Required(), mcp.Description("Absolute path to your Go project root directory (e.g., '/home/user/myproject' or 'C:\\Users\\Dev\\myproject')")),
		mcp.WithString("file", mcp.Required(), mcp.Description("Path to the Go file containing your target function, relative to project root (e.g., 'internal/handlers/user.go')")),
		mcp.WithString("func", mcp.Required(), mcp.Description("Exact name of the function or method you want to analyze (e.g., 'ProcessUserData' or 'HandleRequest')")),
		mcp.WithNumber("depth", mcp.Required(), mcp.Description("How many levels deep to trace dependencies. Start with 1-2 for initial exploration, use 3-4 for comprehensive analysis. Higher values generate more extensive reports.")),
	)
	s.AddTool(fullReportTool, fullReportHandler)

	// Tool 2: retrieve the source code of a specific target function.
	funcCodeTool := mcp.NewTool("func_code",
		mcp.WithDescription("Get the complete, formatted source code for any Go function or method. This tool is perfect for examining specific functions you've discovered through 'called_funcs' or 'ref_types' analysis. Use this when you need to see the actual implementation details, understand function logic, or review code before making changes."),
		mcp.WithString("project", mcp.Required(), mcp.Description("Absolute path to your Go project root directory (same as used in previous analysis)")),
		mcp.WithString("file", mcp.Required(), mcp.Description("Path to the Go file containing the function, relative to project root (e.g., 'pkg/database/user.go')")),
		mcp.WithString("func", mcp.Required(), mcp.Description("Exact function or method name to retrieve (case-sensitive, e.g., 'CreateUser' or 'ValidateEmail')")),
	)
	s.AddTool(funcCodeTool, funcCodeHandler)

	// Tool 3: get all referenced data types within a function.
	refTypesTool := mcp.NewTool("ref_types",
		mcp.WithDescription("Discover all custom data types (structs, interfaces, type aliases) used by a function and its dependencies. This is your primary tool for understanding data structures and type relationships in Go code. Perfect for mapping out the data flow and identifying important types before diving into implementation details. Recommended workflow: Start with depth 1-2 for quick type overview, then use depth 3 for comprehensive type analysis."),
		mcp.WithString("project", mcp.Required(), mcp.Description("Absolute path to your Go project root directory")),
		mcp.WithString("file", mcp.Required(), mcp.Description("Path to Go file containing your target function, relative to project root")),
		mcp.WithString("func", mcp.Required(), mcp.Description("Function name to analyze for type references (exact name, case-sensitive)")),
		mcp.WithNumber("depth", mcp.Required(), mcp.Description("Analysis depth: 1 = direct types only, 2 = types used by called functions, 3 = comprehensive type analysis. Start with 1-2 for exploration.")),
	)
	s.AddTool(refTypesTool, refTypesHandler)

	// Tool 4: get all functions and methods that a function calls.
	calledFuncsTool := mcp.NewTool("called_funcs",
		mcp.WithDescription("Trace the call chain of a Go function to understand what other functions and methods it depends on. This is your primary tool for mapping execution flow and identifying critical dependencies. Use this to understand the scope of changes, find potential side effects, or trace through complex business logic. Recommended approach: Start with depth 1 to see immediate dependencies, then increase to depth 2-3 to trace the full call chain."),
		mcp.WithString("project", mcp.Required(), mcp.Description("Absolute path to your Go project root directory")),
		mcp.WithString("file", mcp.Required(), mcp.Description("Path to Go file containing your target function, relative to project root")),
		mcp.WithString("func", mcp.Required(), mcp.Description("Function name to trace calls from (exact name, case-sensitive)")),
		mcp.WithNumber("depth", mcp.Required(), mcp.Description("Call tracing depth: 1 = immediate calls only, 2 = calls and their calls, 3 = comprehensive call chain. Most useful at depth 1-2.")),
	)
	s.AddTool(calledFuncsTool, calledFuncsHandler)

	// Tool 5: retrieve a source code snippet for a specific function.
	getSnippetTool := mcp.NewTool("get_snippet",
		mcp.WithDescription("Get a clean, formatted code snippet for any function or method you've discovered during analysis. This is identical to 'func_code' and perfect for quickly inspecting specific functions found through 'called_funcs' or 'ref_types' exploration. Use this when you want to see implementation details without generating a full dependency report."),
		mcp.WithString("project", mcp.Required(), mcp.Description("Absolute path to your Go project root directory")),
		mcp.WithString("file", mcp.Required(), mcp.Description("Path to Go file containing the function, relative to project root")),
		mcp.WithString("func", mcp.Required(), mcp.Description("Exact function or method name to retrieve (case-sensitive)")),
	)
	s.AddTool(getSnippetTool, getSnippetHandler)
}
