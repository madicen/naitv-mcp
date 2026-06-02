# delivery-toggle.tape — open with --demo, toggle an entry between init and
# on-demand delivery with `i`, showing the list glyph and detail pane update.
#
# Regen with: vhs vhs/delivery-toggle.tape
# Output:     screenshots/delivery-toggle.gif

Output screenshots/delivery-toggle.gif

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

# Entries list. Each row shows a delivery glyph: ● init, ○ on-demand.
Sleep 2.5s

# Move down to a rule/workflow entry; the detail pane shows its delivery mode.
Down
Sleep 600ms
Down
Sleep 1.2s

# Toggle this entry to on-demand: glyph turns ○ and the detail "Deliver:" line
# updates. On-demand entries are excluded from the initialize bundle.
Type "i"
Sleep 1.8s

# Toggle it back to init (●), so it is included in initialization again.
Type "i"
Sleep 1.5s

# Do the same on another entry to show it is per-entry.
Down
Sleep 600ms
Type "i"
Sleep 1.8s

Sleep 1.5s

# Exit cleanly.
Hide
Type "q"
Sleep 500ms
