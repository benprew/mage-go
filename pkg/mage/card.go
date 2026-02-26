package mage

import (
	. "github.com/mage/mage/pkg/mage/core"
	"github.com/google/uuid"
)

// Card is the interface for all cards.
type Card interface {
	ID() uuid.UUID
	Name() string
	ManaCost() ManaCost
	Types() []CardType
	SuperTypes() []SuperType
	SubTypes() []string
	Abilities() []Ability
	AttrSeeds() map[Attr]int // keyword/attr seeds (for display and permanent creation)
	Owner() uuid.UUID
	Power() int
	Toughness() int
	Modes() []string
	Copy() Card
	SetOwner(uuid.UUID)
	SetID(uuid.UUID)
	HasType(CardType) bool
	HasSuperType(SuperType) bool
	HasSubType(string) bool
	Expansion() string
	AddType(CardType)
	AddAbility(Ability)
	AddSubType(string)
	CloneFrom(Card)
	SetBasePT(power, toughness int)
	SetModes([]string)
}

// BaseCard provides the common card implementation.
type BaseCard struct {
	id         uuid.UUID
	name       string
	manaCost   ManaCost
	types      []CardType
	superTypes []SuperType
	subTypes   []string
	abilities  []Ability
	owner      uuid.UUID
	power      int
	toughness  int
	isToken        bool
	modes          []string
	expansion      string       // set/expansion name (e.g. "Arabian Nights")
	attrSeeds      map[Attr]int // keyword/attr seeds; NewPermanent copies these to baseAttrs
	additionalCosts []Cost      // additional costs paid when casting (sacrifice, discard, etc.)
}

// AttrSeeds returns the keyword/attr seeds for this card.
// NewPermanent uses these to populate the permanent's baseAttrs.
func (c *BaseCard) AttrSeeds() map[Attr]int { return c.attrSeeds }

func (c *BaseCard) ID() uuid.UUID         { return c.id }
func (c *BaseCard) Name() string          { return c.name }
func (c *BaseCard) ManaCost() ManaCost    { return c.manaCost }
func (c *BaseCard) Types() []CardType       { return c.types }
func (c *BaseCard) SuperTypes() []SuperType { return c.superTypes }
func (c *BaseCard) SubTypes() []string      { return c.subTypes }
func (c *BaseCard) Abilities() []Ability  { return c.abilities }
func (c *BaseCard) Owner() uuid.UUID      { return c.owner }
func (c *BaseCard) Power() int            { return c.power }
func (c *BaseCard) Toughness() int        { return c.toughness }
func (c *BaseCard) Modes() []string         { return c.modes }
func (c *BaseCard) SetModes(m []string)      { c.modes = m }
func (c *BaseCard) SetBasePT(p, t int)       { c.power = p; c.toughness = t }
func (c *BaseCard) Expansion() string           { return c.expansion }
func (c *BaseCard) SetOwner(id uuid.UUID)    { c.owner = id }
func (c *BaseCard) SetID(id uuid.UUID)       { c.id = id }

func (c *BaseCard) HasType(t CardType) bool {
	for _, ct := range c.types {
		if ct == t {
			return true
		}
	}
	return false
}

func (c *BaseCard) HasSuperType(st SuperType) bool {
	for _, s := range c.superTypes {
		if s == st {
			return true
		}
	}
	return false
}

func (c *BaseCard) HasSubType(st string) bool {
	for _, s := range c.subTypes {
		if s == st {
			return true
		}
	}
	return false
}

func (c *BaseCard) AddType(t CardType) {
	c.types = append(c.types, t)
}

