package mage

import "github.com/google/uuid"

// Layer represents a layer in the layer system.
type Layer int

const (
	LayerCopy    Layer = 1
	LayerControl Layer = 2
	LayerText    Layer = 3
	LayerType    Layer = 4
	LayerColor   Layer = 5
	LayerAbility Layer = 6
	LayerPT      Layer = 7
)

// Duration represents how long an effect lasts.
type Duration int

const (
	WhileOnBattlefield Duration = iota
	EndOfTurn
	EndOfCombat
	UntilYourNextTurn
	Indefinite // persists until target leaves battlefield (e.g. Sleight of Mind)
)

// AttachType distinguishes aura vs equipment attachment.
type AttachType int

const (
	AttachAura AttachType = iota
	AttachEquipment
)

// ContinuousEffect represents an ongoing effect on the game.
type ContinuousEffect interface {
	Apply(g *Game) error
	GetLayer() Layer
	GetDuration() Duration
	IsActive(g *Game) bool
	SourceID() uuid.UUID
	SetSourceID(uuid.UUID)
}

// effectSource provides SourceID/SetSourceID to all continuous effects.
type effectSource struct {
	sourceID uuid.UUID
}

func (s *effectSource) SourceID() uuid.UUID      { return s.sourceID }
func (s *effectSource) SetSourceID(id uuid.UUID) { s.sourceID = id }

// EffectManager manages and applies continuous effects.
type EffectManager struct {
	effects                []ContinuousEffect
	powerBonuses           map[uuid.UUID]int
	toughBonuses           map[uuid.UUID]int
	grantedKW              map[uuid.UUID][]Keyword
	removedKW              map[uuid.UUID][]Keyword
	preventAttack          map[uuid.UUID]bool
	regenerationShields    map[uuid.UUID]int
	preventionShields      map[uuid.UUID]int
	preventionRules        []damagePreventionRule
	landUntapLimit         int                     // -1 = no limit; >= 0 = max lands that may untap per turn
	forcefieldShields      map[uuid.UUID]bool      // players with Forcefield active this turn
	subtypeOverrides       map[uuid.UUID][]string  // permanent ID -> replacement subtypes
	reverseDamageShields   map[uuid.UUID]bool      // players with Reverse Damage active this turn
	channelActive          map[uuid.UUID]bool      // players with Channel active this turn
	colorPrevention        map[uuid.UUID][]Color   // player -> colors that prevent next damage source
	spellCostIncrease      map[Color]int           // color -> additional generic cost for spells of that color
	unlimitedLandPlays     bool                    // true if a player can play unlimited lands (Fastbond)
	sanctuaryActive        map[uuid.UUID]bool      // player -> if true, only flying/islandwalk can attack them
	lichActive             map[uuid.UUID]bool      // player -> if true, Lich replacement effects apply
	skipNextDraw           map[uuid.UUID]bool      // player -> if true, skip normal draw in draw step
	preventBlock           map[uuid.UUID]bool      // permanent -> can't block this turn (Raging River)
	manaConversion         map[Color]Color         // from color -> to color (Sunglasses of Urza)
	bodyguard              map[uuid.UUID]uuid.UUID // controller -> bodyguard permanent ID (Veteran Bodyguard)
	playerDamageRedirect   map[uuid.UUID]uuid.UUID // controller -> creature that absorbs ALL damage to player
	creatureDamageRedirect map[uuid.UUID]uuid.UUID // creature -> player who receives damage instead of creature (one-shot)
}

func NewEffectManager() *EffectManager {
	return &EffectManager{
		powerBonuses:           make(map[uuid.UUID]int),
		toughBonuses:           make(map[uuid.UUID]int),
		grantedKW:              make(map[uuid.UUID][]Keyword),
		removedKW:              make(map[uuid.UUID][]Keyword),
		preventAttack:          make(map[uuid.UUID]bool),
		regenerationShields:    make(map[uuid.UUID]int),
		preventionShields:      make(map[uuid.UUID]int),
		landUntapLimit:         -1,
		forcefieldShields:      make(map[uuid.UUID]bool),
		subtypeOverrides:       make(map[uuid.UUID][]string),
		reverseDamageShields:   make(map[uuid.UUID]bool),
		channelActive:          make(map[uuid.UUID]bool),
		colorPrevention:        make(map[uuid.UUID][]Color),
		spellCostIncrease:      make(map[Color]int),
		sanctuaryActive:        make(map[uuid.UUID]bool),
		lichActive:             make(map[uuid.UUID]bool),
		skipNextDraw:           make(map[uuid.UUID]bool),
		preventBlock:           make(map[uuid.UUID]bool),
		manaConversion:         make(map[Color]Color),
		bodyguard:              make(map[uuid.UUID]uuid.UUID),
		playerDamageRedirect:   make(map[uuid.UUID]uuid.UUID),
		creatureDamageRedirect: make(map[uuid.UUID]uuid.UUID),
	}
}

// SetSubTypeOverride replaces the subtypes of a permanent (e.g. Evil Presence, Phantasmal Terrain).
func (em *EffectManager) SetSubTypeOverride(permID uuid.UUID, subTypes []string) {
	em.subtypeOverrides[permID] = subTypes
}

// SubTypeOverride returns the overridden subtypes for a permanent, or nil if none.
func (em *EffectManager) SubTypeOverride(permID uuid.UUID) []string {
	return em.subtypeOverrides[permID]
}

// AddForcefieldShield marks a player as having Forcefield active this turn.
func (em *EffectManager) AddForcefieldShield(playerID uuid.UUID) {
	em.forcefieldShields[playerID] = true
}

// HasForcefieldShield returns true if the player has Forcefield active this turn.
func (em *EffectManager) HasForcefieldShield(playerID uuid.UUID) bool {
	return em.forcefieldShields[playerID]
}

// ClearForcefieldShields resets Forcefield shields at end of turn.
func (em *EffectManager) ClearForcefieldShields() {
	em.forcefieldShields = make(map[uuid.UUID]bool)
}

// LandUntapLimit returns the current land untap limit. -1 means no limit.
func (em *EffectManager) LandUntapLimit() int {
	return em.landUntapLimit
}

// HasUnlimitedLandPlays returns true if a player can play unlimited lands this turn.
func (em *EffectManager) HasUnlimitedLandPlays() bool {
	return em.unlimitedLandPlays
}

// SetSanctuaryActive marks a player as protected by Island Sanctuary.
func (em *EffectManager) SetSanctuaryActive(playerID uuid.UUID) {
	em.sanctuaryActive[playerID] = true
}

// IsSanctuaryActive returns true if the player is protected by Island Sanctuary.
func (em *EffectManager) IsSanctuaryActive(playerID uuid.UUID) bool {
	return em.sanctuaryActive[playerID]
}

// ClearSanctuary clears Island Sanctuary protection for a player.
func (em *EffectManager) ClearSanctuary(playerID uuid.UUID) {
	delete(em.sanctuaryActive, playerID)
}

// SetLichActive marks a player as having Lich replacement effects.
func (em *EffectManager) SetLichActive(playerID uuid.UUID) {
	em.lichActive[playerID] = true
}

// IsLichActive returns true if the player has Lich replacement effects.
func (em *EffectManager) IsLichActive(playerID uuid.UUID) bool {
	return em.lichActive[playerID]
}

// ClearLich clears Lich replacement effects for a player.
func (em *EffectManager) ClearLich(playerID uuid.UUID) {
	delete(em.lichActive, playerID)
}

// SetSkipNextDraw marks a player to skip their next draw step draw.
func (em *EffectManager) SetSkipNextDraw(playerID uuid.UUID) {
	em.skipNextDraw[playerID] = true
}

// ShouldSkipDraw returns true and clears the flag if the player should skip their draw.
func (em *EffectManager) ShouldSkipDraw(playerID uuid.UUID) bool {
	if em.skipNextDraw[playerID] {
		delete(em.skipNextDraw, playerID)
		return true
	}
	return false
}

