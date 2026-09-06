package mage

import (
	"slices"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// Target represents a targeting requirement.
type Target interface {
	Possible(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID
	Choose(controller uuid.UUID, sourceCard Card, g *Game, chosen []uuid.UUID) error
	Chosen() []uuid.UUID
	IsChosen() bool
	Min() int
	Max() int
	Reset()
}

// BaseTarget provides common target fields.
type BaseTarget struct {
	chosen []uuid.UUID
	min    int
	max    int
}

// VariableTarget reports target-count bounds that depend on the announced X.
type VariableTarget interface {
	Target
	BoundsForX(x int) (minimum, maximum int)
}

// TargetBounds returns a target's bounds for the announced X value.
func TargetBounds(target Target, x int) (minimum, maximum int) {
	if variable, ok := target.(VariableTarget); ok {
		return variable.BoundsForX(x)
	}
	return target.Min(), target.Max()
}

type countBoundTarget struct {
	inner  Target
	count  int
	usesX  bool
	upToX  bool
	oneToX bool
}

func (t *countBoundTarget) Possible(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID {
	return t.inner.Possible(controller, sourceCard, g)
}
func (t *countBoundTarget) Choose(controller uuid.UUID, sourceCard Card, g *Game, chosen []uuid.UUID) error {
	return t.inner.Choose(controller, sourceCard, g, chosen)
}
func (t *countBoundTarget) Chosen() []uuid.UUID { return t.inner.Chosen() }
func (t *countBoundTarget) IsChosen() bool      { return t.inner.IsChosen() }
func (t *countBoundTarget) Min() int {
	if t.upToX {
		return 0
	}
	if t.oneToX {
		return 0
	}
	if t.usesX {
		return 0
	}
	return t.count
}
func (t *countBoundTarget) Max() int {
	if t.usesX || t.upToX || t.oneToX {
		return 100
	}
	return t.count
}
func (t *countBoundTarget) Reset() { t.inner.Reset() }
func (t *countBoundTarget) BoundsForX(x int) (minimum, maximum int) {
	if t.upToX {
		return 0, max(x, 0)
	}
	if t.oneToX {
		if x <= 0 {
			return 0, 0
		}
		return 1, x
	}
	if t.usesX {
		return max(x, 0), max(x, 0)
	}
	return t.count, t.count
}

// randomTarget wraps an ordinary targeting requirement. The wrapped target
// still owns legality and count bounds; this wrapper changes only who selects
// the identities and, optionally, the count.
type randomTarget struct {
	inner       Target
	randomCount bool
}

// TargetRandom makes the engine choose the wrapped target's identities
// uniformly at random without replacement. Fixed counts are preserved. For a
// variable count, the controller chooses only the number of targets.
func TargetRandom(inner Target) Target {
	return &randomTarget{inner: inner}
}

// TargetRandomCount makes the engine choose both the target count and the
// distinct target identities uniformly at random from the wrapped target's
// currently legal bounds.
func TargetRandomCount(inner Target) Target {
	return &randomTarget{inner: inner, randomCount: true}
}

func (t *randomTarget) Possible(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID {
	return t.inner.Possible(controller, sourceCard, g)
}
func (t *randomTarget) Choose(controller uuid.UUID, sourceCard Card, g *Game, chosen []uuid.UUID) error {
	return t.inner.Choose(controller, sourceCard, g, chosen)
}
func (t *randomTarget) Chosen() []uuid.UUID { return t.inner.Chosen() }
func (t *randomTarget) IsChosen() bool      { return t.inner.IsChosen() }
func (t *randomTarget) Min() int            { return t.inner.Min() }
func (t *randomTarget) Max() int            { return t.inner.Max() }
func (t *randomTarget) Reset()              { t.inner.Reset() }
func (t *randomTarget) BoundsForX(x int) (minimum, maximum int) {
	return TargetBounds(t.inner, x)
}

func isRandomTarget(t Target) bool {
	_, ok := t.(*randomTarget)
	return ok
}

func uniqueTargetCandidates(ids []uuid.UUID) []uuid.UUID {
	unique := make([]uuid.UUID, 0, len(ids))
	seen := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		if id == uuid.Nil || seen[id] {
			continue
		}
		seen[id] = true
		unique = append(unique, id)
	}
	return unique
}

func (g *Game) chooseRandomTargets(controller uuid.UUID, sourceCard Card, target *randomTarget, x int) []uuid.UUID {
	if target == nil {
		return nil
	}
	target.Reset()
	candidates := uniqueTargetCandidates(target.Possible(controller, sourceCard, g))
	minimum, maximum := TargetBounds(target, x)
	maximum = min(maximum, len(candidates))
	if maximum < minimum {
		return nil
	}
	count := minimum
	if maximum > minimum {
		if target.randomCount {
			count += g.RandIntn(maximum - minimum + 1)
		} else if player := g.GetPlayer(controller); player != nil {
			count = player.ChooseNumber(minimum, maximum, "choose number of random targets")
			count = min(max(count, minimum), maximum)
		}
	}
	chosen := make([]uuid.UUID, 0, count)
	for range count {
		idx := g.RandIntn(len(candidates))
		chosen = append(chosen, candidates[idx])
		candidates = append(candidates[:idx], candidates[idx+1:]...)
	}
	_ = target.Choose(controller, sourceCard, g, chosen)
	return chosen
}

// acquireRandomTargets replaces random target declarations with engine-chosen
// identities. Supplied UUIDs correspond only to ordinary declarations; random
// declarations obtain their variable count from the controller or RNG.
func (g *Game) acquireRandomTargets(controller uuid.UUID, sourceCard Card, specs []Target, supplied []uuid.UUID, x int) []uuid.UUID {
	if !slices.ContainsFunc(specs, isRandomTarget) {
		return supplied
	}
	out := make([]uuid.UUID, 0, len(supplied))
	offset := 0
	hasOrdinary := false
	for i, spec := range specs {
		if random, ok := spec.(*randomTarget); ok {
			out = append(out, g.chooseRandomTargets(controller, sourceCard, random, x)...)
			continue
		}
		hasOrdinary = true
		_, maximum := TargetBounds(spec, x)
		remainingRequired := 0
		for _, later := range specs[i+1:] {
			if isRandomTarget(later) {
				continue
			}
			laterMinimum, _ := TargetBounds(later, x)
			remainingRequired += laterMinimum
		}
		available := max(0, len(supplied)-offset-remainingRequired)
		count := min(maximum, available)
		out = append(out, supplied[offset:offset+count]...)
		offset += count
	}
	if hasOrdinary {
		out = append(out, supplied[offset:]...)
	}
	return out
}

type randomActivePlayerExchangePairTarget struct {
	BaseTarget
}

// TargetRandomActivePlayerExchangePair chooses two targets for Power
// Struggle-style triggers. The first is an active-player-controlled artifact,
// creature, or land with a compatible opposing partner; the second is a
// uniformly random compatible opposing permanent.
func TargetRandomActivePlayerExchangePair() Target {
	return &randomActivePlayerExchangePairTarget{BaseTarget: BaseTarget{min: 0, max: 2}}
}

func (t *randomActivePlayerExchangePairTarget) Possible(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID {
	possible := make([]uuid.UUID, 0)
	for _, permanent := range g.battlefield {
		if !isExchangePermanent(permanent) || !permanent.CanBeTargetedBy(sourceCard, controller, g) {
			continue
		}
		possible = append(possible, permanent.ID())
	}
	return possible
}

func (t *randomActivePlayerExchangePairTarget) Choose(_ uuid.UUID, _ Card, _ *Game, chosen []uuid.UUID) error {
	t.chosen = append([]uuid.UUID(nil), chosen...)
	return nil
}

func (t *randomActivePlayerExchangePairTarget) chooseForTrigger(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID {
	t.Reset()
	active := g.ActivePlayerObj()
	if active == nil {
		return nil
	}
	activeID := active.PlayerID()
	targetable := make([]*Permanent, 0)
	for _, permanent := range g.battlefield {
		if isExchangePermanent(permanent) && permanent.CanBeTargetedBy(sourceCard, controller, g) {
			targetable = append(targetable, permanent)
		}
	}
	type pairCandidate struct {
		first    *Permanent
		partners []*Permanent
	}
	pairs := make([]pairCandidate, 0)
	for _, first := range targetable {
		if first.ControllerID() != activeID {
			continue
		}
		candidate := pairCandidate{first: first}
		for _, second := range targetable {
			if second.ControllerID() == activeID || !shareExchangePermanentType(first, second) {
				continue
			}
			candidate.partners = append(candidate.partners, second)
		}
		if len(candidate.partners) > 0 {
			pairs = append(pairs, candidate)
		}
	}
	if len(pairs) == 0 {
		return nil
	}
	pair := pairs[g.RandIntn(len(pairs))]
	second := pair.partners[g.RandIntn(len(pair.partners))]
	chosen := []uuid.UUID{pair.first.ID(), second.ID()}
	_ = t.Choose(controller, sourceCard, g, chosen)
	return chosen
}

func isExchangePermanent(permanent *Permanent) bool {
	return permanent != nil && (permanent.HasType(TypeArtifact) || permanent.HasType(TypeCreature) || permanent.HasType(TypeLand))
}

func (t *BaseTarget) Chosen() []uuid.UUID { return t.chosen }
func (t *BaseTarget) IsChosen() bool      { return len(t.chosen) > 0 }
func (t *BaseTarget) Min() int            { return t.min }
func (t *BaseTarget) Max() int            { return t.max }
func (t *BaseTarget) Reset()              { t.chosen = nil }

// opponentChosenTarget wraps another Target so the *opponent* of the
// trigger's controller is prompted to choose, not the controller. Used
// for "of an opponent's choice" effects (Mausoleum Turnkey).
//
// The wrapper forwards Possible/Choose/Min/Max/Reset and exposes
// OpponentChoosesTarget() bool so chooseTriggerTargets routes the
// ChooseTargets prompt to the opponent.
type opponentChosenTarget struct {
	inner Target
}

// TargetOpponentChoice wraps inner so the opponent of the ability's
// controller picks the target from inner's legal candidates. Combine
// with any existing Target constructor:
//
//	mage.TargetOpponentChoice(mage.TargetUpToNCardsInYourGraveyard(1, mage.IsCreatureCard))
func TargetOpponentChoice(inner Target) Target {
	return &opponentChosenTarget{inner: inner}
}

func (t *opponentChosenTarget) Possible(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID {
	return t.inner.Possible(controller, sourceCard, g)
}

func (t *opponentChosenTarget) Choose(controller uuid.UUID, sourceCard Card, g *Game, chosen []uuid.UUID) error {
	return t.inner.Choose(controller, sourceCard, g, chosen)
}

func (t *opponentChosenTarget) Chosen() []uuid.UUID         { return t.inner.Chosen() }
func (t *opponentChosenTarget) IsChosen() bool              { return t.inner.IsChosen() }
func (t *opponentChosenTarget) Min() int                    { return t.inner.Min() }
func (t *opponentChosenTarget) Max() int                    { return t.inner.Max() }
func (t *opponentChosenTarget) Reset()                      { t.inner.Reset() }
func (t *opponentChosenTarget) OpponentChoosesTarget() bool { return true }

func isOpponentChosenTarget(t Target) bool {
	opponentChoice, ok := t.(interface{ OpponentChoosesTarget() bool })
	return ok && opponentChoice.OpponentChoosesTarget()
}

func hasOpponentChosenTarget(targets []Target) bool {
	return slices.ContainsFunc(targets, isOpponentChosenTarget)
}

// acquireOpponentChosenTargets replaces the supplied slice positions owned by
// TargetOpponentChoice with choices made by the opposing player. Legal
// candidates are still computed using the action's controller and source, so
// changing who makes the choice does not change control restrictions,
// protection, hexproof, or any other targeting requirement.
func (g *Game) acquireOpponentChosenTargets(controller uuid.UUID, sourceCard Card, specs []Target, supplied []uuid.UUID) []uuid.UUID {
	if len(specs) == 0 {
		return supplied
	}
	if !hasOpponentChosenTarget(specs) {
		return supplied
	}

	out := make([]uuid.UUID, 0, len(supplied))
	offset := 0
	for i, spec := range specs {
		remainingRequired := 0
		for _, later := range specs[i+1:] {
			remainingRequired += later.Min()
		}
		available := max(0, len(supplied)-offset-remainingRequired)
		count := min(spec.Max(), available)
		if count < spec.Min() {
			count = spec.Min()
		}
		consumed := min(count, len(supplied)-offset)

		if !isOpponentChosenTarget(spec) {
			out = append(out, supplied[offset:offset+consumed]...)
			offset += consumed
			continue
		}

		spec.Reset()
		possible := spec.Possible(controller, sourceCard, g)
		chooser := g.GetOpponent(controller)
		var requested []uuid.UUID
		if chooser != nil && len(possible) > 0 {
			requested = chooser.ChooseTargets(possible, spec.Min(), spec.Max(), g)
		}
		chosen := make([]uuid.UUID, 0, min(spec.Max(), len(requested)))
		for _, id := range requested {
			if len(chosen) == spec.Max() {
				break
			}
			if slices.Contains(possible, id) && !slices.Contains(chosen, id) {
				chosen = append(chosen, id)
			}
		}
		for _, id := range possible {
			if len(chosen) >= spec.Min() {
				break
			}
			if !slices.Contains(chosen, id) {
				chosen = append(chosen, id)
			}
		}
		if len(chosen) == 0 && spec.Min() > 0 {
			chosen = append(chosen, uuid.Nil)
		}
		_ = spec.Choose(controller, sourceCard, g, chosen)
		out = append(out, chosen...)
		offset += consumed
	}
	out = append(out, supplied[offset:]...)
	return out
}

func expandTargetSpecs(specs []Target, chosen []uuid.UUID, x int) []Target {
	if len(specs) == 0 || len(chosen) == 0 {
		return nil
	}
	expanded := make([]Target, 0, len(chosen))
	offset := 0
	for i, spec := range specs {
		minimum, maximum := TargetBounds(spec, x)
		remainingRequired := 0
		for _, later := range specs[i+1:] {
			laterMinimum, _ := TargetBounds(later, x)
			remainingRequired += laterMinimum
		}
		available := max(0, len(chosen)-offset-remainingRequired)
		count := min(maximum, available)
		if count < minimum {
			count = min(minimum, len(chosen)-offset)
		}
		for range count {
			expanded = append(expanded, spec)
		}
		offset += count
	}
	for len(expanded) < len(chosen) {
		expanded = append(expanded, nil)
	}
	return expanded
}

// CreatureTarget targets a creature on the battlefield.
type CreatureTarget struct {
	BaseTarget
	Filters        []PermanentFilter
	excludeSource  bool
	controllerOnly bool
	opponentOnly   bool
}

// TargetCreature creates a target that selects a creature on the battlefield,
// optionally narrowed by PermanentFilter predicates.
func TargetCreature(filters ...PermanentFilter) Target {
	return &CreatureTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
		Filters:    filters,
	}
}

// TargetNCreatures requires exactly n distinct target creatures.
func TargetNCreatures(n int, filters ...PermanentFilter) Target {
	return &countBoundTarget{inner: TargetCreature(filters...), count: n}
}

// TargetXCreatures requires exactly the announced value of X in distinct
// target creatures.
func TargetXCreatures(filters ...PermanentFilter) Target {
	return &countBoundTarget{inner: TargetCreature(filters...), usesX: true}
}

// TargetUpToXCreatures allows from zero through the announced value of X
// distinct creature targets.
func TargetUpToXCreatures(filters ...PermanentFilter) Target {
	return &countBoundTarget{inner: TargetCreature(filters...), upToX: true}
}

// TargetOneToXCreatures requires between one and the announced value of X
// distinct creature targets. When X is zero, it requires zero targets.
func TargetOneToXCreatures(filters ...PermanentFilter) Target {
	return &countBoundTarget{inner: TargetCreature(filters...), oneToX: true}
}

// TargetOtherCreature creates a target that selects a creature other than the
// source permanent, optionally narrowed by PermanentFilter predicates.
func TargetOtherCreature(filters ...PermanentFilter) Target {
	return &CreatureTarget{
		BaseTarget:    BaseTarget{min: 1, max: 1},
		Filters:       filters,
		excludeSource: true,
	}
}

// TargetCreatureYouControl creates a target that selects a creature you control,
// optionally narrowed by PermanentFilter predicates.
func TargetCreatureYouControl(filters ...PermanentFilter) Target {
	return &CreatureTarget{
		BaseTarget:     BaseTarget{min: 1, max: 1},
		Filters:        filters,
		controllerOnly: true,
	}
}

// TargetAnotherCreatureYouControl creates a target that selects a creature
// you control other than the source permanent, optionally narrowed by
// PermanentFilter predicates. Used by Kira-style abilities that target
// "another creature you control".
func TargetAnotherCreatureYouControl(filters ...PermanentFilter) Target {
	return &CreatureTarget{
		BaseTarget:     BaseTarget{min: 1, max: 1},
		Filters:        filters,
		controllerOnly: true,
		excludeSource:  true,
	}
}

// TargetCreatureOpponentControls creates a target that selects a creature an
// opponent controls, optionally narrowed by PermanentFilter predicates. This
// pairs with TargetCreatureYouControl in a multi-target spell whose targets
// must have distinct controllers (e.g. Peel from Reality, Nature's Way).
func TargetCreatureOpponentControls(filters ...PermanentFilter) Target {
	return &CreatureTarget{
		BaseTarget:   BaseTarget{min: 1, max: 1},
		Filters:      filters,
		opponentOnly: true,
	}
}

// TargetUpToNCreatures creates a target that selects from 0 up to n creatures
// on the battlefield, optionally narrowed by PermanentFilter predicates. Use
// this for "up to N target creatures" spells (Dauntless Onslaught, Tandem
// Tactics, Rishkar's ETB).
func TargetUpToNCreatures(n int, filters ...PermanentFilter) Target {
	return &CreatureTarget{
		BaseTarget: BaseTarget{min: 0, max: n},
		Filters:    filters,
	}
}

// TargetUpToOneCreature creates a target that selects from 0 up to one
// creature on the battlefield, optionally narrowed by PermanentFilter
// predicates. Per CR 115.1b, "up to one target creature" abilities can
// resolve with no chosen target. Used by Brightmare ("tap up to one
// target creature") and similar effects.
func TargetUpToOneCreature(filters ...PermanentFilter) Target {
	return TargetUpToNCreatures(1, filters...)
}

// TargetUpToNCreaturesYouControl creates a target that selects from 0 up to n
// creatures the controller controls, optionally narrowed by PermanentFilter
// predicates. Used for "any number of target creatures you control" effects
// such as Rabid Attack.
func TargetUpToNCreaturesYouControl(n int, filters ...PermanentFilter) Target {
	return &CreatureTarget{
		BaseTarget:     BaseTarget{min: 0, max: n},
		Filters:        filters,
		controllerOnly: true,
	}
}

// TargetUpToNCreaturesOpponentControls creates a target that selects from zero
// up to n creatures controlled by an opponent.
func TargetUpToNCreaturesOpponentControls(n int, filters ...PermanentFilter) Target {
	return &CreatureTarget{
		BaseTarget:   BaseTarget{min: 0, max: n},
		Filters:      filters,
		opponentOnly: true,
	}
}

// TargetUpToNCreaturesOrPlayers creates a target that selects from 0 up to n
// creatures or players. Used for divided-damage spells like Flames of the
// Firebrand ("3 damage divided as you choose among any number of targets").
func TargetUpToNCreaturesOrPlayers(n int) Target {
	return &DamageAnyTarget{
		BaseTarget: BaseTarget{min: 0, max: n},
	}
}

// TargetUpToNCardsInYourGraveyard creates a target that selects from 0 up to n
// cards in the controller's graveyard, optionally narrowed by CardFilter
// predicates. Used for Macabre Waltz, Soul Salvage, etc.
func TargetUpToNCardsInYourGraveyard(n int, filters ...CardFilter) Target {
	return &GraveyardCardTarget{
		BaseTarget: BaseTarget{min: 0, max: n},
		Filters:    filters,
	}
}

func (t *CreatureTarget) Possible(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID {
	var result []uuid.UUID
	sourceID := sourceCard.ID()
	for _, p := range g.battlefield {
		if !p.HasType(TypeCreature) {
			continue
		}
		if t.excludeSource && p.ID() == sourceID {
			continue
		}
		if t.controllerOnly && p.ControllerID() != controller {
			continue
		}
		if t.opponentOnly && p.ControllerID() == controller {
			continue
		}
		if !p.CanBeTargetedBy(sourceCard, controller, g) {
			continue
		}
		match := true
		for _, f := range t.Filters {
			if !f.Match(p, g) {
				match = false
				break
			}
		}
		if match {
			result = append(result, p.ID())
		}
	}
	return result
}

func (t *CreatureTarget) Choose(controller uuid.UUID, sourceCard Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}

// Filter returns a combined PermanentFilter that checks creature type and all filters.
func (t *CreatureTarget) Filter() PermanentFilter {
	return And(append([]PermanentFilter{IsCreature}, t.Filters...)...)
}

// PlayerTarget targets a player.
type PlayerTarget struct {
	BaseTarget
}

// TargetPlayer creates a target that selects any player in the game.
func TargetPlayer() Target {
	return &PlayerTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
	}
}

func (t *PlayerTarget) Possible(controller uuid.UUID, _ Card, g *Game) []uuid.UUID {
	result := make([]uuid.UUID, 0, len(g.players))
	for _, p := range g.players {
		result = append(result, p.PlayerID())
	}
	return result
}

func (t *PlayerTarget) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}

// ControllerTarget auto-selects the controller as the target. Used for effects
// that always apply to "you" (e.g. Conservator's "prevent damage to you").
type ControllerTarget struct {
	BaseTarget
}

// TargetController creates a target that auto-resolves to the ability's controller.
func TargetController() Target {
	return &ControllerTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
	}
}

func (t *ControllerTarget) Possible(controller uuid.UUID, _ Card, _ *Game) []uuid.UUID {
	return []uuid.UUID{controller}
}

func (t *ControllerTarget) Choose(controller uuid.UUID, _ Card, _ *Game, _ []uuid.UUID) error {
	t.chosen = []uuid.UUID{controller}
	return nil
}

// ForcedActivationTargets returns the auto-chosen targets for an activated
// ability whose targeting offers no genuine choice — every declared target
// requires a fixed number of objects (Min == Max > 0) and exactly that many
// are legal. Examples: ControllerTarget's "you", or a "target creature"
// ability when only one creature is in play. It returns nil when any target
// involves a real decision (an optional target, or more candidates than
// required), so the controller is prompted normally.
//
// The interactive layer uses this to suppress pointless "pick a target"
// prompts, and ActivateAbilityByIndex uses it to fill those targets in, since
// activated-ability targets are otherwise supplied by the caller.
func ForcedActivationTargets(controller uuid.UUID, card Card, declared []Target, g *Game) []uuid.UUID {
	if len(declared) == 0 {
		return nil
	}
	var out []uuid.UUID
	for _, t := range declared {
		t.Reset()
		possible := t.Possible(controller, card, g)
		if t.Min() <= 0 || t.Min() != t.Max() || len(possible) != t.Min() {
			return nil
		}
		out = append(out, possible...)
	}
	return out
}

// DamageAnyTarget targets "any target" for damage: a creature, planeswalker, or player (CR 115.4).
type DamageAnyTarget struct {
	BaseTarget
}

// TargetDamageAnyTarget creates a target that selects any creature, planeswalker, or player ("any target").
func TargetDamageAnyTarget() Target {
	return &DamageAnyTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
	}
}

func (t *DamageAnyTarget) Possible(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID {
	var result []uuid.UUID
	for _, p := range g.battlefield {
		if (p.HasType(TypeCreature) || p.HasType(TypePlaneswalker)) && p.CanBeTargetedBy(sourceCard, controller, g) {
			result = append(result, p.ID())
		}
	}
	for _, p := range g.players {
		result = append(result, p.PlayerID())
	}
	return result
}

func (t *DamageAnyTarget) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}

// PlayerOrPlaneswalkerTarget targets a player or a planeswalker (e.g. "target
// player or planeswalker" damage).
type PlayerOrPlaneswalkerTarget struct {
	BaseTarget
}

// TargetPlayerOrPlaneswalker creates a target that selects a player or planeswalker.
func TargetPlayerOrPlaneswalker() Target {
	return &PlayerOrPlaneswalkerTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
	}
}

func (t *PlayerOrPlaneswalkerTarget) Possible(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID {
	var result []uuid.UUID
	for _, p := range g.battlefield {
		if p.HasType(TypePlaneswalker) && p.CanBeTargetedBy(sourceCard, controller, g) {
			result = append(result, p.ID())
		}
	}
	for _, p := range g.players {
		result = append(result, p.PlayerID())
	}
	return result
}

func (t *PlayerOrPlaneswalkerTarget) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}

// TargetCreatureInYourGraveyard creates a target that selects a creature card in the controller's graveyard.
// Convenience wrapper for TargetCardInYourGraveyard(IsCreatureCard).
func TargetCreatureInYourGraveyard() Target {
	return TargetCardInYourGraveyard(IsCreatureCard)
}

// ControlledCreatureTarget targets a creature you control.
type ControlledCreatureTarget struct {
	BaseTarget
}

// TargetControlledCreature creates a target that selects a creature the controller owns.
func TargetControlledCreature() Target {
	return &ControlledCreatureTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
	}
}

