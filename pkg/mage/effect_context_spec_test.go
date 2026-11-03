package mage

import (
	"testing"
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestPipelineVariablesUseCurrentContext(t *testing.T) {
	value := VarInt("amount")
	player := VarPlayer("player")
	first := &EffectContext{Vars: map[string]any{"amount": 3, "player": uuid.New()}}
	second := &EffectContext{Vars: map[string]any{"amount": 7, "player": uuid.New()}}
	if value.Resolve(first) != 3 || value.Resolve(second) != 7 || value.Resolve(first) != 3 {
		t.Fatal("value does not use the supplied context")
	}
	if player.Select(first)[0] != first.TryGetUUID("player") || player.Select(second)[0] != second.TryGetUUID("player") {
		t.Fatal("selector does not use the supplied context")
	}
}

type contextProbe struct{ inspect func(*EffectContext) }

func (contextProbe) Text() string                     { return "inspect context" }
func (contextProbe) Properties() EffectProperties     { return EffectProperties{} }
func (e contextProbe) Apply(ctx *EffectContext) error { e.inspect(ctx); return nil }

func TestTriggerEventIsSeparateFromDeclaredTargets(t *testing.T) {
	g := newPriorityTestGame()
	owner := g.players[0].PlayerID()
	eventSource := uuid.New()
	called := false
	ability := NewTriggered(EvtSpellCast, false, contextProbe{inspect: func(ctx *EffectContext) {
		called = true
		if len(ctx.Targets) != 0 || len(ctx.TargetGroups) != 0 {
			t.Fatal("event references stored as targets")
		}
		if got := EventSource().Select(ctx); len(got) != 1 || got[0] != eventSource {
			t.Fatal("event source was lost")
		}
		if got := EventPlayer().Select(ctx); len(got) != 1 || got[0] != owner {
			t.Fatal("event player was lost")
		}
		if EventAmount().Resolve(ctx) != 4 {
			t.Fatal("event amount was lost")
		}
	}})
	g.PutOnBattlefield(NewArtifact("Observer", "{0}", WithAbility(ability)), owner)
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: eventSource, PlayerID: owner, Amount: 4})
	g.PutTriggersOnStack()
	object := g.stack.Peek()
	if object == nil || len(object.Targets) != 0 || object.TriggerEvent == nil {
		t.Fatal("stack mixes event references and targets")
	}
	branch := g.Clone()
	branch.stack.Peek().TriggerEvent.Amount = 9
	if object.TriggerEvent.Amount != 4 {
		t.Fatal("branch changed event snapshot")
	}
	g.ResolveStack()
	if !called {
		t.Fatal("trigger did not resolve")
	}
}

func TestResolutionPreservesEmptyTargetGroups(t *testing.T) {
	g := newPriorityTestGame()
	owner := g.players[0].PlayerID()
	creature := g.PutOnBattlefield(NewCreature("Target", "{0}", 2, 2), owner)
	called := false
	object := newStackObject(owner, uuid.New(), nil, []Effect{contextProbe{inspect: func(ctx *EffectContext) {
		called = true
		if len(TargetGroup(0).Select(ctx)) != 0 {
			t.Fatal("empty group acquired another group's target")
		}
		group := TargetGroup(1).Select(ctx)
		if len(group) != 1 || group[0] != creature.ID() {
			t.Fatal("group boundary changed")
		}
	}}}, []uuid.UUID{creature.ID()}, 0, true)
	object.TargetGroups = [][]uuid.UUID{nil, {creature.ID()}}
	object.TargetSpecs = []Target{TargetCreature()}
	object.TargetSource = creature.Card
	g.pushStack(object)
	g.ResolveStack()
	if !called {
		t.Fatal("effect did not resolve")
	}
}

func TestSpellAnnouncementKeepsOptionalTargetGroups(t *testing.T) {
	g := newPriorityTestGame()
	player := g.players[0]
	creature := g.PutOnBattlefield(NewCreature("Target", "{0}", 2, 2), player.PlayerID())
	called := false
	spell := NewInstant("Grouped", "{0}", NewSpell(
		contextProbe{inspect: func(ctx *EffectContext) {
			called = true
			if len(ctx.ModesChosen) > 0 {
				t.Fatal("nonmodal spell has chosen modes")
			}
			if len(TargetGroup(0).Select(ctx)) != 0 || len(TargetGroup(1).Select(ctx)) != 1 {
				t.Fatal("announcement changed optional group boundaries")
			}
		}}, WithTarget(TargetUpToNCreatures(2)), WithTarget(TargetUpToNCreatures(1))))
	player.AddToHand(spell)
	if err := g.CastSpellWithChoices(player.PlayerID(), spell.ID(), SpellChoices{TargetGroups: [][]uuid.UUID{nil, {creature.ID()}}}); err != nil {
		t.Fatal(err)
	}
	object := g.stack.Peek()
	if len(object.TargetGroups) != 2 || len(object.TargetGroups[0]) != 0 || len(object.TargetGroups[1]) != 1 {
		t.Fatal("stack lost group boundaries")
	}
	g.ResolveStack()
	if !called {
		t.Fatal("grouped spell did not resolve")
	}
}

