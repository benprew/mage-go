package ron

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTextStaticGrantLayerRecords(t *testing.T) {
	names := map[string]bool{}
	records := 0
	files, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil {
		t.Fatal(err)
	}
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
			eligible := false
			for _, rule := range node.Fields["Rules"].Args {
				if rule.Name != "EachPermanentLayerEffect" || len(rule.Args) != 2 {
					continue
				}
				hasGrant, valid := false, true
				for _, part := range rule.Args[1].Args {
					if part.Name == "AdjustPT" {
						continue
					}
					if part.Name != "AddAbility" || len(part.Args) != 1 || part.Args[0].Kind != List {
						valid = false
						break
					}
					hasGrant = true
					for _, ability := range part.Args[0].Args {
						_, keyword := keywords[ability.Name]
						if (!keyword || ability.Kind != Symbol) && ability.Name != "Landwalk" {
							valid = false
						}
					}
				}
				eligible = eligible || (hasGrant && valid)
			}
			if !eligible {
				continue
			}
			factory, err := Compile(node)
			if err != nil {
				t.Fatal(err)
			}
			testStaticGrantLayerRecord(t, node.Fields["Name"].String, factory)
			names[node.Fields["Name"].String] = true
			records++
		}
	}
	if len(names) != 9 || records != 24 {
		t.Fatalf("unmapped eligible names: %v", names)
	}
	t.Logf("eligible global grants: %d unique names / %d records: %v", len(names), records, names)
}

func TestRONTextStaticGrantLayerLords(t *testing.T) {
	for _, tc := range []struct {
		name, creature string
		from           core.BasicLandType
		old            core.Attr
	}{
		{"Goblin King", "Mons's Goblin Raiders", core.MountainLandType, core.Mountainwalk},
		{"Lord of Atlantis", "Merfolk of the Pearl Trident", core.IslandLandType, core.Islandwalk},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEA", tc.name)
			source := g.PutOnBattlefield(card, a.PlayerID())
			own := g.PutOnBattlefield(repositoryCard(t, "LEA", tc.creature), a.PlayerID())
			other := g.PutOnBattlefield(repositoryCard(t, "LEA", tc.creature), b.PlayerID())
			id := textStaticID(t, source)
			check := func(game *mage.Game, keyword core.Attr, active bool) {
				t.Helper()
				for _, p := range []*mage.Permanent{own, other} {
					power := 1
					if active {
						power++
					}
					staticPTCheck(t, game, p, power, power)
					current := game.FindPermanent(p.ID())
					for _, walk := range []core.Attr{tc.old, core.Forestwalk, core.Swampwalk} {
						if current.HasKeyword(walk) != (active && walk == keyword) {
							t.Fatalf("unexpected walk %v: %v", walk, current.HasKeyword(walk))
						}
					}
				}
				if present := game.FindPermanent(source.ID()); present != nil {
					staticPTCheck(t, game, source, 2, 2)
					if present.HasKeyword(keyword) {
						t.Fatal("source received its own grant")
					}
				}
			}
			change, _ := mage.ReplaceBasicLandWord(tc.from, core.ForestLandType)
			if err := g.AddPrintedStaticGrantTextEffect(source, id, core.EndOfTurn, change); err != nil {
				t.Fatal(err)
			}
			check(g, core.Forestwalk, true)
			for range 3 {
				g.ApplyContinuousEffects()
				check(g, core.Forestwalk, true)
			}
			next, _ := mage.ReplaceBasicLandWord(core.ForestLandType, core.SwampLandType)
			if err := g.RevisePrintedStaticGrant(g.FindPermanent(source.ID()), id, next); err == nil {
				t.Fatal("direct revision changed layered result")
			}
			if err := g.AddPrintedStaticGrantTextEffect(g.FindPermanent(source.ID()), id, core.Indefinite, next); err != nil {
				t.Fatal(err)
			}
			check(g, core.Swampwalk, true)
			clone := g.Clone()
			clone.GetEffects().RemoveEndOfTurn()
			clone.ApplyContinuousEffects()
			check(clone, tc.old, true)
			check(g, core.Swampwalk, true)
			phased := g.PhaseOut(source.ID())
			g.ApplyContinuousEffects()
			check(g, tc.old, false)
			if err := g.AddPrintedStaticGrantTextEffect(source, id, core.EndOfTurn, change); err == nil {
				t.Fatal("phased source accepted")
			}
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			g.PhaseIn(phased)
			g.ApplyContinuousEffects()
			check(g, tc.old, true)
			if err := g.AddPrintedStaticGrantTextEffect(g.FindPermanent(source.ID()), id, core.Indefinite, change); err != nil {
				t.Fatal(err)
			}
			check(g, core.Forestwalk, true)
			remover := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), a.PlayerID())
			removal := mage.RemoveAllAbilities(source.ID(), core.WhileOnBattlefield)
			removal.SetSourceID(remover.ID())
			g.AddContinuousEffect(removal)
			check(g, tc.old, false)
			g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
			check(g, core.Forestwalk, true)
			g.BouncePermanentToHand(g.FindPermanent(own.ID()))
			lki := g.LKI(own.ID())
			if lki == nil || lki.Power != 2 || !lki.Snapshot.HasKeyword(core.Forestwalk) {
				t.Fatal("LKI lost revised grant")
			}
			g.BouncePermanentToHand(g.FindPermanent(source.ID()))
			returned := g.PutOnBattlefield(card, a.PlayerID())
			if textStaticID(t, returned) != id {
				t.Fatal("printed identity changed on return")
			}
			if err := g.AddPrintedStaticGrantTextEffect(source, id, core.Indefinite, change); err == nil {
				t.Fatal("old visit accepted")
			}
			if !g.FindPermanent(other.ID()).HasKeyword(tc.old) || g.FindPermanent(other.ID()).HasKeyword(core.Forestwalk) {
				t.Fatal("return kept prior text")
			}
		})
	}
}