// PreventFromBlocking prevents a creature from blocking this turn (Raging River).
func (em *EffectManager) PreventFromBlocking(permID uuid.UUID) {
	em.preventBlock[permID] = true
}

// CanBlock returns true if the creature is not prevented from blocking.
func (em *EffectManager) CanBlockCheck(permID uuid.UUID) bool {
	return !em.preventBlock[permID]
}

// ClearBlockPrevention clears all block prevention.
func (em *EffectManager) ClearBlockPrevention() {
	em.preventBlock = make(map[uuid.UUID]bool)
}

// SetManaConversion sets a mana color conversion (e.g. Red→White for Sunglasses of Urza).
func (em *EffectManager) SetManaConversion(from, to Color) {
	em.manaConversion[from] = to
}

// GetManaConversion returns the converted color for a given color, if any.
func (em *EffectManager) GetManaConversion(from Color) (Color, bool) {
	to, ok := em.manaConversion[from]
	return to, ok
}

// ClearManaConversion clears all mana conversions.
func (em *EffectManager) ClearManaConversion() {
	em.manaConversion = make(map[Color]Color)
}

// SetBodyguard marks a bodyguard permanent for a player (Veteran Bodyguard).
func (em *EffectManager) SetBodyguard(controllerID, permID uuid.UUID) {
	em.bodyguard[controllerID] = permID
}

// GetBodyguard returns the bodyguard permanent ID for a player, or uuid.Nil if none.
func (em *EffectManager) GetBodyguard(controllerID uuid.UUID) uuid.UUID {
	return em.bodyguard[controllerID]
}

// ClearBodyguard clears the bodyguard for a player.
func (em *EffectManager) ClearBodyguard(controllerID uuid.UUID) {
	delete(em.bodyguard, controllerID)
}

// SetPlayerDamageRedirect sets a creature that absorbs ALL damage dealt to a player.
func (em *EffectManager) SetPlayerDamageRedirect(controllerID, permID uuid.UUID) {
	em.playerDamageRedirect[controllerID] = permID
}

// GetPlayerDamageRedirect returns the creature absorbing damage for a player, or uuid.Nil.
func (em *EffectManager) GetPlayerDamageRedirect(controllerID uuid.UUID) uuid.UUID {
	return em.playerDamageRedirect[controllerID]
}

// SetCreatureDamageRedirect sets a one-shot redirect: next damage dealt to creatureID
// is dealt to targetPlayerID instead (Jade Monolith).
func (em *EffectManager) SetCreatureDamageRedirect(creatureID, targetPlayerID uuid.UUID) {
	em.creatureDamageRedirect[creatureID] = targetPlayerID
}

// GetCreatureDamageRedirect returns the player who should receive damage instead of a creature, or uuid.Nil.
func (em *EffectManager) GetCreatureDamageRedirect(creatureID uuid.UUID) uuid.UUID {
	return em.creatureDamageRedirect[creatureID]
}

// ClearCreatureDamageRedirect clears the one-shot creature damage redirect.
func (em *EffectManager) ClearCreatureDamageRedirect(creatureID uuid.UUID) {
	delete(em.creatureDamageRedirect, creatureID)
}

// AddRegenerationShield increments the regeneration shield count for a permanent.
func (em *EffectManager) AddRegenerationShield(targetID uuid.UUID) {
	em.regenerationShields[targetID]++
}

// ConsumeRegenerationShield returns true and decrements if a shield is available.
func (em *EffectManager) ConsumeRegenerationShield(targetID uuid.UUID) bool {
	if em.regenerationShields[targetID] > 0 {
		em.regenerationShields[targetID]--
		return true
	}
	return false
}

// ClearRegenerationShields clears all regeneration shields for permanents
// controlled by the given player (per MTG rules, cleared during untap step).
func (em *EffectManager) ClearRegenerationShields(playerID uuid.UUID, g *Game) {
	for _, p := range g.Battlefield {
		if p.Controller == playerID {
			delete(em.regenerationShields, p.ID())
		}
	}
}

// AddPreventionShield adds to the damage prevention shield for a permanent.
func (em *EffectManager) AddPreventionShield(targetID uuid.UUID, amount int) {
	em.preventionShields[targetID] += amount
}

// PreventDamage consumes prevention shields to prevent damage, returning the
// amount actually prevented.
func (em *EffectManager) PreventDamage(targetID uuid.UUID, amount int) int {
	shield := em.preventionShields[targetID]
	if shield <= 0 {
		return 0
	}
	prevented := min(amount, shield)
	em.preventionShields[targetID] -= prevented
	return prevented
}

// ClearPreventionShields resets all damage prevention shields.
func (em *EffectManager) ClearPreventionShields() {
	em.preventionShields = make(map[uuid.UUID]int)
}

// AddReverseDamageShield marks a player as having Reverse Damage active.
func (em *EffectManager) AddReverseDamageShield(playerID uuid.UUID) {
	em.reverseDamageShields[playerID] = true
}

// HasReverseDamageShield returns true if the player has Reverse Damage active.
func (em *EffectManager) HasReverseDamageShield(playerID uuid.UUID) bool {
	return em.reverseDamageShields[playerID]
}

// ClearReverseDamageShield clears Reverse Damage shield for a player.
func (em *EffectManager) ClearReverseDamageShield(playerID uuid.UUID) {
	delete(em.reverseDamageShields, playerID)
}

// SetChannelActive marks a player as having Channel active this turn.
func (em *EffectManager) SetChannelActive(playerID uuid.UUID) {
	em.channelActive[playerID] = true
}

// IsChannelActive returns true if the player has Channel active.
func (em *EffectManager) IsChannelActive(playerID uuid.UUID) bool {
	return em.channelActive[playerID]
}

// ClearChannelActive clears Channel state (called at end of turn).
func (em *EffectManager) ClearChannelActive() {
	em.channelActive = make(map[uuid.UUID]bool)
}

// AddColorPrevention adds a color prevention shield (prevents all damage from one source of that color).
func (em *EffectManager) AddColorPrevention(playerID uuid.UUID, color Color) {
	em.colorPrevention[playerID] = append(em.colorPrevention[playerID], color)
}

// CheckColorPrevention returns true and consumes a shield if the player has
// color prevention matching the source's color.
func (em *EffectManager) CheckColorPrevention(playerID uuid.UUID, sourceCard Card) bool {
	colors := em.colorPrevention[playerID]
	if len(colors) == 0 || sourceCard == nil {
		return false
	}
	sourceColors := sourceCard.ManaCost().Colors()
	for i, shield := range colors {
		for _, sc := range sourceColors {
			if sc == shield {
				// Consume this shield
				em.colorPrevention[playerID] = append(colors[:i], colors[i+1:]...)
				return true
			}
		}
	}
	return false
}

type damagePreventionRule struct {
	from    PermanentFilter
	to      PermanentFilter
	oneShot bool
}

type damagePreventionRuleOption func(*damagePreventionRule)

func (dpr *damagePreventionRule) WithFrom(from PermanentFilter) {
	dpr.from = from
}

func (dpr *damagePreventionRule) WithTo(to PermanentFilter) {
	dpr.to = to
}

func (dpr *damagePreventionRule) WithOneShot(oneshot bool) {
	dpr.oneShot = oneshot
}

func (em *EffectManager) AddDamagePreventionRule(opts ...damagePreventionRuleOption) {
	dpr := &damagePreventionRule{}
	for _, opt := range opts {
		opt(dpr)
	}

	em.preventionRules = append(em.preventionRules, *dpr)
}

func (em *EffectManager) ClearDamagePreventionRules() {
	em.preventionRules = make([]damagePreventionRule, 0)
}

func (em *EffectManager) Add(e ContinuousEffect) {
	em.effects = append(em.effects, e)
}

func (em *EffectManager) Remove(sourceID uuid.UUID) {
	filtered := em.effects[:0]
	for _, e := range em.effects {
		if e.SourceID() != sourceID {
			filtered = append(filtered, e)
		}
	}
	em.effects = filtered
}

