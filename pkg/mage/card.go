package mage

import (
	"iter"
	"slices"
	"strings"
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// Card owns card identity and ownership.
// A definition supplies its printed characteristics and rules.
type Card struct {
	*CardDefinition
	definition      *CardDefinition
	unflippedValues *CopiableValues
	id              uuid.UUID
	owner           uuid.UUID
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
	c := &Card{CardDefinition: definition, id: uuid.New()}
	if len(opts) > 0 {
		c.CardDefinition = c.cloneDefinition()
		for _, opt := range opts {
			opt(c)
		}
	}
	c.definition = definition
	return c
}

func (c *Card) Definition() *CardDefinition { return c.definition }
func (c *Card) ID() uuid.UUID               { return c.id }
func (c *Card) Owner() uuid.UUID            { return c.owner }
func (c *Card) SetID(id uuid.UUID)          { c.id = id }
func (c *Card) SetOwner(owner uuid.UUID)    { c.owner = owner }
func (c *Card) Name() string                { return c.CardDefinition.Name() }
func (c *Card) ManaCost() ManaCost          { return c.CardDefinition.ManaCost() }
func (c *Card) Types() []CardType           { return c.CardDefinition.Types() }
func (c *Card) SuperTypes() []SuperType     { return c.CardDefinition.SuperTypes() }
func (c *Card) SubTypes() []string          { return c.CardDefinition.SubTypes() }
func (c *Card) Power() int                  { return c.CardDefinition.Power() }
func (c *Card) Toughness() int              { return c.CardDefinition.Toughness() }
func (c *Card) Modes() []string             { return c.CardDefinition.Modes() }

// Abilities returns a private slice of all card abilities, including keywords.
// Use AllAbilities to inspect abilities without a slice copy.
// The returned slice is private. Ability objects retain their identity.
// This list does not include temporary abilities granted to a permanent.
func (c *Card) Abilities() []Ability {
	return slices.Clone(c.abilities)
}

// AllAbilities visits all card abilities in list order without a slice copy.
// Do not add or remove abilities during iteration. Ability objects retain
// their identity. Temporary permanent abilities are not part of this list.
func (c *Card) AllAbilities() iter.Seq2[int, Ability] {
	return func(yield func(int, Ability) bool) {
		for index, ability := range c.abilities {
			if !yield(index, ability) {
				return
			}
		}
	}
}
func (c *Card) HasType(t CardType) bool       { return slices.Contains(c.types, t) }
func (c *Card) HasSuperType(t SuperType) bool { return slices.Contains(c.superTypes, t) }
func (c *Card) HasSubType(t string) bool      { return slices.Contains(c.subTypes, t) }
func (c *Card) Colors() []Color {
	if c.colorOverride != nil {
		return slices.Clone(c.colorOverride)
	}
	return c.manaCost.Colors()
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
func (c *Card) mutableDefinition() *CardDefinition {
	if c.CardDefinition == nil || c.definition != nil {
		c.CardDefinition = c.cloneDefinition()
	}
	return c.CardDefinition
}

func (c *Card) mutableCharacteristics() *cardCharacteristics {
	return &c.mutableDefinition().cardCharacteristics
}
func (c *Card) SetBasePT(p, t int) {
	data := c.mutableCharacteristics()
	data.power, data.toughness = p, t
}
func (c *Card) SetModes(labels []string) {
	c.mutableDefinition()
	found := false
	for i, ability := range c.abilities {
		action, ok := ability.(*ActionDefinition)
		if !ok {
			continue
		}
		updated := *action
		updated.legacyModes = true
		updated.modes = make([]Mode, len(labels))
		for index, label := range labels {
			updated.modes[index] = Mode{Label: label, Targets: slices.Clone(action.targets), Effects: slices.Clone(action.effects)}
		}
		c.abilities[i] = &updated
		found = true
	}
	if !found && len(labels) > 0 {
		action := NewAction(ActionSpell)
		action.legacyModes = true
		for _, label := range labels {
			action.modes = append(action.modes, Mode{Label: label})
		}
		c.AddAbility(action)
	}
}
func (c *Card) AddType(t CardType) {
	data := c.mutableCharacteristics()
	if !slices.Contains(data.types, t) {
		data.types = append(data.types, t)
	}
}
func (c *Card) AddSubType(t string) {
	data := c.mutableCharacteristics()
	data.subTypes = append(data.subTypes, t)
}

// AddAbility adds an ability to the card.
func (c *Card) AddAbility(a Ability) {
	c.mutableDefinition()
	if _, simple := a.(*AttributeAbility); simple {
		c.abilities = append(c.abilities, a)
	} else {
		// Keep actions and triggers at their existing indices.
		index := len(c.abilities)
		for i, ability := range c.abilities {
			if _, simple := ability.(*AttributeAbility); simple {
				index = i
				break
			}
		}
		c.abilities = slices.Insert(c.abilities, index, a)
	}
}

func (c *Card) addCapability(attr Attr, count int) { c.AddAbility(StaticCapability(attr, count)) }

// CloneFrom applies the source's copiable values to this card.
// Use ApplyCopiableValues for new code.
func (c *Card) CloneFrom(other *Card) {
	c.ApplyCopiableValues(other.CopiableValues())
}

// Copy creates a new object from this card's copiable values.
// Use NewCopy for a new identity and CloneInstance to preserve identity.
func (c *Card) Copy() *Card {
	return c.NewCopy()
}

// CloneInstance preserves identity in a private card instance.
func (c *Card) CloneInstance() *Card {
	return newGameCloneContext().card(c)
}

// CardOption configures a card during construction.
type CardOption func(*Card)

// WithCastTarget sets the targeting requirement for casting this card.
// Used for auras that enchant non-creature permanents (e.g. "enchant land").
func WithCastTarget(t Target) CardOption {
	return func(c *Card) { c.mutableDefinition().castTargets = []Target{t} }
}

// WithAuraAIProfile records how an aura affects its enchanted creature so the
// AI can pick appropriate targets (own creatures for buffs, opponent creatures
// for debuffs/steal effects). See AuraAIProfile.
func WithAuraAIProfile(profile AuraAIProfile) CardOption {
	return func(c *Card) {
		p := profile
		c.mutableDefinition().auraAIProfile = &p
	}
}

// WithAdditionalCost adds an additional cost that must be paid when casting this spell
// (e.g. sacrifice a creature, discard a card, pay life).
func WithAdditionalCost(cost Cost) CardOption {
	return func(c *Card) { c.mutableDefinition().additionalCosts = append(slices.Clone(c.additionalCosts), cost) }
}

// AdditionalCosts returns the additional costs for this card.
func (c *Card) AdditionalCosts() []Cost { return c.CardDefinition.AdditionalCosts() }

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

// cardHasKeyword reports whether the card has the specified keyword ability.
func cardHasKeyword(c *Card, kw Attr) bool {
	if c == nil {
		return false
	}
	for _, ability := range c.abilities {
		if AbilityAttribute(ability) == kw {
			return true
		}
	}
	return false
}

// WithKeyword adds an ability and its attribute to a card.
// Capability attributes do not add an ability.
func WithKeyword(kw Keyword) CardOption {
	return func(c *Card) {
		if IsAbilityAttr(kw) {
			c.AddAbility(NewAttributeAbility(kw))
		} else {
			c.addCapability(kw, 1)
		}
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
	if c.definition == nil {
		c.definition = c.CardDefinition
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
		CardDefinition: &CardDefinition{
			name:      name,
			manaCost:  ParseManaCost(cost),
			types:     []CardType{TypeCreature},
			power:     power,
			toughness: toughness},
	}
	applyCardOpts(c, opts)
	return c
}

// NewToken creates a token with the given colors and no mana cost.
func NewToken(name string, power, toughness int, types []CardType, subTypes []string, colors []Color, keywords ...Keyword) *Card {
	c := &Card{
		id: uuid.New(),
		CardDefinition: &CardDefinition{
			name:          name,
			types:         slices.Clone(types),
			subTypes:      slices.Clone(subTypes),
			power:         power,
			toughness:     toughness,
			colorOverride: append([]Color(nil), colors...)},
	}
	for _, kw := range keywords {
		WithKeyword(kw)(c)
	}
	c.definition = c.CardDefinition
	return c
}

// NewInstant creates a new instant card. Pass effects and action options
// directly, or pass an explicit [*SpellAbility] for compatibility.
func NewInstant(name, cost string, parts ...any) *Card {
	spell, opts := splitSpellConstructorParts(parts...)
	c := &Card{
		id: uuid.New(),
		CardDefinition: &CardDefinition{
			name:     name,
			manaCost: ParseManaCost(cost),
			types:    []CardType{TypeInstant}},
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
		CardDefinition: &CardDefinition{
			name:     name,
			manaCost: ParseManaCost(cost),
			types:    []CardType{TypeSorcery}},
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
		CardDefinition: &CardDefinition{
			name:  name,
			types: []CardType{TypeLand}},
	}
	applyCardOpts(c, opts)
	return c
}

// NewArtifact creates a new artifact card.
func NewArtifact(name, cost string, opts ...CardOption) *Card {
	c := &Card{
		id: uuid.New(),
		CardDefinition: &CardDefinition{
			name:     name,
			manaCost: ParseManaCost(cost),
			types:    []CardType{TypeArtifact}},
	}
	applyCardOpts(c, opts)
	return c
}

// NewEnchantment creates a new enchantment card.
func NewEnchantment(name, cost string, opts ...CardOption) *Card {
	c := &Card{
		id: uuid.New(),
		CardDefinition: &CardDefinition{
			name:     name,
			manaCost: ParseManaCost(cost),
			types:    []CardType{TypeEnchantment}},
	}
	applyCardOpts(c, opts)
	return c
}

// NewPlaneswalker creates a card with printed starting loyalty.
// Entry places loyalty counters through the replacement system.
func NewPlaneswalker(name, cost string, startingLoyalty int, opts ...CardOption) *Card {
	c := &Card{
		id: uuid.New(),
		CardDefinition: &CardDefinition{
			name:     name,
			manaCost: ParseManaCost(cost),
			types:    []CardType{TypePlaneswalker}, baseLoyalty: startingLoyalty},
	}
	applyCardOpts(c, opts)
	return c
}

// NewAura creates a new aura enchantment card. Defaults to "enchant creature"
// targeting. Use WithCastTarget() to override (e.g. enchant land, enchant artifact).
func NewAura(name, cost string, opts ...CardOption) *Card {
	c := &Card{
		id: uuid.New(),
		CardDefinition: &CardDefinition{
			name:     name,
			manaCost: ParseManaCost(cost),
			types:    []CardType{TypeEnchantment},
			subTypes: []string{"Aura"}, castTargets: []Target{TargetCreature()}},
	}
	applyCardOpts(c, opts)
	return c
}

// NewEquipment creates a new equipment artifact card.
func NewEquipment(name, cost string, opts ...CardOption) *Card {
	c := &Card{
		id: uuid.New(),
		CardDefinition: &CardDefinition{
			name:     name,
			manaCost: ParseManaCost(cost),
			types:    []CardType{TypeArtifact},
			subTypes: []string{"Equipment"}},
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
	Flipped                    bool
	FaceDown                   bool // true when face-down (e.g. Illusionary Mask)
	IsToken                    bool // true for token permanents (cease to exist outside battlefield)

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
		RuntimeAbilities:   slices.Clone(card.abilities),
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
	for _, ability := range p.RuntimeAbilities {
		if attr := AbilityAttribute(ability); attr != 0 {
			amount := 1
			if specification, ok := ability.(*AttributeAbility); ok {
				amount = specification.amount
			}
			p.baseAttrs[attr] += int8(amount)
			if IsAbilityAttr(attr) {
				p.printedAbilityAttrs[attr] += int8(amount)
			}
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
	return p.AttachedTo != uuid.Nil()
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
