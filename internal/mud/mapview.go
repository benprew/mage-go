package mud

import (
	"strings"

	"github.com/mage/mage/internal/worlddata"
)

// Room box dimensions (fixed; every room is the same size).
const (
	roomW = 9 // includes walls
	roomH = 5 // includes walls
)

// cellInfo carries metadata about a tile for colorisation purposes.
type cellInfo struct {
	roomID string // ID of room whose box contains this tile; "" if none
	npcID  string // NPC ID placed at this tile; "" if none
}

// canvasDims computes the minimum grid dimensions that fit all room boxes.
func canvasDims(rooms []worlddata.RoomDef) (cols, rows int) {
	for _, r := range rooms {
		if c := r.MapX + roomW; c > cols {
			cols = c
		}
		if r := r.MapY + roomH; r > rows {
			rows = r
		}
	}
	return
}

// WorldMap builds the ASCII tile grid for the world.
// The canvas is sized to fit all rooms exactly. Returned as grid[y][x].
// npcAvail is passed for callers that need it (e.g. the colorise function);
// WorldMap places all NPCs regardless of availability.
func WorldMap(
	rooms []worlddata.RoomDef,
	currentRoomID string,
	npcAvail func(npcID string) bool,
) [][]rune {
	cols, rows := canvasDims(rooms)
	grid := makeGrid(cols, rows)

	// Draw room boxes.
	for _, room := range rooms {
		drawRoomBox(grid, room)
	}

	// Draw corridors (deduplicated — each pair drawn once).
	seen := map[string]bool{}
	roomMap := make(map[string]worlddata.RoomDef, len(rooms))
	for _, r := range rooms {
		roomMap[r.ID] = r
	}
	for _, room := range rooms {
		for dir, targetID := range room.Exits {
			key := canonKey(room.ID, targetID)
			if seen[key] {
				continue
			}
			seen[key] = true
			if target, ok := roomMap[targetID]; ok {
				drawCorridor(grid, room, target, dir)
			}
		}
	}

	// Place NPC glyphs (all NPCs, available or not; styling is caller's concern).
	for _, room := range rooms {
		col := 0
		for _, npcID := range room.NPCIDs {
			if col >= roomW-2 {
				break
			}
			npc, ok := worlddata.NPCByID(npcID)
			if !ok {
				col++
				continue
			}
			g := npc.Glyph
			if g == 0 {
				g = '?'
			}
			setCell(grid, room.MapX+1+col, room.MapY+2, g)
			col++
		}
	}

	// Place player at interior centre of current room.
	if cur, ok := roomMap[currentRoomID]; ok {
		setCell(grid, cur.MapX+4, cur.MapY+2, '@')
	}

	return grid
}

// buildCellInfo returns a map of grid position → cellInfo for all room tiles.
// Corridor tiles (#) between rooms are not included (they have no room ID).
func buildCellInfo(rooms []worlddata.RoomDef) map[[2]int]cellInfo {
	cells := map[[2]int]cellInfo{}
	for _, room := range rooms {
		// Mark every tile in the 9×5 box.
		for dy := range roomH {
			for dx := range roomW {
				x, y := room.MapX+dx, room.MapY+dy
				cells[[2]int{x, y}] = cellInfo{roomID: room.ID}
			}
		}
		// Overlay NPC positions (same placement logic as WorldMap).
		for col, npcID := range room.NPCIDs {
			if col >= roomW-2 {
				break
			}
			x := room.MapX + 1 + col
			y := room.MapY + 2
			ci := cells[[2]int{x, y}]
			ci.npcID = npcID
			cells[[2]int{x, y}] = ci
		}
	}
	return cells
}

