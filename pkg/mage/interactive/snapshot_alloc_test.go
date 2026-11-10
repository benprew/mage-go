package interactive

import (
	"reflect"
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func snapshotAllocationGame() (*mage.Game, *HumanPlayer, *mage.Card) {
	human := NewHumanPlayer("You")
	game := mage.NewGame(human, mage.NewBasePlayer("Opponent"))
	card := mage.NewCreature("Bird", "{1}{U}", 1, 2, mage.WithSubTypes("Bird"), mage.WithKeyword(core.Flying), mage.WithAbility(mage.ProtectionFromColor(core.Red)))
	human.AddToHand(card)
	human.AddToHand(card.NewCopy())
	return game, human, card
}

func TestSnapshotMetadataAllocationBudget(t *testing.T) {
	game, _, _ := snapshotAllocationGame()
	SnapshotGameState(game, 0)
	allocations := testing.AllocsPerRun(100, func() { SnapshotGameState(game, 0) })
	if allocations > 8 {
		t.Fatalf("snapshot allocations = %.0f, budget 8", allocations)
	}
}

func TestSnapshotMetadataTracksDefinitionChanges(t *testing.T) {
	game, _, card := snapshotAllocationGame()
	before := SnapshotGameState(game, 0)
	card.AddType(core.TypeArtifact)
	card.AddSubType("Wizard")
	card.AddAbility(mage.StaticKeyword(core.Haste))
	after := SnapshotGameState(game, 0)
	if after.You.Hand[0].Types != "Creature Artifact" || after.You.Hand[0].SubTypes != "Bird Wizard" || !strings.Contains(after.You.Hand[0].RulesText, "Haste") {
		t.Fatalf("stale metadata after card changes: %+v", after.You.Hand[0])
	}
	if !reflect.DeepEqual(before.You.Hand[1], after.You.Hand[1]) {
		t.Fatalf("changing one instance changed its copy: before=%+v copy=%+v", before.You.Hand[0], after.You.Hand[1])
	}
	card.CloneFrom(mage.NewCreature("Bird", "{R}", 4, 5, mage.WithSubTypes("Dragon"), mage.WithKeyword(core.Trample)))
	copied := SnapshotGameState(game, 0).You.Hand[0]
	if copied.ManaCost != "{R}" || copied.SubTypes != "Dragon" || copied.RulesText != "Trample" || copied.Power != 4 || copied.Toughness != 5 {
		t.Fatalf("stale metadata after copy: %+v", copied)
	}
}

func TestSnapshotMetadataDoesNotAliasSnapshots(t *testing.T) {
	game, human, _ := snapshotAllocationGame()
	before := SnapshotGameState(game, 0)
	before.You.Hand[0].RulesText = "modified"
	human.AddToGraveyard(mage.NewCreature("Bear", "{G}", 2, 2))
	after := SnapshotGameState(game, 0)
	if after.You.Hand[0].RulesText == "modified" || len(before.You.Graveyard) != 0 || len(after.You.Graveyard) != 1 {
		t.Fatal("snapshots share mutable state")
	}
}

func BenchmarkSnapshotMetadata(b *testing.B) {
	game, _, _ := snapshotAllocationGame()
	SnapshotGameState(game, 0)
	b.ReportAllocs()
	for b.Loop() {
		SnapshotGameState(game, 0)
	}
}

func TestSnapshotMetadataKeepsBattlefieldValuesLive(t *testing.T) {
	game, human, card := snapshotAllocationGame()
	creature := mage.NewPermanent(card.NewCopy(), human.PlayerID())
	land := mage.NewPermanent(mage.NewLand("City of Brass"), human.PlayerID())
	game.AddToBattlefield(creature, land)
	before := SnapshotGameState(game, 0)
	game.AddContinuousEffect(mage.SetBasePT(creature.ID(), 5, 6))
	game.AddContinuousEffect(mage.BecomesBasicLandTargetEffect(land.ID(), core.Indefinite, "Forest"))
	creature.Tapped = true
	creature.Damage = 2
	after := SnapshotGameState(game, 0)
	if got := after.You.Battlefield[0]; got.Power != 5 || got.Toughness != 6 || !got.Tapped || got.Damage != 2 {
		t.Fatalf("stale battlefield values: %+v", got)
	}
	if got := after.You.Battlefield[1].SubTypes; got != "Forest" {
		t.Fatalf("subtypes = %q, want Forest", got)
	}
	if before.You.Battlefield[0].Power != 1 || before.You.Battlefield[1].SubTypes != "" {
		t.Fatal("later snapshot changed the earlier snapshot")
	}
}

func TestSnapshotMetadataResetsForNewGame(t *testing.T) {
	game, human, _ := snapshotAllocationGame()
	SnapshotGameState(game, 0)
	if len(human.snapshotCache.cards) == 0 {
		t.Fatal("cache was not populated")
	}
	human.SetHand(nil)
	next := mage.NewGame(human, mage.NewBasePlayer("Next"))
	SnapshotGameState(next, 0)
	if len(human.snapshotCache.cards) != 0 {
		t.Fatal("cache retained definitions from the previous game")
	}
}
