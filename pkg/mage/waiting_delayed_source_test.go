package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedPermanentSourcesLifecycle(t *testing.T) {
	for _, change := range []string{"live", "departed", "returned", "phased", "phase-cycle", "color", "control", "face-down", "shroud"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			target := DeclarePermanentTarget("saved", IsCreature)
			card := NewArtifact("Observer", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, Destroy(target)), WithTargets(target))))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			p := g.PutOnBattlefield(card, a.PlayerID())
			saved := NewCreature("Saved", "{U}", 1, 3)
			saved.SetOwner(b.PlayerID())
			q := g.PutOnBattlefield(saved, b.PlayerID())
			original := g.CaptureDamageSource(q.ID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{q.ID()}); err != nil {
				t.Fatal(err)
			}
			if len(g.WaitingDelayedPermanentSources()) != 0 {
				t.Fatal("future producer supplied a source")
			}
			g.ResolveStack()
			g.BouncePermanentToHand(p)
			switch change {
			case "departed", "returned":
				g.BouncePermanentToHand(q)
				if change == "returned" {
					b.RemoveFromHand(saved.ID())
					g.PutOnBattlefield(saved, a.PlayerID())
					g.ChangeObjectColor(saved.ID(), core.Red)
				}
			case "phased", "phase-cycle":
				ids := g.PhaseOut(q.ID())
				if change == "phase-cycle" {
					g.PhaseIn(ids)
				}
			case "color":
				g.ChangeObjectColor(q.ID(), core.Red)
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: uuid.New(), TargetID: q.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "face-down":
				g.MutablePermanent(q.ID()).FaceDown = true
			case "shroud":
				g.MutablePermanent(q.ID()).GrantBaseAttr(core.Shroud)
			}
			got := g.WaitingDelayedPermanentSources()
			if len(got) != 1 || !got[0].SameObject(original) {
				t.Fatal("waiting frame lost its exact target")
			}
			if got[0].HasColor(core.Red) != (change == "color") || got[0].HasColor(core.Blue) != (change != "color" && change != "face-down") {
				t.Fatal("wrong current or retained color")
			}
			controller := b.PlayerID()
			if change == "control" {
				controller = a.PlayerID()
			}
			if got[0].Controller() != controller || (change == "face-down" && got[0].Name() != "") {
				t.Fatal("wrong control or hidden data")
			}
			got[0].card.AddType(core.TypeArtifact)
			if g.WaitingDelayedPermanentSources()[0].HasType(core.TypeArtifact) {
				t.Fatal("query exposed saved card data")
			}
			clone := g.Clone()
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
			if len(g.WaitingDelayedPermanentSources()) != 0 || len(g.StackBoundPermanentSources()) != 0 || len(clone.WaitingDelayedPermanentSources()) != 1 {
				t.Fatal("waiting, pending or clone boundary failed")
			}
			g.PutTriggersOnStack()
			if len(g.StackBoundPermanentSources()) != 1 || len(g.WaitingDelayedPermanentSources()) != 0 {
				t.Fatal("placement category failed")
			}
			g.ResolveStack()
			survives := change == "returned" || change == "phased"
			if (g.FindPermanentIncludingPhased(q.ID()) != nil) != survives {
				t.Fatal("delayed action changed identity")
			}
		})
	}
}

