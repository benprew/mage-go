package mage

import (
	"slices"
	"uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// DamageRecipientSelector selects the objects or players that receive damage.
type DamageRecipientSelector = ObjectSelector

type selectedLifeEffect struct {
	player PlayerSelector
	amount ValueSource
	lose   bool
}

func (e *selectedLifeEffect) Targeting(player PlayerSelector) *selectedLifeEffect {
	e.player = player
	return e
}

func (e *selectedLifeEffect) Text() string {
	if e.lose {
		return "lose " + e.amount.Text() + " life"
	}
	return "gain " + e.amount.Text() + " life"
}
func (e *selectedLifeEffect) Properties() EffectProperties {
	if e.lose {
		return EffectProperties{Outcome: OutcomeDetriment}
	}
	properties := EffectProperties{Outcome: OutcomeBenefit}
	if amount, ok := e.amount.(fixedValue); ok {
		properties.LifeGain = amount.n
	}
	return properties
}
func (e *selectedLifeEffect) Apply(ctx *EffectContext) error {
	amount := e.amount.Resolve(ctx)
	if amount <= 0 {
		return nil
	}
	for _, id := range e.player.Select(ctx) {
		player := ctx.Game.GetPlayer(id)
		if player == nil {
			continue
		}
		if e.lose {
			ctx.Game.PlayerLoseLife(player, amount)
		} else {
			ctx.Game.PlayerGainLife(player, amount)
			if !ctx.Game.IsLichActive(id) {
				ctx.Game.FireEvent(core.GameEvent{Type: core.EvtLifeGained, PlayerID: id, Amount: amount})
			}
		}
	}
	return nil
}

type legacyFirstReference struct{}

func (legacyFirstReference) Text() string { return "target" }
func (legacyFirstReference) Select(ctx *EffectContext) []uuid.UUID {
	targets := ctx.legacyTargets()
	if len(targets) == 0 {
		return nil
	}
	return slices.Clone(targets[:1])
}

type legacyPlayerReference struct{}

func (legacyPlayerReference) Text() string { return "that player" }
func (legacyPlayerReference) Select(ctx *EffectContext) []uuid.UUID {
	ids := legacyFirstReference{}.Select(ctx)
	if len(ids) > 0 && ctx.Game.GetPlayer(ids[0]) != nil {
		return ids
	}
	return []uuid.UUID{ctx.Controller}
}

type battlefieldObjects struct{ filter PermanentFilter }

func (battlefieldObjects) Text() string { return "matching permanents" }
func (s battlefieldObjects) Select(ctx *EffectContext) []uuid.UUID {
	permanents := ctx.Game.FilterBattlefield(s.filter)
	ids := make([]uuid.UUID, len(permanents))
	for i, p := range permanents {
		ids[i] = p.ID()
	}
	return ids
}

// BattlefieldObjects selects matching permanents without targeting them.
func BattlefieldObjects(filter PermanentFilter) ObjectSelector {
	return battlefieldObjects{filter: filter}
}

type destroyOptions struct{ noRegeneration bool }

// DestroyOpt configures a destruction effect.
type DestroyOpt func(*destroyOptions)

// WithoutRegeneration prevents regeneration for this destruction.
func WithoutRegeneration() DestroyOpt {
	return func(options *destroyOptions) { options.noRegeneration = true }
}

type selectedDestroyEffect struct {
	target  ObjectSelector
	options destroyOptions
}

// Destroy destroys all selected permanents as one event.
func Destroy(target ObjectSelector, options ...DestroyOpt) Effect {
	effect := &selectedDestroyEffect{target: target}
	for _, option := range options {
		option(&effect.options)
	}
	return effect
}
func (*selectedDestroyEffect) Text() string { return "destroy" }
func (e *selectedDestroyEffect) Properties() EffectProperties {
	_, mass := e.target.(battlefieldObjects)
	return EffectProperties{Outcome: OutcomeDetriment, Destroys: !e.options.noRegeneration, Mass: mass}
}
func (e *selectedDestroyEffect) Apply(ctx *EffectContext) error {
	var permanents []*Permanent
	seen := make(map[uuid.UUID]bool)
	for _, id := range currentObjectIDs(ctx, e.target) {
		if seen[id] {
			continue
		}
		seen[id] = true
		permanent := ctx.Game.MutablePermanent(id)
		if permanent == nil || permanent.HasKeyword(core.Indestructible) {
			continue
		}
		if e.options.noRegeneration {
			permanent.GrantBaseAttr(core.CantRegenerate)
		}
		permanents = append(permanents, permanent)
	}
	ctx.Game.DestroyPermanents(permanents)
	return nil
}

// DrawForPlayers draws the supplied number of cards for each selected player.
func DrawForPlayers(player PlayerSelector, amount ValueSource) *drawCardsTargetEffect {
	return DrawCards(amount).Targeting(player)
}

// Discard discards the supplied number of cards for each selected player.
type discardOptions struct{ random bool }

// DiscardOpt configures a discard effect.
type DiscardOpt func(*discardOptions)

// AtRandom makes discard choices use the game's random source.
func AtRandom() DiscardOpt { return func(options *discardOptions) { options.random = true } }
func Discard(player PlayerSelector, amount ValueSource, options ...DiscardOpt) *discardCardsEffect {
	configuration := discardOptions{}
	for _, option := range options {
		option(&configuration)
	}
	effect := DiscardCards(amount).Targeting(player)
	effect.random = configuration.random
	return effect
}

type variableObjectReference struct{ name string }

func (r variableObjectReference) Text() string { return r.name }
func (r variableObjectReference) Select(ctx *EffectContext) []uuid.UUID {
	id := ctx.TryGetUUID(r.name)
	if id == uuid.Nil() {
		return nil
	}
	return []uuid.UUID{id}
}

// VarObject selects an object stored in the execution variables.
func VarObject(name string) ObjectSelector { return variableObjectReference{name: name} }

type legacyDamageSourceReference struct{}

func (legacyDamageSourceReference) Text() string { return "source" }
func (legacyDamageSourceReference) Select(ctx *EffectContext) []uuid.UUID {
	return []uuid.UUID{ctx.SourceID}
}