func (c *BaseCard) AddSubType(st string) {
	c.subTypes = append(c.subTypes, st)
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
	c.superTypes = make([]SuperType, len(other.SuperTypes()))
	copy(c.superTypes, other.SuperTypes())
	c.subTypes = make([]string, len(other.SubTypes()))
	copy(c.subTypes, other.SubTypes())
	c.abilities = make([]Ability, len(other.Abilities()))
	copy(c.abilities, other.Abilities())
	if m := other.Modes(); len(m) > 0 {
		c.modes = make([]string, len(m))
		copy(c.modes, m)
	}
	c.expansion = other.Expansion()
	if bc, ok := other.(*BaseCard); ok && len(bc.attrSeeds) > 0 {
		c.attrSeeds = make(map[Attr]int, len(bc.attrSeeds))
		for k, v := range bc.attrSeeds {
			c.attrSeeds[k] = v
		}
	}
}

func (c *BaseCard) Copy() Card {
	cp := *c
	cp.id = uuid.New()
	cp.types = make([]CardType, len(c.types))
	copy(cp.types, c.types)
	cp.superTypes = make([]SuperType, len(c.superTypes))
	copy(cp.superTypes, c.superTypes)
	cp.subTypes = make([]string, len(c.subTypes))
	copy(cp.subTypes, c.subTypes)
	cp.abilities = make([]Ability, len(c.abilities))
	copy(cp.abilities, c.abilities)
	if len(c.modes) > 0 {
		cp.modes = make([]string, len(c.modes))
		copy(cp.modes, c.modes)
	}
	if len(c.attrSeeds) > 0 {
		cp.attrSeeds = make(map[Attr]int, len(c.attrSeeds))
		for k, v := range c.attrSeeds {
			cp.attrSeeds[k] = v
		}
	}
	return &cp
}

// CardOption configures a card during construction.
type CardOption func(*BaseCard)

// WithAdditionalCost adds an additional cost that must be paid when casting this spell
// (e.g. sacrifice a creature, discard a card, pay life).
func WithAdditionalCost(cost Cost) CardOption {
	return func(c *BaseCard) { c.additionalCosts = append(c.additionalCosts, cost) }
}

// AdditionalCosts returns the additional costs for this card.
func (c *BaseCard) AdditionalCosts() []Cost { return c.additionalCosts }

// WithSuperTypes adds supertypes (Legendary, Basic, Snow, World) to a card.
func WithExpansion(name string) CardOption {
	return func(c *BaseCard) { c.expansion = name }
}

func WithSuperTypes(sts ...SuperType) CardOption {
	return func(c *BaseCard) { c.superTypes = append(c.superTypes, sts...) }
}

// WithSubTypes adds creature/land subtypes to a card.
func WithSubTypes(subTypes ...string) CardOption {
	return func(c *BaseCard) { c.subTypes = append(c.subTypes, subTypes...) }
}

// WithKeyword adds a keyword ability to a card.
// Seeds the card's attrSeeds so that NewPermanent can populate baseAttrs.
func WithKeyword(kw Keyword) CardOption {
	return func(c *BaseCard) {
		if c.attrSeeds == nil {
			c.attrSeeds = make(map[Attr]int)
		}
		c.attrSeeds[kw]++
	}
}

// WithAbility adds an ability to a card.
func WithAbility(a Ability) CardOption {
	return func(c *BaseCard) { c.AddAbility(a) }
}

// WithETBEffect adds an effect that runs inline when this permanent enters
// the battlefield, receiving the spell's targets. Used for non-aura permanents
// that need to act on their spell targets on entry (e.g. Oubliette).
func WithETBEffect(effect Effect) CardOption {
	return func(c *BaseCard) { c.AddAbility(ETBWithTargets(effect)) }
}

