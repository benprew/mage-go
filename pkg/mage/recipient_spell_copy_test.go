package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRecipientSpellCopyDefinitions(t *testing.T) {
	target := DeclareDamageTarget("recipient")
	valid := NewSorcery("Copy", "", WithAbility(Spell(CopySpellFor(PlayerOrControllerOfDamageTarget(target), ThisSpell), WithTargets(target))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, body := range []ResolvingEffect{
		CopySpellFor(nil, ThisSpell), CopySpellFor(You, nil),
		CopySpellFor(PlayerOrControllerOfDamageTarget(nil), ThisSpell),
		CopySpellFor(PlayerOrControllerOfDamageTarget(target), ThisSpell),
	} {
		c := NewSorcery("Invalid", "", WithAbility(Spell(body)))
		if ValidateCardDefinition(c.Name(), c) == nil {
			t.Fatal("invalid definition accepted")
		}
	}
}

func TestRecipientSpellCopyResolving(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	target := DeclareDamageTarget("recipient")
	c := NewSorcery("Copy", "", WithAbility(Spell(Sequence(
		GainLifeFor(PlayerOrControllerOfDamageTarget(target), Fixed(2)),
		CopySpellFor(PlayerOrControllerOfDamageTarget(target), ThisSpell),
	), WithTargets(target))))
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveTopOfStack()
	if b.Life() != 22 || g.StackSize() != 1 || !g.StackPeek().IsCopy || g.StackPeek().Controller != b.PlayerID() {
		t.Fatal("resolving copy or recipient")
	}
	if g.trackers.Turn.sorceriesCast[a.PlayerID()] != 1 || g.trackers.Turn.sorceriesCast[b.PlayerID()] != 0 {
		t.Fatal("copy was cast")
	}
	if g.StackPeek().Card.ID() == c.ID() || len(a.Graveyard()) != 1 {
		t.Fatal("copy identity or original destination")
	}
}

func TestRecipientSpellCopyInformation(t *testing.T) {
	for _, state := range []string{"present", "departed", "returned", "phase", "control", "unbound", "player"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Subject", "", 1, 2)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			target := DeclareDamageTarget("target")
			ctx := &EffectContext{Game: g, Controller: b.PlayerID(), targetBindings: map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{id: p.ID(), incarnation: p.incarnationID}}}}
			switch state {
			case "departed", "returned":
				g.DestroyPermanent(p)
				if state == "returned" {
					a.RemoveFromGraveyard(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "phase":
				g.PhaseOut(p.ID())
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "unbound":
				ctx.targetBindings = nil
			case "player":
				ctx.targetBindings[target.targetDeclaration] = []capturedPermanent{{id: b.PlayerID()}}
			}
			who, ok := PlayerOrControllerOfDamageTarget(target).resolvePlayer(ctx)
			if state == "unbound" {
				if ok {
					t.Fatal("unbound read")
				}
				return
			}
			want := a.PlayerID()
			if state == "control" || state == "player" {
				want = b.PlayerID()
			}
			if !ok || who != want {
				t.Fatal("controller information", state, who, want)
			}
		})
	}
}
