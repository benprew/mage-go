package ron

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONStaticLandwalkAuras(t *testing.T) {
	for _, tc := range []struct {
		name, set, land string
		keyword         core.Attr
		color           core.Color
	}{
		{"Burrowing", "LEB", "Mountain", core.Mountainwalk, core.Red},
		{"Fishliver Oil", "ARN", "Island", core.Islandwalk, core.Blue},
	} {
		for _, change := range []string{"none", "returned", "phased"} {
			t.Run(tc.name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				host := g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), b.PlayerID())
				other := g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), a.PlayerID())
				land := g.PutOnBattlefield(repositoryCard(t, "LEB", tc.land), a.PlayerID())
				card := repositoryCard(t, tc.set, tc.name)
				if err := mage.ValidateCardDefinition(tc.name, card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				a.ManaPool().Add(tc.color, 2)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{land.ID()}, 0); err == nil {
					t.Fatal("noncreature enchant target accepted")
				}
				if a.ManaPool().TotalMana() != 2 || g.Stack().Size() != 0 {
					t.Fatal("illegal target changed stack")
				}
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
					t.Fatal(err)
				}
				if host.HasKeyword(tc.keyword) {
					t.Fatal("grant started before resolution")
				}
				switch change {
				case "returned":
					g.RemoveFromBattlefield(host)
					host = g.PutOnBattlefield(host.Card, b.PlayerID())
				case "phased":
					g.PhaseOut(host.ID())
				}
				g.ResolveTopOfStack()
				if change != "none" {
					if g.FindPermanent(card.ID()) != nil || host.HasKeyword(tc.keyword) {
						t.Fatal("illegal target received Aura grant")
					}
					return
				}
				if !host.HasKeyword(tc.keyword) || other.HasKeyword(tc.keyword) {
					t.Fatal("wrong host membership")
				}
				if !mage.HasLandwalkEvasion(host, a.PlayerID(), g) || mage.HasLandwalkEvasion(host, b.PlayerID(), g) {
					t.Fatal("landwalk used wrong defender")
				}
				phasedLand := g.PhaseOut(land.ID())
				g.ApplyContinuousEffects()
				if mage.HasLandwalkEvasion(g.FindPermanent(host.ID()), a.PlayerID(), g) {
					t.Fatal("phased land enabled evasion")
				}
				g.PhaseIn(phasedLand)
				g.ApplyContinuousEffects()
				if !mage.HasLandwalkEvasion(g.FindPermanent(host.ID()), a.PlayerID(), g) {
					t.Fatal("phase-in lost landwalk")
				}
				clone := g.Clone()
				clone.RemoveFromBattlefield(clone.FindPermanent(card.ID()))
				if clone.FindPermanent(host.ID()).HasKeyword(tc.keyword) || !g.FindPermanent(host.ID()).HasKeyword(tc.keyword) {
					t.Fatal("clone changed original grant")
				}
				g.Attach(card.ID(), other.ID())
				g.ApplyContinuousEffects()
				if g.FindPermanent(host.ID()).HasKeyword(tc.keyword) || !g.FindPermanent(other.ID()).HasKeyword(tc.keyword) {
					t.Fatal("grant did not follow attachment")
				}
				remover := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), b.PlayerID())
				removal := mage.RemoveAllAbilities(card.ID(), core.WhileOnBattlefield)
				removal.SetSourceID(remover.ID())
				g.AddContinuousEffect(removal)
				if g.FindPermanent(other.ID()).HasKeyword(tc.keyword) {
					t.Fatal("disabled Aura still granted landwalk")
				}
				g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
				if !g.FindPermanent(other.ID()).HasKeyword(tc.keyword) {
					t.Fatal("Aura grant did not resume")
				}
				phasedHost := g.PhaseOut(other.ID())
				g.ApplyContinuousEffects()
				if g.FindPermanent(card.ID()) != nil {
					t.Fatal("Aura did not phase with host")
				}
				g.PhaseIn(phasedHost)
				g.ApplyContinuousEffects()
				if !g.FindPermanent(other.ID()).HasKeyword(tc.keyword) {
					t.Fatal("host phase cycle lost grant")
				}
				g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
				if g.FindPermanent(other.ID()).HasKeyword(tc.keyword) {
					t.Fatal("departed Aura grant remained")
				}
			})
		}
	}
}