func TestRONTextStaticGrantLayerMembership(t *testing.T) {
	factory := permanentScopeFactory(t, "Enchantment", `EachPermanentLayerEffect(And([IsColor(Red),IsCardtype(Creature),DoesntHaveAbility(Flying)]),[AdjustPT(1,2),AddAbility([Flying,Haste])])`)
	g, a, b := ronGame()
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	red := g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
	green := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	blue := g.PutOnBattlefield(textColorCard(core.Blue), b.PlayerID())
	id := textStaticID(t, source)
	first, _ := mage.ReplaceColorWord(core.Red, core.Green)
	if err := g.AddPrintedStaticGrantTextEffect(source, id, core.EndOfTurn, first); err != nil {
		t.Fatal(err)
	}
	check := func(p *mage.Permanent, active bool) {
		t.Helper()
		power, toughness := 1, 1
		if active {
			power, toughness = 2, 3
		}
		staticPTCheck(t, g, p, power, toughness)
		for _, keyword := range []core.Attr{core.Flying, core.Haste} {
			if g.FindPermanent(p.ID()).HasKeyword(keyword) != active {
				t.Fatal("shared membership lost a grant")
			}
		}
	}
	check(red, false)
	check(green, true)
	check(blue, false)
	for range 3 {
		g.ApplyContinuousEffects()
		check(green, true)
	}
	late := g.PutOnBattlefield(textColorCard(core.Green), a.PlayerID())
	check(late, true)
	second, _ := mage.ReplaceColorWord(core.Green, core.Blue)
	if err := g.AddPrintedStaticGrantTextEffect(g.FindPermanent(source.ID()), id, core.Indefinite, second); err != nil {
		t.Fatal(err)
	}
	check(green, false)
	check(blue, true)
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	check(red, true)
	check(green, false)
	check(blue, false)
}