func (t *ControlledCreatureTarget) Possible(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID {
	var result []uuid.UUID
	for _, p := range g.battlefield {
		if p.HasType(TypeCreature) && p.ControllerID() == controller && p.CanBeTargetedBy(sourceCard, controller, g) {
			result = append(result, p.ID())
		}
	}
	return result
}

func (t *ControlledCreatureTarget) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}

// PermanentTarget targets any permanent on the battlefield with optional filters.
type PermanentTarget struct {
	BaseTarget
	Filters       []PermanentFilter
	opponentOnly  bool
	opponentOwned bool
}

// TargetPermanent creates a target that selects any permanent on the battlefield,
// optionally narrowed by PermanentFilter predicates.
func TargetPermanent(filters ...PermanentFilter) Target {
	return &PermanentTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
		Filters:    filters,
	}
}

// TargetXPermanents requires exactly the announced value of X in distinct
// target permanents matching all filters.
func TargetXPermanents(filters ...PermanentFilter) Target {
	return &countBoundTarget{inner: TargetPermanent(filters...), usesX: true}
}

// TargetPermanentOpponentControls creates a target that selects a permanent an
// opponent controls, optionally narrowed by PermanentFilter predicates.
func TargetPermanentOpponentControls(filters ...PermanentFilter) Target {
	return &PermanentTarget{
		BaseTarget:   BaseTarget{min: 1, max: 1},
		Filters:      filters,
		opponentOnly: true,
	}
}

