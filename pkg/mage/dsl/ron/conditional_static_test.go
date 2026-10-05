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

func checkConditionalPT(t *testing.T, g *mage.Game, id uuid.UUID, power, toughness int) {
	t.Helper()
	p := g.FindPermanent(id)
	if p == nil || p.CurrentPower(g) != power || p.CurrentToughness(g) != toughness {
		t.Fatalf("P/T for %s = %v; want %d/%d", id, p, power, toughness)
	}
}

func TestRONConditionalStaticController(t *testing.T) {
	for _, tc := range []struct {
		name, set, land  string
		power, toughness int
	}{
		{"Kird Ape", "ARN", "Forest", 2, 3}, {"Sedge Troll", "LEB", "Swamp", 3, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, tc.set, tc.name)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Red, 3)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(card.ID()) != nil {
				t.Fatal("static source entered before resolution")
			}
			g.ResolveStack()
			source := g.FindPermanent(card.ID())
			checkConditionalPT(t, g, source.ID(), card.Power(), card.Toughness())
			if tc.name == "Sedge Troll" {
				a.ManaPool().Clear()
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, nil); err == nil {
					t.Fatal("unpaid regeneration accepted")
				}
				if !g.Stack().IsEmpty() {
					t.Fatal("failed cost changed stack")
				}
			}
			land := g.PutOnBattlefield(repositoryCard(t, "LEB", tc.land), b.PlayerID())
			checkConditionalPT(t, g, source.ID(), card.Power(), card.Toughness())
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: land.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			checkConditionalPT(t, g, source.ID(), tc.power, tc.toughness)
			clone := g.Clone()
			phased := g.PhaseOut(land.ID())
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, source.ID(), card.Power(), card.Toughness())
			checkConditionalPT(t, clone, source.ID(), tc.power, tc.toughness)
			g.PhaseIn(phased)
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, source.ID(), tc.power, tc.toughness)
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			checkConditionalPT(t, g, source.ID(), card.Power(), card.Toughness())
			g.PutOnBattlefield(repositoryCard(t, "LEB", tc.land), b.PlayerID())
			checkConditionalPT(t, g, source.ID(), tc.power, tc.toughness)
			if tc.name == "Sedge Troll" {
				b.ManaPool().Add(core.Black, 1)
				if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 1, nil); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				g.DestroyPermanent(g.FindPermanent(source.ID()))
				if g.FindPermanent(source.ID()) == nil || !g.FindPermanent(source.ID()).Tapped || b.ManaPool().TotalMana() != 0 {
					t.Fatal("regeneration or payment")
				}
				checkConditionalPT(t, g, source.ID(), tc.power, tc.toughness)
			}
		})
	}
}

func TestRONConditionalStaticUntapped(t *testing.T) {
	g, a, _ := ronGame()
	card := repositoryCard(t, "ARN", "Giant Tortoise")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Blue, 2)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	checkConditionalPT(t, g, card.ID(), 1, 4)
	clone := g.Clone()
	g.MutablePermanent(card.ID()).Tapped = true
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, card.ID(), 1, 1)
	checkConditionalPT(t, clone, card.ID(), 1, 4)
	g.MutablePermanent(card.ID()).Tapped = false
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, card.ID(), 1, 4)
	g.MutablePermanent(card.ID()).AddCounter(core.P1P1, 1)
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, card.ID(), 2, 5)
	phased := g.PhaseOut(card.ID())
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, card.ID(), 2, 5)
	g.BouncePermanentToHand(g.FindPermanent(card.ID()))
	a.RemoveFromHand(card.ID())
	returned := g.PutOnBattlefield(card, a.PlayerID())
	checkConditionalPT(t, g, returned.ID(), 1, 4)
}