// RemoveEndOfTurn removes all effects with EndOfTurn duration.
func (em *EffectManager) RemoveEndOfTurn() {
	filtered := em.effects[:0]
	for _, e := range em.effects {
		if e.GetDuration() != EndOfTurn {
			filtered = append(filtered, e)
		}
	}
	em.effects = filtered
}

// RemoveEndOfCombat removes all effects with EndOfCombat duration.
func (em *EffectManager) RemoveEndOfCombat() {
	filtered := em.effects[:0]
	for _, e := range em.effects {
		if e.GetDuration() != EndOfCombat {
			filtered = append(filtered, e)
		}
	}
	em.effects = filtered
}

// Apply resets computed bonuses and reapplies all active effects in layer order.
func (em *EffectManager) Apply(g *Game) {
	em.powerBonuses = make(map[uuid.UUID]int)
	em.toughBonuses = make(map[uuid.UUID]int)
	em.grantedKW = make(map[uuid.UUID][]Keyword)
	em.removedKW = make(map[uuid.UUID][]Keyword)
	em.preventAttack = make(map[uuid.UUID]bool)
	em.subtypeOverrides = make(map[uuid.UUID][]string)
	em.landUntapLimit = -1
	em.unlimitedLandPlays = false
	em.spellCostIncrease = make(map[Color]int)
	em.manaConversion = make(map[Color]Color)
	em.bodyguard = make(map[uuid.UUID]uuid.UUID)
	em.playerDamageRedirect = make(map[uuid.UUID]uuid.UUID)

	// Reset granted runtime abilities and subtype overrides from effects
	for _, p := range g.Battlefield {
		if p.FaceDown {
			// Face-down permanents keep their overrides and empty abilities
			continue
		}
		var base []Ability
		for _, a := range p.RuntimeAbilities {
			if _, ok := a.(*grantedByEffect); !ok {
				base = append(base, a)
			}
		}
		p.RuntimeAbilities = base
		p.SubTypeOverride = nil
		p.TypesAdded = nil
		p.BasePTOverride = nil
		p.ColorOverride = nil
	}

	// Remove effects whose source is no longer on the battlefield
	// EndOfTurn and EndOfCombat effects persist until cleanup regardless of source (e.g., Giant Growth, Jade Statue)
	// Indefinite effects persist as long as they self-report active (e.g., Sleight of Mind)
	active := em.effects[:0]
	for _, e := range em.effects {
		dur := e.GetDuration()
		if dur == EndOfTurn || dur == EndOfCombat || dur == Indefinite || g.FindPermanent(e.SourceID()) != nil {
			active = append(active, e)
		}
	}
	em.effects = active

	// Apply in layer order (1, 2, 4, 5, 6, 7)
	for _, layer := range []Layer{LayerCopy, LayerControl, LayerType, LayerColor, LayerAbility, LayerPT} {
		for _, e := range em.effects {
			if e.GetLayer() == layer && e.IsActive(g) {
				e.Apply(g)
			}
		}
	}

	// Sync mana conversions to all player mana pools
	for _, p := range g.Players {
		if len(em.manaConversion) > 0 {
			p.ManaPool().ManaConversions = em.manaConversion
		} else {
			p.ManaPool().ManaConversions = nil
		}
	}

	// Remove keywords that were stripped by effects (e.g. Earthbind removes Flying)
	for permID, keywords := range em.removedKW {
		perm := g.FindPermanent(permID)
		if perm == nil {
			continue
		}
		filtered := perm.RuntimeAbilities[:0]
		for _, a := range perm.RuntimeAbilities {
			ab := UnwrapAbility(a)
			if ka, ok := ab.(*KeywordAbility); ok {
				removed := false
				for _, kw := range keywords {
					if ka.Keyword == kw {
						removed = true
						break
					}
				}
				if removed {
					continue
				}
			}
			filtered = append(filtered, a)
		}
		perm.RuntimeAbilities = filtered
	}
}

func (em *EffectManager) PowerBonus(id uuid.UUID) int {
	return em.powerBonuses[id]
}

func (em *EffectManager) ToughnessBonus(id uuid.UUID) int {
	return em.toughBonuses[id]
}

func (em *EffectManager) GrantedKeywords(id uuid.UUID) []Keyword {
	return em.grantedKW[id]
}

func (em *EffectManager) CanAttack(id uuid.UUID) bool {
	return !em.preventAttack[id]
}

// grantedByEffect is a marker wrapper to identify abilities granted by continuous effects.
type grantedByEffect struct {
	Ability
}

// BoostAttached creates a continuous effect that boosts the attached creature's P/T.
func BoostAttached(power, toughness int, at AttachType) ContinuousEffect {
	return &boostAttachedEffect{
		power:      power,
		toughness:  toughness,
		attachType: at,
	}
}

type boostAttachedEffect struct {
	power      int
	toughness  int
	attachType AttachType
	effectSource
}

func (e *boostAttachedEffect) GetLayer() Layer       { return LayerPT }
func (e *boostAttachedEffect) GetDuration() Duration { return WhileOnBattlefield }

func (e *boostAttachedEffect) IsActive(g *Game) bool {
	src := g.FindPermanent(e.sourceID)
	if src == nil {
		return false
	}
	return src.IsAttached()
}

func (e *boostAttachedEffect) Apply(g *Game) error {
	src := g.FindPermanent(e.sourceID)
	if src == nil || !src.IsAttached() {
		return nil
	}
	g.Effects.powerBonuses[src.AttachedTo] += e.power
	g.Effects.toughBonuses[src.AttachedTo] += e.toughness
	return nil
}

// GrantAbilityToAttached creates a continuous effect granting a keyword to the attached creature.
func GrantAbilityToAttached(kw Keyword, at AttachType) ContinuousEffect {
	return &grantKeywordAttachedEffect{
		keyword:    kw,
		attachType: at,
	}
}

type grantKeywordAttachedEffect struct {
	keyword    Keyword
	attachType AttachType
	effectSource
}

func (e *grantKeywordAttachedEffect) GetLayer() Layer       { return LayerAbility }
func (e *grantKeywordAttachedEffect) GetDuration() Duration { return WhileOnBattlefield }

func (e *grantKeywordAttachedEffect) IsActive(g *Game) bool {
	src := g.FindPermanent(e.sourceID)
	if src == nil {
		return false
	}
	return src.IsAttached()
}

func (e *grantKeywordAttachedEffect) Apply(g *Game) error {
	src := g.FindPermanent(e.sourceID)
	if src == nil || !src.IsAttached() {
		return nil
	}
	target := g.FindPermanent(src.AttachedTo)
	if target == nil {
		return nil
	}
	g.Effects.grantedKW[target.ID()] = append(g.Effects.grantedKW[target.ID()], e.keyword)
	// Also add to runtime abilities so HasAbility works (wrapped so it can be removed on reapply)
	target.RuntimeAbilities = append(target.RuntimeAbilities, &grantedByEffect{NewKeywordAbility(e.keyword)})
	return nil
}

// GrantProtectionToAttached creates a continuous effect granting protection from
// a color to the attached creature (e.g. Black Ward, Blue Ward).
func GrantProtectionToAttached(color Color, at AttachType) ContinuousEffect {
	return &grantProtectionAttachedEffect{
		color:      color,
		attachType: at,
	}
}

type grantProtectionAttachedEffect struct {
	color      Color
	attachType AttachType
	effectSource
}

func (e *grantProtectionAttachedEffect) GetLayer() Layer       { return LayerAbility }
func (e *grantProtectionAttachedEffect) GetDuration() Duration { return WhileOnBattlefield }

func (e *grantProtectionAttachedEffect) IsActive(g *Game) bool {
	src := g.FindPermanent(e.sourceID)
	if src == nil {
		return false
	}
	return src.IsAttached()
}

