package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestManaColorsCouldProduceOptions(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ability := NewManaAbilityOfColorsOpponentPermanentsCouldProduce(IsLand)
	stone := g.PutOnBattlefield(NewArtifact("Options", "", WithAbility(ability)), a.PlayerID())
	g.PutOnBattlefield(NewLand("Own", WithManaAbility(Black)), a.PlayerID())
	g.PutOnBattlefield(NewArtifact("Other", "", WithManaAbility(White)), b.PlayerID())
	g.PutOnBattlefield(NewLand("Colorless", WithManaAbility(Colorless)), b.PlayerID())
	if got := g.ManaTypesAbilityCouldProduce(ability, stone.ID()); len(got) != 0 {
		t.Fatalf("empty options = %v", got)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), stone.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if !stone.Tapped || a.ManaPool().TotalMana() != 0 || g.StackSize() != 0 {
		t.Fatal("empty production must pay the tap cost without using the stack")
	}
	stone.Tapped = false
	land := g.PutOnBattlefield(NewLand("Mixed", WithMultiManaAbility(ManaProduction{Color: Blue, Amount: 2}, ManaProduction{Color: Green, Amount: 1})), b.PlayerID())
	land.Tapped = true
	for range 3 {
		if got := g.ManaTypesAbilityCouldProduce(ability, stone.ID()); !slices.Equal(got, []Color{Blue, Green}) {
			t.Fatalf("options = %v", got)
		}
		if !g.CanAfford(a.PlayerID(), ManaCost{Blue: 1}, nil) || !g.CanAfford(a.PlayerID(), ManaCost{Green: 1}, nil) || g.CanAfford(a.PlayerID(), ManaCost{White: 1}, nil) || g.CanAfford(a.PlayerID(), ManaCost{Blue: 1, Green: 1}, nil) {
			t.Fatal("solver changed one color choice into all colors or an unrestricted choice")
		}
	}
	if stone.Tapped || a.ManaPool().TotalMana() != 0 || !land.Tapped {
		t.Fatal("query changed resources")
	}
	clone := g.Clone()
	if err := g.AutoTapForCost(a.PlayerID(), ManaCost{Green: 1}); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(stone.ID()).Tapped || a.ManaPool().Count(Green) != 1 || a.ManaPool().TotalMana() != 1 || clone.FindPermanent(stone.ID()).Tapped {
		t.Fatal("automatic payment or clone isolation")
	}
}

