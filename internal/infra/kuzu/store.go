package kuzu

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	kuzudb "github.com/kuzudb/go-kuzu"

	graphpayload "github.com/guillaume-galp/cge/internal/domain/payload"
	"github.com/guillaume-galp/cge/internal/infra/repo"
)

const StoreFileName = "db"

const (
	entityTableName               = "Entity"
	relationTableName             = "EntityRelation"
	revisionNodeSnapshotTableName = "RevisionNodeState"
	revisionEdgeSnapshotTableName = "RevisionEdgeState"
	reasoningUnitKind             = "ReasoningUnit"
	agentSessionKind              = "AgentSession"
	graphRevisionKind             = "GraphRevision"
	revisionTitlePrefix           = "Graph revision "
)

type Store struct {
	recordRevisionAnchor func(conn *kuzudb.Connection, metadata graphpayload.Metadata, summary WriteSummary) (RevisionWriteSummary, error)
}

func NewStore() *Store {
	return &Store{}
}

type WriteSummary struct {
	Nodes    NodeWriteSummary     `json:"nodes"`
	Edges    EdgeWriteSummary     `json:"edges"`
	Revision RevisionWriteSummary `json:"revision"`
}

type NodeWriteSummary struct {
	Created      []NodeRef `json:"created"`
	Updated      []NodeRef `json:"updated"`
	CreatedCount int       `json:"created_count"`
	UpdatedCount int       `json:"updated_count"`
}

type EdgeWriteSummary struct {
	Created      []EdgeRef `json:"created"`
	Updated      []EdgeRef `json:"updated"`
	CreatedCount int       `json:"created_count"`
	UpdatedCount int       `json:"updated_count"`
}

type NodeRef struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
}

type EdgeRef struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind"`
}

type RevisionWriteSummary struct {
	ID        string `json:"id"`
	Anchor    string `json:"anchor"`
	Reason    string `json:"reason,omitempty"`
	NodeCount int    `json:"node_count"`
	EdgeCount int    `json:"edge_count"`
}

type PersistenceError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

