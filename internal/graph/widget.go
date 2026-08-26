package graph

import (
	"image/color"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// GraphView is a custom widget showing nodes as circles + labels and edges as lines
// with a custom graphical renderer (graphRenderer) and force layout + viewport pan/zoom.
type GraphView struct {
	widget.BaseWidget
	nodes     []Node
	edges     []Edge
	fileTags  map[string][]string
	viewport  Viewport
	layout    *ForceLayout
	renderer  *graphRenderer
	OnSelect  func(id string)
	showTags  bool
}

func NewGraphView() *GraphView {
	g := &GraphView{viewport: NewViewport()}
	g.ExtendBaseWidget(g)
	return g
}

func (g *GraphView) SetData(nodes []Node, edges []Edge, fileTags map[string][]string, showTags bool) {
	g.nodes = nodes
	g.edges = edges
	g.fileTags = fileTags
	g.showTags = showTags
	// init layout with current size or default
	w, h := 800.0, 600.0
	if g.Size().Width > 10 {
		w = float64(g.Size().Width)
		h = float64(g.Size().Height)
	}
	g.layout = NewForceLayout(nodes, edges, w, h)
	if g.renderer != nil {
		g.renderer.rebuild()
	}
	g.Refresh()
}

// SetLocal sets local subgraph for center.
func (g *GraphView) SetLocal(center string, depth int, forward, inverse map[string]map[string]int, fileTags map[string][]string) {
	nodes, edges := LocalSubgraph(center, depth, forward, inverse, fileTags)
	g.SetData(nodes, edges, fileTags, g.showTags)
}

func (g *GraphView) CreateRenderer() fyne.WidgetRenderer {
	r := newGraphRenderer(g)
	g.renderer = r
	return r
}

// Interaction: Tappable, Draggable, Scrollable
func (g *GraphView) Tapped(ev *fyne.PointEvent) {
	// hit test nodes
	for _, n := range g.layout.Nodes() {
		sx, sy := g.viewport.WorldToScreen(n.X, n.Y)
		dx := float64(ev.Position.X) - sx
		dy := float64(ev.Position.Y) - sy
		rad := nodeRadius(n)
		if dx*dx+dy*dy <= rad*rad {
			if g.OnSelect != nil {
				g.OnSelect(n.ID)
			}
			return
		}
	}
}
func (g *GraphView) TappedSecondary(ev *fyne.PointEvent) {}

func (g *GraphView) Dragged(ev *fyne.DragEvent) {
	// if near a node, pin and move; else pan
	for _, n := range g.layout.Nodes() {
		sx, sy := g.viewport.WorldToScreen(n.X, n.Y)
		dx := float64(ev.Position.X-ev.Dragged.DX) - sx
		dy := float64(ev.Position.Y-ev.Dragged.DY) - sy
		if dx*dx+dy*dy < 400 { // 20px radius
			wx, wy := g.viewport.ScreenToWorld(float64(ev.Position.X), float64(ev.Position.Y))
			g.layout.Pin(n.ID, wx, wy)
			g.Refresh()
			return
		}
	}
	g.viewport.Pan(float64(ev.Dragged.DX), float64(ev.Dragged.DY))
	g.Refresh()
}
func (g *GraphView) DragEnd() {}

func (g *GraphView) Scrolled(ev *fyne.ScrollEvent) {
	factor := math.Pow(1.1, float64(-ev.Scrolled.DY)/10)
	cx := float64(g.Size().Width) / 2
	cy := float64(g.Size().Height) / 2
	g.viewport.Zoom(factor, cx, cy)
	g.Refresh()
}

// Tick advances layout one step if animating.
func (g *GraphView) Tick() bool {
	if g.layout == nil {
		return false
	}
	moving := g.layout.Step()
	if moving {
		g.Refresh()
	}
	return moving
}

func nodeRadius(n *Node) float64 {
	r := 8 + float64(n.Degree)*1.5
	if r > 18 {
		r = 18
	}
	if r < 6 {
		r = 6
	}
	// tag nodes smaller
	if len(n.ID) > 5 && n.ID[:5] == "tag:/" {
		return 10
	}
	return r
}

// graphRenderer implements fyne.WidgetRenderer with custom graphical logic.
type graphRenderer struct {
	g       *GraphView
	bg      *canvas.Rectangle
	lines   []*canvas.Line
	circles []*canvas.Circle
	labels  []*canvas.Text
	objects []fyne.CanvasObject
	size    fyne.Size
}

func newGraphRenderer(g *GraphView) *graphRenderer {
	bg := canvas.NewRectangle(color.RGBA{0x1a, 0x1a, 0x1e, 0xff})
	r := &graphRenderer{g: g, bg: bg}
	r.objects = []fyne.CanvasObject{bg}
	r.rebuild()
	return r
}

func (r *graphRenderer) rebuild() {
	if r.g.layout == nil {
		return
	}
	nodes := r.g.layout.Nodes()
	edges := r.g.layout.Edges()
	// edges: create lines
	// resize slices
	for len(r.lines) < len(edges) {
		ln := canvas.NewLine(color.RGBA{0x55, 0x55, 0x66, 0xff})
		ln.StrokeWidth = 1
		r.lines = append(r.lines, ln)
		r.objects = append(r.objects, ln)
	}
	for len(r.circles) < len(nodes) {
		c := canvas.NewCircle(color.RGBA{0x4a, 0x90, 0xe2, 0xff})
		c.StrokeColor = color.RGBA{0xff, 0xff, 0xff, 0x22}
		c.StrokeWidth = 1
		r.circles = append(r.circles, c)
		r.objects = append(r.objects, c)
	}
	for len(r.labels) < len(nodes) {
		t := canvas.NewText("", theme.ForegroundColor())
		t.TextSize = 10
		t.Alignment = fyne.TextAlignCenter
		r.labels = append(r.labels, t)
		r.objects = append(r.objects, t)
	}
	// hide extras via Zero size? Keep but will be invisible if unused.
}

func (l *ForceLayout) Edges() []Edge { return l.edges }

func (r *graphRenderer) Layout(s fyne.Size) {
	r.size = s
	r.bg.Resize(s)
	if r.g.layout != nil {
		r.g.layout.SetSize(float64(s.Width), float64(s.Height))
	}
	r.applyTransforms()
}

func (r *graphRenderer) applyTransforms() {
	if r.g.layout == nil {
		return
	}
	nodes := r.g.layout.Nodes()
	edges := r.g.layout.Edges()
	vp := r.g.viewport
	// lines
	for i, e := range edges {
		if i >= len(r.lines) {
			break
		}
		// find node positions
		var fx, fy, tx, ty float64
		foundF, foundT := false, false
		for _, n := range nodes {
			if n.ID == e.From {
				fx, fy = vp.WorldToScreen(n.X, n.Y)
				foundF = true
			}
			if n.ID == e.To {
				tx, ty = vp.WorldToScreen(n.X, n.Y)
				foundT = true
			}
		}
		if !foundF || !foundT {
			continue
		}
		r.lines[i].Position1 = fyne.NewPos(float32(fx), float32(fy))
		r.lines[i].Position2 = fyne.NewPos(float32(tx), float32(ty))
		r.lines[i].Refresh()
	}
	// circles + labels
	for i, n := range nodes {
		if i >= len(r.circles) {
			break
		}
		sx, sy := vp.WorldToScreen(n.X, n.Y)
		rad := float32(nodeRadius(n) * vp.Scale)
		if rad < 3 {
			rad = 3
		}
		c := r.circles[i]
		c.FillColor = colorForGroup(n.Group)
		if n.Group == "" {
			c.FillColor = color.RGBA{0x4a, 0x90, 0xe2, 0xff}
		}
		// highlight ghost
		if len(n.ID) > 6 && n.ID[:6] == "ghost/" {
			c.FillColor = color.RGBA{0x66, 0x33, 0x33, 0xff}
		}
		if len(n.ID) > 5 && n.ID[:5] == "tag:/" {
			c.FillColor = color.RGBA{0x2e, 0xcc, 0x71, 0xff}
		}
		c.Resize(fyne.NewSize(rad*2, rad*2))
		c.Move(fyne.NewPos(float32(sx)-rad, float32(sy)-rad))
		c.Refresh()
		// label
		if i < len(r.labels) {
			t := r.labels[i]
			t.Text = n.Label
			// LOD: hide label when zoomed out or many nodes
			if vp.Scale < 0.6 && len(nodes) > 100 {
				t.Hidden = true
			} else {
				t.Hidden = false
				t.Move(fyne.NewPos(float32(sx)-30, float32(sy)+rad+2))
				t.Resize(fyne.NewSize(60, 14))
			}
			t.Refresh()
		}
	}
}

func (r *graphRenderer) MinSize() fyne.Size { return fyne.NewSize(400, 300) }
func (r *graphRenderer) Refresh() {
	r.applyTransforms()
	canvas.Refresh(r.bg)
}
func (r *graphRenderer) Objects() []fyne.CanvasObject { return r.objects }
func (r *graphRenderer) Destroy()                      {}

func colorForGroup(g string) color.Color {
	// deterministic hash to RGB
	if g == "" {
		return color.RGBA{0x4a, 0x90, 0xe2, 0xff}
	}
	h := 0
	for _, c := range g {
		h = h*31 + int(c)
	}
	// HSL to RGB approx: use palette
	palette := []color.RGBA{
		{0x4a, 0x90, 0xe2, 0xff}, {0x2e, 0xcc, 0x71, 0xff}, {0xe7, 0x4c, 0x3c, 0xff}, {0xf3, 0x9c, 0x12, 0xff},
		{0x9b, 0x59, 0xb6, 0xff}, {0x1a, 0xbc, 0x9c, 0xff}, {0xe6, 0x7e, 0x22, 0xff}, {0x34, 0x49, 0x5e, 0xff},
	}
	return palette[h%len(palette)]
}
