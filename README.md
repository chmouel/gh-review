# gh-prreview

A GitHub CLI extension to apply review comments and suggestions directly to
your local code.

## Overview

`gh-prreview` helps you applying the Github code review locally. It fetches review
comments from a pull request, extracts suggested changes, and allows you to
apply them interactively to your local files.

- Fetch review comments from pull requests
- View GitHub suggestions in your terminal
- Apply suggested changes directly to your local files
- Interactively choose which suggestions to apply

## Installation

```bash
gh extension install chmouel/gh-prreview
```

Or build from source:

```bash
git clone https://github.com/chmouel/gh-prreview
cd gh-prreview
go build
gh extension install .
```

## Usage

### Global options

All commands accept `-R, --repo <owner/repo>` to target a different repository
than the current directory. Use `--debug` where available for verbose logs.

### List review comments

```bash
# List unresolved review comments (default)
gh prreview list [PR_NUMBER]

# List all review comments including resolved/done ones
gh prreview list --all [PR_NUMBER]
```

If no PR number is provided, it will use the PR for the current branch.

Available flags:

- `--all` – include resolved/done suggestions in the output
- `--debug` – enable extra logging (printed to stderr)
- `--llm` – output in a machine-friendly format for LLM processing

### Apply review suggestions

```bash
# Interactive mode - review and apply suggestions one by one
gh prreview apply [PR_NUMBER]

# Apply all suggestions automatically
gh prreview apply --all [PR_NUMBER]

# Apply suggestions for a specific file
gh prreview apply --file path/to/file.go [PR_NUMBER]

# Include resolved/done suggestions
gh prreview apply --include-resolved [PR_NUMBER]

# Enable verbose logs
gh prreview apply --debug [PR_NUMBER]
```

> The apply command requires a clean working tree. Stash or commit your changes
> before running it.

### AI-assisted application

Use AI to intelligently apply suggestions that might have conflicts or outdated context:

```bash
# Interactive mode with AI option available
gh prreview apply [PR_NUMBER]
# Then select 'a' when prompted to use AI for that suggestion
# You can review the AI-generated patch and optionally edit it in $EDITOR
# before applying

# Auto-apply all suggestions using AI
gh prreview apply --ai-auto [PR_NUMBER]

# Use specific AI model
gh prreview apply --ai-auto --ai-model gemini-1.5-flash [PR_NUMBER]

# Force a specific AI provider
gh prreview apply --ai-auto --ai-provider gemini [PR_NUMBER]

# Provide API key via flag instead of environment variable
gh prreview apply --ai-auto --ai-token YOUR_API_KEY [PR_NUMBER]

# Load a custom prompt template
gh prreview apply --ai-template ./path/to/template.tmpl [PR_NUMBER]
```

**Prerequisites:** Set `GEMINI_API_KEY` or `GOOGLE_API_KEY` environment
variable, or use `--ai-token` flag.

See [docs/AI_INTEGRATION.md](docs/AI_INTEGRATION.md) for detailed AI feature documentation.

### Resolve review threads

```bash
# Resolve a comment thread (PR inferred from current branch)
gh prreview resolve <COMMENT_ID>

# Resolve a comment thread while specifying the PR
gh prreview resolve <PR_NUMBER> <COMMENT_ID>

# Mark the thread as unresolved instead
gh prreview resolve --unresolve <PR_NUMBER> <COMMENT_ID>

# Enable verbose logging when resolving
gh prreview resolve --debug <PR_NUMBER> <COMMENT_ID>
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
