package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/guillaume-galp/cge/internal/app/cmdsupport"
	graphpayload "github.com/guillaume-galp/cge/internal/domain/payload"
	"github.com/guillaume-galp/cge/internal/infra/kuzu"
	"github.com/guillaume-galp/cge/internal/infra/repo"
)

const (
	workflowSeedAgentID          = "workflow-init"
	workflowSeedSource           = "baseline_repo_graph_knowledge"
	workflowSeedRevisionReason   = "Seed baseline repo graph knowledge during workflow init"
	defaultRepoSeedSummaryPrefix = "Repository-scoped graph workspace for "
)

type SeedWriter interface {
	Write(ctx context.Context, workspace repo.Workspace, envelope graphpayload.Envelope) (kuzu.WriteSummary, error)
}

type baselineSeedSpec struct {
	Path     string
	WorkKind string
	Discover func(repoRoot, repoID string) (discoveredSeedSource, bool, error)
}

type discoveredSeedSource struct {
	WorkKind   string
	Path       string
	NodeID     string
	NodeKind   string
	Title      string
	Summary    string
	Content    string
	Tags       []string
	Properties map[string]any
}

type seedNode struct {
	ID         string         `json:"id"`
	Kind       string         `json:"kind"`
	Title      string         `json:"title,omitempty"`
	Summary    string         `json:"summary,omitempty"`
	Content    string         `json:"content,omitempty"`
	RepoPath   string         `json:"repo_path,omitempty"`
	Language   string         `json:"language,omitempty"`
	Tags       []string       `json:"tags,omitempty"`
	Properties map[string]any `json:"properties,omitempty"`
}

type seedEdge struct {
	From       string         `json:"from"`
	Kind       string         `json:"kind"`
	To         string         `json:"to"`
	Properties map[string]any `json:"properties,omitempty"`
}

func (s *Service) seedBaseline(ctx context.Context, workspace repo.Workspace) (WorkSummary, error) {
	if s.writer == nil {
		s.writer = kuzu.NewStore()
	}
	if s.now == nil {
		s.now = func() time.Time { return time.Now().UTC() }
	}

	result := WorkSummary{Items: []WorkItem{}}
	discovered := make([]discoveredSeedSource, 0, len(baselineSeedSpecs()))

	for _, spec := range baselineSeedSpecs() {
		source, found, err := spec.Discover(workspace.RepoRoot, workspace.Config.Repository.ID)
		if err != nil {
			return WorkSummary{}, classifySeedDiscoveryError(spec.Path, err)
		}
		if !found {
			result.Items = append(result.Items, WorkItem{
				Kind:   spec.WorkKind,
				Path:   filepath.ToSlash(spec.Path),
				Source: workflowSeedSource,
				Status: "skipped",
				Reason: "optional seed source not found",
			})
			continue
		}
		discovered = append(discovered, source)
	}

	if len(discovered) == 0 {
		result.Count = len(result.Items)
		return result, nil
	}

	currentGraph, err := kuzu.NewStore().ReadGraph(ctx, workspace)
	if err != nil {
		return WorkSummary{}, classifySeedPersistenceError(err, discovered)
	}

	desiredGraph := s.buildBaselineSeedGraph(workspace, discovered)
	if baselineSeedUpToDate(currentGraph, desiredGraph) {
		for _, source := range discovered {
			result.Items = append(result.Items, WorkItem{
				Kind:   source.WorkKind,
				Path:   source.Path,
				Source: workflowSeedSource,
				Status: "skipped",
				Reason: "baseline repository knowledge already matches discoverable sources",
			})
		}
		result.Count = len(result.Items)
		return result, nil
	}

	envelope, err := s.buildBaselineSeedEnvelope(workspace, discovered)
	if err != nil {
		return WorkSummary{}, classifySeedDiscoveryError("baseline seed envelope", err)
	}

	if _, err := s.writer.Write(ctx, workspace, envelope); err != nil {
		return WorkSummary{}, classifySeedPersistenceError(err, discovered)
	}

	for _, source := range discovered {
		result.Items = append(result.Items, WorkItem{
			Kind:   source.WorkKind,
			Path:   source.Path,
			Source: workflowSeedSource,
			Status: "seeded",
			Reason: "seeded baseline repository knowledge into the graph",
		})
	}

	result.Count = len(result.Items)
	return result, nil
}