func (e *PersistenceError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

type entityInput struct {
	ID       string
	Kind     string
	Title    string
	Summary  string
	Content  string
	RepoPath string
	Language string
	Tags     []string
	Props    map[string]any
}

type edgeInput struct {
	From  string
	To    string
	Kind  string
	Props map[string]any
}

func (s *Store) Write(_ context.Context, workspace repo.Workspace, envelope graphpayload.Envelope) (WriteSummary, error) {
	nodes, err := parseNodes(envelope.Nodes)
	if err != nil {
		return WriteSummary{}, err
	}

	edges, err := parseEdges(envelope.Edges)
	if err != nil {
		return WriteSummary{}, err
	}

	dbPath := filepath.Join(workspace.WorkspacePath, "kuzu", StoreFileName)
	storeExists := fileExists(dbPath)
	incomingNodeIDs := nodeIDSet(nodes)
	if !storeExists {
		if err := ensureIncomingRelationshipEndpoints(incomingNodeIDs, edges); err != nil {
			return WriteSummary{}, err
		}
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return WriteSummary{}, fmt.Errorf("create graph store directory: %w", err)
	}

	db, err := kuzudb.OpenDatabase(dbPath, kuzudb.DefaultSystemConfig())
	if err != nil {
		return WriteSummary{}, fmt.Errorf("open kuzu database: %w", err)
	}
	defer db.Close()

	conn, err := kuzudb.OpenConnection(db)
	if err != nil {
		return WriteSummary{}, fmt.Errorf("open kuzu connection: %w", err)
	}
	defer conn.Close()

	if err := ensureSchema(conn); err != nil {
		return WriteSummary{}, err
	}

	if storeExists {
		if err := ensureRelationshipEndpoints(conn, incomingNodeIDs, edges); err != nil {
			return WriteSummary{}, err
		}
	}

	summary := WriteSummary{}
	nodeExists := make(map[string]bool, len(nodes))
	for _, node := range nodes {
		exists, err := entityExists(conn, node.ID)
		if err != nil {
			return WriteSummary{}, err
		}
		nodeExists[node.ID] = exists
		ref := NodeRef{ID: node.ID, Kind: node.Kind}
		if exists {
			summary.Nodes.Updated = append(summary.Nodes.Updated, ref)
			continue
		}
		summary.Nodes.Created = append(summary.Nodes.Created, ref)
	}

	edgeExistsMap := make(map[string]bool, len(edges))
	for _, edge := range edges {
		exists, err := relationExists(conn, edge.From, edge.Kind, edge.To)
		if err != nil {
			return WriteSummary{}, err
		}
		edgeExistsMap[edgeKey(edge.From, edge.Kind, edge.To)] = exists
		ref := EdgeRef{From: edge.From, To: edge.To, Kind: edge.Kind}
		if exists {
			summary.Edges.Updated = append(summary.Edges.Updated, ref)
			continue
		}
		summary.Edges.Created = append(summary.Edges.Created, ref)
	}

	if err := beginTransaction(conn); err != nil {
		return WriteSummary{}, err
	}

	committed := false
	defer func() {
		if committed {
			return
		}
		_ = rollbackTransaction(conn)
	}()

	for _, node := range nodes {
		if nodeExists[node.ID] {
			if err := updateEntity(conn, envelope.Metadata, node); err != nil {
				return WriteSummary{}, err
			}
			continue
		}
		if err := createEntity(conn, envelope.Metadata, node); err != nil {
			return WriteSummary{}, err
		}
	}

	for _, edge := range edges {
		if edgeExistsMap[edgeKey(edge.From, edge.Kind, edge.To)] {
			if err := updateRelation(conn, envelope.Metadata, edge); err != nil {
				return WriteSummary{}, err
			}
			continue
		}
		if err := createRelation(conn, envelope.Metadata, edge); err != nil {
			return WriteSummary{}, err
		}
	}

	sort.Slice(summary.Nodes.Created, func(i, j int) bool { return summary.Nodes.Created[i].ID < summary.Nodes.Created[j].ID })
	sort.Slice(summary.Nodes.Updated, func(i, j int) bool { return summary.Nodes.Updated[i].ID < summary.Nodes.Updated[j].ID })
	sort.Slice(summary.Edges.Created, func(i, j int) bool {
		return edgeRefLess(summary.Edges.Created[i], summary.Edges.Created[j])
	})
	sort.Slice(summary.Edges.Updated, func(i, j int) bool {
		return edgeRefLess(summary.Edges.Updated[i], summary.Edges.Updated[j])
	})

	summary.Nodes.CreatedCount = len(summary.Nodes.Created)
	summary.Nodes.UpdatedCount = len(summary.Nodes.Updated)
	summary.Edges.CreatedCount = len(summary.Edges.Created)
	summary.Edges.UpdatedCount = len(summary.Edges.Updated)

	revision, err := s.recordRevision(conn, envelope.Metadata, summary)
	if err != nil {
		return WriteSummary{}, err
	}
	summary.Revision = revision

	if err := commitTransaction(conn); err != nil {
		return WriteSummary{}, err
	}
	committed = true

	return summary, nil
}

func ensureSchema(conn *kuzudb.Connection) error {
	queries := []string{
		fmt.Sprintf(`CREATE NODE TABLE IF NOT EXISTS %s(
			id STRING,
			kind STRING,
			title STRING,
			summary STRING,
			content STRING,
			repo_path STRING,
			language STRING,
			tags STRING[],
			props_json STRING,
			created_at STRING,
			updated_at STRING,
			created_by STRING,
			updated_by STRING,
			created_session_id STRING,
			updated_session_id STRING,
			PRIMARY KEY(id)
		);`, entityTableName),
		fmt.Sprintf(`CREATE REL TABLE IF NOT EXISTS %s(
			FROM %s TO %s,
			kind STRING,
			props_json STRING,
			created_at STRING,
			updated_at STRING,
			created_by STRING,
			updated_by STRING,
			created_session_id STRING,
			updated_session_id STRING,
			MANY_MANY
		);`, relationTableName, entityTableName, entityTableName),
		fmt.Sprintf(`CREATE NODE TABLE IF NOT EXISTS %s(
			id STRING,
			revision_id STRING,
			entity_id STRING,
			kind STRING,
			title STRING,
			summary STRING,
			content STRING,
			repo_path STRING,
			language STRING,
			tags STRING[],
			props_json STRING,
			created_at STRING,
			updated_at STRING,
			created_by STRING,
			updated_by STRING,
			created_session_id STRING,
			updated_session_id STRING,
			PRIMARY KEY(id)
		);`, revisionNodeSnapshotTableName),
		fmt.Sprintf(`CREATE NODE TABLE IF NOT EXISTS %s(
			id STRING,
			revision_id STRING,
			from_id STRING,
			to_id STRING,
			kind STRING,
			props_json STRING,
			created_at STRING,
			updated_at STRING,
			created_by STRING,
			updated_by STRING,
			created_session_id STRING,
			updated_session_id STRING,
			PRIMARY KEY(id)
		);`, revisionEdgeSnapshotTableName),
	}

	for _, query := range queries {
		if err := executeQuery(conn, query); err != nil {
			return fmt.Errorf("ensure kuzu schema: %w", err)
		}
	}

	return nil
}

func entityExists(conn *kuzudb.Connection, id string) (bool, error) {
	result, err := executePrepared(conn,
		fmt.Sprintf(`MATCH (e:%s {id: $id}) RETURN e.id LIMIT 1;`, entityTableName),
		map[string]any{"id": id},
	)
	if err != nil {
		return false, fmt.Errorf("check entity existence for %q: %w", id, err)
	}
	defer result.Close()

	return result.HasNext(), nil
}

func relationExists(conn *kuzudb.Connection, from, kind, to string) (bool, error) {
	result, err := executePrepared(conn,
		fmt.Sprintf(`MATCH (from:%s {id: $from})-[r:%s {kind: $kind}]->(to:%s {id: $to}) RETURN r.kind LIMIT 1;`,
			entityTableName, relationTableName, entityTableName),
		map[string]any{
			"from": from,
			"kind": kind,
			"to":   to,
		},
	)
	if err != nil {
		return false, fmt.Errorf("check relationship existence for %q-%q->%q: %w", from, kind, to, err)
	}
	defer result.Close()

	return result.HasNext(), nil
}

func createEntity(conn *kuzudb.Connection, metadata graphpayload.Metadata, node entityInput) error {
	result, err := executePrepared(conn,
		fmt.Sprintf(`CREATE (:%s {
			id: $id,
			kind: $kind,
			title: $title,
			summary: $summary,
			content: $content,
			repo_path: $repo_path,
			language: $language,
			tags: $tags,
			props_json: $props_json,
			created_at: $created_at,
			updated_at: $updated_at,
			created_by: $created_by,
			updated_by: $updated_by,
			created_session_id: $created_session_id,
			updated_session_id: $updated_session_id
		});`, entityTableName),
		createEntityParams(metadata, node),
	)
	if err != nil {
		return fmt.Errorf("create entity %q: %w", node.ID, err)
	}
	defer result.Close()
	return nil
}

func updateEntity(conn *kuzudb.Connection, metadata graphpayload.Metadata, node entityInput) error {
	result, err := executePrepared(conn,
		fmt.Sprintf(`MATCH (e:%s {id: $id})
		SET e.kind = $kind,
			e.title = $title,
			e.summary = $summary,
			e.content = $content,
			e.repo_path = $repo_path,
			e.language = $language,
			e.tags = $tags,
			e.props_json = $props_json,
			e.updated_at = $updated_at,
			e.updated_by = $updated_by,
			e.updated_session_id = $updated_session_id
		RETURN e.id;`, entityTableName),
		updateEntityParams(metadata, node),
	)
	if err != nil {
		return fmt.Errorf("update entity %q: %w", node.ID, err)
	}
	defer result.Close()
	return nil
}

func createRelation(conn *kuzudb.Connection, metadata graphpayload.Metadata, edge edgeInput) error {
	result, err := executePrepared(conn,
		fmt.Sprintf(`MATCH (from:%s {id: $from}), (to:%s {id: $to})
		CREATE (from)-[:%s {
			kind: $kind,
			props_json: $props_json,
			created_at: $created_at,
			updated_at: $updated_at,
			created_by: $created_by,
			updated_by: $updated_by,
			created_session_id: $created_session_id,
			updated_session_id: $updated_session_id
		}]->(to);`, entityTableName, entityTableName, relationTableName),
		createRelationParams(metadata, edge),
	)
	if err != nil {
		return fmt.Errorf("create relationship %q-%q->%q: %w", edge.From, edge.Kind, edge.To, err)
	}
	defer result.Close()
	return nil
}

func updateRelation(conn *kuzudb.Connection, metadata graphpayload.Metadata, edge edgeInput) error {
	result, err := executePrepared(conn,
		fmt.Sprintf(`MATCH (from:%s {id: $from})-[r:%s {kind: $kind}]->(to:%s {id: $to})
		SET r.props_json = $props_json,
			r.updated_at = $updated_at,
			r.updated_by = $updated_by,
			r.updated_session_id = $updated_session_id
		RETURN r.kind;`, entityTableName, relationTableName, entityTableName),
		updateRelationParams(metadata, edge),
	)
	if err != nil {
		return fmt.Errorf("update relationship %q-%q->%q: %w", edge.From, edge.Kind, edge.To, err)
	}
	defer result.Close()
	return nil
}

func beginTransaction(conn *kuzudb.Connection) error {
	if err := executeQuery(conn, "BEGIN TRANSACTION;"); err != nil {
		return fmt.Errorf("begin kuzu transaction: %w", err)
	}
	return nil
}

func commitTransaction(conn *kuzudb.Connection) error {
	if err := executeQuery(conn, "COMMIT;"); err != nil {
		return fmt.Errorf("commit kuzu transaction: %w", err)
	}
	return nil
}

func rollbackTransaction(conn *kuzudb.Connection) error {
	if err := executeQuery(conn, "ROLLBACK;"); err != nil {
		return fmt.Errorf("rollback kuzu transaction: %w", err)
	}
	return nil
}

func executeQuery(conn *kuzudb.Connection, query string) error {
	result, err := conn.Query(query)
	if result != nil {
		defer result.Close()
	}
	if err != nil {
		return err
	}
	return nil
}

func executePrepared(conn *kuzudb.Connection, query string, args map[string]any) (*kuzudb.QueryResult, error) {
	statement, err := conn.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer statement.Close()

	result, err := conn.Execute(statement, args)
	if err != nil {
		if result != nil {
			result.Close()
		}
		return nil, err
	}

	return result, nil
}

type graphSnapshot struct {
	Nodes []graphSnapshotNode `json:"nodes"`
	Edges []graphSnapshotEdge `json:"edges"`
}

type graphSnapshotNode struct {
	ID               string         `json:"id"`
	Kind             string         `json:"kind"`
	Title            string         `json:"title,omitempty"`
	Summary          string         `json:"summary,omitempty"`
	Content          string         `json:"content,omitempty"`
	RepoPath         string         `json:"repo_path,omitempty"`
	Language         string         `json:"language,omitempty"`
	Tags             []string       `json:"tags,omitempty"`
	Props            map[string]any `json:"props,omitempty"`
	CreatedAt        string         `json:"created_at,omitempty"`
	UpdatedAt        string         `json:"updated_at,omitempty"`
	CreatedBy        string         `json:"created_by,omitempty"`
	UpdatedBy        string         `json:"updated_by,omitempty"`
	CreatedSessionID string         `json:"created_session_id,omitempty"`
	UpdatedSessionID string         `json:"updated_session_id,omitempty"`
}

type graphSnapshotEdge struct {
	From             string         `json:"from"`
	To               string         `json:"to"`
	Kind             string         `json:"kind"`
	Props            map[string]any `json:"props,omitempty"`
	CreatedAt        string         `json:"created_at,omitempty"`
	UpdatedAt        string         `json:"updated_at,omitempty"`
	CreatedBy        string         `json:"created_by,omitempty"`
	UpdatedBy        string         `json:"updated_by,omitempty"`
	CreatedSessionID string         `json:"created_session_id,omitempty"`
	UpdatedSessionID string         `json:"updated_session_id,omitempty"`
}

func (s *Store) recordRevision(conn *kuzudb.Connection, metadata graphpayload.Metadata, summary WriteSummary) (RevisionWriteSummary, error) {
	if s != nil && s.recordRevisionAnchor != nil {
		return s.recordRevisionAnchor(conn, metadata, summary)
	}
	return defaultRecordRevisionAnchor(conn, metadata, summary)
}

func defaultRecordRevisionAnchor(conn *kuzudb.Connection, metadata graphpayload.Metadata, summary WriteSummary) (RevisionWriteSummary, error) {
	snapshot, err := readComparableSnapshot(conn)
	if err != nil {
		return RevisionWriteSummary{}, &PersistenceError{
			Code:    "revision_anchor_unavailable",
			Message: "graph write could not record revision metadata",
			Details: map[string]any{
				"reason": err.Error(),
			},
		}
	}

	anchor, err := comparableSnapshotAnchor(snapshot)
	if err != nil {
		return RevisionWriteSummary{}, &PersistenceError{
			Code:    "revision_anchor_unavailable",
			Message: "graph write could not record revision metadata",
			Details: map[string]any{
				"reason": fmt.Sprintf("compute revision anchor: %v", err),
			},
		}
	}

	revisionID, err := generateRevisionID()
	if err != nil {
		return RevisionWriteSummary{}, &PersistenceError{
			Code:    "revision_anchor_unavailable",
			Message: "graph write could not record revision metadata",
			Details: map[string]any{
				"reason": fmt.Sprintf("generate revision id: %v", err),
			},
		}
	}

	revisionNode := entityInput{
		ID:      revisionID,
		Kind:    graphRevisionKind,
		Title:   revisionTitlePrefix + revisionID,
		Summary: revisionSummary(metadata.Revision.Reason, metadata),
		Props: revisionProps(
			anchor,
			metadata.Revision,
			summary,
			len(snapshot.Nodes),
			len(snapshot.Edges),
		),
	}

	if err := createEntity(conn, metadata, revisionNode); err != nil {
		return RevisionWriteSummary{}, &PersistenceError{
			Code:    "revision_anchor_unavailable",
			Message: "graph write could not record revision metadata",
			Details: map[string]any{
				"reason": err.Error(),
			},
		}
	}

	if err := persistRevisionSnapshot(conn, revisionID, snapshot); err != nil {
		return RevisionWriteSummary{}, &PersistenceError{
			Code:    "revision_anchor_unavailable",
			Message: "graph write could not record revision metadata",
			Details: map[string]any{
				"reason": err.Error(),
			},
		}
	}

	return RevisionWriteSummary{
		ID:        revisionID,
		Anchor:    anchor,
		Reason:    metadata.Revision.Reason,
		NodeCount: len(snapshot.Nodes),
		EdgeCount: len(snapshot.Edges),
	}, nil
}

func revisionSummary(reason string, metadata graphpayload.Metadata) string {
	if strings.TrimSpace(reason) != "" {
		return reason
	}
	return fmt.Sprintf("Graph write by %s in session %s at %s", metadata.AgentID, metadata.SessionID, metadata.Timestamp)
}

func revisionProps(anchor string, metadata graphpayload.RevisionMetadata, summary WriteSummary, nodeCount, edgeCount int) map[string]any {
	props := cloneProps(metadata.Properties)
	if props == nil {
		props = map[string]any{}
	}

	props["anchor"] = anchor
	props["node_count"] = nodeCount
	props["edge_count"] = edgeCount
	props["touched_nodes"] = map[string]any{
		"created": summary.Nodes.Created,
		"updated": summary.Nodes.Updated,
	}
	props["touched_edges"] = map[string]any{
		"created": summary.Edges.Created,
		"updated": summary.Edges.Updated,
	}
	if strings.TrimSpace(metadata.Reason) != "" {
		props["reason"] = metadata.Reason
	}

	return props
}

func persistRevisionSnapshot(conn *kuzudb.Connection, revisionID string, snapshot graphSnapshot) error {
	for _, node := range snapshot.Nodes {
		result, err := executePrepared(conn,
			fmt.Sprintf(`CREATE (:%s {
				id: $id,
				revision_id: $revision_id,
				entity_id: $entity_id,
				kind: $kind,
				title: $title,
				summary: $summary,
				content: $content,
				repo_path: $repo_path,
				language: $language,
				tags: $tags,
				props_json: $props_json,
				created_at: $created_at,
				updated_at: $updated_at,
				created_by: $created_by,
				updated_by: $updated_by,
				created_session_id: $created_session_id,
				updated_session_id: $updated_session_id
			});`, revisionNodeSnapshotTableName),
			map[string]any{
				"id":                 revisionSnapshotNodeID(revisionID, node.ID),
				"revision_id":        revisionID,
				"entity_id":          node.ID,
				"kind":               node.Kind,
				"title":              nullIfEmpty(node.Title),
				"summary":            nullIfEmpty(node.Summary),
				"content":            nullIfEmpty(node.Content),
				"repo_path":          nullIfEmpty(node.RepoPath),
				"language":           nullIfEmpty(node.Language),
				"tags":               nullIfEmptyStrings(node.Tags),
				"props_json":         marshalProps(node.Props),
				"created_at":         nullIfEmpty(node.CreatedAt),
				"updated_at":         nullIfEmpty(node.UpdatedAt),
				"created_by":         nullIfEmpty(node.CreatedBy),
				"updated_by":         nullIfEmpty(node.UpdatedBy),
				"created_session_id": nullIfEmpty(node.CreatedSessionID),
				"updated_session_id": nullIfEmpty(node.UpdatedSessionID),
			},
		)
		if err != nil {
			return fmt.Errorf("persist revision node snapshot %q: %w", node.ID, err)
		}
		result.Close()
	}

	for _, edge := range snapshot.Edges {
		result, err := executePrepared(conn,
			fmt.Sprintf(`CREATE (:%s {
				id: $id,
				revision_id: $revision_id,
				from_id: $from_id,
				to_id: $to_id,
				kind: $kind,
				props_json: $props_json,
				created_at: $created_at,
				updated_at: $updated_at,
				created_by: $created_by,
				updated_by: $updated_by,
				created_session_id: $created_session_id,
				updated_session_id: $updated_session_id
			});`, revisionEdgeSnapshotTableName),
			map[string]any{
				"id":                 revisionSnapshotEdgeID(revisionID, edge.From, edge.Kind, edge.To),
				"revision_id":        revisionID,
				"from_id":            edge.From,
				"to_id":              edge.To,
				"kind":               edge.Kind,
				"props_json":         marshalProps(edge.Props),
				"created_at":         nullIfEmpty(edge.CreatedAt),
				"updated_at":         nullIfEmpty(edge.UpdatedAt),
				"created_by":         nullIfEmpty(edge.CreatedBy),
				"updated_by":         nullIfEmpty(edge.UpdatedBy),
				"created_session_id": nullIfEmpty(edge.CreatedSessionID),
				"updated_session_id": nullIfEmpty(edge.UpdatedSessionID),
			},
		)
		if err != nil {
			return fmt.Errorf("persist revision relationship snapshot %q-%q->%q: %w", edge.From, edge.Kind, edge.To, err)
		}
		result.Close()
	}

	return nil
}

func revisionSnapshotNodeID(revisionID, entityID string) string {
	return revisionID + "|node|" + entityID
}

func revisionSnapshotEdgeID(revisionID, from, kind, to string) string {
	return revisionID + "|edge|" + hex.EncodeToString([]byte(edgeKey(from, kind, to)))
}

func generateRevisionID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return "rev:" + hex.EncodeToString(raw[:]), nil
}

