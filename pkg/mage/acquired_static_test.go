package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func acquiredRemovalRule() StaticRule {
	condition := StateConditionFunc("always", []StateDependency{StateBattlefield}, func(StateReader, StateContext) bool { return true })
	return While(condition, RemoveKeyword(AttachedPermanent, core.Flying))
}

func TestAcquiredStaticLifecycle(t *testing.T) {
	for _, change := range []string{"normal", "reattach", "donor-left", "donor-return", "host-return", "phase", "remove-before", "remove-after", "later-flying"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			hostCard := NewCreature("Host", "", 2, 5, WithKeyword(core.Flying))
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			donorCard := NewAura("Donor", "")
			donor := g.PutOnBattlefield(donorCard, a.PlayerID())
			g.Attach(donor.ID(), host.ID())
			if change == "remove-before" {
				g.AddContinuousEffect(RemoveAllAbilities(donor.ID(), core.Indefinite))
			}
			if err := ApplyEffect(g, AcquireStaticAbility(This, acquiredRemovalRule()), donor.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(host.ID()).HasKeyword(core.Flying) {
				t.Fatal("acquired rule did not run in its first layer cycle")
			}
			clone := g.Clone()
			switch change {
			case "reattach":
				other := g.PutOnBattlefield(NewCreature("Other", "", 2, 5, WithKeyword(core.Flying)), b.PlayerID())
				g.Attach(donor.ID(), other.ID())
				if g.FindPermanent(other.ID()).HasKeyword(core.Flying) {
					t.Fatal("rule did not follow current attachment")
				}
			case "donor-left", "donor-return":
				g.RemoveFromBattlefield(donor)
				if change == "donor-return" {
					returned := g.PutOnBattlefield(donorCard, a.PlayerID())
					g.Attach(returned.ID(), host.ID())
				}
			case "host-return":
				g.RemoveFromBattlefield(host)
				g.PutOnBattlefield(hostCard, b.PlayerID())
			case "phase":
				g.PhaseOut(donor.ID())
			case "remove-after":
				g.AddContinuousEffect(RemoveAllAbilities(donor.ID(), core.Indefinite))
			case "later-flying":
				if err := ApplyEffect(g, Until(core.Indefinite, GrantKeywordModification(AttachedPermanent, core.Flying)), donor.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			g.ApplyContinuousEffects()
			want := change != "normal" && change != "remove-before"
			if g.FindPermanent(host.ID()).HasKeyword(core.Flying) != want {
				t.Fatalf("flying=%v want %v", g.FindPermanent(host.ID()).HasKeyword(core.Flying), want)
			}
			clone.ApplyContinuousEffects()
			if clone.FindPermanent(host.ID()).HasKeyword(core.Flying) {
				t.Fatal("clone lost its independent acquired rule")
			}
			if change == "phase" {
				g.PhaseIn([]uuid.UUID{donor.ID()})
				g.ApplyContinuousEffects()
				if g.FindPermanent(host.ID()).HasKeyword(core.Flying) {
					t.Fatal("phase cycle lost rule")
				}
			}
		})
	}
}

func TestAcquiredStaticInvalid(t *testing.T) {
	condition := StateConditionFunc("always", []StateDependency{StateBattlefield}, func(StateReader, StateContext) bool { return true })
	for _, rule := range []StaticRule{nil, While(condition), While(nil, RemoveKeyword(This, core.Flying)), While(condition, ModifyPT(This, Fixed(1), Fixed(1))), While(condition, GrantKeywordModification(From(BindPermanent("unbound")), core.Flying))} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), AcquireStaticAbility(This, rule))))
		if err := ValidateCardDefinition("Invalid", card); err == nil {
			t.Fatal("invalid acquired declaration accepted")
		}
	}
}

func TestAcquiredStaticIndependentDeclarations(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	host := g.PutOnBattlefield(NewCreature("Host", "", 2, 5, WithKeyword(core.Flying)), b.PlayerID())
	donor := g.PutOnBattlefield(NewAura("Donor", ""), a.PlayerID())
	g.Attach(donor.ID(), host.ID())
	for range 2 {
		if err := ApplyEffect(g, AcquireStaticAbility(This, acquiredRemovalRule()), donor.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	p := g.FindPermanent(donor.ID())
	if len(p.RuntimeAbilities) != 2 || p.RuntimeAbilities[0].AbilityID() == p.RuntimeAbilities[1].AbilityID() {
		t.Fatal("independent acquisitions shared declaration identity")
	}
	for range 3 {
		g.ApplyContinuousEffects()
	}
	if len(g.FindPermanent(donor.ID()).RuntimeAbilities) != 2 {
		t.Fatal("layer cycles duplicated declarations")
	}
	cloned := g.Clone()
	g.RemoveFromBattlefield(donor)
	g.ApplyContinuousEffects()
	cloned.ApplyContinuousEffects()
	if !g.FindPermanent(host.ID()).HasKeyword(core.Flying) || cloned.FindPermanent(host.ID()).HasKeyword(core.Flying) {
		t.Fatal("source lifetime or clone isolation failed")
	}
}

func TestAcquiredStaticRecipientContext(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	donor := g.PutOnBattlefield(NewAura("Donor", ""), a.PlayerID())
	recipient := g.PutOnBattlefield(NewCreature("Recipient", "", 2, 5), b.PlayerID())
	g.Attach(donor.ID(), recipient.ID())
	condition := StateConditionFunc("recipient controller", []StateDependency{StateBattlefield}, func(_ StateReader, context StateContext) bool {
		return context.SourceID == recipient.ID() && context.ControllerID == b.PlayerID()
	})
	rule := While(condition, GrantKeywordModification(This, core.Flying))
	if err := ApplyEffect(g, AcquireStaticAbility(AttachedPermanent, rule), donor.ID(), a.PlayerID(), []uuid.UUID{recipient.ID()}); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(recipient.ID()).HasKeyword(core.Flying) || g.FindPermanent(donor.ID()).HasKeyword(core.Flying) {
		t.Fatal("acquired rule used donor context")
	}
	g.RemoveFromBattlefield(donor)
	g.ApplyContinuousEffects()
	if !g.FindPermanent(recipient.ID()).HasKeyword(core.Flying) {
		t.Fatal("donor departure ended recipient's acquired ability")
	}
}

func TestAcquiredStaticWaitingReferences(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewAura("Source", "")
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	saved := g.CaptureDamageSource(p.ID())
	if err := ApplyEffect(g, AtNext(EachEndStep, AcquireStaticAbility(This, acquiredRemovalRule())), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(p)
	a.RemoveFromHand(card.ID())
	g.PutOnBattlefield(card, b.PlayerID())
	clone := g.Clone()
	for range 3 {
		got := g.WaitingDelayedPermanentSources()
		if len(got) != 1 || !got[0].SameObject(saved) {
			t.Fatal("waiting acquisition lost original source visit")
		}
	}
	g.triggers.SetDelayed(nil)
	if len(clone.WaitingDelayedPermanentSources()) != 1 {
		t.Fatal("clone lost waiting lifetime")
	}
	if err := ApplyEffect(g, AtNext(EachEndStep, AcquireStaticAbility(AttachedPermanent, acquiredRemovalRule())), card.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.WaitingDelayedPermanentSources()) != 0 {
		t.Fatal("static rule or implicit attachment leaked a waiting source read")
	}
}
