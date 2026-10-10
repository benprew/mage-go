package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTextLandTapLayerRecords(t *testing.T) {
	for _, tc := range opponentLandTapCards {
		t.Run(tc.set+"/"+tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, tc.set, tc.name)
			p := g.PutOnBattlefield(card, a.PlayerID())
			id := p.RuntimeAbilities[0].AbilityID()
			old := g.PutOnBattlefield(repositoryCard(t, "LEA", tc.land), b.PlayerID())
			island := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
			word := core.ForestLandType
			if tc.land == "Mountain" {
				word = core.MountainLandType
			}
			change, err := mage.ReplaceBasicLandWord(word, core.IslandLandType)
			if err != nil {
				t.Fatal(err)
			}
			g.TapPermanent(old)
			if err := g.AddPrintedLandTapTextEffect(p, id, core.Indefinite, change); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(p.ID()).RuntimeAbilities[0].AbilityID() != id {
				t.Fatal("active identity changed")
			}
			g.UntapPermanent(old)
			g.TapPermanent(old)
			g.TapPermanent(island)
			g.PutTriggersOnStack()
			if g.StackSize() != 2 || a.Life() != 20 {
				t.Fatal("old and revised event timing changed")
			}
			clone := g.Clone()
			if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			g.ResolveStack()
			if a.Life() != 22 || b.Life() != 21 {
				t.Fatal("queued or copied body changed")
			}
			if clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("clone shared resolution")
			}
			clone.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != 22 {
				t.Fatal("clone lost queued bodies")
			}
			change, _ = mage.ReplaceBasicLandWord(core.IslandLandType, core.SwampLandType)
			if err := g.AddPrintedLandTapTextEffect(p, id, core.Indefinite, change); err != nil {
				t.Fatal(err)
			}
			g.UntapPermanent(island)
			g.TapPermanent(island)
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("ordered revision lost prior definition")
			}
			clone.UntapPermanent(clone.FindPermanent(island.ID()))
			clone.TapPermanent(clone.FindPermanent(island.ID()))
			clone.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != 23 {
				t.Fatal("clone inherited later revision")
			}
		})
	}
}

func TestRONTextLandTapLayerLifetime(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEA", "Lifetap")
	p := g.PutOnBattlefield(card, a.PlayerID())
	other := g.PutOnBattlefield(repositoryCard(t, "LEA", "Lifetap"), a.PlayerID())
	id := p.RuntimeAbilities[0].AbilityID()
	change, _ := mage.ReplaceBasicLandWord(core.ForestLandType, core.IslandLandType)
	if err := g.AddPrintedLandTapTextEffect(p, other.RuntimeAbilities[0].AbilityID(), core.Indefinite, change); err == nil {
		t.Fatal("foreign declaration accepted")
	}
	if err := g.AddPrintedLandTapTextEffect(p, id, core.Indefinite, change); err != nil {
		t.Fatal(err)
	}
	island := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
	g.TapPermanent(island)
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.ApplyContinuousEffects()
	phased := g.PhaseOut(p.ID())
	if err := g.AddPrintedLandTapTextEffect(p, id, core.Indefinite, change); err == nil {
		t.Fatal("phased source accepted")
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatal("pending controller changed")
	}
	g.PhaseIn(phased)
	ownIsland := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), a.PlayerID())
	g.TapPermanent(ownIsland)
	g.ResolveStack()
	if b.Life() != 21 {
		t.Fatal("phase cycle lost revision or current controller")
	}
	stale := g.FindPermanent(p.ID())
	g.BouncePermanentToHand(stale)
	a.RemoveFromHand(card.ID())
	g.PutOnBattlefield(card, a.PlayerID())
	if err := g.AddPrintedLandTapTextEffect(stale, id, core.Indefinite, change); err == nil {
		t.Fatal("old visit accepted")
	}
	g.UntapPermanent(island)
	g.TapPermanent(island)
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatal("returned source inherited revision")
	}
	forest := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
	g.TapPermanent(forest)
	g.ResolveStack()
	if a.Life() != 23 {
		t.Fatal("returned and sibling printed rules changed")
	}
}

