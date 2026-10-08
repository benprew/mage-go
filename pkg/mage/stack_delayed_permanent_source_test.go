package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestStackDelayedPermanentSourcesEventFrame(t *testing.T) {
	for _, change := range []string{"live", "departed", "returned", "phased", "phase-cycle", "color", "control", "face-down", "shroud"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			bound := BindPermanent("dead")
			observer := NewEnchantment("Observer", "", WithAbility(WhenEvent(AnotherCreatureEnters().WithPermanent(bound),
				AtNext(EachEndStep, DealDamageFrom(DamageSourceOf(From(bound)), You, Fixed(2))))))
			if err := ValidateCardDefinition(observer.Name(), observer); err != nil {
				t.Fatal(err)
			}
			observer.SetOwner(a.PlayerID())
			watcher := g.PutOnBattlefield(observer, a.PlayerID())
			card := NewCreature("Saved", "{U}", 1, 2)
			card.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(card, b.PlayerID())
			original := g.CaptureDamageSource(p.ID())
			g.ResolveStack()
			g.BouncePermanentToHand(watcher)
			if len(g.DelayedTriggers()) != 1 || len(g.StackBoundPermanentSources()) != 0 {
				t.Fatal("waiting trigger entered stack category")
			}
			switch change {
			case "departed", "returned":
				g.BouncePermanentToHand(p)
				if change == "returned" {
					b.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, a.PlayerID())
					g.ChangeObjectColor(card.ID(), core.Red)
				}
			case "phased", "phase-cycle":
				ids := g.PhaseOut(p.ID())
				if change == "phase-cycle" {
					g.PhaseIn(ids)
				}
			case "color":
				g.ChangeObjectColor(p.ID(), core.Red)
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: uuid.New(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "face-down":
				g.MutablePermanent(p.ID()).FaceDown = true
			case "shroud":
				g.MutablePermanent(p.ID()).GrantBaseAttr(core.Shroud)
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
			if len(g.StackBoundPermanentSources()) != 0 {
				t.Fatal("pending trigger entered stack category")
			}
			g.PutTriggersOnStack()
			got := g.StackBoundPermanentSources()
			if len(got) != 1 || !got[0].SameObject(original) {
				t.Fatal("saved delayed event frame is absent")
			}
			if got[0].HasColor(core.Red) != (change == "color") || got[0].HasColor(core.Blue) != (change != "color" && change != "face-down") {
				t.Fatal("live or retained color is wrong")
			}
			controller := b.PlayerID()
			if change == "control" {
				controller = a.PlayerID()
			}
			if got[0].Controller() != controller || (change == "face-down" && got[0].Name() != "") {
				t.Fatal("controller or hidden information is wrong")
			}
			originalObject := g.StackPeek()
			if g.CopyStackObjectDirect(originalObject, b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			clone := g.Clone()
			got[0].card.AddType(core.TypeArtifact)
			if len(g.StackBoundPermanentSources()) != 1 || g.StackBoundPermanentSources()[0].HasType(core.TypeArtifact) {
				t.Fatal("duplicate reference or exposed card data")
			}
			g.CounterStackObject(originalObject.ID)
			if len(g.StackBoundPermanentSources()) != 1 {
				t.Fatal("copy lost saved reference")
			}
			g.CounterStackObject(g.StackPeek().ID)
			if len(g.StackBoundPermanentSources()) != 0 || len(clone.StackBoundPermanentSources()) != 1 {
				t.Fatal("final reference or clone lifetime failed")
			}
			clone.ResolveStack()
			wantLife := 18
			if change == "phased" {
				wantLife = 20
			}
			if clone.GetPlayer(a.PlayerID()).Life() != wantLife || clone.GetPlayer(b.PlayerID()).Life() != wantLife {
				t.Fatal("delayed body or copied controller changed")
			}
		})
	}
}

func TestStackDelayedPermanentSourcesTokens(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	tokens := BindCreatedTokens("tokens")
	effect := Sequence(CreateTokenCopiesInto(You, Fixed(2), DefineToken("Cat", TokenTypes(core.TypeCreature), TokenPT(1, 1)), tokens),
		AtNext(EachEndStep, ReturnEachToHand(FromCreatedTokens(tokens))))
	if err := ValidateCardDefinition("Maker", NewInstant("Maker", "", Spell(effect))); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, effect, a.PlayerID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	created := g.FilterBattlefield(IsToken)
	g.resolution.Clear()
	g.BouncePermanentToHand(created[0])
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	got := g.StackBoundPermanentSources()
	if len(got) != 2 {
		t.Fatal("saved token set lost a member")
	}
	firstID, secondID := got[0].ID(), got[1].ID()
	if slices.Compare(firstID[:], secondID[:]) > 0 {
		t.Fatal("token map order changed the source list")
	}
	for _, token := range created {
		if !slices.ContainsFunc(got, func(source DamageSource) bool {
			return source.ID() == token.ID() && source.Incarnation() == token.incarnationID
		}) {
			t.Fatal("token identity was lost")
		}
	}
	clone := g.Clone()
	g.ResolveStack()
	if len(g.FilterBattlefield(IsToken)) != 0 || len(g.StackBoundPermanentSources()) != 0 || len(clone.StackBoundPermanentSources()) != 2 {
		t.Fatal("token body, lifetime or clone isolation failed")
	}
}

