package ron

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTextStaticGrantLords(t *testing.T) {
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
			change, _ := mage.ReplaceBasicLandWord(tc.from, core.ForestLandType)
			if err := g.RevisePrintedStaticGrant(source, id, change); err != nil {
				t.Fatal(err)
			}
			check := func(game *mage.Game, walk core.Attr) {
				t.Helper()
				for _, p := range []*mage.Permanent{own, other} {
					staticPTCheck(t, game, p, 2, 2)
					current := game.FindPermanent(p.ID())
					if !current.HasKeyword(walk) || current.HasKeyword(tc.old) {
						t.Fatal("landwalk revision lost or old word retained")
					}
				}
				staticPTCheck(t, game, source, 2, 2)
				if game.FindPermanent(source.ID()).HasKeyword(walk) {
					t.Fatal("source exclusion lost")
				}
			}
			check(g, core.Forestwalk)
			forest := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
			if !mage.HasLandwalkEvasion(g.FindPermanent(own.ID()), b.PlayerID(), g) {
				t.Fatal("revised landwalk did not affect evasion")
			}
			phasedLand := g.PhaseOut(forest.ID())
			g.ApplyContinuousEffects()
			if mage.HasLandwalkEvasion(g.FindPermanent(own.ID()), b.PlayerID(), g) {
				t.Fatal("phased land enabled evasion")
			}
			g.PhaseIn(phasedLand)
			clone := g.Clone()
			next, _ := mage.ReplaceBasicLandWord(core.ForestLandType, core.SwampLandType)
			if err := clone.RevisePrintedStaticGrant(clone.FindPermanent(source.ID()), id, next); err != nil {
				t.Fatal(err)
			}
			check(clone, core.Swampwalk)
			check(g, core.Forestwalk)
			phased := g.PhaseOut(source.ID())
			g.ApplyContinuousEffects()
			staticPTCheck(t, g, own, 1, 1)
			if g.FindPermanent(own.ID()).HasKeyword(core.Forestwalk) {
				t.Fatal("phased source kept grant")
			}
			if err := g.RevisePrintedStaticGrant(source, id, next); err == nil {
				t.Fatal("phased source accepted")
			}
			g.PhaseIn(phased)
			g.ApplyContinuousEffects()
			check(g, core.Forestwalk)
			remover := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), a.PlayerID())
			removal := mage.RemoveAllAbilities(source.ID(), core.WhileOnBattlefield)
			removal.SetSourceID(remover.ID())
			g.AddContinuousEffect(removal)
			staticPTCheck(t, g, own, 1, 1)
			if g.FindPermanent(own.ID()).HasKeyword(core.Forestwalk) {
				t.Fatal("removed source kept grant")
			}
			if err := g.RevisePrintedStaticGrant(source, id, next); err == nil {
				t.Fatal("removed source accepted")
			}
			g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
			check(g, core.Forestwalk)
			g.BouncePermanentToHand(g.FindPermanent(own.ID()))
			lki := g.LKI(own.ID())
			if lki == nil || lki.Power != 2 || !lki.Snapshot.HasKeyword(core.Forestwalk) {
				t.Fatal("departure lost revised mixed characteristics")
			}
			g.BouncePermanentToHand(g.FindPermanent(source.ID()))
			returned := g.PutOnBattlefield(card, a.PlayerID())
			if err := g.RevisePrintedStaticGrant(source, id, next); err == nil {
				t.Fatal("old source visit accepted")
			}
			if !g.FindPermanent(other.ID()).HasKeyword(tc.old) || g.FindPermanent(other.ID()).HasKeyword(core.Forestwalk) {
				t.Fatal("return retained revision")
			}
			if textStaticID(t, returned) != id {
				t.Fatal("printed declaration identity lost")
			}
		})
	}
}

func TestRONTextStaticGrantMembership(t *testing.T) {
	factory := permanentScopeFactory(t, "Enchantment", `EachPermanentLayerEffect(And([IsColor(Red),IsCardtype(Creature),DoesntHaveAbility(Flying)]),[AdjustPT(1,2),AddAbility([Flying])])`)
	g, a, b := ronGame()
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	red := g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
	green := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	change, _ := mage.ReplaceColorWord(core.Red, core.Green)
	if err := g.RevisePrintedStaticGrant(source, textStaticID(t, source), change); err != nil {
		t.Fatal(err)
	}
	staticPTCheck(t, g, red, 1, 1)
	staticPTCheck(t, g, green, 2, 3)
	if !g.FindPermanent(green.ID()).HasKeyword(core.Flying) {
		t.Fatal("grant lost membership before P/T layer")
	}
	g.ApplyContinuousEffects()
	staticPTCheck(t, g, green, 2, 3)
	late := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	staticPTCheck(t, g, late, 2, 3)
}