func (s *Service) buildBaselineSeedEnvelope(workspace repo.Workspace, sources []discoveredSeedSource) (graphpayload.Envelope, error) {
	nodes, edges := s.baselineSeedObjects(workspace, sources)
	timestamp := s.now().Format(time.RFC3339)
	sessionID := fmt.Sprintf("workflow-init-%d", s.now().Unix())
	rawNodes, err := marshalSeedObjects(nodes)
	if err != nil {
		return graphpayload.Envelope{}, err
	}
	rawEdges, err := marshalSeedObjects(edges)
	if err != nil {
		return graphpayload.Envelope{}, err
	}

	return graphpayload.Envelope{
		SchemaVersion: graphpayload.SchemaVersionV1,
		Metadata: graphpayload.Metadata{
			AgentID:   workflowSeedAgentID,
			SessionID: sessionID,
			Timestamp: timestamp,
			Revision: graphpayload.RevisionMetadata{
				Reason: workflowSeedRevisionReason,
				Properties: map[string]any{
					"workflow": map[string]any{
						"command":          "workflow.init",
						"seed_source":      workflowSeedSource,
						"seeded_artifacts": seedSourcePaths(sources),
					},
				},
			},
		},
		Nodes: rawNodes,
		Edges: rawEdges,
	}, nil
}

func (s *Service) buildBaselineSeedGraph(workspace repo.Workspace, sources []discoveredSeedSource) kuzu.Graph {
	nodes, edges := s.baselineSeedObjects(workspace, sources)
	graph := kuzu.Graph{
		Nodes: make([]kuzu.EntityRecord, 0, len(nodes)),
		Edges: make([]kuzu.RelationRecord, 0, len(edges)),
	}
	for _, node := range nodes {
		graph.Nodes = append(graph.Nodes, kuzu.EntityRecord{
			ID:       node.ID,
			Kind:     node.Kind,
			Title:    node.Title,
			Summary:  node.Summary,
			Content:  node.Content,
			RepoPath: node.RepoPath,
			Tags:     append([]string(nil), node.Tags...),
			Props:    cloneAnyMap(node.Properties),
		})
	}
	for _, edge := range edges {
		graph.Edges = append(graph.Edges, kuzu.RelationRecord{
			From:  edge.From,
			To:    edge.To,
			Kind:  edge.Kind,
			Props: cloneAnyMap(edge.Properties),
		})
	}
	return graph
}

func (s *Service) baselineSeedObjects(workspace repo.Workspace, sources []discoveredSeedSource) ([]seedNode, []seedEdge) {
	repoID := workspace.Config.Repository.ID
	repoNodeID := "repo:" + repoID

	repoSummary := repositorySeedSummary(workspace.RepoRoot, sources)
	repoNode := seedNode{
		ID:       repoNodeID,
		Kind:     "Repository",
		Title:    filepath.Base(workspace.RepoRoot),
		Summary:  repoSummary,
		RepoPath: ".",
		Tags:     []string{"repository", "workflow"},
		Properties: map[string]any{
			"repository_id":  repoID,
			"git_common_dir": filepath.ToSlash(workspace.Config.Repository.GitCommonDir),
			"root_path":      filepath.ToSlash(workspace.Config.Repository.RootPath),
		},
	}

	nodes := []seedNode{repoNode}
	edges := make([]seedEdge, 0, len(sources))
	for _, source := range sources {
		nodes = append(nodes, seedNode{
			ID:         source.NodeID,
			Kind:       source.NodeKind,
			Title:      source.Title,
			Summary:    source.Summary,
			Content:    source.Content,
			RepoPath:   source.Path,
			Tags:       append([]string(nil), source.Tags...),
			Properties: cloneAnyMap(source.Properties),
		})
		edges = append(edges, seedEdge{
			From: source.NodeID,
			Kind: "PART_OF",
			To:   repoNodeID,
			Properties: map[string]any{
				"source_path": source.Path,
			},
		})
	}
	return nodes, edges
}

func baselineSeedSpecs() []baselineSeedSpec {
	return []baselineSeedSpec{
		{
			Path:     "README.md",
			WorkKind: "document",
			Discover: discoverReadmeSeedSource,
		},
		{
			Path:     "docs/architecture",
			WorkKind: "documentation_directory",
			Discover: discoverArchitectureSeedSource,
		},
		{
			Path:     "docs/plan/backlog.yaml",
			WorkKind: "backlog",
			Discover: discoverBacklogSeedSource,
		},
	}
}

