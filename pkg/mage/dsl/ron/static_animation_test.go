package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONStaticAnimationRealCards(t *testing.T) {
	for _, tc := range []struct {
		set, name, land, excluded string
		black                     bool
		color                     core.Color
		mana                      int
	}{
		{"LEB", "Living Lands", "Forest", "Swamp", false, core.Green, 4},
		{"LEG", "Living Plane", "Island", "", false, core.Green, 4},
		{"LEB", "Kormus Bell", "Swamp", "Forest", true, core.Black, 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			land := g.PutOnBattlefield(repositoryCard(t, "LEB", tc.land), b.PlayerID())
			var excluded *mage.Permanent
			if tc.excluded != "" {
				excluded = g.PutOnBattlefield(repositoryCard(t, "LEB", tc.excluded), a.PlayerID())
			}
			card := repositoryCard(t, tc.set, tc.name)
			if err := mage.ValidateCardDefinition(tc.name, card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid casting succeeded")
			}
			if len(a.Hand()) != 1 || g.Stack().Size() != 0 || land.HasType(core.TypeCreature) {
				t.Fatal("failed casting changed game")
			}
			a.ManaPool().Add(tc.color, tc.mana)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if land.HasType(core.TypeCreature) || a.ManaPool().TotalMana() != 0 {
				t.Fatal("animation before resolution or wrong cost")
			}
			clone := g.Clone()
			clone.ResolveTopOfStack()
			if !clone.FindPermanent(land.ID()).HasType(core.TypeCreature) || g.FindPermanent(land.ID()).HasType(core.TypeCreature) {
				t.Fatal("clone resolution changed original")
			}
			g.ResolveTopOfStack()
			check := func(game *mage.Game, id *mage.Permanent) {
				t.Helper()
				p := game.FindPermanent(id.ID())
				if !p.HasType(core.TypeCreature) || !p.HasType(core.TypeLand) || p.CurrentPower(game) != 1 || p.CurrentToughness(game) != 1 || !p.HasSubType(tc.land) {
					t.Fatal("wrong animated characteristics")
				}
				if tc.black && (len(p.Colors()) != 1 || p.Colors()[0] != core.Black) {
					t.Fatal("missing black color")
				}
			}
			check(g, land)
			if excluded != nil && g.FindPermanent(excluded.ID()).HasType(core.TypeCreature) {
				t.Fatal("nonmatching land animated")
			}
			late := g.PutOnBattlefield(repositoryCard(t, "LEB", tc.land), a.PlayerID())
			check(g, late)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), late.ID(), 0, nil); err == nil {
				t.Fatal("new animated land ignored summoning sickness")
			}
			source := g.FindPermanent(card.ID())
			phased := g.PhaseOut(source.ID())
			g.ApplyContinuousEffects()
			if g.FindPermanent(land.ID()).HasType(core.TypeCreature) {
				t.Fatal("phased source animated")
			}
			g.PhaseIn(phased)
			g.ApplyContinuousEffects()
			check(g, land)
			targetPhased := g.PhaseOut(land.ID())
			g.ApplyContinuousEffects()
			g.PhaseIn(targetPhased)
			g.ApplyContinuousEffects()
			check(g, land)
			removal := mage.RemoveAllAbilities(source.ID(), core.EndOfTurn)
			g.AddContinuousEffect(removal)
			check(g, land)
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			if p := g.FindPermanent(land.ID()); p.HasType(core.TypeCreature) || p.ColorOverride != nil || p.BasePTOverride != nil {
				t.Fatal("departed animation retained")
			}
			if !g.CanAfford(b.PlayerID(), core.ParseManaCost("{1}"), nil) {
				t.Fatal("land lost mana ability")
			}
			g.PutOnBattlefield(card, a.PlayerID())
			check(g, land)
		})
	}
}

