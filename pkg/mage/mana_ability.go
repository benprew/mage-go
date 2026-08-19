package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// ManaProduction represents a single color+amount pair produced by a mana ability.
// If AnyCombination is true and Color is AnyColor, the controller chooses a
// color independently for each of the Amount mana produced (e.g. "Add X mana
// in any combination of colors"). When false, AnyColor with Amount > 1 picks a
// single color for all of them ("X mana of any one color").
type ManaProduction struct {
	Color          Color
	Amount         int
	AnyCombination bool
}

// ManaProductionFunc derives a mana ability's current production from its
// source permanent. Implementations must be pure queries: solvers may call the
// function more than once without activating the ability.
type ManaProductionFunc func(g GameReader, sourceID uuid.UUID) []ManaProduction

// ManaAbility is a mana ability that taps to add mana.
// Productions defines what mana is produced. Each entry is a color+amount pair.
// Use AnyColor as the color to let the player choose when activated.
type ManaAbility struct {
	BaseAbility
	Productions        []ManaProduction
	dynamicProductions ManaProductionFunc
	postProduction     []Effect
}

// ProducedAmount returns the total mana this ability produces when activated.
func (ma *ManaAbility) ProducedAmount() int {
	total := 0
	for _, p := range ma.Productions {
		if p.Amount <= 0 {
			total++
		} else {
			total += p.Amount
		}
	}
	return total
}

// HasAnyColor returns true if any production uses AnyColor.
func (ma *ManaAbility) HasAnyColor() bool {
	for _, p := range ma.Productions {
		if p.Color == AnyColor {
			return true
		}
	}
	return false
}

// PrimaryColor returns the color of the first production, or Colorless if empty.
func (ma *ManaAbility) PrimaryColor() Color {
	if len(ma.Productions) > 0 {
		return ma.Productions[0].Color
	}
	return Colorless
}

// ManaProductionsForAbility returns the mana produced by an ability that acts
// as a tap-for-mana source. It recognizes both ManaAbility and activated
// abilities whose only cost is tapping and whose effects only add mana.
func ManaProductionsForAbility(a Ability) []ManaProduction {
	productions := abilityManaProductions(UnwrapAbility(a), nil, uuid.Nil)
	if productions == nil {
		return nil
	}
	return append([]ManaProduction(nil), productions...)
}

// ManaProductionsForAbilityInGame returns the mana an ability currently
// produces for sourceID. Unlike ManaProductionsForAbility, it resolves dynamic
// production callbacks against live game state.
func ManaProductionsForAbilityInGame(a Ability, g GameReader, sourceID uuid.UUID) []ManaProduction {
	productions := abilityManaProductions(UnwrapAbility(a), g, sourceID)
	if productions == nil {
		return nil
	}
	return append([]ManaProduction(nil), productions...)
}

// ChosenColorManaProductions produces one mana of the source permanent's
// current ChosenColor. An unset or nonconcrete choice produces nothing.
func ChosenColorManaProductions(g GameReader, sourceID uuid.UUID) []ManaProduction {
	if g == nil {
		return nil
	}
	perm := g.FindPermanent(sourceID)
	if perm == nil || perm.ChosenColor == Colorless || perm.ChosenColor == AnyColor {
		return nil
	}
	return []ManaProduction{{Color: perm.ChosenColor, Amount: 1}}
}

// NewManaAbility creates a tap-for-mana ability that produces one mana of the given color.
func NewManaAbility(c Color) *ManaAbility {
	return &ManaAbility{
		BaseAbility: BaseAbility{
			id:          uuid.New(),
			abilityType: AbilityMana,
		},
		Productions: []ManaProduction{{Color: c, Amount: 1}},
	}
}

// NewMultiManaAbility creates a mana ability from one or more productions.
// Use for cards that produce multiple mana or multiple colors at once.
//
//	NewMultiManaAbility(ManaProduction{Colorless, 2})           // Sol Ring
//	NewMultiManaAbility(ManaProduction{Green, 1}, ManaProduction{White, 1}) // {G}{W}
func NewMultiManaAbility(productions ...ManaProduction) *ManaAbility {
	return &ManaAbility{
		BaseAbility: BaseAbility{
			id:          uuid.New(),
			abilityType: AbilityMana,
		},
		Productions: productions,
	}
}

// NewDynamicManaAbility creates a tap mana ability whose production is derived
// from its source permanent at query and activation time. afterProduction
// effects run immediately, in order, after mana is added and without using the
// stack.
func NewDynamicManaAbility(productions ManaProductionFunc, afterProduction ...Effect) *ManaAbility {
	return &ManaAbility{
		BaseAbility: BaseAbility{
			id:          uuid.New(),
			abilityType: AbilityMana,
		},
		dynamicProductions: productions,
		postProduction:     append([]Effect(nil), afterProduction...),
	}
}

func (ma *ManaAbility) currentProductions(g GameReader, sourceID uuid.UUID) []ManaProduction {
	if ma.dynamicProductions != nil {
		productions := ma.dynamicProductions(g, sourceID)
		if productions == nil {
			return []ManaProduction{}
		}
		return productions
	}
	return ma.Productions
}

// ManaBonusAbility grants bonus mana when matching permanents are tapped for mana.
// E.g. Gauntlet of Might: "Whenever a Mountain is tapped for mana, add {R}."
// If AttachedOnly is true, the filter is ignored and only the attached permanent
// triggers the bonus (for auras like Wild Growth).
// If MatchProduced is true, the bonus mana color matches whatever color the land
// produced (e.g. Mana Flare: "adds one mana of any type that land produced").
type ManaBonusAbility struct {
	BaseAbility
	Filter        PermanentFilter
	BonusMana     Color
	AttachedOnly  bool
	MatchProduced bool
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

// NewManaFlareAbility creates a mana bonus that adds one mana of the same type
// produced when a matching land is tapped (e.g. Mana Flare).
func NewManaFlareAbility(filter PermanentFilter) *ManaBonusAbility {
	return &ManaBonusAbility{
		BaseAbility: BaseAbility{
			id:          uuid.New(),
			abilityType: AbilityStatic,
		},
		Filter:        filter,
		MatchProduced: true,
	}
}
