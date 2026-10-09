package ron

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTextOpponentPTRecords(t *testing.T) {
	for _, tc := range []struct {
		set, name            string
		power, toughness, dp int
		original             core.Color
	}{
		{"DRK", "Water Wurm", 1, 1, 0, core.Blue},
		{"LEG", "Beasts of Bogardan", 3, 3, 1, core.White},
		{"LEG", "Ivory Guardians", 3, 3, 1, core.Red},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, tc.set, tc.name)
			p := g.PutOnBattlefield(card, a.PlayerID())
			id := textStaticID(t, p)
			change, _ := mage.ReplaceColorWord(tc.original, core.Blue)
			next, _ := mage.ReplaceColorWord(core.Blue, core.Green)
			oldCard := mage.NewEnchantment("Old condition", map[core.Color]string{core.White: "{W}", core.Red: "{R}", core.Blue: "{U}"}[tc.original])
			newCard := mage.NewEnchantment("New condition", "{U}")
			if tc.name == "Water Wurm" {
				change, _ = mage.ReplaceBasicLandWord(core.IslandLandType, core.MountainLandType)
				next, _ = mage.ReplaceBasicLandWord(core.MountainLandType, core.ForestLandType)
				oldCard = repositoryCard(t, "LEA", "Island").(*mage.BaseCard)
				newCard = repositoryCard(t, "LEA", "Mountain").(*mage.BaseCard)
			}
			if err := g.RevisePrintedStaticPTAbility(p, id, change); err != nil {
				t.Fatal(err)
			}
			if textStaticID(t, g.FindPermanent(p.ID())) != id {
				t.Fatal("identity changed")
			}
			if tc.name != "Water Wurm" {
				red := g.PutOnBattlefield(mage.NewCreature("Red source", "{R}", 1, 1), a.PlayerID())
				g.DealDamageToPermanent(g.FindPermanent(p.ID()), 1, red.ID())
				if g.FindPermanent(p.ID()).Damage != 0 {
					t.Fatal("revision changed separate protection")
				}
				g.RemoveFromBattlefield(red)
			}
			g.PutOnBattlefield(oldCard, b.PlayerID())
			staticPTCheck(t, g, p, tc.power, tc.toughness)
			qualifier := g.PutOnBattlefield(newCard, b.PlayerID())
			staticPTCheck(t, g, p, tc.power+tc.dp, tc.toughness+1)
			clone := g.Clone()
			if err := clone.RevisePrintedStaticPTAbility(clone.FindPermanent(p.ID()), id, next); err != nil {
				t.Fatal(err)
			}
			staticPTCheck(t, clone, p, tc.power, tc.toughness)
			staticPTCheck(t, g, p, tc.power+tc.dp, tc.toughness+1)
			phased := g.PhaseOut(qualifier.ID())
			g.ApplyContinuousEffects()
			staticPTCheck(t, g, p, tc.power, tc.toughness)
			g.PhaseIn(phased)
			g.ApplyContinuousEffects()
			staticPTCheck(t, g, p, tc.power+tc.dp, tc.toughness+1)
			phased = g.PhaseOut(p.ID())
			g.ApplyContinuousEffects()
			if err := g.RevisePrintedStaticPTAbility(p, id, next); err == nil {
				t.Fatal("phased source revised")
			}
			g.PhaseIn(phased)
			g.ApplyContinuousEffects()
			staticPTCheck(t, g, p, tc.power+tc.dp, tc.toughness+1)
			remover := g.PutOnBattlefield(repositoryCard(t, "LEA", "Plains"), a.PlayerID())
			removal := mage.RemoveAllAbilities(p.ID(), core.WhileOnBattlefield)
			removal.SetSourceID(remover.ID())
			g.AddContinuousEffect(removal)
			staticPTCheck(t, g, p, tc.power, tc.toughness)
			if err := g.RevisePrintedStaticPTAbility(p, id, next); err == nil {
				t.Fatal("removed declaration revised")
			}
			g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
			staticPTCheck(t, g, p, tc.power+tc.dp, tc.toughness+1)
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			staticPTCheck(t, g, p, tc.power, tc.toughness)
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: qualifier.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			staticPTCheck(t, g, p, tc.power+tc.dp, tc.toughness+1)
			g.BouncePermanentToHand(g.FindPermanent(p.ID()))
			if lki := g.LKI(p.ID()); lki == nil || lki.Power != tc.power+tc.dp || lki.Toughness != tc.toughness+1 {
				t.Fatal("revised LKI lost")
			}
			returned := g.PutOnBattlefield(card, a.PlayerID())
			staticPTCheck(t, g, returned, tc.power+tc.dp, tc.toughness+1)
			if err := g.RevisePrintedStaticPTAbility(p, id, change); err == nil {
				t.Fatal("old visit revised")
			}
			if returned.Card.ManaCost().String() != repositoryCard(t, tc.set, tc.name).ManaCost().String() {
				t.Fatal("printed cost changed")
			}
			if tc.name != "Water Wurm" {
				red := g.PutOnBattlefield(mage.NewCreature("Red", "{R}", 1, 1), b.PlayerID())
				g.DealDamageToPermanent(returned, 1, red.ID())
				if g.FindPermanent(returned.ID()).Damage != 0 {
					t.Fatal("separate protection changed")
				}
			}
		})
	}
}

