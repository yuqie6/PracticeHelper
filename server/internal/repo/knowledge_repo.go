package repo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"practicehelper/server/internal/domain"
)

func (s *Store) EnsureKnowledgeSeeds(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin ensure knowledge seeds: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	topics, err := s.listSeedTopics(ctx, tx)
	if err != nil {
		return err
	}
	for _, topic := range topics {
		if _, err := s.ensureKnowledgeNodeTx(ctx, tx, domain.KnowledgeNode{
			ScopeType:   domain.MemoryScopeGlobal,
			Label:       topic,
			NodeType:    domain.KnowledgeNodeTypeTopic,
			Proficiency: 0,
			Confidence:  0.5,
		}); err != nil {
			return err
		}
	}

	profile, err := s.GetUserProfile(ctx)
	if err != nil {
		return err
	}
	if profile != nil {
		for _, tech := range profile.TechStacks {
			label := normalizeTopicLabel(tech)
			if label == "" {
				continue
			}
			if _, err := s.ensureKnowledgeNodeTx(ctx, tx, domain.KnowledgeNode{
				ScopeType:   domain.MemoryScopeGlobal,
				Label:       label,
				NodeType:    domain.KnowledgeNodeTypeTopic,
				Proficiency: 0,
				Confidence:  0.45,
			}); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (s *Store) GetKnowledgeSubgraph(
	ctx context.Context,
	topic string,
	projectID string,
	limit int,
) (*domain.KnowledgeSubgraph, error) {
	if limit <= 0 {
		limit = 8
	}
	topic = normalizeTopicLabel(topic)
	if topic == "" {
		return &domain.KnowledgeSubgraph{}, nil
	}

	args := []any{topic}
	query := `
		SELECT id, scope_type, scope_id, parent_id, label, node_type, proficiency, confidence, hit_count, last_assessed_at, created_at, updated_at
		FROM knowledge_nodes
		WHERE lower(label) = ? AND parent_id = '' AND (
			scope_type = 'global'
	`
	if strings.TrimSpace(projectID) != "" {
		query += ` OR (scope_type = 'project' AND scope_id = ?)`
		args = append(args, projectID)
	}
	query += `)
		ORDER BY CASE WHEN scope_type = 'project' THEN 0 ELSE 1 END, confidence DESC, proficiency DESC, updated_at DESC`

	rootRows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query knowledge root nodes: %w", err)
	}
	defer func() { _ = rootRows.Close() }()

	nodes := make([]domain.KnowledgeNode, 0, limit)
	rootIDs := make([]string, 0, 2)
	for rootRows.Next() {
		node, err := scanKnowledgeNode(rootRows)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, *node)
		rootIDs = append(rootIDs, node.ID)
	}
	if err := rootRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate knowledge root nodes: %w", err)
	}
	if len(rootIDs) == 0 {
		return &domain.KnowledgeSubgraph{}, nil
	}

	if len(nodes) < limit {
		children, err := s.listKnowledgeChildren(ctx, rootIDs, limit-len(nodes))
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, children...)
	}

	if len(nodes) < limit {
		neighbors, err := s.listKnowledgeNeighborTopics(ctx, rootIDs, limit-len(nodes), nodes)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, neighbors...)
	}

	nodeIDs := make([]string, 0, len(nodes))
	for _, node := range nodes {
		nodeIDs = append(nodeIDs, node.ID)
	}
	edges, err := s.listKnowledgeEdges(ctx, nodeIDs)
	if err != nil {
		return nil, err
	}

	return &domain.KnowledgeSubgraph{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

func (s *Store) UpsertKnowledgeNodes(
	ctx context.Context,
	sessionID string,
	updates []domain.KnowledgeUpdate,
) error {
	_, err := s.UpsertKnowledgeNodesWithCount(ctx, sessionID, updates)
	return err
}

func (s *Store) UpsertKnowledgeNodesWithCount(
	ctx context.Context,
	sessionID string,
	updates []domain.KnowledgeUpdate,
) (int, error) {
	if len(updates) == 0 {
		return 0, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin upsert knowledge nodes: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	indexEntries := make([]domain.MemoryIndexEntry, 0, len(updates))
	for _, update := range updates {
		nodeID, node, err := s.upsertKnowledgeUpdateTx(ctx, tx, sessionID, update)
		if err != nil {
			return 0, err
		}
		if node == nil {
			continue
		}
		if node.ParentID != "" {
			if err := s.ensureKnowledgeEdgeTx(
				ctx,
				tx,
				node.ParentID,
				nodeID,
				domain.KnowledgeEdgeContains,
			); err != nil {
				return 0, err
			}
		}
		topic, err := s.resolveRootTopicTx(ctx, tx, node)
		if err != nil {
			return 0, err
		}
		indexEntries = append(indexEntries, domain.MemoryIndexEntry{
			MemoryType: "knowledge_node",
			ScopeType:  node.ScopeType,
			ScopeID:    node.ScopeID,
			Topic:      topic,
			SessionID:  sessionID,
			Summary:    node.Label,
			Salience:   0.55,
			Confidence: node.Confidence,
			Freshness:  1.0,
			RefTable:   "knowledge_nodes",
			RefID:      nodeID,
		})
	}

	if err := s.upsertMemoryIndexEntries(ctx, tx, indexEntries); err != nil {
		return 0, err
	}

	return len(indexEntries), tx.Commit()
}

func (s *Store) EnsureKnowledgePrerequisiteEdge(
	ctx context.Context,
	prerequisiteTopic string,
	dependentTopic string,
) error {
	prerequisiteTopic = normalizeTopicLabel(prerequisiteTopic)
	dependentTopic = normalizeTopicLabel(dependentTopic)
	if prerequisiteTopic == "" || dependentTopic == "" || prerequisiteTopic == dependentTopic {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin ensure prerequisite edge: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	sourceID, err := s.ensureKnowledgeNodeTx(ctx, tx, domain.KnowledgeNode{
		ScopeType:  domain.MemoryScopeGlobal,
		Label:      prerequisiteTopic,
		NodeType:   domain.KnowledgeNodeTypeTopic,
		Confidence: 0.5,
	})
	if err != nil {
		return err
	}
	targetID, err := s.ensureKnowledgeNodeTx(ctx, tx, domain.KnowledgeNode{
		ScopeType:  domain.MemoryScopeGlobal,
		Label:      dependentTopic,
		NodeType:   domain.KnowledgeNodeTypeTopic,
		Confidence: 0.5,
	})
	if err != nil {
		return err
	}
	if err := s.ensureKnowledgeEdgeTx(
		ctx,
		tx,
		sourceID,
		targetID,
		domain.KnowledgeEdgePrerequisite,
	); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) listSeedTopics(ctx context.Context, tx *sql.Tx) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT DISTINCT topic
		FROM question_templates
		WHERE mode = ? AND topic <> '' AND topic <> ?
		ORDER BY topic ASC
	`, domain.ModeBasics, domain.BasicsTopicMixed)
	if err != nil {
		return nil, fmt.Errorf("query seed topics: %w", err)
	}
	defer func() { _ = rows.Close() }()

	topics := make([]string, 0)
	for rows.Next() {
		var topic string
		if err := rows.Scan(&topic); err != nil {
			return nil, fmt.Errorf("scan seed topic: %w", err)
		}
		topics = append(topics, normalizeTopicLabel(topic))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate seed topics: %w", err)
	}

	return topics, nil
}

func (s *Store) listKnowledgeChildren(
	ctx context.Context,
	parentIDs []string,
	limit int,
) ([]domain.KnowledgeNode, error) {
	if len(parentIDs) == 0 || limit <= 0 {
		return nil, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(parentIDs)), ",")
	args := make([]any, 0, len(parentIDs)+1)
	for _, id := range parentIDs {
		args = append(args, id)
	}
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, scope_type, scope_id, parent_id, label, node_type, proficiency, confidence, hit_count, last_assessed_at, created_at, updated_at
		FROM knowledge_nodes
		WHERE parent_id IN (%s)
		ORDER BY confidence DESC, proficiency DESC, hit_count DESC, updated_at DESC
		LIMIT ?
	`, placeholders), args...)
	if err != nil {
		return nil, fmt.Errorf("list knowledge children: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]domain.KnowledgeNode, 0, limit)
	for rows.Next() {
		item, err := scanKnowledgeNode(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate knowledge children: %w", err)
	}

	return items, nil
}

func (s *Store) listKnowledgeNeighborTopics(
	ctx context.Context,
	rootIDs []string,
	limit int,
	existing []domain.KnowledgeNode,
) ([]domain.KnowledgeNode, error) {
	if len(rootIDs) == 0 || limit <= 0 {
		return nil, nil
	}

	seen := make(map[string]struct{}, len(existing))
	for _, node := range existing {
		seen[node.ID] = struct{}{}
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(rootIDs)), ",")
	args := make([]any, 0, len(rootIDs)*3+2)
	for _, id := range rootIDs {
		args = append(args, id)
	}
	for _, id := range rootIDs {
		args = append(args, id)
	}
	for _, id := range rootIDs {
		args = append(args, id)
	}
	args = append(args, domain.KnowledgeEdgePrerequisite, domain.KnowledgeEdgeRelated)

	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT DISTINCT
			CASE
				WHEN source_id IN (%s) THEN target_id
				ELSE source_id
			END AS neighbor_id
		FROM knowledge_edges
		WHERE (source_id IN (%s) OR target_id IN (%s))
		  AND edge_type IN (?, ?)
		ORDER BY created_at ASC
	`, placeholders, placeholders, placeholders), args...)
	if err != nil {
		return nil, fmt.Errorf("list knowledge neighbor topics: %w", err)
	}
	defer func() { _ = rows.Close() }()

	neighborIDs := make([]string, 0, limit)
	for rows.Next() {
		var neighborID string
		if err := rows.Scan(&neighborID); err != nil {
			return nil, fmt.Errorf("scan knowledge neighbor id: %w", err)
		}
		if _, ok := seen[neighborID]; ok {
			continue
		}
		seen[neighborID] = struct{}{}
		neighborIDs = append(neighborIDs, neighborID)
		if len(neighborIDs) == limit {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate knowledge neighbor ids: %w", err)
	}

	if len(neighborIDs) == 0 {
		return nil, nil
	}

	placeholders = strings.TrimRight(strings.Repeat("?,", len(neighborIDs)), ",")
	nodeArgs := make([]any, 0, len(neighborIDs))
	for _, id := range neighborIDs {
		nodeArgs = append(nodeArgs, id)
	}
	nodeRows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, scope_type, scope_id, parent_id, label, node_type, proficiency, confidence, hit_count, last_assessed_at, created_at, updated_at
		FROM knowledge_nodes
		WHERE id IN (%s)
	`, placeholders), nodeArgs...)
	if err != nil {
		return nil, fmt.Errorf("load knowledge neighbor nodes: %w", err)
	}
	defer func() { _ = nodeRows.Close() }()

	byID := make(map[string]domain.KnowledgeNode, len(neighborIDs))
	for nodeRows.Next() {
		item, err := scanKnowledgeNode(nodeRows)
		if err != nil {
			return nil, err
		}
		byID[item.ID] = *item
	}
	if err := nodeRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate knowledge neighbor nodes: %w", err)
	}

	items := make([]domain.KnowledgeNode, 0, len(neighborIDs))
	for _, id := range neighborIDs {
		item, ok := byID[id]
		if !ok {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Store) listKnowledgeEdges(
	ctx context.Context,
	nodeIDs []string,
) ([]domain.KnowledgeEdge, error) {
	if len(nodeIDs) == 0 {
		return nil, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(nodeIDs)), ",")
	args := make([]any, 0, len(nodeIDs)*2)
	for _, id := range nodeIDs {
		args = append(args, id)
	}
	for _, id := range nodeIDs {
		args = append(args, id)
	}

	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT source_id, target_id, edge_type, created_at
		FROM knowledge_edges
		WHERE source_id IN (%s) AND target_id IN (%s)
		ORDER BY created_at ASC
	`, placeholders, placeholders), args...)
	if err != nil {
		return nil, fmt.Errorf("list knowledge edges: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]domain.KnowledgeEdge, 0)
	for rows.Next() {
		item, err := scanKnowledgeEdge(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate knowledge edges: %w", err)
	}

	return items, nil
}

func (s *Store) ensureKnowledgeNodeTx(
	ctx context.Context,
	tx *sql.Tx,
	node domain.KnowledgeNode,
) (string, error) {
	scopeType := normalizeMemoryScope(node.ScopeType)
	label := normalizeTopicLabel(node.Label)
	if label == "" {
		return "", nil
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO knowledge_nodes (
			id, scope_type, scope_id, parent_id, label, node_type,
			proficiency, confidence, hit_count, last_assessed_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(scope_type, scope_id, parent_id, label) DO UPDATE SET
			node_type = excluded.node_type,
			updated_at = excluded.updated_at
	`,
		newID("kn"),
		scopeType,
		node.ScopeID,
		node.ParentID,
		label,
		node.NodeType,
		node.Proficiency,
		node.Confidence,
		node.HitCount,
		toNullableTimeString(node.LastAssessedAt),
		nowUTC(),
		nowUTC(),
	); err != nil {
		return "", fmt.Errorf("ensure knowledge node %s: %w", label, err)
	}

	var nodeID string
	if err := tx.QueryRowContext(ctx, `
		SELECT id
		FROM knowledge_nodes
		WHERE scope_type = ? AND scope_id = ? AND parent_id = ? AND label = ?
	`,
		scopeType,
		node.ScopeID,
		node.ParentID,
		label,
	).Scan(&nodeID); err != nil {
		return "", fmt.Errorf("load ensured knowledge node %s: %w", label, err)
	}

	return nodeID, nil
}

