package ron

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTextConditionalPTLayerRecords(t *testing.T) {
	names := map[string]int{}
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
			name := node.Fields["Name"].String
			switch name {
			case "Kird Ape", "Sedge Troll", "Water Wurm", "Beasts of Bogardan", "Ivory Guardians", "Goblin Caves", "Goblin Shrine":
			default:
				continue
			}
			factory, err := Compile(node)
			if err != nil {
				t.Fatal(err)
			}
			names[name]++
			t.Run(filepath.Base(file)+"/"+name, func(t *testing.T) { testConditionalPTLayerRecord(t, name, factory) })
		}
	}
	want := map[string]int{"Kird Ape": 2, "Sedge Troll": 6, "Water Wurm": 1, "Beasts of Bogardan": 1, "Ivory Guardians": 1, "Goblin Caves": 1, "Goblin Shrine": 1}
	for name, count := range want {
		if names[name] != count {
			t.Fatalf("%s: %d records, want %d", name, names[name], count)
		}
	}
	if len(names) != len(want) {
		t.Fatal(names)
	}
}

func testConditionalPTLayerRecord(t *testing.T, name string, factory mage.CardFactory) {
	t.Helper()
	g, a, b := ronGame()
	card := factory()
	source := g.PutOnBattlefield(card, a.PlayerID())
	id := textStaticID(t, source)
	oldLand, newLand := "Forest", "Island"
	oldType := core.ForestLandType
	bp, bt, dp, dt := 1, 1, 1, 2
	opponent, host, color := false, false, false
	oldColor := core.White
	switch name {
	case "Sedge Troll":
		oldLand, oldType, bp, bt, dt = "Swamp", core.SwampLandType, 2, 2, 1
	case "Water Wurm":
		oldLand, oldType, dp, dt, opponent = "Island", core.IslandLandType, 0, 1, true
		newLand = "Forest"
	case "Beasts of Bogardan", "Ivory Guardians":
		bp, bt, dp, dt, opponent, color = 3, 3, 1, 1, true, true
		if name == "Ivory Guardians" {
			oldColor = core.Red
		}
	case "Goblin Caves", "Goblin Shrine":
		host = true
		oldLand, oldType = "Mountain", core.MountainLandType
		bp, bt, dp, dt = 2, 3, 0, 2
		if name == "Goblin Shrine" {
			dp, dt = 1, 0
		}
	}
	newType := core.IslandLandType
	if newLand == "Forest" {
		newType = core.ForestLandType
	}
	change, _ := mage.ReplaceBasicLandWord(oldType, newType)
	next, _ := mage.ReplaceBasicLandWord(newType, core.PlainsLandType)
	oldCard, newCard := repositoryCard(t, "LEA", oldLand), repositoryCard(t, "LEA", newLand)
	if color {
		oldCard = mage.NewEnchantment("Old condition", map[core.Color]string{core.White: "{W}", core.Red: "{R}"}[oldColor])
		newCard = mage.NewEnchantment("New condition", "{U}")
		change, _ = mage.ReplaceColorWord(oldColor, core.Blue)
		next, _ = mage.ReplaceColorWord(core.Blue, core.Green)
	}
	payer := a
	if opponent {
		payer = b
	}
	old := g.PutOnBattlefield(oldCard, payer.PlayerID())
	qualifier := g.PutOnBattlefield(newCard, payer.PlayerID())
	subject := source
	var survivingGoblin *mage.Permanent
	if host {
		subject = g.PutOnBattlefield(mage.NewCreature("Member", "", bp, bt, mage.WithSubTypes("Goblin")), b.PlayerID())
		g.Attach(source.ID(), old.ID())
		survivingGoblin = g.PutOnBattlefield(mage.NewCreature("Survivor", "", 2, 3, mage.WithSubTypes("Goblin")), a.PlayerID())
	}
	check := func(game *mage.Game, active bool) {
		t.Helper()
		p, tu := bp, bt
		if active {
			p += dp
			tu += dt
		}
		staticPTCheck(t, game, subject, p, tu)
	}
	check(g, true)
	if err := g.AddPrintedStaticPTTextEffect(source, id, core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	if host {
		check(g, false)
		g.Attach(source.ID(), qualifier.ID())
	}
	for range 3 {
		g.ApplyContinuousEffects()
		check(g, true)
	}
	if textStaticID(t, g.FindPermanent(source.ID())) != id {
		t.Fatal("declaration identity changed")
	}
	if err := g.RevisePrintedStaticPTAbility(g.FindPermanent(source.ID()), id, next); err == nil {
		t.Fatal("direct revision replaced layered text")
	}
	clone := g.Clone()
	if err := clone.AddPrintedStaticPTTextEffect(clone.FindPermanent(source.ID()), id, core.Indefinite, next); err != nil {
		t.Fatal(err)
	}
	check(clone, false)
	check(g, true)
	clone.GetEffects().RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	check(clone, !host)
	check(g, true)
	phased := g.PhaseOut(qualifier.ID())
	g.ApplyContinuousEffects()
	check(g, false)
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	check(g, true)
	remover := g.PutOnBattlefield(mage.NewArtifact("Remover", ""), a.PlayerID())
	removal := mage.RemoveAllAbilities(source.ID(), core.WhileOnBattlefield)
	removal.SetSourceID(remover.ID())
	g.AddContinuousEffect(removal)
	check(g, false)
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
	check(g, !host)
	if host {
		g.Attach(source.ID(), old.ID())
		check(g, true)
	}
	if err := g.AddPrintedStaticPTTextEffect(g.FindPermanent(source.ID()), id, core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	if host {
		g.Attach(source.ID(), qualifier.ID())
	}
	check(g, true)
	phased = g.PhaseOut(source.ID())
	g.ApplyContinuousEffects()
	if err := g.AddPrintedStaticPTTextEffect(source, id, core.EndOfTurn, change); err == nil {
		t.Fatal("phased source accepted")
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	check(g, !host)
	if host {
		g.Attach(source.ID(), old.ID())
	}
	if err := g.AddPrintedStaticPTTextEffect(g.FindPermanent(source.ID()), id, core.Indefinite, change); err != nil {
		t.Fatal(err)
	}
	if host {
		g.Attach(source.ID(), qualifier.ID())
	}
	check(g, true)
	if name == "Sedge Troll" {
		a.ManaPool().Add(core.Black, 1)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		g.DestroyPermanent(g.FindPermanent(source.ID()))
		if g.FindPermanent(source.ID()) == nil || !g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != 0 {
			t.Fatal("separate regeneration changed")
		}
	}
	if color {
		red := g.PutOnBattlefield(mage.NewCreature("Damage source", "{R}", 1, 1), a.PlayerID())
		g.DealDamageToPermanent(g.FindPermanent(source.ID()), 1, red.ID())
		if g.FindPermanent(source.ID()).Damage != 0 {
			t.Fatal("separate protection changed")
		}
	}
	g.BouncePermanentToHand(g.FindPermanent(subject.ID()))
	if lki := g.LKI(subject.ID()); lki == nil || lki.Power != bp+dp || lki.Toughness != bt+dt {
		t.Fatal("departure lost layered P/T")
	}
	if host {
		g.BouncePermanentToHand(g.FindPermanent(source.ID()))
		g.PutTriggersOnStack()
		g.ResolveStack()
		wantDamage := 0
		if name == "Goblin Shrine" {
			wantDamage = 1
		}
		if g.FindPermanent(survivingGoblin.ID()).Damage != wantDamage {
			t.Fatal("separate leave trigger changed")
		}
		staticPTCheck(t, g, survivingGoblin, 2, 3)
	}
	returned := g.PutOnBattlefield(card, a.PlayerID())
	if host {
		g.Attach(returned.ID(), old.ID())
	} else {
		staticPTCheck(t, g, returned, bp+dp, bt+dt)
	}
	if err := g.AddPrintedStaticPTTextEffect(source, id, core.EndOfTurn, change); err == nil {
		t.Fatal("old visit accepted")
	}
}

func TestRONTextConditionalPTLayerLiveConditions(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		opponent  bool
	}{{"ARN", "Kird Ape", false}, {"DRK", "Water Wurm", true}, {"LEG", "Beasts of Bogardan", true}} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
			id := textStaticID(t, source)
			oldType := core.ForestLandType
			if tc.opponent {
				oldType = core.IslandLandType
			}
			change, _ := mage.ReplaceBasicLandWord(oldType, core.MountainLandType)
			qualifierCard := repositoryCard(t, "LEA", "Forest")
			bp, bt, dp, dt := 1, 1, 1, 2
			if tc.name == "Water Wurm" {
				dp, dt = 0, 1
			}
			if tc.name == "Beasts of Bogardan" {
				change, _ = mage.ReplaceColorWord(core.White, core.Blue)
				qualifierCard = mage.NewEnchantment("Qualifier", "{R}")
				bp, bt, dp, dt = 3, 3, 1, 1
			}
			if err := g.AddPrintedStaticPTTextEffect(source, id, core.EndOfTurn, change); err != nil {
				t.Fatal(err)
			}
			player := a
			if tc.opponent {
				player = b
			}
			q := g.PutOnBattlefield(qualifierCard, player.PlayerID())
			staticPTCheck(t, g, source, bp, bt)
			if tc.name == "Beasts of Bogardan" {
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetColors(mage.This, core.Blue)), q.ID(), player.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			} else {
				effect := mage.BecomesBasicLandTargetEffect(q.ID(), core.WhileOnBattlefield, "Mountain")
				effect.SetSourceID(source.ID())
				g.AddContinuousEffect(effect)
			}
			g.ApplyContinuousEffects()
			staticPTCheck(t, g, source, bp+dp, bt+dt)
			clone := g.Clone()
			clone.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			staticPTCheck(t, clone, source, bp, bt)
			staticPTCheck(t, g, source, bp+dp, bt+dt)
			clone.GetEffects().RemoveEndOfTurn()
			clone.ApplyContinuousEffects()
			staticPTCheck(t, g, source, bp+dp, bt+dt)
		})
	}
}