// WithCumulativeUpkeep adds a cumulative upkeep trigger to a card.
// Each upkeep, an age counter is added, then the controller must pay the cost
// multiplied by the number of age counters, or sacrifice the permanent.
// costPerAge is a mana cost string (e.g. "{1}" or "{G}") paid per age counter.
func WithCumulativeUpkeep(costPerAge string) CardOption {
	return func(c *BaseCard) {
		c.AddAbility(
			BeginningOfUpkeepTrigger(
				FuncEffect("cumulative upkeep",
					EffectProperties{Outcome: OutcomeDetriment},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						perm := g.FindPermanent(sourceID)
						if perm == nil {
							return nil
						}
						perm.AddCounter(Age, 1)
						count := perm.Counters[Age]
						// Build total cost: costPerAge repeated count times
						totalCost := ""
						for range count {
							totalCost += costPerAge
						}
						if g.TryPayCostFromLands(controller, totalCost) {
							return nil
						}
						g.Sacrifice(perm)
						return nil
					}), false,
			),
		)
	}
}

// WithCardType adds an additional card type (e.g. TypeArtifact on a creature).
func WithCardType(t CardType) CardOption {
	return func(c *BaseCard) { c.AddType(t) }
}

// WithManaAbility adds a mana ability for the given color.
func WithManaAbility(color Color) CardOption {
	return func(c *BaseCard) { c.AddAbility(NewManaAbility(color)) }
}

// WithAnyColorMana adds an any-color mana ability.
func WithAnyColorMana() CardOption {
	return func(c *BaseCard) { c.AddAbility(NewAnyColorManaAbility()) }
}

// WithActivatedAbility adds an activated ability built from the given effect,
// primary cost, and optional AbilityOption modifiers.
func WithActivatedAbility(effect Effect, cost Cost, opts ...AbilityOption) CardOption {
	return func(c *BaseCard) { c.AddAbility(NewActivatedAbility(effect, cost, opts...)) }
}


// WithStaticAbility adds a static ability that applies continuous effects.
func WithStaticAbility(effects ...ContinuousEffect) CardOption {
	return func(c *BaseCard) { c.AddAbility(StaticAbility(effects...)) }
}

func applyCardOpts(c *BaseCard, opts []CardOption) {
	for _, opt := range opts {
		opt(c)
	}
}

// NewCreature creates a new creature card.
func NewCreature(name, cost string, power, toughness int, opts ...CardOption) *BaseCard {
	c := &BaseCard{
		id:        uuid.New(),
		name:      name,
		manaCost:  ParseManaCost(cost),
		types:     []CardType{TypeCreature},
		power:     power,
		toughness: toughness,
	}
	applyCardOpts(c, opts)
	return c
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
		if c.attrSeeds == nil {
			c.attrSeeds = make(map[Attr]int)
		}
		c.attrSeeds[kw]++
	}
	return c
}

func (c *BaseCard) IsToken() bool { return c.isToken }

// NewInstant creates a new instant card. The spell parameter defines what
// happens when the spell resolves (use [NewTargetedSpell] or [NewSpellAbility]).
// Pass nil for placeholder cards with no effect.
func NewInstant(name, cost string, spell *SpellAbility, opts ...CardOption) *BaseCard {
	c := &BaseCard{
		id:       uuid.New(),
		name:     name,
		manaCost: ParseManaCost(cost),
		types:    []CardType{TypeInstant},
	}
	if spell != nil {
		c.AddAbility(spell)
	}
	applyCardOpts(c, opts)
	return c
}

// NewSorcery creates a new sorcery card. The spell parameter defines what
// happens when the spell resolves (use [NewTargetedSpell] or [NewSpellAbility]).
// Pass nil for placeholder cards with no effect.
func NewSorcery(name, cost string, spell *SpellAbility, opts ...CardOption) *BaseCard {
	c := &BaseCard{
		id:       uuid.New(),
		name:     name,
		manaCost: ParseManaCost(cost),
		types:    []CardType{TypeSorcery},
	}
	if spell != nil {
		c.AddAbility(spell)
	}
	applyCardOpts(c, opts)
	return c
}

// NewLand creates a new land card.
func NewLand(name string, opts ...CardOption) *BaseCard {
	c := &BaseCard{
		id:    uuid.New(),
		name:  name,
		types: []CardType{TypeLand},
	}
	applyCardOpts(c, opts)
	return c
}

