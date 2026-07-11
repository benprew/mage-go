package mage

import (
	"maps"
	"slices"
	"strings"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
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
	AddType(CardType)
	AddAbility(Ability)
	AddSubType(string)
	CastTargets() []Target
	CloneFrom(Card)
	SetBasePT(power, toughness int)
	SetModes([]string)
	IsToken() bool
}

// BaseCard provides the common card implementation.
type BaseCard struct {
	id              uuid.UUID
	name            string
	manaCost        ManaCost
	types           []CardType
	superTypes      []SuperType
	subTypes        []string
	abilities       []Ability
	owner           uuid.UUID
	power           int
	toughness       int
	colorOverride   []Color
	modes           []string
	attrSeeds       map[Attr]int    // keyword/attr seeds; NewPermanent copies these to baseAttrs
	additionalCosts []Cost          // additional costs paid when casting (sacrifice, discard, etc.)
	alternateCosts  []AlternateCost // card-level alternate casting costs (CR 117.9); see alternate_cost.go
	castTargets     []Target        // targeting requirements when casting (auras, targeted ETBs)
	uncounterable   bool            // intrinsic "can't be countered" flag (set via WithUncounterable)
	isToken         bool            // true for token cards (created by NewToken)
	auraAIProfile   *AuraAIProfile  // AI targeting hint for auras (set via WithAuraAIProfile)
}

// AuraAIProfile tells the AI how an aura affects the creature it enchants so
// that cast-time target selection can choose own vs. opponent creatures.
// Boost auras populate PowerBoost/ToughnessBoost; control-stealing auras
// (e.g. Control Magic) set StealsControl. Continuous effects are opaque
// functions, so this metadata is recorded on the card at construction time.
type AuraAIProfile struct {
	PowerBoost     int
	ToughnessBoost int
	StealsControl  bool
}

// AuraAIProfile returns the AI targeting profile for an aura, if one was set.
func (c *BaseCard) AuraAIProfile() (AuraAIProfile, bool) {
	if c.auraAIProfile == nil {
		return AuraAIProfile{}, false
	}
	return *c.auraAIProfile, true
}

// AttrSeeds returns the keyword/attr seeds for this card.
// NewPermanent uses these to populate the permanent's baseAttrs.
func (c *BaseCard) AttrSeeds() map[Attr]int { return c.attrSeeds }

// SetColorOverride sets the colors of a card directly, bypassing mana cost
// derivation. Used by token-creation paths that produce colored tokens
// (CreateColoredToken).
func (c *BaseCard) SetColorOverride(colors []Color) {
	c.colorOverride = append([]Color(nil), colors...)
}

func (c *BaseCard) ID() uuid.UUID           { return c.id }
func (c *BaseCard) Name() string            { return c.name }
func (c *BaseCard) ManaCost() ManaCost      { return c.manaCost }
func (c *BaseCard) Types() []CardType       { return c.types }
func (c *BaseCard) SuperTypes() []SuperType { return c.superTypes }
func (c *BaseCard) SubTypes() []string      { return c.subTypes }
func (c *BaseCard) Abilities() []Ability    { return c.abilities }
func (c *BaseCard) Owner() uuid.UUID        { return c.owner }
func (c *BaseCard) Power() int              { return c.power }
func (c *BaseCard) Toughness() int          { return c.toughness }
func (c *BaseCard) Modes() []string         { return c.modes }
func (c *BaseCard) IsToken() bool           { return c.isToken }
func (c *BaseCard) SetModes(m []string)     { c.modes = m }
func (c *BaseCard) SetBasePT(p, t int)      { c.power = p; c.toughness = t }
func (c *BaseCard) SetOwner(id uuid.UUID)   { c.owner = id }
func (c *BaseCard) SetID(id uuid.UUID)      { c.id = id }

func (c *BaseCard) HasType(t CardType) bool {
	return slices.Contains(c.types, t)
}

func (c *BaseCard) HasSuperType(st SuperType) bool {
	return slices.Contains(c.superTypes, st)
}

