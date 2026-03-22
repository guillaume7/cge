package graphhealth

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/guillaume-galp/cge/internal/infra/kuzu"
)

const (
	ActionConsolidateDuplicate = "consolidate_duplicate_nodes"
	ActionPruneOrphan          = "prune_orphan_nodes"
	ActionResolveContradiction = "resolve_contradiction"
)

type Indicators struct {
	DuplicationRate    float64 `json:"duplication_rate"`
	OrphanRate         float64 `json:"orphan_rate"`
	ContradictoryFacts int     `json:"contradictory_facts"`
	DensityScore       float64 `json:"density_score"`
	ClusteringScore    float64 `json:"clustering_score"`
}

type Analysis struct {
	SnapshotAnchor string          `json:"snapshot_anchor"`
	Snapshot       kuzu.GraphStats `json:"snapshot"`
	Indicators     Indicators      `json:"indicators"`
	Plan           HygienePlan     `json:"plan"`
}

type HygienePlan struct {
	SnapshotAnchor    string             `json:"snapshot_anchor"`
	Snapshot          kuzu.GraphStats    `json:"snapshot"`
	Suggestions       HygieneSuggestions `json:"suggestions"`
	Actions           []HygieneAction    `json:"actions"`
	SelectedActionIDs []string           `json:"selected_action_ids,omitempty"`
}

type HygieneSuggestions struct {
	DuplicateGroups []DuplicateGroup `json:"duplicate_groups"`
	OrphanNodes     []OrphanNode     `json:"orphan_nodes"`
	Contradictions  []Contradiction  `json:"contradictions"`
}

type DuplicateGroup struct {
	ActionID        string   `json:"action_id"`
	NodeIDs         []string `json:"node_ids"`
	CanonicalNodeID string   `json:"canonical_node_id"`
	Reason          string   `json:"reason"`
	Signature       string   `json:"signature"`
}

type OrphanNode struct {
	ActionID string `json:"action_id"`
	NodeID   string `json:"node_id"`
	Kind     string `json:"kind"`
	Reason   string `json:"reason"`
}

type Contradiction struct {
	ActionID        string              `json:"action_id"`
	Subject         string              `json:"subject"`
	NodeIDs         []string            `json:"node_ids"`
	CanonicalNodeID string              `json:"canonical_node_id"`
	Reason          string              `json:"reason"`
	Conflicts       []ContradictionFact `json:"conflicts"`
	Resolution      ResolutionPath      `json:"resolution"`
}

type ContradictionFact struct {
	NodeID string `json:"node_id"`
	Value  string `json:"value"`
}

type ResolutionPath struct {
	Strategy        string            `json:"strategy"`
	CanonicalNodeID string            `json:"canonical_node_id"`
	RetireNodeIDs   []string          `json:"retire_node_ids,omitempty"`
	ResolvedFields  map[string]string `json:"resolved_fields,omitempty"`
	Explanation     string            `json:"explanation"`
}

type HygieneAction struct {
	ID              string          `json:"action_id"`
	Type            string          `json:"type"`
	TargetIDs       []string        `json:"target_ids"`
	CanonicalNodeID string          `json:"canonical_node_id,omitempty"`
	Explanation     string          `json:"explanation"`
	Resolution      *ResolutionPath `json:"resolution,omitempty"`
	Metadata        map[string]any  `json:"metadata,omitempty"`
}

func (a *HygieneAction) UnmarshalJSON(data []byte) error {
	type hygieneActionDTO struct {
		ID              string          `json:"id"`
		ActionID        string          `json:"action_id"`
		Type            string          `json:"type"`
		TargetIDs       []string        `json:"target_ids"`
		CanonicalNodeID string          `json:"canonical_node_id"`
		Explanation     string          `json:"explanation"`
		Resolution      *ResolutionPath `json:"resolution"`
		Metadata        map[string]any  `json:"metadata"`
	}

	var dto hygieneActionDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return err
	}

	identifier := strings.TrimSpace(dto.ActionID)
	if identifier == "" {
		identifier = strings.TrimSpace(dto.ID)
	}

	*a = HygieneAction{
		ID:              identifier,
		Type:            dto.Type,
		TargetIDs:       append([]string(nil), dto.TargetIDs...),
		CanonicalNodeID: dto.CanonicalNodeID,
		Explanation:     dto.Explanation,
		Resolution:      dto.Resolution,
		Metadata:        cloneProps(dto.Metadata),
	}
	return nil
}

