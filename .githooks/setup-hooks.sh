#!/bin/sh
# Activate version-controlled Git hooks for GoContact Studio on POSIX systems
set -e

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$REPO_ROOT"

echo "Configuring Git hooks path to .githooks..."
git config core.hooksPath .githooks

chmod +x .githooks/pre-commit .githooks/commit-msg .githooks/pre-push .githooks/post-merge

echo "[SUCCESS] Git hooks successfully activated (.githooks)!"
