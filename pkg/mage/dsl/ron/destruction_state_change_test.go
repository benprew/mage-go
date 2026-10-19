package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONDestructionStateChangeRealCards(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, name := range []string{"Shatter", "Wrath of God"} {
			for _, protected := range []bool{false, true} {
				t.Run(set+"/"+name+map[bool]string{false: "/change", true: "/unchanged"}[protected], func(t *testing.T) {
					g, a, b := ronGame()
					card := mage.NewArtifact("Prefix", "")
					if name == "Wrath of God" {
						card = mage.NewCreature("Prefix", "", 2, 2)
					}
					p := g.PutOnBattlefield(card, b.PlayerID())
					land := g.PutOnBattlefield(mage.NewLand("Watched"), a.PlayerID())
					if protected {
						p.GrantBaseAttr(core.Indestructible)
					}
					query, _ := mage.InspectContextualPermanentCollection(mage.MatchingPermanents(mage.IsLand))
					watched := query.Select(&mage.EffectContext{Game: g, Controller: a.PlayerID()})
					var obj *mage.StackObject
					if name == "Shatter" {
						obj = castRONSpell(t, g, a, repositoryCard(t, set, name), p.ID())
					} else {
						obj = castRONSpell(t, g, a, repositoryCard(t, set, name))
					}
					instructions, ok := mage.InspectDirectDestruction(mage.Sequence(obj.Effects...))
					if !ok || len(instructions) != 1 || instructions[0].CannotRegenerate != (name == "Wrath of God") {
						t.Fatal("RON lost native destruction declaration")
					}
					clone := g.Clone()
					clone.ResolveStack()
					if (clone.FindPermanent(p.ID()) != nil) != protected || clone.FindPermanent(land.ID()) == nil {
						t.Fatal("real card gameplay differs")
					}
					obj.Effects = append(obj.Effects, (mage.DirectDestructionInstruction{Subjects: mage.MatchingPermanents(mage.IsLand)}).Effect())
					for range 2 {
						if certain, known := g.InspectSpellDestruction(obj, watched); certain != protected || known != protected {
							t.Fatalf("prefix proof=%v/%v protected=%v", certain, known, protected)
						}
					}
					g.ResolveStack()
					if g.FindPermanent(land.ID()) != nil || (g.FindPermanent(p.ID()) != nil) != protected {
						t.Fatal("composition execution differs")
					}
				})
			}
		}
	}
}

func TestRONDestructionStateChangeSequence(t *testing.T) {
	for _, action := range []string{"DestroyPermanent", "DestroyPermanentNoRegen"} {
		for _, state := range []string{"indestructible", "ordinary", "regenerate", "phase", "cycle", "return", "shroud", "copy", "all-illegal"} {
			t.Run(action+"/"+state, func(t *testing.T) {
				factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Land)),TargetPermanent(IsCardtype(Land))],ActionList([`+action+`(Ref_TargetPermanent1),DestroyPermanent(Ref_TargetPermanent2)])))`)
				g, a, b := ronGame()
				one := g.PutOnBattlefield(mage.NewLand("Prefix"), b.PlayerID())
				two := g.PutOnBattlefield(mage.NewLand("Watched"), b.PlayerID())
				one.GrantBaseAttr(core.Indestructible)
				query, _ := mage.InspectContextualPermanentCollection(mage.MatchingPermanents(mage.IsLand))
				visits := query.Select(&mage.EffectContext{Game: g, Controller: a.PlayerID()})
				for i, visit := range visits {
					if visit.ID == two.ID() {
						visits = visits[i : i+1]
						break
					}
				}
				if len(visits) != 1 || visits[0].ID != two.ID() {
					t.Fatal("watched visit missing")
				}
				obj := castRONSpell(t, g, a, factory(), one.ID(), two.ID())
				switch state {
				case "ordinary", "regenerate":
					one.RevokeBaseAttr(core.Indestructible)
					if state == "regenerate" {
						g.AddRegenerationShield(one.ID())
					}
				case "phase":
					g.PhaseOut(one.ID())
				case "cycle":
					g.PhaseIn(g.PhaseOut(one.ID()))
				case "return":
					g.BouncePermanentToHand(one)
					g.PutOnBattlefield(one.Card, b.PlayerID())
				case "shroud", "all-illegal":
					one.GrantBaseAttr(core.Shroud)
					if state == "all-illegal" {
						two.GrantBaseAttr(core.Shroud)
					}
				case "copy":
					if g.CopyStackObjectDirect(obj, b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				want := state != "ordinary" && state != "regenerate" && state != "all-illegal"
				wantKnown := state != "ordinary" && state != "regenerate"
				for _, game := range []*mage.Game{g, g.Clone()} {
					if certain, known := game.InspectSpellDestruction(game.StackPeek(), visits); certain != want || known != wantKnown {
						t.Fatalf("proof=%v/%v want=%v/%v", certain, known, want, wantKnown)
					}
				}
				g.ResolveStack()
				if (g.FindPermanent(two.ID()) == nil) != (state != "all-illegal") {
					t.Fatal("target legality differs during execution")
				}
			})
		}
	}
}
