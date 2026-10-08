package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type waitingTokenCount struct{ reads *int }

func (v waitingTokenCount) Resolve(GameReader, uuid.UUID, uuid.UUID, []uuid.UUID) int {
	*v.reads++
	return 2
}

func (waitingTokenCount) Text() string { return "token count" }

func TestWaitingDelayedTokenPurity(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	p := g.PutOnBattlefield(NewCreature("Source", "", 2, 4), a.PlayerID())
	reads := 0
	token := DefineToken("Token", TokenPT(1, 1), TokenTypes(core.TypeCreature))
	body := CreateTokenCopies(ControllerOf(This), ValueAtResolution(waitingTokenCount{&reads}), token)
	if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || reads != 0 || len(g.FilterBattlefield(IsToken)) != 0 {
			t.Fatal("query evaluated token creation")
		}
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if reads != 1 || len(g.FilterBattlefield(IsToken)) != 2 {
		t.Fatal("token count did not run once at resolution")
	}
}

func TestWaitingDelayedTokenOperands(t *testing.T) {
	token := DefineToken("Token", TokenPT(1, 1), TokenTypes(core.TypeCreature))
	for name, body := range map[string]ResolvingEffect{
		"player":       CreateTokenCopies(ControllerOfWithLKI(This), Fixed(1), token),
		"count":        CreateTokenCopies(You, ManaValueOfWithLKI(This), token),
		"zero":         CreateTokenCopies(You, MultiplyValues(Fixed(0), PowerOfWithLKI(This)), token),
		"one":          CreateOneToken(ControllerOfWithLKI(This), token, BindPermanent("created")),
		"batch player": CreateTokenCopiesInto(ControllerOfWithLKI(This), Fixed(1), token, BindCreatedTokens("created")),
		"batch count":  CreateTokenCopiesInto(You, ManaValueOfWithLKI(This), token, BindCreatedTokens("created")),
	} {
		t.Run(name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Source", "{U}{U}", 2, 4)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			delay := AtNext(EachEndStep, body)
			if err := ValidateCardDefinition("Source", NewArtifact("Source", "", WithAbility(Activate(Costs(), delay)))); err != nil {
				t.Fatal(err)
			}
			if err := ApplyEffect(g, delay, p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			check := func() {
				t.Helper()
				got := g.WaitingDelayedPermanentSources()
				if len(got) != 1 || !got[0].SameObject(saved) || got[0].HasColor(core.Red) || len(g.FilterBattlefield(IsToken)) != 0 {
					t.Fatal("token operands lost saved source or query created tokens")
				}
			}
			check()
			phased := g.PhaseOut(p.ID())
			check()
			g.PhaseIn(phased)
			g.BouncePermanentToHand(p)
			a.RemoveFromHand(card.ID())
			g.PutOnBattlefield(card, b.PlayerID())
			g.ChangeObjectColor(card.ID(), core.Red)
			check()
			clone := g.Clone()
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			if len(g.WaitingDelayedPermanentSources()) != 0 || len(clone.WaitingDelayedPermanentSources()) != 1 {
				t.Fatal("clone shares waiting lifetime")
			}
			g.PutTriggersOnStack()
			g.ResolveStack()
			want := 1
			if name == "count" || name == "batch count" {
				want = 2
			}
			if name == "zero" {
				want = 0
			}
			tokens := g.FilterBattlefield(IsToken)
			if len(tokens) != want || len(clone.FilterBattlefield(IsToken)) != 0 {
				t.Fatal("wrong token count or shared clone state")
			}
			for _, token := range tokens {
				if token.ControllerID() != a.PlayerID() || token.Card.Owner() != a.PlayerID() {
					t.Fatal("token followed returned controller")
				}
			}
		})
	}
}

func TestWaitingDelayedTokenScope(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	p := g.PutOnBattlefield(NewCreature("Source", "", 2, 4), a.PlayerID())
	token := DefineToken("Token", TokenPT(1, 1), TokenTypes(core.TypeCreature), TokenAbilities(WhenEvent(DealsDamageToPlayer(This, BindPlayer("damaged")), GainLifeFor(You, PowerOfWithLKI(This)))))
	for _, body := range []ResolvingEffect{
		CreateTokenCopies(You, Fixed(0), token),
		CreateOneToken(ControllerOf(AttachedPermanent), token, BindPermanent("created")),
		AtNext(EachEndStep, CreateTokenCopies(ControllerOf(This), Fixed(1), token)),
	} {
		if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(g.WaitingDelayedPermanentSources()) != 0 {
		t.Fatal("token ability, implicit attachment or future delay inferred producer source")
	}
}
