package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestUntilNextTurnLifetime(t *testing.T) {
	for _, change := range []string{"none", "source-control", "source-left", "source-return", "source-phase", "subject-phase", "subject-return", "source-abilities"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			g.RunStepWithPriority(Untap)
			card := NewCreature("Subject", "", 2, 4)
			p := g.PutOnBattlefield(card, b.PlayerID())
			target := DeclarePermanentTarget("subject", IsCreature)
			rule := Activate(Costs(), UntilNextTurn(You, GrantKeywordModification(target, Flying), ModifyPT(target, Fixed(1), Fixed(2))), WithTargets(target))
			sourceCard := NewArtifact("Source", "", WithAbility(rule))
			source := g.PutOnBattlefield(sourceCard, a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{p.ID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			switch change {
			case "source-control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			case "source-left", "source-return":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				if change == "source-return" {
					g.PutOnBattlefield(sourceCard, b.PlayerID())
				}
			case "source-phase":
				g.PhaseOut(source.ID())
			case "subject-phase":
				g.PhaseOut(p.ID())
			case "subject-return":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				p = g.PutOnBattlefield(card, b.PlayerID())
			case "source-abilities":
				if err := ApplyEffect(g, Until(EndOfTurn, RemoveAbilities(This)), source.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			clone := g.Clone()
			g.RunStepWithPriority(Upkeep)
			g.RunStepWithPriority(Cleanup)
			g.AdvanceToNextTurn()
			g.RunStepWithPriority(Upkeep)
			if change == "subject-phase" {
				g.PhaseIn([]uuid.UUID{p.ID()})
			}
			g.ApplyContinuousEffects()
			want := change != "subject-return"
			if g.FindPermanent(p.ID()).HasKeyword(Flying) != want {
				t.Fatal("duration ended in current or other player's turn")
			}
			g.AdvanceToNextTurn()
			g.SkipNextOccurrenceOfStep(Untap)
			g.RunStepWithPriority(PrecombatMain)
			if g.FindPermanent(p.ID()).HasKeyword(Flying) || g.FindPermanent(p.ID()).CurrentPower(g) != 2 {
				t.Fatal("next actual turn did not expire all layers")
			}
			if change == "subject-phase" {
				clone.PhaseIn([]uuid.UUID{p.ID()})
			}
			clone.ApplyContinuousEffects()
			if clone.FindPermanent(p.ID()).HasKeyword(Flying) != want {
				t.Fatal("expiry changed clone")
			}
			clone.GrantExtraTurn(a.PlayerID())
			clone.AdvanceToNextTurn()
			clone.RunStepWithPriority(Untap)
			if clone.FindPermanent(p.ID()).HasKeyword(Flying) {
				t.Fatal("own extra turn did not expire clone")
			}
		})
	}
}

func TestUntilNextTurnSkippedTurnAndUpkeep(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.RunStepWithPriority(Untap)
	p := g.PutOnBattlefield(NewCreature("Subject", "", 2, 4), a.PlayerID())
	if err := ApplyEffect(g, UntilNextTurn(You, GrantKeywordModification(This, Flying)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.RunStepWithPriority(Upkeep)
	g.AppendExtraStep(Upkeep)
	g.RunStepWithPriority(Upkeep)
	if !g.FindPermanent(p.ID()).HasKeyword(Flying) {
		t.Fatal("extra upkeep ended turn lifetime")
	}
	g.SkipNextTurnFor(a.PlayerID())
	g.AdvanceToNextTurn()
	g.RunStepWithPriority(PrecombatMain)
	g.AdvanceToNextTurn()
	if !g.turns.ConsumeTurnSkip(a.PlayerID()) {
		t.Fatal("missing turn skip")
	}
	g.AdvanceToNextTurn()
	g.RunStepWithPriority(PrecombatMain)
	if !g.FindPermanent(p.ID()).HasKeyword(Flying) {
		t.Fatal("skipped turn expired duration")
	}
	g.GrantExtraTurn(a.PlayerID())
	g.AdvanceToNextTurn()
	g.RunStepWithPriority(Untap)
	if g.FindPermanent(p.ID()).HasKeyword(Flying) {
		t.Fatal("actual extra turn did not expire duration")
	}
}

func TestUntilNextTurnDeclarations(t *testing.T) {
	for _, body := range []ResolvingEffect{
		UntilNextTurn(nil, GrantKeywordModification(This, Flying)), UntilNextTurn(You), UntilNextTurn(You, nil),
		UntilNextTurn(DeclarePlayerTarget("missing"), GrantKeywordModification(This, Flying)),
		UntilNextTurn(You, ChangeController(This, You)),
	} {
		if err := ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), body)))); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Subject", "", 2, 4), a.PlayerID())
	if err := ApplyEffect(g, UntilNextTurn(You, GrantKeywordModification(This, Flying)), p.ID(), uuid.New(), nil); err != nil {
		t.Fatal(err)
	}
	if p.HasKeyword(Flying) {
		t.Fatal("absent player installed effect")
	}
}

