package interactive

import (
	"fmt"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// HumanPlayer wraps BasePlayer for interactive TUI play. It channels all
// player decisions through the TUI rather than using silent defaults.
type HumanPlayer struct {
	*mage.BasePlayer
	toTUI       chan GameMsg
	fromTUI     chan PriorityAction
	gameLog     []string // kept in sync by RunGameLoop; used when building GameMsg
	choiceReqs  chan ChoiceRequest
	choiceResps chan ChoiceResponse
}

// NewHumanPlayer creates a new human player for the TUI.
func NewHumanPlayer(name string) *HumanPlayer {
	return &HumanPlayer{
		BasePlayer:  mage.NewBasePlayer(name),
		toTUI:       make(chan GameMsg, 1),
		fromTUI:     make(chan PriorityAction, 1),
		choiceReqs:  make(chan ChoiceRequest, 1),
		choiceResps: make(chan ChoiceResponse, 1),
	}
}

// NewHumanPlayerWithChannels creates a HumanPlayer wired to pre-allocated channels.
// Used by the SSH lobby so the TUI model can be wired before the game starts.
func NewHumanPlayerWithChannels(name string, toTUI chan GameMsg, fromTUI chan PriorityAction, choiceReqs chan ChoiceRequest, choiceResps chan ChoiceResponse) *HumanPlayer {
	return &HumanPlayer{
		BasePlayer:  mage.NewBasePlayer(name),
		toTUI:       toTUI,
		fromTUI:     fromTUI,
		choiceReqs:  choiceReqs,
		choiceResps: choiceResps,
	}
}

// ToTUI returns the read end of the game→TUI channel.
func (p *HumanPlayer) ToTUI() <-chan GameMsg { return p.toTUI }

// FromTUI returns the write end of the TUI→game channel.
func (p *HumanPlayer) FromTUI() chan<- PriorityAction { return p.fromTUI }

// DeclareAttackers implements the Player interface for human players by sending
// eligible attackers to the TUI and waiting for the player's selection.
func (p *HumanPlayer) DeclareAttackers(g *mage.Game) []uuid.UUID {
	eligible := getEligibleAttackers(g, p.PlayerID())
	if len(eligible) == 0 {
		return nil
	}
	idx := findPlayerIndex(g, p.PlayerID())
	state := SnapshotGameState(g, idx)
	p.toTUI <- GameMsg{
		State:   state,
		Prompt:  PromptDeclareAttackers,
		Options: attackerOptions(eligible),
		Log:     append([]string(nil), p.gameLog...),
	}
	action := <-p.fromTUI
	if action.Type == ActionSelectAttackers {
		return action.Attackers
	}
	return nil
}

// DeclareBlockers implements the Player interface for human players by sending
// eligible blockers to the TUI and waiting for the player's assignments.
func (p *HumanPlayer) DeclareBlockers(g *mage.Game) []mage.BlockAssignment {
	eligible := getEligibleBlockers(g, p.PlayerID())
	if len(eligible) == 0 {
		return nil
	}
	idx := findPlayerIndex(g, p.PlayerID())
	state := SnapshotGameState(g, idx)
	p.toTUI <- GameMsg{
		State:   state,
		Prompt:  PromptDeclareBlockers,
		Options: blockerOptions(g, p.PlayerID(), eligible),
		Log:     append([]string(nil), p.gameLog...),
	}
	action := <-p.fromTUI
	if action.Type == ActionSelectBlockers {
		return action.Blockers
	}
	return nil
}

// ChoiceRequests returns a channel the TUI should read for incoming choice requests.
func (p *HumanPlayer) ChoiceRequests() <-chan ChoiceRequest {
	return p.choiceReqs
}

// ChoiceResponses returns a channel the TUI should write choice responses to.
func (p *HumanPlayer) ChoiceResponses() chan<- ChoiceResponse {
	return p.choiceResps
}

func (p *HumanPlayer) ChooseMode(modes []string, reason string) int {
	opts := make([]ChoiceOption, len(modes))
	for i, m := range modes {
		opts[i] = ChoiceOption{Label: m}
	}
	p.choiceReqs <- ChoiceRequest{Type: ChoiceMode, Reason: reason, Options: opts}
	resp := <-p.choiceResps
	return resp.SelectedIndex
}

func (p *HumanPlayer) ChoosePermanent(candidates []*mage.Permanent, reason string, g mage.GameReader) *mage.Permanent {
	if len(candidates) == 0 {
		return nil
	}
	opts := make([]ChoiceOption, len(candidates))
	for i, c := range candidates {
		opts[i] = ChoiceOption{ID: c.ID(), Label: c.Name()}
	}
	p.choiceReqs <- ChoiceRequest{Type: ChoicePermanent, Reason: reason, Options: opts}
	resp := <-p.choiceResps
	if len(resp.SelectedIDs) > 0 {
		for _, c := range candidates {
			if c.ID() == resp.SelectedIDs[0] {
				return c
			}
		}
	}
	return candidates[0]
}

func (p *HumanPlayer) ChooseCardsFromHand(amount int, reason string, g mage.GameReader) []mage.Card {
	hand := p.Hand()
	if amount <= 0 || len(hand) == 0 {
		return nil
	}
	if amount > len(hand) {
		amount = len(hand)
	}
	opts := make([]ChoiceOption, len(hand))
	for i, c := range hand {
		opts[i] = ChoiceOption{ID: c.ID(), Label: c.Name() + " " + c.ManaCost().String()}
	}
	p.choiceReqs <- ChoiceRequest{Type: ChoiceCardsFromHand, Reason: reason, Amount: amount, Options: opts}
	resp := <-p.choiceResps
	idSet := make(map[uuid.UUID]bool, len(resp.SelectedIDs))
	for _, id := range resp.SelectedIDs {
		idSet[id] = true
	}
	var result []mage.Card
	for _, c := range hand {
		if idSet[c.ID()] {
			result = append(result, c)
		}
	}
	if len(result) > amount {
		result = result[:amount]
	}
	return result
}

func (p *HumanPlayer) ChooseManaColor(reason string) core.Color {
	opts := []ChoiceOption{
		{Label: "White", Color: core.White},
		{Label: "Blue", Color: core.Blue},
		{Label: "Black", Color: core.Black},
		{Label: "Red", Color: core.Red},
		{Label: "Green", Color: core.Green},
	}
	p.choiceReqs <- ChoiceRequest{Type: ChoiceManaColor, Reason: reason, Options: opts}
	resp := <-p.choiceResps
	return resp.SelectedColor
}

func (p *HumanPlayer) ChooseCardFromLibrary(candidates []mage.Card, reason string, g mage.GameReader) mage.Card {
	if len(candidates) == 0 {
		return nil
	}
	opts := make([]ChoiceOption, len(candidates))
	for i, c := range candidates {
		opts[i] = ChoiceOption{ID: c.ID(), Label: c.Name() + " " + c.ManaCost().String()}
	}
	p.choiceReqs <- ChoiceRequest{Type: ChoiceCardFromLibrary, Reason: reason, Options: opts}
	resp := <-p.choiceResps
	if len(resp.SelectedIDs) > 0 {
		for _, c := range candidates {
			if c.ID() == resp.SelectedIDs[0] {
				return c
			}
		}
	}
	return candidates[0]
}

func (p *HumanPlayer) ChooseMayAbility(description string) bool {
	p.choiceReqs <- ChoiceRequest{
		Type:   ChoiceMay,
		Reason: description,
		Options: []ChoiceOption{
			{Label: "Yes"},
			{Label: "No"},
		},
	}
	resp := <-p.choiceResps
	return resp.Accepted
}

func (p *HumanPlayer) ChooseNumber(min, max int, reason string) int {
	opts := make([]ChoiceOption, max-min+1)
	for i := min; i <= max; i++ {
		opts[i-min] = ChoiceOption{Label: fmt.Sprintf("%d", i)}
	}
	p.choiceReqs <- ChoiceRequest{Type: ChoiceNumber, Reason: reason, Options: opts}
	resp := <-p.choiceResps
	n := resp.SelectedIndex + min
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}