// TargetPermanentOpponentOwns creates a target that selects a permanent an
// opponent currently owns, regardless of who controls it.
func TargetPermanentOpponentOwns(filters ...PermanentFilter) Target {
	return &PermanentTarget{
		BaseTarget:    BaseTarget{min: 1, max: 1},
		Filters:       filters,
		opponentOwned: true,
	}
}

// TargetUpToNPermanents creates a target that selects from 0 up to n
// permanents on the battlefield, optionally narrowed by PermanentFilter
// predicates. Mirrors TargetUpToNCreatures for spells like Proctor's Gaze
// that say "up to one target nonland permanent".
func TargetUpToNPermanents(n int, filters ...PermanentFilter) Target {
	return &PermanentTarget{
		BaseTarget: BaseTarget{min: 0, max: n},
		Filters:    filters,
	}
}

// TargetUpToOnePermanent is shorthand for TargetUpToNPermanents(1, filters...).
func TargetUpToOnePermanent(filters ...PermanentFilter) Target {
	return TargetUpToNPermanents(1, filters...)
}

func (t *PermanentTarget) Possible(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID {
	var result []uuid.UUID
	for _, p := range g.battlefield {
		if t.opponentOnly && p.ControllerID() == controller {
			continue
		}
		if t.opponentOwned && p.Card.Owner() == controller {
			continue
		}
		if !p.CanBeTargetedBy(sourceCard, controller, g) {
			continue
		}
		match := true
		for _, f := range t.Filters {
			if !f.Match(p, g) {
				match = false
				break
			}
		}
		if match {
			result = append(result, p.ID())
		}
	}
	return result
}

func (t *PermanentTarget) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}

