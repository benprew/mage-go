package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestStaticGrantedTargetActivation(t *testing.T) {
	for _, scenario := range []string{"paid", "wrong controller", "no mana", "donor leaves", "recipient leaves", "target returns", "target phases", "target control changes"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			target := DeclarePermanentTarget("creature", IsCreature, TargetYouControl())
			donor := NewEnchantment("Donor", "", WithAbility(Static(GrantEachActivatedAbility(
				MatchingPermanents(IsArtifact), Costs(ManaPayment("{1}"), TapCost(This)),
				Sequence(ReturnToHand(target), GainLifeFor(You, Fixed(2))), WithTargets(target),
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
			g.ApplyContinuousEffects()
			mana := 1
			if scenario == "no mana" {
				mana = 0
			}
			b.ManaPool().Add(Colorless, mana)
			err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, []uuid.UUID{card.ID()})
			if scenario == "wrong controller" || scenario == "no mana" {
				if err == nil || b.ManaPool().TotalMana() != mana || g.FindPermanent(recipient.ID()).Tapped || g.StackSize() != 0 {
					t.Fatal("invalid grant activation changed payment or stack")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if g.StackSize() != 1 || !g.FindPermanent(recipient.ID()).Tapped || b.ManaPool().TotalMana() != 0 || len(b.Hand()) != 0 {
				t.Fatal("grant used incorrect payment or resolution timing")
			}
			clone := g.Clone()
			switch scenario {
			case "donor leaves":
				clone.RemoveFromBattlefield(clone.FindPermanent(donor.ID()))
			case "recipient leaves":
				clone.RemoveFromBattlefield(clone.FindPermanent(recipient.ID()))
			case "target returns":
				clone.RemoveFromBattlefield(clone.FindPermanent(card.ID()))
				clone.PutOnBattlefield(card, b.PlayerID())
			case "target phases":
				clone.PhaseOut(card.ID())
			case "target control changes":
				clone.AddControlEffect(ControlEffectSpec{SourceID: donor.ID(), TargetID: card.ID(), ControllerID: a.PlayerID(), Duration: Indefinite})
			}
			clone.ApplyContinuousEffects()
			clone.ResolveStack()
			want := 1
			if scenario == "target returns" || scenario == "target phases" || scenario == "target control changes" {
				want = 0
			}
			if len(clone.GetPlayer(b.PlayerID()).Hand()) != want || clone.GetPlayer(b.PlayerID()).Life() != 20+2*want || clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("grant lost target legality or recipient controller")
			}
			if g.StackSize() != 1 || len(b.Hand()) != 0 || b.Life() != 20 {
				t.Fatal("clone changed the original activation")
			}
		})
	}
}

func TestStaticGrantedTargetCopies(t *testing.T) {
	g, a, b := newSpellCopyGame()
	target := DeclarePlayerTarget("opponent", TargetOpponentOnly())
	donor := NewEnchantment("Donor", "", WithAbility(Static(GrantEachActivatedAbility(
		MatchingPermanents(IsArtifact), Costs(), LoseLifeFor(target, Fixed(3)), WithTargets(target),
	))))
	g.PutOnBattlefield(donor, a.PlayerID())
	first, second := NewArtifact("First", ""), NewArtifact("Second", "")
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	g.ApplyContinuousEffects()
	for _, entry := range []struct{ player, source, target uuid.UUID }{{a.PlayerID(), first.ID(), b.PlayerID()}, {b.PlayerID(), second.ID(), a.PlayerID()}} {
		if err := g.ActivateAbilityByIndex(entry.player, entry.source, 0, []uuid.UUID{entry.target}); err != nil {
			t.Fatal(err)
		}
	}
	original := g.GetStack().Peek()
	a.chooseQueue = [][]uuid.UUID{{b.PlayerID()}}
	cp := g.CopyStackObjectDirect(original, a.PlayerID(), true)
	if cp == nil || cp.Targets[0] != b.PlayerID() || original.Targets[0] != a.PlayerID() || cp.SourceID != second.ID() {
		t.Fatal("grant copy changed original target or recipient source")
	}
	g.RemoveFromBattlefield(g.FindPermanent(donor.ID()))
	g.ApplyContinuousEffects()
	g.ResolveStack()
	if a.Life() != 17 || b.Life() != 14 {
		t.Fatalf("grant targets leaked between recipients or copy: %d, %d", a.Life(), b.Life())
	}
}

func TestStaticGrantedTargetValidation(t *testing.T) {
	target := DeclarePermanentTarget("creature", IsCreature)
	valid := GrantEachActivatedAbility(MatchingPermanents(IsCreature), Costs(), Regenerate(target), WithTargets(target))
	card := NewEnchantment("Valid", "", WithAbility(Static(valid)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	undeclared := GrantEachActivatedAbility(MatchingPermanents(IsCreature), Costs(), Regenerate(target))
	card = NewEnchantment("Separate scopes", "", WithAbility(Static(valid, undeclared)))
	if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "target binding requires a declaration") {
		t.Fatalf("sibling grant target escaped its scope: %v", err)
	}
}

func TestStaticGrantedTargetPartialLegality(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	firstTarget := DeclarePermanentTarget("first", IsCreature)
	secondTarget := DeclarePermanentTarget("second", IsCreature)
	donor := NewEnchantment("Donor", "", WithAbility(Static(GrantEachActivatedAbility(
		MatchingPermanents(IsArtifact), Costs(),
		Sequence(ReturnToHand(firstTarget), ReturnToHand(secondTarget)), WithTargets(firstTarget, secondTarget),
	))))
	g.PutOnBattlefield(donor, a.PlayerID())
	recipient := NewArtifact("Recipient", "")
	g.PutOnBattlefield(recipient, a.PlayerID())
	first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
	first.SetOwner(b.PlayerID())
	second.SetOwner(b.PlayerID())
	g.PutOnBattlefield(first, b.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	g.ApplyContinuousEffects()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, []uuid.UUID{first.ID(), second.ID()}); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
	g.PutOnBattlefield(first, b.PlayerID())
	g.ResolveStack()
	if len(b.Hand()) != 1 || b.Hand()[0].ID() != second.ID() || g.FindPermanent(first.ID()) == nil {
		t.Fatal("partial legality lost target group positions or incarnation")
	}
}

func TestStaticGrantedTargetManaUsesStack(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	target := DeclarePlayerTarget("player")
	donor := NewEnchantment("Donor", "", WithAbility(Static(GrantEachActivatedAbility(
		MatchingPermanents(IsArtifact), Costs(), AddManaAmount(Blue, Fixed(1)), WithTargets(target),
	))))
	g.PutOnBattlefield(donor, a.PlayerID())
	recipient := NewArtifact("Recipient", "")
	g.PutOnBattlefield(recipient, b.PlayerID())
	g.ApplyContinuousEffects()
	if err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, []uuid.UUID{a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.StackSize() != 1 || b.ManaPool().TotalMana() != 0 {
		t.Fatal("targeted mana grant resolved as a mana ability")
	}
	g.ResolveStack()
	if b.ManaPool().Count(Blue) != 1 || a.ManaPool().TotalMana() != 0 {
		t.Fatal("grant mana used the donor or target controller")
	}
}
