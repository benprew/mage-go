package mage

import (
	"fmt"
	"math/rand"

	"github.com/google/uuid"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// BandFormer is an optional interface for players that declare attacking bands.
// The game engine calls GetBandFormations after attackers are declared.
type BandFormer interface {
	GetBandFormations(turn int, g *Game) [][]uuid.UUID
}

// BandingDamageDistributor is an optional interface for players that control
// how incoming combat damage is distributed across banded creatures.
// The attacking player uses this for an attacking band; the defending player
// uses this for a blocking band.
type BandingDamageDistributor interface {
	GetBandingDamageDistribution(members []*Permanent) map[uuid.UUID]int
}

// CombatDamageAssigner is an optional interface for the attacking player to
// (a) order multiple blockers for damage assignment (CR 510.1c, CR 702.19b)
// and (b) divide the attacker's combat damage among them, subject to the
// "lethal damage to each blocker before the next" constraint. Returning a
// nil order or distribution falls back to the engine default (BlockerIDs
// order, lethal-first greedy).
type CombatDamageAssigner interface {
	GetBlockerOrder(attacker *Permanent, blockers []*Permanent) []uuid.UUID
	GetCombatDamageAssignment(attacker *Permanent, blockers []*Permanent, totalPower int) map[uuid.UUID]int
}

// BlockAssignment represents a single blocker-attacker pair.
type BlockAssignment struct {
	BlockerID  uuid.UUID
	AttackerID uuid.UUID
}

// Player interface for game players.
type Player interface {
	PlayerID() uuid.UUID
	Name() string
	Life() int
	SetLife(int)
	StartingLife() int
	SetStartingLife(int)
	GainLife(int)
	LoseLife(int)
	IsAlive() bool
	DrewFromEmpty() bool
	ClearDrewFromEmpty()
	SetLost()
	Hand() []Card
	AddToHand(Card)
	SetHand([]Card)
	RemoveFromHand(uuid.UUID) (Card, bool)
	DiscardCard(uuid.UUID) (Card, bool)
	Graveyard() []Card
	AddToGraveyard(Card)
	RemoveFromGraveyard(uuid.UUID) (Card, bool)
	Library() []Card
	SetLibrary([]Card)
	DrawCard() (Card, bool)
	ManaPool() *ManaPool

	AddToLibrary(Card)
	ShuffleLibrary()
	ClearGraveyard()

	// Ante zone
	Ante() []Card
	AddToAnte(Card)
	RemoveFromAnte(uuid.UUID) (Card, bool)

	// Decision-making (overridden by TestPlayer)
	ChooseTargets(possible []uuid.UUID, min, max int, g *Game) []uuid.UUID
	DeclareAttackers(g *Game) []uuid.UUID
	DeclareBlockers(g *Game) []BlockAssignment
	ChooseMayAbility(description string) bool

	// Last drawn card tracking (for Jandor's Ring)
	LastDrawnCardID() uuid.UUID
	ClearLastDrawnCard()

	// Poison counters
	PoisonCounters() int
	AddPoisonCounters(int)

	// Player choice methods (overridden by TestPlayer for scripted choices)
	ChooseMode(modes []string, reason string) int
	ChoosePermanent(candidates []*Permanent, reason string, g GameReader) *Permanent
	ChooseCardsFromHand(amount int, reason string, g GameReader) []Card
	// ChooseCardFromHand picks one card from a pre-filtered candidate list
	// drawn from the player's hand. Used when an effect or cost limits the
	// choice to a subset of hand (e.g. "discard an artifact card"). Callers
	// must pre-filter candidates; implementations should return a card from
	// `candidates` (or nil if empty).
	ChooseCardFromHand(candidates []Card, reason string, g GameReader) Card
	ChooseManaColor(reason string) Color
	ChooseCardFromLibrary(candidates []Card, reason string, g GameReader) Card
	ChooseNumber(min, max int, reason string) int

	// ChooseString asks the player to pick one option from a string list,
	// e.g. a creature type for "as ~ enters, choose a creature type"
	// (Herald's Horn). Implementations must return a value from `options`;
	// callers pre-filter the list (e.g. exclude the source's own colors).
	// If `options` is empty, implementations should return "".
	ChooseString(options []string, reason string) string

	// ChooseDamageDistribution asks the player how to divide a fixed total of
	// damage among the given target IDs. The implementation must return a map
	// whose values are non-negative, whose keys are a subset of `possible`,
	// and whose values sum to exactly `total` (CR 601.2d). At least one
	// target must receive at least 1 damage if total > 0 and possible is
	// non-empty. The engine validates the response and falls back to a 1-per-
	// target dump if the returned distribution is malformed.
	ChooseDamageDistribution(possible []uuid.UUID, total int, reason string, g *Game) map[uuid.UUID]int

	// ChooseScryPlacement implements the controller's choice for a scry (CR 701.18).
	// `top` is the top N cards of the library in their current order (top first).
	// The implementation returns:
	//   - bottom: the subset of `top` (by ID) that go to the bottom of the library,
	//     in the order they will be placed (last ID becomes the new bottom card).
	//   - topOrder: the remaining cards (by ID) in the order they will be placed
	//     back on top of the library (first ID becomes the new top card).
	// The union of bottom and topOrder must equal the IDs in `top` exactly once each;
	// the engine validates this and falls back to the original order on any mismatch.
	ChooseScryPlacement(top []Card, reason string, g GameReader) (bottom []uuid.UUID, topOrder []uuid.UUID)

	// ChooseSurveilPlacement implements the controller's choice for a surveil
	// (CR 701.42). `top` is the top N cards of the library in their current
	// order (top first). The implementation returns:
	//   - graveyard: the subset of `top` (by ID) that go to the graveyard,
	//     in the order they will be placed.
	//   - topOrder: the remaining cards (by ID) in the order they will be
	//     placed back on top of the library (first ID becomes the new top card).
	// The union must equal the IDs in `top` exactly once each; the engine
	// validates this and falls back to the original order on any mismatch.
	ChooseSurveilPlacement(top []Card, reason string, g GameReader) (graveyard []uuid.UUID, topOrder []uuid.UUID)
}

// BasePlayer implements Player with basic functionality.
type BasePlayer struct {
	id              uuid.UUID
	name            string
	life            int
	startingLife    int
	lost            bool // true if the player has lost the game (e.g. deck-out)
	drewFromEmpty   bool // set when a draw is attempted from an empty library
	hand            []Card
	graveyard       []Card
	library         []Card
	ante            []Card
	manaPool        *ManaPool
	lastDrawnCardID uuid.UUID // ID of the last card drawn this turn (for Jandor's Ring)
	poisonCounters  int
}

func NewBasePlayer(name string) *BasePlayer {
	return &BasePlayer{
		id:       uuid.New(),
		name:     name,
		life:     20,
		manaPool: NewManaPool(),
	}
}

// NewBasePlayerWithID creates a BasePlayer with a specific ID (used for game cloning).
func NewBasePlayerWithID(id uuid.UUID, name string) *BasePlayer {
	return &BasePlayer{
		id:       id,
		name:     name,
		life:     20,
		manaPool: NewManaPool(),
	}
}

func (p *BasePlayer) PlayerID() uuid.UUID { return p.id }
func (p *BasePlayer) Name() string        { return p.name }
func (p *BasePlayer) Life() int           { return p.life }
func (p *BasePlayer) SetLife(n int)       { p.life = n }
func (p *BasePlayer) StartingLife() int {
	if p.startingLife == 0 {
		return 20
	}
	return p.startingLife
}
func (p *BasePlayer) SetStartingLife(n int) { p.startingLife = n }
func (p *BasePlayer) IsAlive() bool         { return p.life > 0 && !p.lost }
func (p *BasePlayer) DrewFromEmpty() bool   { return p.drewFromEmpty }
func (p *BasePlayer) ClearDrewFromEmpty()   { p.drewFromEmpty = false }
func (p *BasePlayer) SetLost()              { p.lost = true }
func (p *BasePlayer) ManaPool() *ManaPool   { return p.manaPool }

func (p *BasePlayer) GainLife(n int) {
	p.life += n
}

func (p *BasePlayer) LoseLife(n int) {
	p.life -= n
}

func (p *BasePlayer) Hand() []Card { return p.hand }

func (p *BasePlayer) AddToHand(c Card) {
	p.assertOwner(c)
	p.hand = append(p.hand, c)
}

func (p *BasePlayer) SetHand(cards []Card) {
	p.hand = cards
}

func (p *BasePlayer) RemoveFromHand(id uuid.UUID) (Card, bool) {
	for i, c := range p.hand {
		if c.ID() == id {
			p.hand = append(p.hand[:i], p.hand[i+1:]...)
			return c, true
		}
	}
	return nil, false
}

func (p *BasePlayer) DiscardCard(id uuid.UUID) (Card, bool) {
	c, ok := p.RemoveFromHand(id)
	if !ok {
		return nil, false
	}
	p.AddToGraveyard(c)
	return c, true
}

func (p *BasePlayer) Graveyard() []Card { return p.graveyard }

func (p *BasePlayer) AddToGraveyard(c Card) {
	p.assertOwner(c)
	p.graveyard = append(p.graveyard, c)
}

func (p *BasePlayer) RemoveFromGraveyard(id uuid.UUID) (Card, bool) {
	for i, c := range p.graveyard {
		if c.ID() == id {
			p.graveyard = append(p.graveyard[:i], p.graveyard[i+1:]...)
			return c, true
		}
	}
	return nil, false
}

func (p *BasePlayer) Library() []Card     { return p.library }
func (p *BasePlayer) SetLibrary(l []Card) { p.library = l }

func (p *BasePlayer) AddToLibrary(c Card) {
	p.assertOwner(c)
	p.library = append(p.library, c)
}

func (p *BasePlayer) ClearGraveyard() {
	p.graveyard = nil
}

func (p *BasePlayer) Ante() []Card { return p.ante }

func (p *BasePlayer) AddToAnte(c Card) {
	p.ante = append(p.ante, c)
}

func (p *BasePlayer) RemoveFromAnte(id uuid.UUID) (Card, bool) {
	for i, c := range p.ante {
		if c.ID() == id {
			p.ante = append(p.ante[:i], p.ante[i+1:]...)
			return c, true
		}
	}
	return nil, false
}

func (p *BasePlayer) ShuffleLibrary() {
	rand.Shuffle(len(p.library), func(i, j int) {
		p.library[i], p.library[j] = p.library[j], p.library[i]
	})
}

func (p *BasePlayer) DrawCard() (Card, bool) {
	if len(p.library) == 0 {
		p.drewFromEmpty = true
		return nil, false
	}
	c := p.library[0]
	p.library = p.library[1:]
	p.AddToHand(c)
	p.lastDrawnCardID = c.ID()
	return c, true
}

// LastDrawnCardID returns the ID of the last card drawn this turn.
func (p *BasePlayer) LastDrawnCardID() uuid.UUID { return p.lastDrawnCardID }

// ClearLastDrawnCard resets the last drawn card tracking (called at turn start).
func (p *BasePlayer) ClearLastDrawnCard() { p.lastDrawnCardID = uuid.Nil }

// PoisonCounters returns the number of poison counters this player has.
func (p *BasePlayer) PoisonCounters() int { return p.poisonCounters }

// AddPoisonCounters adds n poison counters to this player.
func (p *BasePlayer) AddPoisonCounters(n int) { p.poisonCounters += n }

// Default decision implementations (overridden by TestPlayer).
func (p *BasePlayer) ChooseTargets(possible []uuid.UUID, min, max int, g *Game) []uuid.UUID {
	if len(possible) >= min {
		return possible[:min]
	}
	return nil
}

func (p *BasePlayer) DeclareAttackers(g *Game) []uuid.UUID {
	return nil
}

func (p *BasePlayer) DeclareBlockers(g *Game) []BlockAssignment {
	return nil
}

func (p *BasePlayer) ChooseMayAbility(description string) bool {
	return false
}

// ChooseDamageDistribution defaults to dumping all damage onto the first
// possible target. Test/AI players override this for richer behavior.
func (p *BasePlayer) ChooseDamageDistribution(possible []uuid.UUID, total int, reason string, g *Game) map[uuid.UUID]int {
	if total <= 0 || len(possible) == 0 {
		return nil
	}
	return map[uuid.UUID]int{possible[0]: total}
}

// Default choice implementations (pick first available option).

func (p *BasePlayer) ChooseMode(modes []string, reason string) int {
	return 0
}

func (p *BasePlayer) ChoosePermanent(candidates []*Permanent, reason string, g GameReader) *Permanent {
	if len(candidates) > 0 {
		return candidates[0]
	}
	return nil
}

func (p *BasePlayer) ChooseCardsFromHand(amount int, reason string, g GameReader) []Card {
	hand := p.Hand()
	if amount > len(hand) {
		amount = len(hand)
	}
	result := make([]Card, amount)
	copy(result, hand[:amount])
	return result
}

func (p *BasePlayer) ChooseCardFromHand(candidates []Card, reason string, g GameReader) Card {
	if len(candidates) == 0 {
		return nil
	}
	return candidates[0]
}

func (p *BasePlayer) ChooseManaColor(reason string) Color {
	return White // default to White
}

func (p *BasePlayer) ChooseCardFromLibrary(candidates []Card, reason string, g GameReader) Card {
	if len(candidates) > 0 {
		return candidates[0]
	}
	return nil
}

func (p *BasePlayer) ChooseNumber(min, max int, reason string) int {
	return max // default: choose maximum
}

func (p *BasePlayer) ChooseString(options []string, reason string) string {
	if len(options) == 0 {
		return ""
	}
	return options[0]
}

// ChooseScryPlacement: deterministic default keeps every revealed card on top
// in its original order. Card implementations and AI players may override this.
func (p *BasePlayer) ChooseScryPlacement(top []Card, reason string, g GameReader) (bottom []uuid.UUID, topOrder []uuid.UUID) {
	topOrder = make([]uuid.UUID, len(top))
	for i, c := range top {
		topOrder[i] = c.ID()
	}
	return nil, topOrder
}

// ChooseSurveilPlacement: deterministic default keeps every revealed card on
// top in its original order (no cards milled to the graveyard). Card
// implementations and AI players may override this.
func (p *BasePlayer) ChooseSurveilPlacement(top []Card, reason string, g GameReader) (graveyard []uuid.UUID, topOrder []uuid.UUID) {
	topOrder = make([]uuid.UUID, len(top))
	for i, c := range top {
		topOrder[i] = c.ID()
	}
	return nil, topOrder
}

func (p *BasePlayer) assertOwner(c Card) {
	owner := c.Owner()
	if owner == uuid.Nil {
		c.SetOwner(p.id)
	} else if owner != p.id {
		panic(fmt.Sprintf("card %s (ID %s) has owner %s, expected %s (%s)",
			c.Name(), c.ID(), owner, p.id, p.name))
	}
}
