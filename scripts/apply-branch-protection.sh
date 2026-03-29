#!/usr/bin/env bash
# apply-branch-protection.sh
#
# Applies branch protection rules and required CI status checks to the
# main branch of the Maintify repository.
#
# Prerequisites:
#   - GitHub CLI installed: https://cli.github.com/
#   - Authenticated: gh auth login  (needs admin:repo scope)
#   - GITHUB_OWNER set to your GitHub username/org
#
# Usage:
#   GITHUB_OWNER=your-username bash scripts/apply-branch-protection.sh

set -euo pipefail

OWNER="${GITHUB_OWNER:?Error: GITHUB_OWNER env var must be set}"
REPO="Maintify"

if ! command -v gh >/dev/null 2>&1; then
  echo "ERROR: GitHub CLI (gh) is required. Install: https://cli.github.com/" >&2
  exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "ERROR: jq is required but not installed" >&2
  exit 1
fi

required_checks=(
  "ci-passed"
  "dependency-review"
  "secret-scan"
  "filesystem-vuln-scan"
  "pr-security-checklist"
)

echo ""
echo "══════════════════════════════════════════════════════"
echo " Protecting: ${OWNER}/${REPO}  →  branch: main"
echo "══════════════════════════════════════════════════════"

checks_json=$(printf '%s\n' "${required_checks[@]}" | jq -R '{"context":.}' | jq -s '.')

gh api \
  --method PUT \
  -H "Accept: application/vnd.github+json" \
  "/repos/${OWNER}/${REPO}/branches/main/protection" \
  --input - <<EOF
{
  "required_status_checks": {
    "strict": true,
    "checks": ${checks_json}
  },
  "enforce_admins": true,
  "required_pull_request_reviews": {
    "dismiss_stale_reviews": true,
    "require_code_owner_reviews": true,
    "required_approving_review_count": 1
  },
  "restrictions": null,
  "allow_force_pushes": false,
  "allow_deletions": false,
  "block_creations": false,
  "required_conversation_resolution": true
}
EOF

echo ""
echo "════════════════════════════════════════════════════════"
echo " Branch protection applied to ${OWNER}/${REPO}."
echo " Run once to verify:"
echo "   gh api /repos/${OWNER}/${REPO}/branches/main/protection | jq '.required_status_checks'"
echo "════════════════════════════════════════════════════════"
