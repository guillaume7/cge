package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/guillaume-galp/cge/internal/app/cmdsupport"
	"github.com/guillaume-galp/cge/internal/infra/repo"
)

const workflowAssetsDirName = "assets"

type workflowAssetSpec struct {
	Path    string
	Kind    string
	Mode    os.FileMode
	Content string
}

func (s *Service) ensureAssets(workspace repo.Workspace, manifest Manifest) (Manifest, WorkSummary, WorkSummary, WorkSummary, error) {
	installed := WorkSummary{Items: []WorkItem{}}
	refreshed := WorkSummary{Items: []WorkItem{}}
	skipped := WorkSummary{Items: []WorkItem{}}
	managedSpecs := managedWorkflowAssetSpecs()
	workflowDir := filepath.Join(workspace.RepoRoot, repo.WorkspaceDirName, repo.WorkflowDirName)
	assetsRoot := filepath.Join(workflowDir, workflowAssetsDirName)
	needsSync := false

	for _, spec := range managedSpecs {
		override, overridden := findOverrideForAsset(manifest.PreservedOverrides, spec.Path)
		if overridden {
			manifest.Assets = upsertAsset(manifest.Assets, Asset{
				Path:         spec.Path,
				Kind:         spec.Kind,
				Status:       "preserved",
				OverridePath: firstNonEmpty(override.Path, spec.Path),
			})
			continue
		}

		action, err := workflowAssetAction(workspace.RepoRoot, spec)
		if err != nil {
			return Manifest{}, WorkSummary{}, WorkSummary{}, WorkSummary{}, err
		}
		if action != "skipped" {
			needsSync = true
		}

		manifest.Assets = upsertAsset(manifest.Assets, Asset{
			Path:   spec.Path,
			Kind:   spec.Kind,
			Status: "installed",
			Digest: checksumText(spec.Content),
		})

		item := WorkItem{
			Kind:   spec.Kind,
			Path:   spec.Path,
			Status: action,
		}
		switch action {
		case "installed":
			item.Reason = "installed managed workflow asset"
			installed.Items = append(installed.Items, item)
		case "refreshed":
			item.Reason = "refreshed managed workflow asset"
			refreshed.Items = append(refreshed.Items, item)
		default:
			item.Reason = "managed workflow asset already matched current content"
			skipped.Items = append(skipped.Items, item)
		}
	}

	if needsSync {
		if err := stageAndCommitWorkflowAssets(assetsRoot, manifest.PreservedOverrides, managedSpecs); err != nil {
			return Manifest{}, WorkSummary{}, WorkSummary{}, WorkSummary{}, err
		}
	}

	manifest.Assets = normalizeAssets(manifest.Assets)
	installed.Count = len(installed.Items)
	refreshed.Count = len(refreshed.Items)
	skipped.Count = len(skipped.Items)

	return manifest, installed, refreshed, skipped, nil
}

