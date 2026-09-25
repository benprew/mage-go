package mage

import (
	"strings"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestComposableKeywordRemoval(t *testing.T) {
	for _, keyword := range []Attr{Forestwalk, Flying, Banding} {
		for _, printed := range []bool{false, true} {
			t.Run(keyword.String()+map[bool]string{false: "/granted", true: "/printed"}[printed], func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				card := NewCreature("Recipient", "", 2, 2, WithKeyword(Vigilance))
				if printed {
					card = NewCreature("Recipient", "", 2, 2, WithKeyword(Vigilance), WithKeyword(keyword))
				}
				g.PutOnBattlefield(card, a.PlayerID())
				grant := Until(EndOfTurn, GrantKeywordModification(This, keyword))
				apply := func(effect ResolvingEffect) {
					t.Helper()
					if err := ApplyEffect(g, effect, card.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				}
				if !printed {
					apply(grant)
					apply(grant)
				}
				removal := Until(EndOfTurn, RemoveKeyword(This, keyword))
				apply(removal)
				if g.FindPermanent(card.ID()).HasKeyword(keyword) || !g.FindPermanent(card.ID()).HasKeyword(Vigilance) {
					t.Fatal("removal must remove all instances of only the selected keyword")
				}
				apply(grant)
				if !g.FindPermanent(card.ID()).HasKeyword(keyword) {
					t.Fatal("later grant must restore the keyword")
				}
				apply(removal)
				if g.FindPermanent(card.ID()).HasKeyword(keyword) {
					t.Fatal("later removal must remove the keyword")
				}
				g.effects.RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				if g.FindPermanent(card.ID()).HasKeyword(keyword) != printed {
					t.Fatal("cleanup must restore printed abilities")
				}
			})
		}
	}
}

func TestComposableKeywordRemovalLifetime(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewArtifact("Source", "")
	card := NewCreature("Recipient", "", 2, 2, WithKeyword(Forestwalk))
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(card, b.PlayerID())
	target := BindPermanent("creature")
	body := Sequence(ChooseRandomPermanent(MatchingPermanents(IsCreature), target), Until(EndOfTurn, RemoveKeyword(From(target), Forestwalk)))
	if err := ApplyEffect(g, body, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, b.PlayerID())
	g.PhaseIn(g.PhaseOut(card.ID()))
	g.ApplyContinuousEffects()
	if g.FindPermanent(card.ID()).HasKeyword(Forestwalk) {
		t.Fatal("source departure and recipient phasing must preserve removal")
	}
	clone := g.Clone()
	clone.effects.RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	if !clone.FindPermanent(card.ID()).HasKeyword(Forestwalk) || g.FindPermanent(card.ID()).HasKeyword(Forestwalk) {
		t.Fatal("cleanup must affect only the clone")
	}
	g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
	g.ApplyContinuousEffects()
	g.PutOnBattlefield(card, b.PlayerID())
	g.ApplyContinuousEffects()
	if !g.FindPermanent(card.ID()).HasKeyword(Forestwalk) {
		t.Fatal("removal must not affect a returned incarnation")
	}
}

func TestComposableKeywordRemovalMetadata(t *testing.T) {
	target := DeclarePermanentTarget("creature", IsCreature)
	body := Until(EndOfTurn, RemoveKeyword(target, Forestwalk))
	info := InspectDefinition(body)
	if info.Properties.Outcome != OutcomeDetriment || info.Properties.GrantedKeyword != 0 || !strings.Contains(info.Text, "remove Forestwalk") {
		t.Fatalf("removal metadata = %+v", info)
	}
	if _, ok := body.(*untilEffect).KeywordGrantTargetIndex([]Target{target.targetDeclaration}); ok {
		t.Fatal("removal is not a keyword grant")
	}
	if err := ValidateCardDefinition("Unbound", NewInstant("Unbound", "", Spell(body))); err == nil {
		t.Fatal("undeclared removal target must fail validation")
	}
}
