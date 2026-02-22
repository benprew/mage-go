package mud

import (
	"math/rand"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mage/mage/internal/collection"
	"github.com/mage/mage/internal/world"
	"github.com/mage/mage/internal/worlddata"
)

type mudState int

const (
	stateRoom         mudState = iota // viewing the current room
	stateExamine                      // reading NPC description
	stateChallenge                    // confirm fight + ante option
	stateLoot                         // post-combat rewards screen
	stateInventory                    // collection summary
	stateStarterChest                 // new player picks starter archetype
	stateSearch                       // searching for hidden treasure
)

// refreshMsg is sent by tea.Tick to refresh the room view (for other-player presence).
type refreshMsg struct{}

// Model is the MUD world navigation bubbletea model.
type Model struct {
	world       *world.World
	coll        *collection.Collection
	fingerprint string
	username    string

	roomID string
	state  mudState

	// stateRoom: cursor over NPCs
	npcCursor int

	// stateExamine / stateChallenge
	selectedNPC *worlddata.NPCDef
	anteEnabled bool

	// stateChallenge: deck selection (for future multi-deck support; currently just confirms)
	// (placeholder)

	// stateLoot: result to display
	loot *LootResult

	// stateStarterChest: which archetype is highlighted
	chestCursor int

	// stateSearch: result lines
	searchLines []string

	// stateInventory: scroll offset
	invScroll int

	// Error / status message shown at bottom of room view
	statusMsg string

	// Final result — populated when the model is about to tea.Quit
	result Result

	width, height int
}

// NewModel creates a mud Model in the start room.
// Pass pendingLoot != nil to show the loot screen immediately (after combat).
func NewModel(w *world.World, fingerprint, username, roomID string, coll *collection.Collection, pendingLoot *LootResult) Model {
	m := Model{
		world:       w,
		coll:        coll,
		fingerprint: fingerprint,
		username:    username,
		roomID:      roomID,
	}

	// New player with no decks gets the starter chest flow.
	if !coll.HasDeck() {
		roomDef, ok := worlddata.RoomByID(roomID)
		if ok && roomDef.StartChest {
			m.state = stateStarterChest
			return m
		}
	}

	if pendingLoot != nil {
		m.loot = pendingLoot
		m.state = stateLoot
		return m
	}

	m.state = stateRoom
	return m
}

func (m Model) Init() tea.Cmd {
	return tickRefresh()
}

func tickRefresh() tea.Cmd {
	return tea.Tick(1*time.Second, func(_ time.Time) tea.Msg {
		return refreshMsg{}
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case refreshMsg:
		// Just re-render to pick up any new players in the room.
		return m, tickRefresh()

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

// FinalResult returns the action result that was set before tea.Quit.
func (m Model) FinalResult() Result {
	return m.result
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case stateRoom:
		return m.handleRoomKey(msg)
	case stateExamine:
		return m.handleExamineKey(msg)
	case stateChallenge:
		return m.handleChallengeKey(msg)
	case stateLoot:
		return m.handleLootKey(msg)
	case stateInventory:
		return m.handleInventoryKey(msg)
	case stateStarterChest:
		return m.handleChestKey(msg)
	case stateSearch:
		return m.handleSearchKey(msg)
	}
	return m, nil
}

// ── stateRoom ────────────────────────────────────────────────────────────────

func (m Model) handleRoomKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.statusMsg = ""
	room, ok := worlddata.RoomByID(m.roomID)
	if !ok {
		return m, nil
	}

	switch msg.String() {
	case "ctrl+c", "q":
		m.result = Result{Action: "quit", RoomID: m.roomID}
		return m, tea.Quit

	case "up", "k":
		if m.npcCursor > 0 {
			m.npcCursor--
		}

	case "down", "j":
		if m.npcCursor < len(room.NPCIDs)-1 {
			m.npcCursor++
		}

	case "n", "s", "e", "w", "h", "l",
		"north", "south", "east", "west":
		dir := directionFromKey(msg.String())
		if targetRoom, ok := room.Exits[dir]; ok {
			canEnter, reason := m.world.CanEnter(m.fingerprint, targetRoom, m.coll)
			if !canEnter {
				m.statusMsg = reason
				return m, nil
			}
			// Pay gold lock if applicable.
			m.world.PayGoldLock(targetRoom, m.coll)
			m.roomID = targetRoom
			m.world.Enter(m.fingerprint, targetRoom)
			m.npcCursor = 0
		} else {
			m.statusMsg = "No exit in that direction."
		}

	case "t":
		// Talk to the currently highlighted NPC.
		npcs := availableNPCs(room, m.world, m.fingerprint)
		if len(npcs) > 0 && m.npcCursor < len(npcs) {
			npc := npcs[m.npcCursor]
			if len(npc.Dialog) > 0 {
				m.statusMsg = npc.Dialog[rand.Intn(len(npc.Dialog))]
			}
		}

	case "x":
		// Examine highlighted NPC.
		npcs := availableNPCs(room, m.world, m.fingerprint)
		if len(npcs) > 0 && m.npcCursor < len(npcs) {
			m.selectedNPC = &worlddata.NPCDef{}
			*m.selectedNPC = npcs[m.npcCursor]
			m.state = stateExamine
		}

	case "f", "enter":
		// Fight highlighted NPC.
		npcs := availableNPCs(room, m.world, m.fingerprint)
		if len(npcs) == 0 {
			m.statusMsg = "There is no one here to challenge."
			return m, nil
		}
		if m.npcCursor >= len(npcs) {
			m.npcCursor = 0
		}
		npc := npcs[m.npcCursor]
		if !m.coll.HasDeck() {
			m.statusMsg = "You have no deck. Open the deckbuilder (d) first."
			return m, nil
		}
		m.selectedNPC = &worlddata.NPCDef{}
		*m.selectedNPC = npc
		m.anteEnabled = npc.AnteForced
		m.state = stateChallenge

	case "r":
		// Rummage/search for hidden treasure.
		m.doSearch(room)
		m.state = stateSearch

	case "i":
		m.state = stateInventory
		m.invScroll = 0

	case "d":
		m.result = Result{Action: "deckbuilder", RoomID: m.roomID}
		return m, tea.Quit
	}

	return m, nil
}

func directionFromKey(k string) string {
	switch k {
	case "n", "north":
		return "north"
	case "s", "south":
		return "south"
	case "e", "east", "l":
		return "east"
	case "w", "west", "h":
		return "west"
	}
	return k
}

func availableNPCs(room worlddata.RoomDef, w *world.World, fingerprint string) []worlddata.NPCDef {
	var out []worlddata.NPCDef
	for _, id := range room.NPCIDs {
		if w.NPCAvailable(fingerprint, id) {
			if npc, ok := worlddata.NPCByID(id); ok {
				out = append(out, npc)
			}
		}
	}
	return out
}

// ── stateExamine ──────────────────────────────────────────────────────────────

func (m Model) handleExamineKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "x", "q":
		m.state = stateRoom
		m.selectedNPC = nil
	case "f", "enter":
		if m.selectedNPC != nil {
			if !m.coll.HasDeck() {
				m.statusMsg = "You have no deck. Open the deckbuilder (d) first."
				m.state = stateRoom
				return m, nil
			}
			m.anteEnabled = m.selectedNPC.AnteForced
			m.state = stateChallenge
		}
	}
	return m, nil
}

