package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingPreventionPermanentSources(t *testing.T) {
	for _, kind := range []string{"finite", "unlimited", "source"} {
		for _, state := range []string{"live", "depart", "return", "phase", "cycle", "control", "cleanup", "consume"} {
			t.Run(kind+"/"+state, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				card := NewCreature("Subject", "{R}", 1, 20)
				p := g.PutOnBattlefield(card, a.PlayerID())
				saved := g.CaptureDamageSource(p.ID())
				var effect ResolvingEffect
				switch kind {
				case "finite":
					effect = PreventNextDamage(This, Fixed(2))
				case "unlimited":
					effect = PreventDamageThisTurn(This, PermanentFilter{})
				case "source":
					effect = PreventNextDamageFromPermanent(This, You, PermanentFilter{})
				}
				if err := ApplyEffect(g, effect, p.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				if got := g.WaitingPreventionPermanentSources(); len(got) != 1 || !got[0].SameObject(saved) {
					t.Fatal("missing installed reference")
				}
				switch state {
				case "depart", "return":
					g.RemoveFromBattlefield(p)
					if state == "return" {
						g.PutOnBattlefield(card, b.PlayerID())
					}
					g.zones.ClearLKI()
				case "phase", "cycle":
					ids := g.PhaseOut(p.ID())
					if state == "cycle" {
						g.PhaseIn(ids)
					}
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "cleanup":
					g.effects.ClearReplacementsEndOfTurn()
				case "consume":
					if kind == "source" {
						g.DealDamageToPlayerFromSource(b, 2, saved)
					} else {
						g.DealDamageToPermanent(p, 2, uuid.New())
					}
				}
				clone := g.Clone()
				for _, game := range []*Game{g, clone} {
					got := game.WaitingPreventionPermanentSources()
					want := 1
					if state == "cleanup" || (state == "consume" && kind != "unlimited") {
						want = 0
					}
					if len(got) != want {
						t.Fatalf("references=%d want=%d", len(got), want)
					}
					if want == 0 {
						continue
					}
					if !got[0].SameObject(saved) {
						t.Fatal("new incarnation replaced saved identity")
					}
					if state == "control" && got[0].Controller() != b.PlayerID() {
						t.Fatal("stale controller")
					}
				}
			})
		}
	}
}

func TestWaitingPreventionChoicesAndBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Subject", "{R}", 1, 20), a.PlayerID())
	for _, effect := range []Effect{PreventNextDamage(This, Fixed(2)), PreventDamageThisTurn(This, PermanentFilter{}), PreventNextDamageFromPermanent(This, You, PermanentFilter{})} {
		if err := ApplyEffect(g, effect, p.ID(), b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	g.AddPreventionShield(b.PlayerID(), 2)
	g.effects.AddReplacement(&sourcePreventionReplacement{dmgSource: uuid.New(), incarnation: uuid.New()})
	if got := g.WaitingPreventionPermanentSources(); len(got) != 1 {
		t.Fatal("invalid or duplicate references")
	}
	first, second := BindDamageSource("first"), BindDamageSource("second")
	for _, binding := range []*DamageSourceBinding{first, second} {
		effect := ChooseDamageSourceFromWaitingPreventionReferences(You, DamageSourceFilter{}, binding)
		card := NewArtifact("Choice", "", WithAbility(Activate(Costs(), effect)))
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		if err := effect.Apply(&EffectContext{Game: g, Controller: b.PlayerID()}); err != nil {
			t.Fatal(err)
		}
	}
	if !g.resolution.damageSourceBindings[first].SameObject(g.resolution.damageSourceBindings[second]) {
		t.Fatal("separate choices lost identity")
	}
	clone := g.Clone()
	g.effects.ClearReplacementsEndOfTurn()
	g.effects.replacements = nil
	if err := ChooseDamageSourceFromWaitingPreventionReferences(You, DamageSourceFilter{}, first).Apply(&EffectContext{Game: g, Controller: b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if _, ok := g.resolution.damageSourceBindings[first]; ok {
		t.Fatal("empty choice kept result")
	}
	if !g.resolution.damageSourceBindings[second].Valid() || len(clone.WaitingPreventionPermanentSources()) != 1 {
		t.Fatal("clearing changed independent state")
	}
	for _, effect := range []Effect{ChooseDamageSourceFromWaitingPreventionReferences(nil, DamageSourceFilter{}, first), ChooseDamageSourceFromWaitingPreventionReferences(You, DamageSourceFilter{}, nil)} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid declaration")
		}
	}
}

func TestWaitingPreventionInformationAndChoice(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Subject", "{R}", 1, 20), a.PlayerID())
	saved := g.CaptureDamageSource(p.ID())
	if err := ApplyEffect(g, PreventDamageThisTurn(This, PermanentFilter{}), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, Until(core.Indefinite, SetColors(This, core.Blue)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	got := g.WaitingPreventionPermanentSources()
	if len(got) != 1 || !got[0].HasColor(core.Blue) || got[0].HasColor(core.Red) {
		t.Fatal("query did not read current colors")
	}
	got[0].card.SetOwner(b.PlayerID())
	if g.WaitingPreventionPermanentSources()[0].card.Owner() == b.PlayerID() {
		t.Fatal("query shared mutable card information")
	}
	chosen := BindDamageSource("chosen")
	if err := ChooseDamageSourceFromWaitingPreventionReferences(You, DamageSourceHasColor(core.Red), chosen).Apply(&EffectContext{Game: g, Controller: b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if _, ok := g.resolution.damageSourceBindings[chosen]; ok {
		t.Fatal("filter used installation color")
	}
	g.PhaseOut(p.ID())
	if got = g.WaitingPreventionPermanentSources(); len(got) != 1 || !got[0].HasColor(core.Blue) {
		t.Fatal("phasing lost information")
	}
	g.PhaseIn([]uuid.UUID{p.ID()})
	g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
	g.zones.ClearLKI()
	if got = g.WaitingPreventionPermanentSources(); len(got) != 1 || !got[0].SameObject(saved) || !got[0].HasColor(core.Blue) {
		t.Fatal("departure lost exact information")
	}
	if err := ChooseDamageSourceFromWaitingPreventionReferences(You, DamageSourceHasColor(core.Blue), chosen).Apply(&EffectContext{Game: g, Controller: b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if !g.resolution.damageSourceBindings[chosen].SameObject(saved) {
		t.Fatal("departed reference could not be chosen")
	}
	if err := ApplyEffect(g, ChooseDamageSourceFromWaitingPreventionReferences(You, DamageSourceFilter{}, chosen), uuid.Nil, uuid.New(), nil); err != nil {
		t.Fatal(err)
	}
}
