package ron

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func textLandCheck(t *testing.T, g *mage.Game, land *mage.Permanent, subtype string, color core.Color) {
	t.Helper()
	p := g.FindPermanent(land.ID())
	if p == nil || !p.HasSubType(subtype) || !p.HasType(core.TypeLand) {
		t.Fatalf("want land type %s: %v", subtype, p)
	}
	for _, other := range []string{"Plains", "Island", "Swamp", "Mountain", "Forest"} {
		if other != subtype && p.HasSubType(other) {
			t.Fatalf("old land type retained: %s", other)
		}
	}
	if !g.CanAfford(p.ControllerID(), core.ParseManaCost("{"+map[core.Color]string{core.White: "W", core.Blue: "U", core.Black: "B", core.Red: "R", core.Green: "G"}[color]+"}"), nil) {
		t.Fatal("intrinsic mana unavailable", color)
	}
}

func TestRONTextStaticLandTypeRecords(t *testing.T) {
	files, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), file)
		if err != nil {
			t.Fatal(err)
		}
		for _, node := range nodes {
			name := node.Fields["Name"].String
			if name != "Conversion" && name != "Blood Moon" {
				continue
			}
			count++
			t.Run(filepath.Base(file)+"/"+name, func(t *testing.T) {
				factory, err := Compile(node)
				if err != nil {
					t.Fatal(err)
				}
				g, a, b := ronGame()
				landCard := repositoryCard(t, "ATQ", "Strip Mine")
				if name == "Conversion" {
					landCard = repositoryCard(t, "LEA", "Mountain")
				}
				land := g.PutOnBattlefield(landCard, b.PlayerID())
				card := factory()
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
					t.Fatal("unpaid casting succeeded")
				}
				if g.StackSize() != 0 || len(a.Hand()) != 1 || land.HasSubType("Plains") {
					t.Fatal("failed casting changed state")
				}
				color, amount, from := core.Red, 1, core.MountainLandType
				if name == "Conversion" {
					color, amount, from = core.White, 2, core.PlainsLandType
				}
				a.ManaPool().Add(color, amount)
				a.ManaPool().Add(core.Colorless, 2)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				if name == "Blood Moon" && g.FindPermanent(land.ID()).HasSubType("Mountain") {
					t.Fatal("rule applied before entry")
				}
				if name == "Conversion" && g.FindPermanent(land.ID()).HasSubType("Plains") {
					t.Fatal("rule applied before entry")
				}
				g.ResolveStack()
				source := g.FindPermanent(card.ID())
				id := textStaticID(t, source)
				change, _ := mage.ReplaceBasicLandWord(from, core.IslandLandType)
				if err := g.RevisePrintedStaticLandType(source, id, change); err != nil {
					t.Fatal(err)
				}
				if textStaticID(t, g.FindPermanent(source.ID())) != id || source.Card.Name() != name || source.Card.ManaCost().String() != factory().ManaCost().String() {
					t.Fatal("printed identity or metadata changed")
				}
				textLandCheck(t, g, land, "Island", core.Blue)
				if name == "Blood Moon" {
					if g.FindPermanent(land.ID()).Card.HasSuperType(core.SuperBasic) {
						t.Fatal("type setting added basic supertype")
					}
					target := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), a.PlayerID())
					if err := g.ActivateAbilityByIndex(b.PlayerID(), land.ID(), 1, []uuid.UUID{target.ID()}); err == nil {
						t.Fatal("removed Strip Mine ability activated")
					}
					if g.FindPermanent(land.ID()).Tapped || g.StackSize() != 0 {
						t.Fatal("failed activation paid costs")
					}
				}
				clone := g.Clone()
				next, _ := mage.ReplaceBasicLandWord(core.IslandLandType, core.SwampLandType)
				if err := clone.RevisePrintedStaticLandType(clone.FindPermanent(source.ID()), id, next); err != nil {
					t.Fatal(err)
				}
				textLandCheck(t, clone, land, "Swamp", core.Black)
				textLandCheck(t, g, land, "Island", core.Blue)
				lateCard := repositoryCard(t, "ATQ", "Strip Mine")
				if name == "Conversion" {
					lateCard = repositoryCard(t, "LEA", "Mountain")
				}
				late := g.PutOnBattlefield(lateCard, b.PlayerID())
				textLandCheck(t, g, late, "Island", core.Blue)
				phased := g.PhaseOut(source.ID())
				g.ApplyContinuousEffects()
				if g.FindPermanent(land.ID()).HasSubType("Island") {
					t.Fatal("phased source retained rule")
				}
				if err := g.RevisePrintedStaticLandType(source, id, next); err == nil {
					t.Fatal("phased source accepted")
				}
				g.PhaseIn(phased)
				g.ApplyContinuousEffects()
				textLandCheck(t, g, land, "Island", core.Blue)
				phased = g.PhaseOut(land.ID())
				g.PhaseIn(phased)
				g.ApplyContinuousEffects()
				textLandCheck(t, g, land, "Island", core.Blue)
				remover := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), a.PlayerID())
				removal := mage.RemoveAllAbilities(source.ID(), core.WhileOnBattlefield)
				removal.SetSourceID(remover.ID())
				g.AddContinuousEffect(removal)
				textLandCheck(t, g, land, "Island", core.Blue)
				if err := g.RevisePrintedStaticLandType(g.FindPermanent(source.ID()), id, next); err == nil {
					t.Fatal("removed declaration accepted")
				}
				g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
				if err := g.RevisePrintedStaticLandType(g.FindPermanent(source.ID()), id, next); err != nil {
					t.Fatal(err)
				}
				textLandCheck(t, g, land, "Swamp", core.Black)
				g.BouncePermanentToHand(g.FindPermanent(late.ID()))
				if lki := g.LKI(late.ID()); lki == nil || !lki.Snapshot.HasSubType("Swamp") {
					t.Fatal("departure lost revised type")
				}
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				if g.FindPermanent(land.ID()).HasSubType("Swamp") {
					t.Fatal("departed source retained rule")
				}
				returned := g.PutOnBattlefield(card, a.PlayerID())
				if err := g.RevisePrintedStaticLandType(source, id, change); err == nil {
					t.Fatal("old source visit accepted")
				}
				if !g.FindPermanent(land.ID()).HasSubType(from.String()) || textStaticID(t, returned) != id {
					t.Fatal("return did not restore printed declaration")
				}
			})
		}
	}
	if count != 7 {
		t.Fatal("record coverage", count)
	}
}

