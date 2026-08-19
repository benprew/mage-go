package mage

import (
	"fmt"

	"github.com/google/uuid"
)

type applyToRandomPermanentEffect struct {
	effect Effect
	filter PermanentFilter
}

// ApplyToRandomPermanent creates an effect that applies effect to one random
// battlefield permanent matching filter. It does nothing when none match.
func ApplyToRandomPermanent(effect Effect, filter PermanentFilter) Effect {
	return &applyToRandomPermanentEffect{effect: effect, filter: filter}
}

func (e *applyToRandomPermanentEffect) Text() string {
	return fmt.Sprintf("apply %s to a random %s", e.effect.Text(), e.filter.Text())
}
func (e *applyToRandomPermanentEffect) Properties() EffectProperties {
	return e.effect.Properties()
}
func (e *applyToRandomPermanentEffect) Apply(ctx *EffectContext) error {
	permanent := ctx.Game.RandomPermanent(e.filter)
	if permanent == nil {
		return nil
	}
	return ApplyEffect(ctx.Game, e.effect, ctx.SourceID, ctx.Controller, []uuid.UUID{permanent.ID()})
}

type applyToRandomSpellOrPermanentEffect struct {
	effect Effect
}

// ApplyToRandomSpellOrPermanent creates an effect that applies effect to one
// random spell on the stack or permanent on the battlefield. It does nothing
// when neither kind of object exists.
func ApplyToRandomSpellOrPermanent(effect Effect) Effect {
	return &applyToRandomSpellOrPermanentEffect{effect: effect}
}

func (e *applyToRandomSpellOrPermanentEffect) Text() string {
	return fmt.Sprintf("apply %s to a random spell or permanent", e.effect.Text())
}
func (e *applyToRandomSpellOrPermanentEffect) Properties() EffectProperties {
	return e.effect.Properties()
}
func (e *applyToRandomSpellOrPermanentEffect) Apply(ctx *EffectContext) error {
	targetID := ctx.Game.RandomSpellOrPermanent()
	if targetID == uuid.Nil {
		return nil
	}
	return ApplyEffect(ctx.Game, e.effect, ctx.SourceID, ctx.Controller, []uuid.UUID{targetID})
}

type applyToRandomPlayerEffect struct {
	effect Effect
}

// ApplyToRandomPlayer creates an effect that applies effect to one random
// player. It does nothing when the game has no players.
func ApplyToRandomPlayer(effect Effect) Effect {
	return &applyToRandomPlayerEffect{effect: effect}
}

func (e *applyToRandomPlayerEffect) Text() string {
	return fmt.Sprintf("apply %s to a random player", e.effect.Text())
}
func (e *applyToRandomPlayerEffect) Properties() EffectProperties {
	return e.effect.Properties()
}
func (e *applyToRandomPlayerEffect) Apply(ctx *EffectContext) error {
	player := ctx.Game.RandomPlayer()
	if player == nil {
		return nil
	}
	return ApplyEffect(ctx.Game, e.effect, ctx.SourceID, ctx.Controller, []uuid.UUID{player.PlayerID()})
}

type applyToRandomDamageTargetEffect struct {
	effect Effect
}

// ApplyToRandomDamageTarget creates an effect that applies effect to one
// random creature or player. It does nothing when no candidate exists.
func ApplyToRandomDamageTarget(effect Effect) Effect {
	return &applyToRandomDamageTargetEffect{effect: effect}
}

func (e *applyToRandomDamageTargetEffect) Text() string {
	return fmt.Sprintf("apply %s to a random creature or player", e.effect.Text())
}
func (e *applyToRandomDamageTargetEffect) Properties() EffectProperties {
	return e.effect.Properties()
}
func (e *applyToRandomDamageTargetEffect) Apply(ctx *EffectContext) error {
	targetID := ctx.Game.RandomDamageTarget()
	if targetID == uuid.Nil {
		return nil
	}
	return ApplyEffect(ctx.Game, e.effect, ctx.SourceID, ctx.Controller, []uuid.UUID{targetID})
}

type setSourceChosenColorAtRandomEffect struct{}

// SetSourceChosenColorAtRandom creates an effect that stores a random color in
// the source permanent's ChosenColor field.
func SetSourceChosenColorAtRandom() Effect { return &setSourceChosenColorAtRandomEffect{} }

func (*setSourceChosenColorAtRandomEffect) Text() string { return "choose a random color" }
func (*setSourceChosenColorAtRandomEffect) Properties() EffectProperties {
	return EffectProperties{}
}
func (*setSourceChosenColorAtRandomEffect) Apply(ctx *EffectContext) error {
	if source := ctx.Game.FindPermanent(ctx.SourceID); source != nil {
		source.ChosenColor = ctx.Game.RandomColor()
	}
	return nil
}

type changeSourceToRandomColorEffect struct{}

// ChangeSourceToRandomColor creates an effect that permanently changes the
// source permanent to one random color.
func ChangeSourceToRandomColor() Effect { return &changeSourceToRandomColorEffect{} }

func (*changeSourceToRandomColorEffect) Text() string {
	return "becomes a random color permanently"
}
func (*changeSourceToRandomColorEffect) Properties() EffectProperties {
	return EffectProperties{}
}
func (*changeSourceToRandomColorEffect) Apply(ctx *EffectContext) error {
	if ctx.Game.FindPermanent(ctx.SourceID) == nil {
		return nil
	}
	return ApplyEffect(ctx.Game, ChangeColorEffect(ctx.Game.RandomColor()), ctx.SourceID, ctx.Controller, []uuid.UUID{ctx.SourceID})
}
