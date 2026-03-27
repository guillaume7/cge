#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  bash .github/hooks/scripts/repo-delegated-workflow.sh kickoff --task "<task>" [--max-tokens N] [--output FILE] [--no-init] [--opt-out]
  bash .github/hooks/scripts/repo-delegated-workflow.sh handoff --file task-outcome.json [--output FILE] [--opt-out]
  bash .github/hooks/scripts/repo-delegated-workflow.sh assets

Thin, inspectable repo-local helper around:
  graph workflow init
  graph workflow start
  graph workflow finish

Opt out explicitly with --opt-out or CGE_REPO_WORKFLOW_OPTOUT=1.
EOF
}

log_command() {
  {
    printf '+'
    for part in "$@"; do
      printf ' %q' "$part"
    done
    printf '\n'
  } >&2
}

run_logged() {
  log_command "$@"
  "$@"
}

opted_out() {
  [[ "${CGE_REPO_WORKFLOW_OPTOUT:-0}" == "1" ]]
}

require_graph() {
  if ! command -v graph >/dev/null 2>&1; then
    echo "graph command not found in PATH" >&2
    exit 127
  fi
}

subcommand="${1:-}"
if [[ -z "$subcommand" ]]; then
  usage
  exit 64
fi
shift

case "$subcommand" in
  kickoff)
    task=""
    output=""
    max_tokens="1200"
    no_init=0
    local_opt_out=0

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --task)
          task="${2:-}"
          shift 2
          ;;
        --max-tokens)
          max_tokens="${2:-}"
          shift 2
          ;;
        --output)
          output="${2:-}"
          shift 2
          ;;
        --no-init)
          no_init=1
          shift
          ;;
        --opt-out)
          local_opt_out=1
          shift
          ;;
        -h|--help)
          usage
          exit 0
          ;;
        *)
          echo "unknown kickoff argument: $1" >&2
          usage >&2
          exit 64
          ;;
      esac
    done

    if [[ -z "$task" ]]; then
      echo "kickoff requires --task" >&2
      usage >&2
      exit 64
    fi

    if (( local_opt_out == 1 )); then
      export CGE_REPO_WORKFLOW_OPTOUT=1
    fi

    if opted_out; then
      printf 'Repo delegated workflow opt-out honored; skipped graph workflow start for task: %s\n' "$task"
      exit 0
    fi

    require_graph
    if (( no_init == 0 )) && [[ ! -f ".graph/workflow/manifest.json" ]]; then
      run_logged graph workflow init
    fi

    command=(graph workflow start --task "$task" --max-tokens "$max_tokens")
    if [[ -n "$output" ]]; then
      command+=(--output "$output")
    fi
    run_logged "${command[@]}"
    ;;
  handoff)
    file=""
    output=""
    local_opt_out=0

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --file)
          file="${2:-}"
          shift 2
          ;;
        --output)
          output="${2:-}"
          shift 2
          ;;
        --opt-out)
          local_opt_out=1
          shift
          ;;
        -h|--help)
          usage
          exit 0
          ;;
        *)
          echo "unknown handoff argument: $1" >&2
          usage >&2
          exit 64
          ;;
      esac
    done

    if [[ -z "$file" ]]; then
      echo "handoff requires --file" >&2
      usage >&2
      exit 64
    fi

    if (( local_opt_out == 1 )); then
      export CGE_REPO_WORKFLOW_OPTOUT=1
    fi

    if opted_out; then
      printf 'Repo delegated workflow opt-out honored; skipped graph workflow finish for payload: %s\n' "$file"
      exit 0
    fi

    require_graph
    command=(graph workflow finish --file "$file")
    if [[ -n "$output" ]]; then
      command+=(--output "$output")
    fi
    run_logged "${command[@]}"
    ;;
  assets)
    printf '%s\n' ".graph/workflow/assets"
    ;;
  -h|--help)
    usage
    ;;
  *)
    echo "unknown subcommand: $subcommand" >&2
    usage >&2
    exit 64
    ;;
esac