type ApplyResult struct {
	TargetGraph      kuzu.Graph
	AppliedActions   []HygieneAction
	AppliedActionIDs []string
	AppliedSummary   AppliedSummary
	BeforeAnchor     string
	AfterAnchor      string
	RevisionRequired bool
}

type AppliedSummary struct {
	TotalActions           int `json:"total_actions"`
	ConsolidatedDuplicates int `json:"consolidated_duplicates"`
	PrunedOrphans          int `json:"pruned_orphans"`
	ResolvedContradictions int `json:"resolved_contradictions"`
}

func AnalyzeGraph(graph kuzu.Graph) (Analysis, error) {
	anchor, err := SnapshotAnchor(graph)
	if err != nil {
		return Analysis{}, err
	}

	duplicates := detectDuplicateGroups(graph)
	contradictions := detectContradictions(graph)
	orphans := filterOrphanNodes(detectOrphanNodes(graph), duplicates, contradictions)
	actions := buildActions(duplicates, orphans, contradictions)

	plan := HygienePlan{
		SnapshotAnchor: anchor,
		Snapshot: kuzu.GraphStats{
			Nodes:         len(graph.Nodes),
			Relationships: len(graph.Edges),
		},
		Suggestions: HygieneSuggestions{
			DuplicateGroups: duplicates,
			OrphanNodes:     orphans,
			Contradictions:  contradictions,
		},
		Actions: actions,
	}

	indicators := Indicators{
		DuplicationRate:    duplicationRate(len(graph.Nodes), duplicates),
		OrphanRate:         ratio(len(orphans), len(graph.Nodes)),
		ContradictoryFacts: len(contradictions),
		DensityScore:       densityScore(len(graph.Nodes), len(graph.Edges)),
		ClusteringScore:    clusteringScore(graph),
	}

	return Analysis{
		SnapshotAnchor: anchor,
		Snapshot: kuzu.GraphStats{
			Nodes:         len(graph.Nodes),
			Relationships: len(graph.Edges),
		},
		Indicators: indicators,
		Plan:       plan,
	}, nil
}

