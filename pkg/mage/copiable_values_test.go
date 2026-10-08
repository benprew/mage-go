package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func copiableTestCard(t *testing.T) Card {
	t.Helper()
	plan, err := NewCardDefinitionPlan(func() Card {
		target := DeclarePermanentTarget("victim", IsCreature)
		return NewCreature("Snapshot fixture", "{1}{U}", 1, 3,
			WithSuperTypes(core.SuperLegendary), WithSubTypes("Bird"), WithKeyword(core.Flying),
			WithAbility(Activate(Costs(Tap()), TapSubject(target), WithTargets(target))))
	})
	if err != nil {
		t.Fatal(err)
	}
	card, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	return card
}

func TestCopiableValuesIndependence(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := copiableTestCard(t)
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	p.Tapped = true
	p.ChosenPlayer = b.PlayerID()
	p.AddCounter(core.P1P1, 2)
	p.BasePTOverride = &[2]int{8, 8}
	p.ColorOverride = &[]core.Color{core.Red}
	p.SubTypeOverride = []string{"Dragon"}
	p.RuntimeAbilities = nil
	p.RevokeBaseAttr(core.Flying)
	values, err := g.CaptureCopiableValues(p)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		fresh, err := values.Build()
		if err != nil {
			t.Fatal(err)
		}
		if fresh.Name() != card.Name() || fresh.Power() != 1 || fresh.Toughness() != 3 || fresh.ManaCost().String() != "{1}{U}" || !fresh.HasSubType("Bird") || !fresh.HasSuperType(core.SuperLegendary) || fresh.AttrSeeds()[core.Flying] == 0 {
			t.Fatal("snapshot acquired noncopy changes")
		}
		if fresh.Owner() != uuid.Nil || fresh.IsToken() || fresh.Abilities()[0].Source() != uuid.Nil {
			t.Fatal("snapshot kept instance state")
		}
		if err := independentCardDefinitions(card, fresh); err != nil {
			t.Fatal(err)
		}
		fresh.(*BaseCard).types[0] = core.TypeArtifact
		fresh.(*BaseCard).attrSeeds[core.Flying] = 0
	}
	g.BouncePermanentToHand(p)
	a.RemoveFromHand(card.ID())
	returned := g.PutOnBattlefield(card, b.PlayerID())
	if _, err := g.CaptureCopiableValues(p); err == nil {
		t.Fatal("capture followed a returned source")
	}
	if _, err := values.Build(); err != nil {
		t.Fatal("departure invalidated saved values", err)
	}
	g.PhaseOut(returned.ID())
	if _, err := g.CaptureCopiableValues(returned); err == nil {
		t.Fatal("captured an absent phased object")
	}
	g.PhaseIn([]uuid.UUID{returned.ID()})
	if _, err := g.CaptureCopiableValues(returned); err != nil {
		t.Fatal("phase cycle lost provenance", err)
	}
}

