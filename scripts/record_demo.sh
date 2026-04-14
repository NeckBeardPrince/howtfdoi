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

# Read the Output path from the tape so the success message and optimization
# commands stay in sync with the actual VHS output path.
TAPE_DIR="$(dirname "$TAPE_FILE")"
OUTPUT_PATH="$(awk '/^[[:space:]]*Output[[:space:]]+/ {
	sub(/^[[:space:]]*Output[[:space:]]+/, "", $0)
	gsub(/^"|"$/, "", $0)
	print
	exit
}' "$TAPE_FILE")"

if [ -z "$OUTPUT_PATH" ]; then
	OUTPUT_PATH="demo.gif"
fi

case "$OUTPUT_PATH" in
	/*) DEMO_GIF="$OUTPUT_PATH" ;;
	*) DEMO_GIF="$TAPE_DIR/$OUTPUT_PATH" ;;
esac

OPTIMIZED_GIF="${DEMO_GIF%.gif}-optimized.gif"

# Record the demo
echo "Recording demo from $TAPE_FILE..."
vhs "$TAPE_FILE"

echo "Demo GIF created: $DEMO_GIF"
echo ""
echo "To optimize the GIF size, run:"
echo "  gifsicle -O3 --colors 128 \"$DEMO_GIF\" -o \"$OPTIMIZED_GIF\""
echo ""
echo "Or use gifski for better quality:"
echo "  gifski -o \"$OPTIMIZED_GIF\" \"$DEMO_GIF\" --quality 80"