func TestGameCloneKeepsResolvingSourceBinding(t *testing.T) {
	g := newPriorityTestGame()
	owner := g.players[0].PlayerID()
	source := g.PutOnBattlefield(NewCreature("Source", "{0}", 2, 2), owner)
	object := newStackObject(owner, source.ID(), nil, nil, nil, 0, true)
	object.SourceIncarnation = source.incarnationID
	defer g.resolution.Begin(object)()
	branch := g.Clone()
	if branch.resolution.resolvingSourceID != source.ID() || branch.resolution.resolvingSourceIncarnation != source.incarnationID {
		t.Fatal("clone lost the resolving source incarnation")
	}
}

func TestGraveyardSourceContextUsesTheCardInItsCurrentZone(t *testing.T) {
	g := newPriorityTestGame()
	owner := g.players[0].PlayerID()
	var card *Card
	ability := NewTriggered(EvtUpkeep, false, contextProbe{inspect: func(ctx *EffectContext) {
		if ctx.SourcePerm != nil || ctx.SourceCard == nil || ctx.SourceCard.ID() != card.ID() {
			t.Fatal("graveyard trigger used an old battlefield source")
		}
		if err := MoveObjects(SourceObject(), ZoneHand).Apply(ctx); err != nil {
			t.Fatal(err)
		}
	}}).FromGraveyard()
	card = NewArtifact("Graveyard source", "{0}", WithAbility(ability))
	permanent := g.PutOnBattlefield(card, owner)
	g.PutPermanentIntoGraveyard(permanent)
	g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: owner})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if g.CardZone(card.ID()) != ZoneHand {
		t.Fatal("graveyard source could not move itself")
	}
}

func TestDelayedTriggerSeparatesRememberedObjectsFromTargets(t *testing.T) {
	g := newPriorityTestGame()
	owner := g.players[0].PlayerID()
	remembered := g.PutOnBattlefield(NewCreature("Remembered", "{0}", 2, 2), owner)
	called := false
	g.RegisterDelayedTrigger(&DelayedTrigger{EventType: EvtEndStep, Controller: owner, TargetID: remembered.ID(), Effects: []Effect{contextProbe{inspect: func(ctx *EffectContext) {
		called = true
		if len(ctx.Targets) > 0 || ctx.TriggerEvent == nil {
			t.Fatal("delayed trigger mixed remembered objects with declared targets")
		}
		if ids := RememberedObject(0).Select(ctx); len(ids) != 1 || ids[0] != remembered.ID() {
			t.Fatal("delayed trigger lost its remembered object")
		}
	}}}})
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: owner})
	object := g.stack.Peek()
	if object == nil || len(object.Targets) > 0 {
		t.Fatal("delayed trigger stored a remembered object as a target")
	}
	g.ResolveStack()
	if !called {
		t.Fatal("delayed trigger did not resolve")
	}
}

func TestSpellCopyKeepsTheOriginalTargetIncarnation(t *testing.T) {
	g := newPriorityTestGame()
	player := g.players[0]
	target := g.PutOnBattlefield(NewCreature("Target", "{0}", 2, 2), player.PlayerID())
	called := false
	spell := NewInstant("Copy source", "{0}", NewSpell(contextProbe{inspect: func(*EffectContext) { called = true }}, WithTarget(TargetCreature())))
	player.AddToHand(spell)
	if err := g.CastSpellByID(player.PlayerID(), spell.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(target)
	g.PutOnBattlefield(target.Card, player.PlayerID())
	if g.CopySpellOnStack(spell.ID(), player.PlayerID(), false) == nil {
		t.Fatal("spell was not copied")
	}
	g.ResolveStack()
	if called {
		t.Fatal("spell copy acquired the returned permanent as a new target")
	}
}