func (e *grantProtectionAttachedEffect) Apply(g *Game) error {
	src := g.FindPermanent(e.sourceID)
	if src == nil || !src.IsAttached() {
		return nil
	}
	target := g.FindPermanent(src.AttachedTo)
	if target == nil {
		return nil
	}
	target.RuntimeAbilities = append(target.RuntimeAbilities, ProtectionFromColor(e.color))
	return nil
}

// RemoveKeywordFromAttached creates a continuous effect removing a keyword from the attached creature.
func RemoveKeywordFromAttached(kw Keyword, at AttachType) ContinuousEffect {
	return &removeKeywordAttachedEffect{
		keyword:    kw,
		attachType: at,
	}
}

type removeKeywordAttachedEffect struct {
	keyword    Keyword
	attachType AttachType
	effectSource
}

func (e *removeKeywordAttachedEffect) GetLayer() Layer       { return LayerAbility }
func (e *removeKeywordAttachedEffect) GetDuration() Duration { return WhileOnBattlefield }

func (e *removeKeywordAttachedEffect) IsActive(g *Game) bool {
	src := g.FindPermanent(e.sourceID)
	return src != nil && src.IsAttached()
}

func (e *removeKeywordAttachedEffect) Apply(g *Game) error {
	src := g.FindPermanent(e.sourceID)
	if src == nil || !src.IsAttached() {
		return nil
	}
	target := g.FindPermanent(src.AttachedTo)
	if target == nil {
		return nil
	}
	// Remove the keyword from runtime abilities
	var filtered []Ability
	for _, a := range target.RuntimeAbilities {
		ab := UnwrapAbility(a)
		if ka, ok := ab.(*KeywordAbility); ok && ka.Keyword == e.keyword {
			continue // remove this keyword
		}
		filtered = append(filtered, a)
	}
	target.RuntimeAbilities = filtered
	return nil
}

// ChangeAttachedSubTypes replaces the subtypes of the attached permanent (e.g. Evil Presence
// makes enchanted land a Swamp, Phantasmal Terrain makes it a chosen type).
func ChangeAttachedSubTypes(newSubTypes []string) ContinuousEffect {
	return &changeAttachedSubTypesEffect{newSubTypes: newSubTypes}
}

type changeAttachedSubTypesEffect struct {
	newSubTypes []string
	effectSource
}

func (e *changeAttachedSubTypesEffect) GetLayer() Layer       { return LayerType }
func (e *changeAttachedSubTypesEffect) GetDuration() Duration { return WhileOnBattlefield }

func (e *changeAttachedSubTypesEffect) IsActive(g *Game) bool {
	src := g.FindPermanent(e.sourceID)
	return src != nil && src.IsAttached()
}

func (e *changeAttachedSubTypesEffect) Apply(g *Game) error {
	src := g.FindPermanent(e.sourceID)
	if src == nil || !src.IsAttached() {
		return nil
	}
	target := g.FindPermanent(src.AttachedTo)
	if target == nil {
		return nil
	}
	target.SubTypeOverride = make([]string, len(e.newSubTypes))
	copy(target.SubTypeOverride, e.newSubTypes)
	return nil
}

// GrantActivatedAbilityToAttached grants an activated ability to the attached creature.
func GrantActivatedAbilityToAttached(effect Effect, cost Cost, at AttachType) ContinuousEffect {
	return &grantActivatedAbilityAttachedEffect{
		effect:     effect,
		cost:       cost,
		attachType: at,
	}
}

type grantActivatedAbilityAttachedEffect struct {
	effect     Effect
	cost       Cost
	attachType AttachType
	effectSource
}

func (e *grantActivatedAbilityAttachedEffect) GetLayer() Layer       { return LayerAbility }
func (e *grantActivatedAbilityAttachedEffect) GetDuration() Duration { return WhileOnBattlefield }

func (e *grantActivatedAbilityAttachedEffect) IsActive(g *Game) bool {
	src := g.FindPermanent(e.sourceID)
	return src != nil && src.IsAttached()
}

func (e *grantActivatedAbilityAttachedEffect) Apply(g *Game) error {
	src := g.FindPermanent(e.sourceID)
	if src == nil || !src.IsAttached() {
		return nil
	}
	target := g.FindPermanent(src.AttachedTo)
	if target == nil {
		return nil
	}
	// Create the activated ability and assign it to the target
	ab := NewActivatedAbility(e.effect, e.cost)
	ab.source = target.ID()
	ab.controller = target.Controller
	target.RuntimeAbilities = append(target.RuntimeAbilities, &grantedByEffect{ab})
	return nil
}

// grantActivatedAbilityToAllEffect grants an activated ability to all creatures matching a filter.
type grantActivatedAbilityToAllEffect struct {
	effect Effect
	cost   Cost
	filter PermanentFilter
	effectSource
}

// GrantActivatedAbilityToAll grants an activated ability to all creatures matching filter.
func GrantActivatedAbilityToAll(effect Effect, cost Cost, filter PermanentFilter) ContinuousEffect {
	return &grantActivatedAbilityToAllEffect{
		effect: effect,
		cost:   cost,
		filter: filter,
	}
}

func (e *grantActivatedAbilityToAllEffect) GetLayer() Layer       { return LayerAbility }
func (e *grantActivatedAbilityToAllEffect) GetDuration() Duration { return WhileOnBattlefield }

func (e *grantActivatedAbilityToAllEffect) IsActive(g *Game) bool {
	return g.FindPermanent(e.sourceID) != nil
}

func (e *grantActivatedAbilityToAllEffect) Apply(g *Game) error {
	for _, p := range g.Battlefield {
		if !p.HasType(TypeCreature) {
			continue
		}
		if p.ID() == e.sourceID {
			continue // typically "other" creatures
		}
		if e.filter != nil && !e.filter(p, g) {
			continue
		}
		ab := NewActivatedAbility(e.effect, e.cost)
		ab.source = p.ID()
		ab.controller = p.Controller
		p.RuntimeAbilities = append(p.RuntimeAbilities, &grantedByEffect{ab})
	}
	return nil
}

// PreventAttachedFromUntapping creates a continuous effect preventing the attached creature from untapping.
func PreventAttachedFromUntapping(at AttachType) ContinuousEffect {
	return &preventUntapEffect{
		attachType: at,
	}
}

type preventUntapEffect struct {
	attachType AttachType
	effectSource
}

func (e *preventUntapEffect) GetLayer() Layer       { return LayerAbility }
func (e *preventUntapEffect) GetDuration() Duration { return WhileOnBattlefield }

func (e *preventUntapEffect) IsActive(g *Game) bool {
	src := g.FindPermanent(e.sourceID)
	return src != nil && src.IsAttached()
}

func (e *preventUntapEffect) Apply(g *Game) error {
	src := g.FindPermanent(e.sourceID)
	if src == nil || !src.IsAttached() {
		return nil
	}
	target := g.FindPermanent(src.AttachedTo)
	if target != nil {
		g.Effects.grantedKW[target.ID()] = append(g.Effects.grantedKW[target.ID()], DoesNotUntapKW)
		target.RuntimeAbilities = append(target.RuntimeAbilities, &grantedByEffect{NewKeywordAbility(DoesNotUntapKW)})
	}
	return nil
}

// PreventAttachedFromAttacking creates a continuous effect preventing the attached creature from attacking.
func PreventAttachedFromAttacking(at AttachType) ContinuousEffect {
	return &preventAttackEffect{
		attachType: at,
	}
}

type preventAttackEffect struct {
	attachType AttachType
	effectSource
}

func (e *preventAttackEffect) GetLayer() Layer       { return LayerAbility }
func (e *preventAttackEffect) GetDuration() Duration { return WhileOnBattlefield }

func (e *preventAttackEffect) IsActive(g *Game) bool {
	src := g.FindPermanent(e.sourceID)
	if src == nil {
		return false
	}
	return src.IsAttached()
}

func (e *preventAttackEffect) Apply(g *Game) error {
	src := g.FindPermanent(e.sourceID)
	if src == nil || !src.IsAttached() {
		return nil
	}
	g.Effects.preventAttack[src.AttachedTo] = true
	return nil
}

