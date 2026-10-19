package ron

import (
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const faceUpDomainAction = `CreateReplaceWouldMaskUntil(PermanentWouldAssignDamageDealDamageBeDealDamageOrBecomeTapped(ThePermanentSpellResolvedThisWay),[TurnItFaceUpInstead],UntilPermanentIsTurnedFaceUp(ThePermanentSpellResolvedThisWay))`

func TestRONFaceUpDestructionDomainPlan(t *testing.T) {
	result := &castResultPlan{name: "saved cast"}
	ctx := scope{resolving: true, castResult: result}
	nodes, err := Parse(strings.NewReader(faceUpDomainAction), "plan.ron")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := compileFaceUpReplacementPlan(nodes[0], ctx)
	if err != nil {
		t.Fatal(err)
	}
	bindings := []*mage.DamageSourceBinding{mage.BindDamageSource("first"), mage.BindDamageSource("second")}
	effects := make([]mage.Effect, 2)
	for i, binding := range bindings {
		effects[i] = plan.build(instanceBindings{castResults: map[*castResultPlan]*mage.DamageSourceBinding{result: binding}})
	}
	g, a, _ := ronGame()
	g.SetPlayerAt(0, &ronOptionalCastPlayer{BasePlayer: a, answers: []bool{true, true}})
	ids := make([]uuid.UUID, 0, len(effects))
	for i, effect := range effects {
		card := mage.NewCreature("Hidden", "", 5, 6)
		a.AddToHand(card)
		hand := mage.BindCard("hand")
		body := mage.Sequence(mage.CastHandCardFaceDownWithoutPayingInto(mage.You, mage.FromCard(hand), bindings[i]), effect)
		if err := mage.ApplyEffect(g, mage.ChooseHandCardOrElse(mage.You, mage.AnyCardFilter(), hand, body, mage.GainLifeFor(mage.You, mage.Fixed(9))), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		p := g.FindPermanent(card.ID())
		if p == nil || !p.FaceDown {
			t.Fatal("plan did not retain cast producer")
		}
		ids = append(ids, p.ID())
	}
	for _, id := range ids {
		g.TapPermanent(g.FindPermanent(id))
		if g.FindPermanent(id).FaceDown {
			t.Fatal("plan reused another instance binding")
		}
	}
	for _, bad := range []string{
		strings.Replace(faceUpDomainAction, "CreateReplaceWouldMaskUntil", "Unknown", 1),
		strings.Replace(faceUpDomainAction, "[TurnItFaceUpInstead]", "[TurnItFaceUpInstead()]", 1),
		strings.Replace(faceUpDomainAction, "[TurnItFaceUpInstead]", "[]", 1),
		strings.Replace(faceUpDomainAction, "UntilPermanentIsTurnedFaceUp(ThePermanentSpellResolvedThisWay)", "UntilEndOfTurn", 1),
		strings.Replace(faceUpDomainAction, "ThePermanentSpellResolvedThisWay", "ThisPermanent", 1),
	} {
		nodes, err := Parse(strings.NewReader(bad), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := compileFaceUpReplacementPlan(nodes[0], ctx); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
	for _, badCtx := range []scope{{}, {castResult: result}, {resolving: true}, {resolving: true, castResult: result, enteringSource: true}, {resolving: true, castResult: result, drawReplacementBody: true}} {
		if _, err := compileFaceUpReplacementPlan(nodes[0], badCtx); err == nil {
			t.Fatal("accepted invalid context")
		}
	}
}

func TestRONFaceUpDestructionDomain(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI"} {
		for _, name := range []string{"Stone Rain", "Wrath of God"} {
			for _, state := range []string{"down", "source left", "phase cycle", "regeneration"} {
				t.Run(set+"/"+name+"/"+state, func(t *testing.T) {
					g, a, b := ronGame()
					g.SetPlayerAt(0, &ronOptionalCastPlayer{BasePlayer: a, answers: []bool{true, true}})
					mask := g.PutOnBattlefield(repositoryCard(t, set, "Illusionary Mask"), a.PlayerID())
					card := mage.NewCreature("Hidden", "{U}", 5, 6)
					a.AddToHand(card)
					a.ManaPool().Add(core.Blue, 1)
					g.SetXValue(1)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), mask.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					g.ResolveStack()
					p := g.FindPermanent(card.ID())
					if p == nil || !p.FaceDown {
						t.Fatal("missing face-down result")
					}
					if state == "source left" {
						g.BouncePermanentToHand(mask)
					}
					if state == "phase cycle" {
						g.PhaseIn(g.PhaseOut(p.ID()))
					}
					if name == "Stone Rain" {
						if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.AddCardType(mage.This, core.TypeLand)), p.ID(), a.PlayerID(), nil); err != nil {
							t.Fatal(err)
						}
					}
					if state == "regeneration" {
						g.AddRegenerationShield(p.ID())
					}
					var spell *mage.StackObject
					if name == "Stone Rain" {
						spell = castRONSpell(t, g, a, repositoryCard(t, "LEA", name), p.ID())
					} else {
						spell = castRONSpell(t, g, a, repositoryCard(t, "LEA", name))
					}
					instructions, known := mage.InspectDirectDestruction(spell.Effects[0])
					if !known || len(instructions) != 1 {
						t.Fatal("missing direct destruction")
					}
					instruction := instructions[0]
					var visits []mage.PermanentVisit
					if instruction.Subject != nil {
						query, ok := mage.InspectPermanentTarget(instruction.Subject)
						if !ok {
							t.Fatal("missing target plan")
						}
						visits, known = query.Select(g, spell)
						if !known {
							t.Fatal("unknown target legality")
						}
					} else {
						query, ok := mage.InspectPermanentCollection(instruction.Subjects)
						if !ok {
							t.Fatal("missing collection plan")
						}
						visits = query.Select(g)
					}
					if len(visits) != 1 || visits[0].ID != p.ID() {
						t.Fatal("wrong selected visit")
					}
					want := mage.DestructionToGraveyard
					if state == "regeneration" && !instruction.CannotRegenerate {
						want = mage.DestructionNoMove
					}
					for _, game := range []*mage.Game{g.Clone(), g} {
						revealCount, stackSize := len(game.PublicReveals()), game.StackSize()
						for range 2 {
							if got, known := game.InspectDestructionDestinations(visits[0], instruction.CannotRegenerate); !known || got != want {
								t.Fatalf("destinations %v known %v, want %v", got, known, want)
							}
							if proceeds, known := game.InspectDestructionProposal(visits[0], instruction.CannotRegenerate); !known || proceeds != (want == mage.DestructionToGraveyard) {
								t.Fatal("proposal proof disagrees")
							}
						}
						live := game.FindPermanent(p.ID())
						if live == nil || !live.FaceDown || live.Tapped || game.HasRegenerationShield(p.ID()) != (state == "regeneration") || game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 20 || len(game.PublicReveals()) != revealCount || game.StackSize() != stackSize {
							t.Fatal("inspection changed state")
						}
						game.ResolveStack()
						live = game.FindPermanent(p.ID())
						if want == mage.DestructionNoMove {
							if live == nil || live.FaceDown || !live.Tapped || game.HasRegenerationShield(p.ID()) {
								t.Fatal("native regeneration must turn up during tapping")
							}
						} else if live != nil || !slices.ContainsFunc(game.GetPlayer(a.PlayerID()).Graveyard(), func(c mage.Card) bool { return c.ID() == card.ID() }) {
							t.Fatal("native destruction disagrees")
						}
						if want == mage.DestructionToGraveyard && !slices.ContainsFunc(game.PublicReveals(), func(r mage.PublicCardReveal) bool { return r.CardID == card.ID() && r.Name == "Hidden" }) {
							t.Fatal("face-down departure did not reveal the physical card")
						}
					}
				})
			}
		}
	}
}
