# VHS tape for the simple palette example.
# Regenerate with: task demo NAME=palette/simple   (or vhs demo.tape from this dir)

Output demo.gif

Set FontSize 14
Set Width 720
Set Height 360
Set TypingSpeed 80ms
Set PlaybackSpeed 1.0
Set Theme "Catppuccin Mocha"

Hide
Type "go run ."
Enter
Sleep 1500ms
Show

# Land on default selection (Quit). Fuzzy-filter without any prefix.
Sleep 800ms
Type "open"
Sleep 600ms
Backspace 4
Sleep 400ms
Type "format"
Sleep 600ms
Enter
Sleep 800ms

# Now run Quit by name.
Type "quit"
Sleep 500ms
Enter
Sleep 600ms
