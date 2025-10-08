# AI-Assisted Suggestion Application

## Overview

The AI integration feature helps apply GitHub review suggestions that would normally fail due to code changes, line shifts, or context mismatches. Instead of requiring exact line matches, AI understands the semantic intent of the suggestion and adapts it to the current codebase state.

## Getting Started

### Prerequisites

1. **Get a Gemini API Key**
   - Visit [Google AI Studio](https://makersuite.google.com/app/apikey)
   - Create an API key for Gemini
   - Set it as an environment variable:
     ```bash
     export GEMINI_API_KEY="your-api-key-here"
     # or
     export GOOGLE_API_KEY="your-api-key-here"
     ```

2. **Install gh-review** (if not already installed)
   ```bash
   gh extension install chmouel/gh-review
   ```

### Quick Start

**Interactive Mode (recommended for first-time users):**
```bash
# Apply suggestions interactively with AI available
gh review apply 123

# When prompted, choose 'a' to use AI for tricky suggestions
Apply this suggestion? [y/s/a/q] (yes/skip/ai-apply/quit) a
```

**Batch Mode (apply all with AI):**
```bash
# Automatically apply all suggestions using AI
gh review apply 123 --ai-auto
```

**With Custom Options:**
```bash
# Use specific model
gh review apply 123 --ai-auto --ai-model gemini-1.5-flash

# Provide API key via flag instead of environment
gh review apply 123 --ai-auto --ai-token YOUR_API_KEY

# Use custom prompt template
gh review apply 123 --ai-template my-custom-prompt.tmpl

# Debug mode to see AI prompts
gh review apply 123 --ai-auto --debug
```

## Design Principles

1. **User-controlled** - AI is never invoked automatically; users explicitly choose when to use it
2. **Interface-based** - Clean abstraction allows multiple AI providers with minimal changes
3. **Library-first** - Use established AI SDKs; don't reimplement provider clients
4. **Transparent** - Users see what the AI is doing and can review before application
5. **Configurable** - Prompts and behavior can be customized without code changes

## How It Works

### User Flow

**Interactive Mode:**
When applying suggestions, users get an additional menu option:

```
Apply this suggestion? [y/s/a/q] (yes/skip/ai-apply/quit)
```

- `y` - Traditional application using exact line matching + git apply
- `s` - Skip this suggestion
- `a` - **AI Apply** - Let AI intelligently adapt and apply the suggestion (shows patch and prompts for confirmation)
- `q` - Quit

**Batch Mode:**

```bash
# Apply all suggestions using AI automatically (shows patches but doesn't prompt)
gh review apply --ai-auto

# Combine with filters
gh review apply --ai-auto --file src/main.go
```

### Processing Flow

1. **User selects AI application** (either via 'a' in menu or `--ai-auto` flag)

2. **Context gathering** - System collects all relevant information:
   - Review comment and suggestion code
   - Current file content (full file)
   - Original diff hunk from when review was created
   - Expected vs actual code at target location
   - File metadata (language, path)

3. **Prompt construction** - Build AI prompt from template with gathered context

4. **AI invocation** - Call configured AI provider with the prompt

5. **Response parsing** - Extract generated patch, explanation, and warnings from AI response

6. **User review** - Display:
   - AI's analysis and explanation
   - Confidence level
   - Any warnings identified
   - The complete generated patch (syntax highlighted)
   - **Confirmation prompt** - Ask user to approve, reject, or edit before applying (unless `--ai-auto`)

7. **Optional editing** - If user selects 'edit', open patch in `$EDITOR` for manual adjustments, then re-display and re-prompt

8. **Patch application** - Apply the AI-generated (or edited) unified diff using `git apply` (only if approved)

9. **Verification** - Show the actual changes made via git diff

### When AI Helps

AI-assisted application excels when:

- **Lines have shifted** - Code moved due to unrelated changes
- **Code has evolved** - Minor refactoring changed variable names or structure
- **Context differs** - Surrounding code changed but suggestion still applies
- **Semantic matching** - Need to find functionally equivalent code, not just exact text match

## Architecture

### Module Structure

```
pkg/ai/
├── provider.go        # AIProvider interface + types
├── gemini.go          # Google Gemini implementation
├── prompts.go         # Configurable prompt templates
└── config.go          # Provider configuration and loading
```

### AI Provider Interface

The system defines a clean interface that any AI provider must implement:

**Core contract:**

- Accept a suggestion request with full context
- Return a unified diff patch ready for `git apply`
- Provide explanation and confidence metadata

**Supported providers:**

- **Gemini** (initial implementation using official Google SDK)
- **OpenAI** (future - GPT-4, etc.)
- **Claude** (future - Anthropic models)
- **Ollama** (future - local/self-hosted models)

Adding a new provider only requires:

1. Implement the `AIProvider` interface
2. Add provider initialization logic
3. Update configuration to recognize the new provider name

### Context Sent to AI

The AI receives comprehensive context to make informed decisions:

#### Review Context

- **Comment body** - Reviewer's explanation of why the change is needed
- **Suggested code** - The exact code the reviewer proposed
- **Original diff hunk** - The unified diff showing what code existed when review was made
- **Metadata** - Comment ID, file path, line numbers

#### Current State Context

- **Full file content** - Complete current version of the file
- **Target line number** - Approximate location where change should apply
- **Expected lines** - What we expected to find based on the diff hunk
- **File language** - Detected programming language (Go, Python, etc.)

#### Failure Context (if applicable)

- **Mismatch details** - What lines didn't match and why
- **Attempted strategies** - What traditional approaches were tried

This rich context allows the AI to:

- Understand the reviewer's intent
- Find the semantically correct location in current code
- Adapt the suggestion to match current code style/structure
- Identify potential conflicts or issues

## Prompt Configuration

### Template System

Prompts are constructed from configurable templates, not hardcoded strings. This allows customization without code changes.

**Template locations (in priority order):**

1. `.github/gh-review/prompts/apply-suggestion.tmpl` (repo-specific)
2. `~/.config/gh-review/prompts/apply-suggestion.tmpl` (user-level)
3. Built-in default (shipped with tool)

### Template Variables

Templates have access to all context variables:

```
{{.ReviewComment}}        - Reviewer's comment text
{{.SuggestedCode}}        - Suggested code block
{{.OriginalDiffHunk}}     - Original diff from review
{{.CurrentFileContent}}   - Full current file
{{.FilePath}}             - File path
{{.FileLanguage}}         - Detected language
{{.TargetLine}}           - Target line number
{{.ExpectedLines}}        - Expected code lines
```

### Example Template Structure

```
You are an expert code reviewer helping apply a GitHub review suggestion.

TASK:
Generate a unified diff patch that applies the suggestion to the current file.
The code may have changed, so adapt the suggestion as needed.

FILE: {{.FilePath}} ({{.FileLanguage}})

REVIEWER'S INTENT:
{{.ReviewComment}}

SUGGESTED CHANGE:
{{.SuggestedCode}}

ORIGINAL CONTEXT (when review was made):
{{.OriginalDiffHunk}}

CURRENT FILE:
{{.CurrentFileContent}}

OUTPUT:
Return JSON with: patch, explanation, confidence (0-1), warnings (array)
```

### Customization Use Cases

**Repository-specific:**

- Add project coding standards to prompt
- Include framework-specific guidance
- Reference architecture documentation

**Language-specific:**

- Different templates for different languages
- Template selection based on `{{.FileLanguage}}`

**Provider-specific:**

- Optimize prompts for specific AI models
- Adjust verbosity based on provider capabilities

## Configuration

### Environment Variables

```bash
# AI Provider Selection
export GH_REVIEW_AI_PROVIDER=gemini  # or: openai, claude, ollama

# API Keys (provider-dependent)
export GEMINI_API_KEY="your-key"
export GOOGLE_API_KEY="your-key"      # Alternative for Gemini
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."

# Model Selection (optional)
export GH_REVIEW_AI_MODEL="gemini-1.5-pro"

# Prompt Configuration (optional)
export GH_REVIEW_PROMPT_DIR="$HOME/.config/gh-review/prompts"
```

### Command-Line Flags

```bash
# AI mode
--ai-auto              # Auto-apply all suggestions with AI
--ai-provider=gemini   # Override provider (gemini, openai, claude, ollama)
--ai-model=gpt-4       # Override model name
--ai-token=YOUR_KEY    # Provide API key via flag (alternative to env var)

# Prompt customization
--ai-template=path/to/template.tmpl  # Use custom prompt template

# Debugging
--debug                # Show AI prompts and responses
```

### Configuration File

Future enhancement: Support config file at `.github/gh-review/config.yaml`:

```yaml
ai:
  provider: gemini
  model: gemini-1.5-pro
  auto_apply: false

  # Confidence threshold (0.0-1.0)
  min_confidence: 0.7

  # Prompt templates by language
  prompts:
    default: prompts/apply-suggestion.tmpl
    go: prompts/apply-suggestion-go.tmpl
    python: prompts/apply-suggestion-python.tmpl
```

## AI Provider Libraries

The implementation uses official, well-maintained AI SDKs:

- **Gemini**: `github.com/google/generative-ai-go` (official Google SDK)
- **OpenAI**: `github.com/sashabaranov/go-openai` (de facto standard for Go)
- **Claude**: `github.com/anthropics/anthropic-sdk-go` (official Anthropic SDK)
- **Ollama**: `github.com/ollama/ollama-go` (official Ollama Go client)

No custom HTTP clients or API wrappers are implemented.

## Response Format

### AI Response Structure

The AI returns a structured response (JSON format):

```json
{
  "patch": "diff --git a/file.go...",
  "explanation": "I adapted the suggestion by...",
  "confidence": 0.95,
  "warnings": [
    "This change might conflict with line 45",
    "Variable name differs from original"
  ]
}
```

### User Feedback

While the AI is processing, a spinner shows progress with the provider and model being used:

```
🤖 Using AI to apply suggestion (gemini/gemini-1.5-pro)...
⠋ Analyzing code and generating patch with gemini (gemini-1.5-pro)...
```

After AI generates a response, the user sees:

```
AI Analysis:
I found the code at line 127 (moved from line 89). I adapted the
suggestion to use the new variable name `userConfig` instead of `cfg`
to match the current codebase style.

⚠️  Warnings:
  • Variable name differs from original review

Confidence: 95%

Generated patch:
diff --git a/config.go b/config.go
--- a/config.go
+++ b/config.go
@@ -127,3 +127,3 @@
-  cfg := loadConfig()
+  userConfig := loadConfig()

Apply this AI-generated patch? [y/n/e] (yes/no/edit) y
✅ Applied with AI

Mark this review thread as resolved? [y/n] y
✅ Review thread marked as resolved
```

**Interactive Options:**
- `y` (yes) - Apply the patch as-is
- `n` (no) - Cancel and skip this suggestion
- `e` (edit) - Open the patch in your `$EDITOR` to make manual adjustments before applying

**Note:** In `--ai-auto` mode, the patch is shown but not prompted - it's applied automatically. The resolve prompt is also skipped in auto mode.

### Marking Threads as Resolved

After successfully applying a suggestion (either manually or via AI), you'll be prompted to mark the review thread as resolved on GitHub. This:
- Updates the PR to show the comment has been addressed
- Helps reviewers track which feedback has been incorporated
- Keeps the review workflow organized
- Only appears for threads that aren't already resolved

The prompt is shown for both:
- Traditional `y` (yes) application
- AI-assisted `a` (ai-apply) application

This transparency helps users:

- Understand what changed
- Trust the AI's decisions
- Catch potential issues
- Learn about code evolution

### Manual Patch Editing

Users can choose to edit the AI-generated patch before applying:

**How it works:**
1. AI generates a patch
2. User selects `e` (edit) at the prompt
3. Patch opens in `$EDITOR` (defaults to `vi` if not set)
4. User makes manual adjustments
5. After saving and closing editor, the edited patch is displayed
6. User is prompted again to apply, cancel, or edit further

**Use cases for editing:**
- Fine-tune variable names or formatting
- Adjust line numbers if AI got the location slightly wrong
- Add additional changes while you're at it
- Remove parts of the suggestion you don't want
- Combine multiple changes into one patch

**Environment variable:**
```bash
export EDITOR=vim        # Use vim
export EDITOR=nano       # Use nano
export EDITOR=code       # Use VS Code (may need -w flag for blocking)
export EDITOR="code -w"  # VS Code in blocking mode
```

## Error Handling

### AI Call Failures

**Network/API errors:**

- Show clear error message
- Save request details for debugging
- Offer retry or skip options

**Invalid/unparseable response:**

- Log full response to debug file
- Explain what went wrong
- Fall back to manual application

### Patch Application Failures

When AI-generated patch fails to apply:

1. Save patch to `/tmp/gh-review-ai-patch-<ID>.patch`
2. Show detailed error with file path
3. Display git apply error output
4. Offer to continue with next suggestion

## Future Enhancements

### Smart Context Selection

Instead of sending the entire file:

- Parse import statements and include referenced files
- Include interface definitions the code implements
- Add recent commit messages for context
- Limit to relevant sections (function/class scope)

### Multi-Model Consensus

For critical changes:

- Query multiple AI providers
- Compare generated patches
- Show differences to user
- Higher confidence when models agree

### Learning and Feedback

- Track AI success/failure rate per provider
- User feedback: "This AI suggestion was good/bad"
- Automatic provider/model selection based on historical performance
- A/B testing different prompt templates

### Caching

- Cache AI responses for identical context
- Reuse responses across team members
- Reduce API costs and latency

### Language-Specific Optimizations

- Different prompt strategies per language
- Include language-specific linting rules
- Reference language style guides
- Integrate with LSP for semantic understanding

### Confidence-Based Automation

```yaml
ai:
  auto_apply_threshold: 0.95  # Auto-apply if confidence >= 95%
  review_threshold: 0.70      # Show for review if >= 70%
  reject_threshold: 0.50      # Auto-reject if < 50%
```

## Security Considerations

### API Key Management

- Never commit API keys to repositories
- Use environment variables or secure key stores
- Support multiple key management strategies
- Warn if keys are found in config files

### Code Privacy

Users should be aware:

- AI providers receive code content
- Consider data residency requirements
- Some providers (like Ollama) support local/on-premise deployment
- Document what data is sent to each provider

### Prompt Injection

- Sanitize review comments before including in prompts
- Validate AI responses before execution
- Never execute arbitrary code from AI responses
- Only apply validated unified diff patches

## Testing Strategy

### Mock AI Provider

Create a test implementation of `AIProvider` that:

- Returns predictable patches for testing
- Simulates various error conditions
- Validates request context completeness
- No actual API calls or costs

### Integration Tests

- Test each provider with real API (optional, gated)
- Verify prompt template rendering
- Test configuration loading from various sources
- Validate response parsing

### End-to-End Testing

- Real GitHub PRs with outdated suggestions
- Various programming languages
- Different types of code changes (refactoring, moves, renames)
- Success and failure scenarios
