# Debugging Guide for gh-review

This guide explains how to debug issues when applying review suggestions.

## Debug Flags and Commands

### 1. Enable Debug Mode on Apply

```bash
gh review apply --debug [PR_NUMBER]
```

This shows detailed diagnostic information including:
- Comment position info (Line, OriginalLine, StartLine, EndLine)
- DiffSide (LEFT/RIGHT) and outdated status
- Complete diff hunk from GitHub
- Position mapping strategy used
- Content verification details
- Line-by-line comparison of expected vs actual content

### 2. Get Raw JSON for a Comment

```bash
gh review debug <PR_NUMBER> <COMMENT_ID>
```

This dumps the raw JSON from GitHub's API for debugging.

## Diagnostic Files Generated on Failure

When a suggestion fails to apply, diagnostic files are automatically saved to `/tmp/`:

### Content Mismatch: `/tmp/gh-review-mismatch-<COMMENT_ID>.diff`

Generated when the current file content doesn't match what was expected from the review.

**Contents:**
- Comment metadata (ID, URL, file path, line numbers)
- Original diff hunk from GitHub
- Expected lines (what GitHub review expected)
- Actual lines (current file content)
- Side-by-side diff showing the mismatch
- The suggested change from the review

**Example:**
```diff
# Diagnostic diff for comment ID 12345
# File: docs/guide/example.md
# Comment URL: https://github.com/owner/repo/pull/123#discussion_r12345
# Mismatch at line: 129
# Comment info: Line=135, OriginalLine=129, DiffSide=RIGHT, IsOutdated=false
#
# Original diff hunk from GitHub:
# @@ -126,7 +126,27 @@ ...
# ...
#
# EXPECTED (from GitHub review):
#   [129] ## Disabling all comments for PipelineRuns on GitLab MR
#   [130]
# ! [131] For GitHub (Webhook) and GitLab integrations...
#
# ACTUAL (current file content):
#   [129] ## Controlling Pull/Merge Request comment volume
#   [130]
# ! [131] For GitHub and GitLab integrations, you can control...
#
# Context (showing surrounding lines):
--- a/docs/guide/example.md (expected based on review)
+++ b/docs/guide/example.md (actual current content)
@@ -129,3 +129,3 @@
 ## Previous section

-## Disabling all comments for PipelineRuns on GitLab MR
+## Controlling Pull/Merge Request comment volume
```

### Git Apply Failure: `/tmp/gh-review-patch-<COMMENT_ID>.patch`

Generated when `git apply` fails (less common, usually means the patch format is invalid).

**Contents:**
- Error information
- git apply output
- Generated patch that was attempted

## Common Scenarios

### Scenario 1: Content Has Changed Since Review

**Error:**
```
❌ Failed to apply: content mismatch at line 129 - the code may have changed since the review
Diagnostic diff saved to: /tmp/gh-review-mismatch-12345.diff
```

**What it means:**
- Someone made commits after the review was created
- The code at that location has changed
- The suggestion may be outdated or need rebasing

**How to investigate:**
1. Open the diagnostic diff: `cat /tmp/gh-review-mismatch-12345.diff`
2. Look at the "EXPECTED" vs "ACTUAL" sections
3. Check if the review needs to be updated or dismissed

### Scenario 2: Position Mapping Failed

**Debug output:**
```
[DEBUG] Strategy 1 (position mapping): Found first added line at new position 135 (0-based: 134)
[DEBUG] Target line for replacement: 134 (0-based), which is line 135 (1-based)
[DEBUG] MISMATCH at line 135: got "something else", expected "expected content"
```

**What it means:**
- The diff hunk parsing worked
- But the content at the calculated position doesn't match
- Usually caused by intermediate commits

### Scenario 3: Multi-line Suggestion

**Debug output shows:**
```
[DEBUG] Extracted 10 added lines from diff hunk:
  [0] "For GitHub (Webhook) and GitLab integrations, you can control the types"
  [1] "of Pull/Merge request comments which Pipelines as Code emits using"
  ...
```

For multi-line suggestions, each line must match exactly. Check the diagnostic diff to see which specific line is different.

## Tips for Fixing Issues

### Option 1: Rebase Your Branch

If the review is on an outdated commit:
```bash
git fetch origin
git rebase origin/main
gh review apply [PR_NUMBER]
```

### Option 2: Manually Apply from Diagnostic Diff

1. Open the diagnostic diff
2. Look at the "Suggested change from review" section
3. Manually apply it to the current code
4. Mark the review as resolved

### Option 3: Ask for Updated Review

If the code has changed significantly, ask the reviewer to re-review the updated code.

## Getting More Information

### See All Position Data
```bash
gh review apply --debug [PR_NUMBER] 2>&1 | grep "Comment position"
```

### Extract Just the Diff Hunks
```bash
gh review apply --debug [PR_NUMBER] 2>&1 | grep -A 20 "DiffHunk:"
```

### Find All Mismatches
```bash
ls -lh /tmp/gh-review-mismatch-*.diff
```