func SnapshotAnchor(graph kuzu.Graph) (string, error) {
	normalized := struct {
		Nodes []kuzu.EntityRecord   `json:"nodes"`
		Edges []kuzu.RelationRecord `json:"edges"`
	}{
		Nodes: cloneAndSortNodes(graph.Nodes),
		Edges: cloneAndSortEdges(graph.Edges),
	}

	payload, err := json.Marshal(normalized)
	if err != nil {
		return "", fmt.Errorf("encode graph snapshot: %w", err)
	}

	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func ApplyPlan(graph kuzu.Graph, plan HygienePlan) (ApplyResult, error) {
	beforeAnchor, err := SnapshotAnchor(graph)
	if err != nil {
		return ApplyResult{}, err
	}

	if strings.TrimSpace(plan.SnapshotAnchor) == "" {
		return ApplyResult{}, fmt.Errorf("hygiene plan snapshot anchor is required")
	}
	if beforeAnchor != strings.TrimSpace(plan.SnapshotAnchor) {
		return ApplyResult{}, fmt.Errorf("hygiene plan snapshot anchor does not match the current graph state")
	}

	actionByID := map[string]HygieneAction{}
	for _, action := range plan.Actions {
		actionByID[action.ID] = cloneAction(action)
	}

	selectedIDs := uniqueStrings(plan.SelectedActionIDs)
	appliedActions := make([]HygieneAction, 0, len(selectedIDs))
	mutable := cloneGraph(graph)
	summary := AppliedSummary{}

	for _, actionID := range selectedIDs {
		action, ok := actionByID[actionID]
		if !ok {
			return ApplyResult{}, fmt.Errorf("selected hygiene action %q is not defined in the plan", actionID)
		}
		if err := validateAction(mutable, action); err != nil {
			return ApplyResult{}, err
		}
		var applyErr error
		switch action.Type {
		case ActionConsolidateDuplicate:
			mutable, applyErr = applyConsolidateDuplicate(mutable, action)
			summary.ConsolidatedDuplicates++
		case ActionPruneOrphan:
			mutable, applyErr = applyPruneOrphan(mutable, action)
			summary.PrunedOrphans++
		case ActionResolveContradiction:
			mutable, applyErr = applyResolveContradiction(mutable, action)
			summary.ResolvedContradictions++
		default:
			return ApplyResult{}, fmt.Errorf("unsupported hygiene action type %q", action.Type)
		}
		if applyErr != nil {
			return ApplyResult{}, applyErr
		}
		appliedActions = append(appliedActions, action)
	}

	afterAnchor, err := SnapshotAnchor(mutable)
	if err != nil {
		return ApplyResult{}, err
	}

	summary.TotalActions = len(appliedActions)
	return ApplyResult{
		TargetGraph:      normalizeGraph(mutable),
		AppliedActions:   appliedActions,
		AppliedActionIDs: selectedIDs,
		AppliedSummary:   summary,
		BeforeAnchor:     beforeAnchor,
		AfterAnchor:      afterAnchor,
		RevisionRequired: beforeAnchor != afterAnchor,
	}, nil
}

func validateAction(graph kuzu.Graph, action HygieneAction) error {
	nodeByID := map[string]kuzu.EntityRecord{}
	for _, node := range graph.Nodes {
		nodeByID[node.ID] = node
	}
	edgeKeys := edgeSet(graph.Edges)

	switch action.Type {
	case ActionConsolidateDuplicate:
		if strings.TrimSpace(action.CanonicalNodeID) == "" {
			return fmt.Errorf("duplicate consolidation action %q is missing canonical_node_id", action.ID)
		}
		if _, ok := nodeByID[action.CanonicalNodeID]; !ok {
			return fmt.Errorf("duplicate consolidation action %q references missing canonical node %q", action.ID, action.CanonicalNodeID)
		}
		if len(action.TargetIDs) < 2 {
			return fmt.Errorf("duplicate consolidation action %q requires at least two target nodes", action.ID)
		}
		for _, target := range action.TargetIDs {
			if _, ok := nodeByID[target]; !ok {
				return fmt.Errorf("duplicate consolidation action %q references missing node %q", action.ID, target)
			}
		}
	case ActionPruneOrphan:
		if len(action.TargetIDs) != 1 {
			return fmt.Errorf("orphan prune action %q must target exactly one node", action.ID)
		}
		target := action.TargetIDs[0]
		if _, ok := nodeByID[target]; !ok {
			return fmt.Errorf("orphan prune action %q references missing node %q", action.ID, target)
		}
		if nodeDegree(target, edgeKeys) != 0 {
			return fmt.Errorf("orphan prune action %q targets node %q, which is no longer orphaned", action.ID, target)
		}
	case ActionResolveContradiction:
		if strings.TrimSpace(action.CanonicalNodeID) == "" {
			return fmt.Errorf("contradiction resolution action %q is missing canonical_node_id", action.ID)
		}
		if _, ok := nodeByID[action.CanonicalNodeID]; !ok {
			return fmt.Errorf("contradiction resolution action %q references missing canonical node %q", action.ID, action.CanonicalNodeID)
		}
		if len(action.TargetIDs) < 2 {
			return fmt.Errorf("contradiction resolution action %q requires at least two target nodes", action.ID)
		}
		for _, target := range action.TargetIDs {
			if _, ok := nodeByID[target]; !ok {
				return fmt.Errorf("contradiction resolution action %q references missing node %q", action.ID, target)
			}
		}
		if action.Resolution == nil {
			return fmt.Errorf("contradiction resolution action %q is missing resolution details", action.ID)
		}
	default:
		return fmt.Errorf("unsupported hygiene action type %q", action.Type)
	}

	return nil
}

func buildActions(duplicates []DuplicateGroup, orphans []OrphanNode, contradictions []Contradiction) []HygieneAction {
	actions := make([]HygieneAction, 0, len(duplicates)+len(orphans)+len(contradictions))
	for _, duplicate := range duplicates {
		actions = append(actions, HygieneAction{
			ID:              duplicate.ActionID,
			Type:            ActionConsolidateDuplicate,
			TargetIDs:       append([]string(nil), duplicate.NodeIDs...),
			CanonicalNodeID: duplicate.CanonicalNodeID,
			Explanation:     duplicate.Reason,
			Metadata: map[string]any{
				"signature": duplicate.Signature,
			},
		})
	}
	for _, orphan := range orphans {
		actions = append(actions, HygieneAction{
			ID:          orphan.ActionID,
			Type:        ActionPruneOrphan,
			TargetIDs:   []string{orphan.NodeID},
			Explanation: orphan.Reason,
		})
	}
	for _, contradiction := range contradictions {
		resolution := contradiction.Resolution
		actions = append(actions, HygieneAction{
			ID:              contradiction.ActionID,
			Type:            ActionResolveContradiction,
			TargetIDs:       append([]string(nil), contradiction.NodeIDs...),
			CanonicalNodeID: contradiction.CanonicalNodeID,
			Explanation:     contradiction.Reason,
			Resolution:      &resolution,
		})
	}
	sort.Slice(actions, func(i, j int) bool { return actions[i].ID < actions[j].ID })
	return actions
}

func detectDuplicateGroups(graph kuzu.Graph) []DuplicateGroup {
	groups := map[string][]kuzu.EntityRecord{}
	for _, node := range graph.Nodes {
		titleKey := normalizeText(node.Title)
		bodyKey := normalizeText(strings.Join(textualNodeParts(node), " "))
		if titleKey == "" || bodyKey == "" {
			continue
		}
		signature := strings.Join([]string{normalizeText(node.Kind), titleKey, bodyKey}, "|")
		groups[signature] = append(groups[signature], node)
	}

	duplicates := make([]DuplicateGroup, 0)
	for signature, nodes := range groups {
		if len(nodes) < 2 {
			continue
		}
		sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
		canonical := chooseCanonical(nodes)
		nodeIDs := make([]string, 0, len(nodes))
		for _, node := range nodes {
			nodeIDs = append(nodeIDs, node.ID)
		}
		duplicates = append(duplicates, DuplicateGroup{
			ActionID:        "duplicate:" + canonical.ID,
			NodeIDs:         nodeIDs,
			CanonicalNodeID: canonical.ID,
			Reason:          "nodes share the same normalized title and text fingerprint",
			Signature:       signature,
		})
	}
	sort.Slice(duplicates, func(i, j int) bool { return duplicates[i].ActionID < duplicates[j].ActionID })
	return duplicates
}

func filterOrphanNodes(orphans []OrphanNode, duplicates []DuplicateGroup, contradictions []Contradiction) []OrphanNode {
	excluded := map[string]struct{}{}
	for _, duplicate := range duplicates {
		for _, nodeID := range duplicate.NodeIDs {
			excluded[nodeID] = struct{}{}
		}
	}
	for _, contradiction := range contradictions {
		for _, nodeID := range contradiction.NodeIDs {
			excluded[nodeID] = struct{}{}
		}
	}
	filtered := make([]OrphanNode, 0, len(orphans))
	for _, orphan := range orphans {
		if _, ok := excluded[orphan.NodeID]; ok {
			continue
		}
		filtered = append(filtered, orphan)
	}
	return filtered
}

func detectOrphanNodes(graph kuzu.Graph) []OrphanNode {
	degrees := map[string]int{}
	for _, edge := range graph.Edges {
		degrees[edge.From]++
		degrees[edge.To]++
	}

	orphans := make([]OrphanNode, 0)
	for _, node := range graph.Nodes {
		if degrees[node.ID] != 0 {
			continue
		}
		orphans = append(orphans, OrphanNode{
			ActionID: "orphan:" + node.ID,
			NodeID:   node.ID,
			Kind:     node.Kind,
			Reason:   "node has no incoming or outgoing relationships in the current snapshot",
		})
	}
	sort.Slice(orphans, func(i, j int) bool { return orphans[i].NodeID < orphans[j].NodeID })
	return orphans
}

func detectContradictions(graph kuzu.Graph) []Contradiction {
	type subjectValue struct {
		node  kuzu.EntityRecord
		value string
	}
	groups := map[string][]subjectValue{}
	for _, node := range graph.Nodes {
		subject := contradictionSubject(node)
		value := contradictionValue(node)
		if subject == "" || value == "" {
			continue
		}
		groups[subject] = append(groups[subject], subjectValue{node: node, value: value})
	}

	contradictions := make([]Contradiction, 0)
	for subject, values := range groups {
		valueGroups := map[string][]kuzu.EntityRecord{}
		for _, item := range values {
			valueGroups[item.value] = append(valueGroups[item.value], item.node)
		}
		if len(valueGroups) < 2 {
			continue
		}

		allNodes := make([]kuzu.EntityRecord, 0, len(values))
		conflicts := make([]ContradictionFact, 0, len(values))
		for value, nodes := range valueGroups {
			sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
			for _, node := range nodes {
				allNodes = append(allNodes, node)
				conflicts = append(conflicts, ContradictionFact{NodeID: node.ID, Value: value})
			}
		}
		sort.Slice(allNodes, func(i, j int) bool { return allNodes[i].ID < allNodes[j].ID })
		sort.Slice(conflicts, func(i, j int) bool {
			if conflicts[i].Value != conflicts[j].Value {
				return conflicts[i].Value < conflicts[j].Value
			}
			return conflicts[i].NodeID < conflicts[j].NodeID
		})

		canonical := chooseCanonical(allNodes)
		resolvedValue := selectResolvedValue(canonical, conflicts)
		nodeIDs := make([]string, 0, len(allNodes))
		retireIDs := make([]string, 0, len(allNodes)-1)
		for _, node := range allNodes {
			nodeIDs = append(nodeIDs, node.ID)
			if node.ID != canonical.ID {
				retireIDs = append(retireIDs, node.ID)
			}
		}
		contradictions = append(contradictions, Contradiction{
			ActionID:        "contradiction:" + canonical.ID,
			Subject:         subject,
			NodeIDs:         nodeIDs,
			CanonicalNodeID: canonical.ID,
			Reason:          "multiple nodes describe the same normalized subject with conflicting fact values",
			Conflicts:       conflicts,
			Resolution: ResolutionPath{
				Strategy:        "retain_canonical_fact",
				CanonicalNodeID: canonical.ID,
				RetireNodeIDs:   retireIDs,
				ResolvedFields: map[string]string{
					"summary": resolvedValue,
				},
				Explanation: "retain the canonical fact node, update it with the selected normalized value, and retire conflicting nodes",
			},
		})
	}
	sort.Slice(contradictions, func(i, j int) bool { return contradictions[i].ActionID < contradictions[j].ActionID })
	return contradictions
}

func applyConsolidateDuplicate(graph kuzu.Graph, action HygieneAction) (kuzu.Graph, error) {
	canonical := action.CanonicalNodeID
	if canonical == "" {
		return graph, fmt.Errorf("duplicate consolidation action %q is missing canonical node", action.ID)
	}
	mergeNodeIDs := make([]string, 0, len(action.TargetIDs)-1)
	for _, id := range action.TargetIDs {
		if id == canonical {
			continue
		}
		mergeNodeIDs = append(mergeNodeIDs, id)
	}
	return mergeNodes(graph, canonical, mergeNodeIDs, nil)
}

func applyPruneOrphan(graph kuzu.Graph, action HygieneAction) (kuzu.Graph, error) {
	if len(action.TargetIDs) != 1 {
		return graph, fmt.Errorf("orphan prune action %q must target exactly one node", action.ID)
	}
	return removeNodes(graph, action.TargetIDs), nil
}

func applyResolveContradiction(graph kuzu.Graph, action HygieneAction) (kuzu.Graph, error) {
	resolution := action.Resolution
	if resolution == nil {
		return graph, fmt.Errorf("contradiction resolution action %q is missing resolution details", action.ID)
	}
	updates := resolution.ResolvedFields
	mergeNodeIDs := make([]string, 0, len(action.TargetIDs)-1)
	for _, id := range action.TargetIDs {
		if id == action.CanonicalNodeID {
			continue
		}
		mergeNodeIDs = append(mergeNodeIDs, id)
	}
	return mergeNodes(graph, action.CanonicalNodeID, mergeNodeIDs, updates)
}

func mergeNodes(graph kuzu.Graph, canonicalID string, mergeNodeIDs []string, updates map[string]string) (kuzu.Graph, error) {
	nodeByID := map[string]kuzu.EntityRecord{}
	for _, node := range graph.Nodes {
		nodeByID[node.ID] = cloneNode(node)
	}
	canonical, ok := nodeByID[canonicalID]
	if !ok {
		return graph, fmt.Errorf("canonical node %q does not exist", canonicalID)
	}

	mergeSet := map[string]struct{}{}
	for _, id := range mergeNodeIDs {
		if id == canonicalID {
			continue
		}
		node, ok := nodeByID[id]
		if !ok {
			return graph, fmt.Errorf("merge target node %q does not exist", id)
		}
		mergeSet[id] = struct{}{}
		canonical = mergeNodeState(canonical, node)
	}
	if len(updates) > 0 {
		canonical = applyResolvedFields(canonical, updates)
	}
	nodeByID[canonicalID] = canonical
	for id := range mergeSet {
		delete(nodeByID, id)
	}

	edges := make([]kuzu.RelationRecord, 0, len(graph.Edges))
	seenEdges := map[string]struct{}{}
	for _, edge := range graph.Edges {
		updated := cloneEdge(edge)
		if _, ok := mergeSet[updated.From]; ok {
			updated.From = canonicalID
		}
		if _, ok := mergeSet[updated.To]; ok {
			updated.To = canonicalID
		}
		if updated.From == updated.To {
			continue
		}
		if _, ok := nodeByID[updated.From]; !ok {
			continue
		}
		if _, ok := nodeByID[updated.To]; !ok {
			continue
		}
		key := edgeKey(updated)
		if _, ok := seenEdges[key]; ok {
			continue
		}
		seenEdges[key] = struct{}{}
		edges = append(edges, updated)
	}

	nodes := make([]kuzu.EntityRecord, 0, len(nodeByID))
	for _, node := range nodeByID {
		nodes = append(nodes, node)
	}
	return normalizeGraph(kuzu.Graph{Nodes: nodes, Edges: edges}), nil
}

func removeNodes(graph kuzu.Graph, nodeIDs []string) kuzu.Graph {
	removeSet := map[string]struct{}{}
	for _, id := range nodeIDs {
		removeSet[id] = struct{}{}
	}
	filteredNodes := make([]kuzu.EntityRecord, 0, len(graph.Nodes))
	for _, node := range graph.Nodes {
		if _, ok := removeSet[node.ID]; ok {
			continue
		}
		filteredNodes = append(filteredNodes, cloneNode(node))
	}
	filteredEdges := make([]kuzu.RelationRecord, 0, len(graph.Edges))
	for _, edge := range graph.Edges {
		if _, ok := removeSet[edge.From]; ok {
			continue
		}
		if _, ok := removeSet[edge.To]; ok {
			continue
		}
		filteredEdges = append(filteredEdges, cloneEdge(edge))
	}
	return normalizeGraph(kuzu.Graph{Nodes: filteredNodes, Edges: filteredEdges})
}

func chooseCanonical(nodes []kuzu.EntityRecord) kuzu.EntityRecord {
	best := cloneNode(nodes[0])
	for _, node := range nodes[1:] {
		candidate := cloneNode(node)
		if nodeRichness(candidate) > nodeRichness(best) {
			best = candidate
			continue
		}
		if nodeRichness(candidate) == nodeRichness(best) && candidate.ID < best.ID {
			best = candidate
		}
	}
	return best
}

func nodeRichness(node kuzu.EntityRecord) int {
	score := len(strings.TrimSpace(node.Title)) + len(strings.TrimSpace(node.Summary)) + len(strings.TrimSpace(node.Content))
	for key, value := range node.Props {
		score += len(key) + len(stringifyScalar(value))
	}
	return score
}

func mergeNodeState(base, incoming kuzu.EntityRecord) kuzu.EntityRecord {
	if len(strings.TrimSpace(base.Title)) < len(strings.TrimSpace(incoming.Title)) {
		base.Title = incoming.Title
	}
	if len(strings.TrimSpace(base.Summary)) < len(strings.TrimSpace(incoming.Summary)) {
		base.Summary = incoming.Summary
	}
	if len(strings.TrimSpace(base.Content)) < len(strings.TrimSpace(incoming.Content)) {
		base.Content = incoming.Content
	}
	if len(strings.TrimSpace(base.RepoPath)) == 0 {
		base.RepoPath = incoming.RepoPath
	}
	if len(strings.TrimSpace(base.Language)) == 0 {
		base.Language = incoming.Language
	}
	base.Tags = mergeStringLists(base.Tags, incoming.Tags)
	if base.Props == nil {
		base.Props = map[string]any{}
	}
	for key, value := range incoming.Props {
		if _, ok := base.Props[key]; ok {
			continue
		}
		base.Props[key] = value
	}
	return base
}

func applyResolvedFields(node kuzu.EntityRecord, updates map[string]string) kuzu.EntityRecord {
	for key, value := range updates {
		switch key {
		case "title":
			node.Title = value
		case "summary":
			node.Summary = value
		case "content":
			node.Content = value
		default:
			if node.Props == nil {
				node.Props = map[string]any{}
			}
			node.Props[key] = value
		}
	}
	return node
}

func contradictionSubject(node kuzu.EntityRecord) string {
	parts := []string{normalizeText(node.Kind)}
	for _, candidate := range []string{node.Title, node.RepoPath, stringifyProp(node.Props, "subject"), stringifyProp(node.Props, "key")} {
		normalized := normalizeText(candidate)
		if normalized == "" {
			continue
		}
		parts = append(parts, normalized)
		break
	}
	if len(parts) < 2 {
		return ""
	}
	return strings.Join(parts, "|")
}

func contradictionValue(node kuzu.EntityRecord) string {
	for _, candidate := range []string{
		stringifyProp(node.Props, "value"),
		stringifyProp(node.Props, "status"),
		stringifyProp(node.Props, "state"),
		stringifyProp(node.Props, "enabled"),
		node.Content,
		node.Summary,
	} {
		normalized := normalizeText(candidate)
		if normalized != "" {
			return normalized
		}
	}
	return ""
}

func selectResolvedValue(canonical kuzu.EntityRecord, conflicts []ContradictionFact) string {
	preferred := contradictionValue(canonical)
	if preferred != "" {
		return preferred
	}
	if len(conflicts) == 0 {
		return ""
	}
	return conflicts[0].Value
}

func textualNodeParts(node kuzu.EntityRecord) []string {
	parts := []string{node.Title, node.Summary, node.Content}
	keys := make([]string, 0, len(node.Props))
	for key := range node.Props {
		if isTextualPropKey(key) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		parts = append(parts, stringifyScalar(node.Props[key]))
	}
	return parts
}

func isTextualPropKey(key string) bool {
	normalized := normalizeText(key)
	switch normalized {
	case "summary", "content", "description", "text", "value", "status", "state":
		return true
	default:
		return false
	}
}

func stringifyProp(props map[string]any, key string) string {
	if props == nil {
		return ""
	}
	return stringifyScalar(props[key])
}

func stringifyScalar(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case bool:
		return strconv.FormatBool(typed)
	case int:
		return strconv.Itoa(typed)
	case int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return fmt.Sprint(typed)
	default:
		return ""
	}
}