func TestRONConditionalStaticAura(t *testing.T) {
	for _, scenario := range []string{"legal", "unpaid", "wrong-type", "departed", "phased"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			hostCard := mage.NewCreature("Host", "", 2, 2)
			if scenario == "wrong-type" {
				hostCard = mage.NewLand("Land")
			}
			hostCard.SetOwner(b.PlayerID())
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			card := repositoryCard(t, "LEG", "Spectral Cloak")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if scenario != "unpaid" {
				a.ManaPool().Add(core.Blue, 2)
			}
			before := a.ManaPool().TotalMana()
			err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0)
			if scenario == "unpaid" || scenario == "wrong-type" {
				if err == nil || a.ManaPool().TotalMana() != before || len(a.Hand()) != 1 || !g.Stack().IsEmpty() {
					t.Fatal("illegal cast changed state")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if host.HasKeyword(core.Shroud) {
				t.Fatal("grant before resolution")
			}
			if scenario == "departed" {
				g.BouncePermanentToHand(host)
			}
			if scenario == "phased" {
				g.PhaseOut(host.ID())
			}
			g.ResolveStack()
			if scenario != "legal" {
				if g.FindPermanent(card.ID()) != nil {
					t.Fatal("illegal Aura resolved")
				}
				return
			}
			if !g.FindPermanent(host.ID()).HasKeyword(core.Shroud) {
				t.Fatal("missing shroud")
			}
			bolt := repositoryCard(t, "LEB", "Lightning Bolt")
			bolt.SetOwner(a.PlayerID())
			a.AddToHand(bolt)
			a.ManaPool().Add(core.Red, 1)
			if err := g.CastSpellByID(a.PlayerID(), bolt.ID(), []uuid.UUID{host.ID()}, 0); err == nil {
				t.Fatal("shroud did not restrict targets")
			}
			if a.ManaPool().Count(core.Red) != 1 || !g.Stack().IsEmpty() {
				t.Fatal("illegal target paid cost")
			}
			clone := g.Clone()
			g.MutablePermanent(host.ID()).Tapped = true
			g.ApplyContinuousEffects()
			if g.FindPermanent(host.ID()).HasKeyword(core.Shroud) || !clone.FindPermanent(host.ID()).HasKeyword(core.Shroud) {
				t.Fatal("tap or clone isolation")
			}
			g.MutablePermanent(host.ID()).Tapped = false
			g.ApplyContinuousEffects()
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 2), a.PlayerID())
			g.Attach(card.ID(), other.ID())
			if g.FindPermanent(host.ID()).HasKeyword(core.Shroud) || !g.FindPermanent(other.ID()).HasKeyword(core.Shroud) {
				t.Fatal("attachment did not update")
			}
			phased := g.PhaseOut(card.ID())
			g.ApplyContinuousEffects()
			if g.FindPermanent(other.ID()).HasKeyword(core.Shroud) {
				t.Fatal("phased source grants shroud")
			}
			g.PhaseIn(phased)
			g.ApplyContinuousEffects()
			if !g.FindPermanent(other.ID()).HasKeyword(core.Shroud) {
				t.Fatal("phase cycle lost grant")
			}
			g.BouncePermanentToHand(other)
			a.RemoveFromHand(other.Card.ID())
			returned := g.PutOnBattlefield(other.Card, a.PlayerID())
			if returned.HasKeyword(core.Shroud) {
				t.Fatal("grant followed returned host")
			}
		})
	}
}

func TestRONConditionalStaticCollection(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEG", "Angelic Voices")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.White, 4)
	own := g.PutOnBattlefield(mage.NewCreature("White", "{W}", 2, 2), a.PlayerID())
	opposing := g.PutOnBattlefield(mage.NewCreature("Opponent", "{W}", 2, 2), b.PlayerID())
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	checkConditionalPT(t, g, own.ID(), 3, 3)
	checkConditionalPT(t, g, opposing.ID(), 2, 2)
	artifact := g.PutOnBattlefield(mage.NewCreature("Artifact", "", 2, 2, mage.WithCardType(core.TypeArtifact)), a.PlayerID())
	checkConditionalPT(t, g, artifact.ID(), 3, 3)
	bad := g.PutOnBattlefield(mage.NewCreature("Nonwhite", "{U}", 2, 2), a.PlayerID())
	checkConditionalPT(t, g, own.ID(), 2, 2)
	checkConditionalPT(t, g, bad.ID(), 2, 2)
	clone := g.Clone()
	phased := g.PhaseOut(bad.ID())
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, own.ID(), 3, 3)
	checkConditionalPT(t, clone, own.ID(), 2, 2)
	later := g.PutOnBattlefield(mage.NewCreature("Later", "{W}", 1, 1), a.PlayerID())
	checkConditionalPT(t, g, later.ID(), 2, 2)
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, own.ID(), 2, 2)
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: bad.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	checkConditionalPT(t, g, own.ID(), 3, 3)
	g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
	checkConditionalPT(t, g, own.ID(), 2, 2)
}

