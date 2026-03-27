package kuzu

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	kuzudb "github.com/kuzudb/go-kuzu"

	graphpayload "github.com/guillaume-galp/cge/internal/domain/payload"
	"github.com/guillaume-galp/cge/internal/infra/repo"
)

type CurrentRevisionState struct {
	Exists   bool                 `json:"exists"`
	Revision RevisionDiffMetadata `json:"revision,omitempty"`
}

type GraphSyncSummary struct {
	Nodes    NodeSyncSummary      `json:"nodes"`
	Edges    EdgeSyncSummary      `json:"edges"`
	Revision RevisionWriteSummary `json:"revision,omitempty"`
}

type NodeSyncSummary struct {
	Created      []NodeRef `json:"created"`
	Updated      []NodeRef `json:"updated"`
	Removed      []NodeRef `json:"removed"`
	CreatedCount int       `json:"created_count"`
	UpdatedCount int       `json:"updated_count"`
	RemovedCount int       `json:"removed_count"`
}

type EdgeSyncSummary struct {
	Created      []EdgeRef `json:"created"`
	Updated      []EdgeRef `json:"updated"`
	Removed      []EdgeRef `json:"removed"`
	CreatedCount int       `json:"created_count"`
	UpdatedCount int       `json:"updated_count"`
	RemovedCount int       `json:"removed_count"`
}

func (s *Store) CurrentRevision(_ context.Context, workspace repo.Workspace) (CurrentRevisionState, error) {
	dbPath := filepath.Join(workspace.WorkspacePath, "kuzu", StoreFileName)
	if _, err := os.Stat(dbPath); err != nil {
		if os.IsNotExist(err) {
			return CurrentRevisionState{}, nil
		}
		return CurrentRevisionState{}, &PersistenceError{
			Code:    "revision_unavailable",
			Message: "graph revision metadata could not be read",
			Details: map[string]any{"reason": err.Error()},
		}
	}

	config := kuzudb.DefaultSystemConfig()
	config.ReadOnly = true
	db, err := kuzudb.OpenDatabase(dbPath, config)
	if err != nil {
		return CurrentRevisionState{}, &PersistenceError{
			Code:    "revision_unavailable",
			Message: "graph revision metadata could not be read",
			Details: map[string]any{"reason": fmt.Sprintf("open kuzu database: %v", err)},
		}
	}
	defer db.Close()

	conn, err := kuzudb.OpenConnection(db)
	if err != nil {
		return CurrentRevisionState{}, &PersistenceError{
			Code:    "revision_unavailable",
			Message: "graph revision metadata could not be read",
			Details: map[string]any{"reason": fmt.Sprintf("open kuzu connection: %v", err)},
		}
	}
	defer conn.Close()

	revision, ok, err := latestRevisionRecord(conn)
	if err != nil {
		return CurrentRevisionState{}, &PersistenceError{
			Code:    "revision_unavailable",
			Message: "graph revision metadata could not be read",
			Details: map[string]any{"reason": err.Error()},
		}
	}
	if !ok {
		return CurrentRevisionState{}, nil
	}

	return CurrentRevisionState{Exists: true, Revision: revision.RevisionDiffMetadata}, nil
}

