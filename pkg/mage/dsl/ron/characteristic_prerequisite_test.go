package ron

import (
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCharacteristicPrerequisiteGameplay(t *testing.T) {
	for _, spellName := range []string{"Giant Growth", "Unsummon", "Swords to Plowshares"} {
		t.Run(spellName, func(t *testing.T) {
			g, a, b := ronGame()
			for range 3 {
				g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), b.PlayerID())
			}
			creature := mage.NewCreature("Count fixture", "", 0, 0, mage.WithStaticAbility(mage.PTEqualsControlledCount(mage.IsLand)))
			creature.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(creature, b.PlayerID())
			g.ApplyContinuousEffects()
			spell := repositoryCard(t, "LEB", spellName)
			if err := mage.ValidateCardDefinition(spell.Name(), spell); err != nil {
				t.Fatal(err)
			}
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{target.ID()}, 0); err == nil {
				t.Fatal("unpaid spell")
			}
			if len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("failed cast changed state")
			}
			color := core.Green
			if spellName == "Unsummon" {
				color = core.Blue
			}
			if spellName == "Swords to Plowshares" {
				color = core.White
			}
			a.ManaPool().Add(color, 1)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			g.ResolveStack()
			power, toughness := g.CardPowerToughness(creature)
			if spellName == "Giant Growth" {
				if power != 6 || toughness != 6 {
					t.Fatalf("bonus P/T %d/%d", power, toughness)
				}
				g.RunStepWithPriority(core.Cleanup)
				power, toughness = g.CardPowerToughness(creature)
			} else {
				if g.FindPermanent(target.ID()) != nil {
					t.Fatal("zone action failed")
				}
				if spellName == "Unsummon" && len(b.Hand()) != 1 {
					t.Fatal("owner hand")
				}
				if spellName == "Swords to Plowshares" && b.Life() != 23 {
					t.Fatal("CDA power LKI or controller lost")
				}
			}
			if power != 3 || toughness != 3 {
				t.Fatalf("zone/cleanup P/T %d/%d", power, toughness)
			}
			if p, q := clone.CardPowerToughness(creature); p != 3 || q != 3 {
				t.Fatal("clone changed")
			}
			clone.ResolveStack()
			if spellName == "Swords to Plowshares" && clone.GetPlayer(b.PlayerID()).Life() != 23 {
				t.Fatal("clone LKI")
			}
		})
	}
}

func TestRONCharacteristicPrerequisiteKeepsAdapterGate(t *testing.T) {
	names := map[string]bool{"Dakkon Blackblade": true, "Keldon Warlord": true, "Nightmare": true, "Plague Rats": true, "People of the Woods": true}
	files, err := os.ReadDir("../../../../cards/mtgish")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".ron") {
			continue
		}
		data, err := os.ReadFile("../../../../cards/mtgish/" + file.Name())
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), file.Name())
		if err != nil {
			t.Fatal(err)
		}
		for _, node := range nodes {
			name := node.Fields["Name"].String
			if !names[name] {
				continue
			}
			seen[name] = true
			if _, err := Compile(node); err == nil || !strings.Contains(err.Error(), "unsupported rule (CDA_") {
				t.Fatalf("%s crossed incomplete adapter gate: %v", name, err)
			}
		}
	}
	if len(seen) != len(names) {
		t.Fatal("missing candidate records")
	}
	for _, rule := range []string{`CDA_Power()`, `CDA_Power(ValueX)`, `CDA_Toughness(Ref_TargetPermanent)`, `CDA_Toughness(TheNumberOfPermanentsOnTheBattlefield(And([])))`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),Rules:[`+rule+`],CardPT:(Power:0,Toughness:0))`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("invalid CDA accepted: %s", rule)
		}
	}
}
