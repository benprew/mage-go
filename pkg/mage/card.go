package mage

import "github.com/google/uuid"

//go:generate enumer -type=CardType -trimprefix=Type -output=card_enumer.go

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
	AddType(CardType)
	AddAbility(Ability)
	CloneFrom(Card)
}

// BaseCard provides the common card implementation.
type BaseCard struct {
	id        uuid.UUID
	name      string
	manaCost  ManaCost
	types     []CardType
	subTypes  []string
	abilities []Ability
	owner     uuid.UUID
	power     int
	toughness int
	isToken   bool
}

func (c *BaseCard) ID() uuid.UUID         { return c.id }
func (c *BaseCard) Name() string          { return c.name }
func (c *BaseCard) ManaCost() ManaCost    { return c.manaCost }
func (c *BaseCard) Types() []CardType     { return c.types }
func (c *BaseCard) SubTypes() []string    { return c.subTypes }
func (c *BaseCard) Abilities() []Ability  { return c.abilities }
func (c *BaseCard) Owner() uuid.UUID      { return c.owner }
func (c *BaseCard) Power() int            { return c.power }
func (c *BaseCard) Toughness() int        { return c.toughness }
func (c *BaseCard) SetOwner(id uuid.UUID) { c.owner = id }
func (c *BaseCard) SetID(id uuid.UUID)    { c.id = id }

func (c *BaseCard) HasType(t CardType) bool {
	for _, ct := range c.types {
		if ct == t {
			return true
		}
	}
	return false
}

func (c *BaseCard) AddType(t CardType) {
	c.types = append(c.types, t)
}

func (c *BaseCard) AddAbility(a Ability) {
	c.abilities = append(c.abilities, a)
}

func (c *BaseCard) CloneFrom(other Card) {
	c.name = other.Name()
	c.manaCost = other.ManaCost()
	c.power = other.Power()
	c.toughness = other.Toughness()
	c.types = make([]CardType, len(other.Types()))
	copy(c.types, other.Types())
	c.subTypes = make([]string, len(other.SubTypes()))
	copy(c.subTypes, other.SubTypes())
	c.abilities = make([]Ability, len(other.Abilities()))
	copy(c.abilities, other.Abilities())
}

func (c *BaseCard) Copy() Card {
	cp := *c
	cp.id = uuid.New()
	cp.types = make([]CardType, len(c.types))
	copy(cp.types, c.types)
	cp.subTypes = make([]string, len(c.subTypes))
	copy(cp.subTypes, c.subTypes)
	cp.abilities = make([]Ability, len(c.abilities))
	copy(cp.abilities, c.abilities)
	return &cp
}

// NewCreature creates a new creature card.
func NewCreature(name, cost string, power, toughness int, subTypes ...string) *BaseCard {
	return &BaseCard{
		id:        uuid.New(),
		name:      name,
		manaCost:  ParseManaCost(cost),
		types:     []CardType{TypeCreature},
		subTypes:  subTypes,
		power:     power,
		toughness: toughness,
	}
}

// NewToken creates a token creature card. Tokens have no mana cost.
func NewToken(name string, power, toughness int, types []CardType, subTypes []string, keywords ...Keyword) *BaseCard {
	c := &BaseCard{
		id:        uuid.New(),
		name:      name,
		types:     types,
		subTypes:  subTypes,
		power:     power,
		toughness: toughness,
		isToken:   true,
	}
	for _, kw := range keywords {
		c.AddAbility(NewKeywordAbility(kw))
	}
	return c
}

// Setter methods for cross-package access to unexported fields.

func (c *BaseCard) SetPower(p int)     { c.power = p }
func (c *BaseCard) SetToughness(t int) { c.toughness = t }
func (c *BaseCard) IsToken() bool      { return c.isToken }

// NewInstant creates a new instant card.
func NewInstant(name, cost string) *BaseCard {
	return &BaseCard{
		id:       uuid.New(),
		name:     name,
		manaCost: ParseManaCost(cost),
		types:    []CardType{TypeInstant},
	}
}

// NewSorcery creates a new sorcery card.
func NewSorcery(name, cost string) *BaseCard {
	return &BaseCard{
		id:       uuid.New(),
		name:     name,
		manaCost: ParseManaCost(cost),
		types:    []CardType{TypeSorcery},
	}
}

// NewLand creates a new land card.
func NewLand(name string, subTypes ...string) *BaseCard {
	return &BaseCard{
		id:       uuid.New(),
		name:     name,
		types:    []CardType{TypeLand},
		subTypes: subTypes,
	}
}

// NewArtifact creates a new artifact card.
func NewArtifact(name, cost string) *BaseCard {
	return &BaseCard{
		id:       uuid.New(),
		name:     name,
		manaCost: ParseManaCost(cost),
		types:    []CardType{TypeArtifact},
	}
}