func TestRONTextConditionalPTLayerNamedRecipients(t *testing.T) {
	g, a, b := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "LEG", "Ivory Guardians"), a.PlayerID())
	sibling := g.PutOnBattlefield(repositoryCard(t, "LEG", "Ivory Guardians"), a.PlayerID())
	other := g.PutOnBattlefield(repositoryCard(t, "LEG", "Ivory Guardians"), b.PlayerID())
	unrelated := g.PutOnBattlefield(mage.NewCreature("Unrelated", "", 3, 3), a.PlayerID())
	noncreature := g.PutOnBattlefield(mage.NewArtifact("Ivory Guardians", ""), b.PlayerID())
	red := g.PutOnBattlefield(mage.NewEnchantment("Red", "{R}"), b.PlayerID())
	change, _ := mage.ReplaceColorWord(core.Red, core.Blue)
	if err := g.AddPrintedStaticPTTextEffect(source, textStaticID(t, sibling), core.EndOfTurn, change); err == nil {
		t.Fatal("foreign declaration accepted")
	}
	if err := g.AddPrintedStaticPTTextEffect(source, textStaticID(t, source), core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	for _, p := range []*mage.Permanent{source, sibling, other} {
		staticPTCheck(t, g, p, 4, 4)
	}
	token := mage.DefineToken("Ivory Guardians", mage.TokenTypes(core.TypeCreature), mage.TokenPT(3, 3), mage.TokenColors(core.Blue))
	if err := mage.ApplyEffect(g, mage.CreateTokenCopies(mage.You, mage.Fixed(1), token), noncreature.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.ApplyContinuousEffects()
	staticPTCheck(t, g, source, 4, 4)
	g.PutOnBattlefield(mage.NewEnchantment("Blue", "{U}"), b.PlayerID())
	later := g.PutOnBattlefield(mage.NewCreature("Ivory Guardians", "", 3, 3), b.PlayerID())
	for _, p := range []*mage.Permanent{source, sibling, other, later} {
		staticPTCheck(t, g, p, 5, 5)
	}
	for _, p := range g.FilterBattlefield(mage.IsToken) {
		staticPTCheck(t, g, p, 5, 5)
	}
	staticPTCheck(t, g, unrelated, 3, 3)
	staticPTCheck(t, g, noncreature, 0, 0)
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	for _, p := range []*mage.Permanent{source, sibling, other, later} {
		staticPTCheck(t, g, p, 5, 5)
	}
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetColors(mage.This, core.Green)), red.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.ApplyContinuousEffects()
	for _, p := range []*mage.Permanent{source, sibling, other, later} {
		staticPTCheck(t, g, p, 3, 3)
	}
}

