package mage

import "github.com/google/uuid"

// AbilityType classifies abilities.
type AbilityType int

const (
	AbilitySpell AbilityType = iota
	AbilityActivated
	AbilityTriggered
	AbilityMana
	AbilityStatic
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
	ID_         uuid.UUID
	Source_     uuid.UUID
	Controller_ uuid.UUID
	Type_       AbilityType
}

func (a *BaseAbility) AbilityID() uuid.UUID      { return a.ID_ }
func (a *BaseAbility) Source() uuid.UUID          { return a.Source_ }
func (a *BaseAbility) SetSource(id uuid.UUID)     { a.Source_ = id }
func (a *BaseAbility) Controller() uuid.UUID      { return a.Controller_ }
func (a *BaseAbility) SetController(id uuid.UUID) { a.Controller_ = id }
func (a *BaseAbility) Type() AbilityType          { return a.Type_ }

// KeywordAbility is a static ability granting a keyword.
type KeywordAbility struct {
	BaseAbility
	Keyword Keyword
}

func HasKeyword(k Keyword) *KeywordAbility {
	return &KeywordAbility{
		BaseAbility: BaseAbility{
			ID_:   uuid.New(),
			Type_: AbilityStatic,
		},
		Keyword: k,
	}
}

// ProtectionAbility grants protection from specific colors.
type ProtectionAbility struct {
	BaseAbility
	FromColors []Color
	Filter     func(Card) bool
}

func ProtectionFromColor(c Color) *ProtectionAbility {
	return &ProtectionAbility{
		BaseAbility: BaseAbility{
			ID_:   uuid.New(),
			Type_: AbilityStatic,
		},
		FromColors: []Color{c},
		Filter: func(card Card) bool {
			for _, col := range card.ManaCost().Colors() {
				if col == c {
					return true
				}
			}
			return false
		},
	}
}

func ProtectionFromColors(cs ...Color) *ProtectionAbility {
	colorSet := make(map[Color]bool)
	for _, c := range cs {
		colorSet[c] = true
	}
	return &ProtectionAbility{
		BaseAbility: BaseAbility{
			ID_:   uuid.New(),
			Type_: AbilityStatic,
		},
		FromColors: cs,
		Filter: func(card Card) bool {
			for _, col := range card.ManaCost().Colors() {
				if colorSet[col] {
					return true
				}
			}
			return false
		},
	}
}

// Blocks returns true if this protection prevents interaction with the given card.
func (pa *ProtectionAbility) Blocks(card Card) bool {
	if pa.Filter != nil {
		return pa.Filter(card)
	}
	return false
}

// StaticAbilityHolder holds continuous effects as a static ability.
type StaticAbilityHolder struct {
	BaseAbility
	Effects []ContinuousEffect
}

func StaticAbility(effects ...ContinuousEffect) *StaticAbilityHolder {
	return &StaticAbilityHolder{
		BaseAbility: BaseAbility{
			ID_:   uuid.New(),
			Type_: AbilityStatic,
		},
		Effects: effects,
	}
}

// ManaAbilityImpl is a mana ability that taps to add mana.
type ManaAbilityImpl struct {
	BaseAbility
	Color Color
}

func NewManaAbility(c Color) *ManaAbilityImpl {
	return &ManaAbilityImpl{
		BaseAbility: BaseAbility{
			ID_:   uuid.New(),
			Type_: AbilityMana,
		},
		Color: c,
	}
}

// ManaBonusAbility grants bonus mana when matching permanents are tapped for mana.
// E.g. Gauntlet of Might: "Whenever a Mountain is tapped for mana, add {R}."
type ManaBonusAbility struct {
	BaseAbility
	Filter    PermanentFilter
	BonusMana Color
}

func NewManaBonusAbility(filter PermanentFilter, bonusMana Color) *ManaBonusAbility {
	return &ManaBonusAbility{
		BaseAbility: BaseAbility{
			ID_:   uuid.New(),
			Type_: AbilityStatic,
		},
		Filter:    filter,
		BonusMana: bonusMana,
	}
}
