package graph

// Viewport holds pan/zoom transform world <-> screen.
type Viewport struct {
	Scale   float64
	OffsetX float64
	OffsetY float64
}

func NewViewport() Viewport { return Viewport{Scale: 1, OffsetX: 0, OffsetY: 0} }

func (v Viewport) WorldToScreen(x, y float64) (float64, float64) {
	return x*v.Scale + v.OffsetX, y*v.Scale + v.OffsetY
}
func (v Viewport) ScreenToWorld(sx, sy float64) (float64, float64) {
	return (sx - v.OffsetX) / v.Scale, (sy - v.OffsetY) / v.Scale
}
func (v *Viewport) Pan(dx, dy float64) { v.OffsetX += dx; v.OffsetY += dy }
func (v *Viewport) Zoom(factor, cx, cy float64) {
	// zoom around center cx,cy in screen coords
	old := v.Scale
	v.Scale *= factor
	if v.Scale < 0.15 {
		v.Scale = 0.15
	}
	if v.Scale > 4 {
		v.Scale = 4
	}
	// adjust offset to keep cx,cy anchored: world = (screen - offset)/scale
	// newOffset = screen - world*newScale
	wx := (cx - v.OffsetX) / old
	wy := (cy - v.OffsetY) / old
	v.OffsetX = cx - wx*v.Scale
	v.OffsetY = cy - wy*v.Scale
}