func TestRONStaticAnimationInvalid(t *testing.T) {
	for _, mods := range []string{
		`SetPT(PT(1)), AddCardtype(Creature)`, `SetPT(PT(ValueX, 1)), AddCardtype(Creature)`,
		`SetPT(ManualPT(1, 1)), AddCardtype(Creature)`, `SetPT(PT(1, 1)), AddCardtype(Creature())`,
		`SetPT(PT(1, 1)), AddCardtype(Artifact)`, `SetPT(PT(1, 1))`, `AddCardtype(Creature)`,
		`SetPT(PT(1, 1)), SetPT(PT(2, 2)), AddCardtype(Creature)`,
		`SetPT(PT(1, 1)), AddCardtype(Creature), SetColor(SimpleColorList([Colorless]))`,
		`SetPT(PT(1, 1)), AddCardtype(Creature), SetColor(TheChosenColor)`,
	} {
		t.Run(mods, func(t *testing.T) {
			src := fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Enchantment], Subtypes: []), Rules: [EachPermanentLayerEffect(IsCardtype(Land), [%s])])`, mods)
			nodes, err := Parse(strings.NewReader(src), "invalid-animation.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Compile(nodes[0]); err == nil {
				t.Fatal("invalid animation accepted")
			}
		})
	}
}

func TestRONStaticAnimationInstancesAndControl(t *testing.T) {
	factory := staticPermanentFixture(t, "Enchantment", "", `EachPermanentLayerEffect(And([IsCardtype(Land), ControlledByAPlayer(SinglePlayer(You)), Other(ThisPermanent)]), [AddCardtype(Creature), SetColor(SimpleColorList([Blue, Red])), SetPT(PT(2, 3))])`)
	g, a, b := ronGame()
	first, second := g.PutOnBattlefield(factory(), a.PlayerID()), g.PutOnBattlefield(factory(), b.PlayerID())
	own, other := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID()), g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), b.PlayerID())
	for _, ref := range []*mage.Permanent{own, other} {
		p := g.FindPermanent(ref.ID())
		if !p.HasType(core.TypeCreature) || len(p.Colors()) != 2 || p.CurrentPower(g) != 2 || p.CurrentToughness(g) != 3 {
			t.Fatal("per-instance animation failed")
		}
	}
	clone := g.Clone()
	clone.RemoveFromBattlefield(clone.FindPermanent(second.ID()))
	if clone.FindPermanent(other.ID()).HasType(core.TypeCreature) || !g.FindPermanent(other.ID()).HasType(core.TypeCreature) {
		t.Fatal("instance or clone selection shared")
	}
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: second.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
	if g.FindPermanent(other.ID()).HasType(core.TypeCreature) || !g.FindPermanent(own.ID()).HasType(core.TypeCreature) {
		t.Fatal("animation used owner or retained old controller")
	}
	g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
	if !g.FindPermanent(own.ID()).HasType(core.TypeCreature) {
		t.Fatal("one source removed another source's animation")
	}
}

func TestRONStaticAnimationColorsAndInventory(t *testing.T) {
	for _, colors := range []string{"", `, SetColor(SimpleColorList([Black]))`, `, SetColor(SimpleColorList([]))`} {
		t.Run(colors, func(t *testing.T) {
			src := `Card(Name: "Colored land", Typeline: (Supertypes: [], Cardtypes: [Land, Artifact], Subtypes: [Forest]), ColorIndicator: [Blue], Rules: [])`
			nodes, err := Parse(strings.NewReader(src), "colored.ron")
			if err != nil {
				t.Fatal(err)
			}
			landFactory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			factory := staticPermanentFixture(t, "Enchantment", "", `EachPermanentLayerEffect(IsCardtype(Land), [SetPT(PT(1, 1)), AddCardtype(Creature)`+colors+`])`)
			g, a, b := ronGame()
			land := g.PutOnBattlefield(landFactory(), a.PlayerID())
			source := g.PutOnBattlefield(factory(), b.PlayerID())
			p := g.FindPermanent(land.ID())
			if !p.HasType(core.TypeArtifact) || !p.HasType(core.TypeLand) || !p.HasType(core.TypeCreature) {
				t.Fatal("animation removed an existing card type")
			}
			want := []core.Color{core.Blue}
			if strings.Contains(colors, "Black") {
				want = []core.Color{core.Black}
			} else if colors != "" {
				want = nil
			}
			if fmt.Sprint(p.Colors()) != fmt.Sprint(want) {
				t.Fatalf("colors = %v, want %v", p.Colors(), want)
			}
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			if p := g.FindPermanent(land.ID()); len(p.Colors()) != 1 || p.Colors()[0] != core.Blue {
				t.Fatal("color override did not expire")
			}
		})
	}
	src := `Card(Name: "Inventory", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [EachPermanentLayerEffect(IsCardtype(Land), [SetPT(PT(1, 1)), SetColor(SimpleColorList([Black])), AddCardtype(Creature)])])`
	nodes, err := Parse(strings.NewReader(src), "inventory-animation.ron")
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string]*InventorySignature{}
	walkInventory(nodes[0], "Inventory", "Card", "card", "collection", false, entries)
	seen := map[string]bool{}
	for _, entry := range entries {
		want := map[string]string{"PT": "fixed_pt", "SimpleColorList": "color_list", "Black": "color", "Creature": "card_type"}[entry.Name]
		if want != "" {
			seen[entry.Name] = true
			if entry.Role != want {
				t.Fatalf("%s role = %s, want %s", entry.Name, entry.Role, want)
			}
		}
		if entry.Support != "unverified" {
			t.Fatal("inventory claims gameplay support")
		}
	}
	if len(seen) != 4 {
		t.Fatal("animation operand missing from inventory")
	}
}
