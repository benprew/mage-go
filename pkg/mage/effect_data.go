package mage

import (
	"fmt"

	"github.com/google/uuid"
)

// EffectData describes an effect without execution logic. Effects that implement
// this interface are inert data — the executor dispatches on their concrete type
// to perform mutations. Use DataEffect() to wrap an EffectData into the Effect
// interface so it can be used anywhere the old interface is expected.
type EffectData interface {
	EffectText() string
	EffectProps() EffectProperties
}

// EffectContext carries runtime state through a pipeline of effects. It is
// created by the dataEffectAdapter when an EffectData is executed through the
// legacy Effect interface, and threaded through Pipeline steps so intermediate
// values (snapshotted permanent properties, chosen IDs, etc.) can flow between
// steps without closures.
type EffectContext struct {
	Game       *Game
	SourceID   uuid.UUID
	Controller uuid.UUID
	Targets    []uuid.UUID
	Vars       map[string]any
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

// dataEffectAdapter wraps an EffectData into the Effect interface, creating an
// EffectContext and dispatching through ExecuteEffect.
type dataEffectAdapter struct {
	data EffectData
}

// DataEffect wraps an EffectData value into the Effect interface.
func DataEffect(data EffectData) Effect {
	return &dataEffectAdapter{data: data}
}

func (a *dataEffectAdapter) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	ctx := &EffectContext{
		Game:       g,
		SourceID:   sourceID,
		Controller: controller,
		Targets:    targets,
		Vars:       make(map[string]any),
	}
	return ExecuteEffect(ctx, a.data)
}

func (a *dataEffectAdapter) Text() string              { return a.data.EffectText() }
func (a *dataEffectAdapter) Properties() EffectProperties { return a.data.EffectProps() }

// Unwrap returns the underlying EffectData, allowing the executor or AI to
// inspect the data structure without going through the Effect interface.
func (a *dataEffectAdapter) Unwrap() EffectData { return a.data }

// UnwrapEffect extracts the EffectData from an Effect that was created via
// DataEffect(). This allows pre-built effects (DealDamage, GainLifeTarget, etc.)
// to be used as pipeline steps inside ModalEffect or Pipeline. Panics if the
// effect is not a dataEffectAdapter.
func UnwrapEffect(e Effect) EffectData {
	if a, ok := e.(*dataEffectAdapter); ok {
		return a.data
	}
	panic(fmt.Sprintf("UnwrapEffect: %T is not a DataEffect", e))
}
