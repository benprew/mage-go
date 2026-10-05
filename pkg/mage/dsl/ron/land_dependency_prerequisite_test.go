package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONLandDependencyPrerequisite(t *testing.T) {
	for _, name := range []string{"Forest", "Tropical Island", "Savannah"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			landCard := repositoryCard(t, "LEB", name)
			if err := mage.ValidateCardDefinition(name, landCard); err != nil {
				t.Fatal(err)
			}
			landCard.SetOwner(b.PlayerID())
			land := g.PutOnBattlefield(landCard, b.PlayerID())
			g.PutOnBattlefield(mage.NewEnchantment("Select Mountains", "", mage.WithStaticAbility(mage.BecomesBasicLandsEffect(mage.HasSubType("Mountain"), "Plains"))), a.PlayerID())
			source := g.PutOnBattlefield(mage.NewEnchantment("Create Mountains", "", mage.WithStaticAbility(mage.BecomesBasicLandsEffect(mage.IsLand, "Mountain"))), a.PlayerID())
			clone := g.Clone()
			if !g.FindPermanent(land.ID()).HasSubType("Plains") || !g.CanAfford(b.PlayerID(), core.ParseManaCost("{W}"), nil) || g.CanAfford(b.PlayerID(), core.ParseManaCost("{G}"), nil) {
				t.Fatal("RON land did not supply final intrinsic mana")
			}
			if err := g.AutoTapForCost(b.PlayerID(), core.ParseManaCost("{W}")); err != nil {
				t.Fatal(err)
			}
			if !g.FindPermanent(land.ID()).Tapped || clone.FindPermanent(land.ID()).Tapped {
				t.Fatal("mana operation changed clone")
			}
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			if !g.FindPermanent(land.ID()).HasSubType("Forest") || !clone.FindPermanent(land.ID()).HasSubType("Plains") {
				t.Fatal("RON land restoration or clone isolation")
			}
		})
	}
}

func TestRONAttachedEffectsRejectReturnedHost(t *testing.T) {
	for _, name := range []string{"Holy Strength", "Flight"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			hostCard := repositoryCard(t, "LEB", "Grizzly Bears")
			hostCard.SetOwner(b.PlayerID())
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			auraCard := repositoryCard(t, "LEB", name)
			auraCard.SetOwner(a.PlayerID())
			a.AddToHand(auraCard)
			color := core.White
			if name == "Flight" {
				color = core.Blue
			}
			if err := g.CastSpellByID(a.PlayerID(), auraCard.ID(), []uuid.UUID{host.ID()}, 0); err == nil {
				t.Fatal("unpaid Aura cast accepted")
			}
			if g.StackSize() != 0 {
				t.Fatal("failed payment changed stack")
			}
			a.ManaPool().Add(color, 1)
			if err := g.CastSpellByID(a.PlayerID(), auraCard.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			check := func(game *mage.Game, boosted bool) {
				t.Helper()
				p := game.FindPermanent(host.ID())
				if name == "Holy Strength" {
					power, toughness := 2, 2
					if boosted {
						power, toughness = 3, 4
					}
					if p.CurrentPower(game) != power || p.CurrentToughness(game) != toughness {
						t.Fatal("attachment boost uses wrong incarnation")
					}
				} else {
					if p.HasKeyword(core.Flying) != boosted {
						t.Fatal("attachment grant uses wrong incarnation")
					}
				}
			}
			check(g, true)
			clone := g.Clone()
			g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
			host = g.PutOnBattlefield(hostCard, b.PlayerID())
			check(g, false)
			check(clone, true)
			g.Attach(auraCard.ID(), host.ID())
			check(g, true)
		})
	}
}

func TestRONStaticLandTypeStrictLoading(t *testing.T) {
	var records []byte
	for _, set := range []string{"2ED", "3ED", "ARN", "ATQ", "CED", "CEI", "DRK", "FEM", "LEA", "LEB", "LEG", "PROMOS"} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), set+".ron")
		if err != nil {
			t.Fatal(err)
		}
		for _, node := range nodes {
			switch node.Fields["Name"].String {
			case "Conversion", "Evil Presence", "Blood Moon":
				factory, err := Compile(node)
				if err != nil {
					t.Fatal(err)
				}
				if err := mage.ValidateCardDefinition(node.Fields["Name"].String, factory()); err != nil {
					t.Fatal(err)
				}
				records = append(records, []byte(strings.Split(string(data), "\n")[node.Position.Line-1]+"\n")...)
			}
		}
	}
	if strings.Count(string(records), "Card(Name:") != 13 {
		t.Fatal("missing reprints")
	}
	files := fstest.MapFS{"types.ron": {Data: records}}
	if _, err := LoadFS(files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[EachPermanentLayerEffect(AnyPermanent,[SetLandType(Desert)])])`)}
	if registry, err := LoadFS(files, "*.ron"); err == nil || registry != nil {
		t.Fatal("invalid collection published a registry")
	}
	for _, rule := range []string{
		`PermanentLayerEffect(Ref_TargetPermanent,[SetLandType(Swamp)])`,
		`PermanentLayerEffect(HostPermanent,[SetLandType(You)])`,
		`PermanentLayerEffect(HostPermanent,[SetLandType(Swamp())])`,
		`PermanentLayerEffect(HostPermanent,[SetLandType(Swamp,Forest)])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[`+rule+`])`), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}
