package mage

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedPlayerValueReferences(t *testing.T) {
	for _, ref := range []PlayerRef{ControllerOf(This), OwnerOf(This), ControllerOfWithLKI(This)} {
		for _, state := range []string{"live", "departed", "returned", "phased", "cycle", "control"} {
			for name, body := range map[string]ResolvingEffect{
				"fixed-paid":               IfPaid(ref, Costs(ManaCostOf("{100}")), GainLifeFor(You, Fixed(0))),
				"fixed-unpaid":             UnlessPaid(ref, Costs(ManaCostOf("{100}")), GainLifeFor(You, Fixed(0))),
				"dynamic-payer":            IfManaPaid(ref, core.ParseManaCost("{X}"), Fixed(100), GainLifeFor(You, Fixed(0))),
				"dynamic-life":             IfManaPaid(You, core.ParseManaCost("{X}"), MultiplyValues(Fixed(0), LifeTotalOf(ref)), GainLifeFor(You, Fixed(0))),
				"action-payer":             PayActionOrElse(ref, GainLifePayment(You, Fixed(0)), GainLifeFor(You, Fixed(0)), nil),
				"action-subject":           PayActionOrElse(You, GainLifePayment(ref, Fixed(0)), GainLifeFor(You, Fixed(0)), nil),
				"action-value":             PayActionOrElse(You, GainLifePayment(You, HandSizeOf(ref)), GainLifeFor(You, Fixed(0)), nil),
				"counter-payer":            PayCountersOrElse(ref, AttachedPermanent, core.Charge, 0, true, GainLifeFor(You, Fixed(0)), nil),
				"sacrifice-payer":          IfSacrificePaid(ref, AttachedPermanent, GainLifeFor(You, Fixed(0))),
				"life-subject":             GainLifeFor(ref, Fixed(0)),
				"saved-life-subject":       savedLifeSubjectEffect(ref),
				"life-total":               LoseLifeFor(You, MultiplyValues(Fixed(0), LifeTotalOf(ref))),
				"life-gained":              GainLifeFor(You, LifeGainedThisTurn(ref)),
				"damage-history":           GainLifeFor(You, DamageTakenThisTurn(ref)),
				"artifact-history":         GainLifeFor(You, ArtifactDamageTakenThisTurn(ref)),
				"damage-payment-recipient": PayActionOrElse(You, DamagePayment(DamageSourceOf(AttachedPermanent), ref, Fixed(0)), GainLifeFor(You, Fixed(0)), nil),
				"damage-payment-value":     PayActionOrElse(You, DamagePayment(DamageSourceOf(AttachedPermanent), You, LifeTotalOf(ref)), GainLifeFor(You, Fixed(0)), nil),
				"draw-subject":             DrawFor(ref, Fixed(0)),
				"draw-value":               DrawFor(You, HandSizeOf(ref)),
			} {
				t.Run(fmt.Sprintf("%T/%s/%s", ref, name, state), func(t *testing.T) {
					a, b := NewBasePlayer("A"), NewBasePlayer("B")
					g := NewGame(a, b)
					card := NewArtifact("Source", "{U}")
					card.SetOwner(a.PlayerID())
					p := g.PutOnBattlefield(card, a.PlayerID())
					original := g.CaptureDamageSource(p.ID())
					delay := AtNext(EachEndStep, Sequence(body))
					if err := ValidateCardDefinition("Source", NewArtifact("Source", "", WithAbility(Activate(Costs(), delay)))); err != nil {
						t.Fatal(err)
					}
					if err := ApplyEffect(g, delay, p.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					switch state {
					case "departed", "returned":
						g.BouncePermanentToHand(p)
						if state == "returned" {
							a.RemoveFromHand(card.ID())
							g.PutOnBattlefield(card, b.PlayerID())
							g.ChangeObjectColor(card.ID(), core.Red)
						}
					case "phased":
						g.PhaseOut(p.ID())
					case "cycle":
						g.PhaseIn(g.PhaseOut(p.ID()))
					case "control":
						g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					}
					for range 3 {
						got := g.WaitingDelayedPermanentSources()
						if len(got) != 1 || !got[0].SameObject(original) {
							t.Fatal("typed player operand lost saved source")
						}
						if got[0].HasColor(core.Red) {
							t.Fatal("returned source supplied current facts")
						}
						if state == "control" && got[0].Controller() != b.PlayerID() {
							t.Fatal("lost current controller")
						}
						got[0].card.AddType(core.TypeEnchantment)
						if g.WaitingDelayedPermanentSources()[0].HasType(core.TypeEnchantment) {
							t.Fatal("query shares card facts")
						}
					}
					if a.Life() != 20 || b.Life() != 20 || g.StackSize() != 0 {
						t.Fatal("query executed body")
					}
					clone := g.Clone()
					g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					if len(g.WaitingDelayedPermanentSources()) != 0 || len(clone.WaitingDelayedPermanentSources()) != 1 {
						t.Fatal("waiting lifetime shared with clone")
					}
				})
			}
		}
	}
}

func TestWaitingDelayedPlayerValueBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	reads := 0
	for _, ref := range []PlayerRef{You, ActivePlayer, ControllerOf(AttachedPermanent), OwnerOf(AttachedPermanent), ControllerOfWithLKI(AttachedPermanent), waitingPaymentPayer{PlayerRef: ControllerOfWithLKI(This), calls: &reads}} {
		body := IfManaPaid(ref, core.ParseManaCost("{X}"), LifeTotalOf(ref), GainLifeFor(You, Fixed(0)))
		if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(g.WaitingDelayedPermanentSources()) != 0 || reads != 0 {
		t.Fatal("query evaluated or inferred an opaque player")
	}
	g.triggers.SetDelayed(nil)
	if err := ApplyEffect(g, AtNext(EachEndStep, AtNext(EachEndStep, GainLifeFor(ControllerOfWithLKI(This), Fixed(1)))), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.WaitingDelayedPermanentSources()) != 0 {
		t.Fatal("future body declared current source")
	}
	for _, body := range []ResolvingEffect{GainLifeFor(nil, Fixed(0)), DrawFor(ControllerOf(nil), Fixed(0)), IfManaPaid(You, core.ParseManaCost("{X}"), LifeTotalOf(nil), GainLifeFor(You, Fixed(0)))} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, body))))) == nil {
			t.Fatal("invalid typed declaration accepted")
		}
	}
}

