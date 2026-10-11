package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPlayerPermanentChoiceHistory(t *testing.T) {
	g, a, b, _ := controllerTestGame(t)
	first := g.PutOnBattlefield(NewArtifact("First", ""), a.PlayerID())
	second := g.PutOnBattlefield(NewEnchantment("Second", ""), a.PlayerID())
	old := BindPermanentChoice("old")
	latest := BindPermanentChoice("latest")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	run := func(e ResolvingEffect) {
		t.Helper()
		if err := e.Apply(ctx); err != nil {
			t.Fatal(err)
		}
	}
	run(ChoosePermanentForPlayer(You, MatchingPermanents(IsArtifact), old))
	run(ChoosePermanentForPlayer(You, MatchingPermanents(IsEnchantment), latest))
	ref := PermanentChosenByPlayer(You, old, latest)
	got, ok := ref.resolvePermanent(ctx)
	if !ok || got.ID() != second.ID() {
		t.Fatal("latest matching player choice was lost")
	}
	clone := g.Clone()
	cloneCtx := &EffectContext{Game: clone, Controller: a.PlayerID()}
	g.BouncePermanentToHand(second)
	a.RemoveFromHand(second.ID())
	g.PutOnBattlefield(second.Card, b.PlayerID())
	if _, ok := ref.resolvePermanent(ctx); ok {
		t.Fatal("returned visit inherited choice")
	}
	if got, ok := ref.resolvePermanent(cloneCtx); !ok || got.ID() != second.ID() {
		t.Fatal("clone lost choice")
	}
	run(ChoosePermanentForPlayer(You, MatchingPermanents(IsLand), latest))
	if _, ok := ref.resolvePermanent(ctx); ok {
		t.Fatal("empty choice reused earlier choice")
	}
	if got, ok := From(old.permanent).resolvePermanent(ctx); !ok || got.ID() != first.ID() {
		t.Fatal("earlier binding changed")
	}
}

func TestPlayerPermanentChoiceIdentityAndDeclarations(t *testing.T) {
	g, a, b, _ := controllerTestGame(t)
	card := NewArtifact("Chosen", "{3}")
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	result := BindPermanentChoice("chosen")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	choice := ChoosePermanentForPlayer(You, MatchingPermanents(IsArtifact), result)
	if err := choice.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	ref := PermanentChosenByPlayer(You, result)
	g.PhaseOut(p.ID())
	if _, ok := ref.resolvePermanent(ctx); ok {
		t.Fatal("phased subject was present")
	}
	if current, lki := readPermanentInformation(ref, ctx); current != nil || lki == nil {
		t.Fatal("phase information missing")
	}
	g.PhaseIn([]uuid.UUID{p.ID()})
	g.AddControlEffect(ControlEffectSpec{TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	if got, ok := ref.resolvePermanent(ctx); !ok || got.ControllerID() != b.PlayerID() {
		t.Fatal("choice followed current controller")
	}
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	if current, lki := readPermanentInformation(ref, ctx); current != nil || lki == nil {
		t.Fatal("departure information missing")
	}
	if got := DamageSourceOf(ref).resolveDamageSource(ctx); got.ID() != p.ID() {
		t.Fatal("damage source lost exact departure facts")
	}
	valid := NewArtifact("Valid", "", WithAbility(Activate(Costs(), Sequence(choice, TapSubject(ref)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, effect := range []ResolvingEffect{
		TapSubject(PermanentChosenByPlayer(You)),
		TapSubject(PermanentChosenByPlayer(You, result)),
		ChoosePermanentForPlayer(You, MatchingPermanents(IsArtifact), nil),
		Sequence(choice, TapSubject(PermanentChosenByPlayer(You, nil))),
	} {
		invalid := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))
		if ValidateCardDefinition(invalid.Name(), invalid) == nil {
			t.Fatal("invalid choice declaration accepted")
		}
	}
}
