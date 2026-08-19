package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// fightTargetEffect implements CR 701.14 ("X fights Y"): the source permanent
// and the target permanent each deal damage equal to their power to the
// other. Both must be creatures on the battlefield at resolution; if either
// has left or is no longer a creature, neither deals damage (CR 701.14b).
// Source and target deal damage simultaneously.
type fightTargetEffect struct{}

// FightTarget creates an effect where the source creature and the target
// creature fight (CR 701.14). Pair with TargetCreature() (or a more
// specific target like "another target creature").
func FightTarget() Effect {
	return &fightTargetEffect{}
}

// FightTargetStep returns the Effect for use in pipelines / ForEach /
// Modal builders.
func FightTargetStep() Effect { return &fightTargetEffect{} }

func (e *fightTargetEffect) Text() string { return "this creature fights target creature" }
func (e *fightTargetEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

func (*fightTargetEffect) Apply(ctx *EffectContext) error {
	if len(ctx.Targets) == 0 {
		return nil
	}
	src := ctx.Game.FindPermanent(ctx.SourceID)
	tgt := ctx.Game.FindPermanent(ctx.Targets[0])
	if src == nil || tgt == nil {
		return nil
	}
	if !src.HasType(TypeCreature) || !tgt.HasType(TypeCreature) {
		return nil
	}

	fightPermanents(ctx, src, tgt)
	return nil
}

type fightGatheredEffect struct {
	firstVar  string
	secondVar string
}

// FightGathered creates a pipeline step that has two variable-bound creatures
// fight. If either variable no longer identifies a battlefield creature,
// neither deals damage (CR 701.14b).
func FightGathered(firstVar, secondVar string) Effect {
	return &fightGatheredEffect{firstVar: firstVar, secondVar: secondVar}
}

func (*fightGatheredEffect) Text() string { return "those creatures fight each other" }
func (*fightGatheredEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

func (e *fightGatheredEffect) Apply(ctx *EffectContext) error {
	first := ctx.Game.FindPermanent(ctx.TryGetUUID(e.firstVar))
	second := ctx.Game.FindPermanent(ctx.TryGetUUID(e.secondVar))
	if first == nil || second == nil || !first.HasType(TypeCreature) || !second.HasType(TypeCreature) {
		return nil
	}
	fightPermanents(ctx, first, second)
	return nil
}

func fightPermanents(ctx *EffectContext, first, second *Permanent) {
	firstPower := first.CurrentPower(ctx.Game)
	secondPower := second.CurrentPower(ctx.Game)
	firstCard := ctx.Game.FindCardAnywhere(first.ID())
	secondCard := ctx.Game.FindCardAnywhere(second.ID())

	if firstPower > 0 && (secondCard == nil || !second.HasProtectionFromInGame(firstCard, ctx.Game)) {
		ctx.Game.DealDamageToPermanent(second, firstPower, first.ID())
	}
	if secondPower > 0 && (firstCard == nil || !first.HasProtectionFromInGame(secondCard, ctx.Game)) {
		ctx.Game.DealDamageToPermanent(first, secondPower, second.ID())
	}
	ctx.Game.FireEvent(GameEvent{
		Type:     EvtFight,
		SourceID: first.ID(),
		TargetID: second.ID(),
		PlayerID: ctx.Controller,
	})
}

// onPermanentDiesEffect registers a delayed trigger that fires the first
// time a specific permanent (identified by ID) dies — i.e. moves from the
// battlefield to a graveyard, generating EvtCreatureDied with the target's
// permanent ID as SourceID. The trigger is one-shot. The permID can be
// supplied directly or read from a TargetVar saved on the EffectContext.
type onPermanentDiesEffect struct {
	permID  uuid.UUID
	effects []Effect
}

// OnPermanentDiesThisTurn registers a one-shot delayed trigger that fires
// when the specified permanent dies this turn. The provided effects run
// (with the original ability's controller) when the death event fires.
// Use this to model "When [target] dies this turn, do X" delayed triggers
// such as "if it would die this turn, exile it instead" / "create a token
// when this dies" patterns.
func OnPermanentDiesThisTurn(permID uuid.UUID, effects ...Effect) Effect {
	return &onPermanentDiesEffect{permID: permID, effects: effects}
}

// OnTargetDiesThisTurn is the target-resolving variant: at resolution time
// it pulls the permanent ID from ctx.Targets[0] and registers the delayed
// trigger against that ID. Suitable for spells like "Target creature
// fights another. When that creature dies this turn, ___."
func OnTargetDiesThisTurn(effects ...Effect) Effect {
	return &onPermanentDiesEffect{effects: effects}
}

func (e *onPermanentDiesEffect) Text() string {
	return "when that permanent dies this turn, ..."
}
func (e *onPermanentDiesEffect) Properties() EffectProperties { return EffectProperties{} }

func (e *onPermanentDiesEffect) Apply(ctx *EffectContext) error {
	id := e.permID
	if id == uuid.Nil {
		if len(ctx.Targets) == 0 {
			return nil
		}
		id = ctx.Targets[0]
	}
	dt := &DelayedTrigger{
		EventType:     EvtZoneChange,
		MatchEventID:  id,
		MatchFromZone: ZoneBattlefield,
		MatchToZone:   ZoneGraveyard,
		Effects:       e.effects,
		SourceID:      ctx.SourceID,
		Controller:    ctx.Controller,
	}
	ctx.Game.RegisterDelayedTrigger(dt)
	return nil
}
