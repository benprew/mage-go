package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTokenDefinitionPlanInvalid(t *testing.T) {
	shared := BindPlayer("damaged player")
	for _, build := range []TokenDefinitionFactory{
		nil,
		func() TokenDefinition { return nil },
		func() TokenDefinition { panic("invalid declaration") },
		func() TokenDefinition { return DefineToken("Bad", TokenTypes(core.TypeCreature)) },
		func() TokenDefinition {
			return DefineToken("Bad", TokenTypes(core.TypeCreature), TokenPT(1, 1), TokenKeywords(core.AttrCanAttack))
		},
		func() TokenDefinition {
			return DefineToken("Shared", TokenTypes(core.TypeCreature), TokenPT(1, 1), TokenAbilities(WhenEvent(DealsDamageToPlayer(This, shared), GainLifeFor(FromPlayer(shared), ValueAtResolution(Fixed(1))))))
		},
	} {
		if _, err := NewTokenDefinitionFactory(build); err == nil {
			t.Fatal("invalid token plan accepted")
		}
	}
}

func TestTokenDefinitionPlanResultPaths(t *testing.T) {
	for _, single := range []bool{false, true} {
		power := 1
		factory, err := NewTokenDefinitionFactory(func() TokenDefinition {
			return DefineToken("Citizen Token", TokenTypes(core.TypeCreature), TokenSubtypes("Citizen"), TokenPT(power, 1))
		})
		if err != nil {
			t.Fatal(err)
		}
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		one, many := BindPermanent("one"), BindCreatedTokens("many")
		ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
		effect := CreateTokenCopiesInto(You, ValueAtResolution(Fixed(2)), factory(), many)
		if single {
			effect = CreateOneToken(You, factory(), one)
		}
		if err := effect.Apply(ctx); err != nil {
			t.Fatal(err)
		}
		count := 2
		if single {
			count = 1
		}
		if len(g.FilterBattlefield(IsToken)) != count {
			t.Fatal("result count")
		}
		for _, p := range g.FilterBattlefield(IsToken) {
			if _, err := g.CaptureCopiableValues(p); err != nil {
				t.Fatal(err)
			}
		}
		if single && len(g.resolution.permanentBindings) != 1 || !single && len(g.resolution.createdTokenBindings[many]) != 2 {
			t.Fatal("result binding")
		}
		power = 2
		if err := effect.Apply(ctx); err == nil || len(g.FilterBattlefield(IsToken)) != count {
			t.Fatal("failed result creation changed battlefield")
		}
		if _, ok := g.resolution.permanentBindings[one]; single && ok {
			t.Fatal("failed creation retained single result")
		}
		if !single && len(g.resolution.createdTokenBindings[many]) != 0 {
			t.Fatal("failed creation retained batch result")
		}
	}
}

func TestTokenDefinitionPlanCreationAndDrift(t *testing.T) {
	builds, failOn := 0, 0
	factory, err := NewTokenDefinitionFactory(func() TokenDefinition {
		builds++
		power := 1
		if builds == failOn {
			power = 2
		}
		return DefineToken("Citizen Token", TokenPT(power, 1), TokenTypes(core.TypeCreature), TokenSubtypes("Citizen"), TokenColors(core.White))
	})
	if err != nil {
		t.Fatal(err)
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	token := factory()
	effect := CreateTokenCopies(You, ValueAtResolution(Fixed(2)), token)
	if err := ApplyEffect(g, effect, a.PlayerID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	created := g.FilterBattlefield(IsToken)
	if len(created) != 2 {
		t.Fatal("batch count")
	}
	for _, p := range created {
		values, err := g.CaptureCopiableValues(p)
		if err != nil {
			t.Fatal(err)
		}
		card, err := values.Build()
		if err != nil || card.IsToken() || values.HasManaCost() || card.Power() != 1 {
			t.Fatal("token snapshot", err)
		}
	}
	failOn = builds + 2
	if err := ApplyEffect(g, effect, a.PlayerID(), a.PlayerID(), nil); err == nil || len(g.FilterBattlefield(IsToken)) != 2 {
		t.Fatal("builder drift changed battlefield")
	}
}
