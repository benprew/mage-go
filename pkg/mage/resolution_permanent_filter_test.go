package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestIfPermanentMatchesInformation(t *testing.T) {
	for _, reference := range []string{"source", "target", "saved", "attachment"} {
		for _, change := range []string{"live", "lower", "control", "leave", "return", "phase", "phase-cycle", "unknown"} {
			t.Run(reference+"/"+change, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				card := NewCreature("Subject", "", 2, 8, WithKeyword(Flying))
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				target := DeclarePermanentTarget("subject", AnyPermanent)
				saved := BindPermanent("subject")
				ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: p.ID(), sourceIncarnation: p.incarnationID, sourceCaptured: true, targetBindings: map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{p.ID(), p.incarnationID}}}}
				if err := Snapshot(target, saved).Apply(ctx); err != nil {
					t.Fatal(err)
				}
				var ref PermanentRef = This
				switch reference {
				case "target":
					ref = target
				case "saved":
					ref = From(saved)
				case "attachment":
					aura := g.PutOnBattlefield(NewEnchantment("Aura", "", WithSubTypes("Aura")), a.PlayerID())
					g.Attach(aura.ID(), p.ID())
					ctx.SourceID, ctx.sourceIncarnation = aura.ID(), aura.incarnationID
					ref = AttachedPermanent
				}
				switch change {
				case "lower":
					if err := Until(EndOfTurn, ModifyPT(target, Fixed(-2), Fixed(0))).Apply(ctx); err != nil {
						t.Fatal(err)
					}
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
				case "leave", "return":
					g.BouncePermanentToHand(g.FindPermanent(p.ID()))
					if change == "return" {
						a.RemoveFromHand(card.ID())
						q := g.PutOnBattlefield(card, a.PlayerID())
						q.RevokeBaseAttr(Flying)
						q.AddCounter(M1M1, 2)
					}
				case "phase":
					g.PhaseOut(p.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "unknown":
					ctx.sourceIncarnation = uuid.New()
					ctx.targetBindings[target.targetDeclaration][0].incarnation = uuid.New()
					g.resolution.permanentBindings[saved] = capturedPermanent{p.ID(), uuid.New()}
				}
				filter := And(HasPowerGTE(1), HasKeywordFilter(Flying))
				effect := IfPermanentMatches(ref, filter, GainLifeFor(You, Fixed(3)))
				clone := g.Clone()
				for _, game := range []*Game{g, clone} {
					local := *ctx
					local.Game = game
					if err := effect.Apply(&local); err != nil {
						t.Fatal(err)
					}
					want := 23
					if change == "lower" || change == "unknown" {
						want = 20
					}
					if game.GetPlayer(a.PlayerID()).Life() != want || game.GetPlayer(b.PlayerID()).Life() != 20 {
						t.Fatal("wrong condition or controller", game.GetPlayer(a.PlayerID()).Life())
					}
				}
				if change == "leave" || change == "return" || change == "phase" {
					if _, ok := ref.resolvePermanent(ctx); ok {
						t.Fatal("information made action reference live")
					}
				}
			})
		}
	}
}

func TestIfPermanentMatchesDeclaration(t *testing.T) {
	for _, ref := range []PermanentRef{nil, DeclarePermanentTarget("unbound", AnyPermanent)} {
		card := NewCreature("Invalid", "", 2, 2, WithAbility(Activate(Costs(), IfPermanentMatches(ref, AnyPermanent, GainLifeFor(You, Fixed(1))))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid reference accepted")
		}
	}
	card := NewCreature("Invalid", "", 2, 2, WithAbility(Activate(Costs(), IfPermanentMatches(This, AnyPermanent, nil))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("nil body accepted")
	}
}

func TestIfPermanentMatchesSequenceAndSnapshot(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Subject", "", 2, 8)
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	effect := Sequence(TapSubject(This), IfPermanentMatches(This, IsTapped, GainLifeFor(You, Fixed(1))), Until(EndOfTurn, ModifyPT(This, Fixed(3), Fixed(0))))
	if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 21 {
		t.Fatal("condition was evaluated before the preceding action")
	}
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: p.ID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	g.RunStepWithPriority(Cleanup)
	a.RemoveFromHand(card.ID())
	g.PutOnBattlefield(card, a.PlayerID())
	if err := IfPermanentMatches(This, And(IsTapped, HasPowerEqual(5)), GainLifeFor(You, Fixed(2))).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 23 {
		t.Fatal("condition lost the original effective power or tapped state")
	}
	if err := IfPermanentMatches(This, AnyPermanent, TapSubject(This)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(card.ID()).Tapped {
		t.Fatal("LKI condition permitted an action on the new object")
	}
}
