package geometry

import (
	"math"

	"github.com/RevoTale/esp32-e-paper-manager/engine/style"
)

// Tiles bounds iteration before converting any floating-point index to int.
// Tiny positive CSS lengths cannot cause an unbounded loop or allocation.
func Tiles(tile, clip Rect, repeat string, budget int) ([]Rect, error) {
	if !validTileInput(tile, clip, budget) {
		return nil, style.ErrValue
	}
	if clip.Empty() || repeat == "no-repeat" && tile.Empty() {
		return nil, nil
	}
	if tile.Empty() || !validRepeat(repeat) {
		return nil, style.ErrValue
	}
	return expandTiles(tile, clip, repeat, budget)
}

func validTileInput(tile, clip Rect, budget int) bool {
	return tile.Valid() && clip.Valid() && budget > 0 && budget <= 65536
}

func expandTiles(tile, clip Rect, repeat string, budget int) ([]Rect, error) {
	x0, nx := tileRange(tile.X, tile.Width, clip.X, clip.Right(), repeat == "repeat" || repeat == "repeat-x")
	y0, ny := tileRange(tile.Y, tile.Height, clip.Y, clip.Bottom(), repeat == "repeat" || repeat == "repeat-y")
	if !tileBudget(nx, ny, budget) || !bounded(x0) || !bounded(y0) {
		return nil, style.ErrValue
	}
	result := make([]Rect, 0, int(nx*ny))
	for row := range int(ny) {
		for column := range int(nx) {
			candidate := Rect{x0 + float64(column)*tile.Width, y0 + float64(row)*tile.Height, tile.Width, tile.Height}
			if !candidate.Valid() {
				return nil, style.ErrValue
			}
			if !candidate.Intersect(clip).Empty() {
				result = append(result, candidate)
			}
		}
	}
	return result, nil
}

func validRepeat(value string) bool {
	return value == "repeat" || value == "repeat-x" || value == "repeat-y" || value == "no-repeat"
}

func tileRange(origin, size, minimum, maximum float64, repeat bool) (float64, float64) {
	if !repeat {
		return origin, 1
	}
	start := math.Floor((minimum - origin) / size)
	end := math.Ceil((maximum - origin) / size)
	return origin + start*size, end - start
}

func tileBudget(nx, ny float64, budget int) bool {
	return nx > 0 && ny > 0 && nx <= float64(budget) && ny <= float64(budget)/nx
}
