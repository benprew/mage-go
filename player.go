package mage

import "github.com/google/uuid"

// BlockAssignment represents a single blocker-attacker pair.
type BlockAssignment struct {
	BlockerID  uuid.UUID
	AttackerID uuid.UUID
}

// PlayerRef is used in test harness to refer to players.
type PlayerRef int

const (
	PlayerA PlayerRef = iota
	PlayerB
)

// Player interface for game players.
type Player interface {
	PlayerID() uuid.UUID
	Name() string
	Life() int
	SetLife(int)
	GainLife(int)
	LoseLife(int)
	IsAlive() bool
	Hand() []Card
	AddToHand(Card)
	RemoveFromHand(uuid.UUID) (Card, bool)
	Graveyard() []Card
	AddToGraveyard(Card)
	RemoveFromGraveyard(uuid.UUID) (Card, bool)
	Library() []Card
	SetLibrary([]Card)
	DrawCard() (Card, bool)
	ManaPool() *ManaPool

	AddToLibrary(Card)
	ClearGraveyard()

	// Decision-making (overridden by TestPlayer)
	ChooseTargets(possible []uuid.UUID, min, max int, g *Game) []uuid.UUID
	DeclareAttackers(g *Game) []uuid.UUID
	DeclareBlockers(g *Game) []BlockAssignment
	ChooseMayAbility(description string) bool
}

// BasePlayer implements Player with basic functionality.
type BasePlayer struct {
	id       uuid.UUID
	name     string
	life     int
	hand     []Card
	graveyard []Card
	library  []Card
	manaPool *ManaPool
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
func (p *BasePlayer) IsAlive() bool       { return p.life > 0 }
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

func (p *BasePlayer) DrawCard() (Card, bool) {
	if len(p.library) == 0 {
		return nil, false
	}
	c := p.library[0]
	p.library = p.library[1:]
	p.AddToHand(c)
	return c, true
}

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
