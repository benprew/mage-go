package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDiscardSourcesLifecycle(t *testing.T) {
	for _, state := range []string{"live", "phase", "cycle", "depart", "return", "control", "remove-ability"} {
		t.Run(state, func(t *testing.T) {
			a, b := &waitingDiscardPlayer{BasePlayer: NewBasePlayer("A"), accept: true}, &waitingDiscardPlayer{BasePlayer: NewBasePlayer("B"), accept: true}
			g := NewGame(a, b)
			card := NewArtifact("Discard rule", "{R}", WithStaticAbility(DiscardToLibraryReplacement()))
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			var old ReplacementEffect
			for _, replacement := range g.effects.cycleReplacements {
				if _, ok := replacement.(*discardToLibraryReplacement); ok {
					old = replacement.Clone()
				}
			}
			if old == nil {
				t.Fatal("missing static registration")
			}
			clone := g.Clone()
			want := 1
			switch state {
			case "phase":
				g.PhaseOut(p.ID())
				want = 0
			case "cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "depart", "return":
				g.RemoveFromBattlefield(p)
				g.zones.ClearLKI()
				want = 0
				if state == "return" {
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "remove-ability":
				if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				want = 0
			}
			g.effects.replacements = []ReplacementEffect{old, old.Clone()}
			g.effects.cycleReplacements = nil
			refs := g.WaitingDiscardPermanentSources()
			if len(refs) != want || old.IsActive(g) != (want == 1) {
				t.Fatalf("query/execution lifecycle: sources=%d want=%d", len(refs), want)
			}
			payer := a.PlayerID()
			if state == "control" {
				payer = b.PlayerID()
			}
			action := NewDiscardAction(uuid.Nil, payer, uuid.New())
			if a.calls != 0 || b.calls != 0 {
				t.Fatal("query made a payment choice")
			}
			if old.Matches(action, g) != (want == 1) {
				t.Fatal("query differs from matching")
			}
			result := g.effects.ApplyReplacements(action, g).(*DiscardAction)
			wantDestination := core.ZoneGraveyard
			if want == 1 {
				wantDestination = core.ZoneLibrary
			}
			if result.Destination() != wantDestination || a.calls+b.calls != want {
				t.Fatal("query differs from replacement execution")
			}
			if state == "control" && (a.calls != 0 || b.calls != 1) {
				t.Fatal("choice used the former controller")
			}
			if state == "control" && old.Matches(NewDiscardAction(uuid.Nil, a.PlayerID(), uuid.New()), g) {
				t.Fatal("replacement kept its former controller")
			}
			if want == 1 {
				if !refs[0].SameObject(saved) {
					t.Fatal("wrong incarnation")
				}
				if state == "control" && refs[0].Controller() != b.PlayerID() {
					t.Fatal("stale controller")
				}
				refs[0].card.SetOwner(b.PlayerID())
				if g.WaitingDiscardPermanentSources()[0].card.Owner() != a.PlayerID() {
					t.Fatal("query exposed mutable data")
				}
			}
			if refs := clone.WaitingDiscardPermanentSources(); len(refs) != 1 || !refs[0].SameObject(saved) {
				t.Fatal("clone changed")
			}
			if state == "return" {
				g.ApplyContinuousEffects()
				refs := g.WaitingDiscardPermanentSources()
				if len(refs) != 1 || refs[0].SameObject(saved) || old.IsActive(g) {
					t.Fatal("returned object inherited old rule")
				}
			}
		})
	}
}

func TestWaitingDiscardSourcesChoices(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewArtifact("Discard rule", "{R}", WithStaticAbility(DiscardToLibraryReplacement())), a.PlayerID())
	g.PutOnBattlefield(NewArtifact("Unrelated", "{R}"), a.PlayerID())
	first, second := BindDamageSource("first"), BindDamageSource("second")
	ctx := &EffectContext{Game: g, Controller: b.PlayerID()}
	for _, binding := range []*DamageSourceBinding{first, second} {
		choice := ChooseDamageSourceFromWaitingDiscardReferences(You, DamageSourceHasColor(core.Red), binding)
		if err := ValidateCardDefinition("Choice", NewArtifact("Choice", "", WithAbility(Activate(Costs(), choice)))); err != nil {
			t.Fatal(err)
		}
		if err := choice.Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if !g.resolution.damageSourceBindings[binding].SameObject(g.CaptureDamageSource(source.ID())) {
			t.Fatal("wrong choice")
		}
	}
	if err := ChooseDamageSourceFromWaitingDiscardReferences(You, DamageSourceHasColor(core.Blue), first).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.resolution.damageSourceBindings[first].Valid() || !g.resolution.damageSourceBindings[second].Valid() {
		t.Fatal("binding isolation")
	}
	ctx.Controller = uuid.New()
	if err := ChooseDamageSourceFromWaitingDiscardReferences(You, DamageSourceFilter{}, second).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.resolution.damageSourceBindings[second].Valid() {
		t.Fatal("absent player retained choice")
	}
	for _, effect := range []ResolvingEffect{
		ChooseDamageSourceFromWaitingDiscardReferences(nil, DamageSourceFilter{}, first),
		ChooseDamageSourceFromWaitingDiscardReferences(You, DamageSourceFilter{}, nil),
		ChooseDamageSourceFromWaitingDiscardReferences(You, DamageSourceFilter{}, BindDamageSource("")),
		Sequence(DealDamageFrom(FromDamageSource(first), You, Fixed(1)), ChooseDamageSourceFromWaitingDiscardReferences(You, DamageSourceFilter{}, first)),
	} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))) == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

