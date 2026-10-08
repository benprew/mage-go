package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDrawStepLifeReferences(t *testing.T) {
	for _, state := range []string{"live", "phase", "cycle", "depart", "return", "control", "remove-ability", "restore-ability"} {
		t.Run(state, func(t *testing.T) {
			a := &acceptingMayPlayer{NewBasePlayer("A")}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewEnchantment("Rule", "{W}", WithAbility(Static(SkipDrawStepForLife(2))))
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			saved := g.CaptureDamageSource(source.ID())
			clone := g.Clone()
			var stale ReplacementEffect
			for _, r := range g.effects.cycleReplacements {
				if _, ok := r.(*drawStepLifeReplacement); ok {
					stale = r.Clone()
					break
				}
			}
			if stale == nil {
				t.Fatal("missing replacement")
			}
			want := 1
			switch state {
			case "phase":
				g.PhaseOut(source.ID())
				want = 0
			case "cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "depart", "return":
				g.RemoveFromBattlefield(source)
				g.zones.ClearLKI()
				want = 0
				if state == "return" {
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "remove-ability", "restore-ability":
				g.AddContinuousEffect(RemoveAllAbilities(source.ID(), core.EndOfTurn))
				want = 0
				if state == "restore-ability" {
					g.effects.RemoveEndOfTurn()
					g.ApplyContinuousEffects()
					want = 1
				}
			}
			g.effects.cycleReplacements = []ReplacementEffect{stale}
			refs := g.WaitingDrawStepLifePermanentSources()
			if len(refs) != want {
				t.Fatalf("sources=%d want=%d", len(refs), want)
			}
			if want == 1 {
				if !refs[0].SameObject(saved) {
					t.Fatal("wrong incarnation")
				}
				if state == "control" && refs[0].Controller() != b.PlayerID() {
					t.Fatal("stale controller")
				}
				refs[0].card.SetOwner(b.PlayerID())
				if g.WaitingDrawStepLifePermanentSources()[0].card.Owner() != a.PlayerID() {
					t.Fatal("query changed card")
				}
			}
			if refs := clone.WaitingDrawStepLifePermanentSources(); len(refs) != 1 || !refs[0].SameObject(saved) {
				t.Fatal("clone changed")
			}
			if state == "return" {
				g.ApplyContinuousEffects()
				refs = g.WaitingDrawStepLifePermanentSources()
				if len(refs) != 1 || refs[0].SameObject(saved) {
					t.Fatal("new registration did not use returned incarnation")
				}
			}
		})
	}
}

func TestWaitingDrawStepLifeChoices(t *testing.T) {
	a := &acceptingMayPlayer{NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewEnchantment("Rule", "{W}", WithAbility(Static(SkipDrawStepForLife(2)))), a.PlayerID())
	saved := g.CaptureDamageSource(p.ID())
	rule := g.effects.cycleReplacements[0].Clone()
	g.effects.AddReplacement(rule)
	g.effects.AddReplacement(&drawStepLifeReplacement{replacementBase: replacementBase{sourceID: p.ID()}})
	g.effects.AddReplacement(&drawStepLifeReplacement{replacementBase: replacementBase{sourceID: uuid.New()}, incarnation: uuid.New()})
	g.AddRegenerationShield(p.ID())
	g.PutOnBattlefield(NewEnchantment("Unrelated", "{U}"), b.PlayerID())
	if len(g.WaitingDrawStepLifePermanentSources()) != 1 {
		t.Fatal("invalid or duplicate sources")
	}
	first, second := BindDamageSource("first"), BindDamageSource("second")
	ctx := &EffectContext{Game: g, Controller: b.PlayerID()}
	for _, binding := range []*DamageSourceBinding{first, second} {
		choice := ChooseDamageSourceFromWaitingDrawStepLifeReferences(You, DamageSourceHasColor(core.White), binding)
		if err := ValidateCardDefinition("Choice", NewArtifact("Choice", "", WithAbility(Activate(Costs(), choice)))); err != nil {
			t.Fatal(err)
		}
		if err := choice.Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if !g.resolution.damageSourceBindings[binding].SameObject(saved) {
			t.Fatal("choice identity")
		}
	}
	if err := ChooseDamageSourceFromWaitingDrawStepLifeReferences(You, DamageSourceHasColor(core.Blue), first).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.resolution.damageSourceBindings[first].Valid() || !g.resolution.damageSourceBindings[second].SameObject(saved) {
		t.Fatal("binding isolation")
	}
	if g.effects.ApplyReplacements(&beginDrawStepAction{playerID: a.PlayerID()}, g) != nil {
		t.Fatal("step did not skip")
	}
	if len(g.WaitingDrawStepLifePermanentSources()) != 1 {
		t.Fatal("persistent rule consumed")
	}
	clone := g.Clone()
	g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
	if len(g.WaitingDrawStepLifePermanentSources()) != 0 {
		t.Fatal("pending life gain supplied departed source")
	}
	g.RunStepWithPriority(core.PrecombatMain)
	if a.Life() != 22 || clone.GetPlayer(a.PlayerID()).Life() != 20 || len(clone.WaitingDrawStepLifePermanentSources()) != 1 {
		t.Fatal("pending gain or clone state")
	}
	for _, body := range []ResolvingEffect{
		ChooseDamageSourceFromWaitingDrawStepLifeReferences(nil, DamageSourceFilter{}, first),
		ChooseDamageSourceFromWaitingDrawStepLifeReferences(You, DamageSourceFilter{}, nil),
		ChooseDamageSourceFromWaitingDrawStepLifeReferences(You, DamageSourceHasColor(core.Colorless), first),
		Sequence(DealDamageFrom(FromDamageSource(first), You, Fixed(1)), ChooseDamageSourceFromWaitingDrawStepLifeReferences(You, DamageSourceFilter{}, first)),
	} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), body)))) == nil {
			t.Fatal("invalid declaration")
		}
	}
}
