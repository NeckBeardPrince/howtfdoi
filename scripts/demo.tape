# VHS tape file for howtfdoi demo
# https://github.com/charmbracelet/vhs

# Setup
Output ../assets/demo.gif
Set Shell fish
Set Theme "tokyonight-storm"
Set WindowBar Colorful
Set Framerate 60
Set CursorBlink true
Set FontSize 14
Set FontFamily "JetBrainsMono Nerd Font"
Set Width 1200
Set Height 800
Set Padding 20
Set TypingSpeed 50ms
Set LineHeight 1.4

# Start recording
# Basic usage
Type "howtfdoi list all docker containers"
Enter
Sleep 3.5s

Hide
Type "clear"
Enter

Show
Sleep 1.5s
Type "howtfdoi use nmap to scan all ip addresses and ports but NOT ping?"
Enter
Sleep 3.5s

Hide
Type "clear"
Enter

# Copy to clipboard
Show
Sleep 1.5s
Type "howtfdoi -c find large files"
Enter
Sleep 4.5s

Hide
Type "clear"
Enter

# Show examples
Show
Sleep 1.5s
Type "howtfdoi -e compress files with tar"
Enter
Sleep 6s

Hide
Type "clear"
Enter

Show
Sleep 1.5s
Type "howtfdoi -e copy a file from my local to a Kubernetes pod?"
Enter
Sleep 4.5s

Hide
Type "clear"
Enter

# Interactive mode demo
Show
Sleep 1.5s
Type "howtfdoi"
Enter
Sleep 1.5s
Type "how to count lines in files"
Enter
Sleep 4s
Type "-e find process using port 8080"
Enter
Sleep 6s
Type "exit"
Enter
Sleep 4s
