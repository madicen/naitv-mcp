# create-entry.tape — press n to open new entry form, fill it out, save.
#
# Regen with: vhs vhs/create-entry.tape
# Output:     screenshots/create-entry.gif

Output screenshots/create-entry.gif

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

# Entries list is visible.
Sleep 1.5s

# Open new entry form.
Type "n"
Sleep 800ms

# Fill in the kind field.
Type "tool"
Tab
Sleep 400ms

# Fill in the name field.
Type "ripgrep"
Tab
Sleep 400ms

# Fill in the body.
Type "Fast search tool. Use rg instead of grep."
Tab
Sleep 400ms

# Add a tag.
Type "cli"
Sleep 400ms

# Save the entry with ctrl+s.
Ctrl+S
Sleep 1s

# Entry should now appear in the list.
Sleep 2s

Hide
Type "q"
Sleep 500ms
