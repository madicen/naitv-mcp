# Cursor Implementation Prompts
**Copy-paste ready prompts for executing the naitv-mcp enhancement plan**

---

## Overview

This document contains three categories of prompts:

1. **Phase 1 Prompt** - Core naitv-mcp structural changes (prerequisites)
2. **Plugin Template Prompt** - Reusable for all 67 plugins
3. **Phase-Specific Plugin Prompts** - Customized for each phase

---

## ⚠️ IMPORTANT: Start with Phase 1 FIRST

**Do NOT start plugins until Phase 1 is complete.** Phase 1 establishes the foundation that all plugins depend on.

---

# PHASE 1: CORE STRUCTURAL CHANGES (naitv-mcp)

## Prompt for Cursor (Auto Mode)

Copy and paste this entire prompt into Cursor when you're ready to start Phase 1:

---

```
# Phase 1: Enhance naitv-mcp Core for Dynamic Plugin Execution

## Context
You are implementing Phase 1 of the naitv-mcp enhancement plan. This is the FOUNDATION that all 67 future plugins depend on. Your goal is to enable dynamic plugin execution through subprocesses with JSON I/O, keeping naitv-mcp generic and language-agnostic.

**Reference documents:**
- NAITV_MCP_EXECUTION_PLAN.md → Phase 1 section
- COMPREHENSIVE_NAITV_MCP_PLAN_EXTENDED.md → Phase 1 section
- CURSOR_QUICK_START.md → Phase 1 section

## What You're Building

1. **Enhanced Plugin Manifest** - Add execution config to plugin.json
2. **Subprocess Dispatcher** - Route tool calls to external executables
3. **Plugin Loader Enhancement** - Discover and register both Go and subprocess plugins
4. **Integration with Main Server** - Wire everything together
5. **Documentation** - Plugin development guide

## Acceptance Criteria

- [ ] Plugin interface includes optional Execution() method
- [ ] Subprocess dispatcher spawns executables with JSON I/O
- [ ] Plugin loader scans plugins/ directory and registers all plugins
- [ ] Main server routes tool calls to correct plugin (Go or subprocess)
- [ ] Error handling with descriptive messages
- [ ] Timeout enforcement (configurable per plugin)
- [ ] Backward compatible with existing Go plugins
- [ ] Tests pass for all new code
- [ ] Documentation complete

## Detailed Implementation Tasks

### Task 1.1: Update Plugin Interface (naitv-mcp/plugin/plugin.go)

Add execution configuration to the Plugin interface:

```go
// ExecutionConfig describes how to run a plugin
type ExecutionConfig struct {
    Runtime   string                 `json:"runtime"`      // "python", "go", "shell", "node"
    Entrypoint string                `json:"entrypoint"`   // path to script/binary (relative to plugin folder)
    Version   string                 `json:"version"`      // SemVer, e.g. "1.0.0"
    Timeout   int                    `json:"timeout"`      // seconds (default 30)
    Resources map[string]interface{} `json:"resources"`    // optional: memory, disk, etc.
}

// Update Plugin interface with optional Execution method
type Plugin interface {
    Name() string
    Tools() []ToolDefinition
    Execution() *ExecutionConfig  // NEW - nil for pure Go plugins
    CallTool(ctx context.Context, name string, args map[string]interface{}) (interface{}, error)
}
```

**Requirements:**
- Execution() returns nil for existing Go plugins (backward compatible)
- ExecutionConfig is well-documented with comments
- Version field supports SemVer format
- Timeout is enforced (use context.WithTimeout)

### Task 1.2: Implement Subprocess Dispatcher (NEW FILE: naitv-mcp/host/subprocess.go)

Create a new subprocess dispatcher that:

```go
// SubprocessHost manages subprocess-based plugins
type SubprocessHost struct {
    pluginsDir string
    registry   map[string]*PluginConfig
    cache      map[string]interface{}
    mu         sync.RWMutex
}