func TestCopiableValuesFaceDownAndCopies(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(copiableTestCard(t), a.PlayerID())
	faceUp, err := g.CaptureCopiableValues(p)
	if err != nil {
		t.Fatal(err)
	}
	p.FaceDown = true
	hidden, err := g.CaptureCopiableValues(p)
	if err != nil {
		t.Fatal(err)
	}
	p.FaceDown = false
	for _, values := range []*CopiableValues{faceUp, hidden} {
		first, err := values.Build()
		if err != nil {
			t.Fatal(err)
		}
		copied := g.PutOnBattlefield(first, b.PlayerID())
		if copied.FaceDown {
			t.Fatal("copy acquired face-down status")
		}
		again, err := g.CaptureCopiableValues(copied)
		if err != nil {
			t.Fatal(err)
		}
		second, err := again.Build()
		if err != nil {
			t.Fatal(err)
		}
		if second.Name() != first.Name() || second.Power() != first.Power() || second.Toughness() != first.Toughness() {
			t.Fatal("copy of a copy lost provenance")
		}
		if values == hidden && (second.Name() != "" || second.Power() != 2 || second.Toughness() != 2 || second.ManaCost().CMC() != 0 || second.ManaCost().HasX || len(second.Abilities()) != 0 || len(second.SubTypes()) != 0 || len(second.SuperTypes()) != 0 || len(second.AttrSeeds()) != 0 || len(second.ManaCost().Colors()) != 0 || !second.HasType(core.TypeCreature)) {
			t.Fatal("face-down snapshot kept hidden characteristics")
		}
		clone := g.Clone()
		cloned, err := clone.CaptureCopiableValues(copied)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := cloned.Build(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCopiableValuesInvalid(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for _, source := range []*Permanent{nil, new(Permanent), NewPermanent(copiableTestCard(t), a.PlayerID()), g.PutOnBattlefield(NewCreature("Legacy", "", 1, 1), a.PlayerID())} {
		if _, err := g.CaptureCopiableValues(source); err == nil {
			t.Fatal("invalid source accepted")
		}
	}
	var absent *Game
	if _, err := absent.CaptureCopiableValues(NewPermanent(copiableTestCard(t), a.PlayerID())); err == nil {
		t.Fatal("capture used an absent game")
	}
	var values *CopiableValues
	if _, err := values.Build(); err == nil {
		t.Fatal("nil snapshot built")
	}
	if _, err := new(CopiableValues).Build(); err == nil {
		t.Fatal("uninitialized snapshot built")
	}
	p := g.PutOnBattlefield(copiableTestCard(t), a.PlayerID())
	p.Card.CloneFrom(NewCreature("Legacy replacement", "", 7, 7))
	if _, err := g.CaptureCopiableValues(p); err == nil {
		t.Fatal("legacy replacement retained provenance")
	}
	p = g.PutOnBattlefield(copiableTestCard(t), a.PlayerID())
	p.Card.SetBasePT(9, 9)
	if _, err := g.CaptureCopiableValues(p); err == nil {
		t.Fatal("untracked characteristic changes accepted")
	}
	p = g.PutOnBattlefield(copiableTestCard(t), a.PlayerID())
	p.Card.AddAbility(NewAction(ActionActivated, WithEffects(TapSubject(This))))
	if _, err := g.CaptureCopiableValues(p); err == nil {
		t.Fatal("untracked rules changes accepted")
	}
	p = g.PutOnBattlefield(copiableTestCard(t), a.PlayerID())
	g.GetEffects().AddCopyEffect(p.ID(), g.PutOnBattlefield(NewCreature("Legacy copy source", "", 6, 6), b.PlayerID()))
	if _, err := g.CaptureCopiableValues(p); err == nil || !strings.Contains(err.Error(), "copy layer") {
		t.Fatalf("legacy copy layer: %v", err)
	}
}

func TestCopiableValuesCostPresenceAndTokens(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for _, cost := range []string{"", "{0}", "{W/U}"} {
		plan, err := NewCardDefinitionPlan(func() Card {
			return NewCreature("Cost fixture", cost, 2, 2)
		})
		if err != nil {
			t.Fatal(err)
		}
		card, err := plan.Build()
		if err != nil {
			t.Fatal(err)
		}
		p := g.PutOnBattlefield(card, a.PlayerID())
		values, err := g.CaptureCopiableValues(p)
		if err != nil || values.HasManaCost() != (cost != "") {
			t.Fatal("snapshot lost mana cost presence", err)
		}
		if cost == "{W/U}" {
			card.Copy().ManaCost().Hybrid[0].A = core.Green
			if _, err := g.CaptureCopiableValues(p); err != nil {
				t.Fatal("ordinary copy shares mana cost data", err)
			}
		}
		p.FaceDown = true
		hidden, err := g.CaptureCopiableValues(p)
		if err != nil || hidden.HasManaCost() {
			t.Fatal("face-down values gained a zero mana cost", err)
		}
		fresh, err := hidden.Build()
		if err != nil || fresh.(*BaseCard).HasManaCost() {
			t.Fatal("face-down prototype gained a zero mana cost", err)
		}
	}
	plan, err := NewCardDefinitionPlan(func() Card {
		return NewToken("Token fixture", 1, 1, []core.CardType{core.TypeCreature}, []string{"Soldier"})
	})
	if err != nil {
		t.Fatal(err)
	}
	card, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	values, err := g.CaptureCopiableValues(g.PutOnBattlefield(card, a.PlayerID()))
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := values.Build()
	if err != nil || fresh.IsToken() {
		t.Fatal("token status became copiable", err)
	}
}

func TestCopiableValuesBuilderDrift(t *testing.T) {
	for _, later := range []bool{false, true} {
		t.Run(map[bool]string{false: "validation", true: "build"}[later], func(t *testing.T) {
			calls := 0
			plan, err := NewCardDefinitionPlan(func() Card {
				calls++
				power := 1
				if (!later && calls == 2) || (later && calls > 2) {
					power = 9
				}
				return NewCreature("Drifting fixture", "{G}", power, 1)
			})
			if !later {
				if err == nil {
					t.Fatal("inconsistent printed characteristics accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := plan.CopiableValues().Build(); err == nil {
				t.Fatal("snapshot builder drift accepted")
			}
		})
	}
}

func TestCopiableValuesReadOnlyAndColor(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	calls := 0
	plan, err := NewCardDefinitionPlan(func() Card {
		calls++
		card := NewCreature("Color fixture", "{G}", 1, 1)
		card.SetColorOverride([]core.Color{core.Blue})
		return card
	})
	if err != nil {
		t.Fatal(err)
	}
	card, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	p := g.PutOnBattlefield(card, a.PlayerID())
	before := calls
	values, err := g.CaptureCopiableValues(p)
	if err != nil || calls != before {
		t.Fatal("capture executed the definition builder", err)
	}
	ordinary := card.Copy().(*BaseCard)
	ordinary.colorOverride[0] = core.Red
	if _, err := g.CaptureCopiableValues(p); err != nil {
		t.Fatal("ordinary copy shares color data", err)
	}
	g.AddControlEffect(ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	if _, err := g.CaptureCopiableValues(p); err != nil || calls != before {
		t.Fatal("control changed provenance or capture executed rules", err)
	}
	clone := g.Clone()
	clone.MutablePermanent(p.ID()).FaceDown = true
	hidden, err := clone.CaptureCopiableValues(p)
	if err != nil || hidden.HasManaCost() || calls != before {
		t.Fatal("clone capture used live game state or executed rules", err)
	}
	fresh, err := values.Build()
	if err != nil {
		t.Fatal(err)
	}
	copied := NewPermanent(fresh, b.PlayerID())
	if colors := copied.Colors(); len(colors) != 1 || colors[0] != core.Blue || calls != before+1 {
		t.Fatal("snapshot lost color indicator or rebuilt more than once")
	}
}