func comparableSnapshotAnchor(snapshot graphSnapshot) (string, error) {
	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		return "", err
	}
	anchorSum := sha256.Sum256(snapshotJSON)
	return hex.EncodeToString(anchorSum[:]), nil
}

func readComparableSnapshot(conn *kuzudb.Connection) (graphSnapshot, error) {
	snapshot := graphSnapshot{
		Nodes: []graphSnapshotNode{},
		Edges: []graphSnapshotEdge{},
	}

	nodeResult, err := conn.Query(fmt.Sprintf(`MATCH (e:%s)
WHERE e.kind <> '%s'
RETURN e.id, e.kind, e.title, e.summary, e.content, e.repo_path, e.language, e.tags, e.props_json, e.created_at, e.updated_at, e.created_by, e.updated_by, e.created_session_id, e.updated_session_id
ORDER BY e.id;`, entityTableName, graphRevisionKind))
	if err != nil {
		return graphSnapshot{}, fmt.Errorf("query comparable nodes: %w", err)
	}
	defer nodeResult.Close()

	for nodeResult.HasNext() {
		tuple, err := nodeResult.Next()
		if err != nil {
			return graphSnapshot{}, fmt.Errorf("read comparable node tuple: %w", err)
		}
		values, err := tuple.GetAsSlice()
		if err != nil {
			return graphSnapshot{}, fmt.Errorf("decode comparable node tuple: %w", err)
		}

		snapshot.Nodes = append(snapshot.Nodes, graphSnapshotNode{
			ID:               stringValue(values[0]),
			Kind:             stringValue(values[1]),
			Title:            stringValue(values[2]),
			Summary:          stringValue(values[3]),
			Content:          stringValue(values[4]),
			RepoPath:         stringValue(values[5]),
			Language:         stringValue(values[6]),
			Tags:             stringSliceValue(values[7]),
			Props:            jsonMapValue(values[8]),
			CreatedAt:        stringValue(values[9]),
			UpdatedAt:        stringValue(values[10]),
			CreatedBy:        stringValue(values[11]),
			UpdatedBy:        stringValue(values[12]),
			CreatedSessionID: stringValue(values[13]),
			UpdatedSessionID: stringValue(values[14]),
		})
	}

	seenEdges := map[string]struct{}{}
	edgeResult, err := conn.Query(fmt.Sprintf(`MATCH (from:%s)-[r:%s]->(to:%s)
WHERE from.kind <> '%s' AND to.kind <> '%s'
RETURN from.id, to.id, r.kind, r.props_json, r.created_at, r.updated_at, r.created_by, r.updated_by, r.created_session_id, r.updated_session_id
ORDER BY from.id, r.kind, to.id;`, entityTableName, relationTableName, entityTableName, graphRevisionKind, graphRevisionKind))
	if err != nil {
		return graphSnapshot{}, fmt.Errorf("query comparable relationships: %w", err)
	}
	defer edgeResult.Close()

	for edgeResult.HasNext() {
		tuple, err := edgeResult.Next()
		if err != nil {
			return graphSnapshot{}, fmt.Errorf("read comparable relationship tuple: %w", err)
		}
		values, err := tuple.GetAsSlice()
		if err != nil {
			return graphSnapshot{}, fmt.Errorf("decode comparable relationship tuple: %w", err)
		}
		edge := graphSnapshotEdge{
			From:             stringValue(values[0]),
			To:               stringValue(values[1]),
			Kind:             stringValue(values[2]),
			Props:            jsonMapValue(values[3]),
			CreatedAt:        stringValue(values[4]),
			UpdatedAt:        stringValue(values[5]),
			CreatedBy:        stringValue(values[6]),
			UpdatedBy:        stringValue(values[7]),
			CreatedSessionID: stringValue(values[8]),
			UpdatedSessionID: stringValue(values[9]),
		}
		key := edgeKey(edge.From, edge.Kind, edge.To)
		if _, duplicate := seenEdges[key]; duplicate {
			continue
		}
		seenEdges[key] = struct{}{}
		snapshot.Edges = append(snapshot.Edges, edge)
	}

	return snapshot, nil
}

