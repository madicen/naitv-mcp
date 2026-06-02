# review-approve.tape — switch to review tab, approve a new proposal.
#
# Regen with: vhs vhs/review-approve.tape
# Output:     screenshots/review-approve.gif

Output screenshots/review-approve.gif

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

# Entries tab visible; badge shows pending proposals.
Sleep 2s

# Switch to Review tab with Tab key.
Tab
Sleep 800ms

# Review tab shows pending proposals.
Sleep 2s

# Select first proposal (already selected).
# Approve it with 'a'.
Type "a"
Sleep 1s

# Status message: "Proposal approved."
Sleep 1.5s

# Switch back to Entries tab to see the new entry.
Tab
Sleep 1s

Sleep 2s

Hide
Type "q"
Sleep 500ms
