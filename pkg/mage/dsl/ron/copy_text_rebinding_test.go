package ron

import (
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCopyTextRebindingRecords(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			g, a, b := ronGame()
			chooser := &ronEntryCopyPlayer{Player: a, accept: true}
			g.SetPlayerAt(0, chooser)
			donor := g.PutOnBattlefield(repositoryCard(t, "LEA", "Gauntlet of Might"), b.PlayerID())
			red := g.PutOnBattlefield(textColorCard(core.Red), a.PlayerID())
			green := g.PutOnBattlefield(textColorCard(core.Green), a.PlayerID())
			forest := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), a.PlayerID())
			card := repositoryCard(t, set, "Copy Artifact")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Blue, 1)
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			p := g.FindPermanent(card.ID())
			color, _ := mage.ReplaceColorWord(core.Red, core.Green)
			land, _ := mage.ReplaceBasicLandWord(core.MountainLandType, core.ForestLandType)
			if err := g.AddPrintedStaticPTTextEffect(p, textStaticID(t, p), core.Indefinite, color); err != nil {
				t.Fatal(err)
			}
			p = g.FindPermanent(p.ID())
			if err := g.AddPrintedManaTapTextEffect(p, manaLayerID(t, p), core.Indefinite, land); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(red.ID()).CurrentPower(g) != 2 || g.FindPermanent(green.ID()).CurrentPower(g) != 2 || !g.FindPermanent(p.ID()).HasType(core.TypeEnchantment) {
				t.Fatal("copied text or added enchantment type was lost")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), forest.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().Count(core.Green) != 1 || a.ManaPool().Count(core.Red) != 1 {
				t.Fatal("revised land word or unchanged mana symbol was lost")
			}
			clone := g.Clone()
			if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.AddCardType(mage.This, core.TypeCreature), mage.SetBasePowerToughness(mage.This, mage.Fixed(4), mage.Fixed(4))), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			p = g.FindPermanent(p.ID())
			chooser.pick = func([]*mage.Permanent, mage.GameReader) *mage.Permanent { return p }
			next := g.PutOnBattlefield(repositoryCard(t, set, "Clone"), a.PlayerID())
			if next == nil || !next.HasType(core.TypeEnchantment) || next.HasType(core.TypeCreature) || next.Name() != "Gauntlet of Might" || g.FindPermanent(green.ID()).CurrentPower(g) != 2 || g.FindPermanent(red.ID()).CurrentPower(g) != 3 {
				t.Fatal("later copy inherited text changes or lost copy exceptions")
			}
			second, _ := mage.ReplaceColorWord(core.Green, core.Blue)
			p = g.FindPermanent(p.ID())
			if err := g.AddPrintedStaticPTTextEffect(p, textStaticID(t, p), core.EndOfTurn, second); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(green.ID()).CurrentPower(g) != 1 || clone.FindPermanent(green.ID()).CurrentPower(clone) != 2 {
				t.Fatal("ordered change altered a prior clone")
			}
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			g.PhaseIn(g.PhaseOut(p.ID()))
			if chooser.offers != 2 || g.FindPermanent(green.ID()).CurrentPower(g) != 2 {
				t.Fatal("phase cycle repeated entry or lost text")
			}
			g.BouncePermanentToHand(g.FindPermanent(p.ID()))
			if len(a.Hand()) != 1 || a.Hand()[0].Name() != "Copy Artifact" || a.Hand()[0].HasType(core.TypeArtifact) || g.LKI(p.ID()).Name != "Gauntlet of Might" {
				t.Fatal("physical departure or copied LKI was lost")
			}
			if g.FindPermanent(green.ID()).CurrentPower(g) != 1 || clone.FindPermanent(p.ID()) == nil || clone.FindPermanent(green.ID()).CurrentPower(clone) != 2 || donor.Name() != "Gauntlet of Might" {
				t.Fatal("departure changed a clone, donor or independent copy")
			}
		})
	}
}

func TestRONCopyTextRebindingRetainedColors(t *testing.T) {
	for _, cost := range []string{`[ManaCostU]`, `[]`} {
		g, a, b := ronGame()
		chooser := &ronEntryCopyPlayer{Player: a, accept: true}
		g.SetPlayerAt(0, chooser)
		nodes, err := Parse(strings.NewReader(`Card(Name:"Donor",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[Activated(PayMana([]),Targeted([TargetPermanent(IsColor(Red))],ActionList([TapPermanent(Ref_TargetPermanent)])))])`), "donor.ron")
		if err != nil {
			t.Fatal(err)
		}
		factory, err := Compile(nodes[0])
		if err != nil {
			t.Fatal(err)
		}
		donor := factory()
		g.PutOnBattlefield(donor, b.PlayerID())
		p := g.PutOnBattlefield(copyColorFactory(t, cost, "", `KeepColor,AddCardtypes([Enchantment])`)(), a.PlayerID())
		before := slices.Clone(p.Colors())
		change, _ := mage.ReplaceColorWord(core.Red, core.Green)
		if err := g.RevisePrintedActivatedAbility(p, p.RuntimeAbilities[0].AbilityID(), change); err != nil {
			t.Fatal(err)
		}
		green := g.PutOnBattlefield(textColorCard(core.Green), a.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{green.ID()}); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if !g.FindPermanent(green.ID()).Tapped || !slices.Equal(g.FindPermanent(p.ID()).Colors(), before) || !g.FindPermanent(p.ID()).HasType(core.TypeEnchantment) {
			t.Fatal("fresh target binding, retained color or added type was lost")
		}
	}
}
