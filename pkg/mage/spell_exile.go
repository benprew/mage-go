package mage

import "github.com/benprew/mage-go/pkg/mage/core"

type thisSpellRef struct{}

// ThisSpell reads the exact spell whose body is now resolving.
var ThisSpell StackObjectRef = thisSpellRef{}

func (thisSpellRef) stackObjectRef() {}
func (thisSpellRef) inspectDefinition() definitionNode {
	return definitionNode{Role: "stack-object-ref", Text: "resolving spell"}
}
func (thisSpellRef) resolveStackObject(ctx *EffectContext) (*StackObject, bool) {
	for _, obj := range ctx.Game.stack.resolvingReferenceObjects {
		if obj.ID == ctx.Game.resolution.resolvingStackID && !obj.leftStack && !obj.IsAbility && obj.Card != nil {
			return obj, true
		}
	}
	return nil, false
}

type spellExileEffect struct {
	spell   StackObjectRef
	counter core.CounterType
	count   int
}

// ExileSpell moves one present spell to face-up exile.
func ExileSpell(spell StackObjectRef) ResolvingEffect {
	return ExileSpellWithCounters(spell, core.Scream, 0)
}

// ExileSpellWithCounters sets counters before exile observations can read them.
func ExileSpellWithCounters(spell StackObjectRef, counter core.CounterType, count int) ResolvingEffect {
	return &spellExileEffect{spell: spell, counter: counter, count: count}
}
func (*spellExileEffect) Text() string                 { return "exile a spell" }
func (*spellExileEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *spellExileEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"spell", e.spell}}}
	if e.count < 0 || e.counter < 0 || e.counter >= core.NumCounters {
		n.problem = "spell exile requires a nonnegative count and a valid counter type"
	}
	return n
}
func (e *spellExileEffect) Apply(ctx *EffectContext) error {
	if e.spell == nil || e.count < 0 || e.counter < 0 || e.counter >= core.NumCounters {
		return nil
	}
	obj, ok := e.spell.resolveStackObject(ctx)
	if !ok || obj.leftStack {
		return nil
	}
	ctx.Game.stack.RemoveByID(obj.ID)
	ctx.Game.stack.saveSpellInformation(obj)
	obj.leftStack = true
	ctx.Game.zones.ExileCard(obj.Card, ctx.SourceID)
	exiled := ctx.Game.FindExiledCard(obj.Card.ID())
	exiled.Counters[e.counter] = e.count
	exiled.spellCopy = obj.IsCopy
	ctx.Game.recordCardPutIntoExile(obj.Card)
	ctx.Game.FireEvent(core.GameEvent{Type: core.EvtZoneChange, SourceID: obj.Card.ID(), PlayerID: obj.Controller, FromZone: core.ZoneStack, ToZone: core.ZoneExile})
	return nil
}
