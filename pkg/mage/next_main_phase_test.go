package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestNextMainPhaseTiming(t *testing.T) {
	for _, step := range []core.PhaseStep{core.PrecombatMain, core.PostcombatMain} {
		t.Run(step.String(), func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			g.FireEvent(core.GameEvent{Type: core.EvtMainPhase, PlayerID: a.PlayerID(), Flag: true})
			effect := AtNext(YourMainPhase, GainLifeFor(You, Fixed(2)))
			card := NewInstant("Schedule", "", Spell(effect))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.RunStepWithPriority(core.Cleanup)
			g.SetActivePlayerIndex(1)
			g.RunStepWithPriority(core.PrecombatMain)
			g.RunStepWithPriority(core.PostcombatMain)
			if a.Life() != 20 || len(g.DelayedTriggers()) != 1 {
				t.Fatal("schedule used an earlier or opposing main phase")
			}
			clone := g.Clone()
			g.SetActivePlayerIndex(0)
			g.RunStepWithPriority(step)
			if a.Life() != 22 || b.Life() != 20 || len(g.DelayedTriggers()) != 0 {
				t.Fatal("next own main phase did not resolve once")
			}
			g.RunStepWithPriority(core.PostcombatMain)
			if a.Life() != 22 || clone.GetPlayer(a.PlayerID()).Life() != 20 || len(clone.DelayedTriggers()) != 1 {
				t.Fatal("schedule repeated or changed its clone")
			}
			clone.SetActivePlayerIndex(0)
			clone.RunStepWithPriority(step)
			if clone.GetPlayer(a.PlayerID()).Life() != 22 {
				t.Fatal("clone lost schedule")
			}
		})
	}
}

func TestNextMainPhasePendingAndController(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	if err := ApplyEffect(g, AtNext(YourMainPhase, GainLifeFor(You, Fixed(2))), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PhaseOut(source.ID())
	g.FireEvent(core.GameEvent{Type: core.EvtMainPhase, PlayerID: a.PlayerID()})
	if a.Life() != 20 || g.StackSize() != 0 || len(g.DelayedTriggers()) != 0 {
		t.Fatal("body ran before priority or phased source stopped the schedule")
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || g.StackPeek().Controller != a.PlayerID() {
		t.Fatal("pending trigger lost controller")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 22 {
		t.Fatal("copy did not use its own controller")
	}
}

func TestNextMainPhaseSkippedAndExtra(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	if err := ApplyEffect(g, AtNext(YourMainPhase, GainLifeFor(You, Fixed(2))), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.SkipNextOccurrenceOfStep(core.PrecombatMain)
	g.turns.BuildNextTurn()
	for {
		step, ok := g.turns.PopNextStep()
		if !ok {
			t.Fatal("missing postcombat main phase")
		}
		if step == core.PrecombatMain {
			t.Fatal("skip did not remove precombat main phase")
		}
		if step == core.PostcombatMain {
			if a.Life() != 20 || len(g.DelayedTriggers()) != 1 {
				t.Fatal("skipped main phase consumed schedule")
			}
			g.RunStepWithPriority(step)
			break
		}
	}
	if a.Life() != 22 {
		t.Fatal("postcombat main phase did not trigger")
	}
	if err := ApplyEffect(g, AtNext(YourMainPhase, GainLifeFor(You, Fixed(3))), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.turns.Schedule().Remaining = nil
	g.AppendExtraStep(core.PostcombatMain)
	step, ok := g.turns.PopNextStep()
	if !ok || step != core.PostcombatMain {
		t.Fatal("extra main phase missing")
	}
	g.RunStepWithPriority(step)
	if a.Life() != 25 {
		t.Fatal("extra main phase did not trigger")
	}
}

func TestNextMainPhaseDefinitions(t *testing.T) {
	for _, tc := range []struct {
		effect ResolvingEffect
		valid  bool
	}{
		{AtNext(YourMainPhase, GainLifeFor(You, Fixed(1))), true},
		{AtNext(YourMainPhase, nil), false},
		{AtNext(YourMainPhase, GainLifeFor(FromPlayer(BindPlayer("missing")), Fixed(1))), false},
	} {
		card := NewInstant("Definition", "", Spell(tc.effect))
		if err := ValidateCardDefinition(card.Name(), card); (err == nil) != tc.valid {
			t.Fatalf("valid=%v: %v", tc.valid, err)
		}
	}
}

func TestNextMainPhaseSavedSpellTargets(t *testing.T) {
	for _, change := range []string{"live", "departed", "recast"} {
		t.Run(change, func(t *testing.T) {
			g, a, b, spell := spellSourceFixture(t)
			saved := g.CaptureSpellDamageSource(spell.ID)
			target := DeclareSpellTarget("watched")
			ctx := &EffectContext{Game: g, Controller: a.PlayerID(), targetBindings: map[*targetDeclaration][]capturedPermanent{
				target.targetDeclaration: {{id: spell.ID}, {id: spell.ID}, {id: spell.SourceID}, {id: uuid.New()}},
			}}
			if err := AtNext(YourMainPhase, AddManaAmount(core.Colorless, ManaValueOfSpellWithLKI(target))).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			g.ChangeObjectColor(spell.ID, core.Blue)
			spell.Controller = b.PlayerID()
			if change != "live" {
				g.CounterStackObject(spell.ID)
			}
			if change == "recast" {
				a.RemoveFromGraveyard(spell.Card.ID())
				a.AddToHand(spell.Card)
				a.ManaPool().Add(core.Red, 1)
				if err := g.CastSpellByID(a.PlayerID(), spell.Card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
			}
			for range 2 {
				sources := g.WaitingDelayedSpellSources()
				if len(sources) != 1 || !sources[0].SameObject(saved) || !sources[0].HasColor(core.Blue) || sources[0].Controller() != b.PlayerID() {
					t.Fatal("target identity or current/departure facts lost", sources)
				}
				sources[0].card.AddType(core.TypeArtifact)
				if g.WaitingDelayedSpellSources()[0].HasType(core.TypeArtifact) {
					t.Fatal("query exposed mutable card")
				}
			}
			if got := clone.WaitingDelayedSpellSources(); len(got) != 1 || !got[0].HasColor(core.Red) {
				t.Fatal("clone changed before departure")
			}
			binding := BindDamageSource("choice")
			if err := ChooseDamageSourceFromWaitingDelayedSpellBindings(You, DamageSourceHasColor(core.Blue), binding).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if !FromDamageSource(binding).resolveDamageSource(ctx).SameObject(saved) {
				t.Fatal("waiting source choice lost target")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtMainPhase, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if len(g.WaitingDelayedSpellSources()) != 0 || len(g.StackBoundSpellSources()) != 1 || len(clone.WaitingDelayedSpellSources()) != 1 {
				t.Fatal("waiting/stack source lifetime")
			}
			g.ResolveTopOfStack()
			if len(g.StackBoundSpellSources()) != 0 {
				t.Fatal("resolved body retained reference")
			}
		})
	}
}
