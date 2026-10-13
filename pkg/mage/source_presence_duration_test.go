package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestUntilSourcePresentLifetime(t *testing.T) {
	for _, change := range []string{"none", "untap", "control", "abilities", "cleanup", "source-phase", "source-cycle", "source-indirect-cycle", "source-return", "target-cycle", "target-phase-source-cycle", "target-return"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			var card Card = NewArtifact("Source", "")
			if change == "source-indirect-cycle" {
				card = NewEnchantment("Source", "", WithSubTypes("Aura"))
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			targetCard := NewCreature("Target", "", 2, 3)
			target := g.PutOnBattlefield(targetCard, b.PlayerID())
			g.MutablePermanent(source.ID()).AttachedTo = target.ID()
			effect := UntilSourcePresent(This, ChangeController(AttachedPermanent, You), ModifyEachPT(MatchingPermanents(IsCreature), Fixed(1), Fixed(2)), GrantEachKeyword(MatchingPermanents(IsCreature), Flying))
			if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			switch change {
			case "untap":
				g.TapPermanent(source)
				g.UntapPermanent(source)
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			case "abilities":
				g.effects.Add(RemoveAllAbilities(source.ID(), EndOfTurn))
			case "cleanup":
				g.effects.RemoveEndOfTurn()
			case "source-phase":
				g.PhaseOut(source.ID())
			case "source-cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "source-indirect-cycle":
				g.PhaseIn(g.PhaseOut(target.ID()))
			case "source-return":
				g.RemoveFromBattlefield(source)
				g.PutOnBattlefield(card, a.PlayerID())
			case "target-cycle":
				g.PhaseIn(g.PhaseOut(target.ID()))
			case "target-phase-source-cycle":
				ids := g.PhaseOut(target.ID())
				g.PhaseIn(g.PhaseOut(source.ID()))
				g.PhaseIn(ids)
			case "target-return":
				g.RemoveFromBattlefield(target)
				target = g.PutOnBattlefield(targetCard, b.PlayerID())
			}
			g.ApplyContinuousEffects()
			active := change == "none" || change == "untap" || change == "control" || change == "abilities" || change == "cleanup" || change == "target-cycle"
			power, toughness, controller := 2, 3, b.PlayerID()
			if active {
				power, toughness, controller = 3, 5, a.PlayerID()
			}
			live := g.FindPermanent(target.ID())
			if live.CurrentPower(g) != power || live.CurrentToughness(g) != toughness || live.HasKeyword(Flying) != active || live.ControllerID() != controller {
				t.Fatal("wrong captured duration", live.CurrentPower(g), live.ControllerID())
			}
			late := g.PutOnBattlefield(NewCreature("Late", "", 2, 3), b.PlayerID())
			if late.CurrentPower(g) != 2 || late.HasKeyword(Flying) {
				t.Fatal("captured membership changed")
			}
			if clone.FindPermanent(targetCard.ID()).CurrentPower(clone) != 3 || clone.FindPermanent(targetCard.ID()).ControllerID() != a.PlayerID() {
				t.Fatal("clone changed with original")
			}
			clone.PhaseIn(clone.PhaseOut(source.ID()))
			clone.ApplyContinuousEffects()
			if clone.FindPermanent(targetCard.ID()).CurrentPower(clone) != 2 || clone.FindPermanent(targetCard.ID()).ControllerID() != b.PlayerID() {
				t.Fatal("clone duration restarted")
			}
		})
	}
}

func TestUntilSourcePresentStartAndValidation(t *testing.T) {
	for _, change := range []string{"live", "absent", "returned", "phased", "cycle"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewArtifact("Source", "")
			source := g.PutOnBattlefield(card, a.PlayerID())
			original := source.incarnationID
			target := g.PutOnBattlefield(NewCreature("Target", "", 2, 3), b.PlayerID())
			switch change {
			case "absent", "returned":
				g.RemoveFromBattlefield(source)
				if change == "returned" {
					g.PutOnBattlefield(card, a.PlayerID())
				}
			case "phased":
				g.PhaseOut(source.ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			}
			effect := UntilSourcePresent(This, ModifyEachPT(MatchingPermanents(IsCreature), Fixed(2), Fixed(0)))
			if err := effect.Apply(&EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: original}); err != nil {
				t.Fatal(err)
			}
			if change == "phased" {
				g.PhaseIn([]uuid.UUID{source.ID()})
			}
			g.ApplyContinuousEffects()
			want := 2
			if change == "live" || change == "cycle" {
				want = 4
			}
			if g.FindPermanent(target.ID()).CurrentPower(g) != want {
				t.Fatal("wrong start condition")
			}
		})
	}
	for _, card := range []Card{
		NewInstant("No source", "", WithAbility(Spell(UntilSourcePresent(This, ModifyPT(This, Fixed(1), Fixed(1)))))),
		NewArtifact("Empty", "", WithAbility(Activate(Costs(), UntilSourcePresent(This)))),
		NewArtifact("Nil", "", WithAbility(Activate(Costs(), UntilSourcePresent(This, nil)))),
	} {
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("accepted invalid duration", card.Name())
		}
	}
	valid := NewArtifact("Valid", "", WithAbility(Activate(Costs(), UntilSourcePresent(This, ChangeController(This, You)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
}

func TestUntilSourcePresentControlOrder(t *testing.T) {
	for _, restricted := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		first := g.PutOnBattlefield(NewArtifact("First", ""), a.PlayerID())
		second := g.PutOnBattlefield(NewArtifact("Second", ""), b.PlayerID())
		target := g.PutOnBattlefield(NewArtifact("Target", ""), b.PlayerID())
		if restricted {
			g.MutablePermanent(target.ID()).GrantBaseAttr(AttrCantChangeControl)
		}
		for _, source := range []*Permanent{first, second} {
			g.MutablePermanent(source.ID()).AttachedTo = target.ID()
			if err := ApplyEffect(g, UntilSourcePresent(This, ChangeController(AttachedPermanent, You)), source.ID(), source.ControllerID(), nil); err != nil {
				t.Fatal(err)
			}
		}
		if g.FindPermanent(target.ID()).ControllerID() != b.PlayerID() {
			t.Fatal("later layer did not win")
		}
		g.PhaseOut(second.ID())
		want := a.PlayerID()
		if restricted {
			want = b.PlayerID()
		}
		if g.FindPermanent(target.ID()).ControllerID() != want {
			t.Fatal("expiration did not restore earlier layer or control restriction")
		}
		g.PhaseOut(first.ID())
		if g.FindPermanent(target.ID()).ControllerID() != b.PlayerID() {
			t.Fatal("expiration did not restore base control")
		}
	}
}
