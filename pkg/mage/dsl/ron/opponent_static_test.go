package ron

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONOpponentStaticCards(t *testing.T) {
	for _, tc := range []struct {
		name, set                string
		color                    core.Color
		power, toughness, dp, dt int
	}{
		{"Water Wurm", "DRK", core.Blue, 1, 1, 0, 1},
		{"Beasts of Bogardan", "LEG", core.Red, 3, 3, 1, 1},
		{"Ivory Guardians", "LEG", core.White, 3, 3, 1, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, tc.set, tc.name)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(tc.color, 6)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(card.ID()) != nil {
				t.Fatal("entered before resolution")
			}
			g.ResolveStack()
			checkConditionalPT(t, g, card.ID(), tc.power, tc.toughness)
			var qualifying mage.Card
			switch tc.name {
			case "Water Wurm":
				qualifying = repositoryCard(t, "LEB", "Island")
			case "Beasts of Bogardan":
				qualifying = mage.NewEnchantment("White", "{W}")
			case "Ivory Guardians":
				qualifying = mage.NewArtifact("Red", "{R}")
			}
			qualifying.SetOwner(b.PlayerID())
			own := g.PutOnBattlefield(qualifying, a.PlayerID())
			checkConditionalPT(t, g, card.ID(), tc.power, tc.toughness)
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: own.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			checkConditionalPT(t, g, card.ID(), tc.power+tc.dp, tc.toughness+tc.dt)
			clone := g.Clone()
			phased := g.PhaseOut(own.ID())
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, card.ID(), tc.power, tc.toughness)
			checkConditionalPT(t, clone, card.ID(), tc.power+tc.dp, tc.toughness+tc.dt)
			g.PhaseIn(phased)
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, card.ID(), tc.power+tc.dp, tc.toughness+tc.dt)
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: card.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			checkConditionalPT(t, g, card.ID(), tc.power, tc.toughness)
			g.BouncePermanentToHand(g.FindPermanent(own.ID()))
			b.RemoveFromHand(qualifying.ID())
			returned := g.PutOnBattlefield(qualifying, a.PlayerID())
			checkConditionalPT(t, g, card.ID(), tc.power+tc.dp, tc.toughness+tc.dt)
			g.RemoveFromBattlefield(returned)
			checkConditionalPT(t, g, card.ID(), tc.power, tc.toughness)
		})
	}
}

func TestRONOpponentStaticNonTokenAndColor(t *testing.T) {
	for _, name := range []string{"Beasts of Bogardan", "Ivory Guardians"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, "LEG", name), a.PlayerID())
			color := core.White
			if name == "Ivory Guardians" {
				color = core.Red
			}
			token := mage.DefineToken("Token", mage.TokenPT(1, 1), mage.TokenColors(color))
			if err := mage.ApplyEffect(g, mage.CreateTokenCopies(mage.You, mage.Fixed(1), token), uuid.Nil, b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, source.ID(), 3, 3)
			permanent := g.PutOnBattlefield(mage.NewEnchantment("Blue", "{U}"), b.PlayerID())
			checkConditionalPT(t, g, source.ID(), 3, 3)
			if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetColors(mage.This, color, core.Blue)), permanent.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, source.ID(), 4, 4)
			g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.EndOfTurn))
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, source.ID(), 3, 3)
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, source.ID(), 3, 3)
		})
	}
}

func TestRONOpponentStaticIvoryMembershipAndInstances(t *testing.T) {
	g, a, b := ronGame()
	first := g.PutOnBattlefield(repositoryCard(t, "LEG", "Ivory Guardians"), a.PlayerID())
	second := g.PutOnBattlefield(repositoryCard(t, "LEG", "Ivory Guardians"), a.PlayerID())
	other := g.PutOnBattlefield(repositoryCard(t, "LEG", "Ivory Guardians"), b.PlayerID())
	unrelated := g.PutOnBattlefield(mage.NewCreature("Unrelated", "{W}", 3, 3), a.PlayerID())
	red := g.PutOnBattlefield(mage.NewEnchantment("Red", "{R}"), b.PlayerID())
	for _, id := range []uuid.UUID{first.ID(), second.ID(), other.ID()} {
		checkConditionalPT(t, g, id, 5, 5)
	}
	checkConditionalPT(t, g, unrelated.ID(), 3, 3)
	later := g.PutOnBattlefield(repositoryCard(t, "LEG", "Ivory Guardians"), b.PlayerID())
	checkConditionalPT(t, g, later.ID(), 5, 5)
	phased := g.PhaseOut(first.ID())
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, second.ID(), 4, 4)
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, second.ID(), 5, 5)
	g.RemoveFromBattlefield(first)
	checkConditionalPT(t, g, second.ID(), 4, 4)
	g.PutOnBattlefield(repositoryCard(t, "LEG", "Ivory Guardians"), a.PlayerID())
	checkConditionalPT(t, g, second.ID(), 5, 5)
	g.RemoveFromBattlefield(red)
	for _, id := range []uuid.UUID{second.ID(), other.ID(), later.ID()} {
		checkConditionalPT(t, g, id, 3, 3)
	}
}