func duplicationRate(totalNodes int, duplicates []DuplicateGroup) float64 {
	duplicateNodes := 0
	for _, group := range duplicates {
		if len(group.NodeIDs) > 1 {
			duplicateNodes += len(group.NodeIDs) - 1
		}
	}
	return ratio(duplicateNodes, totalNodes)
}

func densityScore(totalNodes, totalEdges int) float64 {
	if totalNodes < 2 || totalEdges == 0 {
		return 0
	}
	possible := float64(totalNodes * (totalNodes - 1))
	return roundFloat(float64(totalEdges) / possible)
}

func clusteringScore(graph kuzu.Graph) float64 {
	adjacency := map[string]map[string]struct{}{}
	for _, node := range graph.Nodes {
		adjacency[node.ID] = map[string]struct{}{}
	}
	for _, edge := range graph.Edges {
		if adjacency[edge.From] == nil {
			adjacency[edge.From] = map[string]struct{}{}
		}
		if adjacency[edge.To] == nil {
			adjacency[edge.To] = map[string]struct{}{}
		}
		adjacency[edge.From][edge.To] = struct{}{}
		adjacency[edge.To][edge.From] = struct{}{}
	}

	total := 0.0
	eligible := 0.0
	for _, node := range graph.Nodes {
		neighbors := adjacency[node.ID]
		if len(neighbors) < 2 {
			continue
		}
		neighborIDs := make([]string, 0, len(neighbors))
		for id := range neighbors {
			neighborIDs = append(neighborIDs, id)
		}
		sort.Strings(neighborIDs)
		links := 0
		possible := 0
		for i := 0; i < len(neighborIDs); i++ {
			for j := i + 1; j < len(neighborIDs); j++ {
				possible++
				if _, ok := adjacency[neighborIDs[i]][neighborIDs[j]]; ok {
					links++
				}
			}
		}
		if possible == 0 {
			continue
		}
		total += float64(links) / float64(possible)
		eligible++
	}
	if eligible == 0 {
		return 0
	}
	return roundFloat(total / eligible)
}

