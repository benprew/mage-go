package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestManaProductionsForAbility(t *testing.T) {
	t.Run("dedicated mana ability", func(t *testing.T) {
		got := ManaProductionsForAbility(NewMultiManaAbility(
			ManaProduction{Color: Colorless, Amount: 2},
		))
		if len(got) != 1 || got[0].Color != Colorless || got[0].Amount != 2 {
			t.Fatalf("unexpected productions: %+v", got)
		}
	})

	t.Run("tap activated mana ability", func(t *testing.T) {
		ability := NewActivatedAbility(AddMana(Green, 1), Tap())
		got := ManaProductionsForAbility(ability)
		if len(got) != 1 || got[0].Color != Green || got[0].Amount != 1 {
			t.Fatalf("unexpected productions: %+v", got)
		}
	})

	t.Run("non-mana ability", func(t *testing.T) {
		ability := NewActivatedAbility(DrawCards(Fixed(1)), Tap())
		if got := ManaProductionsForAbility(ability); got != nil {
			t.Fatalf("unexpected productions: %+v", got)
		}
	})
}

func chosenColorTestLand(t *testing.T, chosen Color, after Color) (*Game, Player, *Permanent, *ManaAbility) {
	t.Helper()
	g, a, _ := randomTestGame()
	ability := NewDynamicManaAbility(ChosenColorManaProductions,
		FuncEffect("change chosen color", EffectProperties{},
			func(g *Game, sourceID, _ uuid.UUID, _ []uuid.UUID) error {
				perm := g.FindPermanent(sourceID)
				if perm != nil {
					perm.ChosenColor = after
				}
				return nil
			}),
	)
	card := NewLand("Dynamic Mana Test Land", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	perm := g.PutOnBattlefield(card, a.PlayerID())
	perm.ChosenColor = chosen
	return g, a, perm, ability
}

func TestDynamicManaAbility_AdvertisesCurrentChosenColor(t *testing.T) {
	g, _, perm, ability := chosenColorTestLand(t, Blue, Red)

	got := ManaProductionsForAbilityInGame(ability, g, perm.ID())
	if len(got) != 1 || got[0] != (ManaProduction{Color: Blue, Amount: 1}) {
		t.Fatalf("productions = %+v, want one blue", got)
	}
	perm.ChosenColor = Black
	got = ManaProductionsForAbilityInGame(ability, g, perm.ID())
	if len(got) != 1 || got[0].Color != Black {
		t.Fatalf("productions after state change = %+v, want one black", got)
	}
}

func TestDynamicManaAbility_ProducesThenRunsPostProductionEffect(t *testing.T) {
	g, player, perm, _ := chosenColorTestLand(t, Blue, Red)

	if err := g.TapForMana(player.PlayerID(), perm.ID()); err != nil {
		t.Fatalf("TapForMana: %v", err)
	}
	if got := player.ManaPool().Count(Blue); got != 1 {
		t.Fatalf("blue mana = %d, want 1", got)
	}
	if got := player.ManaPool().Count(Red); got != 0 {
		t.Errorf("red mana = %d, post-production state leaked into production", got)
	}
	if perm.ChosenColor != Red {
		t.Errorf("ChosenColor = %s, want Red after production", perm.ChosenColor)
	}
	if g.GetStack().Size() != 0 {
		t.Error("dynamic mana ability used the stack")
	}
}

func TestDynamicManaAbility_SolverUsesOnlyCurrentChosenColor(t *testing.T) {
	g, player, _, _ := chosenColorTestLand(t, Black, Red)

	if !g.CanAfford(player.PlayerID(), ManaCost{Black: 1}, nil) {
		t.Error("CanAfford did not advertise current black production")
	}
	if g.CanAfford(player.PlayerID(), ManaCost{White: 1}, nil) {
		t.Error("CanAfford advertised an unchosen color")
	}
}

func TestDynamicManaAbility_AutoTapExecutesAdvertisedProduction(t *testing.T) {
	g, player, perm, _ := chosenColorTestLand(t, Blue, Red)

	if err := g.AutoTapForCost(player.PlayerID(), ManaCost{Blue: 1}); err != nil {
		t.Fatalf("AutoTapForCost: %v", err)
	}
	if !perm.Tapped {
		t.Error("dynamic mana source was not tapped")
	}
	if got := player.ManaPool().Count(Blue); got != 1 {
		t.Errorf("blue mana = %d, want 1", got)
	}
	if perm.ChosenColor != Red {
		t.Errorf("ChosenColor = %s, want Red after auto-tap", perm.ChosenColor)
	}
}

func TestDynamicManaAbility_CloneReadsIndependentSourceState(t *testing.T) {
	g, player, perm, _ := chosenColorTestLand(t, Black, Red)
	clone := g.Clone()
	clonePerm := clone.FindPermanent(perm.ID())
	if clonePerm == nil {
		t.Fatal("clone missing dynamic mana source")
	}
	g.MutablePermanent(perm.ID()).ChosenColor = White

	if !clone.CanAfford(player.PlayerID(), ManaCost{Black: 1}, nil) {
		t.Error("clone lost its black chosen-color production")
	}
	if clone.CanAfford(player.PlayerID(), ManaCost{White: 1}, nil) {
		t.Error("clone observed original's chosen-color mutation")
	}
	if !g.CanAfford(player.PlayerID(), ManaCost{White: 1}, nil) {
		t.Error("original did not observe its chosen-color mutation")
	}
}

func TestDynamicManaAbility_PostProductionMutationIsCloneSafe(t *testing.T) {
	g, player, perm, _ := chosenColorTestLand(t, Black, Red)
	clone := g.Clone()

	if err := clone.TapForMana(player.PlayerID(), perm.ID()); err != nil {
		t.Fatalf("clone TapForMana: %v", err)
	}
	clonePerm := clone.FindPermanent(perm.ID())
	if clonePerm == nil || clonePerm.ChosenColor != Red {
		t.Fatalf("clone post-production color = %v, want Red", clonePerm)
	}
	if perm.ChosenColor != Black {
		t.Errorf("clone post-production mutation leaked to original: got %s", perm.ChosenColor)
	}
}

func TestManaBonusColorResolve(t *testing.T) {
	if got := ManaBonusColor(Green).Resolve(Blue); got != Green {
		t.Fatalf("fixed green bonus resolved to %v", got)
	}
	if got := MatchProduced.Resolve(Blue); got != Blue {
		t.Fatalf("match-produced bonus resolved to %v", got)
	}
}