// Filter returns a combined PermanentFilter that checks all filters.
func (t *PermanentTarget) Filter() PermanentFilter {
	return And(t.Filters...)
}

// TargetLand creates a target that selects a land on the battlefield.
// Convenience wrapper for TargetPermanent(IsLand).
func TargetLand() Target {
	return TargetPermanent(IsLand)
}

// TargetArtifact creates a target that selects an artifact on the battlefield.
// Convenience wrapper for TargetPermanent(IsArtifact).
func TargetArtifact() Target {
	return TargetPermanent(IsArtifact)
}

// TargetAuraAttachedToCreatureYouControl creates a target that selects an Aura
// attached to a creature the controller controls.
func TargetAuraAttachedToCreatureYouControl() Target {
	return &auraAttachedToCreatureYouControlTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
	}
}

type auraAttachedToCreatureYouControlTarget struct {
	BaseTarget
}

func (t *auraAttachedToCreatureYouControlTarget) Possible(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID {
	var result []uuid.UUID
	for _, p := range g.battlefield {
		if !p.IsAttached() || (!p.HasType(TypeEnchantment) && !p.HasSubType("Aura")) {
			continue
		}
		host := g.FindPermanent(p.AttachedTo)
		if host == nil || !host.HasType(TypeCreature) || host.ControllerID() != controller {
			continue
		}
		if !p.CanBeTargetedBy(sourceCard, controller, g) {
			continue
		}
		result = append(result, p.ID())
	}
	return result
}

func (t *auraAttachedToCreatureYouControlTarget) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}

