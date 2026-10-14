package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestDelayedDamageCaptureFrames(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	wall := g.PutOnBattlefield(NewCreature("Watched", "", 0, 20), b.PlayerID())
	attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 3, 20), a.PlayerID())
	g.GetCombat().AddAttackerIncarnation(attacker.ID(), attacker.incarnationID, b.PlayerID())
	amount := BindNumber("damage")
	effect := WhenReceivesDamageThisTurn(This, And(IsCreature, DamageSourceIsAttacking()), amount, GainLifeFor(You, FromNumber(amount)))
	g.DealDamageToPermanent(wall, 1, attacker.ID())
	if err := ApplyEffect(g, effect, wall.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("captured earlier damage")
	}
	g.AddPreventionShield(wall.ID(), 1)
	g.DealDamageToPermanent(wall, 3, attacker.ID())
	g.DealDamageToPermanent(wall, 4, attacker.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 2 || a.Life() != 20 {
		t.Fatal("damage timing or repeat subscription")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.RemoveFromBattlefield(attacker)
	g.RemoveFromBattlefield(wall)
	clone := g.Clone()
	g.triggers.ClearEndOfTurn()
	g.ResolveStack()
	if a.Life() != 26 || b.Life() != 22 {
		t.Fatalf("life %d/%d", a.Life(), b.Life())
	}
	if clone.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("shared clone frame")
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 26 || clone.GetPlayer(b.PlayerID()).Life() != 22 {
		t.Fatal("lost clone frame")
	}
}

func TestDelayedDamageCaptureIdentity(t *testing.T) {
	for _, change := range []string{"none", "control", "phase", "phase-cycle", "return", "expire", "not-attacking", "noncreature", "prevent-all", "source-lki"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Watched", "", 0, 20)
			wall := g.PutOnBattlefield(card, b.PlayerID())
			attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 3, 20), a.PlayerID())
			g.GetCombat().AddAttackerIncarnation(attacker.ID(), attacker.incarnationID, b.PlayerID())
			amount := BindNumber("damage")
			effect := WhenReceivesDamageThisTurn(This, And(IsCreature, DamageSourceIsAttacking()), amount, GainLifeFor(You, FromNumber(amount)))
			if err := ApplyEffect(g, effect, wall.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			damageSource := g.CaptureDamageSource(attacker.ID())
			switch change {
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: wall.ID(), TargetID: wall.ID(), ControllerID: a.PlayerID(), Duration: Indefinite})
			case "phase":
				g.PhaseOut(wall.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(wall.ID()))
			case "return":
				g.RemoveFromBattlefield(wall)
				wall = g.PutOnBattlefield(card, b.PlayerID())
			case "expire":
				g.triggers.ClearEndOfTurn()
			case "not-attacking":
				g.RemoveFromCombat(attacker.ID())
			case "noncreature":
				attacker = g.PutOnBattlefield(NewArtifact("Artifact", ""), a.PlayerID())
				damageSource = g.CaptureDamageSource(attacker.ID())
			case "prevent-all":
				g.AddPreventionShield(wall.ID(), 9)
			case "source-lki":
				g.RemoveFromBattlefield(attacker)
				g.PutOnBattlefield(attacker.Card, b.PlayerID())
			}
			clone := g.Clone()
			for _, game := range []*Game{g, clone} {
				game.DealDamageToPermanentFromSource(game.FindPermanentIncludingPhased(wall.ID()), 3, damageSource)
				game.PutTriggersOnStack()
				game.ResolveStack()
				want := 20
				if change == "none" || change == "control" || change == "phase-cycle" || change == "source-lki" {
					want = 23
				}
				if game.GetPlayer(a.PlayerID()).Life() != want {
					t.Fatalf("life=%d want %d", game.GetPlayer(a.PlayerID()).Life(), want)
				}
			}
		})
	}
}