func TestRONStaticLandwalkHiddenPath(t *testing.T) {
	g, a, b := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "DRK", "Hidden Path"), a.PlayerID())
	own := g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), a.PlayerID())
	opposing := g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), b.PlayerID())
	nongreen := g.PutOnBattlefield(repositoryCard(t, "LEB", "Merfolk of the Pearl Trident"), a.PlayerID())
	land := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), b.PlayerID())
	check := func(game *mage.Game, active bool) {
		t.Helper()
		for _, p := range []*mage.Permanent{own, opposing} {
			if game.FindPermanent(p.ID()).HasKeyword(core.Forestwalk) != active {
				t.Fatal("global grant membership mismatch")
			}
		}
		if game.FindPermanent(nongreen.ID()).HasKeyword(core.Forestwalk) || game.FindPermanent(land.ID()).HasKeyword(core.Forestwalk) {
			t.Fatal("nonmatching permanent gained landwalk")
		}
	}
	check(g, true)
	colorChange := mage.ColorOverride(own.ID(), core.Blue)
	colorChange.SetSourceID(source.ID())
	g.AddContinuousEffect(colorChange)
	if g.FindPermanent(own.ID()).HasKeyword(core.Forestwalk) {
		t.Fatal("global grant ignored current color")
	}
	colorChange = mage.ColorOverride(own.ID(), core.Green)
	colorChange.SetSourceID(source.ID())
	g.AddContinuousEffect(colorChange)
	check(g, true)
	if !mage.HasLandwalkEvasion(g.FindPermanent(own.ID()), b.PlayerID(), g) {
		t.Fatal("matching defender land did not enable evasion")
	}
	clone := g.Clone()
	clone.RemoveFromBattlefield(clone.FindPermanent(source.ID()))
	check(clone, false)
	check(g, true)
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	check(g, true)
	phased := g.PhaseOut(source.ID())
	g.ApplyContinuousEffects()
	check(g, false)
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	check(g, true)
	phased = g.PhaseOut(own.ID())
	g.ApplyContinuousEffects()
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	check(g, true)
	late := g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), a.PlayerID())
	if !late.HasKeyword(core.Forestwalk) {
		t.Fatal("late entry lacked grant")
	}
	removal := mage.RemoveAllAbilities(source.ID(), core.WhileOnBattlefield)
	removal.SetSourceID(nongreen.ID())
	g.AddContinuousEffect(removal)
	check(g, false)
	g.RemoveFromBattlefield(g.FindPermanent(nongreen.ID()))
	if !g.FindPermanent(own.ID()).HasKeyword(core.Forestwalk) {
		t.Fatal("grant did not resume")
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	if g.FindPermanent(own.ID()).HasKeyword(core.Forestwalk) {
		t.Fatal("departed source grant remained")
	}
	g.PutOnBattlefield(source.Card, a.PlayerID())
	if !g.FindPermanent(own.ID()).HasKeyword(core.Forestwalk) {
		t.Fatal("returned source did not supply fresh grant")
	}
}

