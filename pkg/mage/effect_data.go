package mage

import (
	"fmt"

	"github.com/google/uuid"
)

// EffectContext carries runtime state through a pipeline of effects. It is
// constructed by ApplyEffect (or pipeline steps) and threaded through executor
// dispatch so intermediate values (snapshotted permanent properties, chosen
// IDs, etc.) can flow between steps without closures.
type EffectContext struct {
	Game       *Game
	SourceID   uuid.UUID
	Controller uuid.UUID
	Targets    []uuid.UUID
	Vars       map[string]any

	// DamageDistribution is set when the resolving stack object carries a
	// pre-chosen damage distribution (divided-damage spells/abilities).
	DamageDistribution map[uuid.UUID]int
}

// SetInt stores an integer variable in the context.
func (ctx *EffectContext) SetInt(name string, val int) {
	ctx.Vars[name] = val
}

// GetInt retrieves an integer variable. Panics if the variable is missing or
// not an int — this is a programming error in the effect definition, and should
// be caught during testing.
func (ctx *EffectContext) GetInt(name string) int {
	v, ok := ctx.Vars[name]
	if !ok {
		panic(fmt.Sprintf("EffectContext: missing int variable %q", name))
	}
	n, ok := v.(int)
	if !ok {
		panic(fmt.Sprintf("EffectContext: variable %q is %T, not int", name, v))
	}
	return n
}

// SetUUID stores a UUID variable in the context.
func (ctx *EffectContext) SetUUID(name string, val uuid.UUID) {
	ctx.Vars[name] = val
}

// GetUUID retrieves a UUID variable. Panics on missing or wrong type.
func (ctx *EffectContext) GetUUID(name string) uuid.UUID {
	v, ok := ctx.Vars[name]
	if !ok {
		panic(fmt.Sprintf("EffectContext: missing UUID variable %q", name))
	}
	id, ok := v.(uuid.UUID)
	if !ok {
		panic(fmt.Sprintf("EffectContext: variable %q is %T, not uuid.UUID", name, v))
	}
	return id
}

// TryGetUUID retrieves a UUID variable, returning uuid.Nil if missing.
func (ctx *EffectContext) TryGetUUID(name string) uuid.UUID {
	v, ok := ctx.Vars[name]
	if !ok {
		return uuid.Nil
	}
	id, ok := v.(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return id
}

// SetBool stores a boolean variable in the context.
func (ctx *EffectContext) SetBool(name string, val bool) {
	ctx.Vars[name] = val
}

// GetBool retrieves a boolean variable. Returns false if missing.
func (ctx *EffectContext) GetBool(name string) bool {
	v, ok := ctx.Vars[name]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	if !ok {
		return false
	}
	return b
}
