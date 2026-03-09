package mage

import "github.com/google/uuid"

// DamageToPlayerAction represents damage about to be dealt to a player.
type DamageToPlayerAction struct {
	source       uuid.UUID
	playerID     uuid.UUID
	amount       int
	combatDamage bool
}

func NewDamageToPlayerAction(source, playerID uuid.UUID, amount int, combatDamage bool) *DamageToPlayerAction {
	return &DamageToPlayerAction{
		source:       source,
		playerID:     playerID,
		amount:       amount,
		combatDamage: combatDamage,
	}
}

func (a *DamageToPlayerAction) ActionSource() uuid.UUID { return a.source }
func (a *DamageToPlayerAction) PlayerID() uuid.UUID     { return a.playerID }
func (a *DamageToPlayerAction) Amount() int             { return a.amount }
func (a *DamageToPlayerAction) IsCombatDamage() bool    { return a.combatDamage }

// WithAmount returns a copy with the amount changed.
func (a *DamageToPlayerAction) WithAmount(amount int) *DamageToPlayerAction {
	cp := *a
	cp.amount = amount
	return &cp
}

// DamageToCreatureAction represents damage about to be dealt to a creature.
type DamageToCreatureAction struct {
	source       uuid.UUID
	permanentID  uuid.UUID
	amount       int
	combatDamage bool
}

func NewDamageToCreatureAction(source, permanentID uuid.UUID, amount int, combatDamage bool) *DamageToCreatureAction {
	return &DamageToCreatureAction{
		source:       source,
		permanentID:  permanentID,
		amount:       amount,
		combatDamage: combatDamage,
	}
}

func (a *DamageToCreatureAction) ActionSource() uuid.UUID  { return a.source }
func (a *DamageToCreatureAction) PermanentID() uuid.UUID   { return a.permanentID }
func (a *DamageToCreatureAction) Amount() int              { return a.amount }
func (a *DamageToCreatureAction) IsCombatDamage() bool     { return a.combatDamage }

// WithAmount returns a copy with the amount changed.
func (a *DamageToCreatureAction) WithAmount(amount int) *DamageToCreatureAction {
	cp := *a
	cp.amount = amount
	return &cp
}