func ratio(numerator, denominator int) float64 {
	if denominator == 0 {
		return 0
	}
	return roundFloat(float64(numerator) / float64(denominator))
}

func roundFloat(value float64) float64 {
	return math.Round(value*10000) / 10000
}

func normalizeGraph(graph kuzu.Graph) kuzu.Graph {
	return kuzu.Graph{
		Nodes: cloneAndSortNodes(graph.Nodes),
		Edges: cloneAndSortEdges(graph.Edges),
	}
}

func cloneGraph(graph kuzu.Graph) kuzu.Graph {
	return kuzu.Graph{
		Nodes: cloneAndSortNodes(graph.Nodes),
		Edges: cloneAndSortEdges(graph.Edges),
	}
}

func cloneAndSortNodes(nodes []kuzu.EntityRecord) []kuzu.EntityRecord {
	cloned := make([]kuzu.EntityRecord, 0, len(nodes))
	for _, node := range nodes {
		cloned = append(cloned, cloneNode(node))
	}
	sort.Slice(cloned, func(i, j int) bool { return cloned[i].ID < cloned[j].ID })
	return cloned
}

func cloneAndSortEdges(edges []kuzu.RelationRecord) []kuzu.RelationRecord {
	cloned := make([]kuzu.RelationRecord, 0, len(edges))
	for _, edge := range edges {
		cloned = append(cloned, cloneEdge(edge))
	}
	sort.Slice(cloned, func(i, j int) bool {
		if cloned[i].From != cloned[j].From {
			return cloned[i].From < cloned[j].From
		}
		if cloned[i].Kind != cloned[j].Kind {
			return cloned[i].Kind < cloned[j].Kind
		}
		return cloned[i].To < cloned[j].To
	})
	return cloned
}

