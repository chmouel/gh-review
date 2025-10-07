# gh-review

A GitHub CLI extension to apply review comments and suggestions directly to your local code.

## Overview

`gh-review` help you applying the Github code review locally. It fetches review
comments from a pull request, extracts suggested changes, and allows you to
apply them interactively to your local files.

- Fetch review comments from pull requests
- View GitHub suggestions in your terminal
- Apply suggested changes directly to your local files
- Interactively choose which suggestions to apply

## Installation

```bash
gh extension install chmouel/gh-review
```

Or build from source:

```bash
git clone https://github.com/chmouel/gh-review
cd gh-review
go build
gh extension install .
```

## Usage

### List review comments

```bash
# List unresolved review comments (default)
gh review list [PR_NUMBER]

# List all review comments including resolved/done ones
gh review list --all [PR_NUMBER]
```

If no PR number is provided, it will use the PR for the current branch.

### Apply review suggestions

```bash
# Interactive mode - review and apply suggestions one by one
gh review apply [PR_NUMBER]

# Apply all suggestions automatically
gh review apply --all [PR_NUMBER]

# Apply suggestions for a specific file
gh review apply --file path/to/file.go [PR_NUMBER]

# Include resolved/done suggestions
gh review apply --include-resolved [PR_NUMBER]
```

## Features

- 🔍 Fetches review comments from GitHub PRs
- 💡 Parses GitHub suggestion blocks
- ✨ Interactive UI for reviewing changes with colored diff output
- 🔗 Clickable links (OSC8) to view comments on GitHub
- 🎯 Apply changes directly to local files
- 🔄 Handles multi-line suggestions
- ✅ Filters out resolved/done suggestions by default
- ⚠️  Detects conflicts with local changes

## How it works

GitHub allows reviewers to suggest code changes using the suggestion feature:

\`\`\`suggestion
// Suggested code here
\`\`\`

This plugin:

1. Fetches all review comments with suggestions
2. Parses the suggestion blocks
3. Shows you a preview of the changes
4. Applies them to your local files when confirmed

## Requirements

- GitHub CLI (`gh`) installed and authenticated
- Git repository with a remote on GitHub
- Active pull request

## License

[Apache License 2.0](./LICENSE)
