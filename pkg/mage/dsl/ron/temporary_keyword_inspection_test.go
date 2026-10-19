package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTemporaryKeywordInspection(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, state := range []string{"normal", "opponent", "copy", "clone", "phased", "returned", "shroud", "blocked", "phase-in"} {
			t.Run(set+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				card := mage.NewCreature("Subject", "", 4, 4)
				card.SetOwner(a.PlayerID())
				controller := a.PlayerID()
				if state == "opponent" {
					controller = b.PlayerID()
				}
				p := g.PutOnBattlefield(card, controller)
				if state == "blocked" {
					blocker := g.PutOnBattlefield(mage.NewCreature("Blocker", "", 4, 4), b.PlayerID())
					g.GetCombat().AddAttacker(p.ID(), b.PlayerID())
					g.GetCombat().AddBlocker(blocker.ID(), p.ID())
				}
				obj := castRONSpell(t, g, a, repositoryCard(t, set, "Jump"), p.ID())
				if state == "copy" {
					obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
				}
				if state == "clone" {
					g = g.Clone()
					obj = g.StackPeek()
				}
				switch state {
				case "phased":
					g.PhaseOut(p.ID())
				case "phase-in":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "returned":
					g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
					p = g.PutOnBattlefield(card, a.PlayerID())
				case "shroud":
					g.FindPermanent(p.ID()).GrantBaseAttr(core.Shroud)
				}
				for range 2 {
					if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
						t.Fatal("native temporary keyword tree is unknown")
					}
				}
				if g.FindPermanentIncludingPhased(p.ID()).HasKeyword(core.Flying) {
					t.Fatal("query installed a grant")
				}
				g.ResolveTopOfStack()
				want := state != "phased" && state != "returned" && state != "shroud"
				if g.FindPermanentIncludingPhased(p.ID()).HasKeyword(core.Flying) != want {
					t.Fatal("wrong grant or target visit")
				}
				if state == "blocked" && !g.GetCombat().GroupFor(p.ID()).Blocked {
					t.Fatal("flying removed a blocker")
				}
				g.PhaseIn([]uuid.UUID{p.ID()})
				g.PhaseIn(g.PhaseOut(p.ID()))
				if g.FindPermanent(p.ID()).HasKeyword(core.Flying) != want {
					t.Fatal("phasing changed installed grant")
				}
				g.GetEffects().RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				if g.FindPermanent(p.ID()).HasKeyword(core.Flying) {
					t.Fatal("grant survived cleanup")
				}
			})
		}
	}
}

func TestRONTemporaryKeywordCollectionInspection(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([CreateEachPermanentLayerEffectUntil(And([IsCardtype(Creature), ControlledByAPlayer(SinglePlayer(You))]), [AddAbility([Flying, FirstStrike]), AddAbility([Trample])], UntilEndOfTurn)]))`)
	for _, copied := range []bool{false, true} {
		t.Run(fmt.Sprint(copied), func(t *testing.T) {
			g, a, b := ronGame()
			own := g.PutOnBattlefield(mage.NewCreature("Own", "", 4, 4), a.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 4, 4), b.PlayerID())
			obj := castRONSpell(t, g, a, factory())
			if copied {
				obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
			}
			if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
				t.Fatal("native collection keyword tree is unknown")
			}
			if own.HasKeyword(core.Flying) || other.HasKeyword(core.Flying) {
				t.Fatal("query installed grants")
			}
			early := g.PutOnBattlefield(mage.NewCreature("Early", "", 4, 4), obj.Controller)
			g.ResolveTopOfStack()
			if !g.FindPermanent(early.ID()).HasKeyword(core.Flying) {
				t.Fatal("collection omitted a resolution-time member")
			}
			for _, kw := range []core.Attr{core.Flying, core.FirstStrike, core.Trample} {
				if g.FindPermanent(own.ID()).HasKeyword(kw) != !copied || g.FindPermanent(other.ID()).HasKeyword(kw) != copied {
					t.Fatal("wrong collection controller or incomplete keyword list")
				}
			}
			late := g.PutOnBattlefield(mage.NewCreature("Late", "", 4, 4), obj.Controller)
			if late.HasKeyword(core.Flying) {
				t.Fatal("collection was not captured at resolution")
			}
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			if g.FindPermanent(own.ID()).HasKeyword(core.Flying) || g.FindPermanent(other.ID()).HasKeyword(core.Flying) {
				t.Fatal("collection grant survived cleanup")
			}
		})
	}
}

func TestRONTemporaryKeywordInspectionRejectsDeclarations(t *testing.T) {
	for _, mods := range []string{`[]`, `[AddAbility([])]`, `[AddAbility([Flying()])]`, `[AddAbility([Flying], FirstStrike)]`, `[AddAbility([Flying, Unknown])]`} {
		source := fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Instant], Subtypes: []), Rules: [SpellActions(Targeted([TargetPermanent(AnyPermanent)], ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent, %s, UntilEndOfTurn)])))])`, mods)
		nodes, err := Parse(strings.NewReader(source), "invalid-keywords.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", mods)
		}
	}
	for _, mods := range []string{`AddAbility([Protection(FromColor(Colors([Red])))])`} {
		factory := permanentScopeFactory(t, "Instant", fmt.Sprintf(`SpellActions(Targeted([TargetPermanent(AnyPermanent)], ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent, [%s], UntilEndOfTurn)])))`, mods))
		g, a, _ := ronGame()
		p := g.PutOnBattlefield(mage.NewCreature("Target", "", 4, 4), a.PlayerID())
		obj := castRONSpell(t, g, a, factory(), p.ID())
		if _, known := g.InspectSpellDestruction(obj, nil); known {
			t.Fatalf("accepted unsupported complete tree: %s", mods)
		}
	}
}

func TestRONTemporaryKeywordInspectionCompleteSourcePrevention(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"LEG", "Subdue"}} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, _ := ronGame()
			p := g.PutOnBattlefield(mage.NewCreature("Subject", "", 4, 4), a.PlayerID())
			obj := castRONSpell(t, g, a, repositoryCard(t, tc.set, tc.name), p.ID())
			if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
				t.Fatal("complete source prevention tree failed native inspection")
			}
		})
	}
}
