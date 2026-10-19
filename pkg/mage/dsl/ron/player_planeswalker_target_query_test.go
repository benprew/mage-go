package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONPlayerPlaneswalkerTargetQuery(t *testing.T) {
	for _, domain := range []string{"AnyPlayer", "Opponent"} {
		for _, state := range []string{"player", "walker", "phase", "return", "copy", "clone", "shield"} {
			t.Run(domain+"/"+state, func(t *testing.T) {
				factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPlayerOrPermanent(`+domain+`,IsCardtype(Planeswalker))],ActionList([SpellDealsMultipleDamage(ThisSpell,[DamageToRecipients(Integer(4),Ref_TargetPlayerOrPermanent),DamageToRecipients(Integer(2),Player(You))])])))`)
				g, a, b := ronGame()
				walker := g.PutOnBattlefield(mage.NewPlaneswalker("Walker", "", 10), b.PlayerID())
				id := walker.ID()
				if state == "player" || state == "copy" {
					id = b.PlayerID()
				}
				obj := castRONSpell(t, g, a, factory(), id)
				switch state {
				case "phase":
					g.PhaseOut(walker.ID())
				case "return":
					g.RemoveFromBattlefield(walker)
					g.PutOnBattlefield(walker.Card, b.PlayerID())
				case "copy":
					obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
				case "shield":
					g.AddPreventionShield(id, 2)
				}
				for range 2 {
					if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
						t.Fatal("lowered union packet tree is unknown or proves destruction")
					}
				}
				if a.Life() != 20 || b.Life() != 20 || walker.Counters[core.Loyalty] != 10 {
					t.Fatal("query applied damage")
				}
				if state == "clone" {
					clone := g.Clone()
					if certain, known := clone.InspectSpellDestruction(clone.StackPeek(), nil); certain || !known {
						t.Fatal("clone query lost declaration")
					}
					clone.ResolveTopOfStack()
					if clone.GetPlayer(a.PlayerID()).Life() != 18 || clone.FindPermanent(id).Counters[core.Loyalty] != 6 || a.Life() != 20 || walker.Counters[core.Loyalty] != 10 {
						t.Fatal("clone crossed state")
					}
				}
				g.ResolveTopOfStack()
				wantA, wantB, wantLoyalty := 18, 20, 6
				switch state {
				case "phase", "return":
					wantA, wantLoyalty = 20, 10
				case "player":
					wantB, wantLoyalty = 16, 10
				case "copy":
					wantA, wantB, wantLoyalty = 20, 14, 10
					if domain == "Opponent" {
						wantB = 20
					}
				case "shield":
					wantLoyalty = 8
				}
				if a.Life() != wantA || b.Life() != wantB {
					t.Fatalf("life=%d/%d want=%d/%d", a.Life(), b.Life(), wantA, wantB)
				}
				current := g.FindPermanentIncludingPhased(walker.ID())
				if current == nil || current.Counters[core.Loyalty] != wantLoyalty {
					t.Fatal("wrong planeswalker damage")
				}
			})
		}
	}
}

func TestRONPlayerPlaneswalkerTargetQueryRealCard(t *testing.T) {
	for _, state := range []string{"player", "walker", "phase", "cycle", "depart-source", "phase-source", "return-source", "control-source", "clone"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, "DRK", "Witch Hunter"), a.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			walker := g.PutOnBattlefield(mage.NewPlaneswalker("Walker", "", 10), b.PlayerID())
			id := walker.ID()
			if state == "player" {
				id = b.PlayerID()
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{id}); err != nil {
				t.Fatal(err)
			}
			obj := g.StackPeek()
			instructions, ok := mage.InspectSingularDamageInstructions(mage.Sequence(obj.Effects...))
			if !ok || len(instructions) != 1 {
				t.Fatal("real card lost typed damage")
			}
			predicate, ok := mage.InspectDamageRecipient(instructions[0].Recipient)
			if !ok {
				t.Fatal("real card union recipient is unknown")
			}
			switch state {
			case "phase":
				g.PhaseOut(walker.ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(walker.ID()))
			case "depart-source":
				g.RemoveFromBattlefield(source)
			case "phase-source":
				g.PhaseOut(source.ID())
			case "return-source":
				g.RemoveFromBattlefield(source)
				g.PutOnBattlefield(source.Card, b.PlayerID())
			case "control-source":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "clone":
				g = g.Clone()
				obj = g.StackPeek()
			}
			for range 2 {
				selected, known := predicate.Select(g, obj)
				if !known || selected.Empty() != (state == "phase") {
					t.Fatal("real RON target lost current legality or source information")
				}
			}
			g.ResolveTopOfStack()
			wantLife, wantLoyalty := 20, 9
			if state == "player" {
				wantLife, wantLoyalty = 19, 10
			}
			if state == "phase" {
				wantLoyalty = 10
			}
			if g.GetPlayer(b.PlayerID()).Life() != wantLife || g.FindPermanentIncludingPhased(walker.ID()).Counters[core.Loyalty] != wantLoyalty {
				t.Fatal("real card damage differs from query")
			}
			if state == "clone" && walker.Counters[core.Loyalty] != 10 {
				t.Fatal("clone damage crossed state")
			}
		})
	}
}

func TestRONPlayerPlaneswalkerTargetQueryInstances(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(`+ronPlayerPlaneswalkerDamage+`)`)
	g, a, b := ronGame()
	first := castRONSpell(t, g, a, factory(), a.PlayerID())
	second := castRONSpell(t, g, a, factory(), b.PlayerID())
	for _, obj := range []*mage.StackObject{first, second} {
		if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
			t.Fatal("fresh factory union binding unknown")
		}
	}
	first.TargetSpecs[0] = second.TargetSpecs[0]
	if _, known := g.InspectSpellDestruction(first, nil); known {
		t.Fatal("foreign factory declaration accepted")
	}
	g.ResolveTopOfStack()
	if a.Life() != 20 || b.Life() != 18 {
		t.Fatal("factory target state crossed")
	}
}

func TestRONPlayerPlaneswalkerTargetQueryInvalid(t *testing.T) {
	valid := `Card(Name:"Union",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(` + ronPlayerPlaneswalkerDamage + `)])`
	for _, source := range []string{
		strings.Replace(valid, "AnyPlayer", "AnyPlayer()", 1),
		strings.Replace(valid, "Planeswalker", "Creature", 1),
		strings.Replace(valid, "Ref_TargetPlayerOrPermanent", "Ref_TargetPlayer", 1),
		strings.Replace(valid, "Ref_TargetPlayerOrPermanent", "Ref_TargetPlayerOrPermanent2", 1),
		strings.Replace(valid, ronPlayerPlaneswalkerTarget, "", 1),
	} {
		if _, err := Compile(nameScopeNode(t, source)); err == nil {
			t.Fatalf("invalid union accepted: %s", source)
		}
	}
}