func (s *Store) upsertKnowledgeUpdateTx(
	ctx context.Context,
	tx *sql.Tx,
	sessionID string,
	update domain.KnowledgeUpdate,
) (string, *domain.KnowledgeNode, error) {
	var (
		nodeID string
		node   *domain.KnowledgeNode
		err    error
	)
	if strings.TrimSpace(update.NodeID) != "" {
		node, err = s.getKnowledgeNodeByIDTx(ctx, tx, update.NodeID)
		if err != nil {
			return "", nil, err
		}
		if node == nil {
			return "", nil, nil
		}
	} else {
		nodeID, err = s.ensureKnowledgeNodeTx(ctx, tx, domain.KnowledgeNode{
			ScopeType:   normalizeMemoryScope(update.ScopeType),
			ScopeID:     update.ScopeID,
			ParentID:    update.ParentID,
			Label:       update.Label,
			NodeType:    update.NodeType,
			Proficiency: update.Proficiency,
			Confidence:  maxFloat(update.Confidence, 0.5),
		})
		if err != nil {
			return "", nil, err
		}
		node, err = s.getKnowledgeNodeByIDTx(ctx, tx, nodeID)
		if err != nil {
			return "", nil, err
		}
	}
	if node == nil {
		return "", nil, nil
	}

	assessedAt := nowUTC()
	confidence := node.Confidence
	if update.Confidence > 0 {
		confidence = update.Confidence
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE knowledge_nodes
		SET proficiency = ?,
			confidence = ?,
			hit_count = hit_count + 1,
			last_assessed_at = ?,
			updated_at = ?
		WHERE id = ?
	`,
		update.Proficiency,
		confidence,
		assessedAt,
		assessedAt,
		node.ID,
	); err != nil {
		return "", nil, fmt.Errorf("update knowledge node %s: %w", node.ID, err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO knowledge_snapshots (id, node_id, session_id, proficiency, evidence, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`,
		newID("kgs"),
		node.ID,
		sessionID,
		update.Proficiency,
		update.Evidence,
		assessedAt,
	); err != nil {
		return "", nil, fmt.Errorf("insert knowledge snapshot %s: %w", node.ID, err)
	}

	node.Proficiency = update.Proficiency
	node.Confidence = confidence
	node.HitCount++
	node.LastAssessedAt = parseNullableTime(assessedAt)
	return node.ID, node, nil
}