func discoverReadmeSeedSource(repoRoot, repoID string) (discoveredSeedSource, bool, error) {
	path := filepath.Join(repoRoot, "README.md")
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return discoveredSeedSource{}, false, nil
		}
		return discoveredSeedSource{}, false, fmt.Errorf("inspect README.md: %w", err)
	}
	if info.IsDir() {
		return discoveredSeedSource{}, false, fmt.Errorf("README.md must be a file")
	}

	payload, err := os.ReadFile(path)
	if err != nil {
		return discoveredSeedSource{}, false, fmt.Errorf("read README.md: %w", err)
	}

	title, summary, excerpt := summarizeMarkdownDocument(string(payload), "README")
	return discoveredSeedSource{
		WorkKind: "document",
		Path:     "README.md",
		NodeID:   "artifact:" + repoID + ":readme",
		NodeKind: "Document",
		Title:    firstNonEmpty(title, "README"),
		Summary:  firstNonEmpty(summary, "Repository overview and setup guidance."),
		Content:  excerpt,
		Tags:     []string{"readme", "repository", "workflow"},
		Properties: map[string]any{
			"source_type": "markdown",
		},
	}, true, nil
}

func discoverArchitectureSeedSource(repoRoot, repoID string) (discoveredSeedSource, bool, error) {
	path := filepath.Join(repoRoot, "docs", "architecture")
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return discoveredSeedSource{}, false, nil
		}
		return discoveredSeedSource{}, false, fmt.Errorf("inspect docs/architecture: %w", err)
	}
	if !info.IsDir() {
		return discoveredSeedSource{}, false, fmt.Errorf("docs/architecture must be a directory")
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return discoveredSeedSource{}, false, fmt.Errorf("read docs/architecture: %w", err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		files = append(files, entry.Name())
	}
	sort.Strings(files)

	summary := "Architecture reference directory."
	if len(files) > 0 {
		summary = fmt.Sprintf(
			"Architecture reference directory with %d document(s): %s.",
			len(files),
			strings.Join(files, ", "),
		)
	}

	return discoveredSeedSource{
		WorkKind: "documentation_directory",
		Path:     "docs/architecture",
		NodeID:   "artifact:" + repoID + ":architecture",
		NodeKind: "Document",
		Title:    "Architecture documentation",
		Summary:  summary,
		Tags:     []string{"architecture", "docs", "workflow"},
		Properties: map[string]any{
			"source_type": "directory",
			"files":       files,
		},
	}, true, nil
}

func discoverBacklogSeedSource(repoRoot, repoID string) (discoveredSeedSource, bool, error) {
	path := filepath.Join(repoRoot, "docs", "plan", "backlog.yaml")
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return discoveredSeedSource{}, false, nil
		}
		return discoveredSeedSource{}, false, fmt.Errorf("inspect docs/plan/backlog.yaml: %w", err)
	}
	if info.IsDir() {
		return discoveredSeedSource{}, false, fmt.Errorf("docs/plan/backlog.yaml must be a file")
	}

	payload, err := os.ReadFile(path)
	if err != nil {
		return discoveredSeedSource{}, false, fmt.Errorf("read docs/plan/backlog.yaml: %w", err)
	}

	project := yamlScalarValue(string(payload), "project")
	summary := "Project backlog with themes, epics, and stories."
	if project != "" {
		summary = fmt.Sprintf("Project backlog for %s with themes, epics, and stories.", project)
	}

	properties := map[string]any{
		"source_type": "yaml",
	}
	if project != "" {
		properties["project"] = project
	}

	return discoveredSeedSource{
		WorkKind:   "backlog",
		Path:       "docs/plan/backlog.yaml",
		NodeID:     "artifact:" + repoID + ":backlog",
		NodeKind:   "Document",
		Title:      "Project backlog",
		Summary:    summary,
		Content:    excerptText(string(payload), 320),
		Tags:       []string{"backlog", "planning", "workflow"},
		Properties: properties,
	}, true, nil
}

func classifySeedDiscoveryError(path string, err error) error {
	return cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
		Category: "operational_error",
		Type:     "workflow_error",
		Code:     "workflow_seed_discovery_failed",
		Message:  "workflow bootstrap could not inspect baseline seed sources",
		Details: map[string]any{
			"path":   filepath.ToSlash(path),
			"reason": err.Error(),
		},
	}, err)
}

func classifySeedPersistenceError(err error, sources []discoveredSeedSource) error {
	details := map[string]any{
		"reason":  err.Error(),
		"sources": seedSourcePaths(sources),
	}

	var persistenceErr *kuzu.PersistenceError
	if errors.As(err, &persistenceErr) {
		details["cause_code"] = persistenceErr.Code
		if len(persistenceErr.Details) > 0 {
			details["cause_details"] = persistenceErr.Details
		}
	}

	return cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
		Category: "operational_error",
		Type:     "persistence_error",
		Code:     "workflow_seed_persistence_failed",
		Message:  "workflow bootstrap could not persist baseline graph knowledge",
		Details:  details,
	}, err)
}

