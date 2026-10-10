package ron

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTextStaticAbilityLayerWards(t *testing.T) {
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
			if err := g.AddPrintedStaticAbilityTextEffect(source, id, core.EndOfTurn, change); err != nil {
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
				if err := g.AddPrintedStaticAbilityTextEffect(g.FindPermanent(source.ID()), id, core.EndOfTurn, white); err != nil {
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
			if err := clone.AddPrintedStaticAbilityTextEffect(clone.FindPermanent(source.ID()), id, core.EndOfTurn, blue); err != nil {
				t.Fatal(err)
			}
			check(clone, host, core.White, false)
			check(clone, host, core.Blue, true)
			check(g, host, core.White, true)
			phased := g.PhaseOut(host.ID())
			g.ApplyContinuousEffects()
			if err := g.AddPrintedStaticAbilityTextEffect(source, id, core.EndOfTurn, blue); err == nil {
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
			if err := g.AddPrintedStaticAbilityTextEffect(source, id, core.EndOfTurn, blue); err == nil {
				t.Fatal("removed source accepted")
			}
			g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
			check(g, other, core.White, true)
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			check(g, other, tc.color, true)
			g.BouncePermanentToHand(g.FindPermanent(source.ID()))
			check(g, other, tc.color, false)
			returned := g.PutOnBattlefield(card, a.PlayerID())
			g.Attach(returned.ID(), host.ID())
			g.ApplyContinuousEffects()
			if err := g.AddPrintedStaticAbilityTextEffect(source, id, core.EndOfTurn, change); err == nil {
				t.Fatal("returned source accepted")
			}
			check(g, host, tc.color, true)
		})
	}
}