func TestRONTextStaticLandTypeDependencies(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		g, a, b := ronGame()
		land := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Strip Mine"), b.PlayerID())
		basic := g.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), b.PlayerID())
		var conversion, moon *mage.Permanent
		addConversion := func() { conversion = g.PutOnBattlefield(repositoryCard(t, "LEA", "Conversion"), a.PlayerID()) }
		addMoon := func() { moon = g.PutOnBattlefield(repositoryCard(t, "DRK", "Blood Moon"), a.PlayerID()) }
		if reverse {
			addMoon()
			addConversion()
		} else {
			addConversion()
			addMoon()
		}
		change, _ := mage.ReplaceBasicLandWord(core.MountainLandType, core.IslandLandType)
		if err := g.RevisePrintedStaticLandType(conversion, textStaticID(t, conversion), change); err != nil {
			t.Fatal(err)
		}
		textLandCheck(t, g, land, "Mountain", core.Red)
		textLandCheck(t, g, basic, "Mountain", core.Red)
		if err := g.RevisePrintedStaticLandType(g.FindPermanent(moon.ID()), textStaticID(t, moon), change); err != nil {
			t.Fatal(err)
		}
		for range 3 {
			g.ApplyContinuousEffects()
			textLandCheck(t, g, land, "Plains", core.White)
			textLandCheck(t, g, basic, "Mountain", core.Red)
		}
		output, _ := mage.ReplaceBasicLandWord(core.PlainsLandType, core.SwampLandType)
		if err := g.RevisePrintedStaticLandType(g.FindPermanent(conversion.ID()), textStaticID(t, conversion), output); err != nil {
			t.Fatal(err)
		}
		textLandCheck(t, g, land, "Swamp", core.Black)
		textLandCheck(t, g, basic, "Mountain", core.Red)
		g.RemoveFromBattlefield(g.FindPermanent(moon.ID()))
		if len(g.FindPermanent(land.ID()).SubTypes()) != 0 {
			t.Fatal("revised selection followed an absent member")
		}
	}
}

