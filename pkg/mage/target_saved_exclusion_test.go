package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestTargetSavedExclusionDefinition(t *testing.T) {
	for _, valid := range []bool{false, true} {
		saved := BindPermanent("host")
		target := DeclarePermanentTarget("other", IsCreature, TargetOtherThanSavedPermanent(saved))
		var c Card = NewInstant("Bad", "", Spell(TapSubject(target), WithTargets(target)))
		if valid {
			c = NewAura("Valid", "", WithAbility(WhenEvent(AttachedAttacks(saved), TapSubject(target), WithTargets(target))))
		}
		if err := ValidateCardDefinition(c.Name(), c); (err == nil) != valid {
			t.Fatalf("valid=%t: %v", valid, err)
		}
	}
	target := DeclarePermanentTarget("other", IsCreature, TargetOtherThanSavedPermanent(nil))
	if err := ValidateCardDefinition("Bad", NewInstant("Bad", "", Spell(TapSubject(target), WithTargets(target)))); err == nil {
		t.Fatal("nil binding accepted")
	}
}

func TestTargetSavedExclusionIdentity(t *testing.T) {
	g, a, b := newSpellCopyGame()
	saved := BindPermanent("event")
	target := DeclarePermanentTarget("other", IsCreature, TargetOtherThanSavedPermanent(saved))
	firstCard := NewCreature("First", "", 2, 8)
	first := g.PutOnBattlefield(firstCard, a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 2, 8), b.PlayerID())
	source := g.PutOnBattlefield(NewAura("Watcher", "", WithAbility(WhenEvent(AttachedAttacks(saved), TapSubject(target), WithTargets(target)))), a.PlayerID())
	g.Attach(source.ID(), first.ID())
	g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{saved: {second.ID(), second.incarnationID}}
	a.chooseQueue = [][]uuid.UUID{{first.ID()}, {second.ID()}}
	g.FireEvent(GameEvent{Type: EvtDeclaredAttacker, SourceID: first.ID(), PlayerID: a.PlayerID()})
	g.Attach(source.ID(), second.ID())
	g.FireEvent(GameEvent{Type: EvtDeclaredAttacker, SourceID: second.ID(), PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 || len(a.chooseQueue) != 0 {
		t.Fatalf("pending events used the live binding or current attachment: stack=%d queue=%v", g.StackSize(), a.chooseQueue)
	}
	if g.resolution.permanentBindings[saved].id != second.ID() {
		t.Fatal("target query changed the live frame")
	}
	clone := g.Clone()
	g.ResolveStack()
	if clone.FindPermanent(first.ID()).Tapped || clone.FindPermanent(second.ID()).Tapped {
		t.Fatal("resolution changed clone")
	}
	clone.ResolveStack()
	for _, game := range []*Game{g, clone} {
		if !game.FindPermanent(first.ID()).Tapped || !game.FindPermanent(second.ID()).Tapped {
			t.Fatal("pending event exclusions crossed")
		}
	}
	query := g.targetEventQuery(nil, map[*PermanentBinding]capturedPermanent{saved: {first.ID(), first.incarnationID}})
	if slices.Contains(target.Possible(a.PlayerID(), source.Card, query), first.ID()) {
		t.Fatal("saved visit remained legal")
	}
	g.PhaseIn(g.PhaseOut(first.ID()))
	if slices.Contains(target.Possible(a.PlayerID(), source.Card, query), first.ID()) {
		t.Fatal("phase cycle lost exclusion")
	}
	g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
	first = g.PutOnBattlefield(firstCard, a.PlayerID())
	if !slices.Contains(target.Possible(a.PlayerID(), source.Card, query), first.ID()) {
		t.Fatal("new visit inherited exclusion")
	}
	if len(target.Possible(a.PlayerID(), source.Card, g.targetEventQuery(nil, nil))) != 0 {
		t.Fatal("missing event binding supplied targets")
	}
}

func TestTargetSavedExclusionCopyAndResolution(t *testing.T) {
	for _, scenario := range []string{"copy", "invalid-copy", "phase", "return", "shroud", "source-left", "host-phase", "host-return"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			saved := BindPermanent("host")
			target := DeclarePermanentTarget("other", IsCreature, TargetOtherThanSavedPermanent(saved))
			hostCard := NewCreature("Host", "", 2, 8)
			host := g.PutOnBattlefield(hostCard, a.PlayerID())
			victimCard := NewCreature("Victim", "", 2, 8)
			victim := g.PutOnBattlefield(victimCard, b.PlayerID())
			spare := g.PutOnBattlefield(NewCreature("Spare", "", 2, 8), b.PlayerID())
			source := g.PutOnBattlefield(NewAura("Watcher", "", WithAbility(WhenEvent(AttachedAttacks(saved), Sequence(TapSubject(target), GainLifeFor(You, Fixed(1))), WithTargets(target)))), a.PlayerID())
			g.Attach(source.ID(), host.ID())
			a.chooseQueue = [][]uuid.UUID{{victim.ID()}}
			g.FireEvent(GameEvent{Type: EvtDeclaredAttacker, SourceID: host.ID(), PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("trigger failed to choose other creature")
			}
			legal := true
			switch scenario {
			case "copy", "invalid-copy":
				choice := spare.ID()
				if scenario == "invalid-copy" {
					choice = host.ID()
				}
				b.chooseQueue = [][]uuid.UUID{{choice}}
				copied := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), true)
				want := spare.ID()
				if scenario == "invalid-copy" {
					want = victim.ID()
				}
				if copied == nil || !slices.Equal(copied.Targets, []uuid.UUID{want}) {
					t.Fatal("copy lost event exclusion")
				}
			case "phase":
				g.PhaseOut(victim.ID())
				legal = false
			case "return":
				g.RemoveFromBattlefield(victim)
				g.PutOnBattlefield(victimCard, b.PlayerID())
				legal = false
			case "shroud":
				g.MutablePermanent(victim.ID()).GrantBaseAttr(Shroud)
				legal = false
			case "source-left":
				g.RemoveFromBattlefield(source)
			case "host-phase":
				g.PhaseOut(host.ID())
			case "host-return":
				g.RemoveFromBattlefield(host)
				g.PutOnBattlefield(hostCard, a.PlayerID())
			}
			g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{saved: {victim.ID(), victim.incarnationID}}
			clone := g.Clone()
			g.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("clone shared resolution")
			}
			clone.ResolveStack()
			for _, game := range []*Game{g, clone} {
				want := 20
				if legal {
					want++
				}
				if game.GetPlayer(a.PlayerID()).Life() != want {
					t.Fatal("resolution lost event target legality")
				}
				if scenario == "copy" && !game.FindPermanent(spare.ID()).Tapped {
					t.Fatal("copy did not use new target")
				}
				if (scenario == "copy" || scenario == "invalid-copy") && game.GetPlayer(b.PlayerID()).Life() != 21 {
					t.Fatal("copy controller changed")
				}
			}
		})
	}
}

func TestTargetSavedExclusionModeAvailability(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	saved := BindPermanent("host")
	target := DeclarePermanentTarget("other", IsCreature, TargetOtherThanSavedPermanent(saved))
	card := NewAura("Watcher", "", WithAbility(WhenEventModes(AttachedAttacks(saved),
		ModeOf("Other", TapSubject(target), WithTargets(target)),
		ModeOf("Life", GainLifeFor(You, Fixed(2))))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	host := g.PutOnBattlefield(NewCreature("Only creature", "", 2, 8), a.PlayerID())
	aura := g.PutOnBattlefield(card, a.PlayerID())
	g.Attach(aura.ID(), host.ID())
	g.FireEvent(GameEvent{Type: EvtDeclaredAttacker, SourceID: host.ID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || len(g.StackPeek().Targets) != 0 {
		t.Fatal("unavailable mode used the excluded subject")
	}
	g.ResolveStack()
	if a.Life() != 22 || g.FindPermanent(host.ID()).Tapped {
		t.Fatal("mode availability lost the captured event")
	}
}
