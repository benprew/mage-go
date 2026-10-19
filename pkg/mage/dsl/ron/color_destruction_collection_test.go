package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONColorDestructionCollection(t *testing.T) {
	for _, syntax := range []string{
		"IsColor(Black)", "IsNonColor(Black)",
		"And([IsColor(Black),IsCardtype(Creature)])",
		"Or([IsColor(White),IsColor(Black)])",
		"And([IsNonColor(Black),Or([IsCreatureType(Elf),IsCardtype(Artifact)])])",
	} {
		build, err := compileDestructionCollection(nameScopeNode(t, syntax), scope{resolving: true})
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := mage.InspectPermanentCollection(build(instanceBindings{})); !ok {
			t.Fatalf("lost color declaration: %s", syntax)
		}
	}
	for _, syntax := range []string{
		"IsColor", "IsColor()", "IsColor(Black,White)", "IsColor(Black())", "IsColor(Unknown)",
		"IsNonColor()", "And([])", "Or([IsColor(Black),And([])])", "And([IsColor(Black),Ref_TargetPermanent])",
	} {
		if _, err := compileDestructionCollection(nameScopeNode(t, syntax), scope{resolving: true}); err == nil {
			t.Fatalf("accepted %s", syntax)
		}
	}
}

func TestRONColorDestructionCollectionGameplay(t *testing.T) {
	for _, name := range []string{"Cleanse", "Hellfire"} {
		for _, state := range []string{"normal", "regenerate", "indestructible", "phase", "changed", "return"} {
			t.Run(name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				cost, otherCost := "{B}{W}", ""
				if name == "Hellfire" {
					cost, otherCost = "", "{B}{W}"
				}
				member := g.PutOnBattlefield(mage.NewCreature("Member", cost, 2, 2), b.PlayerID())
				other := g.PutOnBattlefield(mage.NewCreature("Other", otherCost, 2, 2), a.PlayerID())
				noncreature := g.PutOnBattlefield(mage.NewEnchantment("Noncreature", cost), a.PlayerID())
				spell := castRONSpell(t, g, a, repositoryCard(t, "LEG", name))
				switch state {
				case "regenerate":
					g.AddRegenerationShield(member.ID())
				case "indestructible":
					member.GrantBaseAttr(core.Indestructible)
				case "phase":
					g.PhaseOut(member.ID())
				case "changed":
					color := core.White
					if name == "Hellfire" {
						color = core.Black
					}
					g.AddContinuousEffect(mage.BecomesColor(member.ID(), color, core.EndOfTurn))
				case "return":
					g.RemoveFromBattlefield(member)
					member = g.PutOnBattlefield(member.Card, a.PlayerID())
				}
				filter := mage.HasColorFilter(core.Black)
				if name == "Hellfire" {
					filter = mage.Not(filter)
				}
				plan, ok := mage.InspectPermanentCollection(mage.MatchingPermanents(mage.And(filter, mage.IsCreature)))
				if !ok {
					t.Fatal("missing color plan")
				}
				if name == "Cleanse" {
					instructions, ok := mage.InspectDirectDestruction(spell.Effects[0])
					if !ok || len(instructions) != 1 {
						t.Fatal("missing native instruction")
					}
					plan, ok = mage.InspectPermanentCollection(instructions[0].Subjects)
					if !ok {
						t.Fatal("lost RON color plan")
					}
				}
				wantSelected := state != "phase" && state != "changed"
				visits := plan.Select(g)
				if wantSelected && (len(visits) != 1 || visits[0].ID != member.ID()) || !wantSelected && len(visits) != 0 {
					t.Fatalf("selection: %v", visits)
				}
				clone := g.Clone()
				clone.ResolveStack()
				if g.FindPermanentIncludingPhased(member.ID()) == nil || g.StackSize() != 1 {
					t.Fatal("clone changed original")
				}
				g.ResolveStack()
				destroyed := state == "normal" || state == "return"
				for _, game := range []*mage.Game{g, clone} {
					if (game.FindPermanentIncludingPhased(member.ID()) == nil) != destroyed || game.FindPermanent(other.ID()) == nil || game.FindPermanent(noncreature.ID()) == nil {
						t.Fatal("native color selection")
					}
					wantLife := 20
					if name == "Hellfire" {
						wantLife = 17
						if destroyed {
							wantLife--
						}
					}
					if game.GetPlayer(a.PlayerID()).Life() != wantLife {
						t.Fatal("death count or recipient")
					}
				}
			})
		}
	}
}
