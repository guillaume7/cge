package kuzu

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	kuzudb "github.com/kuzudb/go-kuzu"

	"github.com/guillaume-galp/cge/internal/infra/repo"
)

type GraphStats struct {
	Nodes         int `json:"nodes"`
	Relationships int `json:"relationships"`
}

func (s *Store) Stats(_ context.Context, workspace repo.Workspace) (GraphStats, error) {
	dbPath := filepath.Join(workspace.WorkspacePath, "kuzu", StoreFileName)
	if _, err := os.Stat(dbPath); err != nil {
		if os.IsNotExist(err) {
			return GraphStats{}, nil
		}
		return GraphStats{}, statsPersistenceError(err)
	}

	config := kuzudb.DefaultSystemConfig()
	config.ReadOnly = true

	db, err := kuzudb.OpenDatabase(dbPath, config)
	if err != nil {
		return GraphStats{}, statsPersistenceError(fmt.Errorf("open kuzu database: %w", err))
	}
	defer db.Close()

	conn, err := kuzudb.OpenConnection(db)
	if err != nil {
		return GraphStats{}, statsPersistenceError(fmt.Errorf("open kuzu connection: %w", err))
	}
	defer conn.Close()

	nodeCount, err := queryStatsCount(conn, fmt.Sprintf(`MATCH (e:%s)
WHERE e.kind <> '%s'
RETURN count(e);`, entityTableName, graphRevisionKind))
	if err != nil {
		return GraphStats{}, statsPersistenceError(fmt.Errorf("count graph nodes: %w", err))
	}

	relationshipCount, err := queryStatsCount(conn, fmt.Sprintf(`MATCH (from:%s)-[r:%s]->(to:%s)
WHERE from.kind <> '%s' AND to.kind <> '%s'
RETURN count(r);`, entityTableName, relationTableName, entityTableName, graphRevisionKind, graphRevisionKind))
	if err != nil {
		return GraphStats{}, statsPersistenceError(fmt.Errorf("count graph relationships: %w", err))
	}

	return GraphStats{
		Nodes:         nodeCount,
		Relationships: relationshipCount,
	}, nil
}

func queryStatsCount(conn *kuzudb.Connection, query string) (int, error) {
	result, err := conn.Query(query)
	if err != nil {
		return 0, err
	}
	defer result.Close()

	if !result.HasNext() {
		return 0, nil
	}

	tuple, err := result.Next()
	if err != nil {
		return 0, fmt.Errorf("read count tuple: %w", err)
	}

	values, err := tuple.GetAsSlice()
	if err != nil {
		return 0, fmt.Errorf("decode count tuple: %w", err)
	}
	if len(values) == 0 {
		return 0, nil
	}

	return intValue(values[0]), nil
}

func statsPersistenceError(err error) error {
	return &PersistenceError{
		Code:    "stats_unavailable",
		Message: "graph stats could not be read",
		Details: map[string]any{
			"reason": err.Error(),
		},
	}
}
