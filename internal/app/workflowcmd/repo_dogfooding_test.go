package workflowcmd

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/guillaume-galp/cge/internal/app/workflow"
	"github.com/guillaume-galp/cge/internal/infra/repo"
)

func TestRepoDogfoodingMetadataReferencesGraphWorkflowPath(t *testing.T) {
	t.Parallel()

	repoRoot := workflowcmdRepoRoot(t)
	checks := map[string][]string{
		".github/copilot-instructions.md": {
			".github/hooks/scripts/repo-delegated-workflow.sh kickoff",
			"graph workflow start",
			"graph workflow finish",
			"--opt-out",
		},
		".github/agents/orchestrator.agent.md": {
			"repo-delegated-workflow.sh kickoff",
			"repo-delegated-workflow.sh handoff",
			".graph/workflow/assets/",
			"CGE_REPO_WORKFLOW_OPTOUT=1",
		},
		".github/agents/developer.agent.md": {
			"repo-delegated-workflow.sh kickoff",
			"repo-delegated-workflow.sh handoff",
			"graph workflow start",
			"graph workflow finish",
		},
		".github/agents/reviewer.agent.md": {
			"repo-delegated-workflow.sh kickoff",
			"repo-delegated-workflow.sh handoff",
		},
		".github/agents/troubleshooter.agent.md": {
			"repo-delegated-workflow.sh kickoff",
			"repo-delegated-workflow.sh handoff",
		},
		".github/prompts/run-autopilot.prompt.md": {
			"repo-delegated-workflow.sh kickoff",
			"repo-delegated-workflow.sh handoff",
			"graph workflow init",
			"CGE_REPO_WORKFLOW_OPTOUT=1",
		},
		".github/hooks/scripts/announce-delegated-workflow.sh": {
			"repo-delegated-workflow.sh kickoff",
			"repo-delegated-workflow.sh handoff",
			".graph/workflow/assets/",
			"CGE_REPO_WORKFLOW_OPTOUT=1",
		},
		".github/hooks/scripts/repo-delegated-workflow.sh": {
			"graph workflow init",
			"graph workflow start",
			"graph workflow finish",
			"--opt-out",
		},
		".github/hooks/scripts/verify-repo-delegated-workflow.sh": {
			"repo-delegated-workflow.sh kickoff",
			"repo-delegated-workflow.sh handoff",
		},
	}

	for relativePath, requiredSnippets := range checks {
		payload, err := os.ReadFile(filepath.Join(repoRoot, relativePath))
		if err != nil {
			t.Fatalf("os.ReadFile(%s): %v", relativePath, err)
		}
		text := string(payload)
		for _, snippet := range requiredSnippets {
			if !strings.Contains(text, snippet) {
				t.Errorf("%s does not contain %q", relativePath, snippet)
			}
		}
	}
}

func TestRepoDelegatedWorkflowScriptHonorsOptOut(t *testing.T) {
	t.Parallel()

	repoRoot := workflowcmdRepoRoot(t)
	repoDir := initGitRepository(t)
	copyWorkflowFile(t, filepath.Join(repoRoot, ".github", "hooks", "scripts", "repo-delegated-workflow.sh"), filepath.Join(repoDir, ".github", "hooks", "scripts", "repo-delegated-workflow.sh"), 0o755)

	payloadPath := filepath.Join(repoDir, "task-outcome.json")
	writeWorkflowFixture(t, payloadPath, `{"schema_version":"v1"}`)

	fakeBinDir := filepath.Join(repoDir, "fake-bin")
	if err := os.MkdirAll(fakeBinDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll fake-bin: %v", err)
	}
	graphLogPath := filepath.Join(repoDir, "graph.log")
	writeWorkflowFixture(t, filepath.Join(fakeBinDir, "graph"), "#!/usr/bin/env bash\nprintf '%s\\n' \"$*\" >> \""+graphLogPath+"\"\nexit 99\n")
	if err := os.Chmod(filepath.Join(fakeBinDir, "graph"), 0o755); err != nil {
		t.Fatalf("os.Chmod fake graph: %v", err)
	}

	baseEnv := append(os.Environ(), "PATH="+fakeBinDir+":"+os.Getenv("PATH"))
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("bash", append([]string{".github/hooks/scripts/repo-delegated-workflow.sh"}, args...)...)
		cmd.Dir = repoDir
		cmd.Env = baseEnv
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("script %v returned error: %v\n%s", args, err, output)
		}
		return string(output)
	}

	kickoffOutput := run("kickoff", "--task", "demo delegated task", "--opt-out")
	if !strings.Contains(kickoffOutput, "opt-out honored") {
		t.Fatalf("kickoff output = %q, want opt-out notice", kickoffOutput)
	}

	handoffOutput := run("handoff", "--file", payloadPath, "--opt-out")
	if !strings.Contains(handoffOutput, "opt-out honored") {
		t.Fatalf("handoff output = %q, want opt-out notice", handoffOutput)
	}

	if payload, err := os.ReadFile(graphLogPath); err == nil && len(bytes.TrimSpace(payload)) > 0 {
		t.Fatalf("graph command was invoked during opt-out: %s", payload)
	}
}

