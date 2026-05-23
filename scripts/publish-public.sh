#!/bin/bash
# Exit on error
set -e

# Get the name of the current active branch
CURRENT_BRANCH=$(git symbolic-ref --short HEAD)

echo "=== Publishing clean snapshot to public repository ==="

# 1. Create the temporary clean branch
echo "Creating temporary orphan branch..."
git checkout --orphan temp-public

# 2. Commit the current state of files
echo "Committing current files..."
git commit -m "initial release"

# 3. Force-push to overwrite the public repository's main branch
echo "Force-pushing to public/main..."
git push -f public temp-public:main

# 4. Switch back to the original branch
echo "Switching back to original branch ($CURRENT_BRANCH)..."
git checkout "$CURRENT_BRANCH"

# 5. Delete the temporary branch locally
echo "Cleaning up temporary branch..."
git branch -D temp-public

echo "=== Successfully published to public main ==="
