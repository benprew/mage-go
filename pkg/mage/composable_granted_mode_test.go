package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestStaticGrantedModesChoicesAndCopies(t *testing.T) {
	g, a, b := newModalGame()
	target := DeclarePlayerTarget("opponent", TargetOpponentOnly())
	donor := NewEnchantment("Donor", "", WithAbility(Static(GrantEachActivatedModes(
		MatchingPermanents(IsArtifact), Costs(ManaPayment("{1}"), TapCost(This)),
		ModeOf("gain", GainLifeFor(You, Fixed(2))),
		ModeOf("lose", LoseLifeFor(target, Fixed(3)), WithTargets(target)),
	))))
	g.PutOnBattlefield(donor, a.PlayerID())
	first, second := NewArtifact("First", ""), NewArtifact("Second", "")
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	g.ApplyContinuousEffects()
	a.modes, b.modes = []int{0}, []int{1}
	a.ManaPool().Add(Colorless, 1)
	b.ManaPool().Add(Colorless, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if top := g.Stack().Peek(); top.ModeChoice != 0 || len(top.Targets) != 0 {
		t.Fatal("targetless mode acquired another mode's targets")
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), second.ID(), 0, []uuid.UUID{a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	original := g.Stack().Peek()
	if original.ModeChoice != 1 || original.SourceID != second.ID() || original.Targets[0] != a.PlayerID() {
		t.Fatal("recipient lost its mode or source")
	}
	if !g.FindPermanent(first.ID()).Tapped || !g.FindPermanent(second.ID()).Tapped || a.ManaPool().TotalMana() != 0 || b.ManaPool().TotalMana() != 0 || a.Life() != 20 {
		t.Fatal("incorrect payment or resolution timing")
	}
	a.targets = [][]uuid.UUID{{b.PlayerID()}}
	cp := g.CopyStackObjectDirect(original, a.PlayerID(), true)
	if cp == nil || cp.ModeChoice != 1 || cp.Targets[0] != b.PlayerID() || original.Targets[0] != a.PlayerID() {
		t.Fatal("copy changed mode or original target")
	}
	clone := g.Clone()
	clone.RemoveFromBattlefield(clone.FindPermanent(donor.ID()))
	clone.RemoveFromBattlefield(clone.FindPermanent(second.ID()))
	clone.ApplyContinuousEffects()
	if len(clone.FindPermanent(first.ID()).RuntimeAbilities) != 0 {
		t.Fatal("donor departure retained the grant")
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 19 || clone.GetPlayer(b.PlayerID()).Life() != 17 {
		t.Fatal("pending modes lost their controllers or targets after departure")
	}
	if a.Life() != 20 || b.Life() != 20 || g.StackSize() != 3 {
		t.Fatal("clone changed original game")
	}
	g.ResolveStack()
	if a.Life() != 19 || b.Life() != 17 {
		t.Fatal("recipient choices leaked across modes")
	}
}

func TestStaticGrantedModesPaymentAndLegality(t *testing.T) {
	for _, scenario := range []string{"no mana", "wrong controller", "target returns", "recipient returns", "paid"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := newModalGame()
			target := DeclarePermanentTarget("creature", IsCreature, TargetYouControl())
			donor := NewEnchantment("Donor", "", WithAbility(Static(GrantEachActivatedModes(
				MatchingPermanents(IsArtifact), Costs(ManaPayment("{1}"), TapCost(This)),
				ModeOf("return", Sequence(ReturnToHand(target), GainLifeFor(You, Fixed(2))), WithTargets(target)),
				ModeOf("counter", AddCountersTo(This, Charge, Fixed(1))),
			))))
			g.PutOnBattlefield(donor, a.PlayerID())
			recipient := NewArtifact("Recipient", "")
			g.PutOnBattlefield(recipient, b.PlayerID())
			card := NewCreature("Target", "", 2, 2)
			card.SetOwner(b.PlayerID())
			controller := b.PlayerID()
			if scenario == "wrong controller" {
				controller = a.PlayerID()
			}
			g.PutOnBattlefield(card, controller)
			if scenario == "wrong controller" {
				g.PutOnBattlefield(NewCreature("Legal alternative", "", 1, 1), b.PlayerID())
			}
			b.targets = [][]uuid.UUID{{card.ID()}}
			g.ApplyContinuousEffects()
			mana := 1
			if scenario == "no mana" {
				mana = 0
			}
			b.ManaPool().Add(Colorless, mana)
			targets := []uuid.UUID{card.ID()}
			if scenario == "recipient returns" {
				b.modes = []int{1}
				targets = nil
			}
			err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, targets)
			if scenario == "no mana" || scenario == "wrong controller" {
				if err == nil || g.StackSize() != 0 || g.FindPermanent(recipient.ID()).Tapped || b.ManaPool().TotalMana() != mana {
					t.Fatal("failed activation changed payment or stack")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "target returns" {
				g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
				g.PutOnBattlefield(card, b.PlayerID())
			}
			if scenario == "recipient returns" {
				g.RemoveFromBattlefield(g.FindPermanent(recipient.ID()))
				g.PutOnBattlefield(recipient, b.PlayerID())
			}
			g.ResolveStack()
			wantLife := 20
			if scenario == "paid" {
				wantLife = 22
			}
			if b.Life() != wantLife {
				t.Fatal("mode ignored target legality")
			}
			if scenario == "paid" && (len(b.Hand()) != 1 || b.Hand()[0].ID() != card.ID()) {
				t.Fatal("mode did not return its target")
			}
			if g.FindPermanent(recipient.ID()).Counters[Charge] != 0 {
				t.Fatal("source reference followed a returned card or unselected mode ran")
			}
		})
	}
}

func TestStaticGrantedModesValidation(t *testing.T) {
	target := DeclarePermanentTarget("creature", IsCreature)
	valid := ModeOf("return", ReturnToHand(target), WithTargets(target))
	for _, tc := range []struct {
		name       string
		mode       ModeDefinition
		diagnostic string
	}{
		{"valid", ModeOf("gain", GainLifeFor(You, Fixed(1))), ""},
		{"scope", ModeOf("undeclared", ReturnToHand(target)), "target binding requires a declaration"},
		{"nil", ModeOf("missing", nil), "nil"},
		{"label", ModeOf("", Regenerate(This)), "supply a mode label"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewEnchantment(tc.name, "", WithAbility(Static(GrantEachActivatedModes(MatchingPermanents(IsArtifact), Costs(), valid, tc.mode))))
			err := ValidateCardDefinition(card.Name(), card)
			if tc.diagnostic == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.diagnostic) || !strings.Contains(err.Error(), "modes[1]") {
				t.Fatalf("missing mode diagnostic: %v", err)
			}
		})
	}
	for _, modes := range [][]ModalActivationOption{nil, {valid}} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("accepted fewer than two modes")
				}
			}()
			GrantEachActivatedModes(MatchingPermanents(IsArtifact), Costs(), modes...)
		}()
	}
}

