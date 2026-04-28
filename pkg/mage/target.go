package mage

import (
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"github.com/google/uuid"
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

// TargetUpToNCreaturesOrPlayers creates a target that selects from 0 up to n
// creatures or players. Used for divided-damage spells like Flames of the
// Firebrand ("3 damage divided as you choose among any number of targets").
func TargetUpToNCreaturesOrPlayers(n int) Target {
	return &AnyTarget{
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
		if t.controllerOnly && p.Controller != controller {
			continue
		}
		if t.opponentOnly && p.Controller == controller {
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
	var result []uuid.UUID
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

// AnyTarget targets a creature or player.
type AnyTarget struct {
	BaseTarget
}

// TargetAnyTarget creates a target that selects any creature or player ("any target").
func TargetAnyTarget() Target {
	return &AnyTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
	}
}

func (t *AnyTarget) Possible(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID {
	var result []uuid.UUID
	for _, p := range g.battlefield {
		if p.HasType(TypeCreature) && p.CanBeTargetedBy(sourceCard, controller, g) {
			result = append(result, p.ID())
		}
	}
	for _, p := range g.players {
		result = append(result, p.PlayerID())
	}
	return result
}

func (t *AnyTarget) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
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
		if p.HasType(TypeCreature) && p.Controller == controller && p.CanBeTargetedBy(sourceCard, controller, g) {
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
	Filters      []PermanentFilter
	opponentOnly bool
}

// TargetPermanent creates a target that selects any permanent on the battlefield,
// optionally narrowed by PermanentFilter predicates.
func TargetPermanent(filters ...PermanentFilter) Target {
	return &PermanentTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
		Filters:    filters,
	}
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

func (t *PermanentTarget) Possible(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID {
	var result []uuid.UUID
	for _, p := range g.battlefield {
		if t.opponentOnly && p.Controller == controller {
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
	x := g.currentX
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
		if p.Controller == controller && p.Card.Owner() == controller && p.CanBeTargetedBy(sourceCard, controller, g) {
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
		} else {
			// Check if source is a blocker in this group
			for _, bid := range group.BlockerIDs {
				if bid == sourceID {
					// Source is blocking — the attacker is a valid target
					a := g.FindPermanent(group.AttackerID)
					if a != nil && a.CanBeTargetedBy(sourceCard, controller, g) {
						result = append(result, group.AttackerID)
					}
					break
				}
			}
		}
	}
	return result
}

func (t *BlockingOrBlockedBySourceTarget) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}
