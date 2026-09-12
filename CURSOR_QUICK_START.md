# Cursor Quick Start Guide
**Rapid Reference for Executing the naitv-mcp Enhancement Plan**

---

## 📋 Before You Start

1. **Read these in order:**
   - `NAITV_MCP_EXECUTION_PLAN.md` (the main plan)
   - `GEMINI_ANALYSIS_AND_REFINEMENTS.md` (understand the "why")
   - This file (quick reference)

2. **Understand the core philosophy:**
   - Keep naitv-mcp generic (transport layer)
   - Plugins are self-contained, portable, language-agnostic
   - Each plugin runs as subprocess with JSON I/O
   - Local models need tight feedback loops, not clever guessing

3. **Repos involved:**
   - `naitv-mcp` (main) - changes only to core plugin system
   - `naitv-mcp-plugins` (plugins) - all plugin implementations go here

---

## 🎯 Quick Phase Reference

### Phase 1: Foundation (2 weeks)
**Output:** Dynamic plugin execution system is working

**Key Files to Modify:**
- `naitv-mcp/plugin/plugin.go` - Add execution config
- `naitv-mcp/host/subprocess.go` - NEW: subprocess dispatcher
- `naitv-mcp/main.go` - Initialize dynamic plugin loader

**Key Files to Create:**
- Plugin templates in `naitv-mcp-plugins/templates/`
- `naitv-mcp-plugins/plugin-manager/` - Bootstrapping plugin

**Verify:**
- [ ] New plugin can be placed in plugins/ folder
- [ ] Tool call routes to subprocess
- [ ] JSON I/O works end-to-end
- [ ] No breaking changes to existing Go plugins

---

### Phase 2: Context Tools (2 weeks)
**Output:** Models can query code structure without reading files

**Plugins to Build (in order):**
1. `structural-anchor` - Project map generator
2. `symbol-navigator` - Find definitions
3. `context-injector` - Code slice generator

**Key Decisions:**
- Use language servers where available (LSP)
- Fall back to AST parsing (go/parser, ast module)
- Aggressive caching with file hash invalidation

**Verify:**
- [ ] Structural map < 2s on 50K files
- [ ] Symbol lookup < 100ms
- [ ] Context slices are model-friendly (< 100 lines)

---

### Phase 3: Validation (2 weeks)
**Output:** Models receive validation feedback and self-correct

**Plugins to Build (in order):**
1. `lint-fix-loop` - Syntax validation + auto-fix
2. `tdd-runner` - Test generation + execution
3. `validate-and-test` - Integrated validation gate

**Key Decisions:**
- Use native linters (golangci-lint, pylint, eslint)
- Use native test frameworks (go test, pytest, jest)
- Output structured JSON for model consumption

**Verify:**
- [ ] Linter < 1s per file
- [ ] Test runner < 5s
- [ ] Error format is machine-parseable
- [ ] Models can self-correct in feedback loop

---

### Phase 4: Optimization (2 weeks)
**Output:** Models have instant access to docs, patterns, and error solutions

**Plugins to Build (in order):**
1. `doc-scraper` - Fetch and parse documentation
2. `workspace-search` - Semantic code search
3. `cache-manager` - Performance optimization layer
4. `error-matcher` - Historical error matching

**Key Decisions:**
- Markdown parsing + HTML to text conversion
- ripgrep for fast pattern search
- File hash-based cache invalidation
- Git history mining for error solutions

**Verify:**
- [ ] Doc fetch < 3s
- [ ] Workspace search < 500ms
- [ ] Cache hits improve performance 2-5x
- [ ] Error matcher finds relevant solutions

---

### Phase 5: Workflows (2 weeks)
**Output:** Models can solve non-trivial tasks in tight loops

**Plugins to Build (in order):**
1. `think-before-write` - Planning enforcer
2. `golden-example-injector` - Example-based learning
3. `type-import-stubber` - Pre-generate stubs
4. `surgical-patch` - Minimal change enforcement

