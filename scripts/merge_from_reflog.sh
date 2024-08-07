#!/bin/bash

# Check if the correct number of arguments are provided
if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <commit-hash> <branch-name>"
    exit 1
fi

# Get the commit hash and branch name from the arguments
commit_hash=$1
branch_name=$2

# Create and checkout a new branch
echo "Creating and checking out new branch: $branch_name"
git checkout -b "$branch_name"

# Generate the patch from the specified commit to HEAD
echo "Generating patch from commit $commit_hash to HEAD"
git format-patch -1 "$commit_hash" --stdout > changes.patch

# Apply the patch with the three-way merge option
echo "Applying patch..."
if git am --3way < changes.patch; then
    echo "Patch applied successfully"
else
    echo "Merge conflicts detected. Please resolve them manually."
    git status
    echo "After resolving conflicts, run the following commands to continue:"
    echo "  git add ."
    echo "  git am --continue"
fi

# Find all .rej files and process them
echo "Processing rejected patches..."
find . -name "*.rej" | while read -r rej_file; do
    # Extract the original file path
    original_file="${rej_file%.rej}"

    # Check if the original file exists
    if [ ! -f "$original_file" ]; then
        # If the original file does not exist, create it
        echo "Creating new file: $original_file"
        touch "$original_file"
    fi

    # Append the content of the .rej file to the original file
    echo "Applying rejected patch to $original_file"
    cat "$rej_file" >> "$original_file"

    # Remove the .rej file after applying the patch
    rm "$rej_file"
done

# Stage all changes
echo "Staging all changes..."
git add .

# Check for conflicts again, to ensure the manual changes are staged correctly
conflicts=$(git ls-files -u | wc -l)
if [ "$conflicts" -gt 0 ]; then
    echo "Merge conflicts detected. Please resolve them manually."
    git status
    echo "After resolving conflicts, run the following commands to commit the changes:"
    echo "  git add ."
    echo "  git commit -m 'Resolved merge conflicts from reflog commit $commit_hash'"
else
    # Commit the changes
    echo "Committing changes..."
    git commit -m "Merged changes from reflog commit $commit_hash, applied rejected patches and added missing files"
fi

# Clean up
echo "Cleaning up..."
rm changes.patch

echo "Done!"
