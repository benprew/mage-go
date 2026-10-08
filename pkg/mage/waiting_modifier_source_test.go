package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingModifierSourcesLifecycle(t *testing.T) {
	for _, kind := range []string{"mill", "life", "static-mill", "static-life"} {
		for _, state := range []string{"live", "phase", "cycle", "depart", "return", "control", "remove-ability"} {
			t.Run(kind+"/"+state, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				calls := 0
				mill := func(*Game, uuid.UUID, int) int { calls++; return 7 }
				life := func(*Game, uuid.UUID, int) int { calls++; return 7 }
				var options []CardOption
				if kind == "static-mill" {
					options = append(options, WithMillReplacement(func(uuid.UUID, uuid.UUID) MillModifierFn { calls++; return mill }))
				}
				if kind == "static-life" {
					options = append(options, WithLifeGainReplacement(func(uuid.UUID, uuid.UUID) LifeGainModifierFn { calls++; return life }))
				}
				card := NewArtifact("Modifier", "", options...)
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				if kind == "mill" {
					g.AddMillReplacement(p.ID(), mill)
				}
				if kind == "life" {
					g.AddLifeGainModifier(p.ID(), life)
				}
				saved := g.CaptureDamageSource(p.ID())
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
						if len(options) > 0 {
							want = 1
						}
					}
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "remove-ability":
					g.AddContinuousEffect(RemoveAllAbilities(p.ID(), core.EndOfTurn))
					if len(options) > 0 {
						want = 0
					}
				}
				refs := g.WaitingModifierPermanentSources()
				if len(refs) != want {
					t.Fatalf("sources=%d want=%d", len(refs), want)
				}
				if calls != 0 {
					t.Fatal("query called a modifier or factory")
				}
				if want == 1 {
					if refs[0].SameObject(saved) != (state != "return") {
						t.Fatal("wrong incarnation")
					}
					if state == "control" && refs[0].Controller() != b.PlayerID() {
						t.Fatal("stale controller")
					}
					refs[0].card.SetOwner(b.PlayerID())
					if g.WaitingModifierPermanentSources()[0].card.Owner() != a.PlayerID() {
						t.Fatal("query changed source")
					}
				}
				if refs := clone.WaitingModifierPermanentSources(); len(refs) != 1 || !refs[0].SameObject(saved) {
					t.Fatal("clone lost independent registration")
				}
				if got := clone.ApplyMillModifiers(b.PlayerID(), 1); (kind == "mill" || kind == "static-mill") && got != 7 {
					t.Fatal("mill execution changed")
				}
				if got := clone.ApplyLifeGainModifiers(b.PlayerID(), 1); (kind == "life" || kind == "static-life") && got != 7 {
					t.Fatal("life execution changed")
				}
				if state == "return" && len(options) == 0 {
					returned := g.FindPermanent(p.ID())
					if kind == "mill" {
						g.AddMillReplacement(returned.ID(), mill)
					} else {
						g.AddLifeGainModifier(returned.ID(), life)
					}
					refs = g.WaitingModifierPermanentSources()
					if len(refs) != 1 || refs[0].SameObject(saved) {
						t.Fatal("fresh registration lost returned identity")
					}
				}
			})
		}
	}
}

func TestWaitingModifierSourcesChoices(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewArtifact("Modifier", "{R}", WithMillReplacement(DoubleMillForOpponents), WithLifeGainReplacement(DoubleLifeGainForController)), a.PlayerID())
	g.AddMillReplacement(p.ID(), DoubleMillForOpponents(p.ID(), a.PlayerID()))
	g.AddLifeGainModifier(p.ID(), DoubleLifeGainForController(p.ID(), a.PlayerID()))
	g.AddLifeGainModifier(uuid.New(), func(*Game, uuid.UUID, int) int { t.Fatal("query ran callback"); return 0 })
	g.millModifiers = append(g.millModifiers, millModifierEntry{sourceID: p.ID(), sourceIncarnation: uuid.New(), fn: DoubleMillForOpponents(p.ID(), a.PlayerID())})
	g.lifeGainModifiers = append(g.lifeGainModifiers, lifeGainModifierEntry{sourceID: p.ID(), fn: DoubleLifeGainForController(p.ID(), a.PlayerID())})
	if len(g.WaitingModifierPermanentSources()) != 1 {
		t.Fatal("duplicates or malformed references included")
	}
	if len(g.millModifiers) != 2 || len(g.lifeGainModifiers) != 3 {
		t.Fatal("query changed registration storage")
	}
	first, second := BindDamageSource("first"), BindDamageSource("second")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	for _, binding := range []*DamageSourceBinding{first, second} {
		if err := ChooseDamageSourceFromWaitingModifierReferences(You, DamageSourceHasType(core.TypeArtifact), binding).Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if !g.resolution.damageSourceBindings[binding].SameObject(g.CaptureDamageSource(p.ID())) {
			t.Fatal("choice identity")
		}
	}
	if err := ChooseDamageSourceFromWaitingModifierReferences(You, DamageSourceHasColor(core.Blue), first).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.resolution.damageSourceBindings[first].Valid() || !g.resolution.damageSourceBindings[second].Valid() {
		t.Fatal("result isolation")
	}
	if err := ChooseDamageSourceFromWaitingModifierReferences(ControllerOf(From(BindPermanent("absent"))), DamageSourceFilter{}, second).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.resolution.damageSourceBindings[second].Valid() {
		t.Fatal("absent player kept result")
	}
	for _, effect := range []ResolvingEffect{
		ChooseDamageSourceFromWaitingModifierReferences(nil, DamageSourceFilter{}, first),
		ChooseDamageSourceFromWaitingModifierReferences(You, DamageSourceFilter{}, nil),
		ChooseDamageSourceFromWaitingModifierReferences(You, DamageSourceFilter{}, BindDamageSource("")),
		Sequence(DealDamageFrom(FromDamageSource(first), You, Fixed(1)), ChooseDamageSourceFromWaitingModifierReferences(You, DamageSourceFilter{}, first)),
	} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))) == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}