// NewArtifact creates a new artifact card.
func NewArtifact(name, cost string, opts ...CardOption) *BaseCard {
	c := &BaseCard{
		id:       uuid.New(),
		name:     name,
		manaCost: ParseManaCost(cost),
		types:    []CardType{TypeArtifact},
	}
	applyCardOpts(c, opts)
	return c
}

// NewEnchantment creates a new enchantment card.
func NewEnchantment(name, cost string, opts ...CardOption) *BaseCard {
	c := &BaseCard{
		id:       uuid.New(),
		name:     name,
		manaCost: ParseManaCost(cost),
		types:    []CardType{TypeEnchantment},
	}
	applyCardOpts(c, opts)
	return c
}

// NewAura creates a new aura enchantment card.
func NewAura(name, cost string, opts ...CardOption) *BaseCard {
	c := &BaseCard{
		id:       uuid.New(),
		name:     name,
		manaCost: ParseManaCost(cost),
		types:    []CardType{TypeEnchantment},
		subTypes: []string{"Aura"},
	}
	applyCardOpts(c, opts)
	return c
}

// NewEquipment creates a new equipment artifact card.
func NewEquipment(name, cost string, opts ...CardOption) *BaseCard {
	c := &BaseCard{
		id:       uuid.New(),
		name:     name,
		manaCost: ParseManaCost(cost),
		types:    []CardType{TypeArtifact},
		subTypes: []string{"Equipment"},
	}
	applyCardOpts(c, opts)
	return c
}

// Permanent represents a card on the battlefield.
type Permanent struct {
	Card       Card
	Controller uuid.UUID
	Tapped     bool
	PhasedOut  bool // true when phased out (treated as though it doesn't exist)
	Damage     int
	Counters   map[CounterType]int

	AttachedTo  uuid.UUID   // what this permanent is attached to
	Attachments []uuid.UUID // what's attached to this permanent

	RuntimeAbilities []Ability  // base + granted by effects
	SubTypeOverride  []string   // if set, replaces card's subtypes (from continuous effects)
	BasePTOverride   *[2]int    // if set, overrides base P/T (for animate effects)
	ColorOverride    *[]Color   // if set, replaces card's colors (from lace effects)
	FaceDown         bool       // true when face-down (e.g. Illusionary Mask)

	// Attr system: additive/subtractive attribute counts.
	// baseAttrs holds intrinsic attrs (set at creation/ETB; persists until explicitly revoked).
	// grantedAttrs holds effect-cycle deltas (reset and recomputed each Apply() cycle).
	baseAttrs    map[Attr]int
	grantedAttrs map[Attr]int

	// P/T bonuses from continuous effects (LayerPT). Reset and recomputed each Apply() cycle.
	powerBonus int
	toughBonus int

	// ETB choices (e.g. Jihad: choose a color and an opponent)
	ChosenColor  Color
	ChosenPlayer uuid.UUID

	// Control-change tracking (e.g. Old Man of the Sea, Aladdin)
	ControlledPermanent uuid.UUID

	// TurnControlGained records the turn number on which the current controller gained
	// control. Used by cards like Rocket Launcher ("activate only if controlled since
	// the beginning of your most recent turn").
	TurnControlGained int

	// StoredValue holds an arbitrary numeric value chosen at ETB or during upkeep.
	// Used by cards like Shapeshifter (chosen number 0-7 for dynamic P/T).
	StoredValue int
}

