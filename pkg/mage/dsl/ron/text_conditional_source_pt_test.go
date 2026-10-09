package ron

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTextConditionalSourcePTRecords(t *testing.T) {
	for _, tc := range []struct {
		set, name, land  string
		power, toughness int
	}{
		{"ARN", "Kird Ape", "Forest", 1, 1}, {"3ED", "Kird Ape", "Forest", 1, 1},
		{"LEA", "Sedge Troll", "Swamp", 2, 2}, {"LEB", "Sedge Troll", "Swamp", 2, 2},
		{"2ED", "Sedge Troll", "Swamp", 2, 2}, {"3ED", "Sedge Troll", "Swamp", 2, 2},
		{"CED", "Sedge Troll", "Swamp", 2, 2}, {"CEI", "Sedge Troll", "Swamp", 2, 2},
	} {
		t.Run(tc.set+"/"+tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, tc.set, tc.name)
			p := g.PutOnBattlefield(card, a.PlayerID())
			sibling := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
			id := textStaticID(t, p)
			landType := core.ForestLandType
			if tc.land == "Swamp" {
				landType = core.SwampLandType
			}
			change, _ := mage.ReplaceBasicLandWord(landType, core.IslandLandType)
			if err := g.RevisePrintedStaticPTAbility(p, id, change); err != nil {
				t.Fatal(err)
			}
			if textStaticID(t, g.FindPermanent(p.ID())) != id {
				t.Fatal("identity changed")
			}
			bonusT := 1
			if tc.name == "Kird Ape" {
				bonusT = 2
			}
			original := g.PutOnBattlefield(repositoryCard(t, "LEA", tc.land), a.PlayerID())
			staticPTCheck(t, g, p, tc.power, tc.toughness)
			staticPTCheck(t, g, sibling, tc.power+1, tc.toughness+bonusT)
			island := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
			staticPTCheck(t, g, p, tc.power, tc.toughness)
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			staticPTCheck(t, g, p, tc.power+1, tc.toughness+bonusT)
			clone := g.Clone()
			next, _ := mage.ReplaceBasicLandWord(core.IslandLandType, core.MountainLandType)
			if err := clone.RevisePrintedStaticPTAbility(clone.FindPermanent(p.ID()), id, next); err != nil {
				t.Fatal(err)
			}
			staticPTCheck(t, clone, p, tc.power, tc.toughness)
			staticPTCheck(t, g, p, tc.power+1, tc.toughness+bonusT)
			phased := g.PhaseOut(island.ID())
			g.ApplyContinuousEffects()
			staticPTCheck(t, g, p, tc.power, tc.toughness)
			g.PhaseIn(phased)
			g.ApplyContinuousEffects()
			staticPTCheck(t, g, p, tc.power+1, tc.toughness+bonusT)
			phased = g.PhaseOut(p.ID())
			g.ApplyContinuousEffects()
			if err := g.RevisePrintedStaticPTAbility(p, id, next); err == nil {
				t.Fatal("phased source revised")
			}
			g.PhaseIn(phased)
			g.ApplyContinuousEffects()
			staticPTCheck(t, g, p, tc.power+1, tc.toughness+bonusT)
			remover := g.PutOnBattlefield(repositoryCard(t, "LEA", "Plains"), a.PlayerID())
			removal := mage.RemoveAllAbilities(p.ID(), core.WhileOnBattlefield)
			removal.SetSourceID(remover.ID())
			g.AddContinuousEffect(removal)
			staticPTCheck(t, g, p, tc.power, tc.toughness)
			if err := g.RevisePrintedStaticPTAbility(p, id, next); err == nil {
				t.Fatal("removed declaration revised")
			}
			g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
			staticPTCheck(t, g, p, tc.power+1, tc.toughness+bonusT)
			if tc.name == "Sedge Troll" {
				b.ManaPool().Add(core.Black, 1)
				if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 1, nil); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				g.DestroyPermanent(g.FindPermanent(p.ID()))
				if g.FindPermanent(p.ID()) == nil || !g.FindPermanent(p.ID()).Tapped || b.ManaPool().TotalMana() != 0 {
					t.Fatal("regeneration changed")
				}
			}
			g.BouncePermanentToHand(g.FindPermanent(p.ID()))
			if lki := g.LKI(p.ID()); lki == nil || lki.Power != tc.power+1 || lki.Toughness != tc.toughness+bonusT {
				t.Fatal("revised LKI lost")
			}
			returned := g.PutOnBattlefield(card, a.PlayerID())
			staticPTCheck(t, g, returned, tc.power+1, tc.toughness+bonusT)
			if err := g.RevisePrintedStaticPTAbility(p, id, change); err == nil {
				t.Fatal("old visit revised")
			}
			g.BouncePermanentToHand(g.FindPermanent(original.ID()))
			staticPTCheck(t, g, returned, tc.power, tc.toughness)
		})
	}
}

