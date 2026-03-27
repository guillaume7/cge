#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  bash .github/hooks/scripts/verify-repo-delegated-workflow.sh [--task "<task>"] [--artifacts-dir DIR] [--opt-out]

Runs an explicit repo-local delegated-workflow kickoff and handoff using:
  .github/hooks/scripts/repo-delegated-workflow.sh kickoff
  .github/hooks/scripts/repo-delegated-workflow.sh handoff
EOF
}

task="verify repo-local delegated workflow kickoff and handoff"
artifacts_dir=""
opt_out=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --task)
      task="${2:-}"
      shift 2
      ;;
    --artifacts-dir)
      artifacts_dir="${2:-}"
      shift 2
      ;;
    --opt-out)
      opt_out=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 64
      ;;
  esac
done

if (( opt_out == 1 )) || [[ "${CGE_REPO_WORKFLOW_OPTOUT:-0}" == "1" ]]; then
  echo "Repo delegated workflow verification skipped because opt-out was requested."
  exit 0
fi

if [[ -z "$artifacts_dir" ]]; then
  artifacts_dir="$(mktemp -d "${TMPDIR:-/tmp}/cge-repo-workflow-verify.XXXXXX")"
else
  mkdir -p "$artifacts_dir"
fi

kickoff_output="${artifacts_dir}/kickoff.json"
finish_output="${artifacts_dir}/finish.json"
outcome_payload="${artifacts_dir}/task-outcome.json"

cat >"${outcome_payload}" <<'EOF'
{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "repo-dogfood-verifier",
    "session_id": "repo-workflow-verification",
    "timestamp": "2026-03-24T12:00:00Z"
  },
  "task": "verify repo-local delegated workflow kickoff and handoff",
  "summary": "Verified that this repo can complete an explicit graph-backed delegated kickoff and handoff without ad hoc prompt conventions.",
  "decisions": [
    {
      "summary": "Use the repo-local delegated workflow helper instead of inventing new prompt wording",
      "rationale": "Keeps kickoff and handoff explicit, inspectable, and consistent with VP3 dogfooding",
      "status": "accepted"
    }
  ],
  "changed_artifacts": [
    {
      "path": ".github/copilot-instructions.md",
      "summary": "Repo metadata steers delegated work through graph-backed kickoff and handoff",
      "change_type": "reviewed",
      "language": "markdown"
    }
  ],
  "follow_up": []
}
EOF

bash .github/hooks/scripts/repo-delegated-workflow.sh kickoff \
  --task "${task}" \
  --output "${kickoff_output}"

bash .github/hooks/scripts/repo-delegated-workflow.sh handoff \
  --file "${outcome_payload}" \
  --output "${finish_output}"

cat <<EOF
Repo delegated workflow verification completed.
- kickoff_output: ${kickoff_output}
- finish_output: ${finish_output}
- outcome_payload: ${outcome_payload}
EOF