// NewPermanent creates a permanent from a card.
func NewPermanent(card Card, controller uuid.UUID) *Permanent {
	p := &Permanent{
		Card:         card,
		Controller:   controller,
		Counters:     make(map[CounterType]int),
		baseAttrs:    make(map[Attr]int),
		grantedAttrs: make(map[Attr]int),
	}
	// Copy base abilities
	for _, a := range card.Abilities() {
		cp := a
		p.RuntimeAbilities = append(p.RuntimeAbilities, cp)
	}
	// Populate baseAttrs from card types.
	for _, t := range card.Types() {
		switch t {
		case TypeCreature:
			p.baseAttrs[AttrIsCreature]++
			p.baseAttrs[AttrCanAttack]++
			p.baseAttrs[AttrCanBlock]++
			p.baseAttrs[AttrHasPowerToughness]++
			p.baseAttrs[AttrSummonSick]++
		case TypeLand:
			p.baseAttrs[AttrIsLand]++
		case TypeArtifact:
			p.baseAttrs[AttrIsArtifact]++
		case TypeEnchantment:
			p.baseAttrs[AttrIsEnchantment]++
		}
	}
	// Populate baseAttrs from card's keyword seeds via the Card interface.
	for a, count := range card.AttrSeeds() {
		p.baseAttrs[a] += count
	}
	return p
}

// HasAttr returns true if this permanent currently has the given attribute.
// Face-down permanents only expose basic creature attrs (they are 2/2 colorless creatures
// with no other properties).
func (p *Permanent) HasAttr(a Attr) bool {
	if p.FaceDown {
		switch a {
		case AttrIsCreature, AttrCanAttack, AttrCanBlock, AttrHasPowerToughness:
			return true
		default:
			return false
		}
	}
	return p.baseAttrs[a]+p.grantedAttrs[a] > 0
}

// GrantBaseAttr increments the intrinsic count for attr a on this permanent.
// Use this for attrs that are part of the card's identity (e.g. AttrIsCreature, Flying).
func (p *Permanent) GrantBaseAttr(a Attr) {
	p.baseAttrs[a]++
}

// RevokeBaseAttr decrements the intrinsic count, flooring at zero.
// Used to clear transient base attrs such as AttrSummonSick at untap.
func (p *Permanent) RevokeBaseAttr(a Attr) {
	if p.baseAttrs[a] > 0 {
		p.baseAttrs[a]--
		if p.baseAttrs[a] == 0 {
			delete(p.baseAttrs, a)
		}
	}
}

func (p *Permanent) ID() uuid.UUID { return p.Card.ID() }
func (p *Permanent) Name() string  { return p.Card.Name() }

