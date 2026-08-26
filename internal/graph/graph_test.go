package graph

import "testing"

func TestBuildGraph(t *testing.T) {
	forward := map[string]map[string]int{"A.md": {"B.md": 1}, "B.md": {"C.md": 1}}
	fileTags := map[string][]string{"A.md": {"project"}, "B.md": {"project"}, "C.md": {"todo"}}
	nodes, edges := BuildGraph(forward, fileTags, false, false)
	if len(nodes) != 3 {
		t.Fatalf("nodes %d", len(nodes))
	}
	if len(edges) != 2 {
		t.Fatalf("edges %d", len(edges))
	}
	// with tags
	nodes2, edges2 := BuildGraph(forward, fileTags, false, true)
	// tag nodes: project and todo => 5 nodes total
	if len(nodes2) != 5 {
		t.Fatalf("tag nodes %d nodes %v", len(nodes2), nodes2)
	}
	if len(edges2) != 2+3 { // 2 note edges + 3 tag edges (A->project, B->project, C->todo)
		t.Fatalf("tag edges %d", len(edges2))
	}
}

func TestForceLayout(t *testing.T) {
	nodes := []Node{{ID: "A.md", X: 100, Y: 100}, {ID: "B.md", X: 200, Y: 200}}
	edges := []Edge{{From: "A.md", To: "B.md"}}
	fl := NewForceLayout(nodes, edges, 800, 600)
	for i := 0; i < 10; i++ {
		fl.Step()
	}
	if fl.Nodes()[0].X == 100 && fl.Nodes()[0].Y == 100 {
		t.Fatalf("layout didn't move")
	}
}

func TestLocalSubgraph(t *testing.T) {
	forward := map[string]map[string]int{"A.md": {"B.md": 1, "C.md": 1}, "B.md": {"C.md": 1}}
	inverse := map[string]map[string]int{"B.md": {"A.md": 1}, "C.md": {"B.md": 1, "A.md": 1}}
	fileTags := map[string][]string{}
	nodes, edges := LocalSubgraph("A.md", 1, forward, inverse, fileTags)
	if len(nodes) != 3 { // A + B + C (both direct)
		t.Fatalf("local nodes %d %v", len(nodes), nodes)
	}
	if len(edges) < 2 {
		t.Fatalf("local edges %d", len(edges))
	}
}