func managedWorkflowAssetSpecs() []workflowAssetSpec {
	return []workflowAssetSpec{
		{
			Path: filepath.ToSlash(filepath.Join(
				repo.WorkspaceDirName,
				repo.WorkflowDirName,
				workflowAssetsDirName,
				"prompts",
				"delegated-graph-workflow.prompt.md",
			)),
			Kind: "workflow_prompt",
			Mode: 0o644,
			Content: strings.TrimSpace(`
# Graph-backed delegated workflow prompt snippet

Use this snippet when you need a predictable kickoff for non-trivial delegated work
in this repository.

## Bootstrap

- Run `+"`graph workflow init`"+` from the repo root before relying on
  graph-backed delegation assets.
- Inspect `+"`.graph/workflow/manifest.json`"+` and `+"`.graph/workflow/assets/`"+`
  when you need to verify what was installed, refreshed, or preserved.

## Delegation guidance

- Prefer explicit graph commands over hidden automation.
- Keep repo-specific prompt conventions in repo-owned files, and declare explicit
  overrides in the workflow manifest instead of silently editing managed defaults.
`) + "\n",
		},
		{
			Path: filepath.ToSlash(filepath.Join(
				repo.WorkspaceDirName,
				repo.WorkflowDirName,
				workflowAssetsDirName,
				"instructions",
				"delegated-graph-workflow.instructions.md",
			)),
			Kind: "workflow_instruction",
			Mode: 0o644,
			Content: strings.TrimSpace(`
# Graph-backed delegated workflow instruction snippet

- Bootstrap repo-local workflow state with `+"`graph workflow init`"+` before
  delegated work that depends on graph context.
- Treat `+"`.graph/workflow/assets/`"+` as the managed location for inspectable
  workflow defaults.
- If a repo needs a custom replacement, record the preserved override in
  `+"`.graph/workflow/manifest.json`"+` so refreshes do not clobber it silently.
- Keep workflow steps explicit until richer workflow commands are intentionally
  installed later.
`) + "\n",
		},
		{
			Path: filepath.ToSlash(filepath.Join(
				repo.WorkspaceDirName,
				repo.WorkflowDirName,
				workflowAssetsDirName,
				"skills",
				"delegated-graph-workflow.skill.md",
			)),
			Kind: "workflow_skill",
			Mode: 0o644,
			Content: strings.TrimSpace(`
# Skill guidance: graph-backed delegated workflow

When delegating substantial repo work:

1. Run `+"`graph workflow init`"+` to ensure the local workflow workspace and
   baseline graph knowledge are available.
2. Inspect the managed assets under `+"`.graph/workflow/assets/`"+` before
   extending or replacing them.
3. Prefer additive repo overrides declared in the workflow manifest over
   untracked edits to managed defaults.
4. Keep kickoff and handoff reasoning inspectable in prompts, notes, and graph
   writes.
`) + "\n",
		},
		{
			Path: filepath.ToSlash(filepath.Join(
				repo.WorkspaceDirName,
				repo.WorkflowDirName,
				workflowAssetsDirName,
				"hooks",
				"graph-workflow.sh",
			)),
			Kind: "workflow_hook",
			Mode: 0o755,
			Content: strings.TrimSpace(`
#!/usr/bin/env bash
set -euo pipefail

# Thin helpers for explicit graph-backed workflow bootstrap.
# These helpers intentionally wrap only commands that already exist.

graph_workflow_bootstrap() {
  graph workflow init "$@"
}

graph_workflow_assets_dir() {
  printf '%s\n' ".graph/workflow/assets"
}
`) + "\n",
		},
	}
}

func findOverrideForAsset(overrides []Override, assetPath string) (Override, bool) {
	for _, override := range overrides {
		if override.AssetPath != "" && override.AssetPath == assetPath {
			return override, true
		}
		if override.Path == assetPath {
			return override, true
		}
	}
	return Override{}, false
}

func workflowAssetAction(repoRoot string, spec workflowAssetSpec) (string, error) {
	absolutePath := filepath.Join(repoRoot, filepath.FromSlash(spec.Path))
	info, err := os.Stat(absolutePath)
	switch {
	case err == nil && info.IsDir():
		return "", classifyWorkflowAssetSyncError(spec, "write", fmt.Errorf("%s is a directory", spec.Path))
	case err == nil:
		payload, readErr := os.ReadFile(absolutePath)
		if readErr != nil {
			return "", classifyWorkflowAssetSyncError(spec, "inspect", readErr)
		}
		if string(payload) == spec.Content && info.Mode().Perm() == spec.Mode {
			return "skipped", nil
		}
		return "refreshed", nil
	case os.IsNotExist(err):
		return "installed", nil
	default:
		return "", classifyWorkflowAssetSyncError(spec, "inspect", err)
	}
}

func stageAndCommitWorkflowAssets(assetsRoot string, overrides []Override, specs []workflowAssetSpec) error {
	workflowDir := filepath.Dir(assetsRoot)
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		return classifyWorkflowAssetsDirectorySyncError("prepare", err)
	}

	stageDir, err := os.MkdirTemp(workflowDir, ".assets-stage-")
	if err != nil {
		return classifyWorkflowAssetsDirectorySyncError("prepare", err)
	}
	defer func() {
		_ = os.RemoveAll(stageDir)
	}()

	if err := copyWorkflowAssetsDir(assetsRoot, stageDir); err != nil {
		return classifyWorkflowAssetsDirectorySyncError("prepare", err)
	}

	for _, spec := range specs {
		if _, overridden := findOverrideForAsset(overrides, spec.Path); overridden {
			continue
		}
		operation := "refresh"
		if !workflowAssetPathExists(filepath.Join(assetsRoot, filepath.FromSlash(workflowAssetRelativePath(spec.Path)))) {
			operation = "install"
		}
		if err := writeStagedWorkflowAsset(stageDir, spec, operation); err != nil {
			return err
		}
	}

	if err := swapWorkflowAssetsDir(assetsRoot, stageDir); err != nil {
		return classifyWorkflowAssetsDirectorySyncError("commit", err)
	}
	return nil
}