func createEntityParams(metadata graphpayload.Metadata, node entityInput) map[string]any {
	return map[string]any{
		"id":                 node.ID,
		"kind":               node.Kind,
		"title":              nullIfEmpty(node.Title),
		"summary":            nullIfEmpty(node.Summary),
		"content":            nullIfEmpty(node.Content),
		"repo_path":          nullIfEmpty(node.RepoPath),
		"language":           nullIfEmpty(node.Language),
		"tags":               nullIfEmptyStrings(node.Tags),
		"props_json":         marshalProps(node.Props),
		"created_at":         metadata.Timestamp,
		"updated_at":         metadata.Timestamp,
		"created_by":         metadata.AgentID,
		"updated_by":         metadata.AgentID,
		"created_session_id": metadata.SessionID,
		"updated_session_id": metadata.SessionID,
	}
}

func updateEntityParams(metadata graphpayload.Metadata, node entityInput) map[string]any {
	return map[string]any{
		"id":                 node.ID,
		"kind":               node.Kind,
		"title":              nullIfEmpty(node.Title),
		"summary":            nullIfEmpty(node.Summary),
		"content":            nullIfEmpty(node.Content),
		"repo_path":          nullIfEmpty(node.RepoPath),
		"language":           nullIfEmpty(node.Language),
		"tags":               nullIfEmptyStrings(node.Tags),
		"props_json":         marshalProps(node.Props),
		"updated_at":         metadata.Timestamp,
		"updated_by":         metadata.AgentID,
		"updated_session_id": metadata.SessionID,
	}
}

