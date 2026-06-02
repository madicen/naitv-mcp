# edit-entry.tape — select an entry, press e to edit, modify body, save.
#
# Regen with: vhs vhs/edit-entry.tape
# Output:     screenshots/edit-entry.gif

Output screenshots/edit-entry.gif

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

# Entries list visible with demo data.
Sleep 1.5s

# Navigate to the "deploy" workflow entry.
Down
Sleep 300ms
Down
Sleep 300ms
Sleep 500ms

# Open edit form.
Type "e"
Sleep 800ms

# Tab to the body field.
Tab
Tab
Sleep 400ms

# Append to the body (single-line field; cursor is already at the end).
Type " Also update CHANGELOG.md before tagging."
Sleep 500ms

# Save with ctrl+s.
Ctrl+S
Sleep 1s

# Updated entry visible in list.
Sleep 2s

Hide
Type "q"
Sleep 500ms
