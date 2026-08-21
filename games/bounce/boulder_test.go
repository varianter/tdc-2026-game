package bounce

import "testing"

// The boulder's silhouette is smaller than its bounding box, so collision has
// to follow the drawn span. These checks pin that down: a hitbox one pixel
// clear of the drawn rock must miss, and one overlapping it must hit.
func TestBoulderHitsFollowsDrawnSilhouette(t *testing.T) {
	for _, size := range []struct{ w, h float64 }{{20, 20}, {35, 60}, {55, 85}, {15, 15}} {
		o := newObstacle(1000, 40, size.w, size.h)
		rows := o.rows()

		for i := 0; i < rows; i++ {
			x0, x1 := o.spanAt(i)
			y := o.y + float64(i) + 0.25 // inside scanline i only
			const probe = 0.5

			if o.hits(x0-probe-0.01, y, probe, 0.5) {
				t.Fatalf("w=%v h=%v row %d: hit just left of the drawn span", size.w, size.h, i)
			}
			if o.hits(x1+0.01, y, probe, 0.5) {
				t.Fatalf("w=%v h=%v row %d: hit just right of the drawn span", size.w, size.h, i)
			}
			if !o.hits(x0+0.1, y, probe, 0.5) {
				t.Fatalf("w=%v h=%v row %d: missed inside the drawn span", size.w, size.h, i)
			}
		}

		// Corners of the bounding box are empty vinyl for every boulder: the
		// silhouette is at its narrowest at the top and bottom rows.
		if o.hits(o.x, o.y, 1, 1) || o.hits(o.x+o.w-1, o.y+o.h-1, 1, 1) {
			t.Fatalf("w=%v h=%v: bounding-box corner counted as a hit", size.w, size.h)
		}
	}
}
