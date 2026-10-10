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

func TestRONTextStaticAbilityWards(t *testing.T) {
	for _, tc := range []struct {
		name  string
		color core.Color
	}{
		{"Black Ward", core.Black}, {"Blue Ward", core.Blue}, {"Green Ward", core.Green}, {"Red Ward", core.Red}, {"White Ward", core.White},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			host := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
			other := g.PutOnBattlefield(textColorCard(core.Green), a.PlayerID())
			card := repositoryCard(t, "LEA", tc.name)
			source := g.PutOnBattlefield(card, a.PlayerID())
			g.Attach(source.ID(), host.ID())
			g.ApplyContinuousEffects()
			id := textStaticID(t, source)
			nextColor := core.Red
			if tc.color == core.Red {
				nextColor = core.Blue
			}
			change, _ := mage.ReplaceColorWord(tc.color, nextColor)
			if err := g.RevisePrintedStaticAbility(source, id, change); err != nil {
				t.Fatal(err)
			}
			check := func(game *mage.Game, target *mage.Permanent, color core.Color, want bool) {
				t.Helper()
				if game.FindPermanent(target.ID()).HasProtectionFrom(textColorCard(color)) != want {
					t.Fatalf("protection from %v: want %v", color, want)
				}
			}
			check(g, host, tc.color, false)
			check(g, host, nextColor, true)
			check(g, other, nextColor, false)
			if textStaticID(t, g.FindPermanent(source.ID())) != id {
				t.Fatal("declaration identity changed")
			}
			white, _ := mage.ReplaceColorWord(nextColor, core.White)
			if nextColor != core.White {
				if err := g.RevisePrintedStaticAbility(g.FindPermanent(source.ID()), id, white); err != nil {
					t.Fatal(err)
				}
			}
			if g.FindPermanent(host.ID()).HasProtectionFrom(card) {
				t.Fatal("granting Ward lost its exact exception")
			}
			if !g.FindPermanent(host.ID()).HasProtectionFrom(repositoryCard(t, "LEA", "White Ward")) {
				t.Fatal("another white Aura received the exception")
			}
			clone := g.Clone()
			blue, _ := mage.ReplaceColorWord(core.White, core.Blue)
			if err := clone.RevisePrintedStaticAbility(clone.FindPermanent(source.ID()), id, blue); err != nil {
				t.Fatal(err)
			}
			check(clone, host, core.White, false)
			check(clone, host, core.Blue, true)
			check(g, host, core.White, true)
			phased := g.PhaseOut(host.ID())
			g.ApplyContinuousEffects()
			if err := g.RevisePrintedStaticAbility(source, id, blue); err == nil {
				t.Fatal("phased source accepted")
			}
			g.PhaseIn(phased)
			g.ApplyContinuousEffects()
			check(g, host, core.White, true)
			g.Attach(source.ID(), other.ID())
			g.ApplyContinuousEffects()
			check(g, host, core.White, false)
			check(g, other, core.White, true)
			remover := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), a.PlayerID())
			removal := mage.RemoveAllAbilities(source.ID(), core.WhileOnBattlefield)
			removal.SetSourceID(remover.ID())
			g.AddContinuousEffect(removal)
			check(g, other, core.White, false)
			if err := g.RevisePrintedStaticAbility(source, id, blue); err == nil {
				t.Fatal("removed source accepted")
			}
			g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
			check(g, other, core.White, true)
			g.BouncePermanentToHand(g.FindPermanent(source.ID()))
			check(g, other, core.White, false)
			returned := g.PutOnBattlefield(card, a.PlayerID())
			g.Attach(returned.ID(), host.ID())
			g.ApplyContinuousEffects()
			if err := g.RevisePrintedStaticAbility(source, id, change); err == nil {
				t.Fatal("returned source accepted")
			}
			check(g, host, tc.color, true)
		})
	}
}

