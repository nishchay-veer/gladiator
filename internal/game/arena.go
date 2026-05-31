package game

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
)

type Arena struct {
	Width  int
	Height int
	Tiles  []Tile
	Spawn  Point
}

var localLayout = []string{
	"########################################",
	"#P.....................................#",
	"#......##...................##.........#",
	"#......##...................##.........#",
	"#......................................#",
	"#................##....................#",
	"#................##....................#",
	"#......................................#",
	"#...........##.........................#",
	"#............#.........................#",
	"#.....................#................#",
	"#.....................#..........##....#",
	"#.................................#....#",
	"#......................................#",
	"#......................................#",
	"#......................................#",
	"########################################",
}

func NewArena(rows []string) (Arena, error) {
	if len(rows) == 0 {
		return Arena{}, fmt.Errorf("arena has no rows")
	}

	width := len(rows[0])
	if width == 0 {
		return Arena{}, fmt.Errorf("arena has empty first row")
	}

	arena := Arena{
		Width:  width,
		Height: len(rows),
		Tiles:  make([]Tile, 0, width*len(rows)),
		Spawn:  Point{X: -1, Y: -1},
	}

	for y, row := range rows {
		if len(row) != width {
			return Arena{}, fmt.Errorf("arena row %d has width %d, expected %d", y, len(row), width)
		}

		for x := 0; x < len(row); x++ {
			switch row[x] {
			case '#':
				arena.Tiles = append(arena.Tiles, Wall)
			case '.':
				arena.Tiles = append(arena.Tiles, Floor)
			case 'P':
				if arena.Spawn.X >= 0 {
					return Arena{}, fmt.Errorf("arena has multiple player spawns")
				}
				arena.Spawn = Point{X: x, Y: y}
				arena.Tiles = append(arena.Tiles, Floor)
			default:
				return Arena{}, fmt.Errorf("arena contains unsupported tile %q at %d,%d", row[x], x, y)
			}
		}
	}

	if arena.Spawn.X < 0 {
		return Arena{}, fmt.Errorf("arena is missing player spawn")
	}

	return arena, nil
}

func (a Arena) TileAt(p Point) Tile {
	if p.X < 0 || p.Y < 0 || p.X >= a.Width || p.Y >= a.Height {
		return Wall
	}
	return a.Tiles[p.Y*a.Width+p.X]
}

func (a Arena) Hash() uint64 {
	h := fnv.New64a()
	writeArenaInt(h, a.Width)
	writeArenaInt(h, a.Height)
	writeArenaInt(h, a.Spawn.X)
	writeArenaInt(h, a.Spawn.Y)
	for _, tile := range a.Tiles {
		writeArenaInt(h, int(tile))
	}
	return h.Sum64()
}

func (a Arena) IsBlocked(p Point) bool {
	return a.TileAt(p) == Wall
}

func (a Arena) ClearLine(from, to Point) (Direction, bool) {
	switch {
	case from.X == to.X && from.Y < to.Y:
		return Down, a.clearVertical(from.X, from.Y+1, to.Y)
	case from.X == to.X && from.Y > to.Y:
		return Up, a.clearVertical(from.X, to.Y, from.Y-1)
	case from.Y == to.Y && from.X < to.X:
		return Right, a.clearHorizontal(from.Y, from.X+1, to.X)
	case from.Y == to.Y && from.X > to.X:
		return Left, a.clearHorizontal(from.Y, to.X, from.X-1)
	default:
		return Right, false
	}
}

func (a Arena) clearHorizontal(y, startX, endX int) bool {
	for x := startX; x <= endX; x++ {
		if a.IsBlocked(Point{X: x, Y: y}) {
			return false
		}
	}
	return true
}

func (a Arena) clearVertical(x, startY, endY int) bool {
	for y := startY; y <= endY; y++ {
		if a.IsBlocked(Point{X: x, Y: y}) {
			return false
		}
	}
	return true
}

func writeArenaInt(h interface{ Write([]byte) (int, error) }, value int) {
	var buffer [8]byte
	binary.BigEndian.PutUint64(buffer[:], uint64(int64(value)))
	_, _ = h.Write(buffer[:])
}
