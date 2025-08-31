# GoCallTracer

A powerful AST-based code analysis tool for Go projects that helps you quickly trace function dependencies and understand code relationships.

## What it does

GoCallTracer analyzes Go functions and methods to:
- **Trace function calls** - Find all functions called by a target function
- **Extract type references** - Identify all custom types used within functions  
- **Generate dependency reports** - Get complete source code and analysis
- **Support recursive analysis** - Trace dependencies multiple levels deep

## Built with

- **Go AST (Abstract Syntax Tree)** parsing using `go/ast` and `go/types`
- **Static analysis** with `golang.org/x/tools/go/packages` 
- **MCP (Model Context Protocol)** integration for AI assistants
- Project-aware analysis that focuses only on your codebase

## Usage

### CLI Tool

```bash
# Build the CLI tool
go build -o gct-cli cmd/gct-cli/main.go

# Analyze a function
./gct-cli -p /path/to/project -i internal/handler.go -t HandleRequest -deep 2
```

**Parameters:**
- `-p`: Project root directory (required)
- `-i`: File containing target function (required)  
- `-t`: Function name to analyze (required)
- `-o`: Output file (default: analysis_result.txt)
- `-deep`: Recursion depth (default: 0)

### MCP Server (AI Integration)

```bash
# Build and run MCP server
go build -o gct-server cmd/gct-server/main.go

# Run with stdio (for AI assistants)
./gct-server -mode stdio

# Run with HTTP/SSE
./gct-server -mode sse -addr :8080
```

**Available MCP Tools:**
- `full_report` - Complete recursive dependency analysis
- `func_code` - Get source code of specific functions
- `ref_types` - Extract referenced types
- `called_funcs` - List called functions  
- `get_snippet` - Get code snippets

## Example Output

```
Analysis for Function: HandleRequest (depth=2)
Defined in: internal/handler.go

--- Summary of Dependencies ---
Called Functions/Methods:
- myproject/internal/db.GetUser
- myproject/internal/auth.ValidateToken

Referenced Types:  
- myproject/internal/types.User
- myproject/internal/types.Request

--- Code Snippets of Dependencies ---
// Complete source code for each dependency...
```

## Installation

```bash
# Clone and build
git clone https://github.com/takidog/GoCallTracer.git
cd GoCallTracer
go mod tidy
go build -o gct-cli cmd/gct-cli/main.go
go build -o gct-server cmd/gct-server/main.go
```

## Use Cases

- **Code comprehension** - Understand complex function relationships
- **Code reviews** - Get complete context of changes
- **Refactoring** - Assess impact before making changes
- **AI-assisted development** - Integrate with AI tools via MCP protocol

---

Perfect for developers who need to quickly understand and navigate large Go codebases!