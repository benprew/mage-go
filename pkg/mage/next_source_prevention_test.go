package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestNextPermanentSourcePreventionIdentity(t *testing.T) {
	for _, change := range []string{"live", "return", "depart", "phase", "phase-cycle", "control", "cleanup"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Source", "", 2, 10)
			p := g.PutOnBattlefield(card, a.PlayerID())
			old := g.CaptureDamageSource(p.ID())
			if err := ApplyEffect(g, PreventNextDamageFromPermanent(This, You, PermanentFilter{}), p.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "return", "depart":
				g.RemoveFromBattlefield(p)
				if change == "return" {
					g.PutOnBattlefield(card, a.PlayerID())
				}
				g.zones.ClearLKI()
			case "phase", "phase-cycle":
				ids := g.PhaseOut(p.ID())
				if change == "phase-cycle" {
					g.PhaseIn(ids)
				}
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "cleanup":
				g.effects.ClearReplacementsEndOfTurn()
			}
			clone := g.Clone()
			for _, game := range []*Game{g, clone} {
				recipient := game.GetPlayer(b.PlayerID())
				if change == "return" {
					game.DealDamageToPlayer(recipient, 1, p.ID())
					if recipient.Life() != 19 {
						t.Fatal("returned source used the old shield")
					}
				}
				before := recipient.Life()
				game.DealDamageToPlayerFromSource(recipient, 0, old)
				game.DealDamageToPlayerFromSource(game.GetPlayer(a.PlayerID()), 1, old)
				game.damage.SetResolvingCombatDamage(change == "live")
				game.DealDamageToPlayerFromSource(recipient, 2, old)
				want := before
				if change == "cleanup" {
					want -= 2
				}
				if recipient.Life() != want {
					t.Fatalf("first event life=%d want=%d", recipient.Life(), want)
				}
				game.DealDamageToPlayerFromSource(recipient, 3, old)
				if recipient.Life() != want-3 {
					t.Fatal("shield did not expire after one matching event")
				}
			}
		})
	}
}

func TestNextPermanentSourcePreventionDeclarations(t *testing.T) {
	for _, body := range []Effect{
		PreventNextDamageFromPermanent(nil, You, PermanentFilter{}),
		PreventNextDamageFromPermanent(This, nil, PermanentFilter{}),
		PreventNextDamageFromPermanent(From(BindPermanent("unbound")), You, PermanentFilter{}),
		PreventNextDamageFromPermanent(This, FromPlayer(BindPlayer("unbound")), PermanentFilter{}),
		PreventNextDamageFromPermanent(DeclarePermanentTarget("undeclared", IsCreature), You, PermanentFilter{}),
	} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), body)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid declaration")
		}
	}
	for _, change := range []string{"departed", "phased", "returned"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Source", "", 1, 10)
			p := g.PutOnBattlefield(card, a.PlayerID())
			incarnation := p.incarnationID
			if change == "phased" {
				g.PhaseOut(p.ID())
			} else {
				g.RemoveFromBattlefield(p)
				if change == "returned" {
					g.PutOnBattlefield(card, a.PlayerID())
				}
			}
			ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: b.PlayerID(), sourceCaptured: true, sourceIncarnation: incarnation}
			if err := PreventNextDamageFromPermanent(This, You, PermanentFilter{}).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if len(g.effects.replacements) != 0 {
				t.Fatal("absent source installed a shield")
			}
		})
	}
}