func TestRONTextStaticAbilityLayerLandwalk(t *testing.T) {
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
			if err := g.AddPrintedStaticAbilityTextEffect(source, textStaticID(t, source), core.EndOfTurn, change); err != nil {
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

func TestRONTextStaticAbilityLayerKeywords(t *testing.T) {
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
			if err := g.AddPrintedStaticAbilityTextEffect(source, id, core.EndOfTurn, change); err != nil {
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

func TestRONTextStaticAbilityLayerBoundaries(t *testing.T) {
	change, _ := mage.ReplaceColorWord(core.Green, core.Red)
	for _, tc := range []struct{ set, name string }{
		{"LEA", "Holy Strength"}, {"LEA", "Goblin King"}, {"LEG", "Concordant Crossroads"}, {"LEG", "Spectral Cloak"}, {"LEA", "Farmstead"},
	} {
		g, a, _ := ronGame()
		source := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
		if err := g.AddPrintedStaticAbilityTextEffect(source, textStaticID(t, source), core.EndOfTurn, change); err == nil {
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

func TestRONTextStaticAbilityLayerIndependentDeclarations(t *testing.T) {
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
	if err := g.AddPrintedStaticAbilityTextEffect(source, first, core.EndOfTurn, green); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedStaticAbilityTextEffect(g.FindPermanent(source.ID()), second, core.EndOfTurn, blue); err != nil {
		t.Fatal(err)
	}
	current := g.FindPermanent(host.ID())
	if current.HasProtectionFrom(textColorCard(core.Red)) || !current.HasProtectionFrom(textColorCard(core.Green)) || !current.HasProtectionFrom(textColorCard(core.Blue)) {
		t.Fatal("sibling revisions shared text state")
	}
}

func TestRONTextStaticAbilityLayerDamageAndLKI(t *testing.T) {
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
	if err := g.AddPrintedStaticAbilityTextEffect(source, textStaticID(t, source), core.EndOfTurn, change); err != nil {
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

func TestRONTextStaticAbilityLayerRecords(t *testing.T) {
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
			hostCard := mage.NewCreature("Host", "", 3, 3)
			if node.Fields["Name"].String == "Consecrate Land" {
				hostCard = mage.NewLand("Host")
			}
			host := g.PutOnBattlefield(hostCard, a.PlayerID())
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			g.Attach(source.ID(), host.ID())
			g.ApplyContinuousEffects()
			colors := []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green}
			attrs := []core.Attr{core.Flying, core.FirstStrike, core.Vigilance, core.Fear, core.Indestructible, core.Islandwalk, core.Mountainwalk}
			baseline := make([]bool, 0, len(colors)+len(attrs))
			for _, color := range colors {
				baseline = append(baseline, g.FindPermanent(host.ID()).HasProtectionFrom(textColorCard(color)))
			}
			for _, attr := range attrs {
				baseline = append(baseline, g.FindPermanent(host.ID()).HasKeyword(attr))
			}
			change, _ := mage.ReplaceColorWord(core.Red, core.Blue)
			if err := g.AddPrintedStaticAbilityTextEffect(source, textStaticID(t, source), core.EndOfTurn, change); err != nil {
				t.Fatal(node.Fields["Name"].String, err)
			}
			g.ApplyContinuousEffects()
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			restored := make([]bool, 0, len(colors)+len(attrs))
			for _, color := range colors {
				restored = append(restored, g.FindPermanent(host.ID()).HasProtectionFrom(textColorCard(color)))
			}
			for _, attr := range attrs {
				restored = append(restored, g.FindPermanent(host.ID()).HasKeyword(attr))
			}
			if !slices.Equal(baseline, restored) {
				t.Fatal("expiry changed original grants", node.Fields["Name"].String)
			}
			names[node.Fields["Name"].String] = true
			records++
		}
	}
	if len(names) != 12 || records != 61 {
		t.Fatalf("unmapped eligible names: %v", names)
	}
	t.Logf("eligible Aura grants: %d unique names / %d records: %v", len(names), records, names)
}

func TestRONTextStaticAbilityLayerExpiry(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEA", "Red Ward")
	host := g.PutOnBattlefield(mage.NewCreature("Host", "", 3, 3), b.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	g.Attach(source.ID(), host.ID())
	id := textStaticID(t, source)
	first, _ := mage.ReplaceColorWord(core.Red, core.Blue)
	second, _ := mage.ReplaceColorWord(core.Blue, core.White)
	if err := g.AddPrintedStaticAbilityTextEffect(source, id, core.EndOfTurn, first); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedStaticAbilityTextEffect(g.FindPermanent(source.ID()), id, core.Indefinite, second); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		g.ApplyContinuousEffects()
	}
	if !g.FindPermanent(host.ID()).HasProtectionFrom(textColorCard(core.White)) || g.FindPermanent(host.ID()).HasProtectionFrom(card) {
		t.Fatal("ordered donor change or exact Aura exception failed")
	}
	if err := g.RevisePrintedStaticAbility(g.FindPermanent(source.ID()), id, first); err == nil {
		t.Fatal("direct revision bypassed layered baseline")
	}
	clone := g.Clone()
	clone.GetEffects().RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	if !clone.FindPermanent(host.ID()).HasProtectionFrom(textColorCard(core.Red)) || clone.FindPermanent(host.ID()).HasProtectionFrom(textColorCard(core.White)) {
		t.Fatal("expired earlier substitution remained in later rule")
	}
	if !g.FindPermanent(host.ID()).HasProtectionFrom(textColorCard(core.White)) {
		t.Fatal("clone expiry changed original")
	}
	g.BouncePermanentToHand(g.FindPermanent(host.ID()))
	if !g.LKI(host.ID()).Snapshot.HasProtectionFrom(textColorCard(core.White)) {
		t.Fatal("LKI lost revised donor grant")
	}
	g.BouncePermanentToHand(g.FindPermanent(source.ID()))
	returned := g.PutOnBattlefield(card, a.PlayerID())
	newHost := g.PutOnBattlefield(mage.NewCreature("New host", "", 3, 3), a.PlayerID())
	g.Attach(returned.ID(), newHost.ID())
	g.ApplyContinuousEffects()
	if !g.FindPermanent(newHost.ID()).HasProtectionFrom(textColorCard(core.Red)) || g.FindPermanent(newHost.ID()).HasProtectionFrom(textColorCard(core.White)) {
		t.Fatal("returned Aura inherited text layer")
	}
	if err := g.AddPrintedStaticAbilityTextEffect(source, id, core.Indefinite, first); err == nil {
		t.Fatal("old Aura incarnation accepted")
	}
}

func TestRONTextStaticAbilityLayerSuppressedAndPhasedExpiry(t *testing.T) {
	for _, scenario := range []string{"suppressed", "phased"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			host := g.PutOnBattlefield(mage.NewCreature("Host", "", 3, 3), a.PlayerID())
			source := g.PutOnBattlefield(repositoryCard(t, "LEA", "White Ward"), b.PlayerID())
			g.Attach(source.ID(), host.ID())
			change, _ := mage.ReplaceColorWord(core.White, core.Blue)
			if err := g.AddPrintedStaticAbilityTextEffect(source, textStaticID(t, source), core.EndOfTurn, change); err != nil {
				t.Fatal(err)
			}
			var restore func()
			if scenario == "suppressed" {
				remover := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), a.PlayerID())
				effect := mage.RemoveAllAbilities(source.ID(), core.WhileOnBattlefield)
				effect.SetSourceID(remover.ID())
				g.AddContinuousEffect(effect)
				restore = func() { g.RemoveFromBattlefield(g.FindPermanent(remover.ID())) }
			} else {
				phased := g.PhaseOut(host.ID())
				restore = func() { g.PhaseIn(phased) }
			}
			clone := g.Clone()
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			restore()
			g.ApplyContinuousEffects()
			if !g.FindPermanent(host.ID()).HasProtectionFrom(textColorCard(core.White)) || g.FindPermanent(host.ID()).HasProtectionFrom(textColorCard(core.Blue)) {
				t.Fatal("expiry lost printed donor baseline")
			}
			if scenario == "phased" {
				clone.PhaseIn([]uuid.UUID{host.ID(), source.ID()})
			} else {
				for _, p := range clone.FilterBattlefield(mage.AnyPermanent) {
					if p.Name() == "Forest" {
						clone.RemoveFromBattlefield(p)
						break
					}
				}
			}
			clone.ApplyContinuousEffects()
			if !clone.FindPermanent(host.ID()).HasProtectionFrom(textColorCard(core.Blue)) {
				t.Fatal("restored clone lost active text")
			}
		})
	}
}

func TestRONTextStaticAbilityLayerPriorRevisionAndLandExpiry(t *testing.T) {
	g, a, _ := ronGame()
	host := g.PutOnBattlefield(mage.NewCreature("Host", "", 3, 3), a.PlayerID())
	source := g.PutOnBattlefield(repositoryCard(t, "LEA", "Burrowing"), a.PlayerID())
	g.Attach(source.ID(), host.ID())
	id := textStaticID(t, source)
	first, _ := mage.ReplaceBasicLandWord(core.MountainLandType, core.ForestLandType)
	second, _ := mage.ReplaceBasicLandWord(core.ForestLandType, core.IslandLandType)
	if err := g.RevisePrintedStaticAbility(source, id, first); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedStaticAbilityTextEffect(g.FindPermanent(source.ID()), id, core.EndOfTurn, second); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(host.ID()).HasKeyword(core.Islandwalk) {
		t.Fatal("land donor text not applied")
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	if !g.FindPermanent(host.ID()).HasKeyword(core.Forestwalk) || g.FindPermanent(host.ID()).HasKeyword(core.Islandwalk) {
		t.Fatal("expiry lost prior direct revision")
	}
}

func TestRONTextStaticAbilityLayerInvalidInstallation(t *testing.T) {
	for _, scenario := range []string{"nil", "unknown-id", "empty", "invalid", "duration", "face-down", "phased", "departed", "grant"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, _ := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, "LEA", "Blue Ward"), a.PlayerID())
			id := textStaticID(t, source)
			change, _ := mage.ReplaceColorWord(core.Blue, core.Red)
			changes := []mage.TextChange{change}
			input := source
			duration := core.EndOfTurn
			switch scenario {
			case "nil":
				input = nil
			case "unknown-id":
				id = uuid.New()
			case "empty":
				changes = nil
			case "invalid":
				changes = []mage.TextChange{{}}
			case "duration":
				duration = core.WhileOnBattlefield
			case "face-down":
				g.MutablePermanent(source.ID()).FaceDown = true
			case "phased":
				g.PhaseOut(source.ID())
			case "departed":
				g.BouncePermanentToHand(source)
			case "grant":
				g.MutablePermanent(source.ID()).RuntimeAbilities[0] = mage.WrapGrantedAbility(source.RuntimeAbilities[0])
			}
			if err := g.AddPrintedStaticAbilityTextEffect(input, id, duration, changes...); err == nil {
				t.Fatal("invalid text installation accepted")
			}
		})
	}
}
