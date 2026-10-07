package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestGrantedTriggerDefinitions(t *testing.T) {
	trigger := At(YourUpkeep, GainLifeFor(You, Fixed(1))).(*GenericTriggered)
	for _, tc := range []struct {
		name string
		rule StaticRule
		bad  bool
	}{
		{"source", GrantTriggeredAbility(This, trigger), false},
		{"host", GrantTriggeredAbility(AttachedPermanent, trigger), false},
		{"collection", GrantEachTriggeredAbility(MatchingPermanents(IsCreature), trigger), false},
		{"nil subject", GrantTriggeredAbility(nil, trigger), true},
		{"nil set", GrantEachTriggeredAbility(nil, trigger), true},
		{"nil trigger", GrantTriggeredAbility(This, nil), true},
		{"unbound subject", GrantTriggeredAbility(From(BindPermanent("outer")), trigger), true},
		{"unbound body", GrantTriggeredAbility(This, At(YourUpkeep, TapSubject(From(BindPermanent("outer")))).(*GenericTriggered)), true},
		{"undeclared target", GrantTriggeredAbility(This, At(YourUpkeep, TapSubject(DeclarePermanentTarget("missing", IsCreature))).(*GenericTriggered)), true},
		{"nil body", GrantTriggeredAbility(This, At(YourUpkeep, nil).(*GenericTriggered)), true},
		{"legacy event", GrantTriggeredAbility(This, NewTriggered(core.EvtUpkeep, false, GainLifeFor(You, Fixed(1)))), true},
		{"graveyard event", GrantTriggeredAbility(This, At(YourUpkeep, GainLifeFor(You, Fixed(1))).(*GenericTriggered).FromGraveyard()), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := NewAura("Grant", "", WithAbility(Static(tc.rule)))
			if err := ValidateCardDefinition(c.Name(), c); (err != nil) != tc.bad {
				t.Fatalf("validation: %v", err)
			}
		})
	}
}

func TestGrantedTriggerIdentityAndMembership(t *testing.T) {
	for _, change := range []string{"control", "phase", "phase cycle", "recipient return", "donor return", "donor phase", "donor removal", "recipient removal", "second donor"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			prototype := WhenEvent(BecomesTapped(This), GainLifeFor(You, Fixed(1))).(*GenericTriggered)
			donorCard := NewEnchantment("Donor", "", WithAbility(Static(GrantEachTriggeredAbility(ControlledPermanents(IsCreature), prototype))))
			donorCard.SetOwner(a.PlayerID())
			donor := g.PutOnBattlefield(donorCard, a.PlayerID())
			card := NewCreature("Recipient", "", 2, 3)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			id := p.RuntimeAbilities[len(p.RuntimeAbilities)-1].AbilityID()
			clone := g.Clone()
			want := 1
			switch change {
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: donor.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				want = 0
			case "phase":
				g.PhaseOut(p.ID())
				want = 0
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "recipient return":
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(card.ID())
				p = g.PutOnBattlefield(card, a.PlayerID())
			case "donor return":
				g.BouncePermanentToHand(donor)
				a.RemoveFromHand(donorCard.ID())
				g.PutOnBattlefield(donorCard, a.PlayerID())
			case "donor phase":
				g.PhaseOut(donor.ID())
				want = 0
			case "donor removal":
				g.AddContinuousEffect(RemoveAllAbilities(donor.ID(), core.Indefinite))
				want = 0
			case "recipient removal":
				g.AddContinuousEffect(RemoveAllAbilities(p.ID(), core.Indefinite))
				want = 0
			case "second donor":
				second := NewEnchantment("Second", "", WithAbility(Static(GrantEachTriggeredAbility(ControlledPermanents(IsCreature), prototype))))
				g.PutOnBattlefield(second, a.PlayerID())
				want = 2
			}
			g.ApplyContinuousEffects()
			if change == "recipient return" || change == "donor return" {
				if id == g.FindPermanent(p.ID()).RuntimeAbilities[len(g.FindPermanent(p.ID()).RuntimeAbilities)-1].AbilityID() {
					t.Fatal("new incarnation reused grant identity")
				}
			}
			if change == "phase cycle" && id != g.FindPermanent(p.ID()).RuntimeAbilities[len(g.FindPermanent(p.ID()).RuntimeAbilities)-1].AbilityID() {
				t.Fatal("phasing changed grant identity")
			}
			g.TapPermanent(g.FindPermanent(p.ID()))
			g.PutTriggersOnStack()
			if g.StackSize() != want {
				t.Fatalf("triggers=%d want %d", g.StackSize(), want)
			}
			g.ResolveStack()
			if a.Life() != 20+want || b.Life() != 20 {
				t.Fatal("recipient event context")
			}
			clone.TapPermanent(clone.FindPermanent(p.ID()))
			clone.PutTriggersOnStack()
			clone.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != 21 {
				t.Fatal("clone grant was changed")
			}
		})
	}
}

