package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func retainedEntryRule() Ability {
	return PutThenSacrificeFor(OpponentsOf(You), IsLand, IsLand, CountPermanents(PermanentsControlledBy(You, IsLand)))
}

type retainedEntryPlayer struct {
	*BasePlayer
	chooseLast bool
	choices    int
	before     func()
}

func (p *retainedEntryPlayer) ChoosePermanent(candidates []*Permanent, _ string, _ GameReader) *Permanent {
	p.choices++
	if p.before != nil {
		p.before()
	}
	if p.chooseLast {
		return candidates[len(candidates)-1]
	}
	return candidates[0]
}

func TestRetainedEntryCountsAndActor(t *testing.T) {
	for _, n := range []int{0, 1, 2} {
		t.Run(string(rune('0'+n)), func(t *testing.T) {
			a := NewBasePlayer("A")
			b := &retainedEntryPlayer{BasePlayer: NewBasePlayer("B"), chooseLast: true}
			g := NewGame(a, b)
			for range n {
				g.PutOnBattlefield(NewLand("Own land"), a.PlayerID())
			}
			g.PutOnBattlefield(NewEnchantment("Rule", "", WithAbility(retainedEntryRule())), a.PlayerID())
			entrant := g.PutOnBattlefield(NewLand("Entrant"), b.PlayerID())
			if b.choices != 0 || (g.FindPermanent(entrant.ID()) == nil) != (n == 0) {
				t.Fatal("pre-entry count", n, b.choices)
			}
			if n > 0 {
				own := g.PutOnBattlefieldBy(NewLand("Other controller"), a.PlayerID(), b.PlayerID())
				if g.FindPermanent(own.ID()) == nil {
					t.Fatal("sacrificed another controller's entrant")
				}
			}
		})
	}
}

func TestRetainedEntryLifecycleAndSavedSequence(t *testing.T) {
	a := NewBasePlayer("A")
	b := &retainedEntryPlayer{BasePlayer: NewBasePlayer("B"), chooseLast: true}
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewEnchantment("Rule", "", WithAbility(retainedEntryRule())), a.PlayerID())
	clone := g.Clone()
	phased := g.PhaseOut(source.ID())
	first := g.PutOnBattlefield(NewLand("Phased"), b.PlayerID())
	if g.FindPermanent(first.ID()) == nil {
		t.Fatal("phased rule applied")
	}
	g.PhaseIn(phased)
	g.AddContinuousEffect(RemoveAllAbilities(source.ID(), core.EndOfTurn))
	second := g.PutOnBattlefield(NewLand("No ability"), b.PlayerID())
	if g.FindPermanent(second.ID()) == nil {
		t.Fatal("removed ability applied")
	}
	g.effects.RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	b.before = func() { g.RemoveFromBattlefield(g.FindPermanent(source.ID())) }
	third := g.PutOnBattlefield(NewLand("Saved"), b.PlayerID())
	if g.FindPermanent(third.ID()) != nil {
		t.Fatal("source departure erased applied sequence")
	}
	b.before = nil
	fourth := g.PutOnBattlefield(NewLand("Absent"), b.PlayerID())
	if g.FindPermanent(fourth.ID()) == nil {
		t.Fatal("departed rule applied")
	}
	independent := clone.PutOnBattlefield(NewLand("Clone"), b.PlayerID())
	if clone.FindPermanent(independent.ID()) != nil {
		t.Fatal("clone lost rule")
	}
}

func TestRetainedEntryBatchAndEvents(t *testing.T) {
	a := NewBasePlayer("A")
	b := &retainedEntryPlayer{BasePlayer: NewBasePlayer("B"), chooseLast: true}
	g := NewGame(a, b)
	g.PutOnBattlefield(NewEnchantment("Rule", "", WithAbility(retainedEntryRule())), a.PlayerID())
	var events []core.EventType
	g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(NewTriggered(core.EvtZoneChange, false).SetCondition(func(e *core.GameEvent, r GameReader, _, _ uuid.UUID) bool {
		if e.ToZone == core.ZoneBattlefield && e.PlayerID == b.PlayerID() {
			if len(r.(*Game).FilterBattlefield(IsLand)) != 2 {
				t.Fatal("entry batch was not complete")
			}
			events = append(events, e.Type)
		}
		return false
	}))), a.PlayerID())
	g.putTokensOnBattlefieldTogether([]Card{NewLand("One"), NewLand("Two")}, b.PlayerID())
	if len(events) != 2 || len(g.FilterBattlefield(IsLand)) != 0 || b.choices != 1 {
		t.Fatal("batch follow-ups", events, b.choices)
	}
}

func TestRetainedEntryDefinition(t *testing.T) {
	for _, card := range []Card{
		NewInstant("Wrong zone", "", WithAbility(retainedEntryRule())),
		NewArtifact("Nil players", "", WithAbility(PutThenSacrificeFor(nil, IsLand, IsLand, Fixed(0)))),
		NewArtifact("Nil count", "", WithAbility(PutThenSacrificeFor(EachPlayer, IsLand, IsLand, nil))),
	} {
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid declaration")
		}
	}
}

