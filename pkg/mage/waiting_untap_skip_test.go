package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingUntapSkipSourcesLifecycle(t *testing.T) {
	for _, state := range []string{"live", "phase", "cycle", "depart", "return", "control", "remove-ability"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewEnchantment("Skip", "{R}", WithAbility(Static(SkipUntapSteps())))
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			var old ReplacementEffect
			for _, replacement := range g.effects.cycleReplacements {
				if _, ok := replacement.(*untapStepSkipReplacement); ok {
					old = replacement.Clone()
				}
			}
			if old == nil {
				t.Fatal("missing static registration")
			}
			clone := g.Clone()
			want := 1
			switch state {
			case "phase":
				g.PhaseOut(p.ID())
				want = 0
			case "cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "depart", "return":
				g.RemoveFromBattlefield(p)
				g.zones.ClearLKI()
				want = 0
				if state == "return" {
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "remove-ability":
				if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				want = 0
			}
			g.effects.replacements = []ReplacementEffect{old, old.Clone()}
			g.effects.cycleReplacements = nil
			refs := g.WaitingUntapSkipPermanentSources()
			if len(refs) != want || old.IsActive(g) != (want == 1) {
				t.Fatalf("query/execution lifecycle: sources=%d want=%d", len(refs), want)
			}
			action := &beginUntapStepAction{playerID: a.PlayerID()}
			if (g.effects.ApplyReplacements(action, g) == nil) != (want == 1) {
				t.Fatal("query differs from execution")
			}
			if want == 1 {
				if !refs[0].SameObject(saved) {
					t.Fatal("wrong incarnation")
				}
				if state == "control" && refs[0].Controller() != b.PlayerID() {
					t.Fatal("stale controller")
				}
				refs[0].card.SetOwner(b.PlayerID())
				if g.WaitingUntapSkipPermanentSources()[0].card.Owner() != a.PlayerID() {
					t.Fatal("query exposed mutable data")
				}
			}
			if refs := clone.WaitingUntapSkipPermanentSources(); len(refs) != 1 || !refs[0].SameObject(saved) {
				t.Fatal("clone changed")
			}
			if state == "return" {
				g.ApplyContinuousEffects()
				refs := g.WaitingUntapSkipPermanentSources()
				if len(refs) != 1 || refs[0].SameObject(saved) || old.IsActive(g) {
					t.Fatal("returned object inherited old rule")
				}
			}
		})
	}
}

func TestWaitingUntapSkipSourcesChoices(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewEnchantment("Skip", "{R}", WithAbility(Static(SkipUntapSteps()))), a.PlayerID())
	g.PutOnBattlefield(NewArtifact("Unrelated", "{R}"), a.PlayerID())
	first, second := BindDamageSource("first"), BindDamageSource("second")
	ctx := &EffectContext{Game: g, Controller: b.PlayerID()}
	for _, binding := range []*DamageSourceBinding{first, second} {
		choice := ChooseDamageSourceFromWaitingUntapSkipReferences(You, DamageSourceHasColor(core.Red), binding)
		if err := ValidateCardDefinition("Choice", NewArtifact("Choice", "", WithAbility(Activate(Costs(), choice)))); err != nil {
			t.Fatal(err)
		}
		if err := choice.Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if !g.resolution.damageSourceBindings[binding].SameObject(g.CaptureDamageSource(source.ID())) {
			t.Fatal("wrong choice")
		}
	}
	if err := ChooseDamageSourceFromWaitingUntapSkipReferences(You, DamageSourceHasColor(core.Blue), first).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.resolution.damageSourceBindings[first].Valid() || !g.resolution.damageSourceBindings[second].Valid() {
		t.Fatal("binding isolation")
	}
	ctx.Controller = uuid.New()
	if err := ChooseDamageSourceFromWaitingUntapSkipReferences(You, DamageSourceFilter{}, second).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.resolution.damageSourceBindings[second].Valid() {
		t.Fatal("absent player retained choice")
	}
	for _, effect := range []ResolvingEffect{
		ChooseDamageSourceFromWaitingUntapSkipReferences(nil, DamageSourceFilter{}, first),
		ChooseDamageSourceFromWaitingUntapSkipReferences(You, DamageSourceFilter{}, nil),
		ChooseDamageSourceFromWaitingUntapSkipReferences(You, DamageSourceFilter{}, BindDamageSource("")),
		Sequence(DealDamageFrom(FromDamageSource(first), You, Fixed(1)), ChooseDamageSourceFromWaitingUntapSkipReferences(You, DamageSourceFilter{}, first)),
	} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))) == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}