func TestGrantedTriggerRecipientContext(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player := BindPlayer("upkeep player")
	prototype := WhenEvent(TurnStepBegins(YourUpkeep, player), GainLifeFor(FromPlayer(player), Fixed(1))).(*GenericTriggered)
	donorCard := NewEnchantment("Donor", "", WithAbility(Static(GrantEachTriggeredAbility(MatchingPermanents(IsCreature), prototype))))
	donorCard.SetOwner(a.PlayerID())
	donor := g.PutOnBattlefield(donorCard, a.PlayerID())
	one := g.PutOnBattlefield(NewCreature("One", "", 2, 3), a.PlayerID())
	two := g.PutOnBattlefield(NewCreature("Two", "", 2, 3), b.PlayerID())
	grantID := func(p *Permanent) uuid.UUID {
		for _, ability := range p.RuntimeAbilities {
			if _, ok := UnwrapAbility(ability).(*GenericTriggered); ok {
				return ability.AbilityID()
			}
		}
		t.Fatal("missing trigger")
		return uuid.Nil
	}
	id := grantID(one)
	if id == grantID(two) || prototype.Source() != uuid.Nil {
		t.Fatal("recipient ability identity is shared")
	}
	g.ApplyContinuousEffects()
	if id != grantID(g.FindPermanent(one.ID())) {
		t.Fatal("layer pass changed grant identity")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || g.StackPeek().SourceID != two.ID() {
		t.Fatal("trigger used donor context")
	}
	g.DestroyPermanent(donor)
	clone := g.Clone()
	clone.ResolveStack()
	if g.StackSize() != 1 || b.Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 21 {
		t.Fatal("pending clone did not keep its own frame")
	}
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 21 {
		t.Fatal("wrong event binding or controller")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("departed donor still grants")
	}
}

func TestGrantedTriggerTargetsAndInterveningIf(t *testing.T) {
	a, b := newRecordingPlayer("A"), newRecordingPlayer("B")
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewArtifact("First", ""), a.PlayerID())
	second := g.PutOnBattlefield(NewArtifact("Second", ""), a.PlayerID())
	x := g.PutOnBattlefield(NewCreature("X", "", 2, 3), b.PlayerID())
	y := g.PutOnBattlefield(NewCreature("Y", "", 2, 3), b.PlayerID())
	target := DeclarePermanentTarget("grant target", IsCreature)
	prototype := At(YourUpkeep, TapSubject(target), WithTargets(target), WithInterveningIf(PermanentIsTapped(This))).(*GenericTriggered)
	card := NewEnchantment("Donor", "", WithAbility(Static(GrantEachTriggeredAbility(MatchingPermanents(IsArtifact), prototype))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(card, b.PlayerID())
	g.TapPermanent(first)
	g.TapPermanent(second)
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
	if len(a.calls) != 0 {
		t.Fatal("targets chosen before placement")
	}
	a.chooseQueue = [][]uuid.UUID{{x.ID()}, {y.ID()}}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 || len(a.calls) != 2 || len(b.calls) != 0 {
		t.Fatal("recipient target declarations")
	}
	clone := g.Clone()
	g.UntapPermanent(g.FindPermanent(g.StackPeek().SourceID))
	g.ResolveTopOfStack()
	if g.FindPermanent(y.ID()).Tapped || g.FindPermanent(x.ID()).Tapped {
		t.Fatal("intervening-if used donor or other recipient")
	}
	g.ResolveStack()
	if !g.FindPermanent(x.ID()).Tapped || g.FindPermanent(y.ID()).Tapped {
		t.Fatal("independent target frames")
	}
	clone.ResolveStack()
	if !clone.FindPermanent(x.ID()).Tapped || !clone.FindPermanent(y.ID()).Tapped || g.FindPermanent(y.ID()).Tapped {
		t.Fatal("clone targets or conditions leaked")
	}
}
