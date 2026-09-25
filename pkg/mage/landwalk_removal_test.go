package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestRevokeLandwalkIncludesLegendary(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Walker", "", 2, 2, WithKeyword(LegendaryLandwalk), WithKeyword(Flying))
	g.PutOnBattlefield(card, a.PlayerID())
	if err := ApplyEffect(g, RevokeLandwalk().Targeting(ToSource()), card.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(card.ID()).HasKeyword(LegendaryLandwalk) {
		t.Fatal("all landwalk removal must include legendary landwalk")
	}
	if !g.FindPermanent(card.ID()).HasKeyword(Flying) {
		t.Fatal("unrelated abilities must remain")
	}
}

func TestComposableLandwalkRemoval(t *testing.T) {
	for _, keyword := range []Attr{Forestwalk, Islandwalk, Swampwalk, Mountainwalk, Plainswalk, Desertwalk, LegendaryLandwalk} {
		for _, printed := range []bool{false, true} {
			t.Run(keyword.String()+map[bool]string{false: "/granted", true: "/printed"}[printed], func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				card := NewCreature("Walker", "", 2, 2, WithKeyword(Flying))
				if printed {
					card = NewCreature("Walker", "", 2, 2, WithKeyword(Flying), WithKeyword(keyword))
				}
				g.PutOnBattlefield(card, a.PlayerID())
				apply := func(effect ResolvingEffect) {
					t.Helper()
					if err := ApplyEffect(g, effect, card.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				}
				grant := Until(EndOfTurn, GrantKeywordModification(This, keyword))
				if !printed {
					apply(grant)
					apply(grant)
				}
				removal := Until(EndOfTurn, RemoveLandwalk(This))
				apply(removal)
				if g.FindPermanent(card.ID()).HasKeyword(keyword) || !g.FindPermanent(card.ID()).HasKeyword(Flying) {
					t.Fatal("remove every landwalk instance and preserve flying")
				}
				apply(grant)
				if !g.FindPermanent(card.ID()).HasKeyword(keyword) {
					t.Fatal("a later grant must restore landwalk")
				}
				apply(removal)
				if g.FindPermanent(card.ID()).HasKeyword(keyword) {
					t.Fatal("a later removal must remove landwalk")
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
func TestComposableLandwalkRemovalLifetime(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewArtifact("Source", "")
	card := NewCreature("Recipient", "", 2, 2, WithKeyword(LegendaryLandwalk))
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(card, b.PlayerID())
	target := BindPermanent("creature")
	body := Sequence(ChooseRandomPermanent(MatchingPermanents(IsCreature), target), Until(EndOfTurn, RemoveLandwalk(From(target))))
	if err := ApplyEffect(g, body, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, b.PlayerID())
	g.PhaseIn(g.PhaseOut(card.ID()))
	g.ApplyContinuousEffects()
	if g.FindPermanent(card.ID()).HasKeyword(LegendaryLandwalk) {
		t.Fatal("source departure and recipient phasing must preserve removal")
	}
	clone := g.Clone()
	clone.effects.RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	if !clone.FindPermanent(card.ID()).HasKeyword(LegendaryLandwalk) || g.FindPermanent(card.ID()).HasKeyword(LegendaryLandwalk) {
		t.Fatal("cleanup must affect only the clone")
	}
	g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
	g.ApplyContinuousEffects()
	g.PutOnBattlefield(card, b.PlayerID())
	g.ApplyContinuousEffects()
	if !g.FindPermanent(card.ID()).HasKeyword(LegendaryLandwalk) {
		t.Fatal("removal must not affect a returned incarnation")
	}
}

func TestComposableLandwalkRemovalMetadata(t *testing.T) {
	target := DeclarePermanentTarget("creature", IsCreature)
	body := Until(EndOfTurn, RemoveLandwalk(target))
	info := InspectDefinition(body)
	if info.Properties.Outcome != OutcomeDetriment || info.Properties.GrantedKeyword != 0 {
		t.Fatalf("removal metadata = %+v", info)
	}
	if _, ok := body.(*untilEffect).KeywordGrantTargetIndex([]Target{target.targetDeclaration}); ok {
		t.Fatal("landwalk removal is not a keyword grant")
	}
	if err := ValidateCardDefinition("Unbound", NewInstant("Unbound", "", Spell(body))); err == nil {
		t.Fatal("undeclared target must fail validation")
	}
}