func createRelationParams(metadata graphpayload.Metadata, edge edgeInput) map[string]any {
	return map[string]any{
		"from":               edge.From,
		"to":                 edge.To,
		"kind":               edge.Kind,
		"props_json":         marshalProps(edge.Props),
		"created_at":         metadata.Timestamp,
		"updated_at":         metadata.Timestamp,
		"created_by":         metadata.AgentID,
		"updated_by":         metadata.AgentID,
		"created_session_id": metadata.SessionID,
		"updated_session_id": metadata.SessionID,
	}
}

func updateRelationParams(metadata graphpayload.Metadata, edge edgeInput) map[string]any {
	return map[string]any{
		"from":               edge.From,
		"to":                 edge.To,
		"kind":               edge.Kind,
		"props_json":         marshalProps(edge.Props),
		"updated_at":         metadata.Timestamp,
		"updated_by":         metadata.AgentID,
		"updated_session_id": metadata.SessionID,
	}
}

func nullIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func nullIfEmptyStrings(values []string) any {
	if len(values) == 0 {
		return nil
	}
	return cloneStrings(values)
}

func marshalProps(values map[string]any) any {
	if len(values) == 0 {
		return nil
	}
	payload, err := json.Marshal(values)
	if err != nil {
		return fmt.Sprintf(`{"_encoding_error":%q}`, err.Error())
	}
	return string(payload)
}

