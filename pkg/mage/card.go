package mage

import (
	"maps"
	"slices"
	"strings"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// Card owns card identity, ownership, and private execution state.
// A definition supplies its printed characteristics and rules.
type Card struct {
	cardCharacteristics
	definition      *CardDefinition
	characteristics *cardCharacteristics
	id              uuid.UUID
	abilities       []Ability
	owner           uuid.UUID
	additionalCosts []Cost          // additional costs paid when casting (sacrifice, discard, etc.)
	alternateCosts  []AlternateCost // card-level alternate casting costs (CR 117.9); see alternate_cost.go
	castTargets     []Target        // targeting requirements when casting (auras, targeted ETBs)
	uncounterable   bool            // intrinsic "can't be countered" flag (set via WithUncounterable)
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
func (c *Card) AuraAIProfile() (AuraAIProfile, bool) {
	if c.auraAIProfile == nil {
		return AuraAIProfile{}, false
	}
	return *c.auraAIProfile, true
}

// NewCard creates private card state from a shared definition.
// Options change only this card. They do not change the shared definition.
func NewCard(definition *CardDefinition, opts ...CardOption) *Card {
	if definition == nil {
		panic("mage: nil card definition")
	}
	c := &Card{definition: definition, id: uuid.New()}
	for _, rule := range definition.printed.Rules {
		ability := rule.instantiate()
		if ability == nil {
			panic("mage: rule returned a nil ability")
		}
		c.abilities = append(c.abilities, ability)
	}
	applyCardOpts(c, opts)
	return c
}

func (c *Card) Definition() *CardDefinition { return c.definition }
func (c *Card) ID() uuid.UUID               { return c.id }
func (c *Card) Owner() uuid.UUID            { return c.owner }
func (c *Card) SetID(id uuid.UUID)          { c.id = id }
func (c *Card) SetOwner(owner uuid.UUID)    { c.owner = owner }
func (c *Card) Name() string {
	if c.definition == nil {
		return c.name
	}
	if c.characteristics != nil {
		return c.characteristics.name
	}
	return c.definition.Name()
}
func (c *Card) ManaCost() ManaCost {
	if c.definition == nil {
		return c.manaCost
	}
	if c.characteristics != nil {
		return copyPrintedManaCost(c.characteristics.manaCost)
	}
	return c.definition.ManaCost()
}
func (c *Card) Types() []CardType {
	if c.definition == nil {
		return c.types
	}
	if c.characteristics != nil {
		return slices.Clone(c.characteristics.types)
	}
	return c.definition.Types()
}
func (c *Card) SuperTypes() []SuperType {
	if c.definition == nil {
		return c.superTypes
	}
	if c.characteristics != nil {
		return slices.Clone(c.characteristics.superTypes)
	}
	return c.definition.SuperTypes()
}
func (c *Card) SubTypes() []string {
	if c.definition == nil {
		return c.subTypes
	}
	if c.characteristics != nil {
		return slices.Clone(c.characteristics.subTypes)
	}
	return c.definition.SubTypes()
}
func (c *Card) Power() int {
	if c.definition == nil {
		return c.power
	}
	if c.characteristics != nil {
		return c.characteristics.power
	}
	return c.definition.Power()
}
func (c *Card) Toughness() int {
	if c.definition == nil {
		return c.toughness
	}
	if c.characteristics != nil {
		return c.characteristics.toughness
	}
	return c.definition.Toughness()
}
func (c *Card) Modes() []string {
	if c.definition == nil {
		return c.modes
	}
	if c.characteristics != nil {
		return slices.Clone(c.characteristics.modes)
	}
	return c.definition.Modes()
}
func (c *Card) AttrSeeds() map[Attr]int {
	if c.definition == nil {
		return c.attrSeeds
	}
	if c.characteristics != nil {
		return maps.Clone(c.characteristics.attrSeeds)
	}
	return c.definition.AttrSeeds()
}
func (c *Card) Abilities() []Ability {
	if c.definition == nil {
		return c.abilities
	}
	return slices.Clone(c.abilities)
}
func (c *Card) HasType(t CardType) bool {
	if c.definition == nil {
		return slices.Contains(c.types, t)
	}
	if c.characteristics != nil {
		return slices.Contains(c.characteristics.types, t)
	}
	return slices.Contains(c.definition.printed.TypeLine.CardTypes, t)
}
func (c *Card) HasSuperType(t SuperType) bool {
	if c.definition == nil {
		return slices.Contains(c.superTypes, t)
	}
	if c.characteristics != nil {
		return slices.Contains(c.characteristics.superTypes, t)
	}
	return slices.Contains(c.definition.printed.TypeLine.SuperTypes, t)
}
func (c *Card) HasSubType(t string) bool {
	if c.definition == nil {
		return slices.Contains(c.subTypes, t)
	}
	if c.characteristics != nil {
		return slices.Contains(c.characteristics.subTypes, t)
	}
	return slices.Contains(c.definition.printed.TypeLine.SubTypes, t)
}
func (c *Card) Colors() []Color {
	if c.definition == nil {
		if len(c.colorOverride) > 0 {
			return slices.Clone(c.colorOverride)
		}
		return c.manaCost.Colors()
	}
	if c.characteristics != nil {
		if len(c.characteristics.colorOverride) > 0 {
			return slices.Clone(c.characteristics.colorOverride)
		}
		return c.characteristics.manaCost.Colors()
	}
	if len(c.definition.printed.Colors) > 0 {
		return c.definition.Colors()
	}
	return c.ManaCost().Colors()
}
func (c *Card) CastTargets() []Target {
	if len(c.castTargets) > 0 {
		return slices.Clone(c.castTargets)
	}
	for _, ability := range c.abilities {
		if action, ok := ability.(*ActionDefinition); ok && action.Kind() == ActionSpell {
			return slices.Clone(action.Targets())
		}
	}
	return nil
}

// mutableCharacteristics supports legacy copy and construction methods.
// These methods cannot change the printed definition.
func (c *Card) mutableCharacteristics() *cardCharacteristics {
	if c.definition == nil {
		return &c.cardCharacteristics
	}
	if c.characteristics == nil {
		c.characteristics = &cardCharacteristics{
			name: c.Name(), manaCost: c.ManaCost(), types: c.Types(), superTypes: c.SuperTypes(), subTypes: c.SubTypes(),
			power: c.Power(), toughness: c.Toughness(), modes: c.Modes(), attrSeeds: c.AttrSeeds(), colorOverride: c.definition.Colors(),
		}
	}
	return c.characteristics
}
func (c *Card) SetBasePT(p, t int) {
	data := c.mutableCharacteristics()
	data.power, data.toughness = p, t
}
func (c *Card) SetModes(modes []string) {
	c.mutableCharacteristics().modes = slices.Clone(modes)
}
func (c *Card) AddType(t CardType) {
	data := c.mutableCharacteristics()
	data.types = append(data.types, t)
}
func (c *Card) AddSubType(t string) {
	data := c.mutableCharacteristics()
	data.subTypes = append(data.subTypes, t)
}
func (c *Card) AddAbility(a Ability) { c.abilities = append(c.abilities, a) }
func (c *Card) CloneFrom(other *Card) {
	if c.definition == nil {
		c.cloneLegacyFrom(other)
		return
	}
	snapshot := &cardCharacteristics{
		name: other.Name(), manaCost: copyPrintedManaCost(other.ManaCost()),
		types: slices.Clone(other.Types()), superTypes: slices.Clone(other.SuperTypes()), subTypes: slices.Clone(other.SubTypes()),
		power: other.Power(), toughness: other.Toughness(), modes: slices.Clone(other.Modes()),
		attrSeeds: maps.Clone(other.AttrSeeds()), colorOverride: cardColors(other),
	}
	castTargets := cloneInstanceTargets(other.CastTargets())
	abilities := make([]Ability, 0, len(other.Abilities()))
	for _, ability := range other.Abilities() {
		abilities = append(abilities, cloneInstanceAbility(ability))
	}
	c.characteristics = snapshot
	c.castTargets = castTargets
	c.abilities = abilities
}

// Copy creates a distinct card. CloneInstance preserves the identity for search.
func (c *Card) Copy() *Card {
	if c.definition == nil {
		return c.copyLegacy()
	}
	cloned := c.CloneInstance()
	cloned.id = uuid.New()
	for _, ability := range cloned.abilities {
		if setter, ok := ability.(interface{ setAbilityID(uuid.UUID) }); ok {
			setter.setAbilityID(uuid.New())
		}
	}
	return cloned
}

// CloneInstance shares the printed definition and copies the execution state.
func (c *Card) CloneInstance() *Card {
	cloned := *c
	cloned.castTargets = cloneInstanceTargets(c.castTargets)
	cloned.additionalCosts = slices.Clone(c.additionalCosts)
	cloned.alternateCosts = slices.Clone(c.alternateCosts)
	for i := range cloned.alternateCosts {
		cloned.alternateCosts[i].Mana = copyPrintedManaCost(c.alternateCosts[i].Mana)
		cloned.alternateCosts[i].Additional = slices.Clone(c.alternateCosts[i].Additional)
	}
	cloned.abilities = make([]Ability, len(c.abilities))
	for i, ability := range c.abilities {
		cloned.abilities[i] = cloneInstanceAbility(ability)
	}
	cloned.cardCharacteristics = *c.clone()
	cloned.characteristics = c.characteristics.clone()
	return &cloned
}

func (c *Card) cloneLegacyFrom(other *Card) {
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
	if len(other.colorOverride) > 0 {
		c.colorOverride = append([]Color(nil), other.colorOverride...)
	} else {
		c.colorOverride = nil
	}
	if len(other.attrSeeds) > 0 {
		c.attrSeeds = make(map[Attr]int, len(other.attrSeeds))
		maps.Copy(c.attrSeeds, other.attrSeeds)
	}
	c.uncounterable = other.uncounterable
	c.auraAIProfile = other.auraAIProfile
	if len(other.alternateCosts) > 0 {
		c.alternateCosts = append([]AlternateCost(nil), other.alternateCosts...)
	}
}

func (c *Card) copyLegacy() *Card {
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
	if len(c.colorOverride) > 0 {
		cp.colorOverride = append([]Color(nil), c.colorOverride...)
	}
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
type CardOption func(*Card)

// WithCastTarget sets the targeting requirement for casting this card.
// Used for auras that enchant non-creature permanents (e.g. "enchant land").
func WithCastTarget(t Target) CardOption {
	return func(c *Card) { c.castTargets = []Target{t} }
}

// WithAuraAIProfile records how an aura affects its enchanted creature so the
// AI can pick appropriate targets (own creatures for buffs, opponent creatures
// for debuffs/steal effects). See AuraAIProfile.
func WithAuraAIProfile(profile AuraAIProfile) CardOption {
	return func(c *Card) {
		p := profile
		c.auraAIProfile = &p
	}
}

// WithAdditionalCost adds an additional cost that must be paid when casting this spell
// (e.g. sacrifice a creature, discard a card, pay life).
func WithAdditionalCost(cost Cost) CardOption {
	return func(c *Card) { c.additionalCosts = append(c.additionalCosts, cost) }
}

// AdditionalCosts returns the additional costs for this card.
func (c *Card) AdditionalCosts() []Cost { return c.additionalCosts }

// WithSuperTypes adds supertypes (Legendary, Basic, Snow, World) to a card.
func WithSuperTypes(sts ...SuperType) CardOption {
	return func(c *Card) {
		data := c.mutableCharacteristics()
		data.superTypes = append(data.superTypes, sts...)
	}
}

// WithSubTypes adds creature/land subtypes to a card.
func WithSubTypes(subTypes ...string) CardOption {
	return func(c *Card) {
		data := c.mutableCharacteristics()
		data.subTypes = append(data.subTypes, subTypes...)
	}
}

// cardHasKeyword reports whether the card was registered with the given keyword
// attr seed. Used by the casting-permission gate (see Flash, CR 702.8) to inspect
// keywords on a card that is not yet on the battlefield as a Permanent.
func cardHasKeyword(c *Card, kw Attr) bool {
	if c == nil {
		return false
	}
	return c.AttrSeeds()[kw] > 0
}

// WithKeyword adds a keyword ability to a card.
// Seeds the card's attrSeeds so that NewPermanent can populate baseAttrs.
func WithKeyword(kw Keyword) CardOption {
	return func(c *Card) {
		data := c.mutableCharacteristics()
		if data.attrSeeds == nil {
			data.attrSeeds = make(map[Attr]int)
		}
		data.attrSeeds[kw]++
	}
}

// WithAbility adds an ability to a card.
func WithAbility(a Ability) CardOption {
	return func(c *Card) { c.AddAbility(a) }
}

// WithAction adds a spell or activated ability action to a card.
func WithAction(a *ActionDefinition) CardOption {
	return func(c *Card) { c.AddAbility(a) }
}

// WithETBEffect adds an effect that runs inline when this permanent enters
// the battlefield, receiving the spell's targets. Used for non-aura permanents
// that need to act on their spell targets on entry (e.g. Oubliette).
func WithETBEffect(effect Effect) CardOption {
	return func(c *Card) { c.AddAbility(ETBWithTargets(effect)) }
}

// WithCumulativeUpkeep adds a cumulative upkeep trigger to a card.
// Each upkeep, an age counter is added, then the controller must pay the cost
// multiplied by the number of age counters, or sacrifice the permanent.
// costPerAge is a mana cost string (e.g. "{1}" or "{G}") paid per age counter.
func WithCumulativeUpkeep(costPerAge string) CardOption {
	return func(c *Card) {
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
						if g.TryPayMana(controller, totalCost.String()) {
							return nil
						}
						g.DoSacrifice(perm)
						return nil
					}), false,
			),
		)
	}
}

