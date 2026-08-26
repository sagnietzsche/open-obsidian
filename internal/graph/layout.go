package graph

import "math"

// ForceLayout runs Fruchterman-Reingold force-directed simulation.
type ForceLayout struct {
	nodes []*Node
	edges []Edge
	width, height float64
	k float64
	temperature float64
}

// NewForceLayout initializes positions randomly (or at center if zero) and computes k.
func NewForceLayout(nodes []Node, edges []Edge, width, height float64) *ForceLayout {
	ns := make([]*Node, len(nodes))
	for i := range nodes {
		cp := nodes[i]
		if cp.X == 0 && cp.Y == 0 {
			cp.X = width/2 + (randFloat(i)-0.5)*width*0.5
			cp.Y = height/2 + (randFloat(i*2+1)-0.5)*height*0.5
		}
		ns[i] = &cp
	}
	area := width * height
	k := math.Sqrt(area / math.Max(1, float64(len(nodes))))
	return &ForceLayout{nodes: ns, edges: edges, width: width, height: height, k: k, temperature: width / 10}
}

func randFloat(seed int) float64 {
	// deterministic pseudo
	x := float64((seed*9301 + 49297) % 233280)
	return x / 233280.0
}

// Nodes returns current nodes.
func (f *ForceLayout) Nodes() []*Node { return f.nodes }

// Step advances simulation one tick. Returns true if still moving (energy > epsilon).
func (f *ForceLayout) Step() bool {
	if len(f.nodes) == 0 {
		return false
	}
	disp := make([][2]float64, len(f.nodes))
	// repulsive
	for i := 0; i < len(f.nodes); i++ {
		for j := i + 1; j < len(f.nodes); j++ {
			dx := f.nodes[i].X - f.nodes[j].X
			dy := f.nodes[i].Y - f.nodes[j].Y
			dist := math.Sqrt(dx*dx+dy*dy) + 0.01
			force := f.k * f.k / dist
			fx := dx / dist * force
			fy := dy / dist * force
			disp[i][0] += fx
			disp[i][1] += fy
			disp[j][0] -= fx
			disp[j][1] -= fy
		}
	}
	// attractive
	idxMap := map[string]int{}
	for i, n := range f.nodes {
		idxMap[n.ID] = i
	}
	for _, e := range f.edges {
		si, ok1 := idxMap[e.From]
		ti, ok2 := idxMap[e.To]
		if !ok1 || !ok2 {
			continue
		}
		dx := f.nodes[si].X - f.nodes[ti].X
		dy := f.nodes[si].Y - f.nodes[ti].Y
		dist := math.Sqrt(dx*dx+dy*dy) + 0.01
		force := dist * dist / f.k
		fx := dx / dist * force
		fy := dy / dist * force
		disp[si][0] -= fx
		disp[si][1] -= fy
		disp[ti][0] += fx
		disp[ti][1] += fy
	}
	// gravity to center
	cx, cy := f.width/2, f.height/2
	for i, n := range f.nodes {
		if n.FX != nil && n.FY != nil {
			continue
		}
		dx := n.X - cx
		dy := n.Y - cy
		disp[i][0] -= dx * 0.01
		disp[i][1] -= dy * 0.01
	}
	maxDisp := 0.0
	for i, n := range f.nodes {
		if n.FX != nil && n.FY != nil {
			n.X = *n.FX
			n.Y = *n.FY
			continue
		}
		dx, dy := disp[i][0], disp[i][1]
		dist := math.Sqrt(dx*dx+dy*dy)
		if dist > 0 {
			limited := math.Min(dist, f.temperature)
			n.X += dx / dist * limited
			n.Y += dy / dist * limited
			// clamp
			if n.X < 20 {
				n.X = 20
			}
			if n.X > f.width-20 {
				n.X = f.width - 20
			}
			if n.Y < 20 {
				n.Y = 20
			}
			if n.Y > f.height-20 {
				n.Y = f.height - 20
			}
		}
		if dist > maxDisp {
			maxDisp = dist
		}
	}
	f.temperature *= 0.99
	if f.temperature < 1 {
		f.temperature = 1
	}
	return maxDisp > 0.5 && f.temperature > 2
}

// SetSize updates viewport size (recomputes k).
func (f *ForceLayout) SetSize(w, h float64) {
	f.width, f.height = w, h
	f.k = math.Sqrt(w*h / math.Max(1, float64(len(f.nodes))))
}

// Pin fixes node at pos.
func (f *ForceLayout) Pin(id string, x, y float64) {
	for _, n := range f.nodes {
		if n.ID == id {
			n.FX = &x
			n.FY = &y
			n.X, n.Y = x, y
			return
		}
	}
}

// Unpin releases node.
func (f *ForceLayout) Unpin(id string) {
	for _, n := range f.nodes {
		if n.ID == id {
			n.FX, n.FY = nil, nil
			return
		}
	}
}
