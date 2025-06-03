#!/bin/bash

# Generate coverage badge for README
# Usage: ./scripts/coverage-badge.sh

# Run tests with coverage
echo "Running tests with coverage..."
go test -coverprofile=coverage.out ./... > /dev/null 2>&1

# Extract coverage percentage
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')

# Round to integer
COVERAGE_INT=$(printf "%.0f" "$COVERAGE")

# Determine color based on coverage
if [ $COVERAGE_INT -ge 80 ]; then
    COLOR="brightgreen"
elif [ $COVERAGE_INT -ge 60 ]; then
    COLOR="green"
elif [ $COVERAGE_INT -ge 40 ]; then
    COLOR="yellow"
elif [ $COVERAGE_INT -ge 20 ]; then
    COLOR="orange"
else
    COLOR="red"
fi

# Create badge URL
BADGE_URL="https://img.shields.io/badge/coverage-${COVERAGE_INT}%25-${COLOR}"

echo "Coverage: ${COVERAGE_INT}%"
echo "Badge URL: ${BADGE_URL}"
echo ""
echo "Add this to your README.md:"
echo "![Coverage](${BADGE_URL})"