// New creates a subprocess host
func NewSubprocessHost(pluginsDir string) *SubprocessHost {
    return &SubprocessHost{
        pluginsDir: pluginsDir,
        registry:   make(map[string]*PluginConfig),
        cache:      make(map[string]interface{}),
    }
}

// DispatchToolCall routes a tool call to external subprocess
func (h *SubprocessHost) DispatchToolCall(
    ctx context.Context,
    pluginName, toolName string,
    args map[string]interface{},
) (interface{}, error) {
    // 1. Look up plugin in registry
    // 2. Check cache (optional optimization)
    // 3. Marshal input to JSON
    // 4. Spawn subprocess with timeout from context
    // 5. Write JSON to stdin
    // 6. Read JSON from stdout
    // 7. Check exit code (0 = success, non-zero = error)
    // 8. Parse stderr if error occurs
    // 9. Return result or error
}

// RegisterPlugin loads a plugin from disk
func (h *SubprocessHost) RegisterPlugin(pluginPath string) error {
    // 1. Read plugin.json
    // 2. Validate manifest structure
    // 3. Verify Execution config exists
    // 4. Store in registry with resolved paths
    // 5. Return error if validation fails
}

// Interface contract:
// Stdin:  JSON input (single line, newline-terminated)
// Stdout: JSON output (single line, newline-terminated) or empty on error
// Stderr: Error messages (free-form text)
// Exit code: 0 on success, non-zero on error
```

**Requirements:**
- JSON marshaling/unmarshaling with error handling
- Timeout enforcement via context (kill subprocess if exceeds timeout)
- Descriptive error messages
- Support for Python, Go, Shell, Node.js runtimes
- Path resolution relative to plugin folder
- Proper subprocess cleanup on timeout

### Task 1.3: Enhance Plugin Loader (naitv-mcp/plugin/loader.go)

Update existing loader to support dynamic plugins:

```go
// LoadPlugins now discovers and registers both Go and subprocess plugins
func LoadPlugins(pluginsDir string) (map[string]Plugin, error) {
    registry := make(map[string]Plugin)
    
    // 1. Scan pluginsDir for subdirectories
    // 2. For each subdirectory:
    //    a. Check for plugin.json
    //    b. Load manifest
    //    c. Check if it has Execution config
    //    d. If yes: Register as subprocess plugin
    //    e. If no: Try to load as Go plugin (existing logic)
    // 3. Return registry with all plugins
    // 4. Return error if manifest invalid
    
    return registry, nil
}
```

**Requirements:**
- Backward compatible with existing Go plugins
- Clear error messages if plugin.json missing/invalid
- Validate execution config if present
- Support hot-reloading (future: can reload without server restart)
- Log discovered plugins

### Task 1.4: Main Server Integration (naitv-mcp/main.go)

Wire everything together:

```go
// In main():
// 1. Initialize traditional plugin registry (existing Go plugins)
// 2. Initialize SubprocessHost for subprocess plugins
// 3. Load plugins into both
// 4. When tool is called:
//    - Check if plugin is Go or subprocess
//    - Route to correct handler
//    - Return result
```

**Requirements:**
- Both plugin types coexist peacefully
- Single unified tool registry visible to clients
- Proper error handling and logging
- Tests verify both types work

### Task 1.5: Create Plugin Templates

Create template directories in `naitv-mcp-plugins/templates/`:

**Python Template** (`templates/python-plugin/`)
```
plugin.json              # Metadata + execution config
tool.py                  # Main implementation
requirements.txt         # Dependencies
README.md               # Documentation
```

**Shell Template** (`templates/shell-plugin/`)
```
plugin.json
tool.sh
README.md
```

**Go Template** (`templates/go-plugin/`)
```
plugin.json
main.go
go.mod
build.sh
README.md
```

Each template should include:
- Example plugin.json with all required fields
- Commented example tool implementation
- Performance notes
- Error handling example
- Testing guidance

### Task 1.6: Create Plugin-Manager Bootstrap Plugin

Create `naitv-mcp-plugins/plugin-manager/` (Python plugin):

**Tools:**
1. `list_plugins` - List all available plugins and their tools
2. `validate_plugin` - Validate a plugin.json manifest
3. `test_plugin` - Run plugin with sample input and verify JSON output

**Implementation notes:**
- This is a Python plugin demonstrating the pattern
- Shows how to use the plugin.json manifest
- Demonstrates error handling
- Useful for validating new plugins during development

### Task 1.7: Write Documentation

Create or update:
- `naitv-mcp/docs/PLUGIN_DEVELOPMENT.md` - How to create plugins
- `naitv-mcp/docs/PLUGIN_MANIFEST.md` - plugin.json specification
- `naitv-mcp-plugins/README.md` - Overview of plugin ecosystem

## Implementation Order

1. Update Plugin interface
2. Create subprocess dispatcher
3. Enhance plugin loader
4. Integrate with main server
5. Create plugin templates
6. Implement plugin-manager
7. Write documentation
8. Test end-to-end

## Testing Checklist

- [ ] Unit tests for subprocess dispatcher
- [ ] Unit tests for plugin loader
- [ ] Integration test: Go plugin still works
- [ ] Integration test: Python plugin works
- [ ] Integration test: Shell plugin works
- [ ] Integration test: JSON I/O works correctly
- [ ] Integration test: Timeout enforcement works
- [ ] Integration test: Error handling works
- [ ] Performance test: Subprocess startup < 100ms
- [ ] E2E test: Create new plugin, it works immediately

## Verification

Before marking Phase 1 complete:

1. Verify Go plugins still work (backward compatibility)
2. Create a test Python plugin from template
3. Verify Python plugin callable via MCP
4. Verify JSON I/O contract works
5. Verify timeout enforcement
6. Verify error messages are actionable
7. Run all tests
8. Update documentation

## Git Commit

When complete, create a single commit:

```
git commit -m "feat(phase1): implement dynamic plugin execution system