func (s *Store) ReplaceGraph(_ context.Context, workspace repo.Workspace, metadata graphpayload.Metadata, target Graph) (GraphSyncSummary, error) {
	dbPath := filepath.Join(workspace.WorkspacePath, "kuzu", StoreFileName)
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return GraphSyncSummary{}, fmt.Errorf("create graph store directory: %w", err)
	}

	db, err := kuzudb.OpenDatabase(dbPath, kuzudb.DefaultSystemConfig())
	if err != nil {
		return GraphSyncSummary{}, fmt.Errorf("open kuzu database: %w", err)
	}
	defer db.Close()

	conn, err := kuzudb.OpenConnection(db)
	if err != nil {
		return GraphSyncSummary{}, fmt.Errorf("open kuzu connection: %w", err)
	}
	defer conn.Close()

	if err := ensureSchema(conn); err != nil {
		return GraphSyncSummary{}, err
	}

	targetNodes := make([]entityInput, 0, len(target.Nodes))
	for _, node := range target.Nodes {
		targetNodes = append(targetNodes, entityInput{
			ID:       node.ID,
			Kind:     node.Kind,
			Title:    node.Title,
			Summary:  node.Summary,
			Content:  node.Content,
			RepoPath: node.RepoPath,
			Language: node.Language,
			Tags:     cloneStrings(node.Tags),
			Props:    cloneProps(node.Props),
		})
	}
	targetEdges := make([]edgeInput, 0, len(target.Edges))
	for _, edge := range target.Edges {
		targetEdges = append(targetEdges, edgeInput{From: edge.From, To: edge.To, Kind: edge.Kind, Props: cloneProps(edge.Props)})
	}
	if err := ensureIncomingRelationshipEndpoints(nodeIDSet(targetNodes), targetEdges); err != nil {
		return GraphSyncSummary{}, err
	}

	current, err := readComparableSnapshot(conn)
	if err != nil {
		return GraphSyncSummary{}, &PersistenceError{
			Code:    "graph_replace_unavailable",
			Message: "graph content could not be synchronized",
			Details: map[string]any{"reason": err.Error()},
		}
	}

	summary := diffTargetGraph(current, target)
	if !summary.hasChanges() {
		return summary.GraphSyncSummary, nil
	}

	if err := beginTransaction(conn); err != nil {
		return GraphSyncSummary{}, err
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		_ = rollbackTransaction(conn)
	}()

	for _, edge := range summary.Edges.Removed {
		if err := deleteRelation(conn, edge); err != nil {
			return GraphSyncSummary{}, err
		}
	}
	for _, node := range summary.Nodes.Removed {
		if err := deleteEntity(conn, node.ID); err != nil {
			return GraphSyncSummary{}, err
		}
	}
	for _, node := range summary.Nodes.Created {
		if err := createEntity(conn, metadata, summary.targetNodeInputs[node.ID]); err != nil {
			return GraphSyncSummary{}, err
		}
	}
	for _, node := range summary.Nodes.Updated {
		if err := updateEntity(conn, metadata, summary.targetNodeInputs[node.ID]); err != nil {
			return GraphSyncSummary{}, err
		}
	}
	for _, edge := range summary.Edges.Created {
		if err := createRelation(conn, metadata, summary.targetEdgeInputs[edgeKey(edge.From, edge.Kind, edge.To)]); err != nil {
			return GraphSyncSummary{}, err
		}
	}
	for _, edge := range summary.Edges.Updated {
		if err := updateRelation(conn, metadata, summary.targetEdgeInputs[edgeKey(edge.From, edge.Kind, edge.To)]); err != nil {
			return GraphSyncSummary{}, err
		}
	}

	writeSummary := WriteSummary{
		Nodes: NodeWriteSummary{Created: append([]NodeRef(nil), summary.Nodes.Created...), Updated: append([]NodeRef(nil), summary.Nodes.Updated...)},
		Edges: EdgeWriteSummary{Created: append([]EdgeRef(nil), summary.Edges.Created...), Updated: append([]EdgeRef(nil), summary.Edges.Updated...)},
	}
	writeSummary.Nodes.CreatedCount = len(writeSummary.Nodes.Created)
	writeSummary.Nodes.UpdatedCount = len(writeSummary.Nodes.Updated)
	writeSummary.Edges.CreatedCount = len(writeSummary.Edges.Created)
	writeSummary.Edges.UpdatedCount = len(writeSummary.Edges.Updated)

	revision, err := s.recordRevision(conn, metadata, writeSummary)
	if err != nil {
		return GraphSyncSummary{}, err
	}
	summary.Revision = revision

	if err := commitTransaction(conn); err != nil {
		return GraphSyncSummary{}, err
	}
	committed = true
	return summary.GraphSyncSummary, nil
}

type graphSyncComputation struct {
	GraphSyncSummary
	targetNodeInputs map[string]entityInput
	targetEdgeInputs map[string]edgeInput
}

func (s GraphSyncSummary) hasChanges() bool {
	return s.Nodes.CreatedCount+s.Nodes.UpdatedCount+s.Nodes.RemovedCount+s.Edges.CreatedCount+s.Edges.UpdatedCount+s.Edges.RemovedCount > 0
}