func TestWaitingDelayedPlayerValuePurity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	calls := 0
	body := IfManaPaid(ControllerOfWithLKI(This), core.ParseManaCost("{X}"), waitingMultiAmount{&calls}, GainLifeFor(You, Fixed(0)))
	if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(p)
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || calls != 0 {
			t.Fatal("query evaluated payment amount")
		}
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if calls != 1 {
		t.Fatal("resolution did not read the amount once")
	}
}

func savedLifeSubjectEffect(subject PlayerRef) ResolvingEffect {
	player := BindPlayer("life action subject")
	return Sequence(SnapshotPlayer(subject, player), LoseLifeFor(FromPlayer(player), LifeTotalOf(FromPlayer(player))))
}

type lifeSubjectProbe struct {
	PlayerRef
	reads  *int
	absent *bool
}

func (r lifeSubjectProbe) resolvePlayer(ctx *EffectContext) (uuid.UUID, bool) {
	*r.reads++
	if *r.absent {
		return uuid.Nil, false
	}
	return r.PlayerRef.resolvePlayer(ctx)
}

func TestLifeTotalBoundSubjectReadAndClear(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	reads := 0
	absent := false
	effect := savedLifeSubjectEffect(lifeSubjectProbe{PlayerRef: You, reads: &reads, absent: &absent})
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	a.SetLife(13)
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if reads != 1 || a.Life() != 0 || b.Life() != 20 {
		t.Fatal("subject was not saved once")
	}
	a.SetLife(9)
	absent = true
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if reads != 2 || a.Life() != 9 || b.Life() != 20 {
		t.Fatal("absent subject retained an earlier binding")
	}
}