func (s *Store) getKnowledgeNodeByIDTx(
	ctx context.Context,
	tx *sql.Tx,
	nodeID string,
) (*domain.KnowledgeNode, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT id, scope_type, scope_id, parent_id, label, node_type, proficiency, confidence, hit_count, last_assessed_at, created_at, updated_at
		FROM knowledge_nodes
		WHERE id = ?
	`, nodeID)
	node, err := scanKnowledgeNode(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get knowledge node %s: %w", nodeID, err)
	}
	return node, nil
}

func (s *Store) ensureKnowledgeEdgeTx(
	ctx context.Context,
	tx *sql.Tx,
	sourceID string,
	targetID string,
	edgeType string,
) error {
	if sourceID == "" || targetID == "" || edgeType == "" {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO knowledge_edges (source_id, target_id, edge_type, created_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(source_id, target_id, edge_type) DO NOTHING
	`, sourceID, targetID, edgeType, nowUTC()); err != nil {
		return fmt.Errorf("ensure knowledge edge %s -> %s (%s): %w", sourceID, targetID, edgeType, err)
	}
	return nil
}

func (s *Store) resolveRootTopicTx(
	ctx context.Context,
	tx *sql.Tx,
	node *domain.KnowledgeNode,
) (string, error) {
	if node == nil {
		return "", nil
	}
	if node.NodeType == domain.KnowledgeNodeTypeTopic {
		return node.Label, nil
	}

	parentID := node.ParentID
	for depth := 0; depth < 8 && parentID != ""; depth++ {
		parent, err := s.getKnowledgeNodeByIDTx(ctx, tx, parentID)
		if err != nil {
			return "", err
		}
		if parent == nil {
			return "", nil
		}
		if parent.NodeType == domain.KnowledgeNodeTypeTopic {
			return parent.Label, nil
		}
		parentID = parent.ParentID
	}

	return "", nil
}

func maxFloat(left float64, right float64) float64 {
	if left > right {
		return left
	}
	return right
}
