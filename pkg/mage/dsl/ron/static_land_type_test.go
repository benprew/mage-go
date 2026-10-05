package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONStaticLandTypeDependencies(t *testing.T) {
	for _, order := range []bool{false, true} {
		g, a, b := ronGame()
		mountain := g.PutOnBattlefield(repositoryCard(t, "LEB", "Mountain"), a.PlayerID())
		dual := g.PutOnBattlefield(repositoryCard(t, "LEB", "Tropical Island"), b.PlayerID())
		forest := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), b.PlayerID())
		var moon, conversion *mage.Permanent
		addMoon := func() { moon = g.PutOnBattlefield(repositoryCard(t, "DRK", "Blood Moon"), a.PlayerID()) }
		addConversion := func() { conversion = g.PutOnBattlefield(repositoryCard(t, "LEB", "Conversion"), a.PlayerID()) }
		if order {
			addMoon()
			addConversion()
		} else {
			addConversion()
			addMoon()
		}
		if !g.FindPermanent(mountain.ID()).HasSubType("Plains") || !g.FindPermanent(dual.ID()).HasSubType("Plains") || !g.FindPermanent(forest.ID()).HasSubType("Forest") {
			t.Fatal("dependency order or basic selection")
		}
		if !g.CanAfford(b.PlayerID(), core.ParseManaCost("{W}"), nil) || g.CanAfford(b.PlayerID(), core.ParseManaCost("{U}"), nil) {
			t.Fatal("wrong intrinsic mana")
		}
		later := g.PutOnBattlefield(repositoryCard(t, "LEB", "Volcanic Island"), b.PlayerID())
		if !g.FindPermanent(later.ID()).HasSubType("Plains") {
			t.Fatal("static membership did not update")
		}
		clone := g.Clone()
		g.RemoveFromBattlefield(g.FindPermanent(conversion.ID()))
		if !g.FindPermanent(dual.ID()).HasSubType("Mountain") || !g.FindPermanent(mountain.ID()).HasSubType("Mountain") || !clone.FindPermanent(dual.ID()).HasSubType("Plains") {
			t.Fatal("removal or clone isolation")
		}
		ids := g.PhaseOut(moon.ID())
		g.ApplyContinuousEffects()
		if !g.FindPermanent(dual.ID()).HasSubType("Island") {
			t.Fatal("phased source still applies")
		}
		g.PhaseIn(ids)
		g.ApplyContinuousEffects()
		if !g.FindPermanent(dual.ID()).HasSubType("Mountain") {
			t.Fatal("phase return lost static rule")
		}
		g.RemoveFromBattlefield(g.FindPermanent(moon.ID()))
		if !g.FindPermanent(dual.ID()).HasSubType("Forest") || !g.FindPermanent(dual.ID()).HasSubType("Island") {
			t.Fatal("original types not restored")
		}
	}
}

func TestRONEvilPresenceCasting(t *testing.T) {
	for _, scenario := range []string{"legal", "unpaid", "wrong-type", "shroud", "departed", "phased"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			hostCard := repositoryCard(t, "LEB", "Tropical Island")
			if scenario == "wrong-type" {
				hostCard = mage.NewCreature("Wrong", "", 2, 2)
			}
			if scenario == "shroud" {
				hostCard = mage.NewLand("Hidden", mage.WithKeyword(core.Shroud))
			}
			hostCard.SetOwner(b.PlayerID())
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			aura := repositoryCard(t, "LEB", "Evil Presence")
			aura.SetOwner(a.PlayerID())
			a.AddToHand(aura)
			if scenario != "unpaid" {
				a.ManaPool().Add(core.Black, 1)
			}
			before := a.ManaPool().TotalMana()
			err := g.CastSpellByID(a.PlayerID(), aura.ID(), []uuid.UUID{host.ID()}, 0)
			if scenario == "unpaid" || scenario == "wrong-type" || scenario == "shroud" {
				if err == nil || a.ManaPool().TotalMana() != before || g.StackSize() != 0 || len(a.Hand()) != 1 {
					t.Fatal("illegal cast changed state")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(host.ID()).HasSubType("Swamp") {
				t.Fatal("type changed before resolution")
			}
			if scenario == "departed" {
				g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
			}
			if scenario == "phased" {
				g.PhaseOut(host.ID())
			}
			g.ResolveStack()
			if scenario != "legal" {
				if g.FindPermanent(aura.ID()) != nil || len(a.Graveyard()) != 1 {
					t.Fatal("illegal resolution entered Aura")
				}
				return
			}
			p := g.FindPermanent(host.ID())
			if !p.HasSubType("Swamp") || p.HasSubType("Forest") || p.HasSubType("Island") || p.Card.HasSuperType(core.SuperBasic) || !g.CanAfford(b.PlayerID(), core.ParseManaCost("{B}"), nil) || g.CanAfford(b.PlayerID(), core.ParseManaCost("{U}"), nil) {
				t.Fatal("incorrect land replacement")
			}
			clone := g.Clone()
			ids := g.PhaseOut(host.ID())
			g.PhaseIn(ids)
			g.ApplyContinuousEffects()
			if !g.FindPermanent(host.ID()).HasSubType("Swamp") {
				t.Fatal("indirect phasing lost attachment")
			}
			ids = g.PhaseOut(aura.ID())
			g.ApplyContinuousEffects()
			if !g.FindPermanent(host.ID()).HasSubType("Forest") {
				t.Fatal("direct phasing kept effect")
			}
			g.PhaseIn(ids)
			g.ApplyContinuousEffects()
			g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
			host = g.PutOnBattlefield(hostCard, b.PlayerID())
			if host.HasSubType("Swamp") || !clone.FindPermanent(host.ID()).HasSubType("Swamp") {
				t.Fatal("returned identity or clone isolation")
			}
			g.Attach(aura.ID(), host.ID())
			if !g.FindPermanent(host.ID()).HasSubType("Swamp") {
				t.Fatal("new attachment did not apply")
			}
			g.BouncePermanentToHand(g.FindPermanent(aura.ID()))
			if !g.FindPermanent(host.ID()).HasSubType("Island") {
				t.Fatal("departure did not restore host")
			}
		})
	}
}

