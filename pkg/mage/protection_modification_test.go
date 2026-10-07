package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestProtectionModificationCaptureAndLayers(t *testing.T) {
	for _, change := range []string{"none", "phase-cycle", "return", "remove-abilities"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Subject", "", 2, 8)
			p := g.PutOnBattlefield(card, a.PlayerID())
			white := g.PutOnBattlefield(NewCreature("White", "{W}", 2, 2), b.PlayerID())
			red := g.PutOnBattlefield(NewCreature("Red", "{R}", 2, 2), b.PlayerID())
			colors := []Color{White, Blue}
			mod := GrantProtectionColors(This, colors...)
			colors[0] = Red
			if err := ApplyEffect(g, Until(EndOfTurn, mod), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			for range 3 {
				g.ApplyContinuousEffects()
			}
			if !g.FindPermanent(p.ID()).HasProtectionFromInGame(white.Card, g) || g.FindPermanent(p.ID()).HasProtectionFromInGame(red.Card, g) || len(g.FindPermanent(p.ID()).RuntimeAbilities) != 1 {
				t.Fatal("protection colors or layer reset")
			}
			if CanBlock(white, g.FindPermanent(p.ID()), g) || !CanBlock(red, g.FindPermanent(p.ID()), g) {
				t.Fatal("protection blocking contract")
			}
			g.DealDamageToPermanent(g.FindPermanent(p.ID()), 3, white.ID())
			if g.FindPermanent(p.ID()).Damage != 0 {
				t.Fatal("protection did not prevent damage")
			}
			clone := g.Clone()
			switch change {
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				g.PutOnBattlefield(card, a.PlayerID())
			case "remove-abilities":
				if err := ApplyEffect(g, Until(EndOfTurn, RemoveAbilities(This)), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			want := change == "none" || change == "phase-cycle"
			if g.FindPermanent(card.ID()).HasProtectionFromInGame(white.Card, g) != want || !clone.FindPermanent(card.ID()).HasProtectionFromInGame(white.Card, clone) {
				t.Fatal("identity, timestamp or clone isolation")
			}
			g.effects.RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			if g.FindPermanent(card.ID()).HasProtectionFromInGame(white.Card, g) || !clone.FindPermanent(card.ID()).HasProtectionFromInGame(white.Card, clone) {
				t.Fatal("expiry or clone isolation")
			}
		})
	}
}

func TestProtectionModificationValidation(t *testing.T) {
	target := DeclarePermanentTarget("subject", IsCreature)
	for _, colors := range [][]Color{nil, {Colorless}, {AnyColor}, {White, Red}} {
		body := Until(EndOfTurn, GrantProtectionColors(target, colors...))
		card := NewInstant("Grant", "", WithAbility(Spell(body, WithTargets(target))))
		if (ValidateCardDefinition(card.Name(), card) == nil) != (len(colors) == 2) {
			t.Fatalf("validation accepted invalid colors %v", colors)
		}
	}
	body := Until(EndOfTurn, GrantProtectionColors(target, White))
	if ValidateCardDefinition("Unbound", NewInstant("Unbound", "", WithAbility(Spell(body)))) == nil {
		t.Fatal("undeclared subject accepted")
	}
	if ValidateCardDefinition("Nil", NewInstant("Nil", "", WithAbility(Spell(Until(EndOfTurn, GrantProtectionColors(nil, White)))))) == nil {
		t.Fatal("nil subject accepted")
	}
}

func TestProtectionModificationReplacementOrder(t *testing.T) {
	for _, choice := range []int{0, 1} {
		a := &auditChoicePlayer{BasePlayer: NewBasePlayer("A"), choice: choice}
		b := &auditChoicePlayer{BasePlayer: NewBasePlayer("B"), choice: 1 - choice}
		g := NewGame(a, b)
		source := g.PutOnBattlefield(NewCreature("White", "{W}", 2, 2), b.PlayerID())
		target := g.PutOnBattlefield(NewCreature("Protected", "", 2, 10), a.PlayerID())
		if err := ApplyEffect(g, Until(EndOfTurn, GrantProtectionColors(This, White)), target.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		g.AddPreventionShield(target.ID(), 2)
		g.DealDamageToPermanent(g.FindPermanent(target.ID()), 3, source.ID())
		if a.calls != 1 || b.calls != 0 || g.FindPermanent(target.ID()).Damage != 0 {
			t.Fatal("affected player did not choose protection order")
		}
		g.ChangeObjectColor(source.ID(), Blue)
		g.DealDamageToPermanent(g.FindPermanent(target.ID()), 2, source.ID())
		want := 2
		if choice == 1 {
			want = 0
		}
		if g.FindPermanent(target.ID()).Damage != want {
			t.Fatalf("choice %d: remaining shield damage %d want %d", choice, g.FindPermanent(target.ID()).Damage, want)
		}
	}
}

func TestProtectionModificationRedirectionAndBatch(t *testing.T) {
	for _, protectFirst := range []bool{false, true} {
		a := &auditChoicePlayer{BasePlayer: NewBasePlayer("A")}
		b := NewBasePlayer("B")
		g := NewGame(a, b)
		source := g.PutOnBattlefield(NewCreature("White", "{W}", 2, 2), b.PlayerID())
		first := g.PutOnBattlefield(NewCreature("First", "", 2, 10), a.PlayerID())
		second := g.PutOnBattlefield(NewCreature("Second", "", 2, 10), a.PlayerID())
		protected := second
		if protectFirst {
			protected = first
		}
		if err := ApplyEffect(g, Until(EndOfTurn, GrantProtectionColors(This, White)), protected.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		g.effects.AddReplacement(&testPrioritizedReplacement{
			matchFunc: func(action Action) bool {
				d, ok := action.(*DamageToCreatureAction)
				return ok && d.PermanentID() == first.ID()
			},
			replaceFunc: func(action Action, _ *Game) Action {
				d := action.(*DamageToCreatureAction)
				return NewDamageToCreatureAction(d.ActionSource(), second.ID(), d.Amount(), d.IsCombatDamage()).withDamageSource(d.DamageSource())
			},
		})
		g.DealDamageToPermanent(g.FindPermanent(first.ID()), 3, source.ID())
		want := 0
		if protectFirst {
			want = 3
		}
		if g.FindPermanent(first.ID()).Damage != 0 || g.FindPermanent(second.ID()).Damage != want {
			t.Fatal("protection did not recheck redirected recipient")
		}
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	white := g.PutOnBattlefield(NewCreature("White", "{W}", 2, 2), b.PlayerID())
	red := g.PutOnBattlefield(NewCreature("Red", "{R}", 2, 2), b.PlayerID())
	target := g.PutOnBattlefield(NewCreature("Subject", "", 2, 10), a.PlayerID())
	if err := ApplyEffect(g, Until(EndOfTurn, GrantProtectionColors(This, White)), target.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.damage.dealSimultaneousDamageToPermanents(g, []damageRequest{{permanent: target, amount: 3, sourceID: white.ID()}, {permanent: target, amount: 2, sourceID: red.ID()}})
	if g.FindPermanent(target.ID()).Damage != 2 {
		t.Fatal("simultaneous protection")
	}
	g.MutablePermanent(target.ID()).GrantBaseAttr(AttrDamageCantBePreventedOrRedirected)
	g.DealDamageToPermanent(g.FindPermanent(target.ID()), 3, white.ID())
	if g.FindPermanent(target.ID()).Damage != 5 {
		t.Fatal("protection prevented unpreventable damage")
	}
}

func TestProtectionModificationAttachments(t *testing.T) {
	for _, subtype := range []string{"Equipment", "Fortification"} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		var host Card = NewCreature("Host", "", 2, 3)
		if subtype == "Fortification" {
			host = NewLand("Host")
		}
		p := g.PutOnBattlefield(host, a.PlayerID())
		white := g.PutOnBattlefield(NewArtifact("White", "{W}", WithSubTypes(subtype)), b.PlayerID())
		red := g.PutOnBattlefield(NewArtifact("Red", "{R}", WithSubTypes(subtype)), b.PlayerID())
		g.Attach(white.ID(), p.ID())
		g.Attach(red.ID(), p.ID())
		if err := ApplyEffect(g, Until(EndOfTurn, GrantProtectionColors(This, White)), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		clone := g.Clone()
		g.CheckStateBasedActions()
		if g.FindPermanent(white.ID()) == nil || g.FindPermanent(white.ID()).IsAttached() || !g.FindPermanent(red.ID()).IsAttached() {
			t.Fatalf("%s protection attachment legality", subtype)
		}
		if !clone.FindPermanent(white.ID()).IsAttached() || len(clone.FindPermanent(p.ID()).Attachments) != 2 {
			t.Fatal("detachment changed clone")
		}
		clone.CheckStateBasedActions()
		if clone.FindPermanent(white.ID()).IsAttached() || len(clone.FindPermanent(p.ID()).Attachments) != 1 {
			t.Fatal("clone attachment legality")
		}
		g.Attach(white.ID(), p.ID())
		if g.FindPermanent(white.ID()).IsAttached() || len(g.FindPermanent(p.ID()).Attachments) != 1 {
			t.Fatal("protected attachment restored or stale host list")
		}
	}
}
