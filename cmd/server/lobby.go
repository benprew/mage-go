package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/mage/mage/internal/tui"
	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/interactive"
	"github.com/mage/mage/pkg/mage/interactive/ai"

	_ "github.com/mage/mage/cards" // register all card sets
)

// PlayerSession holds the channels used to wire a player's TUI to the game loop.
type PlayerSession struct {
	Name        string
	DeckEntries []tui.DeckEntry
	FromGame    chan interactive.GameMsg       // game → TUI
	ToGame      chan interactive.PriorityAction // TUI → game
	ChoiceReqs  chan interactive.ChoiceRequest  // game → TUI choices
	ChoiceResps chan interactive.ChoiceResponse // TUI → game choices
}

func newPlayerSession(name string, entries []tui.DeckEntry) *PlayerSession {
	return &PlayerSession{
		Name:        name,
		DeckEntries: entries,
		FromGame:    make(chan interactive.GameMsg, 1),
		ToGame:      make(chan interactive.PriorityAction, 1),
		ChoiceReqs:  make(chan interactive.ChoiceRequest, 1),
		ChoiceResps: make(chan interactive.ChoiceResponse, 1),
	}
}

// GameSlot represents a waiting or in-progress game.
type GameSlot struct {
	ID    string
	isAI  bool
	p1    *PlayerSession
	p2    *PlayerSession  // nil while waiting for second human player
	ready chan struct{}    // closed when p2 joins
	done  chan struct{}    // closed when game ends
}

// SlotInfo is a safe copy of open-slot metadata for lobby display.
type SlotInfo struct {
	ID   string
	Name string // p1's name
}

// Lobby manages concurrent game slots.
type Lobby struct {
	mu     sync.Mutex
	slots  []*GameSlot
	nextID int
}

func newLobby() *Lobby {
	return &Lobby{}
}

// CreateSlot creates a new pvp slot with p1 waiting for an opponent.
func (l *Lobby) CreateSlot(sess *PlayerSession) *GameSlot {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.nextID++
	slot := &GameSlot{
		ID:    fmt.Sprintf("%d", l.nextID),
		isAI:  false,
		p1:    sess,
		ready: make(chan struct{}),
		done:  make(chan struct{}),
	}
	l.slots = append(l.slots, slot)
	return slot
}

// JoinSlot adds a second player to an existing slot and starts the game.
// Returns an error if the slot doesn't exist or is already full.
func (l *Lobby) JoinSlot(slotID string, sess *PlayerSession) (*GameSlot, error) {
	l.mu.Lock()
	for _, slot := range l.slots {
		if slot.ID == slotID && slot.p2 == nil && !slot.isAI {
			slot.p2 = sess
			l.mu.Unlock()
			close(slot.ready)
			go l.startPvPGame(slot)
			return slot, nil
		}
	}
	l.mu.Unlock()
	return nil, fmt.Errorf("slot %q not found or already full", slotID)
}

// ListSlots returns a snapshot of open (waiting) pvp slots.
func (l *Lobby) ListSlots() []SlotInfo {
	l.mu.Lock()
	defer l.mu.Unlock()
	var infos []SlotInfo
	for _, slot := range l.slots {
		if !slot.isAI && slot.p2 == nil {
			infos = append(infos, SlotInfo{ID: slot.ID, Name: slot.p1.Name})
		}
	}
	return infos
}

// CancelSlot removes a slot that the creating player is abandoning.
func (l *Lobby) CancelSlot(slot *GameSlot) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i, s := range l.slots {
		if s == slot {
			l.slots = append(l.slots[:i], l.slots[i+1:]...)
			return
		}
	}
}

// removeSlot removes a completed slot from the list.
func (l *Lobby) removeSlot(slot *GameSlot) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i, s := range l.slots {
		if s == slot {
			l.slots = append(l.slots[:i], l.slots[i+1:]...)
			return
		}
	}
}

// StartAIGame starts a game for a single human player against the AI.
// The game runs in a goroutine; the PlayerSession channels are wired to the
// game loop immediately so the TUI can start as soon as startAIGame returns.
func (l *Lobby) StartAIGame(sess *PlayerSession) {
	go func() {
		human := interactive.NewHumanPlayerWithChannels(sess.Name, sess.FromGame, sess.ToGame, sess.ChoiceReqs, sess.ChoiceResps)
		aiPlayer := ai.NewAggroAI("AI")

		humanCards := tui.BuildDeck(sess.DeckEntries, human.PlayerID())
		for _, c := range humanCards {
			human.AddToLibrary(c)
		}
		aiEntries := tui.Archetypes[1].Entries
		aiCards := tui.BuildDeck(aiEntries, aiPlayer.PlayerID())
		for _, c := range aiCards {
			aiPlayer.AddToLibrary(c)
		}

		g := mage.NewGame(human, aiPlayer)
		tui.DrawOpeningHand(human)
		tui.DrawOpeningHand(aiPlayer)

		const aiPause = 400 * time.Millisecond
		interactive.RunGameLoop(g, 0, aiPause)
	}()
}

// startPvPGame is called in a goroutine once both players have joined.
func (l *Lobby) startPvPGame(slot *GameSlot) {
	defer func() {
		close(slot.done)
		l.removeSlot(slot)
	}()

	p1 := slot.p1
	p2 := slot.p2

	h1 := interactive.NewHumanPlayerWithChannels(p1.Name, p1.FromGame, p1.ToGame, p1.ChoiceReqs, p1.ChoiceResps)
	h2 := interactive.NewHumanPlayerWithChannels(p2.Name, p2.FromGame, p2.ToGame, p2.ChoiceReqs, p2.ChoiceResps)

	h1Cards := tui.BuildDeck(p1.DeckEntries, h1.PlayerID())
	for _, c := range h1Cards {
		h1.AddToLibrary(c)
	}
	h2Cards := tui.BuildDeck(p2.DeckEntries, h2.PlayerID())
	for _, c := range h2Cards {
		h2.AddToLibrary(c)
	}

	g := mage.NewGame(h1, h2)
	tui.DrawOpeningHand(h1)
	tui.DrawOpeningHand(h2)

	channels := [2]interactive.PlayerChannels{
		{
			ToPlayer:    p1.FromGame,
			FromPlayer:  p1.ToGame,
			ChoiceReqs:  p1.ChoiceReqs,
			ChoiceResps: p1.ChoiceResps,
		},
		{
			ToPlayer:    p2.FromGame,
			FromPlayer:  p2.ToGame,
			ChoiceReqs:  p2.ChoiceReqs,
			ChoiceResps: p2.ChoiceResps,
		},
	}
	interactive.RunMultiplayerGameLoop(g, channels)
}
