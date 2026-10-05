package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestUntilNextUpkeepIdentity(t *testing.T) {
	for _, change := range []string{"none", "source-control", "source-left", "source-return", "source-phase", "recipient-control", "recipient-return", "recipient-phase"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			target := DeclarePermanentTarget("recipient", IsCreature)
			body := UntilNextUpkeep(You, GrantKeywordModification(target, Flying), ModifyPT(target, Fixed(1), Fixed(2)))
			sourceCard := NewArtifact("Source", "", WithAbility(Activate(Costs(), body, WithTargets(target))))
			source := g.PutOnBattlefield(sourceCard, a.PlayerID())
			card := NewCreature("Recipient", "", 2, 4)
			p := g.PutOnBattlefield(card, b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{p.ID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			switch change {
			case "source-control":
				g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			case "source-left", "source-return":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				if change == "source-return" {
					g.PutOnBattlefield(sourceCard, b.PlayerID())
				}
			case "source-phase":
				g.PhaseOut(source.ID())
			case "recipient-control":
				g.AddControlEffect(ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: Indefinite})
			case "recipient-return":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				g.PutOnBattlefield(card, b.PlayerID())
			case "recipient-phase":
				g.PhaseOut(p.ID())
			}
			clone := g.Clone()
			g.SetActivePlayerIndex(1)
			g.RunStepWithPriority(Upkeep)
			if change == "recipient-phase" {
				g.PhaseIn([]uuid.UUID{p.ID()})
			}
			g.ApplyContinuousEffects()
			p = g.FindPermanent(card.ID())
			want := change != "recipient-return"
			if p.HasKeyword(Flying) != want || (want && (p.CurrentPower(g) != 3 || p.CurrentToughness(g) != 6)) {
				t.Fatal("duration followed a source or recipient change")
			}
			g.SetActivePlayerIndex(0)
			g.RunStepWithPriority(Upkeep)
			if g.FindPermanent(card.ID()).HasKeyword(Flying) || g.FindPermanent(card.ID()).CurrentPower(g) != 2 {
				t.Fatal("effect survived the captured player's next upkeep")
			}
			if change == "recipient-phase" {
				clone.PhaseIn([]uuid.UUID{card.ID()})
			}
			clone.ApplyContinuousEffects()
			if clone.FindPermanent(card.ID()).HasKeyword(Flying) != want {
				t.Fatal("expiry changed the clone")
			}
			clone.RunStepWithPriority(Upkeep)
			if clone.FindPermanent(card.ID()).HasKeyword(Flying) {
				t.Fatal("clone lost the captured expiry player")
			}
		})
	}
}

func TestUntilNextUpkeepDeclarations(t *testing.T) {
	for _, body := range []ResolvingEffect{
		UntilNextUpkeep(nil, GrantKeywordModification(This, Flying)),
		UntilNextUpkeep(You),
		UntilNextUpkeep(You, nil),
		UntilNextUpkeep(DeclarePlayerTarget("missing-player"), GrantKeywordModification(This, Flying)),
		UntilNextUpkeep(You, ChangeController(This, You)),
		UntilNextUpkeep(You, GrantKeywordModification(DeclarePermanentTarget("missing", IsCreature), Flying)),
		Until(UntilYourNextTurn, GrantKeywordModification(This, Flying)),
	} {
		if err := ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), body)))); err == nil {
			t.Fatal("invalid duration declaration accepted")
		}
	}
	valid := UntilNextUpkeep(You, GrantKeywordModification(This, Flying))
	if err := ValidateCardDefinition("Valid", NewArtifact("Valid", "", WithAbility(Activate(Costs(), valid)))); err != nil {
		t.Fatal(err)
	}
}

