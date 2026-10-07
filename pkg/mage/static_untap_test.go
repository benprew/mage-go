package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestStaticUntapSkipWholeStep(t *testing.T) {
	for _, active := range []int{0, 1} {
		t.Run(string(rune('A'+active)), func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			g.SetTurn(3)
			g.SetActivePlayerIndex(active)
			g.SetStep(core.EndStep)
			card := NewEnchantment("Skip", "", WithAbility(Static(SkipUntapSteps())))
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			player := g.ActivePlayerObj()
			subject := g.PutOnBattlefield(NewCreature("Subject", "", 1, 4), player.PlayerID())
			g.TapPermanent(subject)
			g.SkipNextUntap(subject.ID())
			g.MutablePermanent(subject.ID()).AddCounter(core.Stun, 1)
			phased := g.PutOnBattlefield(NewCreature("Phased", "", 1, 4), player.PlayerID())
			g.PhaseOut(phased.ID())
			player.ManaPool().Add(core.Blue, 1)
			g.RunStepWithPriority(core.Untap)
			if g.GetStep() != core.EndStep || !g.FindPermanent(subject.ID()).Tapped || g.FindPermanentIncludingPhased(phased.ID()).PhasedOut != true || g.FindPermanent(subject.ID()).Counters[core.Stun] != 1 || !g.turns.HasSkipNextUntap(subject.ID()) || player.ManaPool().TotalMana() != 1 {
				t.Fatal("skipped step performed an action or consumed a pending restriction")
			}
			clone := g.Clone()
			g.DestroyPermanent(g.FindPermanent(source.ID()))
			g.RunStepWithPriority(core.Untap)
			if g.GetStep() != core.Untap || g.FindPermanent(phased.ID()) == nil || g.turns.HasSkipNextUntap(subject.ID()) || g.FindPermanent(subject.ID()).Counters[core.Stun] != 1 {
				t.Fatal("actual step did not consume the pending restriction")
			}
			g.RunStepWithPriority(core.Untap)
			if g.FindPermanent(subject.ID()).Counters[core.Stun] != 0 || !g.FindPermanent(subject.ID()).Tapped {
				t.Fatal("stun did not replace the next untap")
			}
			g.RunStepWithPriority(core.Untap)
			if g.FindPermanent(subject.ID()).Tapped {
				t.Fatal("untap did not resume")
			}
			clone.RunStepWithPriority(core.Untap)
			if clone.GetStep() != core.EndStep || !clone.FindPermanentIncludingPhased(phased.ID()).PhasedOut {
				t.Fatal("clone lost its skip")
			}
		})
	}
}

func TestConditionalUntapLimitDecision(t *testing.T) {
	for _, scenario := range []string{"untapped", "tapped", "phased-in", "removed-ability", "clone"} {
		t.Run(scenario, func(t *testing.T) {
			a := &untapChoicePlayer{BasePlayer: NewBasePlayer("A")}
			g := NewGame(a, NewBasePlayer("B"))
			source := g.PutOnBattlefield(NewArtifact("Limit", "", WithAbility(Static(LimitUntapsDuringStepIf(IsLand, 1, PermanentIsUntapped(This))))), a.PlayerID())
			first := g.PutOnBattlefield(NewLand("First"), a.PlayerID())
			second := g.PutOnBattlefield(NewLand("Second"), a.PlayerID())
			g.TapPermanent(first)
			g.TapPermanent(second)
			a.choices = []uuid.UUID{first.ID()}
			switch scenario {
			case "tapped":
				g.TapPermanent(source)
			case "phased-in":
				g.PhaseOut(source.ID())
			case "removed-ability":
				if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), source.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			clone := g.Clone()
			if scenario == "clone" {
				clone.TapPermanent(clone.FindPermanent(source.ID()))
				clone.RunStepWithPriority(core.Untap)
				if clone.FindPermanent(second.ID()).Tapped {
					t.Fatal("clone kept false condition")
				}
			}
			g.RunStepWithPriority(core.Untap)
			limited := scenario != "tapped" && scenario != "removed-ability"
			if g.FindPermanent(first.ID()).Tapped || g.FindPermanent(second.ID()).Tapped != limited {
				t.Fatal("condition was not read at the untap decision")
			}
			if scenario == "tapped" {
				third := g.PutOnBattlefield(NewLand("Third"), a.PlayerID())
				g.TapPermanent(first)
				g.TapPermanent(third)
				a.choices = []uuid.UUID{first.ID()}
				g.RunStepWithPriority(core.Untap)
				if !g.FindPermanent(third.ID()).Tapped {
					t.Fatal("condition did not change for the next decision")
				}
			}
		})
	}
}

