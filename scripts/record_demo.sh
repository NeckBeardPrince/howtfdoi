#!/bin/bash
# Script to create demo GIF for howtfdoi
# Requires: vhs (https://github.com/charmbracelet/vhs)

set -e

echo "Recording howtfdoi demo using VHS..."

# Check if vhs is installed
if ! command -v vhs &>/dev/null; then
	echo "VHS is not installed. Install it with:"
	echo "  brew install vhs"
	echo "or"
	echo "  go install github.com/charmbracelet/vhs@latest"
	exit 1
fi

# Check if the tape file exists
TAPE_FILE="$(dirname "$0")/demo.tape"
if [ ! -f "$TAPE_FILE" ]; then
	echo "Error: demo.tape file not found at $TAPE_FILE"
	exit 1
fi

# Record the demo
echo "Recording demo from $TAPE_FILE..."
vhs "$TAPE_FILE"

echo "Demo GIF created: demo.gif"
echo ""
echo "To optimize the GIF size, run:"
echo "  gifsicle -O3 --colors 128 demo.gif -o demo-optimized.gif"
echo ""
echo "Or use gifski for better quality:"
echo "  gifski -o demo-optimized.gif demo.gif --quality 80"
