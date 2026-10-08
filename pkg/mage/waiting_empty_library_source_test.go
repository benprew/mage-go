package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingEmptyLibrarySourcesLifecycle(t *testing.T) {
	for _, state := range []string{"live", "phase", "cycle", "depart", "return", "control", "remove-ability", "nonempty"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewArtifact("Replacement", "{R}")
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			calls := 0
			g.AddEmptyLibraryDrawReplacement(p.ID(), a.PlayerID(), func(*Game, uuid.UUID) { calls++ })
			clone := g.Clone()
			want := 1
			switch state {
			case "phase":
				g.PhaseOut(p.ID())
				want = 0
			case "cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "depart", "return":
				card := p.Card
				g.RemoveFromBattlefield(p)
				g.zones.ClearLKI()
				want = 0
				if state == "return" {
					g.PutOnBattlefield(card, a.PlayerID())
				}
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "remove-ability":
				g.AddContinuousEffect(RemoveAllAbilities(p.ID(), core.EndOfTurn))
			case "nonempty":
				a.AddToLibrary(NewArtifact("Library", ""))
			}
			refs := g.WaitingEmptyLibraryPermanentSources()
			if len(refs) != want || calls != 0 {
				t.Fatalf("query sources=%d want=%d calls=%d", len(refs), want, calls)
			}
			if want == 1 {
				if !refs[0].SameObject(saved) {
					t.Fatal("wrong source identity")
				}
				if state == "control" && refs[0].Controller() != b.PlayerID() {
					t.Fatal("stale controller")
				}
				refs[0].card.SetOwner(b.PlayerID())
				if g.WaitingEmptyLibraryPermanentSources()[0].card.Owner() != a.PlayerID() {
					t.Fatal("query changed source")
				}
			}
			action := NewDrawCardAction(uuid.Nil, a.PlayerID(), false)
			replaced := g.effects.ApplyReplacements(action, g) == nil
			if replaced != (want == 1 && state != "nonempty") {
				t.Fatal("execution and source lifecycle disagree")
			}
			if g.effects.ApplyReplacements(NewDrawCardAction(uuid.Nil, b.PlayerID(), true), g) == nil {
				t.Fatal("replaced wrong player's draw")
			}
			if len(clone.WaitingEmptyLibraryPermanentSources()) != 1 {
				t.Fatal("clone lost registration")
			}
			if clone.effects.ApplyReplacements(action, clone) != nil {
				t.Fatal("clone lost replacement execution")
			}
			if state == "return" {
				g.AddEmptyLibraryDrawReplacement(p.ID(), a.PlayerID(), func(*Game, uuid.UUID) { calls++ })
				refs = g.WaitingEmptyLibraryPermanentSources()
				if len(refs) != 1 || refs[0].SameObject(saved) {
					t.Fatal("fresh registration lost returned identity")
				}
			}
		})
	}
}