**Integration:**
- Document complete workflow
- Create end-to-end example
- Benchmark on real tasks

**Verify:**
- [ ] Models create plans before code
- [ ] Examples inject successfully
- [ ] Type stubs prevent import errors
- [ ] Changes are surgical (not whole-file rewrites)

---

## 🛠️ Plugin Development Checklist

**For each plugin, follow this checklist:**

### Structure
- [ ] Create `naitv-mcp-plugins/plugins/PLUGIN_NAME/` directory
- [ ] Create `plugin.json` manifest
- [ ] Create implementation files (tool.py, tool.sh, main.go, etc.)
- [ ] Create `requirements.txt` or `go.mod` (if applicable)
- [ ] Create `tests/` directory with test suite
- [ ] Create `README.md` with examples

### Implementation
- [ ] All tools declared in plugin.json are implemented
- [ ] Input validation with descriptive errors
- [ ] JSON output on stdout (one line, properly formatted)
- [ ] Errors go to stderr
- [ ] Exit code 0 on success, non-zero on failure
- [ ] Timeout respected (don't run forever)

### Testing
- [ ] Unit tests for each tool
- [ ] Integration test with naitv-mcp (via JSON I/O)
- [ ] Test with real codebase examples
- [ ] Performance benchmarks (compare to targets)
- [ ] Error case testing

### Documentation
- [ ] README with overview
- [ ] Examples section with sample inputs/outputs
- [ ] Performance characteristics documented
- [ ] Dependencies clearly listed
- [ ] Troubleshooting guide

### Integration
- [ ] Plugin works with naitv-mcp startup
- [ ] Tools callable via MCP protocol
- [ ] No logs polluting stderr
- [ ] Graceful failure if dependencies missing
- [ ] Works on Linux and macOS

---

## 📊 Performance Targets Checklist

Before marking a plugin done, verify targets:

| Plugin | Operation | Target | Status |
|--------|-----------|--------|--------|
| structural-anchor | get_project_map | < 2s on 50K files | ☐ |
| symbol-navigator | find_symbol | < 100ms | ☐ |
| context-injector | get_code_slice | < 50ms | ☐ |
| lint-fix-loop | run_linter | < 1s | ☐ |
| tdd-runner | run_tests | < 5s | ☐ |
| doc-scraper | fetch_and_parse | < 3s | ☐ |
| workspace-search | semantic_search | < 500ms | ☐ |

---

## 🚀 Git Workflow

For each plugin:

```bash
# Create feature branch
git checkout -b feature/plugin-PLUGIN_NAME

# Implement plugin
# ... write code, tests, docs ...

# Test locally
cd naitv-mcp-plugins/plugins/PLUGIN_NAME
pytest tests/  # or go test ./...
python tool.py < sample-input.json  # or ./tool.sh

# Commit with clear message
git add .
git commit -m "feat(PLUGIN_NAME): implement PLUGIN_NAME plugin

- Implements tools: tool1, tool2, tool3
- Performance: tool1 < 2s on 50K files
- Tests: 15 unit tests, 3 integration tests
- Docs: README with examples"

# Push and create PR
git push origin feature/plugin-PLUGIN_NAME
# Create PR on GitHub, link to plan
```

---

## 🔍 Code Quality Standards

**Before committing, check:**

### Go
```bash
go fmt ./...
golangci-lint run ./...
go test -v -race ./...
```

### Python
```bash
black tool.py
isort tool.py
flake8 tool.py
mypy tool.py --strict
pytest tests/ -v
```

### Shell
```bash
shellcheck tool.sh
```

### All Languages
- [ ] Comprehensive error messages
- [ ] Type hints/signatures documented
- [ ] No external network calls without explicit handling
- [ ] No hardcoded paths (use relative or config)
- [ ] Proper JSON output (test with `jq`)

---

## 🧪 Testing Approach

### Unit Tests
- Test each tool individually
- Mock external dependencies
- Test both success and error paths

### Integration Tests
```python
# Example: Test plugin with naitv-mcp
import json
import subprocess

def test_plugin_integration():
    input_data = {"root_path": "/tmp/test-repo", "symbol_name": "MyFunc"}
    result = subprocess.run(
        ["python", "tool.py"],
        input=json.dumps(input_data) + "\n",
        capture_output=True,
        text=True,
        timeout=10
    )
    assert result.returncode == 0
    output = json.loads(result.stdout)
    assert "file" in output
    assert "line" in output
```

### Real-World Testing
- Test against actual codebases (this project, Linux kernel, Django, etc.)
- Verify correctness of results
- Measure performance on real data

---

## 📝 Common Patterns

### Pattern 1: Language Detection
```python
def detect_language(file_path):
    ext = file_path.split('.')[-1]
    mapping = {
        'go': 'go',
        'py': 'python',
        'js': 'javascript',
        'ts': 'typescript',
    }
    return mapping.get(ext, 'unknown')
```

### Pattern 2: Safe File Walking
```python
import os
def walk_safe(root_path):
    skip_dirs = {'.git', 'node_modules', '__pycache__', '.venv', 'vendor'}
    for dirpath, dirnames, filenames in os.walk(root_path):
        # Skip hidden/vendor directories
        dirnames[:] = [d for d in dirnames if not d.startswith('.') and d not in skip_dirs]
        yield from filenames
```

### Pattern 3: Timeout Enforcement
```python
import signal
def timeout_handler(signum, frame):
    raise TimeoutError("Operation exceeded timeout")

signal.signal(signal.SIGALRM, timeout_handler)
signal.alarm(30)  # 30 second timeout
try:
    # expensive operation
    pass
finally:
    signal.alarm(0)  # cancel alarm
```

### Pattern 4: JSON Output
```python
import json
import sys

result = {
    "status": "success",
    "data": {...}
}
print(json.dumps(result))  # Single line to stdout
sys.exit(0)
```

### Pattern 5: Error Handling
```python
import sys
import json

try:
    # operation
    pass
except Exception as e:
    error = {
        "status": "error",
        "error_code": "INTERNAL_ERROR",
        "message": str(e)
    }
    print(json.dumps(error), file=sys.stderr)
    sys.exit(1)
```

---

## 🐛 Debugging Tips

### Plugin Not Found
```bash
# Check plugins directory
ls -la naitv-mcp-plugins/plugins/

# Check plugin.json
cat naitv-mcp-plugins/plugins/PLUGIN_NAME/plugin.json | jq

# Check naitv-mcp logs
tail -f /tmp/naitv-mcp.log
```

### Plugin Timeout
- Reduce data size in tests
- Check for infinite loops
- Add progress logging (stderr)

### JSON Parse Error
```bash
# Test I/O manually
echo '{"root_path": "/tmp"}' | python tool.py | jq
```

### Performance Issue
```bash
# Profile Python
python -m cProfile tool.py < input.json

# Profile Go
go test -bench=. -cpuprofile=cpu.prof
go tool pprof cpu.prof
```

---

## 📚 Reference Materials

### Gemini's PDF
- Pages 1-5: Context dumping + self-correction + architectural maps
- Pages 6-7: Plugin architecture proposal
- Pages 8-14: Detailed implementation of structural-anchor + symbol-navigator

### Key Concepts
- **Plugin Manifest:** Declares what tools a plugin provides + how to execute it
- **Subprocess I/O:** JSON stdin/stdout/stderr contract
- **Caching:** File hash invalidation for deterministic results
- **LSP:** Language Server Protocol for symbol resolution
- **AST:** Abstract Syntax Tree for code analysis

### External Resources
- [Model Context Protocol](https://modelcontextprotocol.io/)
- [Language Server Protocol](https://microsoft.github.io/language-server-protocol/)
- [Go Parser](https://pkg.go.dev/go/parser)
- [Python AST](https://docs.python.org/3/library/ast.html)

---

## 🎓 Learning Objectives Per Phase

### Phase 1
- Understand plugin manifest structure
- Subprocess execution model (JSON I/O)
- Plugin loader + registry mechanism

### Phase 2
- AST parsing in Go/Python
- Caching strategies
- Performance optimization

### Phase 3
- Linter integration (golangci-lint, pylint, eslint)
- Test framework integration
- Feedback loop design

### Phase 4
- Document parsing (markdown, HTML)
- Semantic search (ripgrep)
- Git history analysis

### Phase 5
- Workflow design
- LLM interaction patterns
- Multi-step task orchestration

---

## ✅ Completion Criteria Per Phase

### Phase 1 Done When
- [ ] Dynamic plugin loading works
- [ ] Subprocess execution with JSON I/O is robust
- [ ] At least 3 template examples exist
- [ ] plugin-manager bootstrap plugin works
- [ ] No breaking changes to naitv-mcp

### Phase 2 Done When
- [ ] structural-anchor < 2s on large repos
- [ ] symbol-navigator finds all symbols accurately
- [ ] context-injector returns model-friendly slices
- [ ] All three plugins work together
- [ ] Caching measurably improves performance

### Phase 3 Done When
- [ ] Models receive linter errors and auto-correct
- [ ] TDD runner generates + runs tests successfully
- [ ] Feedback loop demonstrates < 3 iterations to success
- [ ] Error messages are actionable for LLM

### Phase 4 Done When
- [ ] Doc scraper fetches external docs successfully
- [ ] Workspace search finds relevant code patterns
- [ ] Cache manager reduces latency 2-5x
- [ ] Error matcher connects errors to historical solutions

### Phase 5 Done When
- [ ] Models create and follow implementation plans
- [ ] Golden examples inject correctly
- [ ] Type stubs eliminate import errors
- [ ] Surgical patches enforce minimal changes
- [ ] End-to-end workflow demonstrates all capabilities

---

## 🚨 Common Pitfalls to Avoid

1. **Don't modify naitv-mcp core without good reason**
   - Plugins go in `naitv-mcp-plugins`
   - Core changes must be backward compatible

2. **Don't hardcode paths**
   - Use relative paths or accept via plugin.json config

3. **Don't ignore error cases**
   - Every tool must handle errors gracefully
   - Return structured error JSON

4. **Don't skip tests**
   - Plugin untested = plugin broken
   - CI/CD should run tests automatically

5. **Don't bundle bloated dependencies**
   - Keep `requirements.txt` minimal
   - Use Go standard library where possible

6. **Don't output to stdout except JSON**
   - All structured output: stdout (JSON)
   - All diagnostics: stderr (free-form)

7. **Don't trust input**
   - Validate all inputs with descriptive errors
   - Bound outputs (max lines, max size)

8. **Don't forget documentation**
   - README + examples mandatory
   - Each tool must be documented

---

## 🎯 Success Looks Like

When a plugin is done:
- ✅ Code reviewed and merged
- ✅ Tests passing (100% locally, CI/CD automated)
- ✅ Performance targets met or exceeded
- ✅ Documentation complete with examples
- ✅ Works end-to-end with naitv-mcp
- ✅ Ready for integration with other plugins

When the whole plan is done:
- ✅ All 5 phases complete
- ✅ 15+ plugins implemented
- ✅ Real project using the system
- ✅ Local model solving non-trivial tasks
- ✅ Community contributors adding plugins
- ✅ Measurable improvement in model quality

---

## 🆘 When Stuck

1. **Reread the relevant phase description** in NAITV_MCP_EXECUTION_PLAN.md
2. **Check the Analysis document** for the "why" behind decisions
3. **Look at similar plugins** for patterns
4. **Test in isolation** before integration
5. **Ask yourself:** "Is this changing the plugin, or changing naitv-mcp?"

---

**Last Updated:** 2025-01-15  
**For:** naitv-mcp Enhancement via Cursor (auto mode)  
**Status:** Ready to begin Phase 1
