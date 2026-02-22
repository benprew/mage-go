package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	gossh "golang.org/x/crypto/ssh"

	cssh "github.com/charmbracelet/ssh"
	"github.com/mage/mage/internal/collection"
	"github.com/mage/mage/internal/worlddata"
)

// PlayerState is the per-session state for a connected player.
type PlayerState struct {
	Fingerprint string
	Username    string
	RoomID      string
	Collection  *collection.Collection
}

// locationFile returns the path for storing a player's last room.
func locationFile(dataDir, fingerprint string) string {
	return filepath.Join(dataDir, "players", fingerprint, "location.json")
}

type locationData struct {
	RoomID string `json:"room_id"`
}

// loadPlayer loads or creates a player's persistent state.
func loadPlayer(dataDir, fingerprint, username string) (*PlayerState, error) {
	coll, err := collection.Load(dataDir, fingerprint)
	if err != nil {
		return nil, err
	}

	ps := &PlayerState{
		Fingerprint: fingerprint,
		Username:    username,
		Collection:  coll,
		RoomID:      worlddata.StartRoom(),
	}

	// Load last known location.
	locPath := locationFile(dataDir, fingerprint)
	data, err := os.ReadFile(locPath)
	if err == nil {
		var loc locationData
		if json.Unmarshal(data, &loc) == nil && loc.RoomID != "" {
			if _, ok := worlddata.RoomByID(loc.RoomID); ok {
				ps.RoomID = loc.RoomID
			}
		}
	}

	return ps, nil
}

// save persists the player's collection and current location.
func (ps *PlayerState) save(dataDir string) error {
	if err := ps.Collection.Save(dataDir, ps.Fingerprint); err != nil {
		return err
	}

	locPath := locationFile(dataDir, ps.Fingerprint)
	if err := os.MkdirAll(filepath.Dir(locPath), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(locationData{RoomID: ps.RoomID}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(locPath, data, 0600)
}

// fingerprint returns the SHA256 fingerprint of the session's public key,
// falling back to the username if no public key is present.
func fingerprintFromSession(sess cssh.Session) string {
	pk := sess.PublicKey()
	if pk == nil {
		return "user:" + sess.User()
	}
	return gossh.FingerprintSHA256(pk)
}