func TestRONTextStaticAbilityLandwalk(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		old       core.BasicLandType
		keyword   core.Attr
	}{
		{"LEA", "Burrowing", core.MountainLandType, core.Mountainwalk}, {"ARN", "Fishliver Oil", core.IslandLandType, core.Islandwalk},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			host := g.PutOnBattlefield(textColorCard(core.Green), a.PlayerID())
			source := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), b.PlayerID())
			g.Attach(source.ID(), host.ID())
			g.ApplyContinuousEffects()
			change, _ := mage.ReplaceBasicLandWord(tc.old, core.ForestLandType)
			if err := g.RevisePrintedStaticAbility(source, textStaticID(t, source), change); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(host.ID()).HasKeyword(tc.keyword) || !g.FindPermanent(host.ID()).HasKeyword(core.Forestwalk) {
				t.Fatal("landwalk not replaced")
			}
			land := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
			if !mage.HasLandwalkEvasion(g.FindPermanent(host.ID()), b.PlayerID(), g) {
				t.Fatal("revised landwalk has no evasion")
			}
			phased := g.PhaseOut(land.ID())
			g.ApplyContinuousEffects()
			if mage.HasLandwalkEvasion(g.FindPermanent(host.ID()), b.PlayerID(), g) {
				t.Fatal("phased land enabled evasion")
			}
			g.PhaseIn(phased)
			g.ApplyContinuousEffects()
			if !mage.HasLandwalkEvasion(g.FindPermanent(host.ID()), b.PlayerID(), g) {
				t.Fatal("phase-in lost evasion")
			}
		})
	}
}

func TestRONTextStaticAbilityKeywords(t *testing.T) {
	for _, tc := range []struct {
		set, name, host string
		keyword         core.Attr
	}{
		{"LEA", "Fear", "Llanowar Elves", core.Fear}, {"LEA", "Flight", "Llanowar Elves", core.Flying},
		{"LEA", "Lance", "Llanowar Elves", core.FirstStrike}, {"LEG", "Eternal Warrior", "Llanowar Elves", core.Vigilance},
		{"LEA", "Consecrate Land", "Forest", core.Indestructible},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			host := g.PutOnBattlefield(repositoryCard(t, "LEA", tc.host), b.PlayerID())
			source := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
			g.Attach(source.ID(), host.ID())
			g.ApplyContinuousEffects()
			change, _ := mage.ReplaceColorWord(core.Black, core.Green)
			id := textStaticID(t, source)
			if err := g.RevisePrintedStaticAbility(source, id, change); err != nil {
				t.Fatal(err)
			}
			if textStaticID(t, g.FindPermanent(source.ID())) != id || !g.FindPermanent(host.ID()).HasKeyword(tc.keyword) {
				t.Fatal("no-match revision lost grant or identity")
			}
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			if g.FindPermanent(host.ID()).HasKeyword(tc.keyword) {
				t.Fatal("departed source kept grant")
			}
		})
	}
}

func TestRONTextStaticAbilityBoundaries(t *testing.T) {
	change, _ := mage.ReplaceColorWord(core.Green, core.Red)
	for _, tc := range []struct{ set, name string }{
		{"LEA", "Holy Strength"}, {"LEA", "Goblin King"}, {"LEG", "Concordant Crossroads"}, {"LEG", "Spectral Cloak"}, {"LEA", "Farmstead"},
	} {
		g, a, _ := ronGame()
		source := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
		if err := g.RevisePrintedStaticAbility(source, textStaticID(t, source), change); err == nil {
			t.Fatal("unsupported declaration accepted", tc.name)
		}
	}
	for _, rule := range []string{
		`PermanentLayerEffect(HostPermanent,[AddAbility([Landwalk(IsLandType(UnknownLand))])])`,
		`PermanentLayerEffect(HostPermanent,[AddAbility([ProtectionAndDoesntRemovePermanents(FromColor(Colors([White])),SinglePermanent(Ref_TargetPermanent))])])`,
		`PermanentLayerEffect(Ref_TargetPermanent,[AddAbility([Flying])])`,
		`PermanentLayerEffect(HostPermanent,[AddAbility(Flying)])`,
		`PermanentLayerEffect(HostPermanent,[AddAbility([Flying(Integer(1))])])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),ManaCost:[],Rules:[EnchantPermanent(IsCardtype(Creature)),`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration accepted", rule)
		}
	}
	for _, name := range []string{"Magical Hack", "Sleight of Mind"} {
		if _, err := Compile(staticBaseRecord(t, "LEA", name)); err != nil {
			t.Fatal("complete text consumer failed", name)
		}
	}
}