// temporaryBoostEffect boosts a specific creature until end of turn.
type temporaryBoostEffect struct {
	targetID  uuid.UUID
	power     int
	toughness int
	effectSource
}

func (e *temporaryBoostEffect) GetLayer() Layer       { return LayerPT }
func (e *temporaryBoostEffect) GetDuration() Duration { return EndOfTurn }

func (e *temporaryBoostEffect) IsActive(g *Game) bool {
	return g.FindPermanent(e.targetID) != nil
}

func (e *temporaryBoostEffect) Apply(g *Game) error {
	g.Effects.powerBonuses[e.targetID] += e.power
	g.Effects.toughBonuses[e.targetID] += e.toughness
	return nil
}

// temporaryKeywordEffect grants a keyword to a specific creature until end of turn.
type temporaryKeywordEffect struct {
	targetID uuid.UUID
	keyword  Keyword
	effectSource
}

func (e *temporaryKeywordEffect) GetLayer() Layer       { return LayerAbility }
func (e *temporaryKeywordEffect) GetDuration() Duration { return EndOfTurn }

func (e *temporaryKeywordEffect) IsActive(g *Game) bool {
	return g.FindPermanent(e.targetID) != nil
}

func (e *temporaryKeywordEffect) Apply(g *Game) error {
	target := g.FindPermanent(e.targetID)
	if target == nil {
		return nil
	}
	g.Effects.grantedKW[e.targetID] = append(g.Effects.grantedKW[e.targetID], e.keyword)
	target.RuntimeAbilities = append(target.RuntimeAbilities, &grantedByEffect{NewKeywordAbility(e.keyword)})
	return nil
}

// BoostAllCreaturesContinuous boosts all creatures matching a filter.
// boostAllCreaturesEffect boosts all creatures matching an optional filter.
// When includeSelf is false (the default for lord effects), the source permanent
// is excluded from the boost. When includeSelf is true, all matching creatures
// including the source are boosted.
type boostAllCreaturesEffect struct {
	power       int
	toughness   int
	filter      PermanentFilter
	includeSelf bool
	effectSource
}

// BoostAllCreatures creates a continuous effect that boosts all matching creatures
// except the source (typical lord behavior).
func BoostAllCreatures(power, toughness int, filter PermanentFilter) ContinuousEffect {
	return &boostAllCreaturesEffect{
		power:     power,
		toughness: toughness,
		filter:    filter,
	}
}

// BoostAllCreaturesIncludingSelf creates a continuous effect that boosts all
// matching creatures including the source.
func BoostAllCreaturesIncludingSelf(power, toughness int, filter PermanentFilter) ContinuousEffect {
	return &boostAllCreaturesEffect{
		power:       power,
		toughness:   toughness,
		filter:      filter,
		includeSelf: true,
	}
}

func (e *boostAllCreaturesEffect) GetLayer() Layer       { return LayerPT }
func (e *boostAllCreaturesEffect) GetDuration() Duration { return WhileOnBattlefield }

func (e *boostAllCreaturesEffect) IsActive(g *Game) bool {
	return g.FindPermanent(e.sourceID) != nil
}

func (e *boostAllCreaturesEffect) Apply(g *Game) error {
	for _, p := range g.Battlefield {
		if !p.HasType(TypeCreature) {
			continue
		}
		if !e.includeSelf && p.ID() == e.sourceID {
			continue // lords typically don't boost themselves
		}
		if e.filter != nil && !e.filter(p, g) {
			continue
		}
		g.Effects.powerBonuses[p.ID()] += e.power
		g.Effects.toughBonuses[p.ID()] += e.toughness
	}
	return nil
}

// ptEqualsCountEffect sets the source permanent's P/T bonus equal to the count of
// matching permanents on the battlefield. Used for Plague Rats, etc.
// Each creature with this effect gets its own instance, which boosts only itself.
type ptEqualsCountEffect struct {
	countFilter    PermanentFilter // what to count
	controllerOnly bool            // if true, only count permanents you control
	effectSource
}

// PTEqualsCount creates a continuous effect where the source creature gets
// +N/+N where N is the count of permanents matching countFilter (on whole battlefield).
func PTEqualsCount(countFilter PermanentFilter) ContinuousEffect {
	return &ptEqualsCountEffect{
		countFilter: countFilter,
	}
}

// PTEqualsControlledCount creates a continuous effect where the source creature gets
// +N/+N where N is the count of permanents matching countFilter that you control.
func PTEqualsControlledCount(countFilter PermanentFilter) ContinuousEffect {
	return &ptEqualsCountEffect{
		countFilter:    countFilter,
		controllerOnly: true,
	}
}

func (e *ptEqualsCountEffect) GetLayer() Layer       { return LayerPT }
func (e *ptEqualsCountEffect) GetDuration() Duration { return WhileOnBattlefield }

func (e *ptEqualsCountEffect) IsActive(g *Game) bool {
	return g.FindPermanent(e.sourceID) != nil
}

func (e *ptEqualsCountEffect) Apply(g *Game) error {
	src := g.FindPermanent(e.sourceID)
	if src == nil {
		return nil
	}
	count := 0
	for _, p := range g.Battlefield {
		if e.controllerOnly && p.Controller != src.Controller {
			continue
		}
		if e.countFilter(p, g) {
			count++
		}
	}
	g.Effects.powerBonuses[src.ID()] += count
	g.Effects.toughBonuses[src.ID()] += count
	return nil
}

// powerEqualsCountEffect sets only a permanent's power bonus equal to a count.
type powerEqualsCountEffect struct {
	countFilter PermanentFilter
	effectSource
}

// PowerEqualsCount creates a continuous effect where the source creature gets
// power bonus equal to count of permanents matching countFilter. Toughness is unchanged.
func PowerEqualsCount(countFilter PermanentFilter) ContinuousEffect {
	return &powerEqualsCountEffect{
		countFilter: countFilter,
	}
}

func (e *powerEqualsCountEffect) GetLayer() Layer       { return LayerPT }
func (e *powerEqualsCountEffect) GetDuration() Duration { return WhileOnBattlefield }

func (e *powerEqualsCountEffect) IsActive(g *Game) bool {
	return g.FindPermanent(e.sourceID) != nil
}

func (e *powerEqualsCountEffect) Apply(g *Game) error {
	count := 0
	for _, p := range g.Battlefield {
		if e.countFilter(p, g) {
			count++
		}
	}
	src := g.FindPermanent(e.sourceID)
	if src != nil {
		g.Effects.powerBonuses[src.ID()] += count
		g.Effects.toughBonuses[src.ID()] += count
	}
	return nil
}

// grantKeywordToAllEffect grants a keyword ability to all matching creatures.
type grantKeywordToAllEffect struct {
	keyword Keyword
	filter  PermanentFilter
	effectSource
}

func GrantKeywordToAll(kw Keyword, filter PermanentFilter) ContinuousEffect {
	return &grantKeywordToAllEffect{
		keyword: kw,
		filter:  filter,
	}
}

func (e *grantKeywordToAllEffect) GetLayer() Layer       { return LayerAbility }
func (e *grantKeywordToAllEffect) GetDuration() Duration { return WhileOnBattlefield }

func (e *grantKeywordToAllEffect) IsActive(g *Game) bool {
	return g.FindPermanent(e.sourceID) != nil
}

func (e *grantKeywordToAllEffect) Apply(g *Game) error {
	for _, p := range g.Battlefield {
		if !p.HasType(TypeCreature) {
			continue
		}
		if p.ID() == e.sourceID {
			continue // don't grant to self
		}
		if e.filter != nil && !e.filter(p, g) {
			continue
		}
		g.Effects.grantedKW[p.ID()] = append(g.Effects.grantedKW[p.ID()], e.keyword)
		p.RuntimeAbilities = append(p.RuntimeAbilities, &grantedByEffect{NewKeywordAbility(e.keyword)})
	}
	return nil
}

