package graph

import (
	"sort"
	"strings"
)

// Node is a note in the graph.
type Node struct {
	ID       string // vault-relative path
	Label    string // base name
	Tags     []string
	Degree   int
	Group    string // first tag for coloring
	X, Y     float64
	VX, VY   float64
	FX, FY   *float64 // pinned position when dragged
}

// Edge is a directed link source->target.
type Edge struct {
	From string
	To   string
}

// BuildGraph builds nodes/edges from index data.
// includeGhost=false hides unresolved ghost nodes; includeTags adds tag nodes.
func BuildGraph(forward map[string]map[string]int, fileTags map[string][]string, includeGhost bool, includeTags bool) ([]Node, []Edge) {
	nodeSet := map[string]*Node{}
	edges := []Edge{}
	for src, tgts := range forward {
		if _, ok := nodeSet[src]; !ok {
			nodeSet[src] = &Node{ID: src, Label: labelFor(src), Tags: fileTags[src]}
			if len(fileTags[src]) > 0 {
				nodeSet[src].Group = fileTags[src][0]
			}
		}
		for tgt := range tgts {
			if strings.HasPrefix(tgt, "ghost/") && !includeGhost {
				continue
			}
			if _, ok := nodeSet[tgt]; !ok {
				// ghost or target not in forward as source
				nodeSet[tgt] = &Node{ID: tgt, Label: labelFor(tgt), Tags: fileTags[tgt]}
				if len(fileTags[tgt]) > 0 {
					nodeSet[tgt].Group = fileTags[tgt][0]
				}
			}
			edges = append(edges, Edge{From: src, To: tgt})
		}
	}
	// ensure isolated files are nodes
	for f, tags := range fileTags {
		if _, ok := nodeSet[f]; !ok {
			nodeSet[f] = &Node{ID: f, Label: labelFor(f), Tags: tags}
			if len(tags) > 0 {
				nodeSet[f].Group = tags[0]
			}
		}
	}
	// compute degree
	deg := map[string]int{}
	for _, e := range edges {
		deg[e.From]++
		deg[e.To]++
	}
	for id, n := range nodeSet {
		n.Degree = deg[id]
	}
	// tag cluster nodes if requested
	if includeTags {
		tagNodes := map[string]*Node{}
		for f, tags := range fileTags {
			for _, t := range tags {
				tid := "tag://" + t
				if _, ok := tagNodes[tid]; !ok {
					tagNodes[tid] = &Node{ID: tid, Label: "#" + t, Group: t}
				}
				edges = append(edges, Edge{From: f, To: tid})
				tagNodes[tid].Degree++
			}
		}
		for id, n := range tagNodes {
			nodeSet[id] = n
		}
	}
	var nodes []Node
	for _, n := range nodeSet {
		nodes = append(nodes, *n)
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From == edges[j].From {
			return edges[i].To < edges[j].To
		}
		return edges[i].From < edges[j].From
	})
	return nodes, edges
}

func labelFor(rel string) string {
	// strip tag:// and ghost/
	rel = strings.TrimPrefix(rel, "tag://")
	rel = strings.TrimPrefix(rel, "ghost/")
	if rel == "" {
		return rel
	}
	// base without extension
	base := rel
	if idx := strings.LastIndex(base, "/"); idx >= 0 {
		base = base[idx+1:]
	}
	base = strings.TrimSuffix(base, ".md")
	return base
}

// LocalSubgraph returns BFS up to depth from center.
func LocalSubgraph(center string, depth int, forward, inverse map[string]map[string]int, fileTags map[string][]string) ([]Node, []Edge) {
	if depth <= 0 {
		depth = 1
	}
	visited := map[string]bool{center: true}
	frontier := []string{center}
	collectedEdges := []Edge{}
	for d := 0; d < depth; d++ {
		var next []string
		for _, cur := range frontier {
			// forward
			for tgt := range forward[cur] {
				if !strings.HasPrefix(tgt, "ghost/") {
					collectedEdges = append(collectedEdges, Edge{From: cur, To: tgt})
				}
				if !visited[tgt] {
					visited[tgt] = true
					next = append(next, tgt)
				}
			}
			// inverse (backlinks)
			for src := range inverse[cur] {
				collectedEdges = append(collectedEdges, Edge{From: src, To: cur})
				if !visited[src] {
					visited[src] = true
					next = append(next, src)
				}
			}
		}
		frontier = next
		if len(frontier) == 0 {
			break
		}
	}
	// build nodes for visited
	var nodes []Node
	for id := range visited {
		nodes = append(nodes, Node{ID: id, Label: labelFor(id), Tags: fileTags[id]})
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	return nodes, collectedEdges
}