func TestRONTextStaticGrantCards(t *testing.T) {
	for _, tc := range []struct {
		set, name, subtype string
		color              core.Color
		keyword            core.Attr
		power, toughness   int
		controlled         bool
	}{
		{"LEG", "Fortified Area", "Wall", core.White, core.Banding, 1, 0, true},
		{"LEG", "Kobold Drill Sergeant", "Kobold", core.Red, core.Trample, 0, 1, true},
		{"LEG", "Kobold Overlord", "Kobold", core.Red, core.FirstStrike, 0, 0, true},
		{"LEG", "Concordant Crossroads", "Goblin", core.Red, core.Haste, 0, 0, false},
		{"FEM", "Goblin War Drums", "Goblin", core.Red, core.Menace, 0, 0, true},
		{"DRK", "Hidden Path", "Goblin", core.Green, core.Forestwalk, 0, 0, false},
		{"LEA", "Zombie Master", "Zombie", core.Black, core.Swampwalk, 0, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			cost := map[core.Color]string{core.White: "{W}", core.Red: "{R}", core.Green: "{G}", core.Black: "{B}"}[tc.color]
			source := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
			own := g.PutOnBattlefield(mage.NewCreature("Own", cost, 1, 1, mage.WithSubTypes(tc.subtype)), a.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", cost, 1, 1, mage.WithSubTypes(tc.subtype)), b.PlayerID())
			id := textStaticID(t, source)
			change, _ := mage.ReplaceColorWord(core.Green, core.Blue)
			if tc.name == "Zombie Master" {
				change, _ = mage.ReplaceBasicLandWord(core.SwampLandType, core.ForestLandType)
			}
			if err := g.RevisePrintedStaticGrant(source, id, change); err != nil {
				t.Fatal(err)
			}
			wantKeyword := tc.keyword
			if tc.name == "Zombie Master" {
				wantKeyword = core.Forestwalk
			}
			active := tc.name != "Hidden Path"
			check := func(p *mage.Permanent, want bool) {
				t.Helper()
				current := g.FindPermanent(p.ID())
				power, toughness := 1, 1
				if want {
					power += tc.power
					toughness += tc.toughness
				}
				staticPTCheck(t, g, p, power, toughness)
				if current.HasKeyword(wantKeyword) != want {
					t.Fatalf("%s keyword %v want %v", tc.name, wantKeyword, want)
				}
			}
			check(own, active)
			check(other, active && !tc.controlled)
			if tc.name == "Hidden Path" {
				blue := g.PutOnBattlefield(textColorCard(core.Blue), b.PlayerID())
				if !g.FindPermanent(blue.ID()).HasKeyword(core.Forestwalk) {
					t.Fatal("color-filter revision failed")
				}
			}
			if tc.controlled {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				check(own, false)
				check(other, true)
			}
			if tc.name == "Zombie Master" && len(g.FindPermanent(own.ID()).RuntimeAbilities) != 1 {
				t.Fatal("revision changed separate granted activation")
			}
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			check(own, false)
			check(other, false)
		})
	}
}

func TestRONTextStaticGrantBoundaries(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"LEA", "Bad Moon"}, {"LEA", "Red Ward"}, {"LEG", "Spectral Cloak"}, {"LEG", "Adventurers' Guildhouse"}} {
		g, a, _ := ronGame()
		source := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
		change, _ := mage.ReplaceColorWord(core.Red, core.Blue)
		if err := g.RevisePrintedStaticGrant(source, textStaticID(t, source), change); err == nil {
			t.Fatal("unsupported declaration accepted", tc.name)
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
}

func TestRONTextStaticGrantRecords(t *testing.T) {
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
			g, a, _ := ronGame()
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			change, _ := mage.ReplaceColorWord(core.Red, core.Blue)
			if err := g.RevisePrintedStaticGrant(source, textStaticID(t, source), change); err != nil {
				t.Fatal(node.Fields["Name"].String, err)
			}
			names[node.Fields["Name"].String] = true
			records++
		}
	}
	if len(names) != 9 {
		t.Fatalf("unmapped eligible names: %v", names)
	}
	t.Logf("eligible global grants: %d unique names / %d records: %v", len(names), records, names)
}