func TestRONStaticLandTypeInvalidAndInventory(t *testing.T) {
	for _, rule := range []string{
		`EachPermanentLayerEffect(AnyPermanent,[SetLandType()])`,
		`EachPermanentLayerEffect(AnyPermanent,[SetLandType(Swamp())])`,
		`EachPermanentLayerEffect(AnyPermanent,[SetLandType(Desert)])`,
		`EachPermanentLayerEffect(ControlledByAPlayer(SinglePlayer(You)),[SetLandType(Swamp)])`,
		`EachPermanentLayerEffect(Other(ThisPermanent),[SetLandType(Swamp)])`,
		`EachPermanentLayerEffect(AnyPermanent,[SetLandType(Swamp),Unknown])`,
		`EachPermanentLayerEffect(IsLandType(Forest),[SetLandType(Swamp),AddAbility([Flying])])`,
		`EachPermanentLayerEffect(IsNonSupertype(Basic()),[SetLandType(Swamp)])`,
		`EachPermanentLayerEffect(IsNonSupertype(Unknown),[SetLandType(Swamp)])`,
		`PermanentLayerEffect(HostPermanent,[SetLandType(Swamp)])`,
		`Activated(PayMana([]),ActionList([SetLandType(Swamp)]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[`+rule+`])`), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
	src := `Card(Name:"Types",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[EachPermanentLayerEffect(IsNonSupertype(Basic),[SetLandType(Swamp)])])`
	report, err := InspectFS(fstest.MapFS{"types.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"Basic": "supertype", "Swamp": "land_subtype"} {
		found := false
		for _, sig := range report.Signatures {
			if sig.Name == name && sig.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing %s role for %s", role, name)
		}
	}
}

func TestRONStaticLandTypeCombinedFactories(t *testing.T) {
	for _, landType := range []string{"Plains", "Island", "Swamp", "Mountain", "Forest"} {
		src := `Card(Name:"Combined",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Land)),PermanentLayerEffect(HostPermanent,[SetLandType(` + landType + `),AdjustPT(1,2),AddAbility([Flying])])])`
		nodes, err := Parse(strings.NewReader(src), "combined.ron")
		if err != nil {
			t.Fatal(err)
		}
		factory, err := Compile(nodes[0])
		if err != nil {
			t.Fatal(err)
		}
		g, a, b := ronGame()
		hosts := make([]*mage.Permanent, 0, 2)
		sources := make([]*mage.Permanent, 0, 2)
		for _, owner := range []mage.Player{a, b} {
			host := g.PutOnBattlefield(mage.NewCreature("Land creature", "", 2, 3, mage.WithCardType(core.TypeLand), mage.WithSubTypes("Forest", "Elf"), mage.WithSuperTypes(core.SuperLegendary)), owner.PlayerID())
			card := factory()
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(card, owner.PlayerID())
			g.Attach(source.ID(), host.ID())
			hosts = append(hosts, host)
			sources = append(sources, source)
		}
		check := func(index int, changed bool) {
			p := g.FindPermanent(hosts[index].ID())
			if changed {
				if !p.HasSubType(landType) || p.CurrentPower(g) != 3 || p.CurrentToughness(g) != 5 || !p.HasKeyword(core.Flying) {
					t.Fatal("combined modifications missing")
				}
			} else {
				if p.CurrentPower(g) != 2 || p.CurrentToughness(g) != 3 || p.HasKeyword(core.Flying) || !p.HasSubType("Forest") {
					t.Fatal("removed source still applies")
				}
			}
			if !p.HasType(core.TypeCreature) || !p.HasSubType("Elf") || !p.Card.HasSuperType(core.SuperLegendary) {
				t.Fatal("other characteristics removed")
			}
		}
		check(0, true)
		check(1, true)
		g.RemoveFromBattlefield(g.FindPermanent(sources[0].ID()))
		check(0, false)
		check(1, true)
	}
}

func TestRONStaticLandTypeNativeSpells(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		color     core.Color
		colored   int
		landType  string
	}{
		{"LEB", "Conversion", core.White, 2, "Plains"}, {"DRK", "Blood Moon", core.Red, 1, "Mountain"},
	} {
		g, a, b := ronGame()
		host := g.PutOnBattlefield(repositoryCard(t, "LEB", "Volcanic Island"), b.PlayerID())
		card := repositoryCard(t, tc.set, tc.name)
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		a.ManaPool().Add(tc.color, tc.colored)
		a.ManaPool().Add(core.Colorless, 2)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		if !g.FindPermanent(host.ID()).HasSubType("Island") {
			t.Fatal("static effect began before resolution")
		}
		g.ResolveStack()
		if !g.FindPermanent(host.ID()).HasSubType(tc.landType) || g.FindPermanent(host.ID()).HasSubType("Island") || a.ManaPool().TotalMana() != 0 {
			t.Fatal("resolved spell or payment")
		}
		g.AddContinuousEffect(mage.RemoveAllAbilities(card.ID(), core.Indefinite))
		g.ApplyContinuousEffects()
		if !g.FindPermanent(host.ID()).HasSubType(tc.landType) {
			t.Fatal("later layer undid type change")
		}
	}
}