- Add Execution config to Plugin interface
- Implement SubprocessHost for external plugin execution
- Enhance plugin loader for dynamic discovery
- Add plugin templates (Python, Go, Shell)
- Implement plugin-manager bootstrap plugin
- Maintain backward compatibility with existing Go plugins
- Complete documentation and examples

Co-Authored-By: Claude Haiku 4.5 <noreply@anthropic.com>
Claude-Session: <session_id>"
```

## Success Criteria (Definition of Done)

Phase 1 is complete when:
- [ ] New plugin can be placed in plugins/ folder
- [ ] Plugin is immediately discoverable without code changes
- [ ] Tool calls route correctly to subprocess
- [ ] JSON I/O works end-to-end
- [ ] No breaking changes to existing plugins
- [ ] Tests pass
- [ ] Documentation complete
- [ ] Commit pushed to main

## Notes

- Keep naitv-mcp generic: NO domain logic, only routing
- Plugin logic goes in plugins/, never in naitv-mcp core
- Error messages should be model-friendly (JSON when possible)
- Performance should be < 200ms overhead per tool call
- Start simple, optimize later if needed
```

---

## What to Do After Phase 1 is Complete

Once Phase 1 passes all acceptance criteria:

1. ✅ Commit to main branch
2. ✅ Move to Phase 2 (plugins start)
3. ✅ Use the "Plugin Template Prompt" below for each plugin

---

# PHASE 2+: PLUGIN IMPLEMENTATION TEMPLATE

## Generic Plugin Prompt Template

Use this template for EACH plugin. Replace `PLUGIN_NAME` and customize tool names/descriptions.

---