type waitingDiscardPlayer struct {
	*BasePlayer
	accept bool
	calls  int
}

func (p *waitingDiscardPlayer) ChooseMayAbility(string) bool {
	p.calls++
	return p.accept
}

func TestWaitingDiscardSourcesExecution(t *testing.T) {
	for _, tc := range []struct {
		name                            string
		accept, cost, cancel            bool
		library, graveyard, hand, calls int
	}{
		{"accept", true, false, false, 1, 0, 0, 1},
		{"decline", false, false, false, 0, 1, 0, 1},
		{"cost", true, true, false, 0, 1, 0, 0},
		{"recheck", true, false, true, 0, 0, 1, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &waitingDiscardPlayer{BasePlayer: NewBasePlayer("A"), accept: tc.accept}
			g := NewGame(a, NewBasePlayer("B"))
			source := g.PutOnBattlefield(NewArtifact("Rule", "", WithStaticAbility(DiscardToLibraryReplacement())), a.PlayerID())
			card := NewArtifact("Discard", "")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if tc.cancel {
				g.effects.AddReplacement(&discardToHandReplacement{})
			}
			if tc.cost {
				g.PlayerDiscard(a, card.ID())
			} else {
				g.PlayerDiscardByEffect(a, card.ID(), source.ID())
			}
			if len(a.Library()) != tc.library || len(a.Graveyard()) != tc.graveyard || len(a.Hand()) != tc.hand || a.calls != tc.calls {
				t.Fatalf("library/graveyard/hand/choices = %d/%d/%d/%d", len(a.Library()), len(a.Graveyard()), len(a.Hand()), a.calls)
			}
			if len(g.WaitingDiscardPermanentSources()) != 1 || a.calls != tc.calls {
				t.Fatal("query consumed the rule or made a choice")
			}
		})
	}
}

func TestWaitingDiscardSourcesInvalidRegistrations(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewArtifact("Rule", "", WithStaticAbility(DiscardToLibraryReplacement())), a.PlayerID())
	var saved *discardToLibraryReplacement
	for _, replacement := range g.effects.cycleReplacements {
		if rule, ok := replacement.(*discardToLibraryReplacement); ok {
			saved = rule
		}
	}
	if saved == nil {
		t.Fatal("missing replacement")
	}
	for _, field := range []string{"source", "incarnation", "rule"} {
		t.Run(field, func(t *testing.T) {
			invalid := saved.Clone().(*discardToLibraryReplacement)
			switch field {
			case "source":
				invalid.sourceID = uuid.New()
			case "incarnation":
				invalid.incarnation = uuid.Nil
			case "rule":
				invalid.ruleID = uuid.New()
			}
			g.effects.replacements = []ReplacementEffect{invalid}
			g.effects.cycleReplacements = nil
			if invalid.IsActive(g) || len(g.WaitingDiscardPermanentSources()) != 0 {
				t.Fatal("invalid registration supplied a source")
			}
		})
	}
	g.effects.replacements = []ReplacementEffect{saved}
	source.RuntimeAbilities = []Ability{StaticAbility(DiscardToLibraryReplacement())}
	if saved.IsActive(g) || len(g.WaitingDiscardPermanentSources()) != 0 {
		t.Fatal("a different ability restored a saved rule")
	}
}