func TestRONTextStaticGrantLayerSuppressedExpiry(t *testing.T) {
	for _, phased := range []bool{false, true} {
		t.Run(map[bool]string{false: "removed", true: "phased"}[phased], func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, "DRK", "Hidden Path"), a.PlayerID())
			green := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
			blue := g.PutOnBattlefield(textColorCard(core.Blue), b.PlayerID())
			change, _ := mage.ReplaceColorWord(core.Green, core.Blue)
			if err := g.AddPrintedStaticGrantTextEffect(source, textStaticID(t, source), core.EndOfTurn, change); err != nil {
				t.Fatal(err)
			}
			if phased {
				visits := g.PhaseOut(source.ID())
				g.GetEffects().RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				g.PhaseIn(visits)
			} else {
				g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.EndOfTurn))
				g.GetEffects().RemoveEndOfTurn()
			}
			g.ApplyContinuousEffects()
			if !g.FindPermanent(green.ID()).HasKeyword(core.Forestwalk) || g.FindPermanent(blue.ID()).HasKeyword(core.Forestwalk) {
				t.Fatal("suppressed expiry failed to restore baseline")
			}
		})
	}
}

func TestRONTextStaticGrantLayerBoundaries(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"LEA", "Bad Moon"}, {"LEA", "Red Ward"}, {"LEG", "Spectral Cloak"}, {"LEG", "Adventurers' Guildhouse"}} {
		g, a, _ := ronGame()
		source := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
		change, _ := mage.ReplaceColorWord(core.Red, core.Blue)
		if err := g.AddPrintedStaticGrantTextEffect(source, textStaticID(t, source), core.EndOfTurn, change); err == nil {
			t.Fatal("unsupported context accepted", tc.name)
		}
	}
	for _, rule := range []string{
		`EachPermanentLayerEffect(IsCardtype(Creature),[AddAbility([])])`,
		`EachPermanentLayerEffect(IsCardtype(Creature),[AddAbility([Flying(Integer(1))])])`,
		`EachPermanentLayerEffect(IsCardtype(Creature),[AddAbility([Landwalk(IsLandType(Bog))])])`,
		`EachPermanentLayerEffect(Ref_TargetPermanent,[AdjustPT(1,1),AddAbility([Flying])])`,
		`EachPermanentLayerEffect(IsCardtype(Creature),[AdjustPT(ValueX,1),AddAbility([Flying])])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),ManaCost:[],Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration accepted", rule)
		}
	}
	g, a, _ := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "LEA", "Goblin King"), a.PlayerID())
	change, _ := mage.ReplaceColorWord(core.Red, core.Blue)
	for _, duration := range []core.Duration{core.WhileOnBattlefield, core.UntilYourNextTurn} {
		if err := g.AddPrintedStaticGrantTextEffect(source, textStaticID(t, source), duration, change); err == nil {
			t.Fatal("unsupported duration accepted")
		}
	}
	if err := g.AddPrintedStaticGrantTextEffect(source, textStaticID(t, source), core.EndOfTurn); err == nil {
		t.Fatal("empty changes accepted")
	}
}

func testStaticGrantLayerRecord(t *testing.T, name string, factory mage.CardFactory) {
	t.Helper()
	type grantCase struct {
		subtype, cost    string
		keyword          core.Attr
		power, toughness int
		controlled       bool
		land             core.BasicLandType
	}
	cases := map[string]grantCase{
		"Goblin King":           {"Goblin", "{R}", core.Mountainwalk, 1, 1, false, core.MountainLandType},
		"Lord of Atlantis":      {"Merfolk", "{U}", core.Islandwalk, 1, 1, false, core.IslandLandType},
		"Zombie Master":         {"Zombie", "{B}", core.Swampwalk, 0, 0, false, core.SwampLandType},
		"Hidden Path":           {"Goblin", "{G}", core.Forestwalk, 0, 0, false, core.ForestLandType},
		"Fortified Area":        {"Wall", "{W}", core.Banding, 1, 0, true, 0},
		"Kobold Drill Sergeant": {"Kobold", "{R}", core.Trample, 0, 1, true, 0},
		"Kobold Overlord":       {"Kobold", "{R}", core.FirstStrike, 0, 0, true, 0},
		"Concordant Crossroads": {"Goblin", "{R}", core.Haste, 0, 0, false, 0},
		"Goblin War Drums":      {"Goblin", "{R}", core.Menace, 0, 0, true, 0},
	}
	tc, ok := cases[name]
	if !ok {
		t.Fatal("unmapped card", name)
	}
	g, a, b := ronGame()
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	own := g.PutOnBattlefield(mage.NewCreature("Own", tc.cost, 1, 1, mage.WithSubTypes(tc.subtype)), a.PlayerID())
	other := g.PutOnBattlefield(mage.NewCreature("Other", tc.cost, 1, 1, mage.WithSubTypes(tc.subtype)), b.PlayerID())
	change, _ := mage.ReplaceColorWord(core.Red, core.Blue)
	revised := tc.keyword
	if tc.land != 0 {
		change, _ = mage.ReplaceBasicLandWord(tc.land, core.PlainsLandType)
		revised = core.Plainswalk
	}
	check := func(keyword core.Attr, controllerIsB bool, present bool) {
		t.Helper()
		for i, p := range []*mage.Permanent{own, other} {
			active := present && (!tc.controlled || (controllerIsB == (i == 1)))
			power, toughness := 1, 1
			if active {
				power += tc.power
				toughness += tc.toughness
			}
			staticPTCheck(t, g, p, power, toughness)
			current := g.FindPermanent(p.ID())
			if current.HasKeyword(keyword) != active {
				t.Fatal(name, "grant membership incorrect", i, keyword)
			}
			if tc.land != 0 && keyword != tc.keyword && current.HasKeyword(tc.keyword) {
				t.Fatal(name, "old text grant remains")
			}
			if name == "Zombie Master" && len(current.RuntimeAbilities) != 1 {
				t.Fatal("separate regeneration donor changed")
			}
		}
	}
	check(tc.keyword, false, true)
	id := textStaticID(t, source)
	if err := g.AddPrintedStaticGrantTextEffect(source, id, core.EndOfTurn, change); err != nil {
		t.Fatal(name, err)
	}
	check(revised, false, true)
	g.ApplyContinuousEffects()
	check(revised, false, true)
	if tc.controlled {
		g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
		check(revised, true, true)
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	check(tc.keyword, tc.controlled, true)
	if textStaticID(t, g.FindPermanent(source.ID())) != id {
		t.Fatal("expiry changed declaration identity")
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	if name != "Zombie Master" {
		check(tc.keyword, tc.controlled, false)
	} else {
		for _, p := range []*mage.Permanent{own, other} {
			if g.FindPermanent(p.ID()).HasKeyword(tc.keyword) || len(g.FindPermanent(p.ID()).RuntimeAbilities) != 0 {
				t.Fatal("departed donor left grants")
			}
		}
	}
}

func TestRONTextStaticGrantLayerPriorRevision(t *testing.T) {
	g, a, b := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "LEA", "Goblin King"), a.PlayerID())
	own := g.PutOnBattlefield(repositoryCard(t, "LEA", "Mons's Goblin Raiders"), a.PlayerID())
	forest := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
	id := textStaticID(t, source)
	first, _ := mage.ReplaceBasicLandWord(core.MountainLandType, core.ForestLandType)
	if err := g.RevisePrintedStaticGrant(source, id, first); err != nil {
		t.Fatal(err)
	}
	second, _ := mage.ReplaceBasicLandWord(core.ForestLandType, core.SwampLandType)
	if err := g.AddPrintedStaticGrantTextEffect(g.FindPermanent(source.ID()), id, core.EndOfTurn, second); err != nil {
		t.Fatal(err)
	}
	if mage.HasLandwalkEvasion(g.FindPermanent(own.ID()), b.PlayerID(), g) {
		t.Fatal("old land word still enables evasion")
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	if !g.FindPermanent(own.ID()).HasKeyword(core.Forestwalk) || !mage.HasLandwalkEvasion(g.FindPermanent(own.ID()), b.PlayerID(), g) {
		t.Fatal("expiry lost direct revision baseline")
	}
	visits := g.PhaseOut(forest.ID())
	g.ApplyContinuousEffects()
	if mage.HasLandwalkEvasion(g.FindPermanent(own.ID()), b.PlayerID(), g) {
		t.Fatal("phased defending land enabled evasion")
	}
	g.PhaseIn(visits)
	g.ApplyContinuousEffects()
	if !mage.HasLandwalkEvasion(g.FindPermanent(own.ID()), b.PlayerID(), g) {
		t.Fatal("returned land failed evasion")
	}
}

func TestRONTextStaticGrantLayerIndependentDeclarations(t *testing.T) {
	factory := permanentScopeFactory(t, "Enchantment", `EachPermanentLayerEffect(And([IsColor(Red),IsCardtype(Creature)]),[AdjustPT(1,1),AddAbility([Flying])]),EachPermanentLayerEffect(And([IsColor(Red),IsCardtype(Creature)]),[AdjustPT(2,2),AddAbility([Trample])])`)
	g, a, b := ronGame()
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	sibling := g.PutOnBattlefield(factory(), a.PlayerID())
	red := g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
	green := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	blue := g.PutOnBattlefield(textColorCard(core.Blue), b.PlayerID())
	firstID, secondID := source.RuntimeAbilities[0].AbilityID(), source.RuntimeAbilities[1].AbilityID()
	first, _ := mage.ReplaceColorWord(core.Red, core.Green)
	if err := g.AddPrintedStaticGrantTextEffect(source, textStaticID(t, sibling), core.EndOfTurn, first); err == nil {
		t.Fatal("foreign declaration accepted")
	}
	if err := g.AddPrintedStaticGrantTextEffect(source, firstID, core.EndOfTurn, first); err != nil {
		t.Fatal(err)
	}
	second, _ := mage.ReplaceColorWord(core.Red, core.Blue)
	if err := g.AddPrintedStaticGrantTextEffect(g.FindPermanent(source.ID()), secondID, core.Indefinite, second); err != nil {
		t.Fatal(err)
	}
	staticPTCheck(t, g, red, 4, 4)
	staticPTCheck(t, g, green, 2, 2)
	staticPTCheck(t, g, blue, 3, 3)
	if !g.FindPermanent(green.ID()).HasKeyword(core.Flying) || g.FindPermanent(green.ID()).HasKeyword(core.Trample) || !g.FindPermanent(blue.ID()).HasKeyword(core.Trample) || g.FindPermanent(blue.ID()).HasKeyword(core.Flying) {
		t.Fatal("sibling text state mixed")
	}
	clone := g.Clone()
	next, _ := mage.ReplaceColorWord(core.Blue, core.Green)
	if err := clone.AddPrintedStaticGrantTextEffect(clone.FindPermanent(source.ID()), secondID, core.Indefinite, next); err != nil {
		t.Fatal(err)
	}
	staticPTCheck(t, clone, green, 4, 4)
	staticPTCheck(t, g, green, 2, 2)
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	staticPTCheck(t, g, red, 5, 5)
	staticPTCheck(t, g, green, 1, 1)
	staticPTCheck(t, g, blue, 3, 3)
}

func TestRONTextStaticGrantLayerCastTiming(t *testing.T) {
	g, a, b := ronGame()
	g.SetStep(core.PrecombatMain)
	own := g.PutOnBattlefield(mage.NewCreature("Own Wall", "", 1, 4, mage.WithSubTypes("Wall")), a.PlayerID())
	other := g.PutOnBattlefield(mage.NewCreature("Other Wall", "", 1, 4, mage.WithSubTypes("Wall")), b.PlayerID())
	card := repositoryCard(t, "LEG", "Fortified Area")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.White, 3)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(own.ID()).HasKeyword(core.Banding) {
		t.Fatal("grant started before resolution")
	}
	g.ResolveTopOfStack()
	source := g.FindPermanent(card.ID())
	change, _ := mage.ReplaceColorWord(core.Red, core.Blue)
	if err := g.AddPrintedStaticGrantTextEffect(source, textStaticID(t, source), core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	staticPTCheck(t, g, own, 2, 4)
	staticPTCheck(t, g, other, 1, 4)
	if !g.FindPermanent(own.ID()).HasKeyword(core.Banding) || a.ManaPool().TotalMana() != 0 {
		t.Fatal("resolved grant or payment failed")
	}
}