func TestRONTextStaticAbilityIndependentDeclarations(t *testing.T) {
	node := staticBaseRecord(t, "LEA", "Red Ward")
	node.Fields["Rules"].Args[1] = cloneTextNode(node.Fields["Rules"].Args[1])
	rules := node.Fields["Rules"]
	rules.Args = append(rules.Args, cloneTextNode(rules.Args[1]))
	node.Fields["Rules"] = rules
	factory, err := Compile(node)
	if err != nil {
		t.Fatal(err)
	}
	g, a, _ := ronGame()
	host := g.PutOnBattlefield(textColorCard(core.Green), a.PlayerID())
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	g.Attach(source.ID(), host.ID())
	g.ApplyContinuousEffects()
	first, second := source.RuntimeAbilities[0].AbilityID(), source.RuntimeAbilities[1].AbilityID()
	green, _ := mage.ReplaceColorWord(core.Red, core.Green)
	blue, _ := mage.ReplaceColorWord(core.Red, core.Blue)
	if err := g.RevisePrintedStaticAbility(source, first, green); err != nil {
		t.Fatal(err)
	}
	if err := g.RevisePrintedStaticAbility(g.FindPermanent(source.ID()), second, blue); err != nil {
		t.Fatal(err)
	}
	current := g.FindPermanent(host.ID())
	if current.HasProtectionFrom(textColorCard(core.Red)) || !current.HasProtectionFrom(textColorCard(core.Green)) || !current.HasProtectionFrom(textColorCard(core.Blue)) {
		t.Fatal("sibling revisions shared text state")
	}
}

func TestRONTextStaticAbilityDamageAndLKI(t *testing.T) {
	g, a, b := ronGame()
	host := g.PutOnBattlefield(mage.NewCreature("Host", "", 3, 3), b.PlayerID())
	other := g.PutOnBattlefield(mage.NewCreature("Other", "", 3, 3), b.PlayerID())
	card := repositoryCard(t, "LEA", "White Ward")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.White, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	if host.HasProtectionFrom(textColorCard(core.White)) {
		t.Fatal("grant preceded resolution")
	}
	g.ResolveTopOfStack()
	source := g.FindPermanent(card.ID())
	change, _ := mage.ReplaceColorWord(core.White, core.Red)
	if err := g.RevisePrintedStaticAbility(source, textStaticID(t, source), change); err != nil {
		t.Fatal(err)
	}
	red := g.PutOnBattlefield(textColorCard(core.Red), a.PlayerID())
	damage := g.CaptureDamageSource(red.ID())
	g.DealSimultaneousDamage(mage.PermanentDamageAssignment(g.FindPermanent(host.ID()), 1, damage), mage.PermanentDamageAssignment(other, 1, damage))
	if g.FindPermanent(host.ID()).Damage != 0 || g.FindPermanent(other.ID()).Damage != 1 {
		t.Fatal("revised protection did not use shared damage prevention")
	}
	g.BouncePermanentToHand(g.FindPermanent(host.ID()))
	lki := g.LKI(host.ID())
	if lki == nil || !lki.Snapshot.HasProtectionFrom(textColorCard(core.Red)) || lki.Snapshot.HasProtectionFrom(textColorCard(core.White)) {
		t.Fatal("departure lost revised protection")
	}
	clone := g.Clone()
	clone.RemoveFromBattlefield(clone.FindPermanent(source.ID()))
	if !g.LKI(host.ID()).Snapshot.HasProtectionFrom(textColorCard(core.Red)) {
		t.Fatal("clone changed departure information")
	}
}

func TestRONTextStaticAbilityRecords(t *testing.T) {
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
			match := false
			for _, rule := range node.Fields["Rules"].Args {
				if rule.Name != "PermanentLayerEffect" || len(rule.Args) != 2 || rule.Args[0].Name != "HostPermanent" {
					continue
				}
				parts := rule.Args[1].Args
				if len(parts) != 1 || parts[0].Name != "AddAbility" || len(parts[0].Args) != 1 {
					continue
				}
				abilities := parts[0].Args[0].Args
				if len(abilities) == 0 {
					continue
				}
				match = true
				for _, ability := range abilities {
					_, keyword := keywords[ability.Name]
					if (!keyword || ability.Kind != Symbol) && ability.Name != "Landwalk" && ability.Name != "ProtectionAndDoesntRemovePermanents" {
						match = false
					}
				}
			}
			if !match {
				continue
			}
			factory, err := Compile(node)
			if err != nil {
				t.Fatal(err)
			}
			g, a, _ := ronGame()
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			change, _ := mage.ReplaceColorWord(core.Red, core.Blue)
			if err := g.RevisePrintedStaticAbility(source, textStaticID(t, source), change); err != nil {
				t.Fatal(node.Fields["Name"].String, err)
			}
			names[node.Fields["Name"].String] = true
			records++
		}
	}
	if len(names) != 12 {
		t.Fatalf("unmapped eligible names: %v", names)
	}
	t.Logf("eligible Aura grants: %d unique names / %d records: %v", len(names), records, names)
}