func TestRONTextConditionalSourcePTBoundaries(t *testing.T) {
	for _, rule := range []string{
		`If(PlayerPassesFilter(You,ControlsA(IsLandType(Forest))),[PermanentLayerEffect(ThisPermanent,[AdjustPT(1)])])`,
		`If(PlayerPassesFilter(You,ControlsA(IsLandType(Forest))),[PermanentLayerEffect(ThisPermanent,[AdjustPT(ValueX,2)])])`,
		`If(PlayerPassesFilter(Ref_TargetPlayer,ControlsA(IsLandType(Forest))),[PermanentLayerEffect(ThisPermanent,[AdjustPT(1,2)])])`,
		`If(PlayerPassesFilter(You,ControlsA(IsLandType(Forest))),[PermanentLayerEffect(Ref_TargetPermanent,[AdjustPT(1,2)])])`,
		`If(PlayerPassesFilter(You,ControlsA(IsLandType(Forest))),[PermanentLayerEffect(ThisPermanent,[AdjustPT(1,2),AddAbility([Flying])])])`,
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
		`Unless(PlayerPassesFilter(You,ControlsA(IsLandType(Forest))),[PermanentLayerEffect(ThisPermanent,[AdjustPT(1,2)])])`,
		`IfElse(PlayerPassesFilter(You,ControlsA(IsLandType(Forest))),[PermanentLayerEffect(ThisPermanent,[AdjustPT(1,2)])],[PermanentLayerEffect(ThisPermanent,[AdjustPT(2,1)])])`,
		`If(PlayerPassesFilter(You,ControlsNo(IsLandType(Forest))),[PermanentLayerEffect(ThisPermanent,[AdjustPT(1,2)])])`,
		`If(PlayerPassesFilter(You,ControlsA(IsLandType(Desert))),[PermanentLayerEffect(ThisPermanent,[AdjustPT(1,2)])])`,
		`If(PlayerPassesFilter(You,ControlsA(IsLandType(Forest))),[EachPermanentLayerEffect(IsCardtype(Creature),[AdjustPT(1,2)])])`,
		`If(PlayerPassesFilter(You,ControlsA(IsLandType(Forest))),[PermanentLayerEffect(ThisPermanent,[AdjustPT(1,2),AdjustPT(2,1)])])`,
	} {
		g, a, _ := ronGame()
		p := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", rule)(), a.PlayerID())
		change, _ := mage.ReplaceBasicLandWord(core.ForestLandType, core.IslandLandType)
		if err := g.RevisePrintedStaticPTAbility(p, textStaticID(t, p), change); err == nil {
			t.Fatal("unmarked context revised", rule)
		}
	}
}

func TestRONTextConditionalSourcePTOrderedRevisions(t *testing.T) {
	g, a, b := ronGame()
	p := g.PutOnBattlefield(repositoryCard(t, "ARN", "Kird Ape"), a.PlayerID())
	sibling := g.PutOnBattlefield(repositoryCard(t, "ARN", "Kird Ape"), a.PlayerID())
	id := textStaticID(t, p)
	first, _ := mage.ReplaceBasicLandWord(core.ForestLandType, core.IslandLandType)
	second, _ := mage.ReplaceBasicLandWord(core.IslandLandType, core.MountainLandType)
	if err := g.RevisePrintedStaticPTAbility(p, textStaticID(t, sibling), first); err == nil {
		t.Fatal("foreign identity accepted")
	}
	if err := g.RevisePrintedStaticPTAbility(p, id, first); err != nil {
		t.Fatal(err)
	}
	if err := g.RevisePrintedStaticPTAbility(g.FindPermanent(p.ID()), id, second); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), a.PlayerID())
	staticPTCheck(t, g, p, 1, 1)
	land := g.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), b.PlayerID())
	staticPTCheck(t, g, p, 1, 1)
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: land.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
	staticPTCheck(t, g, p, 2, 3)
	source := g.PutOnBattlefield(repositoryCard(t, "LEA", "Plains"), a.PlayerID())
	effect := mage.BecomesBasicLandTargetEffect(land.ID(), core.WhileOnBattlefield, "Forest")
	effect.SetSourceID(source.ID())
	g.AddContinuousEffect(effect)
	staticPTCheck(t, g, p, 1, 1)
	staticPTCheck(t, g, sibling, 2, 3)
	if p.Card.ManaCost().String() != repositoryCard(t, "ARN", "Kird Ape").ManaCost().String() {
		t.Fatal("printed cost changed")
	}
}
