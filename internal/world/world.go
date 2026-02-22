// Package world manages the shared runtime state of the MUD world: which
// players are in which rooms, which NPCs have been defeated and when.
// It is initialised from the static worlddata definitions and is safe for
// concurrent use.
package world

import (
	"fmt"
	"sync"
	"time"

	"github.com/mage/mage/internal/collection"
	"github.com/mage/mage/internal/worlddata"
)

// World is the server-wide singleton holding all mutable world state.
type World struct {
	mu    sync.RWMutex
	rooms map[string]*room

	// defeatedNPCs[fingerprint][npcID] = time of last defeat
	defeatedNPCs map[string]map[string]time.Time
}

// room wraps a RoomDef with a mutable player list.
type room struct {
	def     worlddata.RoomDef
	players []string // fingerprints currently in this room
}

// New constructs a World from the worlddata definitions.
func New() *World {
	w := &World{
		rooms:        make(map[string]*room),
		defeatedNPCs: make(map[string]map[string]time.Time),
	}
	for _, def := range worlddata.Rooms() {
		d := def // copy
		w.rooms[def.ID] = &room{def: d}
	}
	return w
}

// Enter records that a player has moved into roomID, removing them from
// whatever room they were in before.
func (w *World) Enter(fingerprint, roomID string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, r := range w.rooms {
		for i, fp := range r.players {
			if fp == fingerprint {
				r.players = append(r.players[:i], r.players[i+1:]...)
				break
			}
		}
	}
	if r, ok := w.rooms[roomID]; ok {
		r.players = append(r.players, fingerprint)
	}
}

// Leave removes the player from whichever room they currently occupy.
func (w *World) Leave(fingerprint string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, r := range w.rooms {
		for i, fp := range r.players {
			if fp == fingerprint {
				r.players = append(r.players[:i], r.players[i+1:]...)
				return
			}
		}
	}
}

// PlayersInRoom returns the fingerprints of all players currently in roomID.
func (w *World) PlayersInRoom(roomID string) []string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	r, ok := w.rooms[roomID]
	if !ok {
		return nil
	}
	out := make([]string, len(r.players))
	copy(out, r.players)
	return out
}

// Room returns the RoomDef for roomID, and whether it was found.
func (w *World) Room(roomID string) (worlddata.RoomDef, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	r, ok := w.rooms[roomID]
	if !ok {
		return worlddata.RoomDef{}, false
	}
	return r.def, ok
}

// RecordDefeat marks npcID as having been defeated by fingerprint now.
func (w *World) RecordDefeat(fingerprint, npcID string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.defeatedNPCs[fingerprint] == nil {
		w.defeatedNPCs[fingerprint] = make(map[string]time.Time)
	}
	w.defeatedNPCs[fingerprint][npcID] = time.Now()
}

// NPCAvailable returns true if npcID can be challenged by fingerprint —
// either never defeated, or the respawn timer has elapsed.
func (w *World) NPCAvailable(fingerprint, npcID string) bool {
	npc, ok := worlddata.NPCByID(npcID)
	if !ok {
		return false
	}
	w.mu.RLock()
	defer w.mu.RUnlock()
	defeatedAt, wasDefeated := w.defeatedNPCs[fingerprint][npcID]
	if !wasDefeated {
		return true
	}
	if npc.RespawnMinutes == 0 {
		return false
	}
	return time.Since(defeatedAt) >= time.Duration(npc.RespawnMinutes)*time.Minute
}

// CanEnter checks whether fingerprint may enter roomID given their collection.
// Returns (true, "") if allowed, or (false, reason) if blocked.
func (w *World) CanEnter(fingerprint, roomID string, coll *collection.Collection) (bool, string) {
	def, ok := worlddata.RoomByID(roomID)
	if !ok {
		return false, "that room doesn't exist"
	}
	if def.GoldLock > 0 && coll.Gold < def.GoldLock {
		return false, fmt.Sprintf("the way is locked — you need %d gold (you have %d)", def.GoldLock, coll.Gold)
	}
	if def.BossLock != "" {
		w.mu.RLock()
		_, defeated := w.defeatedNPCs[fingerprint][def.BossLock]
		w.mu.RUnlock()
		if !defeated {
			npc, _ := worlddata.NPCByID(def.BossLock)
			return false, fmt.Sprintf("the way is barred until %s has been defeated", npc.Name)
		}
	}
	return true, ""
}

// PayGoldLock deducts the gold entrance fee for roomID from coll, if any.
// Call this only after CanEnter has confirmed the player can afford it.
func (w *World) PayGoldLock(roomID string, coll *collection.Collection) {
	def, ok := worlddata.RoomByID(roomID)
	if !ok || def.GoldLock == 0 {
		return
	}
	coll.SpendGold(def.GoldLock)
}
