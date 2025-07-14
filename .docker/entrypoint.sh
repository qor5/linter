#!/bin/bash
set -e

# Default command is fix-lint
COMMAND=${1:-fix-lint}

cd /app

echo "📦 Using local Go mod cache, skipping go mod download..."

# Use custom golangci-lint - fail if not available
GOLANGCI_LINT_CMD="qor5-linter"
CONFIG_FILE=".golangci.yml"

if ! command -v "$GOLANGCI_LINT_CMD" &> /dev/null; then
    echo "❌ Custom golangci-lint not found! Build failed."
    echo "Expected: $GOLANGCI_LINT_CMD"
    exit 1
fi

echo "✅ Using custom golangci-lint with custom plugins"
echo "Using: $GOLANGCI_LINT_CMD with config: $CONFIG_FILE"

case "$COMMAND" in
    lint)
        echo "🔍 Running linter checks..."
        $GOLANGCI_LINT_CMD run --config="$CONFIG_FILE" --timeout 10m0s || (echo "❌ Linting failed"; exit 1)
        echo "✅ Linting passed"
        ;;
    fix-lint)
        echo "🔧 Fixing lint issues..."
        # First run with --fix flag
        $GOLANGCI_LINT_CMD run --config="$CONFIG_FILE" --fix || true
        
        echo "🔧 Fixing unused parameters..."
        # Fix unused parameters
        TERM=dumb $GOLANGCI_LINT_CMD run --config="$CONFIG_FILE" | grep -E "^.*:[0-9]+:[0-9]+: unused-parameter: parameter '([^']*)'" | while read -r line; do
            file=$(echo "$line" | awk -F: '{print $1}')
            line_num=$(echo "$line" | awk -F: '{print $2}')
            param=$(echo "$line" | grep -oP "parameter '\K[^']+")
            if [ ! -z "$file" ] && [ ! -z "$line_num" ] && [ ! -z "$param" ]; then
                echo "Fixing unused parameter '$param' in $file at line $line_num"
                sed -i "${line_num}s/\b${param}\b\s\+/_ /g" "$file"
            fi
        done || true
        
        echo "✅ Lint fixes applied"
        echo "🔍 Verifying fixes by running linter again..."
        
        # Final lint check
        $GOLANGCI_LINT_CMD run --config="$CONFIG_FILE" --timeout 10m0s || (echo "❌ Linting still has issues"; exit 1)
        echo "✅ All lint issues fixed"
        ;;
    *)
        echo "❌ Unknown command: $COMMAND"
        echo "Available commands: lint, fix-lint"
        exit 1
        ;;
esac 
