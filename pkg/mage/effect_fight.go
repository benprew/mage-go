package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// fightTargetEffect implements CR 701.13 ("X fights Y"): the source permanent
// and the target permanent each deal damage equal to their power to the
// other. Both must be creatures on the battlefield at resolution; if either
// has left or is no longer a creature, neither deals damage (CR 701.13c).
// Source and target deal damage simultaneously.
type fightTargetEffect struct{}

// FightTarget creates an effect where the source creature and the target
// creature fight (CR 701.13). Pair with TargetCreature() (or a more
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

	srcPower := src.CurrentPower(ctx.Game)
	tgtPower := tgt.CurrentPower(ctx.Game)

	// Apply protection short-circuits per direction.
	srcCard := ctx.Game.FindCardAnywhere(ctx.SourceID)
	tgtCard := ctx.Game.FindCardAnywhere(tgt.ID())

	if srcPower > 0 && (tgtCard == nil || !tgt.HasProtectionFrom(srcCard)) {
		ctx.Game.DealDamageToPermanent(tgt, srcPower, ctx.SourceID)
	}
	if tgtPower > 0 && (srcCard == nil || !src.HasProtectionFrom(tgtCard)) {
		ctx.Game.DealDamageToPermanent(src, tgtPower, tgt.ID())
	}
	ctx.Game.FireEvent(GameEvent{
		Type:     EvtFight,
		SourceID: ctx.SourceID,
		TargetID: tgt.ID(),
		PlayerID: ctx.Controller,
	})
	return nil
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