func TestNextPermanentSourcePreventionSimultaneous(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Source", "", 2, 10), a.PlayerID())
	q := g.PutOnBattlefield(NewCreature("Other", "", 2, 10), a.PlayerID())
	if err := ApplyEffect(g, PreventNextDamageFromPermanent(This, You, PermanentFilter{}), p.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.DealSimultaneousDamage([]DamageAssignment{PlayerDamageAssignment(b, 2, g.CaptureDamageSource(p.ID())), PlayerDamageAssignment(b, 3, g.CaptureDamageSource(q.ID()))}...)
	if b.Life() != 17 {
		t.Fatalf("life=%d", b.Life())
	}
	g.DealDamageToPlayer(b, 1, p.ID())
	if b.Life() != 16 {
		t.Fatal("batch did not consume shield")
	}
}

func TestNextPermanentSourcePreventionPropertyRechecks(t *testing.T) {
	for _, change := range []string{"color", "type", "depart", "return", "phase", "phase-cycle"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Source", "{R}", 2, 10, WithCardType(core.TypeArtifact))
			p := g.PutOnBattlefield(card, a.PlayerID())
			filter := And(IsArtifact, HasColorFilter(core.Red))
			if err := ApplyEffect(g, PreventNextDamageFromPermanent(This, You, filter), p.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if len(g.effects.replacements) != 1 {
				t.Fatal("matching source did not install shield")
			}
			old := g.CaptureDamageSource(p.ID())
			switch change {
			case "color":
				if err := ApplyEffect(g, Until(core.Indefinite, SetColors(This, core.Blue)), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "type":
				p.RevokeBaseAttr(core.AttrIsArtifact)
			case "depart", "return":
				g.RemoveFromBattlefield(p)
				if change == "return" {
					g.PutOnBattlefield(card, a.PlayerID())
				}
			case "phase", "phase-cycle":
				ids := g.PhaseOut(p.ID())
				if change == "phase-cycle" {
					g.PhaseIn(ids)
				}
			}
			g.DealDamageToPlayerFromSource(b, 2, old)
			want := 20
			if change == "color" || change == "type" {
				want = 18
			}
			if b.Life() != want {
				t.Fatalf("first damage life=%d want=%d", b.Life(), want)
			}
			if change == "color" {
				if err := ApplyEffect(g, Until(core.Indefinite, SetColors(This, core.Red)), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			if change == "type" {
				p.GrantBaseAttr(core.AttrIsArtifact)
			}
			g.DealDamageToPlayerFromSource(b, 1, old)
			if want == 18 {
				if b.Life() != 18 {
					t.Fatal("nonmatching damage consumed shield")
				}
			} else if b.Life() != 19 {
				t.Fatal("matching damage did not consume shield")
			}
		})
	}
}

func TestNextPermanentSourcePreventionReplacementOrder(t *testing.T) {
	for choice := range 2 {
		a := NewBasePlayer("A")
		b := &sourceFilterChoicePlayer{BasePlayer: NewBasePlayer("B"), choice: choice}
		g := NewGame(a, b)
		p := g.PutOnBattlefield(NewCreature("Source", "", 1, 10), a.PlayerID())
		if err := ApplyEffect(g, PreventNextDamageFromPermanent(This, You, PermanentFilter{}), p.ID(), b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		g.AddPreventionShield(b.PlayerID(), 1)
		g.DealDamageToPlayer(b, 1, p.ID())
		if b.Life() != 20 || b.prompts != 1 {
			t.Fatal("affected player did not choose prevention order")
		}
		g.DealDamageToPlayer(b, 2, p.ID())
		want := 19
		if choice == 1 {
			want = 20
		}
		if b.Life() != want {
			t.Fatalf("choice=%d life=%d want=%d", choice, b.Life(), want)
		}
	}
}

func TestNextPermanentSourcePreventionOneSimultaneousEvent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Source", "", 2, 10), a.PlayerID())
	if err := ApplyEffect(g, PreventNextDamageFromPermanent(This, You, PermanentFilter{}), p.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	source := g.CaptureDamageSource(p.ID())
	g.DealSimultaneousDamage(PlayerDamageAssignment(b, 2, source), PlayerDamageAssignment(b, 3, source), PlayerDamageAssignment(a, 1, source))
	if b.Life() != 20 || a.Life() != 19 {
		t.Fatal("one simultaneous source event was split across shield uses")
	}
	g.DealDamageToPlayer(b, 1, p.ID())
	if b.Life() != 19 {
		t.Fatal("shield survived its matching event")
	}
}

func TestNextPermanentSourcePreventionIndependentBindings(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 10), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 10), a.PlayerID())
	target := DeclarePermanentTarget("source", IsCreature)
	card := NewArtifact("Shield", "", WithAbility(Activate(Costs(ManaPayment("{1}")), PreventNextDamageFromPermanent(target, You, PermanentFilter{}), WithTargets(target))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	protector := g.PutOnBattlefield(card, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), protector.ID(), 0, []uuid.UUID{first.ID()}); err == nil || g.StackSize() != 0 {
		t.Fatal("unpaid activation succeeded")
	}
	b.ManaPool().Add(core.Colorless, 3)
	for _, p := range []*Permanent{first, second} {
		if err := g.ActivateAbilityByIndex(b.PlayerID(), protector.ID(), 0, []uuid.UUID{p.ID()}); err != nil {
			t.Fatal(err)
		}
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	clone := g.Clone()
	g.DealDamageToPlayer(b, 1, first.ID())
	g.DealDamageToPlayer(b, 1, first.ID())
	g.DealDamageToPlayer(b, 1, second.ID())
	g.DealDamageToPlayer(b, 1, second.ID())
	g.DealDamageToPlayer(b, 1, second.ID())
	if b.Life() != 18 {
		t.Fatal("independent activations or copies shared a shield")
	}
	clone.DealDamageToPlayer(clone.GetPlayer(b.PlayerID()), 1, first.ID())
	if clone.GetPlayer(b.PlayerID()).Life() != 20 {
		t.Fatal("shield consumption changed clone")
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), protector.ID(), 0, []uuid.UUID{first.ID()}); err != nil {
		t.Fatal(err)
	}
	g.PhaseOut(first.ID())
	g.ResolveStack()
	g.PhaseIn([]uuid.UUID{first.ID()})
	g.DealDamageToPlayer(b, 1, first.ID())
	if b.Life() != 17 {
		t.Fatal("illegal target installed shield")
	}
}

func TestNextPermanentSourcePreventionMissingInformation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Source", "{R}", 1, 10), a.PlayerID())
	if err := ApplyEffect(g, PreventNextDamageFromPermanent(This, You, HasColorFilter(core.Blue)), p.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.effects.replacements) != 0 {
		t.Fatal("nonmatching installation succeeded")
	}
	if err := ApplyEffect(g, PreventNextDamageFromPermanent(This, You, HasColorFilter(core.Red)), p.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	source := g.CaptureDamageSource(p.ID())
	g.DealDamageToPlayerFromSource(b, 1, DamageSource{id: p.ID()})
	g.DealDamageToPlayerFromSource(b, 1, DamageSource{id: p.ID(), incarnation: uuid.New()})
	if b.Life() != 18 {
		t.Fatal("missing information matched captured source")
	}
	g.DealDamageToPlayerFromSource(b, 1, source)
	if b.Life() != 18 {
		t.Fatal("missing information consumed shield")
	}
}