// TargetArtifactWithManaValueX creates a target that selects an artifact on the
// battlefield whose mana value equals the current X value (g.currentX). Used by
// spells like Detonate where the targeting restriction depends on X.
func TargetArtifactWithManaValueX() Target {
	return &artifactWithManaValueXTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
	}
}

type artifactWithManaValueXTarget struct {
	BaseTarget
}

func (t *artifactWithManaValueXTarget) Possible(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID {
	x := g.resolution.X()
	var result []uuid.UUID
	for _, p := range g.battlefield {
		if !p.HasType(TypeArtifact) {
			continue
		}
		if !p.CanBeTargetedBy(sourceCard, controller, g) {
			continue
		}
		if p.Card.ManaCost().CMC() != x {
			continue
		}
		result = append(result, p.ID())
	}
	return result
}

func (t *artifactWithManaValueXTarget) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}

// TargetArtifactOrEnchantment creates a target that selects an artifact or enchantment on the battlefield.
// Convenience wrapper for TargetPermanent(Or(IsArtifact, IsEnchantment)).
func TargetArtifactOrEnchantment() Target {
	return TargetPermanent(Or(IsArtifact, IsEnchantment))
}

// SpellOrPermanentTarget targets either a spell on the stack or a permanent
// on the battlefield.
type SpellOrPermanentTarget struct {
	BaseTarget
}