// boostControlledCreaturesEffect boosts creatures controlled by the source's
// controller that match an optional filter.
type boostControlledCreaturesEffect struct {
	power     int
	toughness int
	filter    PermanentFilter
	effectSource
}

func BoostControlledCreatures(power, toughness int, filter PermanentFilter) ContinuousEffect {
	return &boostControlledCreaturesEffect{
		power:     power,
		toughness: toughness,
		filter:    filter,
	}
}

func (e *boostControlledCreaturesEffect) GetLayer() Layer       { return LayerPT }
func (e *boostControlledCreaturesEffect) GetDuration() Duration { return WhileOnBattlefield }

func (e *boostControlledCreaturesEffect) IsActive(g *Game) bool {
	return g.FindPermanent(e.sourceID) != nil
}

func (e *boostControlledCreaturesEffect) Apply(g *Game) error {
	src := g.FindPermanent(e.sourceID)
	if src == nil {
		return nil
	}
	for _, p := range g.Battlefield {
		if !p.HasType(TypeCreature) {
			continue
		}
		if p.Controller != src.Controller {
			continue
		}
		if e.filter != nil && !e.filter(p, g) {
			continue
		}
		g.Effects.powerBonuses[p.ID()] += e.power
		g.Effects.toughBonuses[p.ID()] += e.toughness
	}
	return nil
}

// controlChangeEffect is a continuous control change effect (e.g., Control Magic).
type controlChangeEffect struct {
	effectSource
}

func ControlChangeContinuous() ContinuousEffect {
	return &controlChangeEffect{}
}

func (e *controlChangeEffect) GetLayer() Layer       { return LayerControl }
func (e *controlChangeEffect) GetDuration() Duration { return WhileOnBattlefield }

func (e *controlChangeEffect) IsActive(g *Game) bool {
	src := g.FindPermanent(e.sourceID)
	return src != nil && src.IsAttached()
}

func (e *controlChangeEffect) Apply(g *Game) error {
	src := g.FindPermanent(e.sourceID)
	if src == nil || !src.IsAttached() {
		return nil
	}
	target := g.FindPermanent(src.AttachedTo)
	if target == nil {
		return nil
	}
	target.Controller = src.Controller
	return nil
}

// boostAttachedByForestCountEffect boosts the attached creature by Forests controlled.
type boostAttachedByForestCountEffect struct {
	effectSource
}

func BoostAttachedByForestCount() ContinuousEffect {
	return &boostAttachedByForestCountEffect{}
}

func (e *boostAttachedByForestCountEffect) GetLayer() Layer       { return LayerPT }
func (e *boostAttachedByForestCountEffect) GetDuration() Duration { return WhileOnBattlefield }

func (e *boostAttachedByForestCountEffect) IsActive(g *Game) bool {
	src := g.FindPermanent(e.sourceID)
	return src != nil && src.IsAttached()
}

func (e *boostAttachedByForestCountEffect) Apply(g *Game) error {
	src := g.FindPermanent(e.sourceID)
	if src == nil || !src.IsAttached() {
		return nil
	}
	// Count Forests controlled by the aura's controller
	forests := 0
	for _, p := range g.Battlefield {
		if p.Controller == src.Controller && p.HasType(TypeLand) && p.HasSubType("Forest") {
			forests++
		}
	}
	// +X/+Y where X = forests/2 rounded down, Y = forests/2 rounded up
	powerBoost := forests / 2
	toughBoost := (forests + 1) / 2
	g.Effects.powerBonuses[src.AttachedTo] += powerBoost
	g.Effects.toughBonuses[src.AttachedTo] += toughBoost
	return nil
}

// preventUntapForMatchingEffect prevents permanents matching a filter from
// untapping during their controller's untap step (e.g. Meekstone).
type preventUntapForMatchingEffect struct {
	filter PermanentFilter
	effectSource
}

// PreventUntapForMatching creates a continuous effect that sets DoesNotUntap
// on all permanents matching the given filter.
func PreventUntapForMatching(filter PermanentFilter) ContinuousEffect {
	return &preventUntapForMatchingEffect{filter: filter}
}

func (e *preventUntapForMatchingEffect) GetLayer() Layer       { return LayerAbility }
func (e *preventUntapForMatchingEffect) GetDuration() Duration { return WhileOnBattlefield }

func (e *preventUntapForMatchingEffect) IsActive(g *Game) bool {
	return g.FindPermanent(e.sourceID) != nil
}

func (e *preventUntapForMatchingEffect) Apply(g *Game) error {
	for _, p := range g.Battlefield {
		if e.filter(p, g) {
			g.Effects.grantedKW[p.ID()] = append(g.Effects.grantedKW[p.ID()], DoesNotUntapKW)
			p.RuntimeAbilities = append(p.RuntimeAbilities, &grantedByEffect{NewKeywordAbility(DoesNotUntapKW)})
		}
	}
	return nil
}

// SpellCostIncrease returns the additional generic cost for spells of the given color.
func (em *EffectManager) SpellCostIncrease(c Color) int {
	return em.spellCostIncrease[c]
}

// IncreaseSpellCostForColor is a continuous effect that increases the cost of
// spells of a given color (e.g. Gloom makes white spells cost {3} more).
func IncreaseSpellCostForColor(color Color, amount int) ContinuousEffect {
	return &increaseSpellCostEffect{color: color, amount: amount}
}

type increaseSpellCostEffect struct {
	color  Color
	amount int
	effectSource
}

func (e *increaseSpellCostEffect) GetLayer() Layer       { return LayerAbility }
func (e *increaseSpellCostEffect) GetDuration() Duration { return WhileOnBattlefield }

func (e *increaseSpellCostEffect) IsActive(g *Game) bool {
	return g.FindPermanent(e.sourceID) != nil
}

func (e *increaseSpellCostEffect) Apply(g *Game) error {
	g.Effects.spellCostIncrease[e.color] += e.amount
	return nil
}

// ChangeSubTypesForAll changes subtypes of all permanents matching fromSubTypes
// to toSubTypes (e.g. Conversion: all Mountains become Plains).
func ChangeSubTypesForAll(fromSubTypes, toSubTypes []string) ContinuousEffect {
	return &changeSubTypesForAllEffect{
		fromSubTypes: fromSubTypes,
		toSubTypes:   toSubTypes,
	}
}

type changeSubTypesForAllEffect struct {
	fromSubTypes []string
	toSubTypes   []string
	effectSource
}

func (e *changeSubTypesForAllEffect) GetLayer() Layer       { return LayerType }
func (e *changeSubTypesForAllEffect) GetDuration() Duration { return WhileOnBattlefield }

func (e *changeSubTypesForAllEffect) IsActive(g *Game) bool {
	return g.FindPermanent(e.sourceID) != nil
}

func (e *changeSubTypesForAllEffect) Apply(g *Game) error {
	// Map basic land subtypes to mana colors
	colorMap := map[string]Color{
		"Plains": White, "Island": Blue, "Swamp": Black,
		"Mountain": Red, "Forest": Green,
	}
	var newColor Color
	hasNewColor := false
	for _, st := range e.toSubTypes {
		if c, ok := colorMap[st]; ok {
			newColor = c
			hasNewColor = true
			break
		}
	}

	for _, p := range g.Battlefield {
		for _, from := range e.fromSubTypes {
			if p.HasSubType(from) {
				p.SubTypeOverride = e.toSubTypes
				// When a land's basic type changes, replace its mana ability
				if hasNewColor && p.HasType(TypeLand) {
					var filtered []Ability
					for _, a := range p.RuntimeAbilities {
						inner := UnwrapAbility(a)
						if _, ok := inner.(*ManaAbility); !ok {
							filtered = append(filtered, a)
						}
					}
					p.RuntimeAbilities = filtered
					p.RuntimeAbilities = append(p.RuntimeAbilities, &grantedByEffect{NewManaAbility(newColor)})
				}
				break
			}
		}
	}
	return nil
}

