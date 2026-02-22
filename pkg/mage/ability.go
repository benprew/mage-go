package mage

import (
	"strings"

	. "github.com/mage/mage/pkg/mage/core"
	"github.com/google/uuid"
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
			for _, col := range card.ManaCost().Colors() {
				if col == c {
					return true
				}
			}
			return false
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

// Blocks returns true if this protection prevents interaction with the given card.
func (pa *ProtectionAbility) Blocks(card Card) bool {
	return pa.Filter.Match(card)
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

// ManaAbility is a mana ability that taps to add mana.
type ManaAbility struct {
	BaseAbility
	Color    Color
	AnyColor bool // player chooses color when activated
}

// NewManaAbility creates a tap-for-mana ability that produces one mana of the given color.
func NewManaAbility(c Color) *ManaAbility {
	return &ManaAbility{
		BaseAbility: BaseAbility{
			id:          uuid.New(),
			abilityType: AbilityMana,
		},
		Color: c,
	}
}

// NewAnyColorManaAbility creates a mana ability that lets the player choose
// which color of mana to add (e.g. Birds of Paradise).
func NewAnyColorManaAbility() *ManaAbility {
	return &ManaAbility{
		BaseAbility: BaseAbility{
			id:          uuid.New(),
			abilityType: AbilityMana,
		},
		AnyColor: true,
	}
}

// ManaBonusAbility grants bonus mana when matching permanents are tapped for mana.
// E.g. Gauntlet of Might: "Whenever a Mountain is tapped for mana, add {R}."
// If AttachedOnly is true, the filter is ignored and only the attached permanent
// triggers the bonus (for auras like Wild Growth).
type ManaBonusAbility struct {
	BaseAbility
	Filter       PermanentFilter
	BonusMana    Color
	AttachedOnly bool
}

// UnwrapAbility returns the inner ability if wrapped by a grantedByEffect, otherwise returns a itself.
func UnwrapAbility(a Ability) Ability {
	if ge, ok := a.(*grantedByEffect); ok {
		return ge.Ability
	}
	return a
}

// NewManaBonusAbility creates a static ability that grants bonus mana when a permanent
// matching the filter is tapped for mana (e.g. Gauntlet of Might for Mountains).
func NewManaBonusAbility(filter PermanentFilter, bonusMana Color) *ManaBonusAbility {
	return &ManaBonusAbility{
		BaseAbility: BaseAbility{
			id:          uuid.New(),
			abilityType: AbilityStatic,
		},
		Filter:    filter,
		BonusMana: bonusMana,
	}
}

// NewAttachedManaBonusAbility creates a mana bonus that triggers only when the
// permanent this aura is attached to is tapped for mana (e.g. Wild Growth).
func NewAttachedManaBonusAbility(bonusMana Color) *ManaBonusAbility {
	return &ManaBonusAbility{
		BaseAbility: BaseAbility{
			id:          uuid.New(),
			abilityType: AbilityStatic,
		},
		BonusMana:    bonusMana,
		AttachedOnly: true,
	}
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

// GraveyardReturnAbility allows a creature card in the graveyard to return to
// the battlefield if enough creature cards are above it in the graveyard.
type GraveyardReturnAbility struct {
	BaseAbility
	MinCreaturesAbove int
}

// GraveyardReturnIfCreaturesAbove creates a static ability that returns this creature card
// from the graveyard to the battlefield if at least n creature cards are above it (e.g. Nether Shadow).
func GraveyardReturnIfCreaturesAbove(n int) *GraveyardReturnAbility {
	return &GraveyardReturnAbility{
		BaseAbility: BaseAbility{
			id:          uuid.New(),
			abilityType: AbilityStatic,
		},
		MinCreaturesAbove: n,
	}
}