func TestRONTextOpponentPTBoundaries(t *testing.T) {
	for _, rule := range []string{
		`If(APlayerPassesFilter(Opponent,ControlsA(IsLandType(Island))),[PermanentLayerEffect(ThisPermanent,[AdjustPT(1)])])`,
		`If(APlayerPassesFilter(Opponent,ControlsA(IsLandType(Island))),[PermanentLayerEffect(ThisPermanent,[AdjustPT(ValueX,2)])])`,
		`If(APlayerPassesFilter(Ref_TargetPlayer,ControlsA(IsLandType(Island))),[PermanentLayerEffect(ThisPermanent,[AdjustPT(1,2)])])`,
		`If(APlayerPassesFilter(Opponent,ControlsA(IsLandType(Island))),[PermanentLayerEffect(Ref_TargetPermanent,[AdjustPT(1,2)])])`,
		`If(APlayerPassesFilter(Opponent,ControlsA(And([IsNonToken,IsColor(Blue)]))),[EachPermanentLayerEffect(And([IsCardtype(Creature),IsNamed(NamedCard(""))]),[AdjustPT(1,2)])])`,
		`If(APlayerPassesFilter(Opponent,ControlsA(IsLandType(Island))),[PermanentLayerEffect(ThisPermanent,[AdjustPT(1,2),AddAbility([Flying])])])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),ManaCost:[],Rules:[`+rule+`],CardPT:(Power:1,Toughness:1))`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration accepted", rule)
		}
	}
	for _, rule := range []string{
		`Unless(APlayerPassesFilter(Opponent,ControlsA(IsLandType(Island))),[PermanentLayerEffect(ThisPermanent,[AdjustPT(1,2)])])`,
		`IfElse(APlayerPassesFilter(Opponent,ControlsA(IsLandType(Island))),[PermanentLayerEffect(ThisPermanent,[AdjustPT(1,2)])],[PermanentLayerEffect(ThisPermanent,[AdjustPT(2,1)])])`,
		`If(APlayerPassesFilter(Opponent,ControlsA(IsLandType(Desert))),[PermanentLayerEffect(ThisPermanent,[AdjustPT(1,2)])])`,
		`If(APlayerPassesFilter(Opponent,ControlsA(IsColor(Red))),[PermanentLayerEffect(ThisPermanent,[AdjustPT(1,2)])])`,
		`If(APlayerPassesFilter(Opponent,ControlsA(And([IsNonToken,IsColor(Red)]))),[EachPermanentLayerEffect(IsCardtype(Creature),[AdjustPT(1,2)])])`,
		`If(APlayerPassesFilter(Opponent,ControlsA(IsLandType(Island))),[PermanentLayerEffect(ThisPermanent,[AdjustPT(1,2),AdjustPT(2,1)])])`,
	} {
		g, a, _ := ronGame()
		p := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", rule)(), a.PlayerID())
		change, _ := mage.ReplaceBasicLandWord(core.IslandLandType, core.MountainLandType)
		if err := g.RevisePrintedStaticPTAbility(p, textStaticID(t, p), change); err == nil {
			t.Fatal("unmarked context revised", rule)
		}
	}
}