```
# Implement Plugin: PLUGIN_NAME

## Context
You are implementing a single plugin for the naitv-mcp enhancement ecosystem. This plugin is part of Phase X (see COMPREHENSIVE_NAITV_MCP_PLAN_EXTENDED.md).

**Plugin Details:**
- Name: PLUGIN_NAME
- Phase: X
- Category: [Context/Validation/Refactoring/Safety/Automation/Specialized]
- Tools: [List the N tools this plugin provides]
- Implementation: [Python/Go/Shell/Node.js]
- Performance Target: [e.g., < 1s]
- Dependencies: [What plugins must exist first]

**Reference:**
- See PLUGIN_REFERENCE_MATRIX.md for complete spec
- See COMPREHENSIVE_NAITV_MCP_PLAN_EXTENDED.md Phase X section
- See CURSOR_QUICK_START.md Phase X section

## What You're Building

A single plugin with:
- `plugin.json` manifest (well-commented)
- Implementation file(s) (tool.py, main.go, tool.sh, etc.)
- `requirements.txt` or `go.mod` (if needed)
- `tests/` directory with comprehensive tests
- `README.md` with examples
- `examples/` directory with sample inputs/outputs

## Plugin Specification

### Tools to Implement

[Copy from PLUGIN_REFERENCE_MATRIX.md - "Tools" column]

Each tool should:
- Have clear inputs and outputs
- Return JSON on stdout
- Return errors on stderr
- Support timeout enforcement
- Handle edge cases gracefully

### Performance Target

[Copy from PLUGIN_REFERENCE_MATRIX.md - "Perf Target" column]

### Dependencies

[Copy from PLUGIN_REFERENCE_MATRIX.md - "Depends On" column]

Must verify these plugins exist before testing.

## File Structure

```
naitv-mcp-plugins/plugins/PLUGIN_NAME/
├── plugin.json              # Manifest (required)
├── tool.py (or .go/.sh)     # Implementation (required)
├── requirements.txt         # If Python (optional)
├── go.mod                   # If Go (optional)
├── tests/
│   ├── test_tool.py         # Unit tests
│   └── test_integration.py  # Integration tests
├── examples/
│   └── sample_input.json    # Example input
└── README.md               # Documentation
```

## Implementation Checklist

### Structure
- [ ] Directory created: naitv-mcp-plugins/plugins/PLUGIN_NAME/
- [ ] plugin.json manifest with all required fields
- [ ] Implementation file(s) created
- [ ] requirements.txt or go.mod (if needed)
- [ ] tests/ directory with tests
- [ ] examples/ directory with sample inputs
- [ ] README.md with overview and examples

### Implementation
- [ ] All tools declared in plugin.json are implemented
- [ ] Input validation with descriptive errors
- [ ] JSON output on stdout (one line, properly formatted)
- [ ] Errors go to stderr (free-form text)
- [ ] Exit code 0 on success, non-zero on error
- [ ] Timeout respected (don't run forever)
- [ ] No external dependencies except in requirements.txt/go.mod
- [ ] Code is clean and well-commented

### Testing
- [ ] Unit tests for each tool
- [ ] Integration test with naitv-mcp
- [ ] Real codebase test (if applicable)
- [ ] Performance test vs. target
- [ ] Error case testing
- [ ] All tests passing locally

### Documentation
- [ ] README with overview
- [ ] Examples section with sample inputs/outputs
- [ ] Performance characteristics documented
- [ ] Dependencies clearly listed
- [ ] Troubleshooting guide
- [ ] plugin.json well-commented

### Integration
- [ ] Plugin works with naitv-mcp startup
- [ ] Tools callable via MCP protocol
- [ ] No logs polluting stderr
- [ ] Graceful failure if dependencies missing
- [ ] Works on Linux and macOS

## Performance Verification

Before marking done, verify performance target:

1. Test with realistic data
2. Measure execution time
3. Compare to target (e.g., < 1s)
4. If slower, investigate bottlenecks
5. Document any optimizations made

## Git Commit

When complete, commit with:

```
git commit -m "feat(PLUGIN_NAME): implement PLUGIN_NAME plugin

- Implements tools: tool1, tool2, tool3
- Performance: < 1s on typical input
- Tests: 15 unit tests, 3 integration tests
- Docs: README with examples

Co-Authored-By: Claude Haiku 4.5 <noreply@anthropic.com>
Claude-Session: <session_id>"
```

## Success Criteria

Plugin is done when:
- [ ] All tools implemented
- [ ] Tests passing
- [ ] Performance target met
- [ ] Documentation complete
- [ ] Works with naitv-mcp
- [ ] Commit pushed
```

