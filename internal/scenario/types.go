// Package scenario provides types and utilities for running AI vs AI scenario
// tests using Shandalar rogue decks.
package scenario

// RogueDeck is a parsed Shandalar rogue deck.
type RogueDeck struct {
	Name       string
	Level      int
	MainCards  []DeckEntry
	Sideboard  []DeckEntry
	SourceFile string
}

// DeckEntry is a card name and count.
type DeckEntry struct {
	Name  string
	Count int
}

// GameResult is one JSONL record per completed game.
type GameResult struct {
	GameID         int      `json:"game_id"`
	Seed           int64    `json:"seed"`
	DeckA          string   `json:"deck_a"`
	DeckB          string   `json:"deck_b"`
	DeckAFile      string   `json:"deck_a_file"`
	DeckBFile      string   `json:"deck_b_file"`
	AIPersonalityA string   `json:"ai_personality_a"`
	AIPersonalityB string   `json:"ai_personality_b"`
	Winner         string   `json:"winner"`
	WinReason      string   `json:"win_reason"`
	FinalLifeA     int      `json:"final_life_a"`
	FinalLifeB     int      `json:"final_life_b"`
	FinalTurn      int      `json:"final_turn"`
	TotalActions   int      `json:"total_actions"`
	PanicMsg       string   `json:"panic_msg,omitempty"`
	PanicStack     string   `json:"panic_stack,omitempty"`
	CardsSkippedA  []string `json:"cards_skipped_a,omitempty"`
	CardsSkippedB  []string `json:"cards_skipped_b,omitempty"`
	Anomalies      []string `json:"anomalies,omitempty"`
	Duration       string   `json:"duration"`
}