func TestRONTextOpponentPTLayers(t *testing.T) {
	for _, land := range []bool{false, true} {
		g, a, b := ronGame()
		set, name := "LEG", "Beasts of Bogardan"
		change, _ := mage.ReplaceColorWord(core.White, core.Blue)
		second, _ := mage.ReplaceColorWord(core.Blue, core.Green)
		qualifierCard := mage.Card(mage.NewEnchantment("Qualifier", "{R}"))
		if land {
			set, name = "DRK", "Water Wurm"
			change, _ = mage.ReplaceBasicLandWord(core.IslandLandType, core.MountainLandType)
			second, _ = mage.ReplaceBasicLandWord(core.MountainLandType, core.ForestLandType)
			qualifierCard = repositoryCard(t, "LEA", "Island")
		}
		p := g.PutOnBattlefield(repositoryCard(t, set, name), a.PlayerID())
		id := textStaticID(t, p)
		if err := g.RevisePrintedStaticPTAbility(p, id, change); err != nil {
			t.Fatal(err)
		}
		qualifier := g.PutOnBattlefield(qualifierCard, b.PlayerID())
		base, dp := 3, 1
		if land {
			base, dp = 1, 0
		}
		staticPTCheck(t, g, p, base, base)
		if land {
			effect := mage.BecomesBasicLandTargetEffect(qualifier.ID(), core.WhileOnBattlefield, "Mountain")
			effect.SetSourceID(p.ID())
			g.AddContinuousEffect(effect)
		} else {
			if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetColors(mage.This, core.Blue, core.Red)), qualifier.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.ApplyContinuousEffects()
		}
		staticPTCheck(t, g, p, base+dp, base+1)
		if err := g.RevisePrintedStaticPTAbility(g.FindPermanent(p.ID()), id, second); err != nil {
			t.Fatal(err)
		}
		staticPTCheck(t, g, p, base, base)
		if !land {
			if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetColors(mage.This, core.Green)), qualifier.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.ApplyContinuousEffects()
			staticPTCheck(t, g, p, base+dp, base+1)
		}
	}
}

func TestRONTextOpponentPTIndependentDeclarations(t *testing.T) {
	rule := `If(APlayerPassesFilter(Opponent,ControlsA(And([IsNonToken,IsColor(Red)]))),[PermanentLayerEffect(ThisPermanent,[AdjustPT(1,1)])]),If(APlayerPassesFilter(Opponent,ControlsA(And([IsColor(Red),IsNonToken]))),[PermanentLayerEffect(ThisPermanent,[AdjustPT(2,2)])])`
	g, a, b := ronGame()
	p := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", rule)(), a.PlayerID())
	first, _ := mage.ReplaceColorWord(core.Red, core.Blue)
	second, _ := mage.ReplaceColorWord(core.Red, core.Green)
	if err := g.RevisePrintedStaticPTAbility(p, p.RuntimeAbilities[0].AbilityID(), first); err != nil {
		t.Fatal(err)
	}
	if err := g.RevisePrintedStaticPTAbility(g.FindPermanent(p.ID()), p.RuntimeAbilities[1].AbilityID(), second); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(mage.NewEnchantment("Red", "{R}"), b.PlayerID())
	staticPTCheck(t, g, p, 0, 0)
	g.PutOnBattlefield(mage.NewEnchantment("Blue", "{U}"), b.PlayerID())
	staticPTCheck(t, g, p, 1, 1)
	g.PutOnBattlefield(mage.NewEnchantment("Green", "{G}"), b.PlayerID())
	staticPTCheck(t, g, p, 3, 3)
}

func TestRONTextOpponentPTNamedRecipients(t *testing.T) {
	g, a, b := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "LEG", "Ivory Guardians"), a.PlayerID())
	sibling := g.PutOnBattlefield(repositoryCard(t, "LEG", "Ivory Guardians"), a.PlayerID())
	other := g.PutOnBattlefield(repositoryCard(t, "LEG", "Ivory Guardians"), b.PlayerID())
	unrelated := g.PutOnBattlefield(mage.NewCreature("Unrelated", "", 3, 3), a.PlayerID())
	noncreature := g.PutOnBattlefield(mage.NewArtifact("Ivory Guardians", ""), a.PlayerID())
	g.PutOnBattlefield(mage.NewArtifact("Red", "{R}"), b.PlayerID())
	change, _ := mage.ReplaceColorWord(core.Red, core.Blue)
	if err := g.RevisePrintedStaticPTAbility(source, textStaticID(t, sibling), change); err == nil {
		t.Fatal("foreign declaration revised")
	}
	if err := g.RevisePrintedStaticPTAbility(source, textStaticID(t, source), change); err != nil {
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
	for _, p := range []*mage.Permanent{source, sibling, other} {
		staticPTCheck(t, g, p, 5, 5)
	}
	staticPTCheck(t, g, unrelated, 3, 3)
	if noncreature.CurrentPower(g) != 0 || noncreature.CurrentToughness(g) != 0 {
		t.Fatal("noncreature matched")
	}
	later := g.PutOnBattlefield(mage.NewCreature("Ivory Guardians", "", 3, 3), b.PlayerID())
	staticPTCheck(t, g, later, 5, 5)
	for _, p := range g.FilterBattlefield(mage.IsToken) {
		staticPTCheck(t, g, p, 5, 5)
	}
}