func TestConditionalUntapLimitDeclarations(t *testing.T) {
	for _, rule := range []StaticRule{LimitUntapsDuringStepIf(IsLand, 1, nil), LimitUntapsDuringStepIf(IsLand, -1, PermanentIsUntapped(This)), LimitUntapsDuringStepIf(IsLand, 1, PermanentIsUntapped(nil))} {
		card := NewArtifact("Invalid", "", WithAbility(Static(rule)))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("accepted invalid conditional limit")
		}
	}
}

type untapReplacementPlayer struct {
	*BasePlayer
	choice int
	offers int
}

func (p *untapReplacementPlayer) ChooseMode(_ []string, _ string) int { p.offers++; return p.choice }

type removeUntapSkipReplacement struct {
	replacementBase
	target  uuid.UUID
	applied int
}

func (*removeUntapSkipReplacement) IsActive(GameReader) bool { return true }
func (*removeUntapSkipReplacement) Matches(a Action, _ GameReader) bool {
	_, ok := a.(*beginUntapStepAction)
	return ok
}
func (r *removeUntapSkipReplacement) Replace(a Action, g *Game) Action {
	r.applied++
	g.BouncePermanentToHand(g.FindPermanent(r.target))
	return a
}
func (r *removeUntapSkipReplacement) Clone() ReplacementEffect { clone := *r; return &clone }

func TestStaticUntapSkipReplacementRecheck(t *testing.T) {
	for _, choice := range []int{0, 1} {
		t.Run(string(rune('0'+choice)), func(t *testing.T) {
			a := &untapReplacementPlayer{BasePlayer: NewBasePlayer("A"), choice: choice}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			g.SetStep(core.EndStep)
			card := NewEnchantment("Skip", "", WithAbility(Static(SkipUntapSteps())))
			card.SetOwner(b.PlayerID())
			source := g.PutOnBattlefield(card, b.PlayerID())
			subject := g.PutOnBattlefield(NewLand("Subject"), a.PlayerID())
			g.TapPermanent(subject)
			removal := &removeUntapSkipReplacement{target: source.ID()}
			g.effects.AddReplacement(removal)
			g.RunStepWithPriority(core.Untap)
			if a.offers != 1 {
				t.Fatal("replacement choice did not belong to the affected player")
			}
			if choice == 0 {
				if removal.applied != 1 || g.FindPermanent(subject.ID()).Tapped || g.GetStep() != core.Untap {
					t.Fatal("departed source skip was not rechecked")
				}
			} else if removal.applied != 0 || !g.FindPermanent(subject.ID()).Tapped || g.GetStep() != core.EndStep {
				t.Fatal("replacement ran after the step was skipped")
			}
		})
	}
}

func TestStaticUntapSkipTurnControl(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetTurn(3)
	g.SetStep(core.EndStep)
	g.PutOnBattlefield(NewEnchantment("Skip", "", WithAbility(Static(SkipUntapSteps()))), b.PlayerID())
	old := g.PutOnBattlefield(NewCreature("Old", "", 1, 4), a.PlayerID())
	facedown := g.PutOnBattlefield(NewCreature("Face down", "", 1, 4), a.PlayerID())
	g.MutablePermanent(facedown.ID()).FaceDown = true
	phased := g.PutOnBattlefield(NewCreature("Phased", "", 1, 4), a.PlayerID())
	g.PhaseOut(phased.ID())
	g.SetTurn(4)
	fresh := g.PutOnBattlefield(NewCreature("Fresh", "", 1, 4), a.PlayerID())
	g.RunStepWithPriority(core.Untap)
	if !g.FindPermanent(old.ID()).CanTapForEffect(g) || !g.FindPermanent(old.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("skipped untap kept summoning sickness from a previous turn")
	}
	if g.FindPermanent(fresh.ID()).CanTapForEffect(g) || g.FindPermanent(fresh.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("new entry lost summoning sickness")
	}
	g.MutablePermanent(facedown.ID()).FaceDown = false
	if !g.FindPermanent(facedown.ID()).CanTapForEffect(g) {
		t.Fatal("face-down state hid continuous control")
	}
	g.PhaseIn([]uuid.UUID{phased.ID()})
	if !g.FindPermanent(phased.ID()).CanTapForEffect(g) {
		t.Fatal("phase cycle lost continuous control")
	}
	clone := g.Clone()
	g.SetActivePlayerIndex(1)
	g.SetTurn(5)
	g.RunStepWithPriority(core.Untap)
	if g.FindPermanent(fresh.ID()).CanTapForEffect(g) {
		t.Fatal("opponent turn cleared sickness")
	}
	clone.SetTurn(5)
	clone.RunStepWithPriority(core.Untap)
	if !clone.FindPermanent(fresh.ID()).CanTapForEffect(clone) || g.FindPermanent(fresh.ID()).CanTapForEffect(g) {
		t.Fatal("extra turn or clone control incorrect")
	}
}