---

## How to Use This Template

For each plugin:

1. **Copy the template above**
2. **Replace placeholders:**
   - `PLUGIN_NAME` → actual plugin name
   - Tool names and descriptions → from PLUGIN_REFERENCE_MATRIX.md
   - Phase X → actual phase
   - Performance target → from matrix
   - Dependencies → from matrix
3. **Paste into Cursor**
4. **Cursor implements the plugin**
5. **Move to next plugin**

---

# PHASE-SPECIFIC PLUGIN PROMPTS

## Phase 2 Plugins (8 total)

### Prompt 2.1: structural-anchor

```
# Implement Plugin: structural-anchor

## Context
This is the FIRST plugin after Phase 1 foundation. It's critical - other plugins depend on it.

## Plugin Details
- Name: structural-anchor
- Phase: 2
- Tools: 
  1. get_project_map() - Extract packages, types, functions from codebase
  2. list_languages() - Detect programming languages in project
  3. get_codebase_statistics() - Size, complexity metrics
- Implementation: Python (with Go/TypeScript support)
- Performance Target: < 2s on 50K files
- Dependencies: None (Phase 1 complete)

## What It Does

Generates a flattened, compressed map of the codebase structure:

**Input:**
```json
{
  "root_path": "/path/to/repo",
  "depth": 3,
  "skip_dirs": ["node_modules", ".git", "vendor"]
}
```

**Output:**
```json
{
  "packages": [
    {
      "name": "mypackage",
      "path": "src/mypackage",
      "symbols": [
        {
          "name": "MyFunction",
          "kind": "function",
          "signature": "func MyFunction(x int) string",
          "line": 42
        }
      ]
    }
  ],
  "languages": ["go", "python"],
  "statistics": {
    "total_files": 1234,
    "total_lines": 50000,
    "complexity": 4.5
  }
}
```

## Implementation Requirements

1. **Language Support:**
   - Go (using go/parser)
   - Python (using ast module)
   - TypeScript/JavaScript (optional: tree-sitter)

2. **Caching:**
   - Cache results per commit hash
   - Invalidate on .git/index changes
   - Save to ~/.cache/naitv-mcp/

3. **Performance:**
   - Must complete < 2s on 50K files
   - Profile and optimize if needed
   - Batch AST parsing

4. **Output Quality:**
   - Only symbols, not full AST
   - Skip hidden dirs (., node_modules, etc.)
   - Include line numbers and signatures

## Testing

- [ ] Test on Linux kernel (~70K files, 25M lines)
- [ ] Test on this repo (~500 files)
- [ ] Measure caching effectiveness
- [ ] Verify JSON output format

## Success Criteria

- Produces accurate symbol map
- Performs < 2s on 50K files
- Handles errors gracefully
- Well-tested
```

### Prompt 2.2: symbol-navigator

```
# Implement Plugin: symbol-navigator

## Context
Second plugin, depends on structural-anchor existing (but doesn't need to call it).

## Plugin Details
- Name: symbol-navigator
- Phase: 2
- Tools:
  1. find_symbol_definition() - Find where symbol is defined
  2. search_by_pattern() - Find symbols matching pattern
  3. get_symbol_references() - Find all usages of symbol
- Implementation: Python + LSP
- Performance Target: < 100ms per lookup
- Dependencies: structural-anchor (must exist)

## What It Does

Precise symbol lookup and reference finding using Language Server Protocol where available, AST parsing as fallback.

**Example:**

Input: `find_symbol_definition` with symbol "UserService"
Output:
```json
{
  "file": "src/auth/user_service.go",
  "line": 42,
  "type": "type",
  "signature": "type UserService struct { ... }",
  "receivers": ["AuthHandler"],
  "imports": ["database/sql", "fmt"]
}
```

## Implementation Requirements

1. **LSP Integration:**
   - Try to connect to language servers first
   - Fall back to AST parsing
   - Graceful handling if no LSP available

2. **Supported Languages:**
   - Go (gopls)
   - Python (pylsp)
   - TypeScript/JavaScript (typescript-language-server)

3. **Performance:**
   - Caching for repeated lookups
   - Fast AST parsing
   - < 100ms per query

4. **Accuracy:**
   - Exact position (file, line, column)
   - Complete signature
   - All references found

## Testing

- [ ] Test on real codebase
- [ ] Verify accuracy of definitions
- [ ] Verify all references found
- [ ] Performance < 100ms
```

