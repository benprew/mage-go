package mage

import (
	"math/rand"

	"github.com/google/uuid"
	. "github.com/mage/mage/pkg/mage/core"
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
	ChooseManaColor(reason string) Color
	ChooseCardFromLibrary(candidates []Card, reason string, g GameReader) Card
}

// BasePlayer implements Player with basic functionality.
type BasePlayer struct {
	id              uuid.UUID
	name            string
	life            int
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

func (p *BasePlayer) PlayerID() uuid.UUID { return p.id }
func (p *BasePlayer) Name() string        { return p.name }
func (p *BasePlayer) Life() int           { return p.life }
func (p *BasePlayer) SetLife(n int)       { p.life = n }
func (p *BasePlayer) IsAlive() bool         { return p.life > 0 && !p.lost }
func (p *BasePlayer) DrewFromEmpty() bool   { return p.drewFromEmpty }
func (p *BasePlayer) ClearDrewFromEmpty()   { p.drewFromEmpty = false }
func (p *BasePlayer) SetLost()              { p.lost = true }
func (p *BasePlayer) ManaPool() *ManaPool { return p.manaPool }

func (p *BasePlayer) GainLife(n int) {
	p.life += n
}

func (p *BasePlayer) LoseLife(n int) {
	p.life -= n
}

func (p *BasePlayer) Hand() []Card { return p.hand }

func (p *BasePlayer) AddToHand(c Card) {
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

func (p *BasePlayer) Graveyard() []Card { return p.graveyard }

func (p *BasePlayer) AddToGraveyard(c Card) {
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

func (p *BasePlayer) Library() []Card    { return p.library }
func (p *BasePlayer) SetLibrary(l []Card) { p.library = l }

func (p *BasePlayer) AddToLibrary(c Card) {
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

func (p *BasePlayer) ChooseManaColor(reason string) Color {
	return White // default to White
}

func (p *BasePlayer) ChooseCardFromLibrary(candidates []Card, reason string, g GameReader) Card {
	if len(candidates) > 0 {
		return candidates[0]
	}
	return nil
}