func TestRepoDelegatedWorkflowCommandFlowCompletesKickoffAndHandoff(t *testing.T) {
	t.Parallel()

	repoDir := initGitRepository(t)
	writeWorkflowFixture(t, filepath.Join(repoDir, "README.md"), "# Cognitive Graph Engine\n\nRepo dogfooding verification.\n")
	writeWorkflowFixture(t, filepath.Join(repoDir, "docs", "architecture", "components.md"), "# Components\n")
	writeWorkflowFixture(t, filepath.Join(repoDir, "docs", "plan", "backlog.yaml"), "project: cognitive-graph-engine\n")
	writeWorkflowFixture(t, filepath.Join(repoDir, ".github", "copilot-instructions.md"), "# Copilot instructions\n")

	manager := repo.NewManager(repo.NewGitRepositoryLocator())
	cmd := newCommand(repoDir, workflow.NewService(manager))
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"init"})
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("workflow init returned error: %v", err)
	}

	stdout.Reset()
	cmd.SetArgs([]string{"start", "--task", "verify repo-local delegated workflow kickoff and handoff", "--max-tokens", "900"})
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("workflow start returned error: %v", err)
	}

	startResponse := decodeWorkflowStartSuccessResponse(t, stdout.Bytes())
	if startResponse.Result.Recommendation != workflow.RecommendationProceed {
		t.Fatalf("recommendation = %q, want %q", startResponse.Result.Recommendation, workflow.RecommendationProceed)
	}
	if startResponse.Result.Kickoff.DelegationBrief.Status == "" {
		t.Fatal("expected non-empty delegation brief status")
	}
	if !strings.Contains(startResponse.Result.Kickoff.DelegationBrief.Prompt, "Task: verify repo-local delegated workflow kickoff and handoff") {
		t.Fatalf("delegation brief prompt = %q, want task details", startResponse.Result.Kickoff.DelegationBrief.Prompt)
	}

	outcomePath := filepath.Join(repoDir, "task-outcome.json")
	writeWorkflowFixture(t, outcomePath, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "repo-dogfood-flow",
    "timestamp": "2026-03-24T12:30:00Z"
  },
  "task": "verify repo-local delegated workflow kickoff and handoff",
  "summary": "Completed the repo-local delegated kickoff and handoff flow without inventing ad hoc prompt conventions.",
  "decisions": [
    {
      "summary": "Use the standard workflow start and finish commands for dogfooding",
      "rationale": "Keeps the repo-local workflow explicit and inspectable",
      "status": "accepted"
    }
  ],
  "changed_artifacts": [
    {
      "path": ".github/copilot-instructions.md",
      "summary": "Repo metadata points non-trivial delegated work at the graph-backed path",
      "change_type": "updated",
      "language": "markdown"
    }
  ],
  "follow_up": []
}`)

	stdout.Reset()
	cmd.SetArgs([]string{"finish", "--file", outcomePath})
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("workflow finish returned error: %v", err)
	}

	finishResponse := decodeWorkflowFinishSuccessResponse(t, stdout.Bytes())
	if finishResponse.Result.WriteSummary.Status != workflow.FinishWriteStatusApplied {
		t.Fatalf("write status = %q, want %q", finishResponse.Result.WriteSummary.Status, workflow.FinishWriteStatusApplied)
	}
	if finishResponse.Result.HandoffBrief == nil {
		t.Fatal("expected handoff brief, got nil")
	}
	if !strings.Contains(finishResponse.Result.HandoffBrief.Prompt, "Next-agent brief:") {
		t.Fatalf("handoff prompt = %q, want next-agent brief", finishResponse.Result.HandoffBrief.Prompt)
	}
	if !finishResponse.Result.AfterRevision.Exists {
		t.Fatal("expected after revision to exist")
	}
}

func workflowcmdRepoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func copyWorkflowFile(t *testing.T, sourcePath, destinationPath string, mode os.FileMode) {
	t.Helper()

	payload, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("os.ReadFile source %s: %v", sourcePath, err)
	}
	if err := os.MkdirAll(filepath.Dir(destinationPath), 0o755); err != nil {
		t.Fatalf("os.MkdirAll destination dir: %v", err)
	}
	if err := os.WriteFile(destinationPath, payload, mode); err != nil {
		t.Fatalf("os.WriteFile destination %s: %v", destinationPath, err)
	}
}