func TestRONOpponentStaticProtection(t *testing.T) {
	for _, name := range []string{"Beasts of Bogardan", "Ivory Guardians"} {
		g, a, b := ronGame()
		source := g.PutOnBattlefield(repositoryCard(t, "LEG", name), a.PlayerID())
		red := g.PutOnBattlefield(mage.NewCreature("Red", "{R}", 1, 1), b.PlayerID())
		g.DealDamageToPermanent(g.FindPermanent(source.ID()), 2, red.ID())
		if g.FindPermanent(source.ID()).Damage != 0 {
			t.Fatal("protection did not prevent red damage")
		}
		bolt := repositoryCard(t, "LEB", "Lightning Bolt")
		bolt.SetOwner(b.PlayerID())
		b.AddToHand(bolt)
		b.ManaPool().Add(core.Red, 1)
		if err := g.CastSpellByID(b.PlayerID(), bolt.ID(), []uuid.UUID{source.ID()}, 0); err == nil {
			t.Fatal("red spell targeted protection")
		}
		if b.ManaPool().Count(core.Red) != 1 || len(b.Hand()) != 1 || !g.Stack().IsEmpty() {
			t.Fatal("illegal target paid cost")
		}
		if mage.CanBlock(red, g.FindPermanent(source.ID()), g) {
			t.Fatal("red creature blocked protection")
		}
		blue := g.PutOnBattlefield(mage.NewCreature("Blue", "{U}", 1, 1), b.PlayerID())
		g.DealDamageToPermanent(g.FindPermanent(source.ID()), 1, blue.ID())
		if g.FindPermanent(source.ID()).Damage != 1 || !mage.CanBlock(blue, g.FindPermanent(source.ID()), g) {
			t.Fatal("protection restricted a blue source")
		}
		aura := repositoryCard(t, "LEB", "Firebreathing")
		aura.SetOwner(b.PlayerID())
		b.AddToHand(aura)
		if err := g.CastSpellByID(b.PlayerID(), aura.ID(), []uuid.UUID{source.ID()}, 0); err == nil {
			t.Fatal("red Aura targeted protection")
		}
	}
}

func TestRONOpponentStaticLayersAndFactories(t *testing.T) {
	g, a, b := ronGame()
	factory := permanentScopeFactory(t, "Artifact", `If(APlayerPassesFilter(Opponent, ControlsA(And([IsNonToken, Or([IsColor(Red), IsLandType(Island)])]))), [PermanentLayerEffect(ThisPermanent, [AddAbility([Flying])])])`)
	first, second := g.PutOnBattlefield(factory(), a.PlayerID()), g.PutOnBattlefield(factory(), b.PlayerID())
	island := g.PutOnBattlefield(repositoryCard(t, "LEB", "Tropical Island"), b.PlayerID())
	wurm := g.PutOnBattlefield(repositoryCard(t, "DRK", "Water Wurm"), a.PlayerID())
	if !g.FindPermanent(first.ID()).HasKeyword(core.Flying) || g.FindPermanent(second.ID()).HasKeyword(core.Flying) {
		t.Fatal("factory conditions shared controller state")
	}
	checkConditionalPT(t, g, wurm.ID(), 1, 2)
	moon := g.PutOnBattlefield(repositoryCard(t, "DRK", "Blood Moon"), a.PlayerID())
	checkConditionalPT(t, g, wurm.ID(), 1, 1)
	if g.FindPermanent(first.ID()).HasKeyword(core.Flying) {
		t.Fatal("condition read land types before layer 4")
	}
	g.RemoveFromBattlefield(moon)
	checkConditionalPT(t, g, wurm.ID(), 1, 2)
	clone := g.Clone()
	g.BouncePermanentToHand(g.FindPermanent(wurm.ID()))
	a.RemoveFromHand(wurm.Card.ID())
	returned := g.PutOnBattlefield(wurm.Card, b.PlayerID())
	checkConditionalPT(t, g, returned.ID(), 1, 1)
	checkConditionalPT(t, clone, wurm.ID(), 1, 2)
	g.RemoveFromBattlefield(island)
	if g.FindPermanent(first.ID()).HasKeyword(core.Flying) || !clone.FindPermanent(first.ID()).HasKeyword(core.Flying) {
		t.Fatal("clone or live collection membership")
	}
}

