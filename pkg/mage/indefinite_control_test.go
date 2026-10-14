package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestIndefiniteControl(t *testing.T) {
	for _, change := range []string{"cleanup", "source-left", "source-return", "source-control", "abilities", "phase-cycle", "target-return", "new-leader", "override"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			target := DeclarePermanentTarget("subject", IsCreature)
			effect := Until(core.Indefinite, ChangeController(target, PlayerWithUniqueHighestLife()))
			card := NewArtifact("Source", "", WithAbility(Activate(Costs(), effect, WithTargets(target))))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			victimCard := NewCreature("Victim", "", 2, 3)
			victim := g.PutOnBattlefield(victimCard, a.PlayerID())
			b.GainLife(1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if g.FindPermanent(victim.ID()).ControllerID() != b.PlayerID() {
				t.Fatal("control did not begin")
			}
			clone := g.Clone()
			switch change {
			case "cleanup":
				g.GetEffects().RemoveEndOfTurn()
			case "source-left", "source-return":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				if change == "source-return" {
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "source-control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "abilities":
				g.AddContinuousEffect(RemoveAllAbilities(source.ID(), core.Indefinite))
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(victim.ID()))
			case "target-return":
				g.RemoveFromBattlefield(g.FindPermanent(victim.ID()))
				g.PutOnBattlefield(victimCard, a.PlayerID())
			case "new-leader":
				a.GainLife(5)
			case "override":
				if err := ApplyEffect(g, Until(core.EndOfTurn, ChangeController(This, You)), victim.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				if g.FindPermanent(victim.ID()).ControllerID() != a.PlayerID() {
					t.Fatal("new timestamp did not win")
				}
				g.GetEffects().RemoveEndOfTurn()
			}
			g.ApplyContinuousEffects()
			want := b.PlayerID()
			if change == "target-return" {
				want = a.PlayerID()
			}
			if g.FindPermanent(victim.ID()).ControllerID() != want {
				t.Fatal("lifetime or exact visit")
			}
			clone.ApplyContinuousEffects()
			if clone.FindPermanent(victim.ID()).ControllerID() != b.PlayerID() {
				t.Fatal("clone changed")
			}
		})
	}
}

func TestIndefiniteControlDeclaration(t *testing.T) {
	for _, controller := range []PlayerRef{You, PlayerWithUniqueHighestLife()} {
		card := NewArtifact("Valid", "", WithAbility(Activate(Costs(), Until(core.Indefinite, ChangeController(This, controller)))))
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	for _, modification := range []ContinuousModification{ChangeController(nil, You), ChangeController(This, nil), ChangeController(This, FromPlayer(BindPlayer("unbound")))} {
		card := NewArtifact("Bad", "", WithAbility(Activate(Costs(), Until(core.Indefinite, modification))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid modification accepted")
		}
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	if err := ApplyEffect(g, Until(core.Indefinite, ChangeController(This, PlayerWithUniqueHighestLife())), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	b.GainLife(2)
	g.ApplyContinuousEffects()
	if g.FindPermanent(source.ID()).ControllerID() != a.PlayerID() {
		t.Fatal("tie installed a latent control effect")
	}
}

func TestIndefiniteControlManaFollowupValidation(t *testing.T) {
	binding := BindPlayer("unbound recipient")
	card := NewLand("Bad", WithAbility(NewManaAbility(core.AnyColor, Until(core.Indefinite, ChangeController(This, FromPlayer(binding))))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("unbound post-production controller accepted")
	}
}
