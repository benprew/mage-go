package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTemporaryTriggerLifecycle(t *testing.T) {
	for _, change := range []string{"none", "return", "phase", "control", "remove", "expire", "two"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Recipient", "", 2, 4)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			prototype := WhenEvent(BecomesBlocked(This), RampageEffect(2)).(*GenericTriggered)
			grant := Until(core.EndOfTurn, TriggeredAbilityModification(This, prototype))
			if err := ApplyEffect(g, grant, p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			id := p.RuntimeAbilities[0].AbilityID()
			g.ApplyContinuousEffects()
			if len(p.RuntimeAbilities) != 1 || p.RuntimeAbilities[0].AbilityID() != id || !HasRampage.Match(p, g) {
				t.Fatal("unstable grant")
			}
			clone := g.Clone()
			want := 4
			switch change {
			case "return":
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(card.ID())
				p = g.PutOnBattlefield(card, a.PlayerID())
				want = 2
			case "phase":
				g.PhaseIn(g.PhaseOut(p.ID()))
				if g.FindPermanent(p.ID()).RuntimeAbilities[0].AbilityID() != id {
					t.Fatal("phase changed identity")
				}
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "remove":
				g.AddContinuousEffect(RemoveAllAbilities(p.ID(), core.EndOfTurn))
				want = 2
			case "expire":
				g.GetEffects().RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				want = 2
			case "two":
				if err := ApplyEffect(g, grant, p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				want = 6
			}
			if HasRampage.Match(g.FindPermanent(p.ID()), g) != (want > 2) {
				t.Fatal("rampage query ignored current abilities")
			}
			g.GetCombat().AddAttacker(p.ID(), b.PlayerID())
			for range 2 {
				g.PutOnBattlefieldBlocking(NewCreature("Blocker", "", 1, 1), b.PlayerID(), p.ID())
			}
			g.PutTriggersOnStack()
			if change == "none" {
				g.GetEffects().RemoveEndOfTurn()
				g.ApplyContinuousEffects()
			}
			g.ResolveStack()
			if g.FindPermanent(p.ID()).CurrentPower(g) != want {
				t.Fatalf("power=%d want %d", p.CurrentPower(g), want)
			}
			if !HasRampage.Match(clone.FindPermanent(card.ID()), clone) || clone.FindPermanent(card.ID()).CurrentPower(clone) != 2 {
				t.Fatal("clone shared grant state")
			}
			if prototype.source == p.ID() {
				t.Fatal("prototype was mutated")
			}
		})
	}
}

func TestTemporaryTriggerDeclarations(t *testing.T) {
	for _, tc := range []struct {
		name    string
		subject PermanentRef
		trigger *GenericTriggered
		bad     bool
	}{
		{"valid", This, WhenEvent(BecomesBlocked(This), RampageEffect(2)).(*GenericTriggered), false},
		{"nil subject", nil, At(YourUpkeep, GainLifeFor(You, Fixed(1))).(*GenericTriggered), true},
		{"nil trigger", This, nil, true},
		{"unbound subject", From(BindPermanent("missing")), At(YourUpkeep, GainLifeFor(You, Fixed(1))).(*GenericTriggered), true},
		{"unbound body", This, At(YourUpkeep, TapSubject(From(BindPermanent("missing")))).(*GenericTriggered), true},
		{"legacy event", This, NewTriggered(core.EvtUpkeep, false, GainLifeFor(You, Fixed(1))), true},
		{"zone", This, At(YourUpkeep, GainLifeFor(You, Fixed(1))).(*GenericTriggered).FromGraveyard(), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := NewArtifact("Grant", "", WithAbility(Activate(Costs(), Until(core.EndOfTurn, TriggeredAbilityModification(tc.subject, tc.trigger)))))
			if err := ValidateCardDefinition(c.Name(), c); (err != nil) != tc.bad {
				t.Fatalf("validation=%v", err)
			}
		})
	}
}

func TestTemporaryTriggerSpellScope(t *testing.T) {
	outer := DeclarePermanentTarget("recipient", IsCreature)
	for _, tc := range []struct {
		name    string
		subject PermanentRef
		body    Effect
		bad     bool
	}{
		{"valid spell grant", outer, GainLifeFor(You, Fixed(1)), false},
		{"spell source", This, GainLifeFor(You, Fixed(1)), true},
		{"outer target read", outer, TapSubject(outer), true},
		{"cast context", outer, IfCast(RevealedSubtypeAtCast("Dragon"), GainLifeFor(You, Fixed(1))), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			trigger := WhenEvent(BecomesTapped(This), tc.body).(*GenericTriggered)
			card := NewInstant("Grant spell", "", WithAbility(Spell(Until(core.EndOfTurn, TriggeredAbilityModification(tc.subject, trigger)), WithTargets(outer))))
			if err := ValidateCardDefinition(card.Name(), card); (err != nil) != tc.bad {
				t.Fatalf("validation=%v", err)
			}
		})
	}
}

func TestTemporaryTriggerRecipientController(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Recipient", "", 2, 4), b.PlayerID())
	target := DeclarePermanentTarget("recipient", IsCreature)
	prototype := WhenEvent(BecomesTapped(This), GainLifeFor(You, Fixed(2))).(*GenericTriggered)
	spell := NewInstant("Grant", "", WithAbility(Spell(Until(core.EndOfTurn, TriggeredAbilityModification(target, prototype)), WithTargets(target))))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.TapPermanent(g.FindPermanent(p.ID()))
	g.PutTriggersOnStack()
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	clone := g.Clone()
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 22 || clone.GetPlayer(b.PlayerID()).Life() != 20 {
		t.Fatal("pending grant lost recipient context or shared state")
	}
	clone.ResolveStack()
	if clone.GetPlayer(b.PlayerID()).Life() != 22 {
		t.Fatal("clone lost pending trigger")
	}
}

func TestTemporaryTriggerStaticLayerOrder(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	prototype := WhenEvent(BecomesTapped(This), GainLifeFor(You, Fixed(1))).(*GenericTriggered)
	condition := StateConditionFunc("always", []StateDependency{StateBattlefield}, func(StateReader, StateContext) bool { return true })
	card := NewCreature("Recipient", "", 2, 4, WithAbility(Static(While(condition, TriggeredAbilityModification(This, prototype)))))
	p := g.PutOnBattlefield(card, a.PlayerID())
	id := p.RuntimeAbilities[len(p.RuntimeAbilities)-1].AbilityID()
	g.ApplyContinuousEffects()
	if p.RuntimeAbilities[len(p.RuntimeAbilities)-1].AbilityID() != id {
		t.Fatal("live grant changed identity")
	}
	if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.TapPermanent(g.FindPermanent(p.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("later removal failed")
	}
	if err := ApplyEffect(g, Until(core.EndOfTurn, TriggeredAbilityModification(This, prototype)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.UntapPermanent(g.FindPermanent(p.ID()))
	g.TapPermanent(g.FindPermanent(p.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("later grant failed")
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatal("wrong recipient controller")
	}
}
