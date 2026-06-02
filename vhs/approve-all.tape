# approve-all.tape — press A on the review tab to approve all proposals at once.
#
# Regen with: vhs vhs/approve-all.tape
# Output:     screenshots/approve-all.gif

Output screenshots/approve-all.gif

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

# Entries tab; badge shows 2 pending.
Sleep 1.5s

# Switch to Review tab.
Tab
Sleep 800ms

# Both proposals visible.
Sleep 2s

# Press shift+A to approve all.
Type "A"
Sleep 1s

# Status: "All proposals approved." Badge disappears.
Sleep 2s

# Switch to Entries tab to see both new entries added.
Tab
Sleep 1s
Sleep 2.5s

Hide
Type "q"
Sleep 500ms
