package kuzu

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	kuzudb "github.com/kuzudb/go-kuzu"

	"github.com/guillaume-galp/cge/internal/infra/repo"
)

type Graph struct {
	Nodes []EntityRecord   `json:"nodes"`
	Edges []RelationRecord `json:"edges"`
}

type EntityRecord struct {
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

type RelationRecord struct {
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

func (s *Store) ReadGraph(_ context.Context, workspace repo.Workspace) (Graph, error) {
	dbPath := filepath.Join(workspace.WorkspacePath, "kuzu", StoreFileName)
	if _, err := os.Stat(dbPath); err != nil {
		if os.IsNotExist(err) {
			return Graph{Nodes: []EntityRecord{}, Edges: []RelationRecord{}}, nil
		}
		return Graph{}, fmt.Errorf("inspect kuzu database: %w", err)
	}

	config := kuzudb.DefaultSystemConfig()
	config.ReadOnly = true

	db, err := kuzudb.OpenDatabase(dbPath, config)
	if err != nil {
		return Graph{}, fmt.Errorf("open kuzu database: %w", err)
	}
	defer db.Close()

	conn, err := kuzudb.OpenConnection(db)
	if err != nil {
		return Graph{}, fmt.Errorf("open kuzu connection: %w", err)
	}
	defer conn.Close()

	snapshot, err := readComparableSnapshot(conn)
	if err != nil {
		return Graph{}, fmt.Errorf("read graph snapshot: %w", err)
	}

	graph := Graph{
		Nodes: make([]EntityRecord, 0, len(snapshot.Nodes)),
		Edges: make([]RelationRecord, 0, len(snapshot.Edges)),
	}

	for _, node := range snapshot.Nodes {
		graph.Nodes = append(graph.Nodes, EntityRecord{
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
		})
	}

	for _, edge := range snapshot.Edges {
		graph.Edges = append(graph.Edges, RelationRecord{
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
		})
	}

	return graph, nil
}
