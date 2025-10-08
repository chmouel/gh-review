# gh-review

A GitHub CLI extension to apply review comments and suggestions directly to your local code.

## Overview

`gh-review` helps you applying the Github code review locally. It fetches review
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

### AI-assisted application

Use AI to intelligently apply suggestions that might have conflicts or outdated context:

```bash
# Interactive mode with AI option available
gh review apply [PR_NUMBER]
# Then select 'a' when prompted to use AI for that suggestion
# You can review the AI-generated patch and optionally edit it in $EDITOR before applying

# Auto-apply all suggestions using AI
gh review apply --ai-auto [PR_NUMBER]

# Use specific AI model
gh review apply --ai-auto --ai-model gemini-1.5-flash [PR_NUMBER]

# Provide API key via flag instead of environment variable
gh review apply --ai-auto --ai-token YOUR_API_KEY [PR_NUMBER]
```

**Prerequisites:** Set `GEMINI_API_KEY` or `GOOGLE_API_KEY` environment variable, or use `--ai-token` flag.

See [docs/AI_INTEGRATION.md](docs/AI_INTEGRATION.md) for detailed AI feature documentation.

## Features

- 🔍 Fetches review comments from GitHub PRs
- 💡 Parses GitHub suggestion blocks
- ✨ Interactive UI for reviewing changes with colored diff output
- 🔗 Clickable links (OSC8) to view comments on GitHub
- 🎯 Apply changes directly to local files
- 🔄 Handles multi-line suggestions
- ✅ Filters out resolved/done suggestions by default
- ⚠️  Detects conflicts with local changes
- 🤖 AI-powered suggestion application (adapts to code changes)
- ✔️  Mark review threads as resolved after applying suggestions

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
