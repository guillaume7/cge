#!/usr/bin/env bash
set -euo pipefail

if [[ "${CGE_REPO_WORKFLOW_REMINDER:-1}" == "0" ]] || [[ "${CGE_REPO_WORKFLOW_OPTOUT:-0}" == "1" ]]; then
  exit 0
fi

cat <<'EOF'
=== REPO DELEGATED WORKFLOW ===
For most non-trivial delegated subtasks in this repo, prefer the explicit
graph-backed path:

  bash .github/hooks/scripts/repo-delegated-workflow.sh kickoff --task "<task>"
  ... do the delegated work ...
  bash .github/hooks/scripts/repo-delegated-workflow.sh handoff --file task-outcome.json

Direct commands remain valid and inspectable:

  graph workflow init
  graph workflow start --task "<task>"
  graph workflow finish --file task-outcome.json

Inspect installed defaults under .graph/workflow/assets/.
Opt out explicitly with --opt-out or CGE_REPO_WORKFLOW_OPTOUT=1.
EOF