func TestRONTextStaticGrantCastTiming(t *testing.T) {
	g, a, b := ronGame()
	g.SetStep(core.PrecombatMain)
	wall := g.PutOnBattlefield(mage.NewCreature("Wall", "", 1, 4, mage.WithSubTypes("Wall")), a.PlayerID())
	other := g.PutOnBattlefield(mage.NewCreature("Other Wall", "", 1, 4, mage.WithSubTypes("Wall")), b.PlayerID())
	card := repositoryCard(t, "LEG", "Fortified Area")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.White, 3)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	staticPTCheck(t, g, wall, 1, 4)
	if g.FindPermanent(wall.ID()).HasKeyword(core.Banding) {
		t.Fatal("grant began before spell resolved")
	}
	g.ResolveTopOfStack()
	source := g.FindPermanent(card.ID())
	change, _ := mage.ReplaceColorWord(core.Red, core.Blue)
	if err := g.RevisePrintedStaticGrant(source, textStaticID(t, source), change); err != nil {
		t.Fatal(err)
	}
	staticPTCheck(t, g, wall, 2, 4)
	staticPTCheck(t, g, other, 1, 4)
	if !g.FindPermanent(wall.ID()).HasKeyword(core.Banding) || a.ManaPool().TotalMana() != 0 {
		t.Fatal("payment or mixed grant failed")
	}
}

func TestRONTextStaticGrantIndependentDeclarations(t *testing.T) {
	factory := permanentScopeFactory(t, "Enchantment", `EachPermanentLayerEffect(And([IsColor(Red),IsCardtype(Creature)]),[AdjustPT(1,1),AddAbility([Flying])]),EachPermanentLayerEffect(And([IsColor(Red),IsCardtype(Creature)]),[AdjustPT(2,2),AddAbility([Trample])])`)
	g, a, b := ronGame()
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	second := g.PutOnBattlefield(factory(), a.PlayerID())
	red := g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
	green := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	blue := g.PutOnBattlefield(textColorCard(core.Blue), b.PlayerID())
	firstID, secondID := source.RuntimeAbilities[0].AbilityID(), source.RuntimeAbilities[1].AbilityID()
	change, _ := mage.ReplaceColorWord(core.Red, core.Green)
	if err := g.RevisePrintedStaticGrant(source, textStaticID(t, second), change); err == nil {
		t.Fatal("foreign declaration accepted")
	}
	if err := g.RevisePrintedStaticGrant(source, firstID, change); err != nil {
		t.Fatal(err)
	}
	change, _ = mage.ReplaceColorWord(core.Red, core.Blue)
	if err := g.RevisePrintedStaticGrant(g.FindPermanent(source.ID()), secondID, change); err != nil {
		t.Fatal(err)
	}
	staticPTCheck(t, g, red, 4, 4)
	staticPTCheck(t, g, green, 2, 2)
	staticPTCheck(t, g, blue, 3, 3)
	if !g.FindPermanent(green.ID()).HasKeyword(core.Flying) || g.FindPermanent(green.ID()).HasKeyword(core.Trample) || !g.FindPermanent(blue.ID()).HasKeyword(core.Trample) || g.FindPermanent(blue.ID()).HasKeyword(core.Flying) {
		t.Fatal("sibling declarations shared text state")
	}
}

func TestRONTextStaticGrantMultipleKeywords(t *testing.T) {
	factory := permanentScopeFactory(t, "Enchantment", `EachPermanentLayerEffect(And([IsColor(Red),IsCardtype(Creature),DoesntHaveAbility(Flying)]),[AddAbility([Flying,Haste])])`)
	g, a, b := ronGame()
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	green := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	change, _ := mage.ReplaceColorWord(core.Red, core.Green)
	if err := g.RevisePrintedStaticGrant(source, textStaticID(t, source), change); err != nil {
		t.Fatal(err)
	}
	current := g.FindPermanent(green.ID())
	if !current.HasKeyword(core.Flying) || !current.HasKeyword(core.Haste) {
		t.Fatal("an earlier grant removed the subject from a later grant")
	}
}