func (p *Permanent) HasType(t CardType) bool {
	if p.FaceDown {
		return t == TypeCreature
	}
	// Check attr-based identity for the four main battlefield types.
	// grantedAttrs (written by EffectManager.Apply()) allows effects to add/remove types.
	switch t {
	case TypeCreature:
		return p.HasAttr(AttrIsCreature)
	case TypeLand:
		return p.HasAttr(AttrIsLand)
	case TypeArtifact:
		return p.HasAttr(AttrIsArtifact)
	case TypeEnchantment:
		return p.HasAttr(AttrIsEnchantment)
	}
	// For other types (Instant, Sorcery, Planeswalker, etc.), fall through to card.
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

// HasKeyword checks if this permanent currently has the given keyword ability.
// Delegates to HasAttr: all keyword storage is now in baseAttrs/grantedAttrs.
func (p *Permanent) HasKeyword(kw Keyword) bool {
	return p.HasAttr(kw)
}

// KeywordNames returns the display names of all active keyword attrs on this permanent.
// Includes both intrinsic keywords (baseAttrs) and those granted by effects (grantedAttrs).
func (p *Permanent) KeywordNames() []string {
	if p.FaceDown {
		return nil
	}
	seen := make(map[Attr]bool)
	var result []string
	check := func(a Attr) {
		if !seen[a] && IsKeywordAttr(a) && p.HasAttr(a) {
			seen[a] = true
			result = append(result, a.String())
		}
	}
	for a := range p.baseAttrs {
		check(a)
	}
	for a := range p.grantedAttrs {
		check(a)
	}
	return result
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
func (p *Permanent) CurrentPower(g GameReader) int {
	pw := p.Card.Power()
	if p.BasePTOverride != nil {
		pw = p.BasePTOverride[0]
	}
	for ct, n := range p.Counters {
		pw += ct.PowerBoost() * n
	}
	// Continuous effects are applied by the EffectManager
	if g != nil {
		pw += p.powerBonus
	}
	return pw
}

// CurrentToughness returns toughness including counters and continuous effects.
func (p *Permanent) CurrentToughness(g GameReader) int {
	tg := p.Card.Toughness()
	if p.BasePTOverride != nil {
		tg = p.BasePTOverride[1]
	}
	for ct, n := range p.Counters {
		tg += ct.ToughnessBoost() * n
	}
	if g != nil {
		tg += p.toughBonus
	}
	return tg
}

// BoostPT adds power and toughness bonuses during continuous effect application.
// Only meaningful within a FuncContinuousEffect callback at LayerPT.
func (p *Permanent) BoostPT(power, toughness int) {
	p.powerBonus += power
	p.toughBonus += toughness
}

// LethalDamage returns true if damage >= current toughness.
func (p *Permanent) LethalDamage(g GameReader) bool {
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

// CanDeclareAsAttacker returns true if this permanent may be declared as an attacker.
// Reads like the rulebook: must be a creature (AttrCanAttack), must be untapped,
// must not be summoning sick (unless it has Haste), must not have Defender.
// Attack prevention by effects writes a negative attrDelta for AttrCanAttack, so
// HasAttr(AttrCanAttack) returning false captures both "not a creature" and "prevented".
func (p *Permanent) CanDeclareAsAttacker(g *Game) bool {
	return p.HasAttr(AttrCanAttack) &&
		!p.Tapped &&
		(!p.HasAttr(AttrSummonSick) || p.HasAttr(Haste)) &&
		!p.HasAttr(Defender)
}

// CanDeclareAsBlocker returns true if this permanent may be declared as a blocker.
func (p *Permanent) CanDeclareAsBlocker(g *Game) bool {
	return p.HasAttr(AttrCanBlock) &&
		!p.Tapped
}

// CanTapForEffect returns true if this permanent may tap to activate an ability
// (mana ability, activated ability). For permanents with AttrHasPowerToughness
// (creatures), summoning sickness applies unless they have Haste. For permanents
// without AttrHasPowerToughness (lands, non-creature artifacts), they tap freely.
func (p *Permanent) CanTapForEffect(g *Game) bool {
	if p.HasAttr(AttrHasPowerToughness) {
		return !p.HasAttr(AttrSummonSick) || p.HasAttr(Haste)
	}
	return true // lands, non-creature artifacts tap freely
}

// ---------------------------------------------------------------------------
// Card template helpers: pre-assembled cards for common patterns.
// These reduce boilerplate for cards that follow well-known formulas.
// ---------------------------------------------------------------------------

// NewLuckyCharm creates a {1} artifact that optionally gains 1 life whenever a
// spell of the given color is cast (e.g. Crystal Rod, Iron Star, Ivory Cup).
func NewLuckyCharm(name, cost string, color Color) *BaseCard {
	return NewArtifact(name, cost,
		WithAbility(WheneverSpellCastTrigger(GainLife(1), true, &color)),
	)
}

// NewLandDestruction creates a sorcery that destroys target land
// (e.g. Stone Rain, Sinkhole, Ice Storm).
func NewLandDestruction(name, cost string) *BaseCard {
	return NewSorcery(name, cost, NewTargetedSpell(TargetLand(), DestroyTargetLand()))
}

// NewBoostAura creates an aura enchantment that gives the enchanted creature
// +power/+toughness (e.g. Holy Strength, Unholy Strength, Giant Growth-style auras).
func NewBoostAura(name, cost string, power, toughness int) *BaseCard {
	return NewAura(name, cost,
		WithAbility(StaticAbility(
			BoostAttached(power, toughness, AttachAura),
		)),
	)
}