func TestRONTextConditionalPTLayerPriorRevisionAndVisits(t *testing.T) {
	g, a, b := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "DRK", "Goblin Caves"), a.PlayerID())
	hostCard := repositoryCard(t, "LEA", "Forest")
	host := g.PutOnBattlefield(hostCard, b.PlayerID())
	g.Attach(source.ID(), host.ID())
	member := g.PutOnBattlefield(mage.NewCreature("Member", "", 2, 3, mage.WithSubTypes("Goblin")), a.PlayerID())
	id := textStaticID(t, source)
	first, _ := mage.ReplaceBasicLandWord(core.MountainLandType, core.ForestLandType)
	second, _ := mage.ReplaceBasicLandWord(core.ForestLandType, core.IslandLandType)
	if err := g.RevisePrintedStaticPTAbility(source, id, first); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedStaticPTTextEffect(g.FindPermanent(source.ID()), id, core.EndOfTurn, second); err != nil {
		t.Fatal(err)
	}
	staticPTCheck(t, g, member, 2, 3)
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	staticPTCheck(t, g, member, 2, 5)
	nonbasic := g.PutOnBattlefield(mage.NewLand("Nonbasic Forest", mage.WithSubTypes("Forest")), a.PlayerID())
	g.Attach(source.ID(), nonbasic.ID())
	staticPTCheck(t, g, member, 2, 3)
	g.Attach(source.ID(), host.ID())
	staticPTCheck(t, g, member, 2, 5)
	g.BouncePermanentToHand(g.FindPermanent(host.ID()))
	staticPTCheck(t, g, member, 2, 3)
	returned := g.PutOnBattlefield(hostCard, b.PlayerID())
	staticPTCheck(t, g, member, 2, 3)
	g.Attach(source.ID(), returned.ID())
	staticPTCheck(t, g, member, 2, 5)
	if err := g.AddPrintedStaticPTTextEffect(g.FindPermanent(source.ID()), id, core.EndOfTurn, second); err != nil {
		t.Fatal(err)
	}
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetSubtypeFamily(mage.This, mage.SubtypeLand, "Island")), returned.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.ApplyContinuousEffects()
	staticPTCheck(t, g, member, 2, 5)
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	staticPTCheck(t, g, member, 2, 5)
}

