# browse.tape — open with --demo flag, navigate entries list, switch kind tabs.
#
# Regen with: vhs vhs/browse.tape
# Output:     screenshots/browse.gif

Output screenshots/browse.gif

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

# Opening shot: entries list with seeded demo data.
Sleep 2.5s

# Navigate down through entries.
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 500ms

# Navigate back up.
Up
Sleep 300ms
Up
Sleep 500ms

# Switch to the "repo" kind tab with right arrow.
Right
Sleep 1s

# Switch to "note" kind tab.
Right
Sleep 1s

# Back to all entries.
Left
Sleep 1s

Sleep 2s

# Exit cleanly.
Hide
Type "q"
Sleep 500ms