type sourceChosenSubtypeCreatureTarget struct {
	BaseTarget
}

// TargetCreatureOfSourceChosenSubtype creates a target matching creatures
// whose subtype equals the source permanent's ChosenSubtype value.
func TargetCreatureOfSourceChosenSubtype() Target {
	return &sourceChosenSubtypeCreatureTarget{BaseTarget: BaseTarget{min: 1, max: 1}}
}

func (t *sourceChosenSubtypeCreatureTarget) Possible(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID {
	source := g.FindPermanent(sourceCard.ID())
	if source == nil || source.ChosenSubtype == "" {
		return nil
	}
	possible := make([]uuid.UUID, 0)
	for _, permanent := range g.AllBattlefield() {
		if permanent.HasType(TypeCreature) && permanent.HasSubType(source.ChosenSubtype) && permanent.CanBeTargetedBy(sourceCard, controller, g) {
			possible = append(possible, permanent.ID())
		}
	}
	return possible
}

func (t *sourceChosenSubtypeCreatureTarget) Choose(_ uuid.UUID, _ Card, _ *Game, chosen []uuid.UUID) error {
	t.chosen = append([]uuid.UUID(nil), chosen...)
	return nil
}

// TargetSpellOrPermanent creates a single target that can select across the
// stack and battlefield. Activated and triggered abilities are not spells and
// are not included.
func TargetSpellOrPermanent() Target {
	return &SpellOrPermanentTarget{BaseTarget: BaseTarget{min: 1, max: 1}}
}

func (t *SpellOrPermanentTarget) Possible(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID {
	permanents := TargetPermanent().Possible(controller, sourceCard, g)
	spells := TargetSpellOnStack().Possible(controller, sourceCard, g)
	return uniqueTargetCandidates(append(permanents, spells...))
}

func (t *SpellOrPermanentTarget) Choose(_ uuid.UUID, _ Card, _ *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}

// DamageSourceTarget targets any permanent or spell on the stack as a source of damage.
type DamageSourceTarget struct {
	BaseTarget
}

// TargetDamageSource creates a target for a damage source (a permanent or a spell on the stack).
func TargetDamageSource() Target {
	return &DamageSourceTarget{BaseTarget: BaseTarget{min: 1, max: 1}}
}

func (t *DamageSourceTarget) Possible(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID {
	permanents := TargetPermanent().Possible(controller, sourceCard, g)
	spells := TargetSpellOnStack().Possible(controller, sourceCard, g)
	return uniqueTargetCandidates(append(permanents, spells...))
}

func (t *DamageSourceTarget) Choose(_ uuid.UUID, _ Card, _ *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}

// SpellOnStackTarget targets a spell on the stack, optionally filtered by CardFilter predicates.
type SpellOnStackTarget struct {
	BaseTarget
	Filters         []CardFilter
	controlledByYou bool
}

// TargetSpellOnStack creates a target that selects a spell currently on the stack (for counterspells),
// optionally narrowed by CardFilter predicates.
func TargetSpellOnStack(filters ...CardFilter) Target {
	return &SpellOnStackTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
		Filters:    filters,
	}
}

// TargetOwnSpellOnStack creates a target that selects a spell you control on the stack,
// optionally narrowed by CardFilter predicates.
func TargetOwnSpellOnStack(filters ...CardFilter) Target {
	return &SpellOnStackTarget{
		BaseTarget:      BaseTarget{min: 1, max: 1},
		Filters:         filters,
		controlledByYou: true,
	}
}

func (t *SpellOnStackTarget) Possible(controller uuid.UUID, _ Card, g *Game) []uuid.UUID {
	var result []uuid.UUID
	for _, obj := range g.stack.Objects() {
		if !obj.IsAbility && obj.Card != nil {
			if t.controlledByYou && obj.Controller != controller {
				continue
			}
			match := true
			for _, f := range t.Filters {
				if !f.Match(obj.Card) {
					match = false
					break
				}
			}
			if match {
				result = append(result, obj.SourceID)
			}
		}
	}
	return result
}

func (t *SpellOnStackTarget) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}

// GraveyardCardTarget targets any card in your graveyard (not just creatures),
// optionally filtered by CardFilter predicates.
type GraveyardCardTarget struct {
	BaseTarget
	Filters       []CardFilter
	excludeSource bool
}

// TargetCardInYourGraveyard creates a target that selects any card in the controller's graveyard,
// optionally narrowed by CardFilter predicates.
func TargetCardInYourGraveyard(filters ...CardFilter) Target {
	return &GraveyardCardTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
		Filters:    filters,
	}
}