func cloneNode(node kuzu.EntityRecord) kuzu.EntityRecord {
	clone := node
	clone.Tags = append([]string(nil), node.Tags...)
	clone.Props = cloneProps(node.Props)
	return clone
}

func cloneEdge(edge kuzu.RelationRecord) kuzu.RelationRecord {
	clone := edge
	clone.Props = cloneProps(edge.Props)
	return clone
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

func cloneAction(action HygieneAction) HygieneAction {
	clone := action
	clone.TargetIDs = append([]string(nil), action.TargetIDs...)
	clone.Metadata = cloneProps(action.Metadata)
	if action.Resolution != nil {
		resolution := *action.Resolution
		resolution.RetireNodeIDs = append([]string(nil), action.Resolution.RetireNodeIDs...)
		if len(action.Resolution.ResolvedFields) > 0 {
			resolution.ResolvedFields = map[string]string{}
			for key, value := range action.Resolution.ResolvedFields {
				resolution.ResolvedFields[key] = value
			}
		}
		clone.Resolution = &resolution
	}
	return clone
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func mergeStringLists(left, right []string) []string {
	set := map[string]struct{}{}
	for _, value := range append(append([]string(nil), left...), right...) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		set[trimmed] = struct{}{}
	}
	if len(set) == 0 {
		return nil
	}
	merged := make([]string, 0, len(set))
	for value := range set {
		merged = append(merged, value)
	}
	sort.Strings(merged)
	return merged
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	unique := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		unique = append(unique, trimmed)
	}
	return unique
}

func edgeSet(edges []kuzu.RelationRecord) map[string]struct{} {
	set := map[string]struct{}{}
	for _, edge := range edges {
		set[edgeKey(edge)] = struct{}{}
	}
	return set
}

func edgeKey(edge kuzu.RelationRecord) string {
	return edge.From + "\x00" + edge.Kind + "\x00" + edge.To
}

func nodeDegree(nodeID string, edges map[string]struct{}) int {
	degree := 0
	prefix := nodeID + "\x00"
	suffix := "\x00" + nodeID
	for key := range edges {
		if strings.HasPrefix(key, prefix) || strings.HasSuffix(key, suffix) {
			degree++
		}
	}
	return degree
}

func normalizeText(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}
	var builder strings.Builder
	lastSpace := true
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			builder.WriteRune(r)
			lastSpace = false
			continue
		}
		if lastSpace {
			continue
		}
		builder.WriteByte(' ')
		lastSpace = true
	}
	return strings.TrimSpace(builder.String())
}