func ensureIncomingRelationshipEndpoints(nodeIDs map[string]struct{}, edges []edgeInput) error {
	for _, edge := range edges {
		missing := make([]map[string]string, 0, 2)
		if _, ok := nodeIDs[edge.From]; !ok {
			missing = append(missing, map[string]string{
				"field": "from",
				"id":    edge.From,
			})
		}
		if _, ok := nodeIDs[edge.To]; !ok {
			missing = append(missing, map[string]string{
				"field": "to",
				"id":    edge.To,
			})
		}
		if len(missing) == 0 {
			continue
		}

		return unresolvedRelationshipError(edge, missing)
	}

	return nil
}

func ensureRelationshipEndpoints(conn *kuzudb.Connection, incomingNodeIDs map[string]struct{}, edges []edgeInput) error {
	for _, edge := range edges {
		missing := make([]map[string]string, 0, 2)
		if _, ok := incomingNodeIDs[edge.From]; !ok {
			exists, err := entityExists(conn, edge.From)
			if err != nil {
				return err
			}
			if !exists {
				missing = append(missing, map[string]string{
					"field": "from",
					"id":    edge.From,
				})
			}
		}
		if _, ok := incomingNodeIDs[edge.To]; !ok {
			exists, err := entityExists(conn, edge.To)
			if err != nil {
				return err
			}
			if !exists {
				missing = append(missing, map[string]string{
					"field": "to",
					"id":    edge.To,
				})
			}
		}
		if len(missing) == 0 {
			continue
		}

		return unresolvedRelationshipError(edge, missing)
	}

	return nil
}