func TestRetainedEntryActorAndVisit(t *testing.T) {
	a := NewBasePlayer("A")
	b := &retainedEntryPlayer{BasePlayer: NewBasePlayer("B")}
	g := NewGame(a, b)
	old := g.PutOnBattlefield(NewLand("Old"), b.PlayerID())
	spare := g.PutOnBattlefield(NewLand("Spare"), b.PlayerID())
	g.PutOnBattlefield(NewEnchantment("Rule", "", WithAbility(retainedEntryRule())), a.PlayerID())
	b.before = func() {
		b.before = nil
		g.RemoveFromBattlefield(g.FindPermanent(old.ID()))
		g.PutOnBattlefieldBy(old.Card, b.PlayerID(), a.PlayerID())
	}
	entrant := g.PutOnBattlefieldBy(NewLand("Other controller"), a.PlayerID(), b.PlayerID())
	if b.choices != 1 || g.FindPermanent(entrant.ID()) == nil || g.FindPermanent(spare.ID()) == nil || g.FindPermanent(old.ID()).incarnationID == old.incarnationID || len(b.Graveyard()) != 0 {
		t.Fatal("actor or old choice visit")
	}
	own := g.PutOnBattlefieldBy(NewLand("Own actor"), b.PlayerID(), a.PlayerID())
	if g.FindPermanent(own.ID()) == nil || b.choices != 1 {
		t.Fatal("entry controller replaced actor")
	}
}

func TestRetainedEntryCancellationAndRecheck(t *testing.T) {
	for _, cancel := range []bool{false, true} {
		t.Run(map[bool]string{false: "recheck", true: "cancel"}[cancel], func(t *testing.T) {
			a := NewBasePlayer("A")
			b := &filteredEntryChooser{BasePlayer: NewBasePlayer("B"), choice: 1}
			g := NewGame(a, b)
			source := g.PutOnBattlefield(NewEnchantment("Rule", "", WithAbility(retainedEntryRule())), a.PlayerID())
			g.effects.AddReplacement(&testPrioritizedReplacement{replacementBase: replacementBase{priority: ReplacementPriorityDefault}, matchFunc: func(action Action) bool { _, ok := action.(*battlefieldEntryAction); return ok }, replaceFunc: func(action Action, _ *Game) Action {
				if cancel {
					return nil
				}
				return action
			}})
			if !cancel {
				b.before = func() { g.RemoveFromBattlefield(g.FindPermanent(source.ID())) }
			}
			entrant := g.PutOnBattlefield(NewLand("Entrant"), b.PlayerID())
			if cancel {
				if entrant != nil || len(b.Graveyard()) != 1 {
					t.Fatal("cancelled entry")
				}
			} else if entrant == nil || g.FindPermanent(entrant.ID()) == nil || len(b.Graveyard()) != 0 {
				t.Fatal("departed replacement applied")
			}
		})
	}
}

func TestRetainedEntryCopyPriorityAndNestedOrder(t *testing.T) {
	a := &optionalEntryPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	g.PutOnBattlefield(NewEnchantment("Rule", "", WithAbility(PutThenSacrificeFor(EachPlayer, IsLand, IsLand, Fixed(0)))), b.PlayerID())
	donorPlan, err := NewCardDefinitionPlan(func() Card { return NewArtifact("Donor", "") })
	if err != nil {
		t.Fatal(err)
	}
	donor, err := donorPlan.Build()
	if err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(donor, b.PlayerID())
	physicalPlan, err := NewCardDefinitionPlan(func() Card { return NewLand("Physical", WithAbility(MayCopyAsEnters(IsArtifact))) })
	if err != nil {
		t.Fatal(err)
	}
	physical, err := physicalPlan.Build()
	if err != nil {
		t.Fatal(err)
	}
	copied := g.PutOnBattlefield(physical, a.PlayerID())
	if copied == nil || !copied.HasType(core.TypeArtifact) || g.FindPermanent(copied.ID()) == nil {
		t.Fatal("count matched before copy")
	}

	h, x, y := newSpellCopyGame()
	h.PutOnBattlefield(NewEnchantment("First", "", WithAbility(PutThenSacrificeFor(EachPlayer, IsLand, IsArtifact, Fixed(0)))), x.PlayerID())
	h.PutOnBattlefield(NewEnchantment("Second", "", WithAbility(PutThenSacrificeFor(EachPlayer, IsLand, IsLand, Fixed(0)))), x.PlayerID())
	var order []string
	h.PutOnBattlefield(NewArtifact("Artifact", ""), y.PlayerID())
	h.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(NewTriggered(core.EvtSacrifice, false).SetCondition(func(e *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		order = append(order, e.SourceID.String())
		return false
	}))), x.PlayerID())
	entrant := h.PutOnBattlefield(NewLand("Nested"), y.PlayerID())
	if len(order) != 2 || order[0] != entrant.ID().String() {
		t.Fatal("inner retained entry did not finish first", order)
	}
}

