# VHS tape for the overlay palette example.
# Regenerate with: task demo NAME=palette/overlay

Output demo.gif

Set FontSize 14
Set Width 800
Set Height 500
Set TypingSpeed 80ms
Set PlaybackSpeed 1.0
Set Theme "Catppuccin Mocha"

Hide
Type "go run ."
Enter
Wait+Screen /Lorem ipsum/
Show

# Background is visible. Open the palette.
Sleep 1000ms
Ctrl+K
Sleep 800ms

# Filter and pick a command — the palette auto-closes.
Type "format"
Sleep 600ms
Enter
Sleep 2000ms
