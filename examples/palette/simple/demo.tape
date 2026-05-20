# VHS tape for the simple palette example.
# Regenerate with: task demo NAME=palette/simple   (or vhs demo.tape from this dir)

Output demo.gif

Set FontSize 14
Set Width 720
Set Height 480
Set TypingSpeed 90ms
Set PlaybackSpeed 1.0
Set Theme "Catppuccin Mocha"

Hide
Type "go run ."
Enter
Wait+Screen /Open file/
Show

# Page through the commands with the arrow keys.
Sleep 800ms
Down@300ms 5
Sleep 800ms
Up@300ms 5
Sleep 600ms

# Filter to Quit and run it.
Type "quit"
Sleep 600ms
Enter
Sleep 600ms
