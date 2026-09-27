package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCreateTokenCopiesIntoKeepsCreatedSet(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	old := g.PutOnBattlefield(NewCreature("Cat", "", 1, 1), b.PlayerID())
	result := BindCreatedTokens("cats")
	member := BindPermanent("cat")
	def := DefineToken("Cat", TokenTypes(core.TypeCreature), TokenPT(0, 0))
	body := Sequence(CreateTokenCopiesInto(You, Fixed(3), def, result), ForEach(FromCreatedTokens(result), member, AddCountersTo(From(member), core.P1P1, Fixed(1))))
	if err := ValidateCardDefinition("Maker", NewInstant("Maker", "", Spell(body))); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, body, a.PlayerID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.CheckStateBasedActions()
	tokens := g.FilterBattlefield(IsToken)
	if len(tokens) != 3 || old.Counters[core.P1P1] != 0 {
		t.Fatal("result set included old permanent or missed created tokens")
	}
	for _, token := range tokens {
		if token.ControllerID() != b.PlayerID() || token.Counters[core.P1P1] != 1 {
			t.Fatal("created token identity or controller changed")
		}
	}
}

func TestCreatedTokenSetAbsentMembersAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	result := BindCreatedTokens("tokens")
	def := DefineToken("Cat", TokenTypes(core.TypeCreature), TokenPT(1, 1))
	if err := CreateTokenCopiesInto(You, Fixed(2), def, result).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	values := g.resolution.createdTokenBindings[result]
	if len(values) != 2 {
		t.Fatal("wrong number of saved tokens")
	}
	clone := g.Clone()
	clone.resolution.createdTokenBindings[result][0] = capturedPermanent{}
	if g.resolution.createdTokenBindings[result][0] != values[0] {
		t.Fatal("clone shared token result storage")
	}
	first := g.FindPermanent(values[0].id)
	second := g.FindPermanent(values[1].id)
	g.RemoveFromBattlefield(first)
	g.PutOnBattlefield(first.Card, a.PlayerID())
	g.PhaseOut(second.ID())
	if got := FromCreatedTokens(result).resolvePermanents(&EffectContext{Game: g}); len(got) != 0 {
		t.Fatal("returned or phased token remained in result set")
	}
	if got := FromCreatedTokens(result).resolvePermanents(&EffectContext{Game: clone}); len(got) != 1 {
		t.Fatal("clone did not keep independent token identities")
	}
}

func TestCreatedTokenSetRegistration(t *testing.T) {
	result := BindCreatedTokens("tokens")
	def := DefineToken("Cat", TokenTypes(core.TypeCreature), TokenPT(1, 1))
	member := BindPermanent("member")
	if got := InspectDefinition(CreateTokenCopiesInto(You, Fixed(2), def, result)); got.Role != "resolving-effect" || len(got.Children) != 4 {
		t.Fatalf("inspection: %+v", got)
	}
	for _, effect := range []ResolvingEffect{
		CreateTokenCopiesInto(nil, Fixed(2), def, result),
		CreateTokenCopiesInto(You, nil, def, result),
		CreateTokenCopiesInto(You, Fixed(2), nil, result),
		CreateTokenCopiesInto(You, Fixed(2), def, nil),
		CreateTokenCopiesInto(You, Fixed(2), def, BindCreatedTokens("")),
	} {
		if err := ValidateCardDefinition("Bad", NewInstant("Bad", "", Spell(effect))); err == nil {
			t.Fatal("accepted invalid token result declaration")
		}
	}
	read := ForEach(FromCreatedTokens(result), member, GainLifeFor(You, Fixed(1)))
	if err := ValidateCardDefinition("Bad", NewInstant("Bad", "", Spell(read))); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("missing producer: %v", err)
	}
	branch := Sequence(IfState(PermanentIsTapped(This), CreateTokenCopiesInto(You, Fixed(2), def, result)), read)
	if err := ValidateCardDefinition("Bad", NewCreature("Bad", "", 1, 1, WithAbility(Activate(Costs(), branch)))); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("non-dominating producer: %v", err)
	}
}

func TestCreatedTokenSetClearsFailedBatch(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	result := BindCreatedTokens("tokens")
	def := DefineToken("Cat", TokenTypes(core.TypeCreature), TokenPT(1, 1))
	effect := CreateTokenCopiesInto(You, Fixed(2), def, result)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(g.resolution.createdTokenBindings[result]) != 2 {
		t.Fatal("missing first batch")
	}
	ctx.Controller = b.PlayerID()
	if err := CreateTokenCopiesInto(You, Fixed(0), def, result).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(g.resolution.createdTokenBindings[result]) != 0 {
		t.Fatal("zero batch retained stale tokens")
	}
	land := DefineToken("Land", TokenTypes(core.TypeLand))
	g.effects.Rules.SetLandsCantEnter(true)
	if err := CreateTokenCopiesInto(You, Fixed(2), land, result).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(g.resolution.createdTokenBindings[result]) != 0 {
		t.Fatal("rejected entries appeared in result")
	}
}

func TestCreatedTokenSetDelayedCaptureAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	result := BindCreatedTokens("tokens")
	def := DefineToken("Cat", TokenTypes(core.TypeCreature), TokenPT(1, 1))
	effect := Sequence(CreateTokenCopiesInto(You, Fixed(2), def, result), AtNext(EachEndStep, ReturnEachToHand(FromCreatedTokens(result))))
	if err := ValidateCardDefinition("Maker", NewInstant("Maker", "", Spell(effect))); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, effect, a.PlayerID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.FilterBattlefield(IsToken)) != 2 {
		t.Fatal("tokens absent before delayed step")
	}
	clone := g.Clone()
	clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
	clone.PutTriggersOnStack()
	clone.ResolveStack()
	if len(clone.FilterBattlefield(IsToken)) != 0 || len(g.FilterBattlefield(IsToken)) != 2 {
		t.Fatal("delayed result set was not captured or clone was not isolated")
	}
}