// NewEnchantment creates a new enchantment card.
func NewEnchantment(name, cost string) *BaseCard {
	return &BaseCard{
		id:       uuid.New(),
		name:     name,
		manaCost: ParseManaCost(cost),
		types:    []CardType{TypeEnchantment},
	}
}

// NewAura creates a new aura enchantment card.
func NewAura(name, cost string) *BaseCard {
	return &BaseCard{
		id:       uuid.New(),
		name:     name,
		manaCost: ParseManaCost(cost),
		types:    []CardType{TypeEnchantment},
		subTypes: []string{"Aura"},
	}
}

// NewEquipment creates a new equipment artifact card.
func NewEquipment(name, cost string) *BaseCard {
	return &BaseCard{
		id:       uuid.New(),
		name:     name,
		manaCost: ParseManaCost(cost),
		types:    []CardType{TypeArtifact},
		subTypes: []string{"Equipment"},
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

	RuntimeAbilities []Ability  // base + granted by effects
	SubTypeOverride  []string   // if set, replaces card's subtypes (from continuous effects)
	TypesAdded       []CardType // types added by continuous effects (e.g. Living Lands)
	BasePTOverride   *[2]int    // if set, overrides base P/T (for animate effects)
	ColorOverride    *[]Color   // if set, replaces card's colors (from lace effects)
	FaceDown         bool       // true when face-down (e.g. Illusionary Mask)
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

func (p *Permanent) ID() uuid.UUID { return p.Card.ID() }
func (p *Permanent) Name() string  { return p.Card.Name() }

func (p *Permanent) HasType(t CardType) bool {
	if p.FaceDown {
		return t == TypeCreature
	}
	for _, added := range p.TypesAdded {
		if added == t {
			return true
		}
	}
	return p.Card.HasType(t)
}

// Colors returns the permanent's current colors, considering color overrides.
func (p *Permanent) Colors() []Color {
	if p.FaceDown {
		return nil // face-down creatures are colorless
	}
	if p.ColorOverride != nil {
		return *p.ColorOverride
	}
	return p.Card.ManaCost().Colors()
}

func (p *Permanent) HasSubType(s string) bool {
	if p.FaceDown {
		return false // face-down creatures have no subtypes
	}
	subs := p.Card.SubTypes()
	if len(p.SubTypeOverride) > 0 {
		subs = p.SubTypeOverride
	}
	for _, st := range subs {
		if st == s {
			return true
		}
	}
	return false
}

// HasAbility checks if this permanent currently has the given keyword.
func (p *Permanent) HasKeyword(kw Keyword) bool {
	if p.FaceDown {
		return false // face-down creatures have no abilities
	}
	for _, a := range p.RuntimeAbilities {
		ab := UnwrapAbility(a)
		if ka, ok := ab.(*KeywordAbility); ok && ka.Keyword == kw {
			return true
		}
	}
	return false
}

// HasProtectionFrom checks if this permanent has protection that blocks the given card.
func (p *Permanent) HasProtectionFrom(card Card) bool {
	if p.FaceDown {
		return false // face-down creatures have no protection
	}
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
	if p.HasKeyword(Shroud) {
		return false
	}
	if p.HasKeyword(Hexproof) && p.Controller != sourceController {
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
	if p.BasePTOverride != nil {
		pw = p.BasePTOverride[0]
	}
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
	if p.BasePTOverride != nil {
		tg = p.BasePTOverride[1]
	}
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

// ---------------------------------------------------------------------------
// Card template helpers: pre-assembled cards for common patterns.
// These reduce boilerplate for cards that follow well-known formulas.
// ---------------------------------------------------------------------------

// NewLuckyCharm creates a {1} artifact that optionally gains 1 life whenever a
// spell of the given color is cast (e.g. Crystal Rod, Iron Star, Ivory Cup).
func NewLuckyCharm(name, cost string, color Color) *BaseCard {
	c := NewArtifact(name, cost)
	c.AddAbility(WheneverSpellCastTrigger(GainLife(1), true, &color))
	return c
}

// NewLandDestruction creates a sorcery that destroys target land
// (e.g. Stone Rain, Sinkhole, Ice Storm).
func NewLandDestruction(name, cost string) *BaseCard {
	c := NewSorcery(name, cost)
	c.AddAbility(NewTargetedSpell(TargetLand(), DestroyTargetLand()))
	return c
}

// NewBoostAura creates an aura enchantment that gives the enchanted creature
// +power/+toughness (e.g. Holy Strength, Unholy Strength, Giant Growth-style auras).
func NewBoostAura(name, cost string, power, toughness int) *BaseCard {
	c := NewAura(name, cost)
	c.AddAbility(StaticAbility(
		BoostAttached(power, toughness, AttachAura),
	))
	return c
}
