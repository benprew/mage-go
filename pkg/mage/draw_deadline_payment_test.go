package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDrawDeadlinePaymentLifetime(t *testing.T) {
	for _, change := range []string{"live", "leave", "return", "phase", "control"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
			effect := UnlessManaPaidBeforeNextDraw(You, core.ParseManaCost("{1}"), LoseLifeFor(You, Fixed(1)))
			for range 2 {
				if err := ApplyEffect(g, effect, source.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			switch change {
			case "leave", "return":
				g.RemoveFromBattlefield(source)
				if change == "return" {
					g.PutOnBattlefield(source.Card, b.PlayerID())
				}
			case "phase":
				g.PhaseOut(source.ID())
			case "control":
				g.AddControlEffect(ControlEffectSpec{TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			if len(g.GetPlayerPayments(b.PlayerID())) != 0 {
				t.Fatal("unaffordable offer")
			}
			b.ManaPool().Add(core.Colorless, 2)
			offers := g.GetPlayerPayments(b.PlayerID())
			if len(offers) != 2 || offers[0].ID == offers[1].ID {
				t.Fatal("obligations merged")
			}
			clone := g.Clone()
			if err := g.TakePlayerPayment(a.PlayerID(), offers[0].ID); err == nil {
				t.Fatal("wrong payer")
			}
			if err := g.TakePlayerPayment(b.PlayerID(), offers[0].ID); err != nil {
				t.Fatal(err)
			}
			if err := g.TakePlayerPayment(b.PlayerID(), offers[0].ID); err == nil {
				t.Fatal("payment repeated")
			}
			if b.Life() != 20 || b.ManaPool().TotalMana() != 1 || g.StackSize() != 0 || len(g.DelayedTriggers()) != 1 {
				t.Fatal("payment did not cancel exactly one delay")
			}
			g.doCleanupActions()
			b.ManaPool().Add(core.Colorless, 1)
			if len(g.GetPlayerPayments(b.PlayerID())) != 1 {
				t.Fatal("cleanup ended deadline")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtDrawStep, PlayerID: a.PlayerID()})
			if len(g.DelayedTriggers()) != 1 {
				t.Fatal("wrong player's draw consumed delay")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtDrawStep, PlayerID: b.PlayerID()})
			if err := g.TakePlayerPayment(b.PlayerID(), offers[1].ID); err == nil {
				t.Fatal("payment after deadline")
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || b.Life() != 20 {
				t.Fatal("deadline did not queue one ability")
			}
			pending := g.Clone()
			g.ResolveStack()
			pending.ResolveStack()
			if b.Life() != 19 || pending.GetPlayer(b.PlayerID()).Life() != 19 {
				t.Fatal("unpaid body lost")
			}
			if clone.GetPlayer(b.PlayerID()).Life() != 20 || len(clone.GetPlayerPayments(b.PlayerID())) != 2 {
				t.Fatal("clone shared cancellation")
			}
			clone.FireEvent(core.GameEvent{Type: core.EvtDrawStep, PlayerID: b.PlayerID()})
			clone.PutTriggersOnStack()
			clone.ResolveStack()
			if clone.GetPlayer(b.PlayerID()).Life() != 18 {
				t.Fatal("clone lost independent obligations")
			}
		})
	}
}

func TestDrawDeadlinePaymentRollbackAndValidation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	effect := UnlessManaPaidBeforeNextDraw(You, core.ParseManaCost("{2}"), LoseLifeFor(You, Fixed(1)))
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	id := g.playerPayments[0].id
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.TakePlayerPayment(a.PlayerID(), id); err == nil {
		t.Fatal("partial payment accepted")
	}
	if a.ManaPool().TotalMana() != 1 || len(g.DelayedTriggers()) != 1 {
		t.Fatal("failed payment changed state")
	}
	for _, e := range []ResolvingEffect{
		UnlessManaPaidBeforeNextDraw(nil, core.ManaCost{}, GainLifeFor(You, Fixed(1))),
		UnlessManaPaidBeforeNextDraw(You, core.ParseManaCost("{X}"), GainLifeFor(You, Fixed(1))),
		UnlessManaPaidBeforeNextDraw(You, core.ManaCost{Generic: -1}, GainLifeFor(You, Fixed(1))),
		UnlessManaPaidBeforeNextDraw(You, core.ManaCost{}, nil),
		UnlessManaPaidBeforeNextDraw(FromPlayer(BindPlayer("absent")), core.ManaCost{}, GainLifeFor(You, Fixed(1))),
	} {
		if err := ValidateCardDefinition("Invalid", NewInstant("Invalid", "", WithAbility(Spell(e)))); err == nil {
			t.Fatal("invalid definition accepted")
		}
	}
}

func TestDrawDeadlinePaymentSkippedAndLateStep(t *testing.T) {
	a := &acceptingMayPlayer{NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetTurn(3)
	g.SetStep(core.Upkeep)
	card := NewEnchantment("Skip draw", "", WithAbility(Static(SkipDrawStepForLife(2))))
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	if err := ApplyEffect(g, UnlessManaPaidBeforeNextDraw(You, core.ManaCost{}, LoseLifeFor(You, Fixed(1))), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.RunStepWithPriority(core.Draw)
	if len(g.GetPlayerPayments(a.PlayerID())) != 1 || len(g.DelayedTriggers()) != 1 || a.Life() != 20 {
		t.Fatal("skipped step consumed deadline")
	}
	g.DestroyPermanent(source)
	a.AddToLibrary(NewLand("Draw card"))
	g.RunStepWithPriority(core.Draw)
	if a.Life() != 21 || len(g.GetPlayerPayments(a.PlayerID())) != 0 {
		t.Fatal("next actual step did not consume deadline")
	}
	if err := ApplyEffect(g, UnlessManaPaidBeforeNextDraw(You, core.ManaCost{}, LoseLifeFor(You, Fixed(1))), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.DelayedTriggers()) != 1 || a.Life() != 21 {
		t.Fatal("installation used past draw event")
	}
	id := g.GetPlayerPayments(a.PlayerID())[0].ID
	g.resolution.resolvingController = a.PlayerID()
	if err := g.TakePlayerPayment(a.PlayerID(), id); err == nil {
		t.Fatal("payment during resolution")
	}
	g.resolution.resolvingController = uuid.Nil
	if err := g.TakePlayerPayment(a.PlayerID(), id); err != nil {
		t.Fatal(err)
	}
	if len(g.DelayedTriggers()) != 0 {
		t.Fatal("zero mana payment did not stop delay")
	}
}

func TestDrawDeadlinePaymentSavedFrame(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	n := BindNumber("saved amount")
	effect := Sequence(SnapshotNumber(Fixed(3), n), UnlessManaPaidBeforeNextDraw(You, core.ManaCost{}, LoseLifeFor(You, FromNumber(n))), SnapshotNumber(Fixed(7), n))
	if err := ValidateCardDefinition("Saved", NewInstant("Saved", "", WithAbility(Spell(effect)))); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.FireEvent(core.GameEvent{Type: core.EvtDrawStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if a.Life() != 14 {
		t.Fatal("delay or copy lost saved amount")
	}
}

func TestDrawDeadlinePaymentSourceLKI(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Source", "", 3, 3)
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	effect := UnlessManaPaidBeforeNextDraw(You, core.ManaCost{}, GainLifeFor(You, PowerOfWithLKI(This)))
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	waiting := g.WaitingDelayedPermanentSources()
	if len(waiting) != 1 || waiting[0].ID() != source.ID() {
		t.Fatal("waiting query lost explicit source reference")
	}
	if a.Life() != 20 || len(g.GetPlayerPayments(a.PlayerID())) != 1 {
		t.Fatal("waiting query executed body")
	}
	g.BouncePermanentToHand(source)
	a.RemoveFromHand(card.ID())
	returned := g.PutOnBattlefield(card, b.PlayerID())
	g.AddContinuousEffect(TemporaryBoost(returned.ID(), 6, 6))
	g.ApplyContinuousEffects()
	g.FireEvent(core.GameEvent{Type: core.EvtDrawStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 23 {
		t.Fatal("delay followed returned source")
	}
}