func TestRONTextLandTapLayerRemovalAndTargets(t *testing.T) {
	factory := permanentScopeFactory(t, "Enchantment", `TriggerA(WhenAPermanentBecomesTapped(And([IsLandType(Forest),ControlledByAPlayer(Opponent)])),Targeted([TargetPermanent(IsLandType(Forest))],ActionList([DestroyPermanent(Ref_TargetPermanent)])))`)
	g, a, b := ronModalGame(0)
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	forest := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
	island := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
	id := p.RuntimeAbilities[0].AbilityID()
	g.TapPermanent(forest)
	change, _ := mage.ReplaceBasicLandWord(core.ForestLandType, core.IslandLandType)
	if err := g.AddPrintedLandTapTextEffect(p, id, core.Indefinite, change); err != nil {
		t.Fatal(err)
	}
	a.targets = []uuid.UUID{forest.ID()}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("pending target declaration changed")
	}
	g.TapPermanent(island)
	a.targets = []uuid.UUID{island.ID()}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("fresh target declaration did not change")
	}
	remover := g.PutOnBattlefield(repositoryCard(t, "LEA", "Swamp"), a.PlayerID())
	removal := mage.RemoveAllAbilities(p.ID(), core.WhileOnBattlefield)
	removal.SetSourceID(remover.ID())
	g.AddContinuousEffect(removal)
	g.ApplyContinuousEffects()
	if err := g.AddPrintedLandTapTextEffect(p, id, core.Indefinite, change); err == nil {
		t.Fatal("removed declaration accepted")
	}
	g.ResolveStack()
	if g.FindPermanent(forest.ID()) != nil || g.FindPermanent(island.ID()) != nil {
		t.Fatal("queued target declarations lost after removal")
	}
	newIsland := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
	g.TapPermanent(newIsland)
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("removed ability detected an event")
	}
	clone := g.Clone()
	clone.RemoveFromBattlefield(clone.FindPermanent(remover.ID()))
	clone.ApplyContinuousEffects()
	clone.UntapPermanent(clone.FindPermanent(newIsland.ID()))
	clone.TapPermanent(clone.FindPermanent(newIsland.ID()))
	clone.PutTriggersOnStack()
	if clone.StackSize() != 1 {
		t.Fatal("clone restoration lost revised declaration")
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("clone restored original ability")
	}
	g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
	g.ApplyContinuousEffects()
	g.UntapPermanent(newIsland)
	g.TapPermanent(newIsland)
	a.targets = []uuid.UUID{newIsland.ID()}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("restored ability lost revision")
	}
	g.BouncePermanentToHand(newIsland)
	b.RemoveFromHand(newIsland.Card.ID())
	returned := g.PutOnBattlefield(newIsland.Card, b.PlayerID())
	g.ResolveStack()
	if g.FindPermanent(returned.ID()) == nil {
		t.Fatal("saved target followed a returned land")
	}
}

func TestRONTextLandTapLayerBoundaries(t *testing.T) {
	for _, rule := range []string{
		`TriggerA(WhenAPlayerCastsASpell(AnyPlayer,IsColor(Red)),ActionList([GainLife(Integer(1))]))`,
		`TriggerA(WhenAPermanentBecomesTapped(IsCardtype(Artifact)),ActionList([GainLife(Integer(1))]))`,
	} {
		g, a, _ := ronGame()
		p := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", rule)(), a.PlayerID())
		change, _ := mage.ReplaceBasicLandWord(core.ForestLandType, core.IslandLandType)
		if err := g.AddPrintedLandTapTextEffect(p, p.RuntimeAbilities[0].AbilityID(), core.Indefinite, change); err == nil {
			t.Fatal("other event acquired land-tap revision")
		}
	}
}