// CyclopeanTombEffect overrides subtypes of all permanents with Mire counters to Swamp.
// Sourced from Cyclopean Tomb; pruned when the Tomb leaves the battlefield.
type cyclopeanTombEffect struct {
	effectSource
}

func CyclopeanTombEffect() ContinuousEffect {
	return &cyclopeanTombEffect{}
}

func (e *cyclopeanTombEffect) GetLayer() Layer       { return LayerType }
func (e *cyclopeanTombEffect) GetDuration() Duration { return WhileOnBattlefield }

func (e *cyclopeanTombEffect) IsActive(g *Game) bool {
	return g.FindPermanent(e.sourceID) != nil
}

func (e *cyclopeanTombEffect) Apply(g *Game) error {
	for _, p := range g.Battlefield {
		if p.HasType(TypeLand) && p.Counters[Mire] > 0 {
			p.SubTypeOverride = []string{"Swamp"}
			// Replace mana ability with Black
			var filtered []Ability
			for _, a := range p.RuntimeAbilities {
				inner := UnwrapAbility(a)
				if _, ok := inner.(*ManaAbility); !ok {
					filtered = append(filtered, a)
				}
			}
			p.RuntimeAbilities = filtered
			p.RuntimeAbilities = append(p.RuntimeAbilities, &grantedByEffect{NewManaAbility(Black)})
		}
	}
	return nil
}

// keywordReplacementContinuous replaces one keyword with another on a target
// (e.g. swampwalk -> forestwalk via Sleight of Mind / Magical Hack).
type keywordReplacementContinuous struct {
	from     Keyword
	to       Keyword
	targetID uuid.UUID
	effectSource
}

func (e *keywordReplacementContinuous) GetLayer() Layer       { return LayerAbility }
func (e *keywordReplacementContinuous) GetDuration() Duration { return Indefinite }

func (e *keywordReplacementContinuous) IsActive(g *Game) bool {
	return g.FindPermanent(e.targetID) != nil
}

func (e *keywordReplacementContinuous) Apply(g *Game) error {
	perm := g.FindPermanent(e.targetID)
	if perm == nil {
		return nil
	}
	g.Effects.removedKW[e.targetID] = append(g.Effects.removedKW[e.targetID], e.from)
	g.Effects.grantedKW[e.targetID] = append(g.Effects.grantedKW[e.targetID], e.to)
	perm.RuntimeAbilities = append(perm.RuntimeAbilities, &grantedByEffect{NewKeywordAbility(e.to)})
	return nil
}

// colorOverrideContinuous permanently changes a target permanent's color (Lace cycle).
type colorOverrideContinuous struct {
	color    Color
	targetID uuid.UUID
	effectSource
}

func (e *colorOverrideContinuous) GetLayer() Layer       { return LayerColor }
func (e *colorOverrideContinuous) GetDuration() Duration { return Indefinite }

func (e *colorOverrideContinuous) IsActive(g *Game) bool {
	return g.FindPermanent(e.targetID) != nil
}

func (e *colorOverrideContinuous) Apply(g *Game) error {
	perm := g.FindPermanent(e.targetID)
	if perm == nil {
		return nil
	}
	colors := []Color{e.color}
	perm.ColorOverride = &colors
	return nil
}

// preventAllUntapsEffect prevents ALL permanents from untapping during untap steps (Stasis).
type preventAllUntapsEffect struct {
	effectSource
}

func PreventAllUntaps() ContinuousEffect {
	return &preventAllUntapsEffect{}
}

func (e *preventAllUntapsEffect) GetLayer() Layer       { return LayerAbility }
func (e *preventAllUntapsEffect) GetDuration() Duration { return WhileOnBattlefield }

func (e *preventAllUntapsEffect) IsActive(g *Game) bool {
	return g.FindPermanent(e.sourceID) != nil
}

func (e *preventAllUntapsEffect) Apply(g *Game) error {
	for _, p := range g.Battlefield {
		g.Effects.grantedKW[p.ID()] = append(g.Effects.grantedKW[p.ID()], DoesNotUntapKW)
		p.RuntimeAbilities = append(p.RuntimeAbilities, &grantedByEffect{NewKeywordAbility(DoesNotUntapKW)})
	}
	return nil
}

// boostSelfWhileControllingEffect boosts the source +P/+T while the controller
// controls a permanent matching a filter.
type boostSelfWhileControllingEffect struct {
	power     int
	toughness int
	condition PermanentFilter
	effectSource
}

func BoostSelfWhileControlling(power, toughness int, condition PermanentFilter) ContinuousEffect {
	return &boostSelfWhileControllingEffect{
		power:     power,
		toughness: toughness,
		condition: condition,
	}
}

func (e *boostSelfWhileControllingEffect) GetLayer() Layer       { return LayerPT }
func (e *boostSelfWhileControllingEffect) GetDuration() Duration { return WhileOnBattlefield }

func (e *boostSelfWhileControllingEffect) IsActive(g *Game) bool {
	return g.FindPermanent(e.sourceID) != nil
}

func (e *boostSelfWhileControllingEffect) Apply(g *Game) error {
	src := g.FindPermanent(e.sourceID)
	if src == nil {
		return nil
	}
	// Check if controller controls a matching permanent
	for _, p := range g.Battlefield {
		if p.Controller == src.Controller && e.condition(p, g) {
			g.Effects.powerBonuses[src.ID()] += e.power
			g.Effects.toughBonuses[src.ID()] += e.toughness
			return nil
		}
	}
	return nil
}

// limitLandUntapsEffect limits how many lands each player can untap per turn
// (e.g. Winter Orb). Only active while the source is untapped.
type limitLandUntapsEffect struct {
	limit int
	effectSource
}

// LimitLandUntaps creates a continuous effect that limits land untaps per turn.
// Only active while the source permanent is untapped.
func LimitLandUntaps(limit int) ContinuousEffect {
	return &limitLandUntapsEffect{limit: limit}
}

func (e *limitLandUntapsEffect) GetLayer() Layer       { return LayerAbility }
func (e *limitLandUntapsEffect) GetDuration() Duration { return WhileOnBattlefield }

func (e *limitLandUntapsEffect) IsActive(g *Game) bool {
	src := g.FindPermanent(e.sourceID)
	return src != nil && !src.Tapped
}

func (e *limitLandUntapsEffect) Apply(g *Game) error {
	if g.Effects.landUntapLimit < 0 || e.limit < g.Effects.landUntapLimit {
		g.Effects.landUntapLimit = e.limit
	}
	return nil
}

// animateLandsEffect makes matching permanents into creatures with given P/T.
// Used by Living Lands (Forests become 1/1 creatures).
type animateLandsEffect struct {
	filter    PermanentFilter
	power     int
	toughness int
	effectSource
}

// AnimateLands creates a continuous effect that turns matching lands into creatures.
func AnimateLands(filter PermanentFilter, power, toughness int) ContinuousEffect {
	return &animateLandsEffect{filter: filter, power: power, toughness: toughness}
}

func (e *animateLandsEffect) GetLayer() Layer       { return LayerType }
func (e *animateLandsEffect) GetDuration() Duration { return WhileOnBattlefield }

func (e *animateLandsEffect) IsActive(g *Game) bool {
	return g.FindPermanent(e.sourceID) != nil
}

func (e *animateLandsEffect) Apply(g *Game) error {
	for _, p := range g.Battlefield {
		if e.filter(p, g) {
			p.TypesAdded = append(p.TypesAdded, TypeCreature)
			p.BasePTOverride = &[2]int{e.power, e.toughness}
		}
	}
	return nil
}

// allowUnlimitedLandPlaysEffect lets the controller play any number of lands.
// Used by Fastbond.
type allowUnlimitedLandPlaysEffect struct {
	effectSource
}

// AllowUnlimitedLandPlays creates a continuous effect that removes the land play limit.
func AllowUnlimitedLandPlays() ContinuousEffect {
	return &allowUnlimitedLandPlaysEffect{}
}