func TestDelayedDamageCaptureDefinition(t *testing.T) {
	amount := BindNumber("damage")
	body := GainLifeFor(You, FromNumber(amount))
	valid := WhenReceivesDamageThisTurn(This, IsCreature, amount, body)
	if err := ValidateCardDefinition("Valid", NewArtifact("Valid", "", WithAbility(Activate(Costs(), valid)))); err != nil {
		t.Fatal(err)
	}
	for _, effect := range []ResolvingEffect{
		WhenReceivesDamageThisTurn(nil, IsCreature, amount, body),
		WhenReceivesDamageThisTurn(This, IsCreature, nil, body),
		WhenReceivesDamageThisTurn(This, IsCreature, BindNumber(""), GainLifeFor(You, Fixed(1))),
		WhenReceivesDamageThisTurn(This, IsCreature, amount, nil),
		Sequence(valid, body),
		WhenReceivesDamageThisTurn(From(BindPermanent("unbound")), IsCreature, amount, body),
	} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))) == nil {
			t.Fatal("invalid definition accepted")
		}
	}
	g := NewGame(NewBasePlayer("A"), NewBasePlayer("B"))
	if err := ApplyEffect(g, valid, uuid.New(), g.ActivePlayerObj().PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.DelayedTriggers()) != 0 {
		t.Fatal("absent subject subscribed")
	}
}

func TestDelayedDamageCaptureSimultaneousAndRedirection(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	wall := g.PutOnBattlefield(NewCreature("Wall", "", 0, 20), b.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 0, 20), b.PlayerID())
	first := g.PutOnBattlefield(NewCreature("First", "", 2, 20), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 2, 20), a.PlayerID())
	g.GetCombat().AddAttackerIncarnation(first.ID(), first.incarnationID, b.PlayerID())
	g.GetCombat().AddAttackerIncarnation(second.ID(), second.incarnationID, b.PlayerID())
	amount := BindNumber("amount")
	if err := ApplyEffect(g, WhenReceivesDamageThisTurn(This, IsCreature, amount, GainLifeFor(You, FromNumber(amount))), wall.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.DealSimultaneousDamage(PermanentDamageAssignment(wall, 2, g.CaptureDamageSource(first.ID())), PermanentDamageAssignment(wall, 5, g.CaptureDamageSource(second.ID())), PermanentDamageAssignment(other, 7, g.CaptureDamageSource(first.ID())))
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("simultaneous damage frames merged")
	}
	clone := g.Clone()
	g.ResolveStack()
	clone.ResolveStack()
	if a.Life() != 27 || clone.GetPlayer(a.PlayerID()).Life() != 27 {
		t.Fatal("simultaneous amount lost")
	}
	source := BindPermanent("redirect source")
	g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{source: {first.ID(), first.incarnationID}}
	if err := RedirectDamageFromPermanentThisTurn(From(source), You, This).Apply(&EffectContext{Game: g, SourceID: wall.ID(), Controller: b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.AddPreventionShield(wall.ID(), 1)
	g.DealDamageToPlayer(b, 4, first.ID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 30 || b.Life() != 20 {
		t.Fatal("final redirected recipient or prevented amount lost")
	}
}

func TestDelayedDamageCaptureWaitingSources(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Watched", "", 0, 20), b.PlayerID())
	saved := g.CaptureDamageSource(p.ID())
	amount := BindNumber("amount")
	effect := WhenReceivesDamageThisTurn(This, IsCreature, amount, GainLifeFor(You, FromNumber(amount)))
	if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for _, change := range []string{"live", "phase", "depart", "return"} {
		switch change {
		case "phase":
			g.PhaseOut(p.ID())
		case "depart":
			g.PhaseIn([]uuid.UUID{p.ID()})
			g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
		case "return":
			g.PutOnBattlefield(p.Card, b.PlayerID())
		}
		refs := g.WaitingDelayedPermanentSources()
		if len(refs) != 1 || !refs[0].SameObject(saved) {
			t.Fatalf("%s: watched visit missing", change)
		}
	}
	clone := g.Clone()
	g.triggers.ClearEndOfTurn()
	if len(g.WaitingDelayedPermanentSources()) != 0 || len(clone.WaitingDelayedPermanentSources()) != 1 {
		t.Fatal("waiting expiry or clone state lost")
	}
	g = NewGame(a, b)
	p = g.PutOnBattlefield(NewCreature("Creator", "", 1, 20), a.PlayerID())
	if err := ApplyEffect(g, AtNext(EachEndStep, effect), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.WaitingDelayedPermanentSources()) != 1 {
		t.Fatal("nested installation lost source read")
	}
}