func unresolvedRelationshipError(edge edgeInput, missing []map[string]string) error {
	return &PersistenceError{
		Code:    "unresolved_relationship_endpoint",
		Message: "relationship references a node that is not present in the payload or graph",
		Details: map[string]any{
			"edge": map[string]string{
				"from": edge.From,
				"to":   edge.To,
				"kind": edge.Kind,
			},
			"missing_endpoints": missing,
		},
	}
}

func nodeIDSet(nodes []entityInput) map[string]struct{} {
	ids := make(map[string]struct{}, len(nodes))
	for _, node := range nodes {
		ids[node.ID] = struct{}{}
	}
	return ids
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func parseNodes(rawNodes []json.RawMessage) ([]entityInput, error) {
	nodes := make([]entityInput, 0, len(rawNodes))
	for i, raw := range rawNodes {
		node, err := parseNode(raw)
		if err != nil {
			return nil, &PersistenceError{
				Code:    "invalid_node_payload",
				Message: "graph payload node is invalid for persistence",
				Details: map[string]any{
					"index":  i,
					"reason": err.Error(),
				},
			}
		}
		if missing := missingRequiredProvenance(node); len(missing) > 0 {
			return nil, &PersistenceError{
				Code:    "incomplete_provenance",
				Message: "graph payload node is missing required provenance metadata",
				Details: map[string]any{
					"index":          i,
					"node_id":        node.ID,
					"kind":           node.Kind,
					"missing_fields": missing,
				},
			}
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func parseEdges(rawEdges []json.RawMessage) ([]edgeInput, error) {
	edges := make([]edgeInput, 0, len(rawEdges))
	for i, raw := range rawEdges {
		edge, err := parseEdge(raw)
		if err != nil {
			return nil, &PersistenceError{
				Code:    "invalid_edge_payload",
				Message: "graph payload relationship is invalid for persistence",
				Details: map[string]any{
					"index":  i,
					"reason": err.Error(),
				},
			}
		}
		edges = append(edges, edge)
	}
	return edges, nil
}

func parseNode(raw json.RawMessage) (entityInput, error) {
	object := map[string]any{}
	if err := json.Unmarshal(raw, &object); err != nil {
		return entityInput{}, fmt.Errorf("node must be a JSON object")
	}

	id, ok := requiredString(object, "id")
	if !ok {
		return entityInput{}, fmt.Errorf("node.id must be a non-empty string")
	}

	kind, ok := firstNonEmptyString(object, "kind", "type")
	if !ok {
		return entityInput{}, fmt.Errorf("node.kind must be a non-empty string")
	}

	props := mergedProperties(object)
	title := firstStringFromMaps([]map[string]any{object, props}, "title")
	summary := firstStringFromMaps([]map[string]any{object, props}, "summary")
	content := firstStringFromMaps([]map[string]any{object, props}, "content")
	repoPath := firstStringFromMaps([]map[string]any{object, props}, "repo_path")
	language := firstStringFromMaps([]map[string]any{object, props}, "language")
	tags := firstStringSlice("tags", object, props)

	delete(props, "title")
	delete(props, "summary")
	delete(props, "content")
	delete(props, "repo_path")
	delete(props, "language")
	delete(props, "tags")

	return entityInput{
		ID:       id,
		Kind:     kind,
		Title:    title,
		Summary:  summary,
		Content:  content,
		RepoPath: repoPath,
		Language: language,
		Tags:     tags,
		Props:    props,
	}, nil
}

func parseEdge(raw json.RawMessage) (edgeInput, error) {
	object := map[string]any{}
	if err := json.Unmarshal(raw, &object); err != nil {
		return edgeInput{}, fmt.Errorf("relationship must be a JSON object")
	}

	from, ok := firstNonEmptyString(object, "from", "source")
	if !ok {
		return edgeInput{}, fmt.Errorf("relationship.from must be a non-empty string")
	}

	to, ok := firstNonEmptyString(object, "to", "target")
	if !ok {
		return edgeInput{}, fmt.Errorf("relationship.to must be a non-empty string")
	}

	kind, ok := firstNonEmptyString(object, "kind", "type")
	if !ok {
		return edgeInput{}, fmt.Errorf("relationship.kind must be a non-empty string")
	}

	props := mergedProperties(object)
	delete(props, "from")
	delete(props, "source")
	delete(props, "to")
	delete(props, "target")
	delete(props, "kind")
	delete(props, "type")

	return edgeInput{
		From:  from,
		To:    to,
		Kind:  kind,
		Props: props,
	}, nil
}

func missingRequiredProvenance(node entityInput) []string {
	switch node.Kind {
	case reasoningUnitKind:
		return missingStringProps(node.Props, "agent_id", "session_id", "timestamp")
	case agentSessionKind:
		return missingStringProps(node.Props, "agent_id", "started_at", "ended_at", "repo_root")
	default:
		return nil
	}
}

func missingStringProps(props map[string]any, fields ...string) []string {
	missing := make([]string, 0, len(fields))
	for _, field := range fields {
		if value, ok := props[field].(string); ok && strings.TrimSpace(value) != "" {
			continue
		}
		missing = append(missing, field)
	}
	return missing
}

func mergedProperties(object map[string]any) map[string]any {
	props := map[string]any{}

	if nested, ok := object["properties"].(map[string]any); ok {
		for key, value := range nested {
			props[key] = value
		}
	}
	if nested, ok := object["props"].(map[string]any); ok {
		for key, value := range nested {
			props[key] = value
		}
	}

	for key, value := range object {
		switch key {
		case "id", "kind", "type", "properties", "props":
			continue
		default:
			props[key] = value
		}
	}

	if len(props) == 0 {
		return nil
	}

	return props
}

func requiredString(object map[string]any, key string) (string, bool) {
	value, ok := object[key]
	if !ok {
		return "", false
	}
	str, ok := value.(string)
	if !ok {
		return "", false
	}
	str = strings.TrimSpace(str)
	return str, str != ""
}

func firstNonEmptyString(object map[string]any, keys ...string) (string, bool) {
	for _, key := range keys {
		if value, ok := requiredString(object, key); ok {
			return value, true
		}
	}
	return "", false
}

func firstStringFromMaps(objects []map[string]any, key string) string {
	for _, object := range objects {
		if object == nil {
			continue
		}
		if value, ok := requiredString(object, key); ok {
			return value
		}
	}
	return ""
}

func firstStringSlice(key string, objects ...map[string]any) []string {
	for _, object := range objects {
		if object == nil || key == "" {
			continue
		}
		if values, ok := stringSlice(object[key]); ok {
			return values
		}
	}
	return nil
}

func stringSlice(value any) ([]string, bool) {
	rawValues, ok := value.([]any)
	if !ok {
		return nil, false
	}
	values := make([]string, 0, len(rawValues))
	for _, item := range rawValues {
		str, ok := item.(string)
		if !ok {
			return nil, false
		}
		str = strings.TrimSpace(str)
		if str == "" {
			continue
		}
		values = append(values, str)
	}
	if len(values) == 0 {
		return nil, true
	}
	return values, true
}

func edgeKey(from, kind, to string) string {
	return from + "\x00" + kind + "\x00" + to
}

func edgeRefLess(left, right EdgeRef) bool {
	if left.From != right.From {
		return left.From < right.From
	}
	if left.Kind != right.Kind {
		return left.Kind < right.Kind
	}
	return left.To < right.To
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	cloned := make([]string, len(values))
	copy(cloned, values)
	return cloned
}

func cloneProps(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}

	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}

	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprint(typed)
	}
}

func stringSliceValue(value any) []string {
	switch typed := value.(type) {
	case nil:
		return nil
	case []string:
		return cloneStrings(typed)
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			str, ok := item.(string)
			if !ok || strings.TrimSpace(str) == "" {
				continue
			}
			values = append(values, str)
		}
		return values
	default:
		return nil
	}
}

func jsonMapValue(value any) map[string]any {
	payload := strings.TrimSpace(stringValue(value))
	if payload == "" {
		return nil
	}

	decoded := map[string]any{}
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		return map[string]any{
			"_decode_error": err.Error(),
			"_raw":          payload,
		}
	}

	if len(decoded) == 0 {
		return nil
	}
	return decoded
}

func intValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int8:
		return int(typed)
	case int16:
		return int(typed)
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case uint:
		return int(typed)
	case uint8:
		return int(typed)
	case uint16:
		return int(typed)
	case uint32:
		return int(typed)
	case uint64:
		return int(typed)
	case float32:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}
