package mage

import "github.com/google/uuid"

// CardType represents a card's type.
type CardType int

const (
	TypeCreature CardType = iota
	TypeInstant
	TypeSorcery
	TypeLand
	TypeArtifact
	TypeEnchantment
)

func (ct CardType) String() string {
	switch ct {
	case TypeCreature:
		return "Creature"
	case TypeInstant:
		return "Instant"
	case TypeSorcery:
		return "Sorcery"
	case TypeLand:
		return "Land"
	case TypeArtifact:
		return "Artifact"
	case TypeEnchantment:
		return "Enchantment"
	default:
		return "Unknown"
	}
}

// Card is the interface for all cards.
type Card interface {
	ID() uuid.UUID
	Name() string
	ManaCost() ManaCost
	Types() []CardType
	SubTypes() []string
	Abilities() []Ability
	Owner() uuid.UUID
	Power() int
	Toughness() int
	Copy() Card
	SetOwner(uuid.UUID)
	SetID(uuid.UUID)
	HasType(CardType) bool
	AddAbility(Ability)
}

// BaseCard provides the common card implementation.
type BaseCard struct {
	ID_        uuid.UUID
	Name_      string
	ManaCost_  ManaCost
	Types_     []CardType
	SubTypes_  []string
	Abilities_ []Ability
	Owner_     uuid.UUID
	Power_     int
	Toughness_ int
}

func (c *BaseCard) ID() uuid.UUID         { return c.ID_ }
func (c *BaseCard) Name() string           { return c.Name_ }
func (c *BaseCard) ManaCost() ManaCost     { return c.ManaCost_ }
func (c *BaseCard) Types() []CardType      { return c.Types_ }
func (c *BaseCard) SubTypes() []string     { return c.SubTypes_ }
func (c *BaseCard) Abilities() []Ability   { return c.Abilities_ }
func (c *BaseCard) Owner() uuid.UUID       { return c.Owner_ }
func (c *BaseCard) Power() int             { return c.Power_ }
func (c *BaseCard) Toughness() int         { return c.Toughness_ }
func (c *BaseCard) SetOwner(id uuid.UUID)  { c.Owner_ = id }
func (c *BaseCard) SetID(id uuid.UUID)     { c.ID_ = id }

func (c *BaseCard) HasType(t CardType) bool {
	for _, ct := range c.Types_ {
		if ct == t {
			return true
		}
	}
	return false
}

func (c *BaseCard) AddAbility(a Ability) {
	c.Abilities_ = append(c.Abilities_, a)
}

func (c *BaseCard) Copy() Card {
	cp := *c
	cp.ID_ = uuid.New()
	cp.Types_ = make([]CardType, len(c.Types_))
	copy(cp.Types_, c.Types_)
	cp.SubTypes_ = make([]string, len(c.SubTypes_))
	copy(cp.SubTypes_, c.SubTypes_)
	cp.Abilities_ = make([]Ability, len(c.Abilities_))
	copy(cp.Abilities_, c.Abilities_)
	return &cp
}

// NewCreature creates a new creature card.
func NewCreature(name, cost string, subTypes ...string) *BaseCard {
	return &BaseCard{
		ID_:       uuid.New(),
		Name_:     name,
		ManaCost_: ParseManaCost(cost),
		Types_:    []CardType{TypeCreature},
		SubTypes_: subTypes,
	}
}

// NewInstant creates a new instant card.
func NewInstant(name, cost string) *BaseCard {
	return &BaseCard{
		ID_:       uuid.New(),
		Name_:     name,
		ManaCost_: ParseManaCost(cost),
		Types_:    []CardType{TypeInstant},
	}
}

// NewSorcery creates a new sorcery card.
func NewSorcery(name, cost string) *BaseCard {
	return &BaseCard{
		ID_:       uuid.New(),
		Name_:     name,
		ManaCost_: ParseManaCost(cost),
		Types_:    []CardType{TypeSorcery},
	}
}

// NewLand creates a new land card.
func NewLand(name string, subTypes ...string) *BaseCard {
	return &BaseCard{
		ID_:       uuid.New(),
		Name_:     name,
		Types_:    []CardType{TypeLand},
		SubTypes_: subTypes,
	}
}

// NewArtifact creates a new artifact card.
func NewArtifact(name, cost string) *BaseCard {
	return &BaseCard{
		ID_:       uuid.New(),
		Name_:     name,
		ManaCost_: ParseManaCost(cost),
		Types_:    []CardType{TypeArtifact},
	}
}

// NewEnchantment creates a new enchantment card.
func NewEnchantment(name, cost string) *BaseCard {
	return &BaseCard{
		ID_:       uuid.New(),
		Name_:     name,
		ManaCost_: ParseManaCost(cost),
		Types_:    []CardType{TypeEnchantment},
	}
}

// NewAura creates a new aura enchantment card.
func NewAura(name, cost string) *BaseCard {
	return &BaseCard{
		ID_:       uuid.New(),
		Name_:     name,
		ManaCost_: ParseManaCost(cost),
		Types_:    []CardType{TypeEnchantment},
		SubTypes_: []string{"Aura"},
	}
}

