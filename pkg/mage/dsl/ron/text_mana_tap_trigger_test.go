package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTextManaTapTriggerRecords(t *testing.T) {
	for _, name := range []string{"Mana Flare", "Wild Growth", "Manabarbs"} {
		for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
			t.Run(set+"/"+name, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, set, name)
				source := g.PutOnBattlefield(card, a.PlayerID())
				land := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
				if name == "Wild Growth" {
					g.Attach(source.ID(), land.ID())
				}
				id := source.RuntimeAbilities[0].AbilityID()
				change, _ := mage.ReplaceColorWord(core.Green, core.Blue)
				if err := g.RevisePrintedManaTapTrigger(source, id, change); err != nil {
					t.Fatal(err)
				}
				if g.FindPermanent(source.ID()).RuntimeAbilities[0].AbilityID() != id {
					t.Fatal("active identity changed")
				}
				if err := g.TapForMana(b.PlayerID(), land.ID()); err != nil {
					t.Fatal(err)
				}
				blue, green := 1, 0
				if name == "Mana Flare" {
					blue = 2
				}
				if name == "Wild Growth" {
					green = 1
				}
				if b.ManaPool().Count(core.Blue) != blue || b.ManaPool().Count(core.Green) != green || b.Life() != 20 {
					t.Fatal("text revision changed mana symbols, actor or timing")
				}
				g.PutTriggersOnStack()
				stack := 0
				if name == "Manabarbs" {
					stack = 1
				}
				if g.StackSize() != stack {
					t.Fatal("trigger classification changed")
				}
				g.ResolveStack()
				life := 20
				if name == "Manabarbs" {
					life = 19
				}
				if b.Life() != life || a.Life() != 20 {
					t.Fatal("damage actor changed")
				}
			})
		}
	}
}

func TestRONTextManaTapTriggerBonusLifecycle(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEA", "Wild Growth")
	source := g.PutOnBattlefield(card, a.PlayerID())
	sibling := g.PutOnBattlefield(repositoryCard(t, "LEA", "Wild Growth"), a.PlayerID())
	land := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
	g.Attach(source.ID(), land.ID())
	id := source.RuntimeAbilities[0].AbilityID()
	change, _ := mage.ReplaceColorWord(core.Green, core.Blue)
	if err := g.RevisePrintedManaTapTrigger(source, sibling.RuntimeAbilities[0].AbilityID(), change); err == nil {
		t.Fatal("foreign identity accepted")
	}
	if err := g.RevisePrintedManaTapTrigger(source, id, change); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	phased := g.PhaseOut(source.ID())
	if err := g.RevisePrintedManaTapTrigger(source, id, change); err == nil {
		t.Fatal("phased source accepted")
	}
	g.PhaseIn(phased)
	remover := g.PutOnBattlefield(repositoryCard(t, "LEA", "Swamp"), a.PlayerID())
	removal := mage.RemoveAllAbilities(source.ID(), core.WhileOnBattlefield)
	removal.SetSourceID(remover.ID())
	g.AddContinuousEffect(removal)
	g.ApplyContinuousEffects()
	if err := g.RevisePrintedManaTapTrigger(source, id, change); err == nil {
		t.Fatal("removed declaration accepted")
	}
	if err := g.TapForMana(b.PlayerID(), land.ID()); err != nil {
		t.Fatal(err)
	}
	if b.ManaPool().Count(core.Green) != 0 {
		t.Fatal("removed bonus produced mana")
	}
	g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
	g.ApplyContinuousEffects()
	g.UntapPermanent(land)
	if err := g.TapForMana(b.PlayerID(), land.ID()); err != nil {
		t.Fatal(err)
	}
	if b.ManaPool().Count(core.Green) != 1 {
		t.Fatal("restoration lost revised bonus")
	}
	if err := clone.TapForMana(b.PlayerID(), land.ID()); err != nil {
		t.Fatal(err)
	}
	if clone.GetPlayer(b.PlayerID()).ManaPool().Count(core.Green) != 1 {
		t.Fatal("clone lost bonus")
	}
	stale := g.FindPermanent(source.ID())
	g.BouncePermanentToHand(stale)
	a.RemoveFromHand(card.ID())
	returned := g.PutOnBattlefield(card, a.PlayerID())
	if err := g.RevisePrintedManaTapTrigger(stale, id, change); err == nil {
		t.Fatal("old visit accepted")
	}
	if returned.RuntimeAbilities[0].AbilityID() != id {
		t.Fatal("printed origin lost")
	}
}