func diffTargetGraph(current graphSnapshot, target Graph) graphSyncComputation {
	result := graphSyncComputation{
		GraphSyncSummary: GraphSyncSummary{
			Nodes: NodeSyncSummary{Created: []NodeRef{}, Updated: []NodeRef{}, Removed: []NodeRef{}},
			Edges: EdgeSyncSummary{Created: []EdgeRef{}, Updated: []EdgeRef{}, Removed: []EdgeRef{}},
		},
		targetNodeInputs: map[string]entityInput{},
		targetEdgeInputs: map[string]edgeInput{},
	}

	currentNodes := map[string]graphSnapshotNode{}
	for _, node := range current.Nodes {
		currentNodes[node.ID] = node
	}
	currentEdges := map[string]graphSnapshotEdge{}
	for _, edge := range current.Edges {
		currentEdges[edgeKey(edge.From, edge.Kind, edge.To)] = edge
	}

	targetNodeIDs := map[string]struct{}{}
	for _, node := range target.Nodes {
		targetNodeIDs[node.ID] = struct{}{}
		result.targetNodeInputs[node.ID] = entityInput{
			ID:       node.ID,
			Kind:     node.Kind,
			Title:    node.Title,
			Summary:  node.Summary,
			Content:  node.Content,
			RepoPath: node.RepoPath,
			Language: node.Language,
			Tags:     cloneStrings(node.Tags),
			Props:    cloneProps(node.Props),
		}
		currentNode, ok := currentNodes[node.ID]
		ref := NodeRef{ID: node.ID, Kind: node.Kind}
		if !ok {
			result.Nodes.Created = append(result.Nodes.Created, ref)
			continue
		}
		if !sameNodeState(currentNode, node) {
			result.Nodes.Updated = append(result.Nodes.Updated, ref)
		}
	}
	for _, node := range current.Nodes {
		if _, ok := targetNodeIDs[node.ID]; ok {
			continue
		}
		result.Nodes.Removed = append(result.Nodes.Removed, NodeRef{ID: node.ID, Kind: node.Kind})
	}

	targetEdgeKeys := map[string]struct{}{}
	for _, edge := range target.Edges {
		key := edgeKey(edge.From, edge.Kind, edge.To)
		targetEdgeKeys[key] = struct{}{}
		result.targetEdgeInputs[key] = edgeInput{From: edge.From, To: edge.To, Kind: edge.Kind, Props: cloneProps(edge.Props)}
		currentEdge, ok := currentEdges[key]
		ref := EdgeRef{From: edge.From, Kind: edge.Kind, To: edge.To}
		if !ok {
			result.Edges.Created = append(result.Edges.Created, ref)
			continue
		}
		if !sameEdgeState(currentEdge, edge) {
			result.Edges.Updated = append(result.Edges.Updated, ref)
		}
	}
	for _, edge := range current.Edges {
		key := edgeKey(edge.From, edge.Kind, edge.To)
		if _, ok := targetEdgeKeys[key]; ok {
			continue
		}
		result.Edges.Removed = append(result.Edges.Removed, EdgeRef{From: edge.From, Kind: edge.Kind, To: edge.To})
	}

	sort.Slice(result.Nodes.Created, func(i, j int) bool { return result.Nodes.Created[i].ID < result.Nodes.Created[j].ID })
	sort.Slice(result.Nodes.Updated, func(i, j int) bool { return result.Nodes.Updated[i].ID < result.Nodes.Updated[j].ID })
	sort.Slice(result.Nodes.Removed, func(i, j int) bool { return result.Nodes.Removed[i].ID < result.Nodes.Removed[j].ID })
	sort.Slice(result.Edges.Created, func(i, j int) bool { return edgeRefLess(result.Edges.Created[i], result.Edges.Created[j]) })
	sort.Slice(result.Edges.Updated, func(i, j int) bool { return edgeRefLess(result.Edges.Updated[i], result.Edges.Updated[j]) })
	sort.Slice(result.Edges.Removed, func(i, j int) bool { return edgeRefLess(result.Edges.Removed[i], result.Edges.Removed[j]) })

	result.Nodes.CreatedCount = len(result.Nodes.Created)
	result.Nodes.UpdatedCount = len(result.Nodes.Updated)
	result.Nodes.RemovedCount = len(result.Nodes.Removed)
	result.Edges.CreatedCount = len(result.Edges.Created)
	result.Edges.UpdatedCount = len(result.Edges.Updated)
	result.Edges.RemovedCount = len(result.Edges.Removed)
	return result
}

