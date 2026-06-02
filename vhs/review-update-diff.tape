# review-update-diff.tape — switch to review tab, inspect an update proposal
# showing the diff between old and new values, then approve.
#
# Regen with: vhs vhs/review-update-diff.tape
# Output:     screenshots/review-update-diff.gif

Output screenshots/review-update-diff.gif

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

# Entries tab visible.
Sleep 1.5s

# Switch to Review tab.
Tab
Sleep 800ms

# Review tab shows proposals. Navigate to the second proposal
# (incident-response workflow).
Down
Sleep 500ms

# Read the diff — target_id is empty on both demo proposals so this is a new
# entry proposal. Pause so viewer reads the detail panel.
Sleep 3s

# Navigate back to first proposal (naitv-mcp repo).
Up
Sleep 500ms

# Approve it.
Type "a"
Sleep 1s

# Navigate to next pending proposal and approve.
Type "a"
Sleep 1s

Sleep 2s

Hide
Type "q"
Sleep 500ms
