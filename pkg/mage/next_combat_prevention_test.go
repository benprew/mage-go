package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestNextCombatPreventionIdentity(t *testing.T) {
	for _, change := range []string{"live", "depart", "return", "phase", "phase-cycle", "control", "cleanup", "type"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Source", "", 4, 10)
			p := g.PutOnBattlefield(card, a.PlayerID())
			old := g.CaptureDamageSource(p.ID())
			if err := ApplyEffect(g, PreventNextCombatDamageFromPermanentExcept(This, You, Fixed(1)), p.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "depart", "return":
				g.RemoveFromBattlefield(p)
				if change == "return" {
					g.PutOnBattlefield(card, a.PlayerID())
				}
				g.zones.ClearLKI()
			case "phase":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "cleanup":
				g.effects.ClearReplacementsEndOfTurn()
			case "type":
				p.RevokeBaseAttr(core.AttrIsCreature)
			}
			g.DealDamageToPlayerFromSource(b, 2, old)
			if b.Life() != 18 {
				t.Fatal("noncombat damage was prevented")
			}
			g.damage.SetResolvingCombatDamage(true)
			g.DealDamageToPlayerFromSource(b, 0, old)
			g.DealDamageToPlayerFromSource(a, 2, old)
			if change == "return" {
				g.DealDamageToPlayer(b, 2, p.ID())
				if b.Life() != 16 {
					t.Fatal("returned source inherited shield")
				}
			}
			clone := g.Clone()
			for _, game := range []*Game{g, clone} {
				recipient := game.GetPlayer(b.PlayerID())
				before := recipient.Life()
				game.DealDamageToPlayerFromSource(recipient, 4, old)
				loss := 1
				if change == "cleanup" || change == "type" {
					loss = 4
				}
				if recipient.Life() != before-loss {
					t.Fatalf("life=%d want=%d", recipient.Life(), before-loss)
				}
				if change == "type" {
					game.MutablePermanent(p.ID()).GrantBaseAttr(core.AttrIsCreature)
				}
				before = recipient.Life()
				game.DealDamageToPlayerFromSource(recipient, 3, old)
				loss = 3
				if change == "type" {
					loss = 1
				}
				if recipient.Life() != before-loss {
					t.Fatal("wrong next-event consumption")
				}
			}
		})
	}
}

func TestNextCombatPreventionBatchAndAmount(t *testing.T) {
	for _, keep := range []int{0, 1, 9} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		p := g.PutOnBattlefield(NewCreature("Source", "", 4, 10), a.PlayerID())
		q := g.PutOnBattlefield(NewCreature("Other", "", 2, 10), a.PlayerID())
		if err := ApplyEffect(g, PreventNextCombatDamageFromPermanentExcept(This, You, Fixed(keep)), p.ID(), b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		source := g.CaptureDamageSource(p.ID())
		g.damage.SetResolvingCombatDamage(true)
		clone := g.Clone()
		g.DealSimultaneousDamage(PlayerDamageAssignment(b, 2, source), PlayerDamageAssignment(b, 3, source), PlayerDamageAssignment(b, 2, g.CaptureDamageSource(q.ID())), PlayerDamageAssignment(a, 2, source))
		if b.Life() != 18-min(5, keep) || a.Life() != 18 {
			t.Fatalf("keep=%d life=%d/%d", keep, a.Life(), b.Life())
		}
		if (len(g.WaitingPreventionPermanentSources()) == 0) != (keep < 5) {
			t.Fatal("consumed rule retained waiting reference")
		}
		g.DealDamageToPlayer(b, 3, p.ID())
		if b.Life() != 15-min(5, keep) {
			t.Fatal("batch did not consume one event")
		}
		clone.DealDamageToPlayer(clone.GetPlayer(b.PlayerID()), 4, p.ID())
		if clone.GetPlayer(b.PlayerID()).Life() != 20-min(4, keep) {
			t.Fatal("clone shared consumption")
		}
	}
}