func TestStackDelayedPermanentSourcesChoiceLifetimeAndDeclarations(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	chooser := &sourceChoicePlayer{BasePlayer: a, index: -1}
	g := NewGame(chooser, b)
	target := DeclarePermanentTarget("saved", IsCreature)
	chosen := BindDamageSource("chosen")
	choice := ChooseDamageSourceFromStackBindings(You, DamageSourceFilter{}, chosen)
	body := Sequence(choice, DealDamageFrom(FromDamageSource(chosen), You, Fixed(2)))
	card := NewArtifact("Observer", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, body), WithTargets(target))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	p := g.PutOnBattlefield(card, a.PlayerID())
	q := g.PutOnBattlefield(NewCreature("Saved", "{U}", 1, 3), b.PlayerID())
	for range 2 {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{q.ID()}); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	clone := g.Clone()
	if len(g.StackBoundPermanentSources()) != 1 {
		t.Fatal("shared declaration frames were duplicated")
	}
	g.ResolveTopOfStack()
	if chooser.calls != 1 || a.Life() != 18 || len(g.StackBoundPermanentSources()) != 1 {
		t.Fatal("remaining stack reference was not chosen")
	}
	g.ResolveTopOfStack()
	if chooser.calls != 2 || a.Life() != 16 || len(g.StackBoundPermanentSources()) != 0 || len(clone.StackBoundPermanentSources()) != 1 {
		t.Fatal("resolving delayed reference or final frame lifetime failed")
	}
	for _, invalid := range []ResolvingEffect{
		AtNext(EachEndStep, ChooseDamageSourceFromStackBindings(nil, DamageSourceFilter{}, chosen)),
		AtNext(EachEndStep, ChooseDamageSourceFromStackBindings(You, DamageSourceFilter{}, nil)),
		AtNext(EachEndStep, Sequence(DealDamageFrom(FromDamageSource(chosen), You, Fixed(1)), choice)),
		AtNext(EachEndStep, Destroy(From(BindPermanent("unbound")))),
	} {
		if err := ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), invalid)))); err == nil {
			t.Fatal("invalid delayed declaration accepted")
		}
	}
}

func TestStackDelayedPermanentSourcesBoundaries(t *testing.T) {
	g, a, _, _ := spellSourceFixture(t)
	p := g.PutOnBattlefield(NewArtifact("Saved", ""), a.PlayerID())
	id := capturedPermanent{p.ID(), p.incarnationID}
	bound, target, tokens := BindPermanent("bound"), DeclarePermanentTarget("target", IsArtifact), BindCreatedTokens("tokens")
	body := &capturedDelayedBody{body: GainLifeFor(You, Fixed(1)),
		permanents:    map[*PermanentBinding]capturedPermanent{bound: id, nil: id, BindPermanent("missing"): {p.ID(), uuid.Nil}},
		targets:       map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {id}, nil: {id}},
		createdTokens: map[*CreatedTokensBinding][]capturedPermanent{tokens: {id}, nil: {id}}}
	g.PushStack(&StackObject{ID: uuid.New(), Effects: []Effect{Sequence(body, AtNext(EachEndStep, body))}})
	if got := g.StackBoundPermanentSources(); len(got) != 1 || !got[0].SameObject(g.CaptureDamageSource(p.ID())) {
		t.Fatal("nested frame references are absent or duplicated")
	}
	g.GetStack().Pop()
	g.PushStack(&StackObject{ID: uuid.New(), Effects: []Effect{AtNext(EachEndStep, body)}})
	g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{bound: id}
	if len(g.StackBoundPermanentSources()) != 0 {
		t.Fatal("future producer or global frame entered source category")
	}
	g.GetStack().Pop()
	invalid := &capturedDelayedBody{body: GainLifeFor(You, Fixed(1)),
		permanents:    map[*PermanentBinding]capturedPermanent{nil: id, bound: {p.ID(), uuid.New()}},
		targets:       map[*targetDeclaration][]capturedPermanent{nil: {id}, target.targetDeclaration: {{p.ID(), uuid.Nil}}},
		createdTokens: map[*CreatedTokensBinding][]capturedPermanent{nil: {id}, tokens: {{a.PlayerID(), uuid.New()}}}}
	g.PushStack(&StackObject{ID: uuid.New(), Effects: []Effect{invalid}})
	if len(g.StackBoundPermanentSources()) != 0 {
		t.Fatal("invalid frame identities entered source category")
	}
}

func TestStackDelayedPermanentSourcesPluralTargets(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	targets := DeclarePermanentTargets("saved", 2, 2, IsCreature)
	card := NewArtifact("Observer", "", WithAbility(Activate(Costs(),
		AtNext(EachEndStep, DestroyEach(targets)), WithTargets(targets))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	p := g.PutOnBattlefield(card, a.PlayerID())
	ids := make([]uuid.UUID, 0, 2)
	for range 2 {
		q := g.PutOnBattlefield(NewCreature("Saved", "{U}", 1, 3), b.PlayerID())
		ids = append(ids, q.ID())
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, ids); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	got := g.StackBoundPermanentSources()
	slices.SortFunc(ids, func(a, b uuid.UUID) int { return slices.Compare(a[:], b[:]) })
	if len(got) != 2 || got[0].ID() != ids[0] || got[1].ID() != ids[1] {
		t.Fatal("plural targets lost identity or stable order")
	}
	clone := g.Clone()
	g.ResolveStack()
	for _, id := range ids {
		if g.FindPermanent(id) != nil || clone.FindPermanent(id) == nil {
			t.Fatal("plural delayed body or clone isolation failed")
		}
	}
}