func TestRONConditionalStaticInvalid(t *testing.T) {
	body := `[PermanentLayerEffect(ThisPermanent, [AdjustPT(1, 1)])]`
	for _, condition := range []string{`PlayerPassesFilter(You, ControlsA(PowerIs(EqualTo(Integer(2)))))`, `PlayerPassesFilter(You, ControlsNo(HasAbility(Flying)))`, `PlayerPassesFilter(You(), ControlsA(IsCardtype(Land)))`, `PlayerPassesFilter(Ref_TargetPlayer, ControlsA(IsCardtype(Land)))`, `PlayerPassesFilter(Trigger_ThatPlayer, ControlsA(IsCardtype(Land)))`, `PlayerPassesFilter(You, ControlsA())`, `PlayerPassesFilter(You, ControlsA(IsCardtype(Land), IsTapped))`, `PlayerPassesFilter(You, ControlsA(ControlledByAPlayer(SinglePlayer(You))))`, `PermanentPassesFilter(HostPermanent, IsUntapped)`, `PermanentPassesFilter(Ref_TargetPermanent, IsUntapped)`, `PermanentPassesFilter(ThisPermanent(), IsUntapped)`, `PermanentPassesFilter(ThisPermanent, IsUntapped())`, `APlayerPassesFilter(Opponent, ControlsNo(IsCardtype(Land)))`} {
		assertConditionalRejected(t, fmt.Sprintf(`If(%s, %s)`, condition, body))
	}
	for _, invalid := range []string{`[]`, `[Activated(PayMana([]), ActionList([GainLife(Integer(1))]))]`, `[PermanentLayerEffect(Ref_TargetPermanent,[AdjustPT(1,1)])]`, `[PermanentLayerEffect(ThisPermanent,[])]`, `[PermanentLayerEffect(ThisPermanent,[SetPT(1,1)])]`, `[PermanentLayerEffect(ThisPermanent,[AddAbility([])])]`, `[PermanentLayerEffect(ThisPermanent,[AddAbility([Flying()])])]`, `[PermanentLayerEffect(ThisPermanent,[AddAbility([Flying]),AdjustPT(1,1)])]`, `[If(PlayerPassesFilter(You,ControlsA(IsCardtype(Land))),[PermanentLayerEffect(ThisPermanent,[AdjustPT(1,1)])])]`} {
		assertConditionalRejected(t, fmt.Sprintf(`If(PlayerPassesFilter(You, ControlsA(IsCardtype(Land))), %s)`, invalid))
	}
}

