package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDamageRecipientProhibition(t *testing.T) {
	for _, state := range []string{"live", "abilities", "phase-cycle", "control", "clone", "expired", "returned"} {
		t.Run(state, func(t *testing.T) {
			g, a, b, subject, source := finiteRedirectFixture(t, 2)
			g.AddPreventionShield(subject.ID(), 3)
			if err := ApplyEffect(g, Until(core.EndOfTurn, ProhibitDamagePreventionAndRedirection(This)), subject.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			switch state {
			case "abilities":
				g.AddContinuousEffect(RemoveAllAbilities(subject.ID(), core.Indefinite))
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(subject.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: subject.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "clone":
				g = g.Clone()
			case "expired":
				g.RunStepWithPriority(core.Cleanup)
			case "returned":
				card := subject.Card
				g.BouncePermanentToHand(subject)
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
			}
			subject = g.FindPermanent(subject.ID())
			g.DealDamageToPermanent(subject, 2, source.ID())
			subject = g.FindPermanent(subject.ID())
			want := 2
			if subject.Damage != want || g.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatalf("damage=%d life=%d", subject.Damage, g.GetPlayer(a.PlayerID()).Life())
			}
			if state != "expired" && state != "returned" {
				g.effects.RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				if subject.HasAttr(core.AttrDamageCantBePreventedOrRedirected) {
					t.Fatal("rule survived cleanup")
				}
			}
		})
	}
}

func TestDamageRecipientProhibitionDefinition(t *testing.T) {
	for _, ref := range []PermanentRef{nil, From(BindPermanent("unbound"))} {
		card := NewInstant("Invalid", "", WithAbility(Spell(Until(core.EndOfTurn, ProhibitDamagePreventionAndRedirection(ref)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid reference accepted")
		}
	}
}

func TestDamageRecipientProhibitionReplacementBatch(t *testing.T) {
	for _, order := range []string{"prevent-first", "redirect-first", "double"} {
		t.Run(order, func(t *testing.T) {
			g, a, _, subject, source := finiteRedirectFixture(t, 2)
			g.AddPreventionShield(subject.ID(), 3)
			if order == "prevent-first" {
				g.effects.PrependReplacement(g.effects.replacements[len(g.effects.replacements)-1])
			}
			if order == "double" {
				g.effects.AddReplacement(&continuedDamageDouble{})
			}
			if err := ApplyEffect(g, Until(core.EndOfTurn, ProhibitDamagePreventionAndRedirection(This)), subject.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			other := g.PutOnBattlefield(NewCreature("Other", "", 1, 20), a.PlayerID())
			g.AddPreventionShield(other.ID(), 2)
			snapshot := g.Clone()
			g.DealSimultaneousDamage(PermanentDamageAssignment(subject, 2, g.CaptureDamageSource(source.ID())), PermanentDamageAssignment(other, 2, g.CaptureDamageSource(source.ID())))
			want, otherWant := 2, 0
			if order == "double" {
				want, otherWant = 4, 2
			}
			if g.FindPermanent(subject.ID()).Damage != want || g.FindPermanent(other.ID()).Damage != otherWant || a.Life() != 20 {
				t.Fatalf("batch damage %d/%d life %d; want %d/%d", g.FindPermanent(subject.ID()).Damage, g.FindPermanent(other.ID()).Damage, a.Life(), want, otherWant)
			}
			for _, replacement := range g.effects.replacements {
				switch r := replacement.(type) {
				case *preventionShieldReplacement:
					if r.targetID == subject.ID() && r.remaining != 3 {
						t.Fatal("prohibited prevention consumed its shield")
					}
				case *finiteDamageRedirectReplacement:
					if r.remaining != 2 {
						t.Fatal("prohibited redirection consumed its shield")
					}
				}
			}
			if snapshot.FindPermanent(subject.ID()).Damage != 0 {
				t.Fatal("batch mutated clone")
			}
			snapshot.DealDamageToPermanent(snapshot.FindPermanent(subject.ID()), 1, source.ID())
			if snapshot.FindPermanent(subject.ID()).Damage == 0 {
				t.Fatal("clone lost damage rule")
			}
		})
	}
}

func TestDamageRecipientProhibitionCapture(t *testing.T) {
	for _, state := range []string{"live", "source-left", "phased", "phase-cycle", "returned", "cleanup"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			sourceCard := NewArtifact("Source", "")
			sourceCard.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(sourceCard, a.PlayerID())
			card := NewCreature("Recipient", "", 1, 20)
			card.SetOwner(b.PlayerID())
			subject := g.PutOnBattlefield(card, b.PlayerID())
			target := DeclarePermanentTarget("recipient", IsCreature)
			effect := Until(core.EndOfTurn, ProhibitDamagePreventionAndRedirection(target))
			ability := Spell(effect, WithTargets(target))
			definition := NewInstant("Valid", "", WithAbility(ability))
			if err := ValidateCardDefinition(definition.Name(), definition); err != nil {
				t.Fatal(err)
			}
			if state == "phased" {
				g.PhaseOut(subject.ID())
			}
			g.PushStack(&StackObject{ID: uuid.New(), SourceID: source.ID(), Card: sourceCard, IsAbility: true, Controller: a.PlayerID(), Targets: []uuid.UUID{subject.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()})
			g.ResolveStack()
			switch state {
			case "source-left":
				g.BouncePermanentToHand(source)
			case "phased":
				g.PhaseIn([]uuid.UUID{subject.ID()})
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(subject.ID()))
			case "returned":
				g.BouncePermanentToHand(subject)
				b.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
			case "cleanup":
				g.RunStepWithPriority(core.Cleanup)
			}
			active := state != "phased" && state != "returned" && state != "cleanup"
			if g.FindPermanent(subject.ID()).HasAttr(core.AttrDamageCantBePreventedOrRedirected) != active {
				t.Fatal("capture lifetime")
			}
		})
	}
	if core.IsAbilityAttr(core.AttrDamageCantBePreventedOrRedirected) {
		t.Fatal("damage prohibition classified as an ability")
	}
	if info := Until(core.EndOfTurn, ProhibitDamagePreventionAndRedirection(This)).Properties(); info.Outcome != OutcomeDetriment {
		t.Fatal("incorrect rule outcome")
	}
}