func TestNextCombatPreventionOrder(t *testing.T) {
	for choice := range 2 {
		a := NewBasePlayer("A")
		b := &sourceFilterChoicePlayer{BasePlayer: NewBasePlayer("B"), choice: choice}
		g := NewGame(a, b)
		p := g.PutOnBattlefield(NewCreature("Source", "", 4, 10), a.PlayerID())
		if err := ApplyEffect(g, PreventNextCombatDamageFromPermanentExcept(This, You, Fixed(1)), p.ID(), b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		g.AddPreventionShield(b.PlayerID(), 1)
		g.damage.SetResolvingCombatDamage(true)
		g.DealDamageToPlayer(b, 2, p.ID())
		firstLife := 20
		if choice == 1 {
			firstLife = 19
		}
		if b.Life() != firstLife || b.prompts != 1 {
			t.Fatal("affected player did not choose order")
		}
		g.DealDamageToPlayer(b, 4, p.ID())
		want := 16
		if choice == 1 {
			want = 18
		}
		if b.Life() != want {
			t.Fatalf("choice=%d life=%d want=%d", choice, b.Life(), want)
		}
	}
}

func TestNextCombatPreventionDeclarations(t *testing.T) {
	for _, effect := range []Effect{
		PreventNextCombatDamageFromPermanentExcept(nil, You, Fixed(1)),
		PreventNextCombatDamageFromPermanentExcept(This, nil, Fixed(1)),
		PreventNextCombatDamageFromPermanentExcept(This, You, nil),
		PreventNextCombatDamageFromPermanentExcept(This, You, Fixed(-1)),
		PreventNextCombatDamageFromPermanentExcept(From(BindPermanent("unbound")), You, Fixed(1)),
		PreventNextCombatDamageFromPermanentExcept(DeclarePermanentTarget("undeclared", IsCreature), You, Fixed(1)),
	} {
		c := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))
		if err := ValidateCardDefinition(c.Name(), c); err == nil {
			t.Fatal("accepted invalid declaration")
		}
	}
	for _, change := range []string{"phase", "depart", "return", "noncreature"} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		card := NewCreature("Source", "", 1, 10)
		p := g.PutOnBattlefield(card, a.PlayerID())
		incarnation := p.incarnationID
		switch change {
		case "phase":
			g.PhaseOut(p.ID())
		case "depart", "return":
			g.RemoveFromBattlefield(p)
			if change == "return" {
				g.PutOnBattlefield(card, a.PlayerID())
			}
		case "noncreature":
			p.RevokeBaseAttr(core.AttrIsCreature)
		}
		ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: b.PlayerID(), sourceCaptured: true, sourceIncarnation: incarnation}
		if err := PreventNextCombatDamageFromPermanentExcept(This, You, Fixed(1)).Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if len(g.effects.replacements) != 0 {
			t.Fatalf("%s installed a shield", change)
		}
	}
}

func TestNextCombatPreventionIndependentBindings(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	target := DeclarePermanentTarget("source", IsCreature)
	card := NewArtifact("Shield", "", WithAbility(Activate(Costs(ManaPayment("{1}")), PreventNextCombatDamageFromPermanentExcept(target, You, Fixed(1)), WithTargets(target))))
	p := g.PutOnBattlefield(NewCreature("Source", "", 4, 10), a.PlayerID())
	shield := g.PutOnBattlefield(card, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), shield.ID(), 0, []uuid.UUID{p.ID()}); err == nil || g.StackSize() != 0 {
		t.Fatal("unpaid activation changed state")
	}
	b.ManaPool().Add(core.Colorless, 1)
	if err := g.ActivateAbilityByIndex(b.PlayerID(), shield.ID(), 0, []uuid.UUID{p.ID()}); err != nil {
		t.Fatal(err)
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	g.damage.SetResolvingCombatDamage(true)
	g.DealDamageToPlayer(b, 4, p.ID())
	g.DealDamageToPlayer(b, 4, p.ID())
	if b.Life() != 18 {
		t.Fatal("copies consumed a shield that prevented no damage")
	}
}

func TestNextCombatPreventionAmountAndWaiting(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Source", "", 2, 10), a.PlayerID())
	if err := ApplyEffect(g, PreventNextCombatDamageFromPermanentExcept(This, You, PowerOf(This)), p.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, Until(core.EndOfTurn, ModifyPT(This, Fixed(3), Fixed(0))), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	refs := g.WaitingPreventionPermanentSources()
	if len(refs) != 1 || refs[0].Incarnation() != p.incarnationID {
		t.Fatal("missing exact waiting source")
	}
	g.damage.SetResolvingCombatDamage(true)
	g.DealDamageToPlayer(b, 5, p.ID())
	if b.Life() != 18 {
		t.Fatal("allowance was not saved at resolution")
	}
	g.damage.SetResolvingCombatDamage(false)
	if err := ApplyEffect(g, AtNext(EachEndStep, PreventNextCombatDamageFromPermanentExcept(AttachedPermanent, You, PowerOfWithLKI(This))), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.WaitingDelayedPermanentSources()) != 1 || len(g.WaitingPreventionPermanentSources()) != 0 {
		t.Fatal("waiting amount read missing or installed shield")
	}
}

func TestNextCombatPreventionActualDamage(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Source", "", 4, 10, WithKeyword(core.Lifelink)), a.PlayerID())
	if err := ApplyEffect(g, PreventNextCombatDamageFromPermanentExcept(This, You, Fixed(1)), p.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	calls := 0
	g.SetOnDamageDealt(func(_, _ string, amount int, combat bool) {
		calls++
		if amount != 1 || !combat {
			t.Fatal("notification did not use actual combat damage")
		}
	})
	g.damage.SetResolvingCombatDamage(true)
	g.DealDamageToPlayer(b, 4, p.ID())
	if a.Life() != 21 || b.Life() != 19 || calls != 1 {
		t.Fatal("partial prevention changed lifelink or actual damage")
	}
}