func TestRONStaticLandTypeRemovesPrintedRules(t *testing.T) {
	for _, name := range []string{"Blood Moon", "Evil Presence"} {
		g, a, b := ronGame()
		card := repositoryCard(t, "ATQ", "Strip Mine")
		card.SetOwner(b.PlayerID())
		land := g.PutOnBattlefield(card, b.PlayerID())
		target := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
		set := "DRK"
		mana := "{R}"
		if name == "Evil Presence" {
			set = "LEB"
			mana = "{B}"
		}
		source := g.PutOnBattlefield(repositoryCard(t, set, name), a.PlayerID())
		if name == "Evil Presence" {
			g.Attach(source.ID(), land.ID())
		}
		if !g.CanAfford(b.PlayerID(), core.ParseManaCost(mana), nil) {
			t.Fatal("replacement intrinsic mana missing")
		}
		if err := g.ActivateAbilityByIndex(b.PlayerID(), land.ID(), 1, []uuid.UUID{target.ID()}); err == nil {
			t.Fatal("removed printed destruction ability activated")
		}
		if g.FindPermanent(land.ID()).Tapped || g.StackSize() != 0 {
			t.Fatal("failed activation paid costs")
		}
		g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
		if err := g.ActivateAbilityByIndex(b.PlayerID(), land.ID(), 1, []uuid.UUID{target.ID()}); err != nil {
			t.Fatal(err)
		}
		if g.FindPermanent(land.ID()) != nil {
			t.Fatal("restored sacrifice cost not paid")
		}
		g.ResolveStack()
		if g.FindPermanent(target.ID()) != nil {
			t.Fatal("restored printed ability did not destroy land")
		}
	}
}

func TestRONPermanentSupertypeFilters(t *testing.T) {
	g, a, _ := ronGame()
	for _, tc := range []struct {
		name  string
		super core.SuperType
	}{{"Basic", core.SuperBasic}, {"Legendary", core.SuperLegendary}} {
		with := g.PutOnBattlefield(mage.NewLand("With", mage.WithSuperTypes(tc.super)), a.PlayerID())
		without := g.PutOnBattlefield(mage.NewLand("Without", mage.WithSubTypes("Forest")), a.PlayerID())
		for _, form := range []string{"IsSupertype", "IsNonSupertype"} {
			nodes, err := Parse(strings.NewReader(form+"("+tc.name+")"), "filter.ron")
			if err != nil {
				t.Fatal(err)
			}
			filter, err := compileFilter(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			positive := form == "IsSupertype"
			if filter.Match(with, g) != positive || filter.Match(without, g) == positive {
				t.Fatal("supertype filter used subtype or negation incorrectly")
			}
		}
	}
}

func TestRONConversionUsesConvertedManaForUpkeep(t *testing.T) {
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	lands := make([]*mage.Permanent, 0, 2)
	for range 2 {
		lands = append(lands, g.PutOnBattlefield(repositoryCard(t, "LEB", "Mountain"), a.PlayerID()))
	}
	card := repositoryCard(t, "LEB", "Conversion")
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	queueRONUpkeep(g, a)
	g.ResolveStack()
	if g.FindPermanent(source.ID()) == nil || a.offers != 1 {
		t.Fatal("conversion could not pay with converted mountains")
	}
	for _, land := range lands {
		if !g.FindPermanent(land.ID()).Tapped || !g.FindPermanent(land.ID()).HasSubType("Plains") {
			t.Fatal("wrong payment mana or surviving static rule")
		}
	}
	a.accept = false
	queueRONUpkeep(g, a)
	g.ResolveStack()
	if g.FindPermanent(source.ID()) != nil {
		t.Fatal("unpaid conversion survived")
	}
	for _, land := range lands {
		if !g.FindPermanent(land.ID()).HasSubType("Mountain") {
			t.Fatal("unpaid sacrifice did not restore land types")
		}
	}
}