func TestRONTextManaTapTriggerQueuedTargets(t *testing.T) {
	rule := `TriggerA(WhenAPlayerTapsAPermanentForMana(AnyPlayer,IsCardtype(Land)),Targeted([TargetPermanent(IsColor(Red))],ActionList([DestroyPermanent(Ref_TargetPermanent)])))`
	g, a, b := ronModalGame(0)
	source := g.PutOnBattlefield(permanentScopeFactory(t, "Enchantment", rule)(), a.PlayerID())
	land := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
	red := g.PutOnBattlefield(repositoryCard(t, "LEA", "Mons's Goblin Raiders"), b.PlayerID())
	green := g.PutOnBattlefield(repositoryCard(t, "LEA", "Llanowar Elves"), b.PlayerID())
	if err := g.TapForMana(b.PlayerID(), land.ID()); err != nil {
		t.Fatal(err)
	}
	change, _ := mage.ReplaceColorWord(core.Red, core.Green)
	if err := g.RevisePrintedManaTapTrigger(source, source.RuntimeAbilities[0].AbilityID(), change); err != nil {
		t.Fatal(err)
	}
	a.targets = []uuid.UUID{red.ID()}
	g.PutTriggersOnStack()
	g.UntapPermanent(land)
	if err := g.TapForMana(b.PlayerID(), land.ID()); err != nil {
		t.Fatal(err)
	}
	a.targets = []uuid.UUID{green.ID()}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("pending or revised targets lost")
	}
	clone := g.Clone()
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.BouncePermanentToHand(source)
	g.ResolveStack()
	if g.FindPermanent(red.ID()) != nil || g.FindPermanent(green.ID()) != nil {
		t.Fatal("queued targets changed after revision or departure")
	}
	if clone.FindPermanent(red.ID()) == nil || clone.FindPermanent(green.ID()) == nil {
		t.Fatal("clone shared resolution")
	}
	clone.ResolveStack()
	if clone.FindPermanent(red.ID()) != nil || clone.FindPermanent(green.ID()) != nil {
		t.Fatal("clone lost queued revisions")
	}
}

func TestRONTextManaTapTriggerBoundaries(t *testing.T) {
	for _, rule := range []string{
		`TriggerA(WhenAPermanentBecomesTapped(And([IsLandType(Forest),ControlledByAPlayer(Opponent)])),ActionList([GainLife(Integer(1))]))`,
		`TriggerA(WhenAPlayerCastsASpell(AnyPlayer,IsColor(Red)),ActionList([GainLife(Integer(1))]))`,
	} {
		g, a, _ := ronGame()
		source := g.PutOnBattlefield(permanentScopeFactory(t, "Enchantment", rule)(), a.PlayerID())
		change, _ := mage.ReplaceColorWord(core.Red, core.Green)
		if err := g.RevisePrintedManaTapTrigger(source, source.RuntimeAbilities[0].AbilityID(), change); err == nil {
			t.Fatal("other event acquired mana-tap revision")
		}
	}
	for _, name := range []string{"Magical Hack", "Sleight of Mind"} {
		if _, err := Compile(staticBaseRecord(t, "LEA", name)); err == nil {
			t.Fatalf("%s lost unsupported boundary", name)
		}
	}
}

func TestRONTextManaTapTriggerManabarbsFacts(t *testing.T) {
	g, a, b := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "LEA", "Manabarbs"), a.PlayerID())
	landCard := repositoryCard(t, "LEA", "Island")
	land := g.PutOnBattlefield(landCard, b.PlayerID())
	if err := g.TapForMana(b.PlayerID(), land.ID()); err != nil {
		t.Fatal(err)
	}
	change, _ := mage.ReplaceBasicLandWord(core.IslandLandType, core.ForestLandType)
	if err := g.RevisePrintedManaTapTrigger(source, source.RuntimeAbilities[0].AbilityID(), change); err != nil {
		t.Fatal(err)
	}
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.ApplyContinuousEffects()
	g.BouncePermanentToHand(land)
	b.RemoveFromHand(landCard.ID())
	returned := g.PutOnBattlefield(landCard, a.PlayerID())
	g.PhaseOut(source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("pending trigger lost after phasing")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if b.Life() != 18 || a.Life() != 20 || returned.Damage != 0 {
		t.Fatal("queued copies lost event player or followed returned land")
	}
}

func TestRONTextManaTapTriggerPayment(t *testing.T) {
	for _, name := range []string{"Mana Flare", "Wild Growth"} {
		t.Run(name, func(t *testing.T) {
			g, a, _ := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, "LEA", name), a.PlayerID())
			land := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), a.PlayerID())
			if name == "Wild Growth" {
				g.Attach(source.ID(), land.ID())
			}
			change, _ := mage.ReplaceColorWord(core.Green, core.Blue)
			if err := g.RevisePrintedManaTapTrigger(source, source.RuntimeAbilities[0].AbilityID(), change); err != nil {
				t.Fatal(err)
			}
			cost := core.ManaCost{Blue: 2}
			if name == "Wild Growth" {
				cost = core.ManaCost{Blue: 1, Green: 1}
			}
			impossible := cost
			impossible.Black = 1
			if err := g.AutoTapForCost(a.PlayerID(), impossible); err == nil {
				t.Fatal("unaffordable revised bonus payment succeeded")
			}
			if land.Tapped || a.ManaPool().TotalMana() != 0 || g.StackSize() != 0 {
				t.Fatal("failed revised bonus payment leaked state")
			}
			if err := g.AutoTapForCost(a.PlayerID(), cost); err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().TotalMana() != 2 || g.StackSize() != 0 {
				t.Fatal("revised bonus planning and execution disagree")
			}
		})
	}
}