func copyWorkflowAssetsDir(sourceRoot, destinationRoot string) error {
	info, err := os.Stat(sourceRoot)
	switch {
	case os.IsNotExist(err):
		return nil
	case err != nil:
		return err
	case !info.IsDir():
		return fmt.Errorf("%s is not a directory", filepath.ToSlash(sourceRoot))
	}

	return filepath.WalkDir(sourceRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relativePath, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}
		if relativePath == "." {
			return nil
		}

		targetPath := filepath.Join(destinationRoot, relativePath)
		if entry.IsDir() {
			info, err := entry.Info()
			if err != nil {
				return err
			}
			return os.MkdirAll(targetPath, info.Mode().Perm())
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}
		return copyWorkflowAssetFile(path, targetPath, info.Mode().Perm())
	})
}

func copyWorkflowAssetFile(sourcePath, destinationPath string, mode os.FileMode) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destinationFile, err := os.OpenFile(destinationPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	if _, err := io.Copy(destinationFile, sourceFile); err != nil {
		return err
	}
	return destinationFile.Chmod(mode)
}

func writeStagedWorkflowAsset(stageDir string, spec workflowAssetSpec, operation string) error {
	stagePath := filepath.Join(stageDir, filepath.FromSlash(workflowAssetRelativePath(spec.Path)))
	if err := os.MkdirAll(filepath.Dir(stagePath), 0o755); err != nil {
		return classifyWorkflowAssetSyncError(spec, "prepare", err)
	}
	if info, err := os.Stat(stagePath); err == nil && info.IsDir() {
		return classifyWorkflowAssetSyncError(spec, operation, fmt.Errorf("%s is a directory", spec.Path))
	} else if err != nil && !os.IsNotExist(err) {
		return classifyWorkflowAssetSyncError(spec, operation, err)
	}
	if err := os.WriteFile(stagePath, []byte(spec.Content), spec.Mode); err != nil {
		return classifyWorkflowAssetSyncError(spec, operation, err)
	}
	if err := os.Chmod(stagePath, spec.Mode); err != nil {
		return classifyWorkflowAssetSyncError(spec, operation, err)
	}
	return nil
}

func workflowAssetRelativePath(assetPath string) string {
	prefix := filepath.ToSlash(filepath.Join(repo.WorkspaceDirName, repo.WorkflowDirName, workflowAssetsDirName)) + "/"
	return strings.TrimPrefix(assetPath, prefix)
}

func workflowAssetPathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func swapWorkflowAssetsDir(assetsRoot, stageDir string) error {
	backupRoot := ""
	if info, err := os.Stat(assetsRoot); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("%s is not a directory", filepath.ToSlash(assetsRoot))
		}
		backupRoot = filepath.Join(filepath.Dir(assetsRoot), fmt.Sprintf(".assets-backup-%s", filepath.Base(stageDir)))
		if err := os.Rename(assetsRoot, backupRoot); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.Rename(stageDir, assetsRoot); err != nil {
		if backupRoot != "" {
			_ = os.Rename(backupRoot, assetsRoot)
		}
		return err
	}

	if backupRoot != "" {
		if err := os.RemoveAll(backupRoot); err != nil {
			return err
		}
	}
	return nil
}

func classifyWorkflowAssetSyncError(spec workflowAssetSpec, operation string, err error) error {
	return cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
		Category: "operational_error",
		Type:     "workflow_error",
		Code:     "workflow_asset_sync_failed",
		Message:  "workflow bootstrap could not install or refresh a required workflow asset",
		Details: map[string]any{
			"path":      spec.Path,
			"kind":      spec.Kind,
			"operation": operation,
			"reason":    err.Error(),
		},
	}, err)
}

func classifyWorkflowAssetsDirectorySyncError(operation string, err error) error {
	return cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
		Category: "operational_error",
		Type:     "workflow_error",
		Code:     "workflow_asset_sync_failed",
		Message:  "workflow bootstrap could not install or refresh a required workflow asset",
		Details: map[string]any{
			"path":      filepath.ToSlash(filepath.Join(repo.WorkspaceDirName, repo.WorkflowDirName, workflowAssetsDirName)),
			"kind":      "workflow_assets_directory",
			"operation": operation,
			"reason":    err.Error(),
		},
	}, err)
}

func checksumText(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func managedWorkflowAssetPaths() []string {
	paths := make([]string, 0, len(managedWorkflowAssetSpecs()))
	for _, spec := range managedWorkflowAssetSpecs() {
		paths = append(paths, spec.Path)
	}
	slices.Sort(paths)
	return paths
}
