# VHS tape for the advanced palette example.
# Regenerate with: task demo NAME=palette/advanced

Output demo.gif

Set FontSize 14
Set Width 760
Set Height 460
Set TypingSpeed 80ms
Set PlaybackSpeed 1.0
Set Theme "Catppuccin Mocha"

Hide
Type "go run ."
Enter
Sleep 1500ms
Show

# Command mode: `>` filters the local command list.
Type ">"
Sleep 400ms
Type "open"
Sleep 800ms
Backspace 5
Sleep 400ms

# Files mode: `@` is sync; results pop instantly.
Type "@palette"
Sleep 1000ms
Backspace 8

# Search mode: no prefix triggers debounced async search.
Type "bubble"
Sleep 1800ms
Backspace 6

# Run the Quit command.
Type ">quit"
Sleep 400ms
Enter
Sleep 600ms