func TestRetainedEntryPendingCloneAndReturn(t *testing.T) {
	g, a, b := newSpellCopyGame()
	card := NewEnchantment("Rule", "", WithAbility(retainedEntryRule()))
	source := g.PutOnBattlefield(card, a.PlayerID())
	entrant := g.prepareBattlefieldEntry(NewLand("Pending"), b.PlayerID(), nil, 0, false)
	g.registerBattlefieldEntry(entrant, nil, uuid.Nil)
	clone := g.Clone()
	g.finishRetainedEntry(entrant)
	if g.FindPermanent(entrant.ID()) != nil || clone.FindPermanent(entrant.ID()) == nil {
		t.Fatal("pending clone isolation")
	}
	clone.finishRetainedEntry(clone.MutablePermanent(entrant.ID()))
	if clone.FindPermanent(entrant.ID()) != nil {
		t.Fatal("pending clone lost follow-up")
	}
	g.RemoveFromBattlefield(source)
	returned := g.PutOnBattlefield(card, b.PlayerID())
	if returned.incarnationID == source.incarnationID {
		t.Fatal("source return reused visit")
	}
	own := g.PutOnBattlefield(NewLand("Ally"), b.PlayerID())
	if g.FindPermanent(own.ID()) == nil {
		t.Fatal("returned rule kept old controller")
	}
	g.PutOnBattlefield(NewLand("Opponent first"), a.PlayerID())
	g.PutOnBattlefield(NewLand("Opponent second"), a.PlayerID())
	if len(g.FilterBattlefield(And(IsLand, ControlledBy(a.PlayerID())))) != 1 {
		t.Fatal("returned rule lost new controller")
	}
}

func TestRetainedEntryBatchUsesOriginalCount(t *testing.T) {
	g, a, b := newSpellCopyGame()
	g.PutOnBattlefield(NewLand("Threshold"), a.PlayerID())
	g.PutOnBattlefield(NewEnchantment("Rule", "", WithAbility(retainedEntryRule())), a.PlayerID())
	entrants := g.putTokensOnBattlefieldTogether([]Card{NewLand("One"), NewLand("Two")}, b.PlayerID())
	if len(entrants) != 2 || len(g.FilterBattlefield(And(IsLand, ControlledBy(b.PlayerID())))) != 2 || len(b.Graveyard()) != 0 {
		t.Fatal("one entrant changed another's pre-entry count")
	}
}

func TestRetainedEntryPhasedEntrantKeepsSequence(t *testing.T) {
	g, a, b := newSpellCopyGame()
	old := g.PutOnBattlefield(NewLand("Old"), b.PlayerID())
	g.PutOnBattlefield(NewEnchantment("Rule", "", WithAbility(retainedEntryRule())), a.PlayerID())
	entrant := g.prepareBattlefieldEntry(NewLand("Pending"), b.PlayerID(), nil, 0, false)
	g.registerBattlefieldEntry(entrant, nil, uuid.Nil)
	clone := g.Clone()
	phased := g.PhaseOut(entrant.ID())
	g.finishRetainedEntry(entrant)
	if g.FindPermanent(old.ID()) != nil {
		t.Fatal("phasing cancelled independent sacrifice")
	}
	g.PhaseIn(phased)
	if g.FindPermanent(entrant.ID()) == nil || len(g.FindPermanent(entrant.ID()).entryFollowUps) != 0 {
		t.Fatal("phased continuation not consumed")
	}
	clone.finishRetainedEntry(clone.MutablePermanent(entrant.ID()))
	if len(clone.FilterBattlefield(IsLand)) != 1 {
		t.Fatal("phased entrant changed clone sequence")
	}
}

func TestRetainedEntryReturnActors(t *testing.T) {
	for _, path := range []string{"graveyard", "graveyard-batch", "exile-batch"} {
		t.Run(path, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			old := g.PutOnBattlefield(NewLand("Actor's land"), b.PlayerID())
			g.PutOnBattlefield(NewEnchantment("Rule", "", WithAbility(retainedEntryRule())), a.PlayerID())
			card := NewCreature("Returning land", "", 2, 2, WithCardType(core.TypeLand))
			card.SetOwner(a.PlayerID())
			switch path {
			case "graveyard":
				a.AddToGraveyard(card)
				if returnGraveyardCardToBattlefieldWithActor(g, card.ID(), a.PlayerID(), b.PlayerID(), false) == nil {
					t.Fatal("return")
				}
			case "graveyard-batch":
				a.AddToGraveyard(card)
				if err := ApplyEffect(g, PutGraveyardCreaturesOntoBattlefieldUnderOwnersControl(), uuid.Nil, b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "exile-batch":
				g.ExileCard(card, uuid.Nil)
				subjects := savedTransportVisits{{id: card.ID(), visit: g.FindExiledCard(card.ID()).Visit, zone: core.ZoneExile}}
				if err := ApplyEffect(g, PutExiledCardsOntoBattlefieldUnderOwnersControl(subjects), uuid.Nil, b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			p := g.FindPermanent(card.ID())
			if p == nil || p.ControllerID() != a.PlayerID() || g.FindPermanent(old.ID()) != nil || len(b.Graveyard()) != 1 {
				t.Fatal("return used owner instead of actor")
			}
		})
	}
}
