package mage

import "github.com/google/uuid"

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

// CreatureTarget targets a creature on the battlefield.
type CreatureTarget struct {
	BaseTarget
	Filters []PermanentFilter
}

// TargetCreature creates a target that selects a creature on the battlefield,
// optionally narrowed by PermanentFilter predicates.
func TargetCreature(filters ...PermanentFilter) Target {
	return &CreatureTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
		Filters:    filters,
	}
}

func (t *CreatureTarget) Possible(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID {
	var result []uuid.UUID
	for _, p := range g.Battlefield {
		if !p.HasType(TypeCreature) {
			continue
		}
		if !p.CanBeTargetedBy(sourceCard, controller, g) {
			continue
		}
		match := true
		for _, f := range t.Filters {
			if !f(p, g) {
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
	return func(p *Permanent, g *Game) bool {
		if !p.HasType(TypeCreature) {
			return false
		}
		for _, f := range t.Filters {
			if !f(p, g) {
				return false
			}
		}
		return true
	}
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
	for _, p := range g.Players {
		result = append(result, p.PlayerID())
	}
	return result
}

func (t *PlayerTarget) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
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
	for _, p := range g.Battlefield {
		if p.HasType(TypeCreature) && p.CanBeTargetedBy(sourceCard, controller, g) {
			result = append(result, p.ID())
		}
	}
	for _, p := range g.Players {
		result = append(result, p.PlayerID())
	}
	return result
}

func (t *AnyTarget) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}

// GraveyardCreatureTarget targets a creature card in your graveyard.
type GraveyardCreatureTarget struct {
	BaseTarget
}

// TargetCreatureInYourGraveyard creates a target that selects a creature card in the controller's graveyard.
func TargetCreatureInYourGraveyard() Target {
	return &GraveyardCreatureTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
	}
}

func (t *GraveyardCreatureTarget) Possible(controller uuid.UUID, _ Card, g *Game) []uuid.UUID {
	p := g.GetPlayer(controller)
	if p == nil {
		return nil
	}
	var result []uuid.UUID
	for _, c := range p.Graveyard() {
		if IsCreatureCard(c) {
			result = append(result, c.ID())
		}
	}
	return result
}

func (t *GraveyardCreatureTarget) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
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
	for _, p := range g.Battlefield {
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
	Filters []PermanentFilter
}

// TargetPermanent creates a target that selects any permanent on the battlefield,
// optionally narrowed by PermanentFilter predicates.
func TargetPermanent(filters ...PermanentFilter) Target {
	return &PermanentTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
		Filters:    filters,
	}
}

func (t *PermanentTarget) Possible(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID {
	var result []uuid.UUID
	for _, p := range g.Battlefield {
		if !p.CanBeTargetedBy(sourceCard, controller, g) {
			continue
		}
		match := true
		for _, f := range t.Filters {
			if !f(p, g) {
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
	return func(p *Permanent, g *Game) bool {
		for _, f := range t.Filters {
			if !f(p, g) {
				return false
			}
		}
		return true
	}
}

// LandTarget targets a land on the battlefield.
type LandTarget struct {
	BaseTarget
}

// TargetLand creates a target that selects a land on the battlefield.
func TargetLand() Target {
	return &LandTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
	}
}

func (t *LandTarget) Possible(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID {
	var result []uuid.UUID
	for _, p := range g.Battlefield {
		if p.HasType(TypeLand) {
			result = append(result, p.ID())
		}
	}
	return result
}

func (t *LandTarget) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}

// ArtifactTarget targets an artifact on the battlefield.
type ArtifactTarget struct {
	BaseTarget
}

// TargetArtifact creates a target that selects an artifact on the battlefield.
func TargetArtifact() Target {
	return &ArtifactTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
	}
}

func (t *ArtifactTarget) Possible(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID {
	var result []uuid.UUID
	for _, p := range g.Battlefield {
		if p.HasType(TypeArtifact) && p.CanBeTargetedBy(sourceCard, controller, g) {
			result = append(result, p.ID())
		}
	}
	return result
}

func (t *ArtifactTarget) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}

// ArtifactOrEnchantmentTarget targets an artifact or enchantment.
type ArtifactOrEnchantmentTarget struct {
	BaseTarget
}

// TargetArtifactOrEnchantment creates a target that selects an artifact or enchantment on the battlefield.
func TargetArtifactOrEnchantment() Target {
	return &ArtifactOrEnchantmentTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
	}
}

func (t *ArtifactOrEnchantmentTarget) Possible(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID {
	var result []uuid.UUID
	for _, p := range g.Battlefield {
		if (p.HasType(TypeArtifact) || p.HasType(TypeEnchantment)) && p.CanBeTargetedBy(sourceCard, controller, g) {
			result = append(result, p.ID())
		}
	}
	return result
}

func (t *ArtifactOrEnchantmentTarget) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}

// SpellOnStackTarget targets a spell on the stack.
type SpellOnStackTarget struct {
	BaseTarget
}

// TargetSpellOnStack creates a target that selects a spell currently on the stack (for counterspells).
func TargetSpellOnStack() Target {
	return &SpellOnStackTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
	}
}

func (t *SpellOnStackTarget) Possible(controller uuid.UUID, _ Card, g *Game) []uuid.UUID {
	var result []uuid.UUID
	for _, obj := range g.Stack.Objects() {
		if !obj.IsAbility && obj.Card != nil {
			result = append(result, obj.SourceID)
		}
	}
	return result
}

func (t *SpellOnStackTarget) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}

// GraveyardCardTarget targets any card in your graveyard (not just creatures).
type GraveyardCardTarget struct {
	BaseTarget
}

// TargetCardInYourGraveyard creates a target that selects any card in the controller's graveyard.
func TargetCardInYourGraveyard() Target {
	return &GraveyardCardTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
	}
}

func (t *GraveyardCardTarget) Possible(controller uuid.UUID, _ Card, g *Game) []uuid.UUID {
	p := g.GetPlayer(controller)
	if p == nil {
		return nil
	}
	var result []uuid.UUID
	for _, c := range p.Graveyard() {
		result = append(result, c.ID())
	}
	return result
}

func (t *GraveyardCardTarget) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}

// HandCreatureTarget targets a creature card in the controller's hand.
type HandCreatureTarget struct {
	BaseTarget
}

// TargetCreatureInHand creates a target that selects a creature card in the controller's hand.
func TargetCreatureInHand() Target {
	return &HandCreatureTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
	}
}

func (t *HandCreatureTarget) Possible(controller uuid.UUID, _ Card, g *Game) []uuid.UUID {
	p := g.GetPlayer(controller)
	if p == nil {
		return nil
	}
	var result []uuid.UUID
	for _, c := range p.Hand() {
		if c.HasType(TypeCreature) {
			result = append(result, c.ID())
		}
	}
	return result
}

func (t *HandCreatureTarget) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
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
	for _, p := range g.Players {
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