func sameNodeState(current graphSnapshotNode, target EntityRecord) bool {
	return current.Kind == target.Kind &&
		current.Title == target.Title &&
		current.Summary == target.Summary &&
		current.Content == target.Content &&
		current.RepoPath == target.RepoPath &&
		current.Language == target.Language &&
		strings.Join(normalizeTags(current.Tags), "\x00") == strings.Join(normalizeTags(target.Tags), "\x00") &&
		jsonValueEqual(current.Props, target.Props)
}

func sameEdgeState(current graphSnapshotEdge, target RelationRecord) bool {
	return current.Kind == target.Kind && jsonValueEqual(current.Props, target.Props)
}

func deleteEntity(conn *kuzudb.Connection, id string) error {
	result, err := executePrepared(conn,
		fmt.Sprintf(`MATCH (e:%s {id: $id}) DELETE e;`, entityTableName),
		map[string]any{"id": id},
	)
	if err != nil {
		return fmt.Errorf("delete entity %q: %w", id, err)
	}
	defer result.Close()
	return nil
}

func deleteRelation(conn *kuzudb.Connection, edge EdgeRef) error {
	result, err := executePrepared(conn,
		fmt.Sprintf(`MATCH (from:%s {id: $from})-[r:%s {kind: $kind}]->(to:%s {id: $to}) DELETE r;`, entityTableName, relationTableName, entityTableName),
		map[string]any{"from": edge.From, "kind": edge.Kind, "to": edge.To},
	)
	if err != nil {
		return fmt.Errorf("delete relationship %q-%q->%q: %w", edge.From, edge.Kind, edge.To, err)
	}
	defer result.Close()
	return nil
}

func latestRevisionRecord(conn *kuzudb.Connection) (revisionRecord, bool, error) {
	result, err := conn.Query(fmt.Sprintf(`MATCH (e:%s)
WHERE e.kind = '%s'
RETURN e.id, e.summary, e.created_at, e.created_by, e.created_session_id, e.props_json
ORDER BY e.created_at DESC, e.id DESC;`, entityTableName, graphRevisionKind))
	if err != nil {
		return revisionRecord{}, false, fmt.Errorf("query latest revision anchor: %w", err)
	}
	defer result.Close()
	if !result.HasNext() {
		return revisionRecord{}, false, nil
	}

	snapshot, err := readComparableSnapshot(conn)
	if err != nil {
		return revisionRecord{}, false, fmt.Errorf("read current graph snapshot: %w", err)
	}
	currentAnchor, err := comparableSnapshotAnchor(snapshot)
	if err != nil {
		return revisionRecord{}, false, fmt.Errorf("compute current graph snapshot anchor: %w", err)
	}

	var fallback *revisionRecord
	for result.HasNext() {
		tuple, err := result.Next()
		if err != nil {
			return revisionRecord{}, false, fmt.Errorf("read latest revision anchor: %w", err)
		}
		values, err := tuple.GetAsSlice()
		if err != nil {
			return revisionRecord{}, false, fmt.Errorf("decode latest revision anchor: %w", err)
		}
		props := jsonMapValue(values[5])
		revision := revisionRecord{
			RevisionDiffMetadata: RevisionDiffMetadata{
				ID:               stringValue(values[0]),
				Anchor:           stringProp(props, "anchor"),
				Reason:           revisionReason(props, values[1]),
				NodeCount:        intProp(props, "node_count"),
				EdgeCount:        intProp(props, "edge_count"),
				CreatedAt:        stringValue(values[2]),
				CreatedBy:        stringValue(values[3]),
				CreatedSessionID: stringValue(values[4]),
			},
		}
		if fallback == nil {
			candidate := revision
			fallback = &candidate
		}
		if revision.Anchor == currentAnchor {
			return revision, true, nil
		}
	}

	return *fallback, true, nil
}