// ── stateChallenge ─────────────────────────────────────────────────────────────

func (m Model) handleChallengeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = stateRoom
		m.selectedNPC = nil
	case "a":
		// Toggle ante proposal (only if NPC doesn't force it).
		if m.selectedNPC != nil && !m.selectedNPC.AnteForced {
			m.anteEnabled = !m.anteEnabled
		}
	case "enter", "f", "y":
		if m.selectedNPC != nil {
			m.result = Result{
				Action:      "fight_npc",
				RoomID:      m.roomID,
				NPC:         m.selectedNPC,
				AnteEnabled: m.anteEnabled,
			}
			return m, tea.Quit
		}
	case "n":
		m.state = stateRoom
		m.selectedNPC = nil
	}
	return m, nil
}

// ── stateLoot ─────────────────────────────────────────────────────────────────

func (m Model) handleLootKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Any key dismisses the loot screen.
	m.loot = nil
	m.state = stateRoom
	return m, nil
}

// ── stateInventory ────────────────────────────────────────────────────────────

func (m Model) handleInventoryKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "i", "q":
		m.state = stateRoom
	case "up", "k":
		if m.invScroll > 0 {
			m.invScroll--
		}
	case "down", "j":
		m.invScroll++
	}
	return m, nil
}

// ── stateStarterChest ─────────────────────────────────────────────────────────

func (m Model) handleChestKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	archetypes := worlddata.StarterArchetypes
	switch msg.String() {
	case "ctrl+c":
		m.result = Result{Action: "quit", RoomID: m.roomID}
		return m, tea.Quit
	case "up", "k":
		if m.chestCursor > 0 {
			m.chestCursor--
		}
	case "down", "j":
		if m.chestCursor < len(archetypes)-1 {
			m.chestCursor++
		}
	case "enter":
		if m.chestCursor < len(archetypes) {
			arch := archetypes[m.chestCursor]
			m.coll.AddDeckEntries(arch.Deck)
			// Create an active deck from the archetype.
			m.coll.Decks = append(m.coll.Decks, collection.SavedDeck{
				Name: arch.Name,
				Main: arch.Deck,
			})
			m.coll.ActiveDeck = 0
			m.state = stateRoom
			m.statusMsg = "You take the deck from the chest. Your journey begins."
		}
	}
	return m, nil
}

// ── stateSearch ───────────────────────────────────────────────────────────────

func (m Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.state = stateRoom
	return m, nil
}

// doSearch checks hidden treasures in the room and awards any found cards.
func (m *Model) doSearch(room worlddata.RoomDef) {
	m.searchLines = nil
	found := false
	for _, t := range room.Hidden {
		roll := 1
		if t.SearchDC > 0 {
			roll = rand.Intn(t.SearchDC) + 1
		}
		if roll == 1 || t.SearchDC == 0 {
			if t.Trap {
				m.searchLines = append(m.searchLines, "You find something, but it appears to be trapped.")
				m.searchLines = append(m.searchLines, "(Traps are not yet implemented — loot awarded anyway.)")
			}
			m.coll.Add(t.Cards)
			for _, c := range t.Cards {
				m.searchLines = append(m.searchLines, "Found: "+c)
			}
			found = true
		}
	}
	if !found {
		m.searchLines = []string{"You search carefully but find nothing."}
	}
}