func (e *allowUnlimitedLandPlaysEffect) GetLayer() Layer       { return LayerAbility }
func (e *allowUnlimitedLandPlaysEffect) GetDuration() Duration { return WhileOnBattlefield }

func (e *allowUnlimitedLandPlaysEffect) IsActive(g *Game) bool {
	return g.FindPermanent(e.sourceID) != nil
}

func (e *allowUnlimitedLandPlaysEffect) Apply(g *Game) error {
	g.Effects.unlimitedLandPlays = true
	return nil
}

// doppelgangerCopyEffect copies another creature's P/T and keyword abilities
// onto the Doppelganger. Operates at LayerCopy (layer 1).
type doppelgangerCopyEffect struct {
	effectSource
	doppelgangerID uuid.UUID
	copiedName     string // name of the creature being copied
	power          int
	toughness      int
	keywords       []Keyword
}

func (e *doppelgangerCopyEffect) GetLayer() Layer       { return LayerCopy }
func (e *doppelgangerCopyEffect) GetDuration() Duration { return Indefinite }

func (e *doppelgangerCopyEffect) IsActive(g *Game) bool {
	return g.FindPermanent(e.doppelgangerID) != nil
}

func (e *doppelgangerCopyEffect) Apply(g *Game) error {
	perm := g.FindPermanent(e.doppelgangerID)
	if perm == nil {
		return nil
	}
	perm.BasePTOverride = &[2]int{e.power, e.toughness}
	for _, kw := range e.keywords {
		perm.RuntimeAbilities = append(perm.RuntimeAbilities, &grantedByEffect{NewKeywordAbility(kw)})
	}
	return nil
}

// AddCopyEffect creates a doppelganger copy effect from the target creature.
func (em *EffectManager) AddCopyEffect(doppelgangerID uuid.UUID, target *Permanent) {
	keywords := extractKeywords(target)
	ce := &doppelgangerCopyEffect{
		doppelgangerID: doppelgangerID,
		copiedName:     target.Name(),
		power:          target.Card.Power(),
		toughness:      target.Card.Toughness(),
		keywords:       keywords,
	}
	ce.sourceID = doppelgangerID
	em.effects = append(em.effects, ce)
}

// UpdateCopyEffect updates the copy effect for a doppelganger to copy a new target.
func (em *EffectManager) UpdateCopyEffect(doppelgangerID uuid.UUID, target *Permanent) {
	keywords := extractKeywords(target)
	for _, e := range em.effects {
		if ce, ok := e.(*doppelgangerCopyEffect); ok && ce.doppelgangerID == doppelgangerID {
			ce.copiedName = target.Name()
			ce.power = target.Card.Power()
			ce.toughness = target.Card.Toughness()
			ce.keywords = keywords
			return
		}
	}
	// No existing effect found, create a new one
	em.AddCopyEffect(doppelgangerID, target)
}

// CopyEffectCurrentName returns the name of the creature currently being copied
// by the doppelganger, or "" if no copy effect exists.
func (em *EffectManager) CopyEffectCurrentName(doppelgangerID uuid.UUID) string {
	for _, e := range em.effects {
		if ce, ok := e.(*doppelgangerCopyEffect); ok && ce.doppelgangerID == doppelgangerID {
			return ce.copiedName
		}
	}
	return ""
}

// extractKeywords returns the keyword abilities from a permanent's card.
func extractKeywords(p *Permanent) []Keyword {
	var keywords []Keyword
	for _, a := range p.Card.Abilities() {
		if ka, ok := a.(*KeywordAbility); ok {
			keywords = append(keywords, ka.Keyword)
		}
	}
	return keywords
}

// manaConversionEffect sets a mana conversion on the EffectManager while the
// source permanent is on the battlefield (e.g. Sunglasses of Urza: red→white).
type manaConversionEffect struct {
	from Color
	to   Color
	effectSource
}

// ManaConversion creates a continuous effect that allows spending one color as another.
func ManaConversion(from, to Color) ContinuousEffect {
	return &manaConversionEffect{from: from, to: to}
}

func (e *manaConversionEffect) GetLayer() Layer       { return LayerAbility }
func (e *manaConversionEffect) GetDuration() Duration { return WhileOnBattlefield }

func (e *manaConversionEffect) IsActive(g *Game) bool {
	return g.FindPermanent(e.sourceID) != nil
}

func (e *manaConversionEffect) Apply(g *Game) error {
	g.Effects.SetManaConversion(e.from, e.to)
	return nil
}

// bodyguardEffect makes an untapped creature redirect combat damage from its
// controller to itself (Veteran Bodyguard).
type bodyguardEffect struct {
	effectSource
}

// BodyguardContinuous creates a continuous effect for Veteran Bodyguard.
func BodyguardContinuous() ContinuousEffect {
	return &bodyguardEffect{}
}

func (e *bodyguardEffect) GetLayer() Layer       { return LayerAbility }
func (e *bodyguardEffect) GetDuration() Duration { return WhileOnBattlefield }

func (e *bodyguardEffect) IsActive(g *Game) bool {
	src := g.FindPermanent(e.sourceID)
	return src != nil && !src.Tapped
}

func (e *bodyguardEffect) Apply(g *Game) error {
	src := g.FindPermanent(e.sourceID)
	if src == nil || src.Tapped {
		return nil
	}
	g.Effects.SetBodyguard(src.Controller, src.ID())
	return nil
}

// personalIncarnationEffect redirects ALL damage from a player to Personal Incarnation.
// Unlike bodyguardEffect (combat damage only), this applies to all damage sources.
type personalIncarnationEffect struct {
	effectSource
}

// PersonalIncarnationRedirect creates a continuous effect for Personal Incarnation.
func PersonalIncarnationRedirect() ContinuousEffect {
	return &personalIncarnationEffect{}
}

func (e *personalIncarnationEffect) GetLayer() Layer       { return LayerAbility }
func (e *personalIncarnationEffect) GetDuration() Duration { return WhileOnBattlefield }

func (e *personalIncarnationEffect) IsActive(g *Game) bool {
	src := g.FindPermanent(e.sourceID)
	return src != nil
}

func (e *personalIncarnationEffect) Apply(g *Game) error {
	src := g.FindPermanent(e.sourceID)
	if src == nil {
		return nil
	}
	g.Effects.SetPlayerDamageRedirect(src.Controller, src.ID())
	return nil
}

// temporaryAnimateEffect turns a specific permanent into a creature with given
// P/T until end of turn (e.g. Jade Statue becoming a 3/6 creature).
type temporaryAnimateEffect struct {
	targetID  uuid.UUID
	power     int
	toughness int
	duration  Duration
	effectSource
}

// TemporaryAnimate creates an end-of-turn effect that animates a permanent.
func TemporaryAnimate(targetID uuid.UUID, power, toughness int) ContinuousEffect {
	return &temporaryAnimateEffect{
		targetID:  targetID,
		power:     power,
		toughness: toughness,
		duration:  EndOfTurn,
	}
}

// TemporaryAnimateUntilEndOfCombat creates an end-of-combat effect that animates a permanent.
func TemporaryAnimateUntilEndOfCombat(targetID uuid.UUID, power, toughness int) ContinuousEffect {
	return &temporaryAnimateEffect{
		targetID:  targetID,
		power:     power,
		toughness: toughness,
		duration:  EndOfCombat,
	}
}

func (e *temporaryAnimateEffect) GetLayer() Layer       { return LayerType }
func (e *temporaryAnimateEffect) GetDuration() Duration { return e.duration }

func (e *temporaryAnimateEffect) IsActive(g *Game) bool {
	return g.FindPermanent(e.targetID) != nil
}

func (e *temporaryAnimateEffect) Apply(g *Game) error {
	perm := g.FindPermanent(e.targetID)
	if perm == nil {
		return nil
	}
	perm.TypesAdded = append(perm.TypesAdded, TypeCreature)
	perm.BasePTOverride = &[2]int{e.power, e.toughness}
	return nil
}