func (c *BaseCard) HasSubType(st string) bool {
	return slices.Contains(c.subTypes, st)
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
	if ct := other.CastTargets(); len(ct) > 0 {
		c.castTargets = make([]Target, len(ct))
		copy(c.castTargets, ct)
	}
	if bc, ok := other.(*BaseCard); ok {
		if len(bc.attrSeeds) > 0 {
			c.attrSeeds = make(map[Attr]int, len(bc.attrSeeds))
			maps.Copy(c.attrSeeds, bc.attrSeeds)
		}
		c.uncounterable = bc.uncounterable
		c.auraAIProfile = bc.auraAIProfile
		if len(bc.alternateCosts) > 0 {
			c.alternateCosts = append([]AlternateCost(nil), bc.alternateCosts...)
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
		maps.Copy(cp.attrSeeds, c.attrSeeds)
	}
	return &cp
}

// CardOption configures a card during construction.
type CardOption func(*BaseCard)

// CastTargets returns the targeting requirements for casting this card.
// For auras, this is the enchant target. For spells with SpellAbility targets,
// this falls back to the first SpellAbility's targets.
func (c *BaseCard) CastTargets() []Target {
	if len(c.castTargets) > 0 {
		return c.castTargets
	}
	// Fall back to SpellAbility targets for existing spells.
	for _, a := range c.abilities {
		if sa, ok := a.(*SpellAbility); ok && sa.Kind() == ActionSpell {
			if targets := sa.Targets(); len(targets) > 0 {
				return targets
			}
		}
	}
	return nil
}

// WithCastTarget sets the targeting requirement for casting this card.
// Used for auras that enchant non-creature permanents (e.g. "enchant land").
func WithCastTarget(t Target) CardOption {
	return func(c *BaseCard) { c.castTargets = []Target{t} }
}

// WithAuraAIProfile records how an aura affects its enchanted creature so the
// AI can pick appropriate targets (own creatures for buffs, opponent creatures
// for debuffs/steal effects). See AuraAIProfile.
func WithAuraAIProfile(profile AuraAIProfile) CardOption {
	return func(c *BaseCard) {
		p := profile
		c.auraAIProfile = &p
	}
}

// WithAdditionalCost adds an additional cost that must be paid when casting this spell
// (e.g. sacrifice a creature, discard a card, pay life).
func WithAdditionalCost(cost Cost) CardOption {
	return func(c *BaseCard) { c.additionalCosts = append(c.additionalCosts, cost) }
}

// AdditionalCosts returns the additional costs for this card.
func (c *BaseCard) AdditionalCosts() []Cost { return c.additionalCosts }

// WithSuperTypes adds supertypes (Legendary, Basic, Snow, World) to a card.
func WithSuperTypes(sts ...SuperType) CardOption {
	return func(c *BaseCard) { c.superTypes = append(c.superTypes, sts...) }
}

// WithSubTypes adds creature/land subtypes to a card.
func WithSubTypes(subTypes ...string) CardOption {
	return func(c *BaseCard) { c.subTypes = append(c.subTypes, subTypes...) }
}

// cardHasKeyword reports whether the card was registered with the given keyword
// attr seed. Used by the casting-permission gate (see Flash, CR 702.8) to inspect
// keywords on a card that is not yet on the battlefield as a Permanent.
func cardHasKeyword(c Card, kw Attr) bool {
	if c == nil {
		return false
	}
	return c.AttrSeeds()[kw] > 0
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

// WithAction adds a spell or activated ability action to a card.
func WithAction(a *ActionDefinition) CardOption {
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
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						perm := g.FindPermanent(sourceID)
						if perm == nil {
							return nil
						}
						perm.AddCounter(Age, 1)
						count := perm.Counters[Age]
						// Build total cost: costPerAge repeated count times
						var totalCost strings.Builder
						for range count {
							totalCost.WriteString(costPerAge)
						}
						if g.TryPayCostFromLands(controller, totalCost.String()) {
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

// WithMultiManaAbility adds a mana ability with custom productions.
func WithMultiManaAbility(productions ...ManaProduction) CardOption {
	return func(c *BaseCard) { c.AddAbility(NewMultiManaAbility(productions...)) }
}

// WithAnyColorMana adds an any-color mana ability.
func WithAnyColorMana() CardOption {
	return func(c *BaseCard) { c.AddAbility(NewManaAbility(AnyColor)) }
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

func splitSpellConstructorParts(parts ...any) (*SpellAbility, []CardOption) {
	var spell *SpellAbility
	var actionParts []any
	var opts []CardOption
	for _, part := range parts {
		switch v := part.(type) {
		case nil:
		case *ActionDefinition:
			if v.Kind() == ActionSpell {
				spell = v
			} else {
				opts = append(opts, WithAction(v))
			}
		case Effect:
			actionParts = append(actionParts, v)
		case ActionOption:
			actionParts = append(actionParts, v)
		case CardOption:
			opts = append(opts, v)
		default:
			panic("mage: unsupported spell constructor argument")
		}
	}
	if len(actionParts) > 0 {
		if spell == nil {
			spell = NewSpell(actionParts...)
		} else {
			for _, opt := range actionPartsToOptions(actionParts...) {
				opt(spell)
			}
		}
	}
	return spell, opts
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

// NewInstant creates a new instant card. Pass effects and action options
// directly, or pass an explicit [*SpellAbility] for compatibility.
func NewInstant(name, cost string, parts ...any) *BaseCard {
	spell, opts := splitSpellConstructorParts(parts...)
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

// NewSorcery creates a new sorcery card. Pass effects and action options
// directly, or pass an explicit [*SpellAbility] for compatibility.
func NewSorcery(name, cost string, parts ...any) *BaseCard {
	spell, opts := splitSpellConstructorParts(parts...)
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

// NewPlaneswalker creates a new planeswalker card. The permanent enters with
// startingLoyalty loyalty counters via an EntersWithNCounters(Loyalty, …)
// replacement (CR 614.1c / 306.5b). This is the minimal planeswalker primitive:
// loyalty-activated abilities, attacking planeswalkers (CR 506.4 / 508.1), and
// the planeswalker damage-redirection rules (CR 117.6, removed in 2018) are NOT
// implemented. The 0-loyalty state-based action (CR 704.5i) IS implemented in
// CheckStateBasedActions.
func NewPlaneswalker(name, cost string, startingLoyalty int, opts ...CardOption) *BaseCard {
	c := &BaseCard{
		id:       uuid.New(),
		name:     name,
		manaCost: ParseManaCost(cost),
		types:    []CardType{TypePlaneswalker},
	}
	if startingLoyalty > 0 {
		c.AddAbility(EntersWithNCounters(Loyalty, startingLoyalty))
	}
	applyCardOpts(c, opts)
	return c
}

// NewAura creates a new aura enchantment card. Defaults to "enchant creature"
// targeting. Use WithCastTarget() to override (e.g. enchant land, enchant artifact).
func NewAura(name, cost string, opts ...CardOption) *BaseCard {
	c := &BaseCard{
		id:          uuid.New(),
		name:        name,
		manaCost:    ParseManaCost(cost),
		types:       []CardType{TypeEnchantment},
		subTypes:    []string{"Aura"},
		castTargets: []Target{TargetCreature()},
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
	Card               Card
	computedController uuid.UUID
	baseController     uuid.UUID
	Tapped             bool
	PhasedOut          bool // true when phased out (treated as though it doesn't exist)
	Damage             int
	// Counters is a fixed-size array indexed by CounterType, so cloning a
	// Permanent is a memcpy instead of a map allocation. Absent counters are
	// zero. To iterate, loop over [0, NumCounters) and skip zeros.
	Counters [NumCounters]uint8

	AttachedTo  uuid.UUID   // what this permanent is attached to
	Attachments []uuid.UUID // what's attached to this permanent

	RuntimeAbilities []Ability // base + granted by effects
	SubTypeOverride  []string  // if set, replaces card's subtypes (from continuous effects)
	SubTypeAdditions []string  // additive subtypes granted by continuous effects (CR 614 layer 4); kept alongside SubTypeOverride or card's intrinsic subtypes
	BasePTOverride   *[2]int   // if set, overrides base P/T (for animate effects)
	ColorOverride    *[]Color  // if set, replaces card's colors (from lace effects)
	FaceDown         bool      // true when face-down (e.g. Illusionary Mask)
	IsToken          bool      // true for token permanents (cease to exist outside battlefield)

	// Attr system: additive/subtractive attribute counts.
	// baseAttrs holds intrinsic attrs (set at creation/ETB; persists until explicitly revoked).
	// grantedAttrs holds effect-cycle deltas (reset and recomputed each Apply() cycle).
	// Fixed arrays (not maps) so Clone is a value copy with no allocation, and
	// HasAttr is two direct array indexes instead of two map probes.
	baseAttrs    [NumAttrs]int8
	grantedAttrs [NumAttrs]int8

	// P/T bonuses from continuous effects (LayerPT). Reset and recomputed each Apply() cycle.
	powerBonus int
	toughBonus int

	// ETB choices (e.g. Jihad: choose a color and an opponent;
	// Herald's Horn: choose a creature type). Set by "as ~ enters" replacement
	// effects (CR 614.12) and read by other abilities of the same permanent.
	ChosenColor   Color
	ChosenPlayer  uuid.UUID
	ChosenSubtype string

	// Control-change tracking (e.g. Old Man of the Sea, Aladdin)
	ControlledPermanent uuid.UUID

	// turnControlGained records the turn number on which the current controller gained
	// control. Used by cards like Rocket Launcher ("activate only if controlled since
	// the beginning of your most recent turn").
	turnControlGained int

	// StoredValue holds an arbitrary numeric value chosen at ETB or during upkeep.
	// Used by cards like Shapeshifter (chosen number 0-7 for dynamic P/T).
	StoredValue int

	// CreatedBy records the permanent ID of the source that created this token.
	// Used by cards like Tetravus that need to track their own tokens.
	CreatedBy uuid.UUID
}

// NewPermanent creates a permanent from a card.
func NewPermanent(card Card, controller uuid.UUID) *Permanent {
	p := &Permanent{
		Card:               card,
		computedController: controller,
		baseController:     controller,
		IsToken:            card.IsToken(),
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
		case TypeLand:
			p.baseAttrs[AttrIsLand]++
		case TypeArtifact:
			p.baseAttrs[AttrIsArtifact]++
		case TypeEnchantment:
			p.baseAttrs[AttrIsEnchantment]++
		}
	}
	// Summoning sickness applies to every permanent on ETB (CR 302.1) so a
	// later type change (e.g. Jade Statue animating an artifact mid-turn,
	// Living Lands turning Forests into creatures) honors the "continuously
	// controlled since most recent turn began" check. Cleared at untap.
	p.baseAttrs[AttrSummonSick]++
	// Populate baseAttrs from card's keyword seeds via the Card interface.
	// count is always 1 per WithKeyword call; stacking well below int8 range.
	for a, count := range card.AttrSeeds() {
		p.baseAttrs[a] += int8(count)
	}
	return p
}

// ControllerID returns the permanent's controller after continuous effects.
func (p *Permanent) ControllerID() uuid.UUID { return p.computedController }

// ControlledSinceTurnStart reports whether the current controller has
// controlled this permanent continuously since their most recent turn began.
func (p *Permanent) ControlledSinceTurnStart(g GameReader) bool {
	return p.turnControlGained < g.CurrentTurn()
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
	if bc, ok := p.Card.(*BaseCard); ok && len(bc.colorOverride) > 0 {
		return bc.colorOverride
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
	if slices.Contains(subs, s) {
		return true
	}
	if slices.Contains(p.SubTypeAdditions, s) {
		return true
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
	var result []string
	for a := Attr(1); a < NumAttrs; a++ {
		if IsKeywordAttr(a) && p.HasAttr(a) {
			result = append(result, a.String())
		}
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
	if p.HasKeyword(Hexproof) && p.ControllerID() != sourceController {
		return false
	}
	if source != nil && p.HasProtectionFrom(source) {
		return false
	}
	// "Can't be enchanted" — block enchantment spells from targeting this permanent.
	if source != nil && source.HasType(TypeEnchantment) && p.HasAttr(AttrCantBeEnchanted) {
		return false
	}
	// "Can't be targeted by abilities from artifact sources" (Artifact Ward).
	if source != nil && source.HasType(TypeArtifact) && p.HasAttr(AttrCantBeTargetedByArtifacts) {
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
	for ct := range NumCounters {
		if n := p.Counters[ct]; n != 0 {
			pw += ct.PowerBoost() * int(n)
		}
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
	for ct := range NumCounters {
		if n := p.Counters[ct]; n != 0 {
			tg += ct.ToughnessBoost() * int(n)
		}
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
	p.Counters[ct] += uint8(n)
}

// RemoveCounter removes counters, returns true if successful.
func (p *Permanent) RemoveCounter(ct CounterType, n int) bool {
	if int(p.Counters[ct]) < n {
		return false
	}
	p.Counters[ct] -= uint8(n)
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
// A granted AttrCanAttack (e.g. Animate Wall) overrides Defender, matching
// "can attack as though it didn't have defender."
func (p *Permanent) CanDeclareAsAttacker(g *Game) bool {
	return p.HasAttr(AttrCanAttack) &&
		!p.Tapped &&
		(!p.HasAttr(AttrSummonSick) || p.HasAttr(Haste)) &&
		(!p.HasAttr(Defender) || p.grantedAttrs[AttrCanAttack] > 0)
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
// Oracle: "Whenever a player casts a [color] spell, you may pay {1}. If you do,
// you gain 1 life." — the inner MayPayMana models the optional mana payment.
func NewLuckyCharm(name, cost string, color Color) *BaseCard {
	return NewArtifact(name, cost,
		WithAbility(WheneverSpellCastTrigger(
			MayPayMana("{1}", "gain 1 life", GainLife(1)),
			false,
			HasColorCardFilter(color),
		)),
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
		WithAuraAIProfile(AuraAIProfile{PowerBoost: power, ToughnessBoost: toughness}),
	)
}
