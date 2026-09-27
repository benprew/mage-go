package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCreateOneTokenBindsCreatedIncarnation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	created := BindPermanent("created token")
	def := DefineToken("Cat", TokenTypes(core.TypeCreature), TokenSubtypes("Cat"), TokenColors(core.White), TokenPT(2, 2))
	body := Sequence(CreateOneToken(You, def, created), AddCountersTo(From(created), core.P1P1, Fixed(1)))
	if err := ValidateCardDefinition("Maker", NewInstant("Maker", "", Spell(body))); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	if err := ApplyEffect(clone, body, a.PlayerID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.FilterBattlefield(IsToken)) != 0 {
		t.Fatal("clone changed original game")
	}
	if err := ApplyEffect(g, body, a.PlayerID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	tokens := g.FilterBattlefield(IsToken)
	if len(tokens) != 1 || tokens[0].ControllerID() != b.PlayerID() || tokens[0].Counters[core.P1P1] != 1 {
		t.Fatalf("created tokens: %+v", tokens)
	}
}

func TestCreateOneTokenResultAbsentAndCleared(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	created := BindPermanent("created token")
	def := DefineToken("Cat", TokenTypes(core.TypeCreature), TokenPT(2, 2))
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	effect := Sequence(Snapshot(This, created), CreateOneToken(ControllerOf(This), def, created), AddCountersTo(From(created), core.P1P1, Fixed(1)))
	g.PhaseOut(source.ID())
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.FilterBattlefield(IsToken)) != 0 || g.resolution.permanentBindings[created].id != (capturedPermanent{}).id {
		t.Fatal("absent player retained stale token binding")
	}
	g.PhaseIn([]uuid.UUID{source.ID()})
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	token := g.FilterBattlefield(IsToken)[0]
	if token.Counters[core.P1P1] != 1 {
		t.Fatal("token result was absent")
	}
	g.RemoveFromBattlefield(token)
	if _, ok := From(created).resolvePermanent(&EffectContext{Game: g}); ok {
		t.Fatal("departed token remained present")
	}
}

func TestCreateOneTokenResultDeclaration(t *testing.T) {
	def := DefineToken("Cat", TokenTypes(core.TypeCreature), TokenPT(2, 2))
	result := BindPermanent("token")
	if got := InspectDefinition(CreateOneToken(You, def, result)); got.Role != "resolving-effect" || len(got.Children) != 3 {
		t.Fatalf("inspection: %+v", got)
	}
	for _, effect := range []ResolvingEffect{
		CreateOneToken(You, def, nil),
		CreateOneToken(You, def, BindPermanent("")),
		CreateOneToken(nil, def, result),
		CreateOneToken(You, nil, result),
	} {
		if err := ValidateCardDefinition("Bad", NewInstant("Bad", "", Spell(effect))); err == nil {
			t.Fatal("accepted invalid token result")
		}
	}
	if err := ValidateCardDefinition("Bad", NewInstant("Bad", "", Spell(AddCountersTo(From(result), core.P1P1, Fixed(1))))); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("missing producer: %v", err)
	}
}

func TestCreateOneTokenResultClearsRejectedEntry(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	old := g.PutOnBattlefield(NewCreature("Old", "", 1, 1), a.PlayerID())
	result := BindPermanent("created land")
	g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{result: {old.ID(), old.incarnationID}}
	g.effects.Rules.SetLandsCantEnter(true)
	def := DefineToken("Land", TokenTypes(core.TypeLand))
	if err := CreateOneToken(You, def, result).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if _, ok := g.resolution.permanentBindings[result]; ok || len(g.FilterBattlefield(IsToken)) != 0 {
		t.Fatal("rejected entry retained a token result")
	}
}
