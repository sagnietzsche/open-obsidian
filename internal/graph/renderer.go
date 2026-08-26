package graph

// This file is the extension point for the custom graphical logic layer.
// The active implementation is in widget.go:graphRenderer which owns the Fyne
// WidgetRenderer lifecycle (Layout/MinSize/Refresh/Objects/Destroy) and issues
// canvas.Circle/Line/Text primitives transformed via Viewport.
//
// For future enhancement:
//   - Add minimap overlay (second canvas.Rectangle + scaled node dots)
//   - Edge arrows: canvas.Line with polygon arrowhead via canvas.NewPolygon
//   - Group hulls: concave hull around tag cluster using canvas.NewLine loop
//   - Export SVG: iterate nodes/edges world coords -> svg writer
// See layout.go for force simulation and viewport.go for pan/zoom.