func TestUntilNextUpkeepSkippedAndExtraStep(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Recipient", "", 2, 4), a.PlayerID())
	body := UntilNextUpkeep(You, GrantKeywordModification(This, Flying))
	if err := ApplyEffect(g, body, p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.turns.Schedule().Remaining = []PhaseStep{Upkeep, PrecombatMain}
	g.SkipNextOccurrenceOfStep(Upkeep)
	step, ok := g.turns.PopNextStep()
	if !ok || step != PrecombatMain {
		t.Fatal("upkeep was not skipped")
	}
	g.RunStepWithPriority(step)
	if !g.FindPermanent(p.ID()).HasKeyword(Flying) {
		t.Fatal("skipped upkeep ended the duration")
	}
	g.AppendExtraStep(Upkeep)
	step, ok = g.turns.PopNextStep()
	if !ok || step != Upkeep {
		t.Fatal("extra upkeep was not scheduled")
	}
	g.RunStepWithPriority(step)
	if g.FindPermanent(p.ID()).HasKeyword(Flying) {
		t.Fatal("extra upkeep did not end the duration")
	}
}

func TestUntilNextUpkeepTiming(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Source", "", 2, 4,
		WithAbility(At(YourUpkeep, GainLifeFor(You, Fixed(1)), WithInterveningIf(Controls(HasKeywordFilter(Flying))))))
	source := g.PutOnBattlefield(card, a.PlayerID())
	body := UntilNextUpkeep(You, GrantKeywordModification(This, Flying))
	if err := ApplyEffect(g, body, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.RunStepWithPriority(Untap)
	if !g.FindPermanent(source.ID()).HasKeyword(Flying) {
		t.Fatal("effect ended before upkeep")
	}
	g.RunStepWithPriority(Upkeep)
	if a.Life() != 20 || g.FindPermanent(source.ID()).HasKeyword(Flying) {
		t.Fatal("upkeep event saw the expired modification")
	}
	if err := ApplyEffect(g, body, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.RunStepWithPriority(Cleanup)
	g.SetActivePlayerIndex(1)
	g.RunStepWithPriority(Upkeep)
	if !g.FindPermanent(source.ID()).HasKeyword(Flying) {
		t.Fatal("cleanup or opponent upkeep ended the new duration")
	}
	g.PhaseOut(source.ID())
	g.SetActivePlayerIndex(0)
	g.RunStepWithPriority(Upkeep)
	g.PhaseIn([]uuid.UUID{source.ID()})
	g.ApplyContinuousEffects()
	if g.FindPermanent(source.ID()).HasKeyword(Flying) {
		t.Fatal("phasing postponed expiry")
	}
}

func TestUntilNextUpkeepIndependentPlayers(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("A", "", 2, 4), a.PlayerID())
	q := g.PutOnBattlefield(NewCreature("B", "", 2, 4), b.PlayerID())
	for _, source := range []*Permanent{p, q} {
		if err := ApplyEffect(g, UntilNextUpkeep(You, GrantKeywordModification(This, Flying)), source.ID(), source.ControllerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	g.SetActivePlayerIndex(1)
	g.RunStepWithPriority(Upkeep)
	if !g.FindPermanent(p.ID()).HasKeyword(Flying) || g.FindPermanent(q.ID()).HasKeyword(Flying) {
		t.Fatal("independent resolutions shared expiry players")
	}
	if err := ApplyEffect(g, UntilNextUpkeep(ActivePlayer, GrantKeywordModification(This, Flying)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.SetActivePlayerIndex(0)
	g.RunStepWithPriority(Upkeep)
	if !g.FindPermanent(p.ID()).HasKeyword(Flying) {
		t.Fatal("explicit player was replaced by the controller")
	}
	g.SetActivePlayerIndex(1)
	g.RunStepWithPriority(Upkeep)
	if g.FindPermanent(p.ID()).HasKeyword(Flying) {
		t.Fatal("explicit player was not saved at resolution")
	}
	if err := ApplyEffect(g, UntilNextUpkeep(You, GrantKeywordModification(This, Flying)), p.ID(), uuid.New(), nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(p.ID()).HasKeyword(Flying) {
		t.Fatal("absent player started a duration")
	}
}
