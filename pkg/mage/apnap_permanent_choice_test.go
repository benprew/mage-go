package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

type apnapChoicePlayer struct {
	Player
	order *[]string
	check func(string)
}

func (p *apnapChoicePlayer) ChoosePermanent(ps []*Permanent, reason string, _ GameReader) *Permanent {
	*p.order = append(*p.order, p.Name())
	if p.check != nil {
		p.check(reason)
	}
	return ps[len(ps)-1]
}

func TestAPNAPPermanentChoices(t *testing.T) {
	g, baseA, baseB, _ := controllerTestGame(t)
	var order []string
	a := &apnapChoicePlayer{Player: baseA, order: &order}
	b := &apnapChoicePlayer{Player: baseB, order: &order}
	g.SetPlayerAt(0, a)
	g.SetPlayerAt(1, b)
	g.SetActivePlayerIndex(1)
	for _, player := range []Player{a, b} {
		g.PutOnBattlefield(NewArtifact("First", "{2}"), player.PlayerID())
		g.PutOnBattlefield(NewArtifact("Last", "{2}"), player.PlayerID())
	}
	first, second := BindPermanentChoice("first"), BindPermanentChoice("second")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	body := ChoosePermanentsAPNAP(
		PermanentChoiceRequest{Player: You, Subjects: PermanentsControlledBy(You, IsArtifact), Result: first},
		PermanentChoiceRequest{Player: ActivePlayer, Subjects: PermanentsControlledBy(ActivePlayer, IsArtifact), Result: second},
	)
	a.check = func(reason string) {
		if !strings.Contains(reason, "Bob: Last") {
			t.Fatal("earlier choice is absent from the public prompt", reason)
		}
		if _, ok := PermanentChosenIn(second).resolvePermanent(ctx); !ok {
			t.Fatal("previous player's choice is unavailable")
		}
	}
	card := NewArtifact("Choices", "", WithAbility(Activate(Costs(), Sequence(body, ExchangePermanentControl(PermanentChosenIn(first), PermanentChosenIn(second))))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if err := body.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(order) != 2 || order[0] != "Bob" || order[1] != "Alice" {
		t.Fatal("wrong APNAP order", order)
	}
	p, ok := PermanentChosenIn(first).resolvePermanent(ctx)
	if !ok || p.Name() != "Last" {
		t.Fatal("choice lost")
	}
	clone := g.Clone()
	g.PhaseOut(p.ID())
	if _, ok := PermanentChosenIn(first).resolvePermanent(ctx); ok {
		t.Fatal("phased choice was present")
	}
	if cp, ok := PermanentChosenIn(first).resolvePermanent(&EffectContext{Game: clone, Controller: a.PlayerID()}); !ok || cp.ID() != p.ID() {
		t.Fatal("clone lost result")
	}
	g.PhaseIn([]uuid.UUID{p.ID()})
	if current, ok := PermanentChosenIn(first).resolvePermanent(ctx); !ok || current.ID() != p.ID() {
		t.Fatal("phase cycle lost choice")
	}
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	if current, lki := readPermanentInformation(PermanentChosenIn(first), ctx); current != nil || lki == nil {
		t.Fatal("departure LKI lost")
	}
	a.RemoveFromHand(p.ID())
	g.PutOnBattlefield(p.Card, a.PlayerID())
	if _, ok := PermanentChosenIn(first).resolvePermanent(ctx); ok {
		t.Fatal("returned visit inherited group choice")
	}

}

func TestAPNAPPermanentChoicesDeclarations(t *testing.T) {
	first, second := BindPermanentChoice("first"), BindPermanentChoice("second")
	request := PermanentChoiceRequest{Player: You, Subjects: MatchingPermanents(IsArtifact), Result: first}
	for _, body := range []ResolvingEffect{
		ChoosePermanentsAPNAP(),
		ChoosePermanentsAPNAP(PermanentChoiceRequest{Player: You, Subjects: MatchingPermanents(IsArtifact)}),
		ChoosePermanentsAPNAP(request, request),
		ChoosePermanentsAPNAP(PermanentChoiceRequest{Subjects: MatchingPermanents(IsArtifact), Result: first}),
		ChoosePermanentsAPNAP(PermanentChoiceRequest{Player: You, Result: first}),
		Sequence(ChoosePermanentsAPNAP(request), TapSubject(PermanentChosenIn(second))),
		ChoosePermanentsAPNAP(request, PermanentChoiceRequest{Player: You, Subjects: SingletonPermanent(PermanentChosenIn(first)), Result: second}),
	} {
		c := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), body)))
		if err := ValidateCardDefinition(c.Name(), c); err == nil {
			t.Fatal("invalid group accepted", body.Text())
		}
	}
}

func TestAPNAPPermanentChoicesEmptyAndAliases(t *testing.T) {
	g, a, _, _ := controllerTestGame(t)
	for range 2 {
		g.PutOnBattlefield(NewArtifact("Candidate", ""), a.PlayerID())
	}
	order := []string{}
	player := &apnapChoicePlayer{Player: a, order: &order}
	g.SetPlayerAt(0, player)
	first, second := BindPermanentChoice("first"), BindPermanentChoice("second")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	body := ChoosePermanentsAPNAP(
		PermanentChoiceRequest{Player: You, Subjects: MatchingPermanents(IsArtifact), Result: first},
		PermanentChoiceRequest{Player: ActivePlayer, Subjects: MatchingPermanents(IsLand), Result: second},
	)
	if err := ChoosePermanentForPlayer(You, MatchingPermanents(IsArtifact), second).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	player.check = func(string) {
		if _, ok := PermanentChosenIn(second).resolvePermanent(ctx); ok {
			t.Fatal("old group result remains visible")
		}
	}
	if err := body.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := PermanentChosenIn(first).resolvePermanent(ctx); !ok {
		t.Fatal("first result lost")
	}
	if _, ok := PermanentChosenByPlayer(You, first, second).resolvePermanent(ctx); ok {
		t.Fatal("empty latest alias reused old choice")
	}
}