func TestManaColorsCouldProduceRecursion(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	leftAbility := NewManaAbilityOfColorsOpponentPermanentsCouldProduce(IsLand)
	rightAbility := NewManaAbilityOfColorsOpponentPermanentsCouldProduce(IsLand)
	left := g.PutOnBattlefield(NewLand("Left", WithAbility(leftAbility)), a.PlayerID())
	right := g.PutOnBattlefield(NewLand("Right", WithAbility(rightAbility)), b.PlayerID())
	for _, p := range []*Permanent{left, right} {
		if got := g.ManaTypesAbilityCouldProduce(p.RuntimeAbilities[0], p.ID()); len(got) != 0 {
			t.Fatalf("undefined cycle = %v", got)
		}
	}
	anchor := g.PutOnBattlefield(NewLand("Anchor", WithManaAbility(Red)), a.PlayerID())
	for _, p := range []*Permanent{left, right} {
		if got := g.ManaTypesAbilityCouldProduce(p.RuntimeAbilities[0], p.ID()); !slices.Equal(got, []Color{Red}) {
			t.Fatalf("anchored cycle = %v", got)
		}
	}
	g.PhaseOut(anchor.ID())
	if got := g.ManaTypesAbilityCouldProduce(leftAbility, left.ID()); len(got) != 0 {
		t.Fatalf("phased anchor = %v", got)
	}
	g.PhaseIn([]uuid.UUID{anchor.ID()})
	if err := ApplyEffect(g, ReplaceTappedManaTypeForPlayerThisTurn(You, IsLand, Blue), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if got := g.ManaTypesAbilityCouldProduce(rightAbility, right.ID()); !slices.Equal(got, []Color{Blue}) {
		t.Fatalf("replacement on anchor = %v", got)
	}
}

func TestManaColorsCouldProducePaymentAndReplacements(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ability := NewManaAbilityOfColorsOpponentPermanentsCouldProduce(IsLand)
	stone := g.PutOnBattlefield(NewArtifact("Options", "", WithAbility(ability)), a.PlayerID())
	land := g.PutOnBattlefield(NewLand("Costly", WithAbility(Activate(Costs(ManaPayment("{9}"), TapCost(This)), AddManaAmount(White, Fixed(1))))), b.PlayerID())
	land.Tapped = true
	if got := g.ManaTypesAbilityCouldProduce(ability, stone.ID()); !slices.Equal(got, []Color{White}) {
		t.Fatalf("unpayable production = %v", got)
	}
	for _, color := range []Color{Blue, Red} {
		if err := ApplyEffect(g, ReplaceTappedManaTypeForPlayerThisTurn(You, IsLand, color), uuid.Nil, b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if got := g.ManaTypesAbilityCouldProduce(ability, stone.ID()); len(got) != 2 || !slices.Contains(got, Blue) || !slices.Contains(got, Red) {
		t.Fatalf("replacement orders = %v", got)
	}
	if err := ApplyEffect(g, ReplaceTappedManaTypeUntilLeaves(This, Blue, Green), stone.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if !g.CanAfford(a.PlayerID(), ManaCost{Green: 1}, nil) || g.CanAfford(a.PlayerID(), ManaCost{Blue: 1}, nil) {
		t.Fatal("source replacement disagrees with options")
	}
	if err := g.AutoTapForCost(a.PlayerID(), ManaCost{Green: 1}); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().Count(Green) != 1 || a.ManaPool().TotalMana() != 1 {
		t.Fatal("source replacement was not applied to the selected color")
	}
}

func TestManaColorsCouldProduceCounterPaymentResult(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindNumber("removed counters")
	land := g.PutOnBattlefield(NewLand("Storage", WithAbility(Activate(Costs(TapCost(This), RemoveAnyCountersCost(Storage, paid)), AddManaAmount(Black, FromNumber(paid))))), b.PlayerID())
	ability := NewManaAbilityOfColorsOpponentPermanentsCouldProduce(IsLand)
	stone := g.PutOnBattlefield(NewArtifact("Options", "", WithAbility(ability)), a.PlayerID())
	g.resolution.numberBindings = map[*NumberBinding]int{paid: -1}
	if got := g.ManaTypesAbilityCouldProduce(ability, stone.ID()); !slices.Equal(got, []Color{Black}) {
		t.Fatalf("hypothetical counter payment = %v", got)
	}
	if land.Counters[Storage] != 0 || land.Tapped || g.resolution.numberBindings[paid] != -1 || b.ManaPool().TotalMana() != 0 {
		t.Fatal("query changed payment state")
	}
}

func TestManaColorsCouldProduceLifecycle(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ability := NewManaAbilityOfColorsOpponentPermanentsCouldProduce(IsLand)
	stone := g.PutOnBattlefield(NewArtifact("Options", "", WithAbility(ability)), a.PlayerID())
	g.PutOnBattlefield(NewLand("Own", WithManaAbility(Black)), a.PlayerID())
	card := NewLand("Opponent", WithManaAbility(Blue))
	card.SetOwner(b.PlayerID())
	land := g.PutOnBattlefield(card, b.PlayerID())
	clone := g.Clone()
	g.AddContinuousEffect(BecomesBasicLandTargetEffect(land.ID(), EndOfTurn, "Forest"))
	if !g.CanAfford(a.PlayerID(), ManaCost{Green: 1}, nil) || g.CanAfford(a.PlayerID(), ManaCost{Blue: 1}, nil) || !clone.CanAfford(a.PlayerID(), ManaCost{Blue: 1}, nil) {
		t.Fatal("current layers or clone queries")
	}
	if err := ApplyEffect(g, Until(EndOfTurn, ChangeController(This, You)), stone.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.ApplyContinuousEffects()
	if got := g.ManaTypesAbilityCouldProduce(ability, stone.ID()); !slices.Equal(got, []Color{Black}) {
		t.Fatalf("new controller options = %v", got)
	}
	g.PhaseOut(stone.ID())
	if got := g.ManaTypesAbilityCouldProduce(ability, stone.ID()); len(got) != 0 {
		t.Fatalf("phased source = %v", got)
	}
	g.PhaseIn([]uuid.UUID{stone.ID()})
	if got := g.ManaTypesAbilityCouldProduce(ability, stone.ID()); !slices.Equal(got, []Color{Black}) {
		t.Fatalf("phase-in control = %v", got)
	}
	if err := ApplyEffect(clone, ReturnToHand(This), land.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if got := clone.ManaTypesAbilityCouldProduce(ability, stone.ID()); len(got) != 0 {
		t.Fatalf("departed land LKI = %v", got)
	}
	returned := clone.PutOnBattlefield(clone.GetPlayer(b.PlayerID()).Hand()[0], b.PlayerID())
	if returned.incarnationID == land.incarnationID {
		t.Fatal("return kept the old visit")
	}
	if got := clone.ManaTypesAbilityCouldProduce(ability, stone.ID()); !slices.Equal(got, []Color{Blue}) {
		t.Fatalf("new visit options = %v", got)
	}
}

func TestManaColorsCouldProduceStalePlan(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	stone := g.PutOnBattlefield(NewArtifact("Options", "", WithAbility(NewManaAbilityOfColorsOpponentPermanentsCouldProduce(IsLand))), a.PlayerID())
	land := g.PutOnBattlefield(NewLand("Blue", WithManaAbility(Blue)), b.PlayerID())
	solution, err := g.mana.PlanManaForCost(g, a.PlayerID(), ManaCost{Blue: 1}, AutoTapHint{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	g.PhaseOut(land.ID())
	if err := g.mana.ApplyManaSolution(g, a.PlayerID(), solution); err == nil || g.FindPermanent(stone.ID()).Tapped || a.ManaPool().TotalMana() != 0 {
		t.Fatal("stale option plan spent resources")
	}
}

func TestManaColorsCouldProduceDynamicAndRiders(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ability := NewManaAbilityOfColorsOpponentPermanentsCouldProduce(IsLand)
	stone := g.PutOnBattlefield(NewArtifact("Options", "", WithAbility(ability)), a.PlayerID())
	land := g.PutOnBattlefield(NewLand("Dynamic", WithDynamicManaAbility(func(g GameReader, sourceID uuid.UUID) []ManaProduction {
		amount := g.FindPermanent(sourceID).Counters[Storage]
		if amount == 0 {
			return nil
		}
		return []ManaProduction{{Color: Green, Amount: amount, Restriction: ArtifactSpellsOnly{}}}
	}, GainLifeFor(You, Fixed(2)))), b.PlayerID())
	if got := g.ManaTypesAbilityCouldProduce(ability, stone.ID()); len(got) != 0 {
		t.Fatalf("zero dynamic production = %v", got)
	}
	land.Counters[Storage] = 3
	if got := g.ManaTypesAbilityCouldProduce(ability, stone.ID()); !slices.Equal(got, []Color{Green}) {
		t.Fatalf("dynamic production = %v", got)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), stone.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().Count(Green) != 1 || !a.ManaPool().CanPay(ManaCost{Green: 1}, nil) || b.Life() != 20 || b.ManaPool().TotalMana() != 0 || land.Counters[Storage] != 3 || land.Tapped {
		t.Fatal("query paid, copied restrictions, or ran a rider")
	}
	defaultAmount := g.PutOnBattlefield(NewLand("Default", WithMultiManaAbility(ManaProduction{Color: White})), b.PlayerID())
	if got := g.ManaTypesAbilityCouldProduce(defaultAmount.RuntimeAbilities[0], defaultAmount.ID()); !slices.Equal(got, []Color{White}) {
		t.Fatalf("default production amount = %v", got)
	}
}

func TestManaColorsCouldProduceAbsentAbility(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := g.PutOnBattlefield(NewLand("White", WithManaAbility(White)), b.PlayerID())
	for _, ability := range []Ability{nil, (*ManaAbility)(nil)} {
		if got := g.ManaTypesAbilityCouldProduce(ability, land.ID()); len(got) != 0 {
			t.Fatalf("absent ability = %v", got)
		}
	}
}

func TestManaColorsCouldProduceRecursiveReplacements(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	left := g.PutOnBattlefield(NewLand("Left", WithAbility(NewManaAbilityOfColorsOpponentPermanentsCouldProduce(IsLand))), a.PlayerID())
	right := g.PutOnBattlefield(NewLand("Right", WithAbility(NewManaAbilityOfColorsOpponentPermanentsCouldProduce(IsLand))), b.PlayerID())
	g.PutOnBattlefield(NewLand("Anchor", WithManaAbility(Red)), b.PlayerID())
	for _, rule := range []struct {
		source   *Permanent
		from, to Color
	}{{left, Red, Green}, {right, Green, Blue}, {left, Blue, White}} {
		if err := ApplyEffect(g, ReplaceTappedManaTypeUntilLeaves(This, rule.from, rule.to), rule.source.ID(), rule.source.ControllerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	got := g.ManaTypesAbilityCouldProduce(left.RuntimeAbilities[0], left.ID())
	if len(got) != 2 || !slices.Contains(got, Green) || !slices.Contains(got, White) {
		t.Fatalf("recursive replacement closure = %v", got)
	}
}
