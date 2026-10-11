package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type requiredGraveyardPlayer struct {
	*BasePlayer
	pick            Card
	choices, offers int
	mutate          func()
}

func (p *requiredGraveyardPlayer) ChooseMayAbility(string) bool { p.offers++; return false }
func (p *requiredGraveyardPlayer) ChooseCardFromGraveyard(cards []Card, _ string, _ GameReader) Card {
	p.choices++
	if p.mutate != nil {
		p.mutate()
	}
	return p.pick
}

func TestChosenGraveyardReturnContracts(t *testing.T) {
	for _, scenario := range []string{"normal", "nil-choice", "foreign-choice", "empty", "returned-visit", "departed-choice", "owner-entry"} {
		t.Run(scenario, func(t *testing.T) {
			a := &requiredGraveyardPlayer{BasePlayer: NewBasePlayer("A")}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			first := NewCreature("First", "", 1, 2)
			first.SetOwner(b.PlayerID())
			second := NewCreature("Second", "", 2, 3)
			second.SetOwner(b.PlayerID())
			artifact := NewArtifact("Excluded", "")
			artifact.SetOwner(b.PlayerID())
			b.AddToGraveyard(artifact)
			if scenario != "empty" {
				b.AddToGraveyard(first)
				b.AddToGraveyard(second)
			}
			a.pick = second
			if scenario == "nil-choice" {
				a.pick = nil
			}
			if scenario == "foreign-choice" {
				a.pick = artifact
			}
			if scenario == "returned-visit" || scenario == "departed-choice" {
				a.mutate = func() {
					c, _, ok := g.MoveFromAnyGraveyard(second.ID(), core.ZoneHand)
					if !ok {
						t.Fatal("choice did not leave")
					}
					b.AddToHand(c)
					if scenario == "returned-visit" {
						b.RemoveFromHand(c.ID())
						b.AddToGraveyard(c)
					}
				}
			}
			graveyard := BindPlayer("graveyard")
			g.resolution.playerBindings = map[*PlayerBinding]uuid.UUID{graveyard: b.PlayerID()}
			var effect = ReturnChosenGraveyardCard(FromPlayer(graveyard), You, IsCreatureCard)
			if scenario == "owner-entry" {
				effect = ReturnChosenGraveyardCardToOwner(FromPlayer(graveyard), IsCreatureCard)
			}
			if err := effect.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			want := second.ID()
			if scenario == "nil-choice" || scenario == "foreign-choice" {
				want = first.ID()
			}
			moved := g.FindPermanent(want)
			if scenario == "empty" || scenario == "returned-visit" || scenario == "departed-choice" {
				if moved != nil {
					t.Fatal("absent or stale choice returned")
				}
			} else {
				if moved == nil {
					t.Fatal("required choice did not return")
				}
				controller := a.PlayerID()
				if scenario == "owner-entry" {
					controller = b.PlayerID()
				}
				if moved.ControllerID() != controller || moved.Card.Owner() != b.PlayerID() {
					t.Fatal("wrong owner or controller")
				}
			}
			if a.offers != 0 {
				t.Fatal("required choice offered optional payment")
			}
			if g.FindPermanent(artifact.ID()) != nil {
				t.Fatal("noncreature returned")
			}
		})
	}
}

func TestChosenGraveyardReturnDefinition(t *testing.T) {
	for _, effect := range []ResolvingEffect{
		ReturnChosenGraveyardCard(nil, You, IsCreatureCard),
		ReturnChosenGraveyardCard(You, nil, IsCreatureCard),
		ReturnChosenGraveyardCardToOwner(nil, IsCreatureCard),
	} {
		card := NewInstant("Invalid", "", Spell(effect))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid return accepted")
		}
	}
}

func TestChosenGraveyardReturnWaitingSource(t *testing.T) {
	for _, body := range []ResolvingEffect{
		ReturnChosenGraveyardCard(ControllerOfWithLKI(This), You, IsCreatureCard),
		ReturnChosenGraveyardCard(You, ControllerOfWithLKI(This), IsCreatureCard),
		ReturnChosenGraveyardCardToOwner(ControllerOfWithLKI(This), IsCreatureCard),
	} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		card := NewCreature("Source", "", 1, 2)
		card.SetOwner(a.PlayerID())
		source := g.PutOnBattlefield(card, a.PlayerID())
		saved := g.CaptureDamageSource(source.ID())
		if err := ApplyEffect(g, AtNext(EachEndStep, body), source.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		g.PhaseOut(source.ID())
		refs := g.WaitingDelayedPermanentSources()
		if len(refs) != 1 || !refs[0].SameObject(saved) {
			t.Fatal("waiting query lost source identity")
		}
		g.PhaseIn([]uuid.UUID{source.ID()})
		g.BouncePermanentToHand(g.FindPermanent(source.ID()))
		a.RemoveFromHand(card.ID())
		g.PutOnBattlefield(card, b.PlayerID())
		refs = g.WaitingDelayedPermanentSources()
		if len(refs) != 1 || !refs[0].SameObject(saved) {
			t.Fatal("waiting query followed returned source")
		}
		candidate := NewCreature("Candidate", "", 2, 3)
		candidate.SetOwner(a.PlayerID())
		a.AddToGraveyard(candidate)
		clone := g.Clone()
		clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		clone.ResolveStack()
		if clone.FindPermanent(candidate.ID()) == nil || g.FindPermanent(candidate.ID()) != nil {
			t.Fatal("delayed return lost LKI or clone isolation")
		}
		g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		g.ResolveStack()
		if g.FindPermanent(candidate.ID()) == nil || g.FindPermanent(candidate.ID()).ControllerID() != a.PlayerID() {
			t.Fatal("returned source supplied controller")
		}
	}
}
