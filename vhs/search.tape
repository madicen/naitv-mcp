# search.tape — press / to open search, type a query, esc to clear.
#
# Regen with: vhs vhs/search.tape
# Output:     screenshots/search.gif

Output screenshots/search.gif

Set FontSize 14
Set Width 1300
Set Height 700
Set Theme "Builtin Dark"
Set TypingSpeed 65ms
Set PlaybackSpeed 1.0

Hide
Type "NAITV_MCP_DEMO_DIR=tmp/demo ./naitv-mcp --demo"
Enter
Sleep 1.5s
Show

# Full entries list visible.
Sleep 1.5s

# Open search.
Type "/"
Sleep 500ms

# Type a search query.
Type "deploy"
Sleep 800ms

# Results narrow to matching entries.
Sleep 1.5s

# Clear search with Escape.
Escape
Sleep 800ms

# Full list restored.
Sleep 1.5s

# Search again with different query.
Type "/"
Sleep 400ms
Type "go"
Sleep 800ms

Sleep 1.5s

# Clear again.
Escape
Sleep 1s

Hide
Type "q"
Sleep 500ms
