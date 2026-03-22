package kuzu

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	kuzudb "github.com/kuzudb/go-kuzu"

	"github.com/guillaume-galp/cge/internal/infra/repo"
)

type DiffError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

func (e *DiffError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

type RevisionDiffMetadata struct {
	Requested        string `json:"requested"`
	ID               string `json:"id"`
	Anchor           string `json:"anchor"`
	Reason           string `json:"reason,omitempty"`
	NodeCount        int    `json:"node_count"`
	EdgeCount        int    `json:"edge_count"`
	CreatedAt        string `json:"created_at,omitempty"`
	CreatedBy        string `json:"created_by,omitempty"`
	CreatedSessionID string `json:"created_session_id,omitempty"`
}

type DiffCategorySummary struct {
	AddedCount    int `json:"added_count"`
	UpdatedCount  int `json:"updated_count"`
	RemovedCount  int `json:"removed_count"`
	RetaggedCount int `json:"retagged_count"`
}

type DiffSummary struct {
	Entities      DiffCategorySummary `json:"entities"`
	Relationships DiffCategorySummary `json:"relationships"`
}

type FieldChange struct {
	From    any      `json:"from,omitempty"`
	To      any      `json:"to,omitempty"`
	Added   []string `json:"added,omitempty"`
	Removed []string `json:"removed,omitempty"`
}

type EntityChange struct {
	Before        EntityRecord           `json:"before"`
	After         EntityRecord           `json:"after"`
	ChangedFields []string               `json:"changed_fields"`
	Changes       map[string]FieldChange `json:"changes"`
}

type RelationshipChange struct {
	Before        RelationRecord         `json:"before"`
	After         RelationRecord         `json:"after"`
	ChangedFields []string               `json:"changed_fields"`
	Changes       map[string]FieldChange `json:"changes"`
}

type EntityDiffSet struct {
	Added    []EntityRecord `json:"added"`
	Updated  []EntityChange `json:"updated"`
	Removed  []EntityRecord `json:"removed"`
	Retagged []EntityChange `json:"retagged"`
}

type RelationshipDiffSet struct {
	Added    []RelationRecord     `json:"added"`
	Updated  []RelationshipChange `json:"updated"`
	Removed  []RelationRecord     `json:"removed"`
	Retagged []RelationshipChange `json:"retagged"`
}

type GraphDiff struct {
	From          RevisionDiffMetadata `json:"from"`
	To            RevisionDiffMetadata `json:"to"`
	Summary       DiffSummary          `json:"summary"`
	Entities      EntityDiffSet        `json:"entities"`
	Relationships RelationshipDiffSet  `json:"relationships"`
}

type revisionRecord struct {
	RevisionDiffMetadata
}

func (s *Store) Diff(_ context.Context, workspace repo.Workspace, fromRef, toRef string) (GraphDiff, error) {
	dbPath := filepath.Join(workspace.WorkspacePath, "kuzu", StoreFileName)
	if _, err := os.Stat(dbPath); err != nil {
		if os.IsNotExist(err) {
			missing := strings.TrimSpace(fromRef)
			flag := "from"
			if missing == "" {
				missing = strings.TrimSpace(toRef)
				flag = "to"
			}
			return GraphDiff{}, &DiffError{
				Code:    "revision_anchor_not_found",
				Message: "requested revision anchor does not exist",
				Details: map[string]any{
					"flag":   flag,
					"anchor": missing,
				},
			}
		}
		return GraphDiff{}, fmt.Errorf("inspect kuzu database: %w", err)
	}

	db, err := kuzudb.OpenDatabase(dbPath, kuzudb.DefaultSystemConfig())
	if err != nil {
		return GraphDiff{}, fmt.Errorf("open kuzu database: %w", err)
	}
	defer db.Close()

	conn, err := kuzudb.OpenConnection(db)
	if err != nil {
		return GraphDiff{}, fmt.Errorf("open kuzu connection: %w", err)
	}
	defer conn.Close()

	if err := ensureSchema(conn); err != nil {
		return GraphDiff{}, err
	}

	fromRevision, err := resolveRevision(conn, strings.TrimSpace(fromRef), "from")
	if err != nil {
		return GraphDiff{}, err
	}
	toRevision, err := resolveRevision(conn, strings.TrimSpace(toRef), "to")
	if err != nil {
		return GraphDiff{}, err
	}

	fromSnapshot, err := readRevisionSnapshot(conn, fromRevision)
	if err != nil {
		return GraphDiff{}, err
	}
	toSnapshot, err := readRevisionSnapshot(conn, toRevision)
	if err != nil {
		return GraphDiff{}, err
	}

	diff := compareSnapshots(fromRevision, toRevision, fromSnapshot, toSnapshot)
	return diff, nil
}

func resolveRevision(conn *kuzudb.Connection, reference, flag string) (revisionRecord, error) {
	result, err := conn.Query(fmt.Sprintf(`MATCH (e:%s)
WHERE e.kind = '%s'
RETURN e.id, e.summary, e.created_at, e.created_by, e.created_session_id, e.props_json
ORDER BY e.created_at, e.id;`, entityTableName, graphRevisionKind))
	if err != nil {
		return revisionRecord{}, fmt.Errorf("query revision anchors: %w", err)
	}
	defer result.Close()

	var anchorMatch *revisionRecord
	for result.HasNext() {
		tuple, err := result.Next()
		if err != nil {
			return revisionRecord{}, fmt.Errorf("read revision anchor tuple: %w", err)
		}
		values, err := tuple.GetAsSlice()
		if err != nil {
			return revisionRecord{}, fmt.Errorf("decode revision anchor tuple: %w", err)
		}

		props := jsonMapValue(values[5])
		revision := revisionRecord{
			RevisionDiffMetadata: RevisionDiffMetadata{
				Requested:        reference,
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
		if revision.ID == reference {
			return revision, nil
		}
		if revision.Anchor == reference && anchorMatch == nil {
			matched := revision
			anchorMatch = &matched
		}
	}
	if anchorMatch != nil {
		return *anchorMatch, nil
	}

	return revisionRecord{}, &DiffError{
		Code:    "revision_anchor_not_found",
		Message: "requested revision anchor does not exist",
		Details: map[string]any{
			"flag":   flag,
			"anchor": reference,
		},
	}
}

func readRevisionSnapshot(conn *kuzudb.Connection, revision revisionRecord) (graphSnapshot, error) {
	snapshot := graphSnapshot{
		Nodes: []graphSnapshotNode{},
		Edges: []graphSnapshotEdge{},
	}

	nodeResult, err := executePrepared(conn,
		fmt.Sprintf(`MATCH (s:%s)
WHERE s.revision_id = $revision_id
RETURN s.entity_id, s.kind, s.title, s.summary, s.content, s.repo_path, s.language, s.tags, s.props_json, s.created_at, s.updated_at, s.created_by, s.updated_by, s.created_session_id, s.updated_session_id
ORDER BY s.entity_id;`, revisionNodeSnapshotTableName),
		map[string]any{"revision_id": revision.ID},
	)
	if err != nil {
		return graphSnapshot{}, revisionSnapshotUnavailableError(revision, err)
	}
	defer nodeResult.Close()

	for nodeResult.HasNext() {
		tuple, err := nodeResult.Next()
		if err != nil {
			return graphSnapshot{}, fmt.Errorf("read revision node snapshot tuple: %w", err)
		}
		values, err := tuple.GetAsSlice()
		if err != nil {
			return graphSnapshot{}, fmt.Errorf("decode revision node snapshot tuple: %w", err)
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

	edgeResult, err := executePrepared(conn,
		fmt.Sprintf(`MATCH (s:%s)
WHERE s.revision_id = $revision_id
RETURN s.from_id, s.to_id, s.kind, s.props_json, s.created_at, s.updated_at, s.created_by, s.updated_by, s.created_session_id, s.updated_session_id
ORDER BY s.from_id, s.kind, s.to_id;`, revisionEdgeSnapshotTableName),
		map[string]any{"revision_id": revision.ID},
	)
	if err != nil {
		return graphSnapshot{}, revisionSnapshotUnavailableError(revision, err)
	}
	defer edgeResult.Close()

	for edgeResult.HasNext() {
		tuple, err := edgeResult.Next()
		if err != nil {
			return graphSnapshot{}, fmt.Errorf("read revision relationship snapshot tuple: %w", err)
		}
		values, err := tuple.GetAsSlice()
		if err != nil {
			return graphSnapshot{}, fmt.Errorf("decode revision relationship snapshot tuple: %w", err)
		}
		snapshot.Edges = append(snapshot.Edges, graphSnapshotEdge{
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
		})
	}

	if len(snapshot.Nodes) != revision.NodeCount || len(snapshot.Edges) != revision.EdgeCount {
		return graphSnapshot{}, &DiffError{
			Code:    "revision_snapshot_unavailable",
			Message: "requested revision anchor does not have a comparable snapshot",
			Details: map[string]any{
				"anchor":              revision.Requested,
				"revision_id":         revision.ID,
				"expected_node_count": revision.NodeCount,
				"actual_node_count":   len(snapshot.Nodes),
				"expected_edge_count": revision.EdgeCount,
				"actual_edge_count":   len(snapshot.Edges),
			},
		}
	}

	return snapshot, nil
}

func revisionSnapshotUnavailableError(revision revisionRecord, err error) error {
	return &DiffError{
		Code:    "revision_snapshot_unavailable",
		Message: "requested revision anchor does not have a comparable snapshot",
		Details: map[string]any{
			"anchor":      revision.Requested,
			"revision_id": revision.ID,
			"reason":      err.Error(),
		},
	}
}

func compareSnapshots(fromRevision, toRevision revisionRecord, fromSnapshot, toSnapshot graphSnapshot) GraphDiff {
	diff := GraphDiff{
		From: fromRevision.RevisionDiffMetadata,
		To:   toRevision.RevisionDiffMetadata,
		Entities: EntityDiffSet{
			Added:    []EntityRecord{},
			Updated:  []EntityChange{},
			Removed:  []EntityRecord{},
			Retagged: []EntityChange{},
		},
		Relationships: RelationshipDiffSet{
			Added:    []RelationRecord{},
			Updated:  []RelationshipChange{},
			Removed:  []RelationRecord{},
			Retagged: []RelationshipChange{},
		},
	}

	fromNodes := make(map[string]graphSnapshotNode, len(fromSnapshot.Nodes))
	for _, node := range fromSnapshot.Nodes {
		fromNodes[node.ID] = node
	}
	toNodes := make(map[string]graphSnapshotNode, len(toSnapshot.Nodes))
	for _, node := range toSnapshot.Nodes {
		toNodes[node.ID] = node
	}

	for _, node := range fromSnapshot.Nodes {
		other, ok := toNodes[node.ID]
		if !ok {
			diff.Entities.Removed = append(diff.Entities.Removed, entityRecordFromSnapshot(node))
			continue
		}
		changes, fields, retagged := diffNode(node, other)
		if len(fields) == 0 {
			continue
		}
		change := EntityChange{
			Before:        entityRecordFromSnapshot(node),
			After:         entityRecordFromSnapshot(other),
			ChangedFields: fields,
			Changes:       changes,
		}
		if retagged {
			diff.Entities.Retagged = append(diff.Entities.Retagged, change)
			continue
		}
		diff.Entities.Updated = append(diff.Entities.Updated, change)
	}
	for _, node := range toSnapshot.Nodes {
		if _, ok := fromNodes[node.ID]; ok {
			continue
		}
		diff.Entities.Added = append(diff.Entities.Added, entityRecordFromSnapshot(node))
	}

	fromEdges := make(map[string]graphSnapshotEdge, len(fromSnapshot.Edges))
	for _, edge := range fromSnapshot.Edges {
		fromEdges[edgeKey(edge.From, edge.Kind, edge.To)] = edge
	}
	toEdges := make(map[string]graphSnapshotEdge, len(toSnapshot.Edges))
	for _, edge := range toSnapshot.Edges {
		toEdges[edgeKey(edge.From, edge.Kind, edge.To)] = edge
	}

	removedEdges := make([]graphSnapshotEdge, 0)
	for _, edge := range fromSnapshot.Edges {
		other, ok := toEdges[edgeKey(edge.From, edge.Kind, edge.To)]
		if !ok {
			removedEdges = append(removedEdges, edge)
			continue
		}
		changes, fields := diffRelation(edge, other)
		if len(fields) == 0 {
			continue
		}
		diff.Relationships.Updated = append(diff.Relationships.Updated, RelationshipChange{
			Before:        relationRecordFromSnapshot(edge),
			After:         relationRecordFromSnapshot(other),
			ChangedFields: fields,
			Changes:       changes,
		})
	}
	addedEdges := make([]graphSnapshotEdge, 0)
	for _, edge := range toSnapshot.Edges {
		if _, ok := fromEdges[edgeKey(edge.From, edge.Kind, edge.To)]; ok {
			continue
		}
		addedEdges = append(addedEdges, edge)
	}

	retaggedRelationships, unmatchedRemoved, unmatchedAdded := pairRetaggedRelationships(removedEdges, addedEdges)
	diff.Relationships.Retagged = append(diff.Relationships.Retagged, retaggedRelationships...)
	for _, edge := range unmatchedRemoved {
		diff.Relationships.Removed = append(diff.Relationships.Removed, relationRecordFromSnapshot(edge))
	}
	for _, edge := range unmatchedAdded {
		diff.Relationships.Added = append(diff.Relationships.Added, relationRecordFromSnapshot(edge))
	}

	sortEntityRecords(diff.Entities.Added)
	sortEntityRecords(diff.Entities.Removed)
	sort.Slice(diff.Entities.Updated, func(i, j int) bool {
		return diff.Entities.Updated[i].After.ID < diff.Entities.Updated[j].After.ID
	})
	sort.Slice(diff.Entities.Retagged, func(i, j int) bool {
		return diff.Entities.Retagged[i].After.ID < diff.Entities.Retagged[j].After.ID
	})
	sortRelationRecords(diff.Relationships.Added)
	sortRelationRecords(diff.Relationships.Removed)
	sort.Slice(diff.Relationships.Updated, func(i, j int) bool {
		return edgeRefLess(edgeRefFromRelation(diff.Relationships.Updated[i].After), edgeRefFromRelation(diff.Relationships.Updated[j].After))
	})
	sort.Slice(diff.Relationships.Retagged, func(i, j int) bool {
		left := diff.Relationships.Retagged[i]
		right := diff.Relationships.Retagged[j]
		if left.After.From != right.After.From {
			return left.After.From < right.After.From
		}
		if left.After.To != right.After.To {
			return left.After.To < right.After.To
		}
		if left.Before.Kind != right.Before.Kind {
			return left.Before.Kind < right.Before.Kind
		}
		return left.After.Kind < right.After.Kind
	})

	diff.Summary = DiffSummary{
		Entities: DiffCategorySummary{
			AddedCount:    len(diff.Entities.Added),
			UpdatedCount:  len(diff.Entities.Updated),
			RemovedCount:  len(diff.Entities.Removed),
			RetaggedCount: len(diff.Entities.Retagged),
		},
		Relationships: DiffCategorySummary{
			AddedCount:    len(diff.Relationships.Added),
			UpdatedCount:  len(diff.Relationships.Updated),
			RemovedCount:  len(diff.Relationships.Removed),
			RetaggedCount: len(diff.Relationships.Retagged),
		},
	}

	return diff
}

func diffNode(before, after graphSnapshotNode) (map[string]FieldChange, []string, bool) {
	changes := map[string]FieldChange{}
	fields := []string{}
	retagged := false

	if before.Kind != after.Kind {
		changes["kind"] = FieldChange{From: before.Kind, To: after.Kind}
		fields = append(fields, "kind")
		retagged = true
	}
	if before.Title != after.Title {
		changes["title"] = FieldChange{From: before.Title, To: after.Title}
		fields = append(fields, "title")
	}
	if before.Summary != after.Summary {
		changes["summary"] = FieldChange{From: before.Summary, To: after.Summary}
		fields = append(fields, "summary")
	}
	if before.Content != after.Content {
		changes["content"] = FieldChange{From: before.Content, To: after.Content}
		fields = append(fields, "content")
	}
	if before.RepoPath != after.RepoPath {
		changes["repo_path"] = FieldChange{From: before.RepoPath, To: after.RepoPath}
		fields = append(fields, "repo_path")
	}
	if before.Language != after.Language {
		changes["language"] = FieldChange{From: before.Language, To: after.Language}
		fields = append(fields, "language")
	}
	beforeTags := normalizeTags(before.Tags)
	afterTags := normalizeTags(after.Tags)
	if !slices.Equal(beforeTags, afterTags) {
		changes["tags"] = FieldChange{
			From:    beforeTags,
			To:      afterTags,
			Added:   stringsOnly(difference(afterTags, beforeTags)),
			Removed: stringsOnly(difference(beforeTags, afterTags)),
		}
		fields = append(fields, "tags")
		retagged = true
	}
	if !jsonValueEqual(before.Props, after.Props) {
		changes["props"] = FieldChange{From: cloneProps(before.Props), To: cloneProps(after.Props)}
		fields = append(fields, "props")
	}

	return changes, fields, retagged
}

func diffRelation(before, after graphSnapshotEdge) (map[string]FieldChange, []string) {
	changes := map[string]FieldChange{}
	fields := []string{}
	if !jsonValueEqual(before.Props, after.Props) {
		changes["props"] = FieldChange{From: cloneProps(before.Props), To: cloneProps(after.Props)}
		fields = append(fields, "props")
	}
	return changes, fields
}

func pairRetaggedRelationships(removed, added []graphSnapshotEdge) ([]RelationshipChange, []graphSnapshotEdge, []graphSnapshotEdge) {
	removedByEndpoints := map[string][]graphSnapshotEdge{}
	for _, edge := range removed {
		key := edgeEndpointsKey(edge.From, edge.To)
		removedByEndpoints[key] = append(removedByEndpoints[key], edge)
	}

	addedByEndpoints := map[string][]graphSnapshotEdge{}
	for _, edge := range added {
		key := edgeEndpointsKey(edge.From, edge.To)
		addedByEndpoints[key] = append(addedByEndpoints[key], edge)
	}

	retagged := make([]RelationshipChange, 0)
	unmatchedRemoved := make([]graphSnapshotEdge, 0, len(removed))
	unmatchedAdded := make([]graphSnapshotEdge, 0, len(added))

	endpointKeys := make([]string, 0, len(removedByEndpoints)+len(addedByEndpoints))
	seenEndpointKeys := map[string]struct{}{}
	for key := range removedByEndpoints {
		endpointKeys = append(endpointKeys, key)
		seenEndpointKeys[key] = struct{}{}
	}
	for key := range addedByEndpoints {
		if _, ok := seenEndpointKeys[key]; ok {
			continue
		}
		endpointKeys = append(endpointKeys, key)
	}
	sort.Strings(endpointKeys)

	for _, key := range endpointKeys {
		removedGroup := removedByEndpoints[key]
		addedGroup := addedByEndpoints[key]
		if len(removedGroup) != 1 || len(addedGroup) != 1 {
			unmatchedRemoved = append(unmatchedRemoved, removedGroup...)
			unmatchedAdded = append(unmatchedAdded, addedGroup...)
			continue
		}

		before := removedGroup[0]
		after := addedGroup[0]
		changes := map[string]FieldChange{
			"kind": {From: before.Kind, To: after.Kind},
		}
		fields := []string{"kind"}
		if !jsonValueEqual(before.Props, after.Props) {
			changes["props"] = FieldChange{From: cloneProps(before.Props), To: cloneProps(after.Props)}
			fields = append(fields, "props")
		}
		retagged = append(retagged, RelationshipChange{
			Before:        relationRecordFromSnapshot(before),
			After:         relationRecordFromSnapshot(after),
			ChangedFields: fields,
			Changes:       changes,
		})
	}

	return retagged, unmatchedRemoved, unmatchedAdded
}

func entityRecordFromSnapshot(node graphSnapshotNode) EntityRecord {
	return EntityRecord{
		ID:               node.ID,
		Kind:             node.Kind,
		Title:            node.Title,
		Summary:          node.Summary,
		Content:          node.Content,
		RepoPath:         node.RepoPath,
		Language:         node.Language,
		Tags:             cloneStrings(node.Tags),
		Props:            cloneProps(node.Props),
		CreatedAt:        node.CreatedAt,
		UpdatedAt:        node.UpdatedAt,
		CreatedBy:        node.CreatedBy,
		UpdatedBy:        node.UpdatedBy,
		CreatedSessionID: node.CreatedSessionID,
		UpdatedSessionID: node.UpdatedSessionID,
	}
}

func relationRecordFromSnapshot(edge graphSnapshotEdge) RelationRecord {
	return RelationRecord{
		From:             edge.From,
		To:               edge.To,
		Kind:             edge.Kind,
		Props:            cloneProps(edge.Props),
		CreatedAt:        edge.CreatedAt,
		UpdatedAt:        edge.UpdatedAt,
		CreatedBy:        edge.CreatedBy,
		UpdatedBy:        edge.UpdatedBy,
		CreatedSessionID: edge.CreatedSessionID,
		UpdatedSessionID: edge.UpdatedSessionID,
	}
}

func sortEntityRecords(values []EntityRecord) {
	sort.Slice(values, func(i, j int) bool {
		return values[i].ID < values[j].ID
	})
}

func sortRelationRecords(values []RelationRecord) {
	sort.Slice(values, func(i, j int) bool {
		return edgeRefLess(edgeRefFromRelation(values[i]), edgeRefFromRelation(values[j]))
	})
}

func edgeRefFromRelation(edge RelationRecord) EdgeRef {
	return EdgeRef{From: edge.From, Kind: edge.Kind, To: edge.To}
}

func edgeRefFromSnapshot(edge graphSnapshotEdge) EdgeRef {
	return EdgeRef{From: edge.From, Kind: edge.Kind, To: edge.To}
}

func edgeEndpointsKey(from, to string) string {
	return from + "|" + to
}

func normalizeTags(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) == 0 {
		return nil
	}
	sort.Strings(normalized)
	return normalized
}

func difference(left, right []string) []string {
	if len(left) == 0 {
		return nil
	}
	rightSet := map[string]struct{}{}
	for _, value := range right {
		rightSet[value] = struct{}{}
	}
	diff := make([]string, 0, len(left))
	for _, value := range left {
		if _, ok := rightSet[value]; ok {
			continue
		}
		diff = append(diff, value)
	}
	if len(diff) == 0 {
		return nil
	}
	return diff
}

func stringsOnly(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	return values
}

func jsonValueEqual(left, right any) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return fmt.Sprintf("%v", left) == fmt.Sprintf("%v", right)
	}
	return string(leftJSON) == string(rightJSON)
}

func stringProp(props map[string]any, key string) string {
	return stringValue(props[key])
}

func intProp(props map[string]any, key string) int {
	return intValue(props[key])
}

func revisionReason(props map[string]any, fallback any) string {
	if reason := strings.TrimSpace(stringProp(props, "reason")); reason != "" {
		return reason
	}
	return strings.TrimSpace(stringValue(fallback))
}
