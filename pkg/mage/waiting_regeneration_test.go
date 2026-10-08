package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingRegenerationReferences(t *testing.T) {
	for _, state := range []string{"live", "depart", "return", "phase", "cycle", "control", "consume", "cleanup", "cant-regenerate"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Subject", "{R}", 1, 20)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			g.AddRegenerationShield(p.ID())
			switch state {
			case "depart", "return":
				g.RemoveFromBattlefield(p)
				if state == "return" {
					g.PutOnBattlefield(card, b.PlayerID())
				}
				g.zones.ClearLKI()
			case "phase":
				g.PhaseOut(p.ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "consume":
				g.DestroyPermanent(p)
			case "cleanup":
				g.effects.ClearReplacementsEndOfTurn()
			case "cant-regenerate":
				if err := ApplyEffect(g, Until(core.Indefinite, GrantKeywordModification(This, core.CantRegenerate)), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			clone := g.Clone()
			for _, game := range []*Game{g, clone} {
				got := game.WaitingRegenerationPermanentSources()
				want := 1
				if state == "consume" || state == "cleanup" {
					want = 0
				}
				if len(got) != want {
					t.Fatalf("sources=%d want=%d", len(got), want)
				}
				if want == 0 {
					continue
				}
				if !got[0].SameObject(saved) {
					t.Fatal("saved incarnation changed")
				}
				if state == "control" && got[0].Controller() != b.PlayerID() {
					t.Fatal("stale controller")
				}
				got[0].card.SetOwner(b.PlayerID())
				if game.WaitingRegenerationPermanentSources()[0].card.Owner() != a.PlayerID() {
					t.Fatal("shared query data")
				}
			}
			g.effects.ClearReplacementsEndOfTurn()
			if state != "consume" && state != "cleanup" && len(clone.WaitingRegenerationPermanentSources()) != 1 {
				t.Fatal("clone lifetime changed")
			}
		})
	}
}

func TestWaitingRegenerationChoices(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Subject", "{R}", 1, 20), a.PlayerID())
	g.AddRegenerationShield(p.ID())
	g.AddRegenerationShield(p.ID())
	g.effects.cycleReplacements = append(g.effects.cycleReplacements, g.effects.replacements[0].Clone())
	g.effects.AddReplacement(&regenerationReplacement{permanentID: p.ID(), shields: 1})
	g.effects.AddReplacement(&regenerationReplacement{permanentID: uuid.New(), incarnation: uuid.New(), shields: 1})
	g.AddPreventionShield(b.PlayerID(), 2)
	if len(g.WaitingRegenerationPermanentSources()) != 1 {
		t.Fatal("invalid or duplicate references")
	}
	first, second := BindDamageSource("first"), BindDamageSource("second")
	ctx := &EffectContext{Game: g, Controller: b.PlayerID()}
	for _, binding := range []*DamageSourceBinding{first, second} {
		choice := ChooseDamageSourceFromWaitingRegenerationReferences(You, DamageSourceHasColor(core.Red), binding)
		card := NewArtifact("Choice", "", WithAbility(Activate(Costs(), choice)))
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		if err := choice.Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if !g.resolution.damageSourceBindings[binding].SameObject(g.CaptureDamageSource(p.ID())) {
			t.Fatal("choice identity")
		}
	}
	if err := ApplyEffect(g, Until(core.Indefinite, SetColors(This, core.Blue)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if err := ChooseDamageSourceFromWaitingRegenerationReferences(You, DamageSourceHasColor(core.Red), first).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := g.resolution.damageSourceBindings[first]; ok {
		t.Fatal("stale choice retained")
	}
	if !g.resolution.damageSourceBindings[second].Valid() {
		t.Fatal("independent result cleared")
	}
	g.effects.cycleReplacements = nil
	g.DestroyPermanent(g.FindPermanent(p.ID()))
	if len(g.WaitingRegenerationPermanentSources()) != 1 {
		t.Fatal("first destruction consumed both shields")
	}
	g.DestroyPermanent(g.FindPermanent(p.ID()))
	if len(g.WaitingRegenerationPermanentSources()) != 0 {
		t.Fatal("consumed shield still supplied source")
	}
	for _, choice := range []ResolvingEffect{
		ChooseDamageSourceFromWaitingRegenerationReferences(nil, DamageSourceFilter{}, first),
		ChooseDamageSourceFromWaitingRegenerationReferences(You, DamageSourceFilter{}, nil),
		ChooseDamageSourceFromWaitingRegenerationReferences(You, DamageSourceHasColor(core.Colorless), first),
		Sequence(DealDamageFrom(FromDamageSource(first), You, Fixed(1)), ChooseDamageSourceFromWaitingRegenerationReferences(You, DamageSourceFilter{}, first)),
	} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), choice)))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestWaitingRegenerationSeparateSubjects(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Subject", "{R}", 1, 20)
	first := g.PutOnBattlefield(card, a.PlayerID())
	old := g.CaptureDamageSource(first.ID())
	g.AddRegenerationShield(first.ID())
	if err := ApplyEffect(g, Until(core.Indefinite, SetColors(This, core.Blue)), first.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
	second := g.PutOnBattlefield(card, b.PlayerID())
	current := g.CaptureDamageSource(second.ID())
	g.AddRegenerationShield(second.ID())
	g.zones.ClearLKI()
	refs := g.WaitingRegenerationPermanentSources()
	if len(refs) != 2 || !refs[0].SameObject(old) || !refs[0].HasColor(core.Blue) || !refs[1].SameObject(current) || !refs[1].HasColor(core.Red) {
		t.Fatal("separate incarnations or retained properties lost")
	}
	binding := BindDamageSource("blue subject")
	if err := ChooseDamageSourceFromWaitingRegenerationReferences(You, DamageSourceHasColor(core.Blue), binding).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if !g.resolution.damageSourceBindings[binding].SameObject(old) {
		t.Fatal("filter selected returned object")
	}
	clone := g.Clone()
	g.DestroyPermanents([]*Permanent{g.FindPermanent(second.ID())})
	refs = g.WaitingRegenerationPermanentSources()
	if len(refs) != 1 || !refs[0].SameObject(old) || len(clone.WaitingRegenerationPermanentSources()) != 2 {
		t.Fatal("consumption changed another subject or clone")
	}
	if g.FindPermanent(second.ID()) == nil || !g.FindPermanent(second.ID()).Tapped {
		t.Fatal("regeneration execution changed")
	}
}
