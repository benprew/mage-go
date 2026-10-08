package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type playerChoiceTester struct {
	Player
	answer []uuid.UUID
	calls  int
}

func (p *playerChoiceTester) ChooseTargets(_ []uuid.UUID, minimum, maximum int, _ *Game) []uuid.UUID {
	if minimum != 1 || maximum != 1 {
		panic("incorrect choice bounds")
	}
	p.calls++
	return p.answer
}

func TestPlayerChoiceResolution(t *testing.T) {
	for _, response := range []string{"valid", "nil", "invalid", "multiple"} {
		t.Run(response, func(t *testing.T) {
			base, b := NewBasePlayer("A"), NewBasePlayer("B")
			a := &playerChoiceTester{Player: base, answer: []uuid.UUID{b.PlayerID()}}
			g := NewGame(a, b)
			switch response {
			case "nil":
				a.answer = nil
			case "invalid":
				a.answer = []uuid.UUID{uuid.New()}
			case "multiple":
				a.answer = []uuid.UUID{b.PlayerID(), base.PlayerID()}
			}
			saved := BindPlayer("chosen")
			if err := ApplyEffect(g, Sequence(ChoosePlayer(You, EachPlayer, saved), GainLifeFor(FromPlayer(saved), Fixed(2))), uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			wantA, wantB := 22, 20
			if response == "valid" {
				wantA, wantB = 20, 22
			}
			if a.Life() != wantA || b.Life() != wantB || a.calls != 1 {
				t.Fatal("incorrect player decision")
			}
			if _, ok := FromPlayer(saved).resolvePlayer(&EffectContext{Game: g}); ok {
				t.Fatal("choice escaped resolution")
			}
		})
	}
}

func TestPlayerChoiceAbsenceAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	saved := BindPlayer("chosen")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := ChoosePlayer(You, OpponentsOf(You), saved).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	cloneCtx := &EffectContext{Game: clone, Controller: a.PlayerID()}
	if err := ChoosePlayer(You, PlayersWhoCastSorceryThisTurn(), saved).Apply(cloneCtx); err != nil {
		t.Fatal(err)
	}
	if _, ok := FromPlayer(saved).resolvePlayer(cloneCtx); ok {
		t.Fatal("empty choice retained old result")
	}
	if id, ok := FromPlayer(saved).resolvePlayer(ctx); !ok || id != b.PlayerID() {
		t.Fatal("clone changed original")
	}
	if err := ChoosePlayer(ControllerOf(This), EachPlayer, saved).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := FromPlayer(saved).resolvePlayer(ctx); ok {
		t.Fatal("absent chooser retained old result")
	}
}

func TestPlayerChoiceDeclaration(t *testing.T) {
	saved := BindPlayer("chosen")
	for _, body := range []Effect{
		ChoosePlayer(nil, EachPlayer, saved), ChoosePlayer(You, nil, saved), ChoosePlayer(You, EachPlayer, nil),
		Sequence(GainLifeFor(FromPlayer(saved), Fixed(1)), ChoosePlayer(You, EachPlayer, saved)),
	} {
		card := NewInstant("Invalid", "", WithAbility(Spell(body)))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
	card := NewInstant("Valid", "", WithAbility(Spell(Sequence(ChoosePlayer(You, PlayersWhoCastSorceryThisTurn(), saved), GainLifeFor(FromPlayer(saved), Fixed(1))))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestPlayerChoiceWaitingSource(t *testing.T) {
	for _, operand := range []string{"chooser", "subjects"} {
		t.Run(operand, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewArtifact("Source", "")
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			original := g.CaptureDamageSource(p.ID())
			chooser := PlayerRef(You)
			subjects := PlayerSet(EachPlayer)
			if operand == "chooser" {
				chooser = ControllerOfWithLKI(This)
			} else {
				subjects = OpponentsOf(ControllerOfWithLKI(This))
			}
			binding := BindPlayer("choice")
			body := Sequence(ChoosePlayer(chooser, subjects, binding), GainLifeFor(FromPlayer(binding), Fixed(1)))
			if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.BouncePermanentToHand(p)
			a.RemoveFromHand(card.ID())
			g.PutOnBattlefield(card, b.PlayerID())
			for range 2 {
				got := g.WaitingDelayedPermanentSources()
				if len(got) != 1 || !got[0].SameObject(original) || a.Life() != 20 || b.Life() != 20 || len(g.resolution.playerBindings) != 0 {
					t.Fatal("query executed choice or lost source identity")
				}
			}
			clone := g.Clone()
			clone.SetPlayerAt(0, &playerChoiceTester{Player: clone.GetPlayer(a.PlayerID()), answer: []uuid.UUID{a.PlayerID()}})
			clone.SetPlayerAt(1, &playerChoiceTester{Player: clone.GetPlayer(b.PlayerID()), answer: []uuid.UUID{b.PlayerID()}})
			clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
			clone.PutTriggersOnStack()
			clone.ResolveStack()
			wantA, wantB := 21, 20
			if operand == "subjects" {
				wantA, wantB = 20, 21
			}
			if clone.GetPlayer(a.PlayerID()).Life() != wantA || clone.GetPlayer(b.PlayerID()).Life() != wantB || len(clone.WaitingDelayedPermanentSources()) != 0 || len(g.WaitingDelayedPermanentSources()) != 1 {
				t.Fatalf("delayed choice lost LKI or clone lifetime: life=%d/%d want=%d/%d waiting=%d/%d", clone.GetPlayer(a.PlayerID()).Life(), clone.GetPlayer(b.PlayerID()).Life(), wantA, wantB, len(clone.WaitingDelayedPermanentSources()), len(g.WaitingDelayedPermanentSources()))
			}
		})
	}
}