func TestRONOpponentStaticInvalid(t *testing.T) {
	for _, condition := range []string{
		`APlayerPassesFilter(You, ControlsA(AnyPermanent))`,
		`APlayerPassesFilter(AnyPlayer, ControlsA(AnyPermanent))`,
		`APlayerPassesFilter(Opponent(), ControlsA(AnyPermanent))`,
		`APlayerPassesFilter(Ref_TargetPlayer, ControlsA(AnyPermanent))`,
		`APlayerPassesFilter(Opponent, ControlsNo(AnyPermanent))`,
		`APlayerPassesFilter(Opponent, ControlsA(IsNonToken()))`,
		`APlayerPassesFilter(Opponent, ControlsA(And([])))`,
		`APlayerPassesFilter(Opponent, ControlsA(HasAbility(Flying)))`,
		`APlayerPassesFilter(Opponent, ControlsA(PowerIs(EqualTo(Integer(2)))))`,
		`APlayerPassesFilter(Opponent, ControlsA(ControlledByAPlayer(SinglePlayer(You))))`,
		`APlayerPassesFilter(Opponent, ControlsA(AnyPermanent), AnyPermanent)`,
	} {
		assertConditionalRejected(t, fmt.Sprintf(`If(%s, [PermanentLayerEffect(ThisPermanent, [AdjustPT(1,1)])])`, condition))
	}
}

func TestRONOpponentStaticInventory(t *testing.T) {
	src := `Card(Name: "Inventory", Typeline: (Supertypes: [], Cardtypes: [Creature], Subtypes: []), CardPT: (Power: 1, Toughness: 1), Rules: [If(APlayerPassesFilter(Opponent, ControlsA(And([IsNonToken, IsColor(White)]))), [PermanentLayerEffect(ThisPermanent, [AdjustPT(1, 1)])])])`
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"Opponent": "player_filter", "ControlsA": "static_player_filter", "IsNonToken": "permanent_filter", "IsColor": "permanent_filter"} {
		found := false
		for _, sig := range report.Signatures {
			if sig.Name == name && sig.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing %s/%s", name, role)
		}
	}
	if len(report.Issues) != 0 {
		t.Fatal(report.Issues)
	}
}

func TestRONOpponentStaticStrictCollection(t *testing.T) {
	names := map[string]bool{"Water Wurm": true, "Beasts of Bogardan": true, "Ivory Guardians": true}
	files, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil {
		t.Fatal(err)
	}
	collection := fstest.MapFS{}
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
			if !names[name] {
				continue
			}
			count++
			factory, err := Compile(node)
			if err != nil {
				t.Fatal(err)
			}
			for range 2 {
				if err := mage.ValidateCardDefinition(name, factory()); err != nil {
					t.Fatal(err)
				}
			}
		}
		var selected []string
		for line := range strings.SplitSeq(string(data), "\n") {
			for name := range names {
				if strings.Contains(line, `Name: "`+name+`"`) {
					selected = append(selected, line)
				}
			}
		}
		if len(selected) > 0 {
			collection[filepath.Base(file)] = &fstest.MapFile{Data: []byte(strings.Join(selected, "\n"))}
		}
	}
	registry, err := LoadFS(collection, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 || len(registry.RegisteredCardNames()) != 3 {
		t.Fatalf("records=%d names=%d", count, len(registry.RegisteredCardNames()))
	}
	collection["invalid.ron"] = &fstest.MapFile{Data: []byte(`Card(Name: "Gap", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [Unknown])`)}
	if registry, err := LoadFS(collection, "*.ron"); err == nil || registry != nil {
		t.Fatal("published incomplete collection")
	}
}