func TestRONTextStaticLandTypeBoundaries(t *testing.T) {
	change, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	for _, tc := range []struct{ set, name string }{{"LEA", "Living Lands"}} {
		g, a, _ := ronGame()
		source := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
		if err := g.RevisePrintedStaticLandType(source, textStaticID(t, source), change); err == nil {
			t.Fatal("unmarked context accepted", tc.name)
		}
	}
	for _, name := range []string{"Magical Hack", "Sleight of Mind"} {
		if _, err := Compile(staticBaseRecord(t, "LEA", name)); err == nil {
			t.Fatal("dependent card accepted", name)
		}
	}
	TestRONStaticLandTypeInvalidAndInventory(t)
}

func TestRONTextStaticLandTypeDeclarations(t *testing.T) {
	factory := permanentScopeFactory(t, "Enchantment", `EachPermanentLayerEffect(IsLandType(Mountain),[SetLandType(Plains)]),EachPermanentLayerEffect(IsLandType(Forest),[SetLandType(Plains)])`)
	g, a, b := ronGame()
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	mountain := g.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), a.PlayerID())
	forest := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
	first, second := source.RuntimeAbilities[0].AbilityID(), source.RuntimeAbilities[1].AbilityID()
	change, _ := mage.ReplaceBasicLandWord(core.PlainsLandType, core.IslandLandType)
	if err := g.RevisePrintedStaticLandType(source, first, change); err != nil {
		t.Fatal(err)
	}
	textLandCheck(t, g, mountain, "Island", core.Blue)
	textLandCheck(t, g, forest, "Plains", core.White)
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	textLandCheck(t, g, mountain, "Island", core.Blue)
	textLandCheck(t, g, forest, "Plains", core.White)
	output, _ := mage.ReplaceBasicLandWord(core.PlainsLandType, core.SwampLandType)
	if err := g.RevisePrintedStaticLandType(g.FindPermanent(source.ID()), second, output); err != nil {
		t.Fatal(err)
	}
	textLandCheck(t, g, mountain, "Island", core.Blue)
	textLandCheck(t, g, forest, "Swamp", core.Black)
	other := factory()
	for _, ability := range other.Abilities() {
		if ability.AbilityID() == first || ability.AbilityID() == second {
			t.Fatal("factory shared declaration identity")
		}
	}
	if err := g.RevisePrintedStaticLandType(g.FindPermanent(source.ID()), other.Abilities()[0].AbilityID(), change); err == nil {
		t.Fatal("foreign declaration accepted")
	}
}

func TestRONTextStaticLandTypeUpkeep(t *testing.T) {
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	for range 2 {
		g.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), a.PlayerID())
	}
	source := g.PutOnBattlefield(repositoryCard(t, "LEA", "Conversion"), a.PlayerID())
	id := textStaticID(t, source)
	change, _ := mage.ReplaceBasicLandWord(core.PlainsLandType, core.IslandLandType)
	if err := g.RevisePrintedStaticLandType(source, id, change); err != nil {
		t.Fatal(err)
	}
	queueRONUpkeep(g, b)
	g.ResolveStack()
	if a.offers != 0 {
		t.Fatal("opponent upkeep offered payment")
	}
	a.ManaPool().Add(core.White, 2)
	queueRONUpkeep(g, a)
	g.ResolveStack()
	if a.offers != 1 || g.FindPermanent(source.ID()) == nil || a.ManaPool().TotalMana() != 0 {
		t.Fatal("unrelated upkeep declaration changed")
	}
	a.accept = false
	queueRONUpkeep(g, a)
	g.ResolveStack()
	if g.FindPermanent(source.ID()) != nil {
		t.Fatal("unpaid revised source survived")
	}
	for _, land := range g.FilterBattlefield(mage.AnyPermanent) {
		if !land.HasSubType("Mountain") {
			t.Fatal("sacrifice retained revised rule")
		}
	}
}