// TargetOtherCreatureInYourGraveyard targets a creature card in the controller's
// graveyard other than the source card. Used for "return another target creature
// card from your graveyard" triggers (CR 109.5 — "another" excludes the source).
func TargetOtherCreatureInYourGraveyard() Target {
	return &GraveyardCardTarget{
		BaseTarget:    BaseTarget{min: 1, max: 1},
		Filters:       []CardFilter{IsCreatureCard},
		excludeSource: true,
	}
}

func (t *GraveyardCardTarget) Possible(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID {
	p := g.GetPlayer(controller)
	if p == nil {
		return nil
	}
	var sourceID uuid.UUID
	if sourceCard != nil {
		sourceID = sourceCard.ID()
	}
	var result []uuid.UUID
	for _, c := range p.Graveyard() {
		if t.excludeSource && c.ID() == sourceID {
			continue
		}
		match := true
		for _, f := range t.Filters {
			if !f.Match(c) {
				match = false
				break
			}
		}
		if match {
			result = append(result, c.ID())
		}
	}
	return result
}

func (t *GraveyardCardTarget) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}

// TargetAnyNumberOfCardsInYourGraveyard creates a target that selects any number
// (0 or more) of cards from the controller's graveyard, optionally filtered by
// CardFilter predicates.
func TargetAnyNumberOfCardsInYourGraveyard(filters ...CardFilter) Target {
	return &GraveyardCardTarget{
		BaseTarget: BaseTarget{min: 0, max: 100},
		Filters:    filters,
	}
}

// AnyGraveyardCardTarget targets a card in any player's graveyard, optionally
// filtered by CardFilter predicates. Used by effects worded "from a graveyard"
// (e.g. Reanimate, "Put target creature card from a graveyard onto the
// battlefield under your control").
type AnyGraveyardCardTarget struct {
	BaseTarget
	Filters []CardFilter
}

// TargetCardInAnyGraveyard creates a target that selects any card in any
// player's graveyard, optionally narrowed by CardFilter predicates.
func TargetCardInAnyGraveyard(filters ...CardFilter) Target {
	return &AnyGraveyardCardTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
		Filters:    filters,
	}
}