func TestRONTextConditionalPTLayerIndependentDeclarations(t *testing.T) {
	rule := `If(APlayerPassesFilter(Opponent,ControlsA(And([IsNonToken,IsColor(Red)]))),[PermanentLayerEffect(ThisPermanent,[AdjustPT(1,1)])]),If(APlayerPassesFilter(Opponent,ControlsA(And([IsColor(Red),IsNonToken]))),[PermanentLayerEffect(ThisPermanent,[AdjustPT(2,2)])])`
	g, a, b := ronGame()
	source := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", rule)(), a.PlayerID())
	blue, _ := mage.ReplaceColorWord(core.Red, core.Blue)
	green, _ := mage.ReplaceColorWord(core.Red, core.Green)
	if err := g.AddPrintedStaticPTTextEffect(source, source.RuntimeAbilities[0].AbilityID(), core.EndOfTurn, blue); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedStaticPTTextEffect(g.FindPermanent(source.ID()), source.RuntimeAbilities[1].AbilityID(), core.Indefinite, green); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(mage.NewEnchantment("Blue", "{U}"), b.PlayerID())
	staticPTCheck(t, g, source, 1, 1)
	g.PutOnBattlefield(mage.NewEnchantment("Green", "{G}"), b.PlayerID())
	staticPTCheck(t, g, source, 3, 3)
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	staticPTCheck(t, g, source, 2, 2)
}

func TestRONTextConditionalPTLayerBoundaries(t *testing.T) {
	rule := `If(PlayerPassesFilter(You,ControlsA(IsLandType(Forest))),[PermanentLayerEffect(ThisPermanent,[AdjustPT(1,2)])])`
	for _, invalid := range []string{
		strings.Replace(rule, "AdjustPT(1,2)", "AdjustPT(1)", 1),
		strings.Replace(rule, "AdjustPT(1,2)", "AdjustPT(ValueX,2)", 1),
		strings.Replace(rule, "You", "Ref_TargetPlayer", 1),
		strings.Replace(rule, "ThisPermanent", "Ref_TargetPermanent", 1),
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),Rules:[`+invalid+`],CardPT:(Power:1,Toughness:1))`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration accepted", invalid)
		}
	}
	for _, unmarked := range []string{
		strings.Replace(rule, "If(", "Unless(", 1),
		strings.Replace(rule, "ControlsA", "ControlsNo", 1),
		strings.Replace(rule, "Forest", "Desert", 1),
		strings.Replace(rule, "AdjustPT(1,2)", "AdjustPT(1,2),AdjustPT(2,1)", 1),
	} {
		g, a, _ := ronGame()
		source := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", unmarked)(), a.PlayerID())
		change, _ := mage.ReplaceBasicLandWord(core.ForestLandType, core.IslandLandType)
		if err := g.AddPrintedStaticPTTextEffect(source, textStaticID(t, source), core.EndOfTurn, change); err == nil {
			t.Fatal("unmarked context accepted", unmarked)
		}
	}
	g, a, _ := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "ARN", "Kird Ape"), a.PlayerID())
	id := textStaticID(t, source)
	change, _ := mage.ReplaceBasicLandWord(core.ForestLandType, core.IslandLandType)
	for _, duration := range []core.Duration{core.WhileOnBattlefield, core.EndOfCombat} {
		if err := g.AddPrintedStaticPTTextEffect(source, id, duration, change); err == nil {
			t.Fatal("unsupported duration accepted")
		}
	}
	if err := g.AddPrintedStaticPTTextEffect(source, id, core.EndOfTurn); err == nil {
		t.Fatal("empty changes accepted")
	}
}
