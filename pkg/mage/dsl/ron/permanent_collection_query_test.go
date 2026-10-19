package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONPermanentCollectionQuery(t *testing.T) {
	for _, syntax := range []string{`IsCardtype(Land)`, `IsNonCardtype(Creature)`, `And([IsCardtype(Land),IsNonCardtype(Creature)])`, `Or([IsCardtype(Land),IsCardtype(Creature)])`} {
		build, err := compileDestructionCollection(nameScopeNode(t, syntax), scope{resolving: true})
		if err != nil {
			t.Fatal(err)
		}
		set := build(instanceBindings{})
		if _, ok := mage.InspectPermanentCollection(set); !ok {
			t.Fatalf("lost pure collection: %s", syntax)
		}
	}
	for _, syntax := range []string{`IsCardtype(Unknown)`, `And([])`, `Or([])`, `And([And([]),IsCardtype(Land)])`, `Or([IsCardtype(Land),Or([])])`, `And([IsCardtype(Land),Unknown])`, `IsCardtype(Land,Creature)`, `Ref_TargetPermanents`} {
		if _, err := compileDestructionCollection(nameScopeNode(t, syntax), scope{resolving: true}); err == nil {
			t.Fatalf("accepted %s", syntax)
		}
	}
}

func TestRONPermanentCollectionQueryGameplay(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, name := range []string{"Armageddon", "Wrath of God"} {
			t.Run(set+"/"+name, func(t *testing.T) {
				g, a, b := ronGame()
				var card mage.Card = mage.NewLand("Member")
				if name == "Wrath of God" {
					card = mage.NewCreature("Member", "", 2, 2)
				}
				card.SetOwner(b.PlayerID())
				member := g.PutOnBattlefield(card, b.PlayerID())
				var phasedCard mage.Card = mage.NewLand("Phased")
				if name == "Wrath of God" {
					phasedCard = mage.NewCreature("Phased", "", 2, 2)
				}
				phased := g.PutOnBattlefield(phasedCard, a.PlayerID())
				g.PhaseOut(phased.ID())
				spell := castRONSpell(t, g, a, repositoryCard(t, set, name))
				instructions, ok := mage.InspectDirectDestruction(spell.Effects[0])
				if !ok || len(instructions) != 1 {
					t.Fatal("missing instruction")
				}
				plan, ok := mage.InspectPermanentCollection(instructions[0].Subjects)
				if !ok {
					t.Fatal("missing pure collection")
				}
				visits := plan.Select(g)
				if len(visits) != 1 || visits[0].ID != member.ID() {
					t.Fatalf("members = %v", visits)
				}
				clone := g.Clone()
				if got := plan.Select(clone); len(got) != 1 || got[0] != visits[0] {
					t.Fatal("clone lost visits")
				}
				clone.ResolveStack()
				if len(plan.Select(g)) != 1 || len(plan.Select(clone)) != 0 {
					t.Fatal("query or clone changed original")
				}
				g.ResolveStack()
				if g.FindPermanent(member.ID()) != nil || g.FindPermanentIncludingPhased(phased.ID()) == nil {
					t.Fatal("wrong native destruction")
				}
			})
		}
	}
}

func TestRONPermanentCollectionQueryDisk(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			g, a, b := ronGame()
			disk := g.PutOnBattlefield(repositoryCard(t, set, "Nevinyrral's Disk"), a.PlayerID())
			member := g.PutOnBattlefield(mage.NewCreature("Member", "", 2, 2), b.PlayerID())
			land := g.PutOnBattlefield(mage.NewLand("Land"), a.PlayerID())
			a.ManaPool().Add(core.Colorless, 1)
			if !disk.Tapped {
				t.Fatal("source did not enter tapped")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), disk.ID(), 0, nil); err == nil || a.ManaPool().TotalMana() != 1 {
				t.Fatal("failed tap cost changed resources")
			}
			g.UntapPermanent(disk)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), disk.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			instructions, ok := mage.InspectDirectDestruction(g.StackPeek().Effects[0])
			if !ok || len(instructions) != 1 {
				t.Fatal("missing group")
			}
			plan, ok := mage.InspectPermanentCollection(instructions[0].Subjects)
			if !ok || len(plan.Select(g)) != 2 {
				t.Fatal("source not selected with the other types")
			}
			g.ResolveStack()
			if g.FindPermanent(disk.ID()) != nil || g.FindPermanent(member.ID()) != nil || g.FindPermanent(land.ID()) == nil {
				t.Fatal("wrong group destroyed")
			}
		})
	}
}
