package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestBaseToughnessLayersAndIdentity(t *testing.T) {
	for _, change := range []string{"none", "phase", "return", "clone", "overwrite", "switch"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Subject", "", 4, 5)
			p := g.PutOnBattlefield(card, a.PlayerID())
			apply := func(mod ContinuousModification) {
				t.Helper()
				if err := ApplyEffect(g, Until(core.Indefinite, mod), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			apply(SetBasePowerToughness(This, Fixed(7), Fixed(8)))
			apply(SetBaseToughness(This, Fixed(2)))
			apply(ModifyPT(This, Fixed(3), Fixed(4)))
			g.MutablePermanent(p.ID()).AddCounter(core.P1P1, 1)
			check := func(game *Game, power, toughness int) {
				t.Helper()
				game.ApplyContinuousEffects()
				q := game.FindPermanent(p.ID())
				if q.CurrentPower(game) != power || q.CurrentToughness(game) != toughness {
					t.Fatalf("got %d/%d want %d/%d", q.CurrentPower(game), q.CurrentToughness(game), power, toughness)
				}
			}
			check(g, 11, 7)
			switch change {
			case "phase":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				g.PutOnBattlefield(card, a.PlayerID())
				check(g, 4, 5)
				return
			case "clone":
				clone := g.Clone()
				if err := ApplyEffect(clone, Until(core.Indefinite, SetBaseToughness(This, Fixed(9))), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				check(clone, 11, 14)
			case "overwrite":
				apply(SetBasePower(This, Fixed(-1)))
				check(g, 3, 7)
				return
			case "switch":
				apply(SwitchPowerToughness(This))
				check(g, 7, 11)
				return
			}
			check(g, 11, 7)
		})
	}
}

func TestBaseToughnessGraveyardCountCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.AddToGraveyard(NewCreature("One", "", 1, 1))
	a.AddToGraveyard(NewArtifact("Not creature", ""))
	b.AddToGraveyard(NewCreature("Opponent", "", 1, 1))
	p := g.PutOnBattlefield(NewCreature("Subject", "", 4, 5), a.PlayerID())
	body := Until(core.Indefinite, SetBaseToughness(This, AddValues(Fixed(1), CountGraveyardCards(You, IsCreatureCard))))
	if err := ValidateCardDefinition("Subject", NewCreature("Subject", "", 4, 5, WithAbility(Activate(Costs(), body)))); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, body, p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	a.AddToGraveyard(NewCreature("Later", "", 1, 1))
	g.ApplyContinuousEffects()
	if p = g.FindPermanent(p.ID()); p.CurrentPower(g) != 4 || p.CurrentToughness(g) != 2 {
		t.Fatal("count was not captured")
	}
	for _, bad := range []ContinuousModification{SetBaseToughness(nil, Fixed(1)), SetBaseToughness(This, nil), SetBaseToughness(This, CountGraveyardCards(nil, IsCreatureCard))} {
		if err := ValidateCardDefinition("Bad", NewCreature("Bad", "", 1, 1, WithAbility(Activate(Costs(), Until(core.Indefinite, bad))))); err == nil {
			t.Fatal("accepted invalid declaration")
		}
	}
}

func TestBaseToughnessCDAAndStaticLayers(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Subject", "", 4, 5), a.PlayerID())
	power := 9
	g.MutablePermanent(p.ID()).cdaPower = &power
	mod := SetBaseToughness(This, Fixed(-2)).modification()
	ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID()}
	captured := mod.capture(ctx)
	if len(captured) != 1 {
		t.Fatal("missing captured subject")
	}
	captured[0].apply(g, core.LayerPT)
	if q := g.FindPermanent(p.ID()); q.BasePTOverride[0] != 9 || q.BasePTOverride[1] != -2 {
		t.Fatal("base toughness discarded CDA power")
	}
	g.MutablePermanent(p.ID()).cdaPower = nil
	source := g.PutOnBattlefield(NewCreature("Static source", "", 4, 5, WithAbility(Static(While(Controls(IsCreature), SetBaseToughness(This, Fixed(7)))))), b.PlayerID())
	if err := ValidateCardDefinition("Static source", source.Card); err != nil {
		t.Fatal(err)
	}
	if source.CurrentPower(g) != 4 || source.CurrentToughness(g) != 7 {
		t.Fatal("static layer did not set toughness")
	}
	g.PhaseOut(source.ID())
	g.PhaseIn([]uuid.UUID{source.ID()})
	if q := g.FindPermanent(source.ID()); q.CurrentToughness(g) != 7 {
		t.Fatal("static source phase cycle lost layer")
	}
}

func TestGraveyardCountAndAdditionQueries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	creature := NewCreature("Creature", "", 1, 1)
	a.AddToGraveyard(creature)
	a.AddToGraveyard(NewArtifact("Artifact", ""))
	b.AddToGraveyard(NewCreature("Other", "", 1, 1))
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	for range 2 {
		if CountGraveyardCards(You, IsCreatureCard).resolveValue(ctx) != 1 || CountGraveyardCards(You, CardFilter{}).resolveValue(ctx) != 2 || AddValues(Fixed(-3), Fixed(2)).resolveValue(ctx) != -1 {
			t.Fatal("wrong query value")
		}
	}
	clone := g.Clone()
	clone.GetPlayer(a.PlayerID()).RemoveFromGraveyard(creature.ID())
	if CountGraveyardCards(You, IsCreatureCard).resolveValue(&EffectContext{Game: clone, Controller: a.PlayerID()}) != 0 || CountGraveyardCards(You, IsCreatureCard).resolveValue(ctx) != 1 {
		t.Fatal("query crossed clone boundary")
	}
	for _, value := range []ResolutionValue{AddValues(nil, Fixed(1)), AddValues(Fixed(1), nil)} {
		if err := ValidateCardDefinition("Bad", NewInstant("Bad", "", WithAbility(Spell(GainLifeFor(You, value))))); err == nil {
			t.Fatal("accepted nil operand")
		}
	}
}