func TestRONStaticLandwalkShapesAndInstances(t *testing.T) {
	for _, land := range []string{"Plains", "Island", "Swamp", "Mountain", "Forest", "Desert"} {
		t.Run(land, func(t *testing.T) {
			src := fmt.Sprintf(`Card(Name: "Walker", Typeline: (Supertypes: [], Cardtypes: [Creature], Subtypes: []), CardPT: (Power: 1, Toughness: 1), Rules: [EachPermanentLayerEffect(IsCardtype(Creature), [AdjustPT(1, 1), AddAbility([Flying, Landwalk(IsLandType(%s))])])])`, land)
			nodes, err := Parse(strings.NewReader(src), "walk.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			first, second := factory(), factory()
			if err := mage.ValidateCardDefinition("Walker", first); err != nil {
				t.Fatal(err)
			}
			one := g.PutOnBattlefield(first, a.PlayerID())
			two := g.PutOnBattlefield(second, b.PlayerID())
			keyword := core.LandwalkAttr(land)
			if one.CurrentPower(g) != 3 || two.CurrentPower(g) != 3 || !one.HasKeyword(keyword) || !two.HasKeyword(core.Flying) {
				t.Fatal("mixed grants or independent sources failed")
			}
			g.RemoveFromBattlefield(g.FindPermanent(one.ID()))
			if g.FindPermanent(two.ID()).CurrentPower(g) != 2 || !g.FindPermanent(two.ID()).HasKeyword(keyword) {
				t.Fatal("remaining instance lost its grant")
			}
		})
	}
}

func TestRONStaticLandwalkInvalidAndInventory(t *testing.T) {
	for _, ability := range []string{`Landwalk`, `Landwalk()`, `Landwalk(IsLandType(Island), IsLandType(Swamp))`, `Landwalk(IsLandType)`, `Landwalk(IsLandType())`, `Landwalk(IsLandType(Island()))`, `Landwalk(IsLandType(You))`, `Landwalk(IsCreatureType(Island))`, `Landwalk(IsSupertype(Basic))`} {
		src := fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Enchantment], Subtypes: []), Rules: [EachPermanentLayerEffect(IsCardtype(Creature), [AddAbility([%s])])])`, ability)
		nodes, err := Parse(strings.NewReader(src), "invalid-walk.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted invalid declaration %s", ability)
		}
	}
	src := `Card(Name: "Inventory", Typeline: (Supertypes: [], Cardtypes: [Enchantment], Subtypes: []), Rules: [EachPermanentLayerEffect(IsCardtype(Creature), [AddAbility([Landwalk(IsLandType(Island))])])])`
	report, err := InspectFS(fstest.MapFS{"walk.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	roles := map[string]string{"Landwalk": "keyword", "IsLandType": "landwalk_filter", "Island": "land_subtype"}
	for _, entry := range report.Signatures {
		if want, ok := roles[entry.Name]; ok {
			if entry.Role != want {
				t.Errorf("%s role = %s, want %s", entry.Name, entry.Role, want)
			}
			delete(roles, entry.Name)
		}
	}
	if len(roles) != 0 {
		t.Fatal("missing nested inventory entries")
	}
}

func TestRONStaticLandwalkControlledExclusion(t *testing.T) {
	src := `Card(Name: "Controlled walker", Typeline: (Supertypes: [], Cardtypes: [Creature], Subtypes: []), CardPT: (Power: 1, Toughness: 1), Rules: [EachPermanentLayerEffect(And([IsCardtype(Creature), Other(ThisPermanent), ControlledByAPlayer(SinglePlayer(You))]), [AddAbility([Landwalk(IsLandType(Island))])])])`
	nodes, err := Parse(strings.NewReader(src), "controlled-walk.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	own := g.PutOnBattlefield(repositoryCard(t, "LEB", "Merfolk of the Pearl Trident"), a.PlayerID())
	opposing := g.PutOnBattlefield(repositoryCard(t, "LEB", "Merfolk of the Pearl Trident"), b.PlayerID())
	if source.HasKeyword(core.Islandwalk) || !own.HasKeyword(core.Islandwalk) || opposing.HasKeyword(core.Islandwalk) {
		t.Fatal("wrong controlled source exclusion")
	}
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	if g.FindPermanent(own.ID()).HasKeyword(core.Islandwalk) || !g.FindPermanent(opposing.ID()).HasKeyword(core.Islandwalk) || g.FindPermanent(source.ID()).HasKeyword(core.Islandwalk) {
		t.Fatal("grant did not follow source control")
	}
}
