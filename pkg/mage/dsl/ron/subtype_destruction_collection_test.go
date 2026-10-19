package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONSubtypeDestructionCollection(t *testing.T) {
	for _, syntax := range []string{
		"IsLandType(Forest)", "IsNonLandType(Island)",
		"And([IsCardtype(Land),IsLandType(Forest)])",
		"Or([IsLandType(Forest),IsLandType(Island)])",
		"And([Or([IsLandType(Forest),IsCardtype(Creature)]),IsNonLandType(Island)])",
	} {
		build, err := compileDestructionCollection(nameScopeNode(t, syntax), scope{resolving: true})
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := mage.InspectPermanentCollection(build(instanceBindings{})); !ok {
			t.Fatalf("lost complete subtype collection: %s", syntax)
		}
	}
	for _, syntax := range []string{
		"IsLandType", "IsLandType()", "IsLandType(Forest,Island)", "IsLandType(Forest())",
		"IsNonLandType()", "And([])", "Or([IsLandType(Forest),And([])])",
		"And([IsLandType(Forest),Ref_TargetPermanent])",
	} {
		if _, err := compileDestructionCollection(nameScopeNode(t, syntax), scope{resolving: true}); err == nil {
			t.Fatalf("accepted invalid collection: %s", syntax)
		}
	}
}

func TestRONSubtypeDestructionCollectionGameplay(t *testing.T) {
	for _, tc := range []struct{ name, subtype string }{
		{"Flashfires", "Plains"}, {"Tsunami", "Island"}, {"Acid Rain", "Forest"}, {"Tivadar's Crusade", "Goblin"},
	} {
		sets := []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"}
		if tc.name == "Acid Rain" {
			sets = []string{"LEG"}
		}
		if tc.name == "Tivadar's Crusade" {
			sets = []string{"DRK"}
		}
		for _, set := range sets {
			t.Run(set+"/"+tc.name, func(t *testing.T) {
				for _, state := range []string{"normal", "regenerate", "indestructible", "phase", "changed", "return"} {
					t.Run(state, func(t *testing.T) {
						g, a, b := ronGame()
						newMember := func(name string) *mage.BaseCard {
							if tc.subtype == "Goblin" {
								return mage.NewCreature(name, "", 2, 2, mage.WithSubTypes(tc.subtype, "Warrior"))
							}
							return mage.NewLand(name, mage.WithSubTypes(tc.subtype, "Desert"))
						}
						card := newMember("Multisubtype member")
						card.SetOwner(b.PlayerID())
						member := g.PutOnBattlefield(card, b.PlayerID())
						other := g.PutOnBattlefield(newMember("Other matching permanent"), a.PlayerID())
						sameName := g.PutOnBattlefield(mage.NewLand(tc.subtype), a.PlayerID())
						spell := castRONSpell(t, g, a, repositoryCard(t, set, tc.name))
						switch state {
						case "regenerate":
							g.AddRegenerationShield(member.ID())
						case "indestructible":
							member.GrantBaseAttr(core.Indestructible)
						case "phase":
							g.PhaseOut(member.ID())
						case "changed":
							family, subtype := mage.SubtypeLand, "Mountain"
							if tc.subtype == "Goblin" {
								family, subtype = mage.SubtypeCreature, "Elf"
							}
							g.AddContinuousEffect(mage.SetSubtypes(member.ID(), family, core.EndOfTurn, subtype))
						case "return":
							g.RemoveFromBattlefield(member)
							member = g.PutOnBattlefield(card, a.PlayerID())
						}
						instructions, ok := mage.InspectDirectDestruction(spell.Effects[0])
						if !ok || len(instructions) != 1 {
							t.Fatal("missing simultaneous instruction")
						}
						plan, ok := mage.InspectPermanentCollection(instructions[0].Subjects)
						if !ok {
							t.Fatal("missing pure subtype plan")
						}
						visits := plan.Select(g)
						wantSelected := state != "phase" && state != "changed"
						wantCount := 1
						if wantSelected {
							wantCount++
						}
						if len(visits) != wantCount {
							t.Fatalf("wrong current membership: %v", visits)
						}
						if wantSelected && visits[0].ID != member.ID() && visits[1].ID != member.ID() {
							t.Fatal("selected by name instead of subtype")
						}
						clone := g.Clone()
						clone.ResolveStack()
						if g.StackSize() != 1 || g.FindPermanentIncludingPhased(member.ID()) == nil {
							t.Fatal("clone changed original")
						}
						g.ResolveStack()
						wantDestroyed := state == "normal" || state == "return"
						for _, game := range []*mage.Game{g, clone} {
							if (game.FindPermanentIncludingPhased(member.ID()) == nil) != wantDestroyed || game.FindPermanent(sameName.ID()) == nil || game.FindPermanent(other.ID()) != nil {
								t.Fatal("native destruction disagrees with subtype selection")
							}
						}
					})
				}
			})
		}
	}
}