// WithCardType adds an additional card type (e.g. TypeArtifact on a creature).
func WithCardType(t CardType) CardOption {
	return func(c *Card) { c.AddType(t) }
}

// WithManaAbility adds a mana ability for the given color.
func WithManaAbility(color Color, afterProduction ...Effect) CardOption {
	return func(c *Card) { c.AddAbility(NewManaAbility(color, afterProduction...)) }
}

// WithMultiManaAbility adds a mana ability with custom productions.
func WithMultiManaAbility(productions ...ManaProduction) CardOption {
	return func(c *Card) { c.AddAbility(NewMultiManaAbility(productions...)) }
}

// WithDynamicManaAbility adds a mana ability whose production is derived from
// the source permanent and whose follow-up effects run immediately after mana
// is produced.
func WithDynamicManaAbility(productions ManaProductionFunc, afterProduction ...Effect) CardOption {
	return func(c *Card) {
		c.AddAbility(NewDynamicManaAbility(productions, afterProduction...))
	}
}

// WithAnyColorMana adds an any-color mana ability.
func WithAnyColorMana() CardOption {
	return func(c *Card) { c.AddAbility(NewManaAbility(AnyColor)) }
}

// WithActivatedAbility adds an activated ability built from the given effect,
// primary cost, and optional AbilityOption modifiers.
func WithActivatedAbility(effect Effect, cost Cost, opts ...AbilityOption) CardOption {
	return func(c *Card) { c.AddAbility(NewActivatedAbility(effect, cost, opts...)) }
}

