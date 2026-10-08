package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingCounterPlacementReferences(t *testing.T) {
	for _, state := range []string{"live", "phase", "cycle", "depart", "return", "control", "cleanup"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewArtifact("Rule", "{W}")
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			g.AddETBAdditionalCounters(p.ID(), core.P1P1, 2, PermanentFilter{}, true)
			rule := g.effects.replacements[len(g.effects.replacements)-1]
			clone := g.Clone()
			want := 1
			switch state {
			case "phase":
				g.PhaseOut(p.ID())
				want = 0
			case "cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "depart", "return":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				g.zones.ClearLKI()
				want = 0
				if state == "return" {
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "cleanup":
				g.effects.ClearReplacementsEndOfTurn()
			}
			g.effects.replacements = []ReplacementEffect{rule}
			target := g.PutOnBattlefield(NewCreature("Target", "", 1, 10), a.PlayerID())
			action := NewAddCountersAction(p.ID(), target.ID(), core.P1P1, 1, true)
			if rule.Matches(action, g) != (want == 1) {
				t.Fatal("match did not check identity")
			}
			if rule.Replace(action, g).(*AddCountersAction).Amount() != 1+2*want {
				t.Fatal("replace did not recheck identity")
			}
			if rule.IsActive(g) != (want == 1) {
				t.Fatal("replacement lifetime")
			}
			for _, game := range []*Game{g, g.Clone()} {
				got := game.WaitingCounterPlacementPermanentSources()
				if len(got) != want {
					t.Fatalf("references=%d want=%d", len(got), want)
				}
				if want == 1 && !got[0].SameObject(saved) {
					t.Fatal("wrong incarnation")
				}
				if want == 1 && state == "control" && got[0].Controller() != b.PlayerID() {
					t.Fatal("stale controller")
				}
				if want == 1 {
					got[0].card.SetOwner(b.PlayerID())
					if game.WaitingCounterPlacementPermanentSources()[0].card.Owner() != a.PlayerID() {
						t.Fatal("query shared card data")
					}
				}
				result := game.effects.ApplyReplacements(action, game).(*AddCountersAction)
				expected := 1
				if want == 1 {
					expected = 3
				}
				if result.Amount() != expected {
					t.Fatal("query and execution disagree")
				}
			}
			if len(clone.WaitingCounterPlacementPermanentSources()) != 1 {
				t.Fatal("clone lost rule")
			}
			if state == "return" {
				g.AddETBAdditionalCounters(p.ID(), core.P1P1, 2, PermanentFilter{}, true)
				refs := g.WaitingCounterPlacementPermanentSources()
				if len(refs) != 1 || refs[0].SameObject(saved) {
					t.Fatal("new registration did not bind returned source")
				}
			}
		})
	}
}

func TestWaitingCounterPlacementChoices(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewArtifact("Rule", "{W}"), a.PlayerID())
	g.AddETBAdditionalCounters(p.ID(), core.P1P1, 2, PermanentFilter{}, false)
	g.AddCounterDoubler(p.ID(), core.P1P1, PermanentFilter{})
	if len(g.WaitingCounterPlacementPermanentSources()) != 0 {
		t.Fatal("generic filters fabricated references")
	}
	g.AddETBAdditionalCounters(p.ID(), core.P1P1, 2, PermanentFilter{}, true)
	g.AddETBAdditionalCounters(p.ID(), core.P1P1, 2, PermanentFilter{}, true)
	g.AddETBAdditionalCounters(uuid.New(), core.P1P1, 2, PermanentFilter{}, true)
	g.effects.AddCycleReplacement(g.effects.replacements[2].Clone())
	if len(g.WaitingCounterPlacementPermanentSources()) != 1 {
		t.Fatal("invalid or duplicate reference")
	}
	first, second := BindDamageSource("first"), BindDamageSource("second")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	for _, binding := range []*DamageSourceBinding{first, second} {
		choice := ChooseDamageSourceFromWaitingCounterPlacementReferences(You, DamageSourceHasColor(core.White), binding)
		if err := ValidateCardDefinition("Choice", NewArtifact("Choice", "", WithAbility(Activate(Costs(), choice)))); err != nil {
			t.Fatal(err)
		}
		if err := choice.Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if !g.resolution.damageSourceBindings[binding].Valid() {
			t.Fatal("missing choice")
		}
	}
	choice := ChooseDamageSourceFromWaitingCounterPlacementReferences(You, DamageSourceHasColor(core.Blue), first)
	if err := choice.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.resolution.damageSourceBindings[first].Valid() || !g.resolution.damageSourceBindings[second].Valid() {
		t.Fatal("bindings not independent")
	}
	invalid := ChooseDamageSourceFromWaitingCounterPlacementReferences(You, DamageSourceFilter{}, nil)
	if err := ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), invalid)))); err == nil {
		t.Fatal("nil binding accepted")
	}
	self := NewAddCountersAction(p.ID(), p.ID(), core.P1P1, 1, true)
	rule := g.effects.replacements[2]
	if rule.Matches(self, g) {
		t.Fatal("source exclusion lost")
	}
	if rule.Matches(NewAddCountersAction(p.ID(), p.ID(), core.P1P1, 1, false), g) {
		t.Fatal("ordinary counters matched")
	}
}
