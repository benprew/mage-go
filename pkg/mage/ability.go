package mage

import (
	"slices"
	"strings"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// Ability is the base interface for all abilities.
type Ability interface {
	AbilityID() uuid.UUID
	Source() uuid.UUID
	SetSource(uuid.UUID)
	Controller() uuid.UUID
	SetController(uuid.UUID)
	Type() AbilityType
}

// BaseAbility provides common ability fields.
type BaseAbility struct {
	id          uuid.UUID
	source      uuid.UUID
	controller  uuid.UUID
	abilityType AbilityType
}

func (a *BaseAbility) AbilityID() uuid.UUID       { return a.id }
func (a *BaseAbility) Source() uuid.UUID          { return a.source }
func (a *BaseAbility) SetSource(id uuid.UUID)     { a.source = id }
func (a *BaseAbility) Controller() uuid.UUID      { return a.controller }
func (a *BaseAbility) SetController(id uuid.UUID) { a.controller = id }
func (a *BaseAbility) Type() AbilityType          { return a.abilityType }

// ProtectionAbility grants protection from specific colors.
type ProtectionAbility struct {
	BaseAbility
	FromColors []Color
	Filter     CardFilter
}

// ProtectionFromColor creates a static ability granting protection from a single color.
func ProtectionFromColor(c Color) *ProtectionAbility {
	return &ProtectionAbility{
		BaseAbility: BaseAbility{
			id:          uuid.New(),
			abilityType: AbilityStatic,
		},
		FromColors: []Color{c},
		Filter: NewCardFilter(c.String(), func(card Card) bool {
			return slices.Contains(card.ManaCost().Colors(), c)
		}),
	}
}

// ProtectionFromColors creates a static ability granting protection from multiple colors.
func ProtectionFromColors(cs ...Color) *ProtectionAbility {
	colorSet := make(map[Color]bool)
	for _, c := range cs {
		colorSet[c] = true
	}
	labels := make([]string, len(cs))
	for i, c := range cs {
		labels[i] = c.String()
	}
	return &ProtectionAbility{
		BaseAbility: BaseAbility{
			id:          uuid.New(),
			abilityType: AbilityStatic,
		},
		FromColors: cs,
		Filter: NewCardFilter(strings.Join(labels, " and "), func(card Card) bool {
			for _, col := range card.ManaCost().Colors() {
				if colorSet[col] {
					return true
				}
			}
			return false
		}),
	}
}

// ProtectionFromCardType creates a static ability granting protection from a card type
// (e.g. "protection from creatures", "protection from artifacts").
func ProtectionFromCardType(ct CardType) *ProtectionAbility {
	return &ProtectionAbility{
		BaseAbility: BaseAbility{
			id:          uuid.New(),
			abilityType: AbilityStatic,
		},
		Filter: NewCardFilter(ct.String(), func(card Card) bool {
			return card.HasType(ct)
		}),
	}
}

// ProtectionFromSubType creates a static ability granting protection from a subtype
// (e.g. "protection from Goblins", "protection from Zombies").
func ProtectionFromSubType(subtype string) *ProtectionAbility {
	return &ProtectionAbility{
		BaseAbility: BaseAbility{
			id:          uuid.New(),
			abilityType: AbilityStatic,
		},
		Filter: NewCardFilter(subtype, func(card Card) bool {
			return slices.Contains(card.SubTypes(), subtype)
		}),
	}
}

// ProtectionFromAll creates a static ability granting protection from everything
// (e.g. Progenitus). Matches all cards.
func ProtectionFromAll() *ProtectionAbility {
	return &ProtectionAbility{
		BaseAbility: BaseAbility{
			id:          uuid.New(),
			abilityType: AbilityStatic,
		},
		Filter: NewCardFilter("everything", func(card Card) bool {
			return true
		}),
	}
}

// Blocks returns true if this protection prevents interaction with the given card.
func (pa *ProtectionAbility) Blocks(card Card) bool {
	return pa.Filter.Match(card)
}

// BlocksInGame evaluates color protection against the source object's current
// colors. Other protection qualities continue to use their CardFilter.
func (pa *ProtectionAbility) BlocksInGame(card Card, g *Game) bool {
	if card == nil {
		return false
	}
	if len(pa.FromColors) == 0 || g == nil {
		return pa.Blocks(card)
	}
	for _, sourceColor := range g.EffectiveColors(card.ID()) {
		if slices.Contains(pa.FromColors, sourceColor) {
			return true
		}
	}
	return false
}

// StaticAbilityHolder holds continuous effects as a static ability.
type StaticAbilityHolder struct {
	BaseAbility
	Effects []ContinuousEffect
}

// StaticAbility creates a static ability that applies continuous effects while the source is on the battlefield.
func StaticAbility(effects ...ContinuousEffect) *StaticAbilityHolder {
	return &StaticAbilityHolder{
		BaseAbility: BaseAbility{
			id:          uuid.New(),
			abilityType: AbilityStatic,
		},
		Effects: effects,
	}
}

// UnwrapAbility returns the inner ability if wrapped by a grantedByEffect, otherwise returns a itself.
func UnwrapAbility(a Ability) Ability {
	if ge, ok := a.(*grantedByEffect); ok {
		return ge.Ability
	}
	return a
}

// SacrificeUnlessLandAbility requires the controller to control a land of a
// specific subtype or the permanent is sacrificed as a state-based action.
type SacrificeUnlessLandAbility struct {
	BaseAbility
	LandSubtype string
}

// SacrificeUnlessLand creates a static ability that sacrifices the source as a state-based action
// if the controller doesn't control a land of the given subtype (e.g. "Island" for Dandân).
func SacrificeUnlessLand(subtype string) *SacrificeUnlessLandAbility {
	return &SacrificeUnlessLandAbility{
		BaseAbility: BaseAbility{
			id:          uuid.New(),
			abilityType: AbilityStatic,
		},
		LandSubtype: subtype,
	}
}

// SacrificeIfControlsAbility requires the controller to sacrifice the permanent
// as a state-based action if they control another permanent of a specific subtype (e.g. Dwarf).
type SacrificeIfControlsAbility struct {
	BaseAbility
	Subtype string
}

// SacrificeIfControls creates a static ability that sacrifices the source as a state-based action
// if the controller controls a permanent of the given subtype (e.g. "Dwarf" for Goblins of the Flarg).
func SacrificeIfControls(subtype string) *SacrificeIfControlsAbility {
	return &SacrificeIfControlsAbility{
		BaseAbility: BaseAbility{
			id:          uuid.New(),
			abilityType: AbilityStatic,
		},
		Subtype: subtype,
	}
}

// EntersWithXCountersAbility is a replacement effect that adds X counters when
// the permanent enters the battlefield.
type EntersWithXCountersAbility struct {
	BaseAbility
	CounterType CounterType
}

// EntersWithXCounters creates a replacement effect that puts X counters of the given
// type on the permanent as it enters the battlefield (e.g. Rock Hydra).
func EntersWithXCounters(ct CounterType) *EntersWithXCountersAbility {
	return &EntersWithXCountersAbility{
		BaseAbility: BaseAbility{
			id:          uuid.New(),
			abilityType: AbilityStatic,
		},
		CounterType: ct,
	}
}

// EntersWithNCountersAbility is a replacement effect that adds a fixed number
// of counters when the permanent enters the battlefield.
type EntersWithNCountersAbility struct {
	BaseAbility
	CounterType CounterType
	Count       int
}

// EntersWithNCounters creates a replacement effect that puts N counters of the given
// type on the permanent as it enters the battlefield (e.g. Clockwork Avian).
func EntersWithNCounters(ct CounterType, n int) *EntersWithNCountersAbility {
	return &EntersWithNCountersAbility{
		BaseAbility: BaseAbility{
			id:          uuid.New(),
			abilityType: AbilityStatic,
		},
		CounterType: ct,
		Count:       n,
	}
}

// EntersWithComputedCountersAbility is a replacement effect that adds a
// dynamically-computed number of counters when the permanent enters the
// battlefield. The Compute closure is evaluated at ETB resolution time, after
// the entering permanent has been transiently exposed to FindPermanent so the
// closure can inspect the entering permanent itself (its ID and controller)
// and the rest of the battlefield. Routed through AddCountersWithReplacement
// so doublers (Branching Evolution) and ETB-additional counter effects
// (Oona's Blackguard) compose correctly. Used for cards like Towering Titan
// whose ETB count depends on board state.
type EntersWithComputedCountersAbility struct {
	BaseAbility
	CounterType CounterType
	Compute     func(g *Game, perm *Permanent) int
}

// EntersWithComputedCounters creates a replacement effect that puts a
// computed number of counters of the given type on the permanent as it
// enters the battlefield. The compute closure receives the game and the
// entering permanent and returns the count.
func EntersWithComputedCounters(ct CounterType, compute func(g *Game, perm *Permanent) int) *EntersWithComputedCountersAbility {
	return &EntersWithComputedCountersAbility{
		BaseAbility: BaseAbility{
			id:          uuid.New(),
			abilityType: AbilityStatic,
		},
		CounterType: ct,
		Compute:     compute,
	}
}

// CopyCreatureOnETBAbility is a replacement effect that copies a target creature
// when this permanent enters the battlefield (e.g., Vesuvan Doppelganger).
type CopyCreatureOnETBAbility struct {
	BaseAbility
}

// CopyCreatureOnETB creates a replacement effect that copies a target creature
// when this permanent enters the battlefield (e.g. Vesuvan Doppelganger, Clone).
func CopyCreatureOnETB() *CopyCreatureOnETBAbility {
	return &CopyCreatureOnETBAbility{
		BaseAbility: BaseAbility{
			id:          uuid.New(),
			abilityType: AbilityStatic,
		},
	}
}

// ETBWithTargetsAbility runs an effect inline during PutOnBattlefield,
// receiving the spell's targets (g.ResolvingTargets). Used for permanents
// that need to act on their spell targets when they enter (e.g. Oubliette).
type ETBWithTargetsAbility struct {
	BaseAbility
	Effect Effect
}

// ETBWithTargets creates an ability that runs the given effect when this
// permanent enters the battlefield, passing the spell's resolving targets.
func ETBWithTargets(effect Effect) *ETBWithTargetsAbility {
	return &ETBWithTargetsAbility{
		BaseAbility: BaseAbility{
			id:          uuid.New(),
			abilityType: AbilityStatic,
		},
		Effect: effect,
	}
}

// ETBEffectAbility runs an effect inline during PutOnBattlefield without
// requiring targets. Used for replacement effects like Primal Clay's form choice.
type ETBEffectAbility struct {
	BaseAbility
	Effect Effect
}

// ETBEffect creates an ability that runs the given effect unconditionally when
// this permanent enters the battlefield.
func ETBEffect(effect Effect) *ETBEffectAbility {
	return &ETBEffectAbility{
		BaseAbility: BaseAbility{
			id:          uuid.New(),
			abilityType: AbilityStatic,
		},
		Effect: effect,
	}
}

// AsEntersBattlefieldAbility is an interface for abilities that modify how a permanent enters the battlefield
// or replace its entry entirely (e.g. Frankenstein's Monster, Nameless Race).
// If OnEnter returns false, the permanent is put into its owner's graveyard instead of the battlefield.
type AsEntersBattlefieldAbility interface {
	Ability
	OnEnter(g *Game, perm *Permanent) bool
}

type frankensteinsMonsterAbility struct {
	BaseAbility
}

// FrankensteinsMonsterAbility creates the ETB replacement ability for Frankenstein's Monster.
func FrankensteinsMonsterAbility() *frankensteinsMonsterAbility {
	return &frankensteinsMonsterAbility{
		BaseAbility: BaseAbility{
			id:          uuid.New(),
			abilityType: AbilityStatic,
		},
	}
}

func (a *frankensteinsMonsterAbility) OnEnter(g *Game, perm *Permanent) bool {
	x := g.resolution.X()
	p := g.GetPlayer(perm.ControllerID())
	if p == nil {
		return false
	}
	var creatureCards []Card
	for _, c := range p.Graveyard() {
		if c.HasType(TypeCreature) {
			creatureCards = append(creatureCards, c)
		}
	}
	if len(creatureCards) < x {
		return false
	}
	toExile := creatureCards[:x]
	for _, c := range toExile {
		removed, ok := g.MoveFromGraveyard(perm.ControllerID(), c.ID(), ZoneExile)
		if ok && removed != nil {
			g.ExileCard(removed, perm.ID())
		}
		choice := p.ChooseString([]string{"+1/+1", "+2/+0", "+0/+2"}, "Choose counter for Frankenstein's Monster")
		ct := P1P1
		switch choice {
		case "+2/+0":
			ct = P2P0
		case "+0/+2":
			ct = P0P2
		}
		g.AddCountersWithReplacement(perm, ct, 1, perm.ID(), true)
	}
	return true
}

type namelessRaceAbility struct {
	BaseAbility
}

// NamelessRaceAbility creates the ETB life-payment ability for Nameless Race.
func NamelessRaceAbility() *namelessRaceAbility {
	return &namelessRaceAbility{
		BaseAbility: BaseAbility{
			id:          uuid.New(),
			abilityType: AbilityStatic,
		},
	}
}

func (a *namelessRaceAbility) OnEnter(g *Game, perm *Permanent) bool {
	p := g.GetPlayer(perm.ControllerID())
	if p == nil {
		return false
	}
	maxPay := 0
	for _, opp := range g.AllPlayers() {
		if opp.PlayerID() == perm.ControllerID() {
			continue
		}
		for _, battlefieldPerm := range g.battlefield {
			if battlefieldPerm.ControllerID() == opp.PlayerID() && !battlefieldPerm.IsToken && slices.Contains(battlefieldPerm.Colors(), White) {
				maxPay++
			}
		}
		for _, c := range opp.Graveyard() {
			if slices.Contains(c.ManaCost().Colors(), White) {
				maxPay++
			}
		}
	}

	amount := max(min(g.resolution.X(), maxPay), 0)
	p.LoseLife(amount)
	g.FireEvent(GameEvent{
		Type:     EvtLifeLost,
		PlayerID: p.PlayerID(),
		Amount:   amount,
	})
	perm.StoredValue = amount
	return true
}
