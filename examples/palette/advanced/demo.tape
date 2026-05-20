# VHS tape for the advanced palette example.
# Regenerate with: task demo NAME=palette/advanced

Output demo.gif

Set FontSize 14
Set Width 760
Set Height 460
Set TypingSpeed 90ms
Set PlaybackSpeed 1.0
Set Theme "Catppuccin Mocha"

Hide
Type "go run ."
Enter
Wait+Screen /mode:/
Show

# Search mode: no prefix triggers debounced async search.
Type "Bubbles"
Wait+Screen /charmbracelet\/bubbles/
Enter
Sleep 1500ms
Backspace 7

# Facet: kind:doc narrows search to documentation results.
Type "kind:"
Sleep 800ms
Type "doc Elm"
Wait+Screen /elm-lang/
Enter
Sleep 1500ms
Backspace 12

# Command mode: `>` filters the local command list.
Type "> Open"
Sleep 400ms
Enter
Sleep 1500ms
Backspace 6

# Files mode: `@` lists files; browse with arrows, then filter.
Type "@"
Sleep 400ms
Down@300ms 6
Sleep 600ms
Type "README"
Sleep 400ms
Enter
Sleep 600ms