func assertConditionalRejected(t *testing.T, rule string) {
	t.Helper()
	src := fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Creature], Subtypes: []), CardPT: (Power: 1, Toughness: 1), Rules: [%s])`, rule)
	nodes, err := Parse(strings.NewReader(src), "invalid-conditional.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Compile(nodes[0]); err == nil {
		t.Fatalf("accepted %s", rule)
	}
}

func TestRONConditionalStaticStrictReprints(t *testing.T) {
	names := map[string]bool{"Kird Ape": true, "Sedge Troll": true, "Giant Tortoise": true, "Spectral Cloak": true, "Angelic Voices": true}
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
		var selected []string
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
	if count != 11 || len(registry.RegisteredCardNames()) != 5 {
		t.Fatalf("records=%d names=%d", count, len(registry.RegisteredCardNames()))
	}
	collection["invalid.ron"] = &fstest.MapFile{Data: []byte(`Card(Name: "Gap", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [Unknown])`)}
	if registry, err := LoadFS(collection, "*.ron"); err == nil || registry != nil {
		t.Fatal("published incomplete collection")
	}
}

func TestRONConditionalStaticInventory(t *testing.T) {
	src := `Card(Name: "Inventory", Typeline: (Supertypes: [], Cardtypes: [Creature], Subtypes: []), CardPT: (Power: 1, Toughness: 1), Rules: [If(PlayerPassesFilter(You, ControlsA(IsLandType(Forest))), [PermanentLayerEffect(ThisPermanent, [AdjustPT(1, 2)])])])`
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"PlayerPassesFilter": "static_condition", "ControlsA": "static_player_filter", "IsLandType": "permanent_filter", "PermanentLayerEffect": "rule", "AdjustPT": "static_modification"} {
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
}

func TestRONConditionalStaticLayerInteractions(t *testing.T) {
	for _, order := range []bool{false, true} {
		g, a, b := ronGame()
		var ape, moon *mage.Permanent
		putApe := func() { ape = g.PutOnBattlefield(repositoryCard(t, "ARN", "Kird Ape"), a.PlayerID()) }
		putMoon := func() { moon = g.PutOnBattlefield(repositoryCard(t, "DRK", "Blood Moon"), b.PlayerID()) }
		dual := g.PutOnBattlefield(repositoryCard(t, "LEB", "Tropical Island"), a.PlayerID())
		if order {
			putApe()
			putMoon()
		} else {
			putMoon()
			putApe()
		}
		checkConditionalPT(t, g, ape.ID(), 1, 1)
		g.RemoveFromBattlefield(moon)
		checkConditionalPT(t, g, ape.ID(), 2, 3)
		g.AddContinuousEffect(mage.RemoveAllAbilities(ape.ID(), core.EndOfTurn))
		g.ApplyContinuousEffects()
		checkConditionalPT(t, g, ape.ID(), 1, 1)
		g.GetEffects().RemoveEndOfTurn()
		g.ApplyContinuousEffects()
		checkConditionalPT(t, g, ape.ID(), 2, 3)
		g.RemoveFromBattlefield(dual)
		checkConditionalPT(t, g, ape.ID(), 1, 1)
	}
}

func TestRONConditionalStaticFactoriesAndBodies(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `If(PlayerPassesFilter(You, ControlsA(IsCardtype(Land))), [PermanentLayerEffect(ThisPermanent, [AddAbility([Flying, Vigilance])]), EachPermanentLayerEffect(And([IsCardtype(Creature), ControlledByAPlayer(SinglePlayer(You)), Other(ThisPermanent)]), [AddAbility([Banding])])])`)
	g, a, b := ronGame()
	first := g.PutOnBattlefield(factory(), a.PlayerID())
	second := g.PutOnBattlefield(factory(), b.PlayerID())
	own := g.PutOnBattlefield(mage.NewCreature("Own", "", 2, 2), a.PlayerID())
	other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 2), b.PlayerID())
	land := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
	for range 3 {
		g.ApplyContinuousEffects()
	}
	if !g.FindPermanent(first.ID()).HasKeyword(core.Flying) || !g.FindPermanent(first.ID()).HasKeyword(core.Vigilance) || g.FindPermanent(first.ID()).HasKeyword(core.Banding) || !g.FindPermanent(own.ID()).HasKeyword(core.Banding) || g.FindPermanent(second.ID()).HasKeyword(core.Flying) || g.FindPermanent(other.ID()).HasKeyword(core.Banding) {
		t.Fatal("factory or body membership")
	}
	clone := g.Clone()
	g.RemoveFromBattlefield(land)
	if g.FindPermanent(first.ID()).HasKeyword(core.Flying) || g.FindPermanent(own.ID()).HasKeyword(core.Banding) || !clone.FindPermanent(first.ID()).HasKeyword(core.Flying) {
		t.Fatal("condition or clone isolation")
	}
}