// NewEquipment creates a new equipment artifact card.
func NewEquipment(name, cost string) *BaseCard {
	return &BaseCard{
		ID_:       uuid.New(),
		Name_:     name,
		ManaCost_: ParseManaCost(cost),
		Types_:    []CardType{TypeArtifact},
		SubTypes_: []string{"Equipment"},
	}
}

// Permanent represents a card on the battlefield.
type Permanent struct {
	Card       Card
	Controller uuid.UUID
	Tapped     bool
	Damage     int
	Counters   map[CounterType]int
	SummonSick bool

	AttachedTo  uuid.UUID   // what this permanent is attached to
	Attachments []uuid.UUID // what's attached to this permanent

	RuntimeAbilities []Ability // base + granted by effects

	RegenerationShield     bool // if true, the next destruction is replaced by tap + remove damage
	DoesNotUntap           bool // if true, does not untap during untap step
	IntrinsicDoesNotUntap  bool // permanent property (e.g. Basalt Monolith)
	DamagePreventionShield int  // amount of damage to prevent
	DestroyAtEndOfTurn     bool // if true, destroy during cleanup
	Unblockable            bool // if true, can't be blocked this turn
	CantBeBlockedByWalls   bool // if true, can't be blocked by Walls (e.g. Juggernaut)
	CanBlockAdditional     int  // number of additional creatures this can block (e.g. Two-Headed Giant)
}

// NewPermanent creates a permanent from a card.
func NewPermanent(card Card, controller uuid.UUID) *Permanent {
	p := &Permanent{
		Card:       card,
		Controller: controller,
		Counters:   make(map[CounterType]int),
		SummonSick: true,
	}
	// Copy base abilities
	for _, a := range card.Abilities() {
		cp := a
		p.RuntimeAbilities = append(p.RuntimeAbilities, cp)
	}
	return p
}

func (p *Permanent) ID() uuid.UUID     { return p.Card.ID() }
func (p *Permanent) Name() string      { return p.Card.Name() }

func (p *Permanent) HasType(t CardType) bool {
	return p.Card.HasType(t)
}

func (p *Permanent) HasSubType(s string) bool {
	for _, st := range p.Card.SubTypes() {
		if st == s {
			return true
		}
	}
	return false
}

// HasAbility checks if this permanent currently has the given keyword.
func (p *Permanent) HasAbility(kw Keyword) bool {
	for _, a := range p.RuntimeAbilities {
		ab := a
		// Unwrap granted-by-effect wrapper
		if ge, ok := ab.(*grantedByEffect); ok {
			ab = ge.Ability
		}
		if ka, ok := ab.(*KeywordAbility); ok && ka.Keyword == kw {
			return true
		}
	}
	return false
}

// HasProtectionFrom checks if this permanent has protection that blocks the given card.
func (p *Permanent) HasProtectionFrom(card Card) bool {
	for _, a := range p.RuntimeAbilities {
		if pa, ok := a.(*ProtectionAbility); ok {
			if pa.Blocks(card) {
				return true
			}
		}
	}
	return false
}

// CanBeTargetedBy checks hexproof, shroud, and protection.
func (p *Permanent) CanBeTargetedBy(source Card, sourceController uuid.UUID, g *Game) bool {
	if p.HasAbility(Shroud) {
		return false
	}
	if p.HasAbility(Hexproof) && p.Controller != sourceController {
		return false
	}
	if source != nil && p.HasProtectionFrom(source) {
		return false
	}
	return true
}

// CurrentPower returns power including counters and continuous effects.
func (p *Permanent) CurrentPower(g *Game) int {
	pw := p.Card.Power()
	for ct, n := range p.Counters {
		pw += ct.PowerBoost() * n
	}
	// Continuous effects are applied by the EffectManager
	if g != nil {
		pw += g.Effects.PowerBonus(p.ID())
	}
	return pw
}

// CurrentToughness returns toughness including counters and continuous effects.
func (p *Permanent) CurrentToughness(g *Game) int {
	tg := p.Card.Toughness()
	for ct, n := range p.Counters {
		tg += ct.ToughnessBoost() * n
	}
	if g != nil {
		tg += g.Effects.ToughnessBonus(p.ID())
	}
	return tg
}

// LethalDamage returns true if damage >= current toughness.
func (p *Permanent) LethalDamage(g *Game) bool {
	return p.Damage >= p.CurrentToughness(g)
}

// AddCounter adds counters of the given type.
func (p *Permanent) AddCounter(ct CounterType, n int) {
	p.Counters[ct] += n
}

// RemoveCounter removes counters, returns true if successful.
func (p *Permanent) RemoveCounter(ct CounterType, n int) bool {
	if p.Counters[ct] < n {
		return false
	}
	p.Counters[ct] -= n
	if p.Counters[ct] == 0 {
		delete(p.Counters, ct)
	}
	return true
}

// IsAttached returns true if this permanent is attached to something.
func (p *Permanent) IsAttached() bool {
	return p.AttachedTo != uuid.Nil
}
