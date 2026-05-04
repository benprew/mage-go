package mage

import (
	"github.com/google/uuid"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
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

// ManaAbility is a mana ability that taps to add mana.
// Productions defines what mana is produced. Each entry is a color+amount pair.
// Use AnyColor as the color to let the player choose when activated.
type ManaAbility struct {
	BaseAbility
	Productions []ManaProduction
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
