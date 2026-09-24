package mage

import (
	"strings"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestModalActivationRestrictions(t *testing.T) {
	for _, granted := range []bool{false, true} {
		for _, scenario := range []string{"allowed", "wrong turn", "wrong step", "condition", "unpaid"} {
			t.Run(map[bool]string{false: "direct", true: "grant"}[granted]+"/"+scenario, func(t *testing.T) {
				g, a, b := newModalGame()
				options := []ModalActivationOption{
					ModeOf("one", GainLifeFor(You, Fixed(1))),
					OnlyAsSorcery(), ActivationIf(SourceHasCounters(P1P1, 1)), ActivationLimit(2),
					ModeOf("three", GainLifeFor(You, Fixed(3))),
				}
				recipient := NewArtifact("Recipient", "")
				donor := NewEnchantment("Donor", "")
				if granted {
					donor = NewEnchantment("Donor", "", WithAbility(Static(GrantEachActivatedModes(MatchingPermanents(IsArtifact), Costs(ManaPayment("{1}")), options...))))
				} else {
					recipient = NewArtifact("Recipient", "", WithAbility(ActivateModes(Costs(ManaPayment("{1}")), options...)))
				}
				g.PutOnBattlefield(donor, a.PlayerID())
				g.PutOnBattlefield(recipient, b.PlayerID())
				g.MutablePermanent(donor.ID()).AddCounter(P1P1, 1)
				if scenario != "condition" {
					g.MutablePermanent(recipient.ID()).AddCounter(P1P1, 1)
				}
				g.turns.SetActivePlayerIndex(1)
				g.turns.SetStep(PrecombatMain)
				if scenario == "wrong turn" {
					g.turns.SetActivePlayerIndex(0)
				}
				if scenario == "wrong step" {
					g.turns.SetStep(Upkeep)
				}
				if scenario != "unpaid" {
					b.ManaPool().Add(Colorless, 3)
				}
				g.ApplyContinuousEffects()
				b.modes = []int{1}
				err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, nil)
				if scenario != "allowed" {
					expectedMana := 3
					if scenario == "unpaid" {
						expectedMana = 0
					}
					if err == nil || g.StackSize() != 0 || b.ManaPool().TotalMana() != expectedMana {
						t.Fatal("invalid activation changed payment or stack", err)
					}
					g.turns.SetActivePlayerIndex(1)
					g.turns.SetStep(PrecombatMain)
					g.MutablePermanent(recipient.ID()).AddCounter(P1P1, 1)
					if scenario == "unpaid" {
						b.ManaPool().Add(Colorless, 3)
					}
					b.modes = []int{1}
					if err = g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
				} else if err != nil {
					t.Fatal(err)
				}
				if g.Stack().Peek().ModeChoice != 1 {
					t.Fatal("options changed mode indices")
				}
				if g.CopyStackObjectDirect(g.Stack().Peek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				clone := g.Clone()
				clone.turns.SetStep(EndStep)
				clone.MutablePermanent(recipient.ID()).RemoveCounter(P1P1, 1)
				clone.RemoveFromBattlefield(clone.FindPermanent(donor.ID()))
				clone.ApplyContinuousEffects()
				clone.ResolveStack()
				if clone.GetPlayer(b.PlayerID()).Life() != 26 || b.Life() != 20 {
					t.Fatal("resolution rechecked restrictions or clone changed original")
				}
				g.ResolveStack()
				b.modes = []int{0}
				if err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, nil); err != nil {
					t.Fatal("failed activation or copy consumed limit", err)
				}
				g.ResolveStack()
				g.ApplyContinuousEffects()
				if err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, nil); err == nil || b.ManaPool().TotalMana() != 1 {
					t.Fatal("modes did not share activation limit", err)
				}
				if b.Life() != 27 {
					t.Fatal("wrong mode effects")
				}
				g.doCleanupActions()
				b.ManaPool().Add(Colorless, 1)
				b.modes = []int{0}
				if err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, nil); err != nil {
					t.Fatal("turn retained limit", err)
				}
			})
		}
	}
}

func TestModalActivationRestrictionValidation(t *testing.T) {
	for _, optionSet := range [][]ModalActivationOption{
		{nil}, {OnlyAsSorcery(), OnlyDuring(Upkeep)}, {ActivationLimit(0)},
		{ActivationLimit(1), ActivationLimit(2)}, {ActivationIf(nil)}, {ActivationIf(AllConditions())},
	} {
		options := append([]ModalActivationOption{ModeOf("one", GainLifeFor(You, Fixed(1))), ModeOf("two", GainLifeFor(You, Fixed(2)))}, optionSet...)
		for _, granted := range []bool{false, true} {
			err := validateFactory("Invalid modal restrictions", func() Card {
				if granted {
					return NewEnchantment("Invalid modal restrictions", "", WithAbility(Static(GrantEachActivatedModes(MatchingPermanents(IsArtifact), Costs(), options...))))
				}
				return NewArtifact("Invalid modal restrictions", "", WithAbility(ActivateModes(Costs(), options...)))
			})
			if err == nil || !strings.Contains(err.Error(), "Invalid modal restrictions") {
				t.Fatal("missing definition diagnostic", err)
			}
		}
	}
}

func TestModalActivationTimingOptions(t *testing.T) {
	for _, timing := range []struct {
		name   string
		option ModalActivationOption
		step   PhaseStep
	}{{"sorcery", OnlyAsSorcery(), PrecombatMain}, {"step", OnlyDuring(Upkeep), Upkeep}, {"own turn", OnlyOnYourTurn(), Upkeep}} {
		for _, scenario := range []string{"allowed", "wrong turn", "wrong step", "stack occupied"} {
			t.Run(timing.name+"/"+scenario, func(t *testing.T) {
				g, a, b := newModalGame()
				donor := NewEnchantment("Donor", "", WithAbility(Static(GrantEachActivatedModes(
					MatchingPermanents(IsArtifact), Costs(ManaPayment("{1}")), timing.option,
					ModeOf("one", GainLifeFor(You, Fixed(1))), ModeOf("two", GainLifeFor(You, Fixed(2))),
				))))
				recipient := NewArtifact("Recipient", "")
				g.PutOnBattlefield(donor, a.PlayerID())
				g.PutOnBattlefield(recipient, b.PlayerID())
				g.ApplyContinuousEffects()
				g.turns.SetActivePlayerIndex(1)
				g.turns.SetStep(timing.step)
				if scenario == "wrong turn" {
					g.turns.SetActivePlayerIndex(0)
				}
				if scenario == "wrong step" {
					g.turns.SetStep(EndStep)
				}
				if scenario == "stack occupied" {
					filler := NewEnchantment("Filler", "", WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)))))
					g.PutOnBattlefield(filler, b.PlayerID())
					if err := g.ActivateAbilityByIndex(b.PlayerID(), filler.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
				}
				before := g.StackSize()
				b.ManaPool().Add(Colorless, 1)
				b.modes = []int{0}
				allowed := (scenario != "wrong turn" || timing.name == "step") && (scenario != "wrong step" || timing.name == "own turn") && (scenario != "stack occupied" || timing.name != "sorcery")
				err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, nil)
				if (err == nil) != allowed {
					t.Fatalf("allowed = %v, error = %v", allowed, err)
				}
				if !allowed && (g.StackSize() != before || b.ManaPool().TotalMana() != 1 || len(b.modes) != 1) {
					t.Fatal("restriction checked after choice or payment")
				}
			})
		}
	}
}