### Prompt 2.3: context-injector

### Prompt 2.4: specification-enforcer

### Prompt 2.5: decision-log-runner

### Prompt 2.6: team-knowledge-extractor

### Prompt 2.7: code-review-simulator

### Prompt 2.8: blame-context-historian

---

## How to Execute

### Execution Flow

```
1. Complete Phase 1
   └─> Verify all acceptance criteria met
       └─> Commit to git
           └─> Now ready for plugins

2. For each phase (2-6):
   a. Use phase-specific prompt below
   b. For each plugin in phase:
      - Copy plugin template prompt
      - Customize with plugin details
      - Paste into Cursor (auto mode)
      - Wait for implementation
      - Verify with checklist
      - Commit to git
   c. Move to next plugin

3. After all plugins:
   └─> Integration testing
       └─> Performance validation
           └─> Release
```

### Recommended Command for Cursor

```bash
# Phase 1 - Core changes
cursor --auto "# Phase 1: Enhance naitv-mcp Core... [entire Phase 1 prompt]"

# Phase 2, Plugin 1
cursor --auto "# Implement Plugin: structural-anchor... [plugin prompt]"

# Phase 2, Plugin 2
cursor --auto "# Implement Plugin: symbol-navigator... [plugin prompt]"

# ... continue for each plugin
```

---

## Template for Each Remaining Plugin

Use this format for Plugins 2.4-2.8 (copy from PLUGIN_REFERENCE_MATRIX.md):

```
# Implement Plugin: [PLUGIN_NAME]

## Context
This is phase [X] plugin [Y] of [Z].

## Plugin Details
- Name: [from MATRIX]
- Phase: [from MATRIX]
- Tools: [from MATRIX - exact tool list]
- Implementation: [from MATRIX]
- Performance Target: [from MATRIX]
- Dependencies: [from MATRIX]

## What It Does
[From COMPREHENSIVE_EXTENDED.md - "What it does" section]

## Detailed Tool Specifications
[Copy from COMPREHENSIVE_EXTENDED.md - each tool description]

## Implementation Requirements
[Specific to this plugin]

## Testing Checklist
- [ ] Unit tests
- [ ] Integration test
- [ ] Real codebase test
- [ ] Performance test vs. target
- [ ] Error cases
- [ ] All passing

## Success Criteria
[Standard: All tests pass, performance target met, docs complete]
```

---

## Automating All Plugins

For maximum efficiency, you can create a batch script:

```bash
#!/bin/bash

# Phase 1
echo "Starting Phase 1..."
cursor --auto < phase1_prompt.txt
read -p "Phase 1 complete? (y/n): " phase1_done

# Phase 2
for plugin in structural-anchor symbol-navigator context-injector specification-enforcer decision-log-runner team-knowledge-extractor code-review-simulator blame-context-historian; do
    echo "Starting $plugin..."
    cursor --auto < "phase2_${plugin}.txt"
    read -p "$plugin complete? (y/n): " done
done

# ... continue for other phases
```

---

## Summary: Your Complete Execution Plan

1. **Use Phase 1 prompt** → Modify naitv-mcp core
2. **Use plugin template** → Create reusable structure for each plugin
3. **Use phase-specific prompts** → Detailed prompts for all 67 plugins
4. **Verify after each** → Follow the checklist
5. **Commit after each** → Clear git history
6. **Move to next** → Rinse and repeat

**Total execution time: 26 weeks for all 67 plugins**

---

**Status:** Ready to start Phase 1
**Next Step:** Copy Phase 1 prompt and paste into Cursor (auto mode)
