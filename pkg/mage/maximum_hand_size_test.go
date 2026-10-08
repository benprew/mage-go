package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestMaximumHandSizeStaticLifecycle(t *testing.T) {
	for _, chosen := range []bool{false, true} {
		t.Run(map[bool]string{false: "controller", true: "chosen"}[chosen], func(t *testing.T) {
			g := newPriorityTestGame()
			a, b := g.players[0], g.players[1]
			var subject PlayerRef = You
			if chosen {
				subject = ChosenPlayerOf(This)
			}
			card := NewArtifact("Hand limit", "", WithAbility(ChooseOpponentAsEnters()), WithStaticAbility(SetMaximumHandSizeStatic(subject, 4)))
			card.SetOwner(a.PlayerID())
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			p := g.PutOnBattlefield(card, a.PlayerID())
			check := func(game *Game, x, y int) {
				t.Helper()
				if game.MaximumHandSize(a.PlayerID()) != x || game.MaximumHandSize(b.PlayerID()) != y {
					t.Fatalf("limits = %d/%d, want %d/%d", game.MaximumHandSize(a.PlayerID()), game.MaximumHandSize(b.PlayerID()), x, y)
				}
			}
			if chosen {
				check(g, 7, 4)
			} else {
				check(g, 4, 7)
			}
			clone := g.Clone()
			g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			check(g, 7, 4)
			if chosen {
				check(clone, 7, 4)
			} else {
				check(clone, 4, 7)
			}
			ids := g.PhaseOut(p.ID())
			check(g, 7, 7)
			g.PhaseIn(ids)
			check(g, 7, 4)
			g.AddContinuousEffect(RemoveAllAbilities(p.ID(), core.EndOfTurn))
			check(g, 7, 7)
			g.RunStepWithPriority(core.Cleanup)
			check(g, 7, 4)
			g.BouncePermanentToHand(g.FindPermanent(p.ID()))
			check(g, 7, 7)
			a.RemoveFromHand(card.ID())
			g.PutOnBattlefield(card, b.PlayerID())
			if chosen {
				check(g, 4, 7)
			} else {
				check(g, 7, 4)
			}
		})
	}
}

func TestMaximumHandSizeStaticOrderAndCleanup(t *testing.T) {
	for _, unlimitedLast := range []bool{false, true} {
		g := newPriorityTestGame()
		a, b := g.players[0], g.players[1]
		fixed := NewArtifact("Fixed", "", WithStaticAbility(SetMaximumHandSizeStatic(You, 4)))
		unlimited := NewArtifact("Unlimited", "", WithStaticAbility(NoMaximumHandSizeStatic(You)))
		var last *Permanent
		if unlimitedLast {
			g.PutOnBattlefield(fixed, a.PlayerID())
			last = g.PutOnBattlefield(unlimited, a.PlayerID())
		} else {
			g.PutOnBattlefield(unlimited, a.PlayerID())
			last = g.PutOnBattlefield(fixed, a.PlayerID())
		}
		want := 4
		if unlimitedLast {
			want = -1
		}
		if g.MaximumHandSize(a.PlayerID()) != want {
			t.Fatal("timestamp order")
		}
		for range 10 {
			a.AddToHand(NewArtifact("A card", ""))
			b.AddToHand(NewArtifact("B card", ""))
		}
		g.RunStepWithPriority(core.Cleanup)
		count := 4
		if unlimitedLast {
			count = 10
		}
		if len(a.Hand()) != count || len(b.Hand()) != 10 || g.StackSize() != 0 {
			t.Fatal("cleanup did not use the active player's limit")
		}
		g.PhaseOut(last.ID())
		want = -1
		if unlimitedLast {
			want = 4
		}
		if g.MaximumHandSize(a.PlayerID()) != want {
			t.Fatal("earlier rule did not resume")
		}
	}
}