func TestUntilNextTurnNativeSkippedTurn(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for range 8 {
		a.AddToLibrary(NewLand("A"))
		b.AddToLibrary(NewLand("B"))
	}
	g.RunStepWithPriority(Untap)
	p := g.PutOnBattlefield(NewCreature("Subject", "", 2, 4), a.PlayerID())
	if err := ApplyEffect(g, UntilNextTurn(You, GrantKeywordModification(This, Flying)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.SkipNextTurnFor(a.PlayerID())
	g.AdvanceToNextTurn()
	g.Run(0, Untap, 4)
	if !g.FindPermanent(p.ID()).HasKeyword(Flying) {
		t.Fatal("native skipped turn ended duration")
	}
	g.SkipNextOccurrenceOfStep(Untap)
	g.Run(0, Untap, 5)
	if g.FindPermanent(p.ID()).HasKeyword(Flying) {
		t.Fatal("native turn with skipped untap kept duration")
	}
}

func TestUntilNextTurnIndependentPlayersAndTiming(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.RunStepWithPriority(Untap)
	p := g.PutOnBattlefield(NewCreature("A", "", 2, 4), a.PlayerID())
	q := g.PutOnBattlefield(NewCreature("B", "", 2, 4), b.PlayerID())
	for _, subject := range []*Permanent{p, q} {
		if err := ApplyEffect(g, UntilNextTurn(You, GrantKeywordModification(This, Flying)), subject.ID(), subject.ControllerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if err := ApplyEffect(g, UntilNextUpkeep(You, GrantKeywordModification(This, Trample)), q.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.AdvanceToNextTurn()
	g.RunStepWithPriority(Untap)
	if !g.FindPermanent(p.ID()).HasKeyword(Flying) || g.FindPermanent(q.ID()).HasKeyword(Flying) || !g.FindPermanent(q.ID()).HasKeyword(Trample) {
		t.Fatal("turn and upkeep endpoints merged")
	}
	g.RunStepWithPriority(Upkeep)
	if g.FindPermanent(q.ID()).HasKeyword(Trample) {
		t.Fatal("old upkeep lifetime changed")
	}
	g.AdvanceToNextTurn()
	g.RunStepWithPriority(Untap)
	if g.FindPermanent(p.ID()).HasKeyword(Flying) {
		t.Fatal("independent saved player lost")
	}
}

type nextTurnDiscardReplacement struct {
	allDiscardToLibraryReplacement
	player uuid.UUID
	after  int
}

func (r *nextTurnDiscardReplacement) NextTurnExpiry() (player uuid.UUID, afterTurn int) {
	return r.player, r.after
}
func (r *nextTurnDiscardReplacement) Clone() ReplacementEffect { cloned := *r; return &cloned }

func TestUntilNextTurnReplacementLifetime(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.RunStepWithPriority(Untap)
	g.effects.AddReplacement(&nextTurnDiscardReplacement{player: a.PlayerID(), after: g.CurrentTurn()})
	clone := g.Clone()
	discard := func(game *Game, library bool) {
		t.Helper()
		player := game.GetPlayer(a.PlayerID())
		card := NewArtifact("Discard", "")
		player.AddToHand(card)
		game.PlayerDiscardByEffect(player, card.ID(), uuid.Nil)
		inLibrary := false
		for _, c := range player.Library() {
			inLibrary = inLibrary || c.ID() == card.ID()
		}
		if inLibrary != library {
			t.Fatal("replacement lifetime has wrong endpoint")
		}
	}
	g.RunStepWithPriority(Upkeep)
	discard(g, true)
	g.AdvanceToNextTurn()
	g.RunStepWithPriority(PrecombatMain)
	discard(g, true)
	g.GrantExtraTurn(a.PlayerID())
	g.AdvanceToNextTurn()
	g.RunStepWithPriority(Untap)
	discard(g, false)
	discard(clone, true)
	clone.GrantExtraTurn(a.PlayerID())
	clone.AdvanceToNextTurn()
	clone.RunStepWithPriority(PrecombatMain)
	discard(clone, false)
}

func TestUntilNextTurnExpiresWhileSubjectPhased(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.RunStepWithPriority(Untap)
	p := g.PutOnBattlefield(NewCreature("Subject", "", 2, 4), b.PlayerID())
	if err := ApplyEffect(g, UntilNextTurn(You, GrantKeywordModification(This, Flying)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PhaseOut(p.ID())
	g.GrantExtraTurn(a.PlayerID())
	g.AdvanceToNextTurn()
	g.RunStepWithPriority(PrecombatMain)
	g.PhaseIn([]uuid.UUID{p.ID()})
	g.ApplyContinuousEffects()
	if g.FindPermanent(p.ID()).HasKeyword(Flying) {
		t.Fatal("phasing postponed expiration")
	}
}