func repositorySeedSummary(repoRoot string, sources []discoveredSeedSource) string {
	for _, source := range sources {
		if source.Path == "README.md" && strings.TrimSpace(source.Summary) != "" {
			return source.Summary
		}
	}
	return defaultRepoSeedSummaryPrefix + filepath.Base(repoRoot) + "."
}

func seedSourcePaths(sources []discoveredSeedSource) []string {
	paths := make([]string, 0, len(sources))
	for _, source := range sources {
		paths = append(paths, source.Path)
	}
	return paths
}

func baselineSeedUpToDate(current, desired kuzu.Graph) bool {
	currentNodes := make(map[string]kuzu.EntityRecord, len(current.Nodes))
	for _, node := range current.Nodes {
		currentNodes[node.ID] = node
	}
	for _, node := range desired.Nodes {
		currentNode, ok := currentNodes[node.ID]
		if !ok || !baselineSeedNodeEqual(currentNode, node) {
			return false
		}
	}

	currentEdges := make(map[string]kuzu.RelationRecord, len(current.Edges))
	for _, edge := range current.Edges {
		currentEdges[baselineSeedEdgeKey(edge)] = edge
	}
	for _, edge := range desired.Edges {
		currentEdge, ok := currentEdges[baselineSeedEdgeKey(edge)]
		if !ok || !baselineSeedEdgeEqual(currentEdge, edge) {
			return false
		}
	}
	return true
}

func baselineSeedNodeEqual(current, desired kuzu.EntityRecord) bool {
	return current.Kind == desired.Kind &&
		current.Title == desired.Title &&
		current.Summary == desired.Summary &&
		current.Content == desired.Content &&
		current.RepoPath == desired.RepoPath &&
		strings.Join(normalizedSeedTags(current.Tags), "\x00") == strings.Join(normalizedSeedTags(desired.Tags), "\x00") &&
		seedMapsEqual(current.Props, desired.Props)
}

func baselineSeedEdgeEqual(current, desired kuzu.RelationRecord) bool {
	return current.From == desired.From &&
		current.To == desired.To &&
		current.Kind == desired.Kind &&
		seedMapsEqual(current.Props, desired.Props)
}

func baselineSeedEdgeKey(edge kuzu.RelationRecord) string {
	return edge.From + "\x00" + edge.Kind + "\x00" + edge.To
}

func normalizedSeedTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	normalized := append([]string(nil), tags...)
	sort.Strings(normalized)
	return normalized
}

func seedMapsEqual(left, right map[string]any) bool {
	if len(left) != len(right) {
		return false
	}
	leftJSON, err := json.Marshal(left)
	if err != nil {
		return false
	}
	rightJSON, err := json.Marshal(right)
	if err != nil {
		return false
	}
	return string(leftJSON) == string(rightJSON)
}

func summarizeMarkdownDocument(content, fallbackTitle string) (string, string, string) {
	lines := strings.Split(content, "\n")
	title := ""
	inCodeFence := false
	paragraphs := []string{}
	current := []string{}

	flush := func() {
		if len(current) == 0 {
			return
		}
		paragraphs = append(paragraphs, strings.Join(current, " "))
		current = nil
	}

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if strings.HasPrefix(line, "```") {
			inCodeFence = !inCodeFence
			flush()
			continue
		}
		if inCodeFence {
			continue
		}
		if title == "" && strings.HasPrefix(line, "# ") {
			title = strings.TrimSpace(strings.TrimPrefix(line, "# "))
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, ">"))
		if line == "" {
			flush()
			continue
		}
		if strings.HasPrefix(line, "#") {
			flush()
			continue
		}
		current = append(current, line)
	}
	flush()

	summary := ""
	if len(paragraphs) > 0 {
		summary = paragraphs[0]
	}
	excerptSource := summary
	if len(paragraphs) > 1 {
		excerptSource = paragraphs[0] + " " + paragraphs[1]
	}

	return firstNonEmpty(title, fallbackTitle), excerptText(summary, 240), excerptText(excerptSource, 400)
}

func yamlScalarValue(payload, key string) string {
	prefix := key + ":"
	for _, line := range strings.Split(payload, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, prefix) {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
		return strings.Trim(value, `"'`)
	}
	return ""
}

func excerptText(value string, max int) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if max <= 0 || len(value) <= max {
		return value
	}
	if max <= 3 {
		return value[:max]
	}
	return strings.TrimSpace(value[:max-3]) + "..."
}

func marshalSeedObjects[T any](values []T) ([]json.RawMessage, error) {
	raw := make([]json.RawMessage, 0, len(values))
	for _, value := range values {
		payload, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("marshal baseline seed payload: %w", err)
		}
		raw = append(raw, payload)
	}
	return raw, nil
}

func cloneAnyMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}