func TestWaitingDelayedPermanentSourcesFramesAndMatchers(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	tokens := BindCreatedTokens("tokens")
	effect := Sequence(CreateTokenCopiesInto(You, Fixed(2), DefineToken("Cat", TokenTypes(core.TypeCreature), TokenPT(1, 1)), tokens), AtNext(EachEndStep, ReturnEachToHand(FromCreatedTokens(tokens))))
	if err := ApplyEffect(g, effect, a.PlayerID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	created := g.FilterBattlefield(IsToken)
	g.resolution.Clear()
	g.BouncePermanentToHand(created[0])
	got := g.WaitingDelayedPermanentSources()
	if len(got) != 2 {
		t.Fatal("waiting token frame lost a member")
	}
	for _, p := range created {
		if !slices.ContainsFunc(got, func(s DamageSource) bool { return s.ID() == p.ID() && s.Incarnation() == p.incarnationID }) {
			t.Fatal("wrong token identity")
		}
	}
	id0, id1 := got[0].ID(), got[1].ID()
	if slices.Compare(id0[:], id1[:]) > 0 {
		t.Fatal("unstable map order")
	}
	g.triggers.SetDelayed(nil)
	p := g.PutOnBattlefield(NewCreature("Matched", "{U}", 1, 2), a.PlayerID())
	original := g.CaptureDamageSource(p.ID())
	if err := ApplyEffect(g, WhenLeavesThisTurn(This, GainLifeFor(You, Fixed(1))), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if got := g.WaitingDelayedPermanentSources(); len(got) != 1 || !got[0].SameObject(original) {
		t.Fatal("waiting event matcher is absent")
	}
	g.BouncePermanentToHand(p)
	g.PutTriggersOnStack()
	if len(g.WaitingDelayedPermanentSources()) != 1 {
		t.Fatal("persistent delayed ability stopped waiting")
	}
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatal("persistent body failed")
	}
	g.triggers.ClearEndOfTurn()
	if len(g.WaitingDelayedPermanentSources()) != 0 {
		t.Fatal("expired ability still supplies sources")
	}
}

func TestWaitingDelayedPermanentSourcesChoiceAndBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	chooser := &sourceChoicePlayer{BasePlayer: a, index: -1}
	g := NewGame(chooser, b)
	p := g.PutOnBattlefield(NewArtifact("Saved", ""), a.PlayerID())
	id := capturedPermanent{p.ID(), p.incarnationID}
	bound := BindPermanent("saved")
	target := DeclarePermanentTargets("targets", 1, 2, IsArtifact)
	tokens := BindCreatedTokens("tokens")
	body := &capturedDelayedBody{body: GainLifeFor(You, Fixed(1)), permanents: map[*PermanentBinding]capturedPermanent{bound: id, nil: id, BindPermanent("bad"): {p.ID(), uuid.Nil}}, targets: map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {id}, nil: {id}}, createdTokens: map[*CreatedTokensBinding][]capturedPermanent{tokens: {id}, nil: {id}}}
	g.RegisterDelayedTrigger(&DelayedTrigger{EventType: core.EvtEndStep, Effects: []Effect{Sequence(body, AtNext(EachEndStep, body))}})
	g.RegisterDelayedTrigger(&DelayedTrigger{EventType: core.EvtEndStep, modes: []Mode{{Effects: []Effect{body}}}})
	g.RegisterDelayedTrigger(&DelayedTrigger{EventType: core.EvtEndStep, MatchEventID: p.ID(), matchIncarnation: uuid.New(), SourceID: p.ID()})
	if got := g.WaitingDelayedPermanentSources(); len(got) != 1 || !got[0].SameObject(g.CaptureDamageSource(p.ID())) {
		t.Fatal("nested or modal frames, invalid references or deduplication failed")
	}
	result := BindDamageSource("choice")
	choice := ChooseDamageSourceFromWaitingDelayedBindings(You, DamageSourceHasType(core.TypeArtifact), result)
	effect := Sequence(choice, DealDamageFrom(FromDamageSource(result), You, Fixed(2)))
	if err := ValidateCardDefinition("Choice", NewInstant("Choice", "", Spell(effect))); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if chooser.calls != 1 || a.Life() != 18 {
		t.Fatal("fallback choice or damage failed")
	}
	g.triggers.SetDelayed(nil)
	if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if chooser.calls != 1 || a.Life() != 18 {
		t.Fatal("empty choice reused a prior result")
	}
	g.triggers.SetDelayed([]*DelayedTrigger{
		nil,
		{EventType: core.EvtEndStep, SourceID: p.ID(), sourceIncarnation: p.incarnationID},
		{EventType: core.EvtEndStep, MatchEventID: p.ID(), matchIncarnation: uuid.New()},
		{EventType: core.EvtEndStep, Effects: []Effect{AtNext(EachEndStep, body)}},
		{EventType: core.EvtEndStep, Effects: []Effect{&capturedDelayedBody{
			body: GainLifeFor(You, Fixed(1)),
			permanents: map[*PermanentBinding]capturedPermanent{
				nil: id, bound: {p.ID(), uuid.Nil}, BindPermanent("wrong kind"): {a.PlayerID(), p.incarnationID},
			},
		}}},
	})
	g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{bound: id}
	if len(g.WaitingDelayedPermanentSources()) != 0 {
		t.Fatal("invalid identities, future producers or global frames supplied sources")
	}
	for _, invalid := range []ResolvingEffect{ChooseDamageSourceFromWaitingDelayedBindings(nil, DamageSourceFilter{}, result), ChooseDamageSourceFromWaitingDelayedBindings(You, DamageSourceFilter{}, nil), Sequence(DealDamageFrom(FromDamageSource(result), You, Fixed(1)), choice)} {
		if err := ValidateCardDefinition("Invalid", NewInstant("Invalid", "", Spell(invalid))); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestWaitingDelayedPermanentSourcesNativeFrames(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	bound := BindPermanent("entry")
	observer := NewEnchantment("Observer", "", WithAbility(WhenEvent(AnotherCreatureEnters().WithPermanent(bound), AtNextModes(EachEndStep, ModeOf("Destroy", Destroy(From(bound))), ModeOf("Return", ReturnToHand(From(bound)))))))
	if err := ValidateCardDefinition(observer.Name(), observer); err != nil {
		t.Fatal(err)
	}
	watcher := g.PutOnBattlefield(observer, a.PlayerID())
	q := g.PutOnBattlefield(NewCreature("Entry", "{U}", 1, 3), b.PlayerID())
	original := g.CaptureDamageSource(q.ID())
	if len(g.WaitingDelayedPermanentSources()) != 0 {
		t.Fatal("pending entry created a waiting source")
	}
	g.ResolveStack()
	g.BouncePermanentToHand(watcher)
	if got := g.WaitingDelayedPermanentSources(); len(got) != 1 || !got[0].SameObject(original) {
		t.Fatal("native event frame is absent")
	}
	g.triggers.SetDelayed(nil)
	targets := DeclarePermanentTargets("saved", 2, 2, IsCreature)
	delayed := AtNext(EachEndStep, DestroyEach(targets))
	card := NewArtifact("Plural delay", "", WithAbility(Activate(Costs(), delayed, WithTargets(targets))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	p := g.PutOnBattlefield(card, a.PlayerID())
	r := g.PutOnBattlefield(NewCreature("Other", "{R}", 1, 3), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{q.ID(), r.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 2 {
			t.Fatal("native plural frames are absent")
		}
	}
	if g.StackSize() != 0 || len(g.DelayedTriggers()) != 1 {
		t.Fatal("query changed timing")
	}
	clone := g.Clone()
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	if len(g.WaitingDelayedPermanentSources()) != 0 || len(clone.WaitingDelayedPermanentSources()) != 2 {
		t.Fatal("plural lifetime or clone failed")
	}
	g.PutTriggersOnStack()
	g.ResolveStack()
	if g.FindPermanent(q.ID()) != nil || g.FindPermanent(r.ID()) != nil {
		t.Fatal("plural body failed")
	}
}
