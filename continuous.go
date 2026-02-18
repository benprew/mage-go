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
	UntilYourNextTurn
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

func (s *effectSource) SourceID() uuid.UUID     { return s.sourceID }
func (s *effectSource) SetSourceID(id uuid.UUID) { s.sourceID = id }


// EffectManager manages and applies continuous effects.
type EffectManager struct {
	effects      []ContinuousEffect
	powerBonuses map[uuid.UUID]int
	toughBonuses map[uuid.UUID]int
	grantedKW    map[uuid.UUID][]Keyword
	removedKW    map[uuid.UUID][]Keyword
	preventAttack map[uuid.UUID]bool
	regenerationShields map[uuid.UUID]int
	preventionShields   map[uuid.UUID]int
	landUntapLimit      int // -1 = no limit; >= 0 = max lands that may untap per turn
	forcefieldShields   map[uuid.UUID]bool // players with Forcefield active this turn
}

func NewEffectManager() *EffectManager {
	return &EffectManager{
		powerBonuses:        make(map[uuid.UUID]int),
		toughBonuses:        make(map[uuid.UUID]int),
		grantedKW:           make(map[uuid.UUID][]Keyword),
		removedKW:           make(map[uuid.UUID][]Keyword),
		preventAttack:       make(map[uuid.UUID]bool),
		regenerationShields: make(map[uuid.UUID]int),
		preventionShields:   make(map[uuid.UUID]int),
		landUntapLimit:      -1,
		forcefieldShields:   make(map[uuid.UUID]bool),
	}
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

// Apply resets computed bonuses and reapplies all active effects in layer order.
func (em *EffectManager) Apply(g *Game) {
	em.powerBonuses = make(map[uuid.UUID]int)
	em.toughBonuses = make(map[uuid.UUID]int)
	em.grantedKW = make(map[uuid.UUID][]Keyword)
	em.removedKW = make(map[uuid.UUID][]Keyword)
	em.preventAttack = make(map[uuid.UUID]bool)
	em.landUntapLimit = -1

	// Reset granted runtime abilities from effects (will be re-granted below)
	for _, p := range g.Battlefield {
		var base []Ability
		for _, a := range p.RuntimeAbilities {
			if _, ok := a.(*grantedByEffect); !ok {
				base = append(base, a)
			}
		}
		p.RuntimeAbilities = base
	}

	// Remove effects whose source is no longer on the battlefield
	// EndOfTurn effects persist until cleanup regardless of source (e.g., Giant Growth)
	active := em.effects[:0]
	for _, e := range em.effects {
		if e.GetDuration() == EndOfTurn || g.FindPermanent(e.SourceID()) != nil {
			active = append(active, e)
		}
	}
	em.effects = active

	// Apply in layer order (2, 6, 7)
	for _, layer := range []Layer{LayerControl, LayerAbility, LayerPT} {
		for _, e := range em.effects {
			if e.GetLayer() == layer && e.IsActive(g) {
				e.Apply(g)
			}
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

func (e *boostAttachedEffect) GetLayer() Layer      { return LayerPT }
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

func (e *grantKeywordAttachedEffect) GetLayer() Layer      { return LayerAbility }
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
	target.RuntimeAbilities = append(target.RuntimeAbilities, &grantedByEffect{HasKeyword(e.keyword)})
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

func (e *removeKeywordAttachedEffect) GetLayer() Layer      { return LayerAbility }
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

func (e *grantActivatedAbilityAttachedEffect) GetLayer() Layer      { return LayerAbility }
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
	effect    Effect
	cost      Cost
	filter    PermanentFilter
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

func (e *grantActivatedAbilityToAllEffect) GetLayer() Layer      { return LayerAbility }
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

func (e *preventUntapEffect) GetLayer() Layer      { return LayerAbility }
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
		target.RuntimeAbilities = append(target.RuntimeAbilities, &grantedByEffect{HasKeyword(DoesNotUntapKW)})
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

func (e *preventAttackEffect) GetLayer() Layer      { return LayerAbility }
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

func (e *temporaryBoostEffect) GetLayer() Layer      { return LayerPT }
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
	targetID  uuid.UUID
	keyword   Keyword
	effectSource
}

func (e *temporaryKeywordEffect) GetLayer() Layer      { return LayerAbility }
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
	target.RuntimeAbilities = append(target.RuntimeAbilities, &grantedByEffect{HasKeyword(e.keyword)})
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

func (e *boostAllCreaturesEffect) GetLayer() Layer      { return LayerPT }
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

func (e *ptEqualsCountEffect) GetLayer() Layer      { return LayerPT }
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

func (e *powerEqualsCountEffect) GetLayer() Layer      { return LayerPT }
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
	keyword   Keyword
	filter    PermanentFilter
	effectSource
}

func GrantKeywordToAll(kw Keyword, filter PermanentFilter) ContinuousEffect {
	return &grantKeywordToAllEffect{
		keyword: kw,
		filter:  filter,
	}
}

func (e *grantKeywordToAllEffect) GetLayer() Layer      { return LayerAbility }
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
		p.RuntimeAbilities = append(p.RuntimeAbilities, &grantedByEffect{HasKeyword(e.keyword)})
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

func (e *boostControlledCreaturesEffect) GetLayer() Layer      { return LayerPT }
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

func (e *controlChangeEffect) GetLayer() Layer      { return LayerControl }
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

func (e *boostAttachedByForestCountEffect) GetLayer() Layer      { return LayerPT }
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

func (e *preventUntapForMatchingEffect) GetLayer() Layer      { return LayerAbility }
func (e *preventUntapForMatchingEffect) GetDuration() Duration { return WhileOnBattlefield }

func (e *preventUntapForMatchingEffect) IsActive(g *Game) bool {
	return g.FindPermanent(e.sourceID) != nil
}

func (e *preventUntapForMatchingEffect) Apply(g *Game) error {
	for _, p := range g.Battlefield {
		if e.filter(p, g) {
			g.Effects.grantedKW[p.ID()] = append(g.Effects.grantedKW[p.ID()], DoesNotUntapKW)
			p.RuntimeAbilities = append(p.RuntimeAbilities, &grantedByEffect{HasKeyword(DoesNotUntapKW)})
		}
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

func (e *boostSelfWhileControllingEffect) GetLayer() Layer      { return LayerPT }
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

func (e *limitLandUntapsEffect) GetLayer() Layer      { return LayerAbility }
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
