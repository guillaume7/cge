package repo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	WorkspaceDirName       = ".graph"
	ConfigFileName         = "config.json"
	WorkspaceSchemaVersion = "v1"
)

var (
	ErrRepositoryRootNotFound  = errors.New("repo-scoped initialization could not determine a repository root")
	ErrWorkspaceNotInitialized = errors.New("repo graph has not been initialized; run \"graph init\" first")
)

type RepositoryLocator interface {
	FindRoot(ctx context.Context, startDir string) (string, error)
	FindGitCommonDir(ctx context.Context, repoRoot string) (string, error)
}

type GitRepositoryLocator struct{}

func NewGitRepositoryLocator() GitRepositoryLocator {
	return GitRepositoryLocator{}
}

func (GitRepositoryLocator) FindRoot(ctx context.Context, startDir string) (string, error) {
	root, err := runGit(ctx, startDir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrRepositoryRootNotFound, strings.TrimSpace(err.Error()))
	}

	return filepath.Clean(root), nil
}

func (GitRepositoryLocator) FindGitCommonDir(ctx context.Context, repoRoot string) (string, error) {
	gitDir, err := runGit(ctx, repoRoot, "rev-parse", "--git-common-dir")
	if err != nil {
		return "", fmt.Errorf("resolve git directory: %w", err)
	}

	if filepath.IsAbs(gitDir) {
		return filepath.Clean(gitDir), nil
	}

	return filepath.Clean(filepath.Join(repoRoot, gitDir)), nil
}

type Manager struct {
	locator RepositoryLocator
}

func NewManager(locator RepositoryLocator) *Manager {
	return &Manager{locator: locator}
}

type InitResult struct {
	WorkspacePath string
	AlreadyExists bool
	Config        WorkspaceConfig
}

type Workspace struct {
	RepoRoot      string
	WorkspacePath string
	Config        WorkspaceConfig
}

type WorkspaceConfig struct {
	SchemaVersion string             `json:"schema_version"`
	Repository    RepositoryMetadata `json:"repository"`
}

type RepositoryMetadata struct {
	ID           string `json:"id"`
	RootPath     string `json:"root_path"`
	GitCommonDir string `json:"git_common_dir"`
}

type resolvedRepository struct {
	RootPath     string
	GitCommonDir string
}

func (m *Manager) InitWorkspace(ctx context.Context, startDir string) (InitResult, error) {
	if m == nil || m.locator == nil {
		return InitResult{}, errors.New("repository manager is not configured")
	}

	resolvedRepo, err := m.resolveRepository(ctx, startDir)
	if err != nil {
		return InitResult{}, err
	}

	config := workspaceConfig(resolvedRepo)
	workspacePath := filepath.Join(resolvedRepo.RootPath, WorkspaceDirName)
	for _, dir := range []string{
		workspacePath,
		filepath.Join(workspacePath, "kuzu"),
		filepath.Join(workspacePath, "index"),
		filepath.Join(workspacePath, "tmp"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return InitResult{}, fmt.Errorf("create workspace directory %s: %w", dir, err)
		}
	}

	configPath := filepath.Join(workspacePath, ConfigFileName)
	alreadyExists, err := ensureConfig(configPath, config)
	if err != nil {
		return InitResult{}, err
	}

	return InitResult{
		WorkspacePath: workspacePath,
		AlreadyExists: alreadyExists,
		Config:        config,
	}, nil
}

func (m *Manager) OpenWorkspace(ctx context.Context, startDir string) (Workspace, error) {
	if m == nil || m.locator == nil {
		return Workspace{}, errors.New("repository manager is not configured")
	}

	resolvedRepo, err := m.resolveRepository(ctx, startDir)
	if err != nil {
		return Workspace{}, err
	}

	workspacePath := filepath.Join(resolvedRepo.RootPath, WorkspaceDirName)
	configPath := filepath.Join(workspacePath, ConfigFileName)
	config, err := loadConfig(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Workspace{}, fmt.Errorf("%w: %s", ErrWorkspaceNotInitialized, workspacePath)
		}
		return Workspace{}, err
	}

	expected := workspaceConfig(resolvedRepo)
	if config != expected {
		return Workspace{}, fmt.Errorf("workspace config at %s does not match repository identity or schema version", configPath)
	}

	return Workspace{
		RepoRoot:      resolvedRepo.RootPath,
		WorkspacePath: workspacePath,
		Config:        config,
	}, nil
}

func (m *Manager) resolveRepository(ctx context.Context, startDir string) (resolvedRepository, error) {
	repoRoot, err := m.locator.FindRoot(ctx, startDir)
	if err != nil {
		return resolvedRepository{}, err
	}

	resolvedRoot, err := filepath.EvalSymlinks(repoRoot)
	if err != nil {
		return resolvedRepository{}, fmt.Errorf("resolve repository root: %w", err)
	}

	gitCommonDir, err := m.locator.FindGitCommonDir(ctx, resolvedRoot)
	if err != nil {
		return resolvedRepository{}, err
	}

	resolvedGitCommonDir, err := filepath.EvalSymlinks(gitCommonDir)
	if err != nil {
		return resolvedRepository{}, fmt.Errorf("resolve git directory: %w", err)
	}

	return resolvedRepository{
		RootPath:     resolvedRoot,
		GitCommonDir: resolvedGitCommonDir,
	}, nil
}

func workspaceConfig(resolvedRepo resolvedRepository) WorkspaceConfig {
	return WorkspaceConfig{
		SchemaVersion: WorkspaceSchemaVersion,
		Repository: RepositoryMetadata{
			ID:           repositoryID(resolvedRepo.RootPath, resolvedRepo.GitCommonDir),
			RootPath:     resolvedRepo.RootPath,
			GitCommonDir: resolvedRepo.GitCommonDir,
		},
	}
}

func ensureConfig(path string, expected WorkspaceConfig) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		existing, err := loadConfig(path)
		if err != nil {
			return false, err
		}

		if existing != expected {
			return false, fmt.Errorf("existing workspace config at %s does not match repository identity or schema version", path)
		}

		return true, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("inspect workspace config: %w", err)
	}

	payload, err := json.MarshalIndent(expected, "", "  ")
	if err != nil {
		return false, fmt.Errorf("encode workspace config: %w", err)
	}
	payload = append(payload, '\n')

	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return false, fmt.Errorf("write workspace config: %w", err)
	}

	return false, nil
}

func loadConfig(path string) (WorkspaceConfig, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return WorkspaceConfig{}, err
	}

	var config WorkspaceConfig
	if err := json.Unmarshal(payload, &config); err != nil {
		return WorkspaceConfig{}, fmt.Errorf("parse workspace config: %w", err)
	}

	return config, nil
}

func repositoryID(repoRoot, gitCommonDir string) string {
	sum := sha256.Sum256([]byte(repoRoot + "\n" + gitCommonDir))
	return hex.EncodeToString(sum[:])
}

func runGit(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}

	return strings.TrimSpace(string(output)), nil
}