func TestMaximumHandSizeStaticDeclarationsAndAbsentChoices(t *testing.T) {
	for _, effect := range []ContinuousEffect{SetMaximumHandSizeStatic(nil, 4), SetMaximumHandSizeStatic(You, -1), NoMaximumHandSizeStatic(nil), SetMaximumHandSizeStatic(ChosenPlayerOf(nil), 4), SetMaximumHandSizeStatic(FromPlayer(BindPlayer("unbound")), 4)} {
		if err := ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithStaticAbility(effect))); err == nil {
			t.Fatal("invalid hand limit accepted")
		}
	}
	g := newPriorityTestGame()
	a := g.players[0]
	card := NewArtifact("Unchosen", "", WithStaticAbility(SetMaximumHandSizeStatic(ChosenPlayerOf(This), 0)))
	p := g.PutOnBattlefield(card, a.PlayerID())
	if g.MaximumHandSize(a.PlayerID()) != 7 {
		t.Fatal("absent choice used controller")
	}
	g.MutablePermanent(p.ID()).ChosenPlayer = uuid.New()
	g.effects.Apply(g)
	if g.MaximumHandSize(a.PlayerID()) != 7 {
		t.Fatal("unknown choice used controller")
	}
}

func TestMaximumHandSizeStaticZeroAndCloneSources(t *testing.T) {
	g := newPriorityTestGame()
	a, b := g.players[0], g.players[1]
	p := g.PutOnBattlefield(NewArtifact("Zero", "", WithStaticAbility(SetMaximumHandSizeStatic(You, 0))), a.PlayerID())
	for range 3 {
		a.AddToHand(NewArtifact("Card", ""))
	}
	clone := g.Clone()
	for _, effect := range clone.effects.effects {
		inner := effect
		if printed, ok := inner.(*printedAbilityContinuousEffect); ok {
			inner = printed.ContinuousEffect
		}
		if rule, ok := inner.(*maximumHandSizeEffect); ok {
			rule.SetSourceID(uuid.New())
		}
	}
	if clone.MaximumHandSize(a.PlayerID()) != 7 || g.MaximumHandSize(a.PlayerID()) != 0 {
		t.Fatal("clones shared effect source state")
	}
	g.RunStepWithPriority(core.Cleanup)
	if len(a.Hand()) != 0 || len(a.Graveyard()) != 3 || g.MaximumHandSize(b.PlayerID()) != 7 {
		t.Fatal("zero limit cleanup")
	}
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	if g.MaximumHandSize(a.PlayerID()) != 7 {
		t.Fatal("source departure left a limit")
	}
}

func TestChosenPlayerOfLiveIncarnation(t *testing.T) {
	g := newPriorityTestGame()
	a, b := g.players[0], g.players[1]
	card := NewArtifact("Choice", "", WithAbility(ChooseOpponentAsEnters()))
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	frame := &EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID(), sourceIncarnation: p.incarnationID, sourceCaptured: true}
	ref := ChosenPlayerOf(This)
	if id, ok := ref.resolvePlayer(frame); !ok || id != b.PlayerID() {
		t.Fatal("live choice missing")
	}
	phased := g.PhaseOut(p.ID())
	if _, ok := ref.resolvePlayer(frame); ok {
		t.Fatal("phased choice used retained information")
	}
	g.PhaseIn(phased)
	if id, ok := ref.resolveRecipient(frame); !ok || id != b.PlayerID() {
		t.Fatal("phase cycle lost identity")
	}
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	a.RemoveFromHand(card.ID())
	g.PutOnBattlefield(card, b.PlayerID())
	if _, ok := ref.resolvePlayer(frame); ok {
		t.Fatal("returned object replaced the original choice")
	}
	if _, ok := ChosenPlayerOf(This).resolvePlayer(&EffectContext{Game: g, SourceID: uuid.New()}); ok {
		t.Fatal("absent source supplied a choice")
	}
}

type handSizeQueryProbeEffect struct {
	ContinuousEffect
	checks *int
}

func (e *handSizeQueryProbeEffect) IsActive(*Game) bool {
	*e.checks++
	return true
}

func TestMaximumHandSizeStaticPureQuery(t *testing.T) {
	g := newPriorityTestGame()
	a := g.players[0]
	g.PutOnBattlefield(NewArtifact("Limit", "", WithStaticAbility(SetMaximumHandSizeStatic(You, 4))), a.PlayerID())
	checks := 0
	g.AddContinuousEffect(&handSizeQueryProbeEffect{ContinuousEffect: FuncContinuousEffect(core.LayerPT, core.Indefinite, func(*Game, uuid.UUID) error { return nil }), checks: &checks})
	before := checks
	for range 3 {
		if g.MaximumHandSize(a.PlayerID()) != 4 {
			t.Fatal("live limit missing")
		}
	}
	if checks != before || len(g.effects.Rules.maxHandSize) != 0 {
		t.Fatal("query changed rules or evaluated unrelated effect lifetimes")
	}
}