// TargetCreatureCardInAnyGraveyard targets a creature card in any player's
// graveyard. Convenience wrapper for TargetCardInAnyGraveyard(IsCreatureCard).
func TargetCreatureCardInAnyGraveyard() Target {
	return TargetCardInAnyGraveyard(IsCreatureCard)
}

func (t *AnyGraveyardCardTarget) Possible(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID {
	var result []uuid.UUID
	for _, p := range g.players {
		for _, c := range p.Graveyard() {
			match := true
			for _, f := range t.Filters {
				if !f.Match(c) {
					match = false
					break
				}
			}
			if match {
				result = append(result, c.ID())
			}
		}
	}
	return result
}

func (t *AnyGraveyardCardTarget) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}

// HandCardTarget targets a card in the controller's hand, optionally filtered by CardFilter predicates.
type HandCardTarget struct {
	BaseTarget
	Filters []CardFilter
}

// TargetCardInHand creates a target that selects a card in the controller's hand,
// optionally narrowed by CardFilter predicates.
func TargetCardInHand(filters ...CardFilter) Target {
	return &HandCardTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
		Filters:    filters,
	}
}

// TargetCreatureInHand creates a target that selects a creature card in the controller's hand.
// Convenience wrapper for TargetCardInHand(IsCreatureCard).
func TargetCreatureInHand() Target {
	return TargetCardInHand(IsCreatureCard)
}

func (t *HandCardTarget) Possible(controller uuid.UUID, _ Card, g *Game) []uuid.UUID {
	p := g.GetPlayer(controller)
	if p == nil {
		return nil
	}
	var result []uuid.UUID
	for _, c := range p.Hand() {
		match := true
		for _, f := range t.Filters {
			if !f.Match(c) {
				match = false
				break
			}
		}
		if match {
			result = append(result, c.ID())
		}
	}
	return result
}

func (t *HandCardTarget) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}

// ControlledPermanentTarget targets a permanent you control.
type ControlledPermanentTarget struct {
	BaseTarget
}

// TargetControlledPermanent creates a target that selects a permanent the controller owns.
func TargetControlledPermanent() Target {
	return &ControlledPermanentTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
	}
}

func (t *ControlledPermanentTarget) Possible(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID {
	var result []uuid.UUID
	for _, p := range g.battlefield {
		if p.ControllerID() == controller && p.Card.Owner() == controller && p.CanBeTargetedBy(sourceCard, controller, g) {
			result = append(result, p.ID())
		}
	}
	return result
}

func (t *ControlledPermanentTarget) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}

// OpponentTarget targets an opponent.
type OpponentTarget struct {
	BaseTarget
}

// TargetOpponent creates a target that selects an opponent (any player other than the controller).
func TargetOpponent() Target {
	return &OpponentTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
	}
}

func (t *OpponentTarget) Possible(controller uuid.UUID, _ Card, g *Game) []uuid.UUID {
	var result []uuid.UUID
	for _, p := range g.players {
		if p.PlayerID() != controller {
			result = append(result, p.PlayerID())
		}
	}
	return result
}

func (t *OpponentTarget) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}

// PowerLESourceCreatureTarget targets a creature with power less than or equal to
// the source creature's power (e.g. Old Man of the Sea).
type PowerLESourceCreatureTarget struct {
	BaseTarget
}

// TargetCreatureWithPowerLESource creates a target that selects a creature whose power
// is less than or equal to the source permanent's power.
func TargetCreatureWithPowerLESource() Target {
	return &PowerLESourceCreatureTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
	}
}

func (t *PowerLESourceCreatureTarget) Possible(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID {
	src := g.FindPermanent(sourceCard.ID())
	if src == nil {
		return nil
	}
	srcPower := src.CurrentPower(g)
	var result []uuid.UUID
	for _, p := range g.battlefield {
		if !p.HasType(TypeCreature) {
			continue
		}
		if !p.CanBeTargetedBy(sourceCard, controller, g) {
			continue
		}
		if p.CurrentPower(g) <= srcPower {
			result = append(result, p.ID())
		}
	}
	return result
}

func (t *PowerLESourceCreatureTarget) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}

// BlockingOrBlockedBySourceTarget targets a creature that is blocking or
// blocked by the source creature (e.g. Lesser Werewolf, Sentinel).
type BlockingOrBlockedBySourceTarget struct {
	BaseTarget
}

// TargetCreatureBlockingOrBlockedBySource creates a target that selects a
// creature blocking or blocked by the source permanent.
func TargetCreatureBlockingOrBlockedBySource() Target {
	return &BlockingOrBlockedBySourceTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
	}
}

func (t *BlockingOrBlockedBySourceTarget) Possible(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID {
	sourceID := sourceCard.ID()
	var result []uuid.UUID
	for _, group := range g.CombatGroups() {
		if group.AttackerID == sourceID {
			// Source is attacking — blockers are valid targets
			for _, bid := range group.BlockerIDs {
				b := g.FindPermanent(bid)
				if b != nil && b.CanBeTargetedBy(sourceCard, controller, g) {
					result = append(result, bid)
				}
			}
		} else if slices.Contains(group.BlockerIDs, sourceID) {
			// Source is blocking — the attacker is a valid target
			a := g.FindPermanent(group.AttackerID)
			if a != nil && a.CanBeTargetedBy(sourceCard, controller, g) {
				result = append(result, group.AttackerID)
			}
		}
	}
	return result
}

func (t *BlockingOrBlockedBySourceTarget) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}
