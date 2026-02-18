package mage

import "github.com/google/uuid"

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
	DeclareBlockers(g *Game) map[uuid.UUID]uuid.UUID // blocker → attacker
	ChooseMayAbility(description string) bool
}

// BasePlayer implements Player with basic functionality.
type BasePlayer struct {
	ID_       uuid.UUID
	Name_     string
	Life_     int
	Hand_     []Card
	Graveyard_ []Card
	Library_  []Card
	ManaPool_ *ManaPool
}

func NewBasePlayer(name string) *BasePlayer {
	return &BasePlayer{
		ID_:       uuid.New(),
		Name_:     name,
		Life_:     20,
		ManaPool_: NewManaPool(),
	}
}

func (p *BasePlayer) PlayerID() uuid.UUID { return p.ID_ }
func (p *BasePlayer) Name() string        { return p.Name_ }
func (p *BasePlayer) Life() int           { return p.Life_ }
func (p *BasePlayer) SetLife(n int)       { p.Life_ = n }
func (p *BasePlayer) IsAlive() bool       { return p.Life_ > 0 }
func (p *BasePlayer) ManaPool() *ManaPool { return p.ManaPool_ }

func (p *BasePlayer) GainLife(n int) {
	p.Life_ += n
}

func (p *BasePlayer) LoseLife(n int) {
	p.Life_ -= n
}

func (p *BasePlayer) Hand() []Card { return p.Hand_ }

func (p *BasePlayer) AddToHand(c Card) {
	p.Hand_ = append(p.Hand_, c)
}

func (p *BasePlayer) RemoveFromHand(id uuid.UUID) (Card, bool) {
	for i, c := range p.Hand_ {
		if c.ID() == id {
			p.Hand_ = append(p.Hand_[:i], p.Hand_[i+1:]...)
			return c, true
		}
	}
	return nil, false
}

func (p *BasePlayer) Graveyard() []Card { return p.Graveyard_ }

func (p *BasePlayer) AddToGraveyard(c Card) {
	p.Graveyard_ = append(p.Graveyard_, c)
}

func (p *BasePlayer) RemoveFromGraveyard(id uuid.UUID) (Card, bool) {
	for i, c := range p.Graveyard_ {
		if c.ID() == id {
			p.Graveyard_ = append(p.Graveyard_[:i], p.Graveyard_[i+1:]...)
			return c, true
		}
	}
	return nil, false
}

func (p *BasePlayer) Library() []Card    { return p.Library_ }
func (p *BasePlayer) SetLibrary(l []Card) { p.Library_ = l }

func (p *BasePlayer) AddToLibrary(c Card) {
	p.Library_ = append(p.Library_, c)
}

func (p *BasePlayer) ClearGraveyard() {
	p.Graveyard_ = nil
}

func (p *BasePlayer) DrawCard() (Card, bool) {
	if len(p.Library_) == 0 {
		return nil, false
	}
	c := p.Library_[0]
	p.Library_ = p.Library_[1:]
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

func (p *BasePlayer) DeclareBlockers(g *Game) map[uuid.UUID]uuid.UUID {
	return nil
}

func (p *BasePlayer) ChooseMayAbility(description string) bool {
	return false
}