// RenderMap renders a viewport of the world map as a string.
// The viewport is centred on currentRoom and clipped to viewW×viewH terminal cells.
// colorize is called once per visible tile and should return a (possibly ANSI-styled)
// string for that character; it may use x, y, and the rune for context.
func RenderMap(
	grid [][]rune,
	currentRoom worlddata.RoomDef,
	viewW, viewH int,
	colorize func(x, y int, r rune) string,
) string {
	if len(grid) == 0 {
		return ""
	}
	gridRows := len(grid)
	gridCols := len(grid[0])

	// Centre viewport on the interior centre of the current room.
	centerX := currentRoom.MapX + 4
	centerY := currentRoom.MapY + 2

	startX := centerX - viewW/2
	startY := centerY - viewH/2

	// Clamp to grid bounds.
	if startX < 0 {
		startX = 0
	}
	if startY < 0 {
		startY = 0
	}
	if startX+viewW > gridCols {
		startX = gridCols - viewW
	}
	if startY+viewH > gridRows {
		startY = gridRows - viewH
	}
	if startX < 0 {
		startX = 0
	}
	if startY < 0 {
		startY = 0
	}

	actualW := viewW
	if actualW > gridCols {
		actualW = gridCols
	}
	actualH := viewH
	if actualH > gridRows {
		actualH = gridRows
	}

	var sb strings.Builder
	for vy := range actualH {
		gy := startY + vy
		for vx := range actualW {
			gx := startX + vx
			var r rune = ' '
			if gy >= 0 && gy < gridRows && gx >= 0 && gx < gridCols {
				r = grid[gy][gx]
			}
			sb.WriteString(colorize(gx, gy, r))
		}
		if vy < actualH-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// ── internal helpers ──────────────────────────────────────────────────────────

func makeGrid(cols, rows int) [][]rune {
	grid := make([][]rune, rows)
	for y := range grid {
		grid[y] = make([]rune, cols)
		for x := range grid[y] {
			grid[y][x] = ' '
		}
	}
	return grid
}

func setCell(grid [][]rune, x, y int, r rune) {
	if len(grid) == 0 || y < 0 || y >= len(grid) {
		return
	}
	if x < 0 || x >= len(grid[y]) {
		return
	}
	grid[y][x] = r
}

// drawRoomBox renders the 9×5 box for a room: corners (+), walls (-/|), floor (.).
func drawRoomBox(grid [][]rune, room worlddata.RoomDef) {
	x, y := room.MapX, room.MapY

	// Horizontal walls.
	for dx := range roomW {
		setCell(grid, x+dx, y, '-')
		setCell(grid, x+dx, y+roomH-1, '-')
	}
	// Vertical walls.
	for dy := range roomH {
		setCell(grid, x, y+dy, '|')
		setCell(grid, x+roomW-1, y+dy, '|')
	}
	// Corners.
	setCell(grid, x, y, '+')
	setCell(grid, x+roomW-1, y, '+')
	setCell(grid, x, y+roomH-1, '+')
	setCell(grid, x+roomW-1, y+roomH-1, '+')
	// Interior floor.
	for dy := 1; dy < roomH-1; dy++ {
		for dx := 1; dx < roomW-1; dx++ {
			setCell(grid, x+dx, y+dy, '.')
		}
	}
}

// doorPos returns the wall tile where a door is placed for dir on room.
func doorPos(room worlddata.RoomDef, dir string) (int, int) {
	x, y := room.MapX, room.MapY
	switch dir {
	case "north":
		return x + 4, y
	case "south":
		return x + 4, y + roomH - 1
	case "east":
		return x + roomW - 1, y + 2
	case "west":
		return x, y + 2
	}
	return -1, -1
}

func reverseDir(dir string) string {
	switch dir {
	case "north":
		return "south"
	case "south":
		return "north"
	case "east":
		return "west"
	case "west":
		return "east"
	}
	return ""
}

// drawCorridor places + doors on each room wall and connects them with # tiles.
// When rooms are column- or row-aligned the corridor is straight; otherwise it
// draws an L-shape with a midpoint bend so any two rooms can be connected.
func drawCorridor(grid [][]rune, from, to worlddata.RoomDef, dir string) {
	fx, fy := doorPos(from, dir)
	tx, ty := doorPos(to, reverseDir(dir))
	if fx < 0 || tx < 0 {
		return
	}

	// Mark door positions on the room walls.
	setCell(grid, fx, fy, '+')
	setCell(grid, tx, ty, '+')

	if dir == "north" || dir == "south" {
		if fx == tx {
			// Straight vertical corridor.
			fillV(grid, fx, fy, ty)
		} else {
			// L-shaped: down/up to mid-row, across, then down/up to target.
			midY := (fy + ty) / 2
			fillV(grid, fx, fy, midY)
			fillH(grid, midY, fx, tx)
			fillV(grid, tx, midY, ty)
		}
	} else {
		if fy == ty {
			// Straight horizontal corridor.
			fillH(grid, fy, fx, tx)
		} else {
			// L-shaped: across to mid-col, down/up, then across to target.
			midX := (fx + tx) / 2
			fillH(grid, fy, fx, midX)
			fillV(grid, midX, fy, ty)
			fillH(grid, ty, midX, tx)
		}
	}
}

// fillV draws '#' on every empty (' ') cell from y1 to y2 inclusive at column x.
// Existing non-space tiles (doors, walls, floors) are left untouched.
func fillV(grid [][]rune, x, y1, y2 int) {
	lo, hi := y1, y2
	if lo > hi {
		lo, hi = hi, lo
	}
	for y := lo; y <= hi; y++ {
		if len(grid) > y && y >= 0 && x >= 0 && x < len(grid[y]) && grid[y][x] == ' ' {
			grid[y][x] = '#'
		}
	}
}

// fillH draws '#' on every empty (' ') cell from x1 to x2 inclusive at row y.
func fillH(grid [][]rune, y, x1, x2 int) {
	lo, hi := x1, x2
	if lo > hi {
		lo, hi = hi, lo
	}
	if y < 0 || y >= len(grid) {
		return
	}
	for x := lo; x <= hi; x++ {
		if x >= 0 && x < len(grid[y]) && grid[y][x] == ' ' {
			grid[y][x] = '#'
		}
	}
}

// canonKey returns an order-independent key for a room pair so each
// corridor is drawn only once.
func canonKey(a, b string) string {
	if a > b {
		a, b = b, a
	}
	return a + "→" + b
}