func TestWaitingEmptyLibrarySourcesChoices(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewArtifact("Replacement", "{R}"), a.PlayerID())
	callback := func(*Game, uuid.UUID) { t.Fatal("query ran callback") }
	for range 2 {
		g.AddEmptyLibraryDrawReplacement(p.ID(), a.PlayerID(), callback)
	}
	g.AddEmptyLibraryDrawReplacement(uuid.New(), a.PlayerID(), callback)
	g.AddEmptyLibraryDrawReplacement(p.ID(), a.PlayerID(), nil)
	if len(g.WaitingEmptyLibraryPermanentSources()) != 1 || len(g.effects.replacements) != 4 {
		t.Fatal("query did not deduplicate or changed storage")
	}
	first, second := BindDamageSource("first"), BindDamageSource("second")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	for _, binding := range []*DamageSourceBinding{first, second} {
		if err := ChooseDamageSourceFromWaitingEmptyLibraryReferences(You, DamageSourceHasType(core.TypeArtifact), binding).Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if !g.resolution.damageSourceBindings[binding].SameObject(g.CaptureDamageSource(p.ID())) {
			t.Fatal("choice identity")
		}
	}
	if err := ChooseDamageSourceFromWaitingEmptyLibraryReferences(You, DamageSourceHasColor(core.Blue), first).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.resolution.damageSourceBindings[first].Valid() || !g.resolution.damageSourceBindings[second].Valid() {
		t.Fatal("result isolation")
	}
	if err := ChooseDamageSourceFromWaitingEmptyLibraryReferences(ControllerOf(From(BindPermanent("absent"))), DamageSourceFilter{}, second).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.resolution.damageSourceBindings[second].Valid() {
		t.Fatal("absent player kept result")
	}
	for _, effect := range []ResolvingEffect{
		ChooseDamageSourceFromWaitingEmptyLibraryReferences(nil, DamageSourceFilter{}, first),
		ChooseDamageSourceFromWaitingEmptyLibraryReferences(You, DamageSourceFilter{}, nil),
		ChooseDamageSourceFromWaitingEmptyLibraryReferences(You, DamageSourceFilter{}, BindDamageSource("")),
		Sequence(DealDamageFrom(FromDamageSource(first), You, Fixed(1)), ChooseDamageSourceFromWaitingEmptyLibraryReferences(You, DamageSourceFilter{}, first)),
	} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))) == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestWaitingEmptyLibrarySourcesSavedRegistrations(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewArtifact("Absent at registration", "")
	g.AddEmptyLibraryDrawReplacement(card.ID(), a.PlayerID(), func(*Game, uuid.UUID) { t.Fatal("absent registration became active") })
	p := g.PutOnBattlefield(card, a.PlayerID())
	action := NewDrawCardAction(uuid.Nil, a.PlayerID(), false)
	if len(g.WaitingEmptyLibraryPermanentSources()) != 0 || g.effects.ApplyReplacements(action, g) != action {
		t.Fatal("later entry inherited registration")
	}
	g.AddEmptyLibraryDrawReplacement(p.ID(), a.PlayerID(), func(*Game, uuid.UUID) { t.Fatal("stale registration became active") })
	saved := g.effects.replacements[len(g.effects.replacements)-1].Clone()
	g.RemoveFromBattlefield(p)
	g.PutOnBattlefield(card, a.PlayerID())
	g.effects.AddReplacement(saved)
	if len(g.WaitingEmptyLibraryPermanentSources()) != 0 || g.effects.ApplyReplacements(action, g) != action {
		t.Fatal("returned object inherited saved registration")
	}
}

func TestWaitingEmptyLibrarySourcesReplacementOrderAndClones(t *testing.T) {
	a := &choiceRecordingPlayer{BasePlayer: NewBasePlayer("A"), chosenMode: 1}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	for _, amount := range []int{2, 5} {
		p := g.PutOnBattlefield(NewArtifact("Replacement", ""), b.PlayerID())
		g.AddEmptyLibraryDrawReplacement(p.ID(), a.PlayerID(), func(game *Game, sourceID uuid.UUID) {
			game.MutablePermanent(sourceID).AddCounter(core.P1P1, amount)
		})
	}
	refs := g.WaitingEmptyLibraryPermanentSources()
	clone := g.Clone()
	action := NewDrawCardAction(uuid.Nil, a.PlayerID(), false)
	if clone.effects.ApplyReplacements(action, clone) != nil {
		t.Fatal("draw was not replaced")
	}
	if clone.FindPermanent(refs[0].ID()).Counters[core.P1P1] != 2 || clone.FindPermanent(refs[1].ID()).Counters[core.P1P1] != 0 {
		t.Fatal("clone did not execute its own replacement choice")
	}
	if g.FindPermanent(refs[1].ID()).Counters[core.P1P1] != 0 {
		t.Fatal("clone callback changed original")
	}
	if len(clone.WaitingEmptyLibraryPermanentSources()) != 2 {
		t.Fatal("draw consumed persistent replacement")
	}
	if g.effects.ApplyReplacements(action, g) != nil || g.FindPermanent(refs[1].ID()).Counters[core.P1P1] != 5 {
		t.Fatal("original lost independent replacement")
	}
}