// WithStaticAbility adds a static ability that applies continuous effects.
func WithStaticAbility(effects ...ContinuousEffect) CardOption {
	return func(c *Card) { c.AddAbility(StaticAbility(effects...)) }
}

func applyCardOpts(c *Card, opts []CardOption) {
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
func NewCreature(name, cost string, power, toughness int, opts ...CardOption) *Card {
	c := &Card{
		id: uuid.New(),
		cardCharacteristics: cardCharacteristics{
			name:      name,
			manaCost:  ParseManaCost(cost),
			types:     []CardType{TypeCreature},
			power:     power,
			toughness: toughness,
		},
	}
	applyCardOpts(c, opts)
	return c
}

// NewToken creates a token with the given colors and no mana cost.
func NewToken(name string, power, toughness int, types []CardType, subTypes []string, colors []Color, keywords ...Keyword) *Card {
	c := &Card{
		id: uuid.New(),
		cardCharacteristics: cardCharacteristics{
			name:          name,
			types:         types,
			subTypes:      subTypes,
			power:         power,
			toughness:     toughness,
			colorOverride: append([]Color(nil), colors...),
		},
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
func NewInstant(name, cost string, parts ...any) *Card {
	spell, opts := splitSpellConstructorParts(parts...)
	c := &Card{
		id: uuid.New(),
		cardCharacteristics: cardCharacteristics{
			name:     name,
			manaCost: ParseManaCost(cost),
			types:    []CardType{TypeInstant},
		},
	}
	if spell != nil {
		c.AddAbility(spell)
	}
	applyCardOpts(c, opts)
	return c
}

// NewSorcery creates a new sorcery card. Pass effects and action options
// directly, or pass an explicit [*SpellAbility] for compatibility.
func NewSorcery(name, cost string, parts ...any) *Card {
	spell, opts := splitSpellConstructorParts(parts...)
	c := &Card{
		id: uuid.New(),
		cardCharacteristics: cardCharacteristics{
			name:     name,
			manaCost: ParseManaCost(cost),
			types:    []CardType{TypeSorcery},
		},
	}
	if spell != nil {
		c.AddAbility(spell)
	}
	applyCardOpts(c, opts)
	return c
}

// NewLand creates a new land card.
func NewLand(name string, opts ...CardOption) *Card {
	c := &Card{
		id: uuid.New(),
		cardCharacteristics: cardCharacteristics{
			name:  name,
			types: []CardType{TypeLand},
		},
	}
	applyCardOpts(c, opts)
	return c
}

// NewArtifact creates a new artifact card.
func NewArtifact(name, cost string, opts ...CardOption) *Card {
	c := &Card{
		id: uuid.New(),
		cardCharacteristics: cardCharacteristics{
			name:     name,
			manaCost: ParseManaCost(cost),
			types:    []CardType{TypeArtifact},
		},
	}
	applyCardOpts(c, opts)
	return c
}

// NewEnchantment creates a new enchantment card.
func NewEnchantment(name, cost string, opts ...CardOption) *Card {
	c := &Card{
		id: uuid.New(),
		cardCharacteristics: cardCharacteristics{
			name:     name,
			manaCost: ParseManaCost(cost),
			types:    []CardType{TypeEnchantment},
		},
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
func NewPlaneswalker(name, cost string, startingLoyalty int, opts ...CardOption) *Card {
	c := &Card{
		id: uuid.New(),
		cardCharacteristics: cardCharacteristics{
			name:     name,
			manaCost: ParseManaCost(cost),
			types:    []CardType{TypePlaneswalker},
		},
	}
	if startingLoyalty > 0 {
		c.AddAbility(EntersWithNCounters(Loyalty, startingLoyalty))
	}
	applyCardOpts(c, opts)
	return c
}

// NewAura creates a new aura enchantment card. Defaults to "enchant creature"
// targeting. Use WithCastTarget() to override (e.g. enchant land, enchant artifact).
func NewAura(name, cost string, opts ...CardOption) *Card {
	c := &Card{
		id:          uuid.New(),
		castTargets: []Target{TargetCreature()},
		cardCharacteristics: cardCharacteristics{
			name:     name,
			manaCost: ParseManaCost(cost),
			types:    []CardType{TypeEnchantment},
			subTypes: []string{"Aura"},
		},
	}
	applyCardOpts(c, opts)
	return c
}

// NewEquipment creates a new equipment artifact card.
func NewEquipment(name, cost string, opts ...CardOption) *Card {
	c := &Card{
		id: uuid.New(),
		cardCharacteristics: cardCharacteristics{
			name:     name,
			manaCost: ParseManaCost(cost),
			types:    []CardType{TypeArtifact},
			subTypes: []string{"Equipment"},
		},
	}
	applyCardOpts(c, opts)
	return c
}

// Permanent represents a card on the battlefield.
type Permanent struct {
	Card               *Card
	incarnationID      uuid.UUID
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

	RuntimeAbilities           []Ability // base + granted by effects
	SubTypeOverride            []string  // if set, replaces card's subtypes (from continuous effects)
	SubTypeAdditions           []string  // additive subtypes granted by continuous effects (CR 613 layer 4); kept alongside SubTypeOverride or card's intrinsic subtypes
	printedAbilitiesSuppressed bool
	subtypeFamilyOverrides     [subtypeFamilyCount][]string
	subtypeFamilyOverrideSet   [subtypeFamilyCount]bool
	subtypeFamilyAdditions     [subtypeFamilyCount][]string
	BasePTOverride             *[2]int  // if set, overrides base P/T (for animate effects)
	ColorOverride              *[]Color // if set, replaces card's colors (from lace effects)
	FaceDown                   bool     // true when face-down (e.g. Illusionary Mask)
	IsToken                    bool     // true for token permanents (cease to exist outside battlefield)

	// Attr system: additive/subtractive attribute counts.
	// baseAttrs holds intrinsic attrs (set at creation/ETB; persists until explicitly revoked).
	// printedAbilityAttrs identifies the portion generated by the card's rules text.
	// grantedAttrs holds effect-cycle deltas (reset and recomputed each Apply() cycle).
	// Fixed arrays (not maps) so Clone is a value copy with no allocation, and
	// HasAttr is two direct array indexes instead of two map probes.
	baseAttrs           [NumAttrs]int8
	printedAbilityAttrs [NumAttrs]int8
	grantedAttrs        [NumAttrs]int8

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
func NewPermanent(card *Card, controller uuid.UUID) *Permanent {
	p := &Permanent{
		Card:               card,
		incarnationID:      uuid.New(),
		computedController: controller,
		baseController:     controller,
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
	// Copy keyword seeds from the card.
	// count is always 1 per WithKeyword call; stacking well below int8 range.
	for a, count := range card.AttrSeeds() {
		p.baseAttrs[a] += int8(count)
		if IsAbilityAttr(a) {
			p.printedAbilityAttrs[a] += int8(count)
		}
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
	base := p.baseAttrs[a]
	if p.printedAbilitiesSuppressed && IsAbilityAttr(a) {
		base -= min(base, p.printedAbilityAttrs[a])
	}
	return base+p.grantedAttrs[a] > 0
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
	return cardColors(p.Card)
}

func (p *Permanent) HasSubType(s string) bool {
	if p.FaceDown {
		return false // face-down creatures have no subtypes
	}
	return slices.Contains(p.computedSubtypes(), s)
}

// SubTypes returns the permanent's current subtypes after continuous effects.
func (p *Permanent) SubTypes() []string {
	if p.FaceDown {
		return nil
	}
	return p.computedSubtypes()
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
func (p *Permanent) HasProtectionFrom(card *Card) bool {
	if p.FaceDown {
		return false // face-down creatures have no protection
	}
	// Protection granted by another permanent arrives wrapped by
	// WrapGrantedAbility, so it has to be unwrapped before the type check.
	for _, a := range p.RuntimeAbilities {
		if pa, ok := UnwrapAbility(a).(*ProtectionAbility); ok {
			if pa.Blocks(card) {
				return true
			}
		}
	}
	return false
}

// HasProtectionFromInGame checks protection against the source's current
// characteristics, including color-changing effects on spells and permanents.
func (p *Permanent) HasProtectionFromInGame(card *Card, g *Game) bool {
	if p.FaceDown {
		return false
	}
	for _, a := range p.RuntimeAbilities {
		if pa, ok := UnwrapAbility(a).(*ProtectionAbility); ok && pa.BlocksInGame(card, g) {
			return true
		}
	}
	return false
}

// CanBeTargetedBy checks hexproof, shroud, and protection.
func (p *Permanent) CanBeTargetedBy(source *Card, sourceController uuid.UUID, g *Game) bool {
	if p.HasKeyword(Shroud) {
		return false
	}
	if p.HasKeyword(Hexproof) && p.ControllerID() != sourceController {
		return false
	}
	if source != nil && p.HasProtectionFromInGame(source, g) {
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
	// "Can't be the target of spells unless it attacked or blocked this turn" (Lurker).
	if p.HasAttr(AttrCantBeTargetedBySpellsUnlessAttackedOrBlocked) && g != nil && !g.PermanentAttackedOrBlockedThisTurn(p.ID()) {
		if source != nil && (source.HasType(TypeInstant) || source.HasType(TypeSorcery) || g.FindPermanent(source.ID()) == nil) {
			return false
		}
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
func NewLuckyCharm(name, cost string, color Color) *Card {
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
func NewLandDestruction(name, cost string) *Card {
	return NewSorcery(name, cost, NewTargetedSpell(TargetLand(), DestroyTargetLand()))
}

// NewBoostAura creates an aura enchantment that gives the enchanted creature
// +power/+toughness (e.g. Holy Strength, Unholy Strength, Giant Growth-style auras).
func NewBoostAura(name, cost string, power, toughness int) *Card {
	return NewAura(name, cost,
		WithAbility(StaticAbility(
			BoostAttached(power, toughness, AttachAura),
		)),
		WithAuraAIProfile(AuraAIProfile{PowerBoost: power, ToughnessBoost: toughness}),
	)
}