func TestStaticGrantedModesUnavailableTarget(t *testing.T) {
	g, a, _ := newModalGame()
	target := DeclarePermanentTarget("creature", IsCreature)
	donor := NewEnchantment("Donor", "", WithAbility(Static(GrantEachActivatedModes(
		MatchingPermanents(IsArtifact), Costs(ManaPayment("{1}")),
		ModeOf("return", ReturnToHand(target), WithTargets(target)),
		ModeOf("gain", GainLifeFor(You, Fixed(2))),
	))))
	g.PutOnBattlefield(donor, a.PlayerID())
	recipient := NewArtifact("Recipient", "")
	g.PutOnBattlefield(recipient, a.PlayerID())
	g.ApplyContinuousEffects()
	a.ManaPool().Add(Colorless, 1)
	a.modes = []int{0}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.Stack().Peek().ModeChoice != 1 {
		t.Fatal("menu did not map the only legal mode to its definition")
	}
	g.ResolveStack()
	if a.Life() != 22 || a.ManaPool().TotalMana() != 0 {
		t.Fatal("legal mode did not resolve after payment")
	}
}

type grantedModeEffectChooser struct {
	*modalChoosePlayer
	offered []Mode
}

func (p *grantedModeEffectChooser) ChooseModeWithEffects(modes []Mode, _ string, _ *Game) int {
	p.offered = modes
	return 0
}

func TestStaticGrantedModesMenuFailures(t *testing.T) {
	for _, scenario := range []string{"none available", "invalid choice", "optional target", "effect chooser"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, _ := newModalGame()
			target := DeclarePermanentTarget("required", IsCreature)
			second := ModeOf("gain", GainLifeFor(You, Fixed(2)))
			if scenario == "none available" {
				second = ModeOf("regenerate", Regenerate(target), WithTargets(target))
			}
			if scenario == "optional target" {
				optional := DeclarePermanentTargets("optional", 0, 1, IsCreature)
				member := BindPermanent("member")
				second = ModeOf("optional", Sequence(ForEach(optional, member, Regenerate(From(member))), GainLifeFor(You, Fixed(2))), WithTargets(optional))
			}
			donor := NewEnchantment("Donor", "", WithAbility(Static(GrantEachActivatedModes(
				MatchingPermanents(IsArtifact), Costs(ManaPayment("{1}"), TapCost(This)),
				ModeOf("return", ReturnToHand(target), WithTargets(target)), second,
			))))
			g.PutOnBattlefield(donor, a.PlayerID())
			recipient := NewArtifact("Recipient", "")
			g.PutOnBattlefield(recipient, a.PlayerID())
			g.ApplyContinuousEffects()
			a.ManaPool().Add(Colorless, 1)
			if scenario == "invalid choice" {
				a.modes = []int{-1}
			}
			var chooser *grantedModeEffectChooser
			if scenario == "effect chooser" {
				chooser = &grantedModeEffectChooser{modalChoosePlayer: a}
				g.players[0] = chooser
			}
			err := g.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, nil)
			if scenario == "none available" || scenario == "invalid choice" {
				if err == nil || a.ManaPool().TotalMana() != 1 || g.FindPermanent(recipient.ID()).Tapped || g.StackSize() != 0 {
					t.Fatal("invalid menu choice changed payment or stack")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if chooser != nil && (len(chooser.offered) != 1 || chooser.offered[0].Label != "gain") {
				t.Fatal("effect chooser received unavailable modes")
			}
			if g.Stack().Peek().ModeChoice != 1 {
				t.Fatal("filtered mode lost its definition index")
			}
			g.ResolveStack()
			if a.Life() != 22 {
				t.Fatal("available mode did not resolve")
			}
		})
	}
}
