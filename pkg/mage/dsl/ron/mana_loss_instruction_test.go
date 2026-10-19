package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONManaLossInstructionRealCards(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, state := range []string{"normal", "empty", "copy", "clone", "phase", "returned", "unpaid", "invalid target"} {
			t.Run(set+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				land := g.PutOnBattlefield(mage.NewLand("No mana"), b.PlayerID())
				other := g.PutOnBattlefield(mage.NewLand("Other"), a.PlayerID())
				nonland := g.PutOnBattlefield(mage.NewCreature("Mana creature", "", 1, 1, mage.WithManaAbility(core.Green)), b.PlayerID())
				card := repositoryCard(t, set, "Mana Short")
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				b.ManaPool().AddRestricted(core.Red, 4, core.ArtifactSpellsOnly{})
				if state != "unpaid" {
					a.ManaPool().Add(core.Blue, 3)
				}
				target := b.PlayerID()
				if state == "invalid target" {
					target = uuid.New()
				}
				err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, 0)
				if state == "unpaid" || state == "invalid target" {
					if err == nil || g.StackSize() != 0 || len(a.Hand()) != 1 || b.ManaPool().TotalMana() != 4 || g.FindPermanent(land.ID()).Tapped {
						t.Fatal("failed cast changed state")
					}
					if state == "invalid target" && a.ManaPool().TotalMana() != 3 {
						t.Fatal("invalid target spent mana")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				a.ManaPool().Add(core.Green, 2)
				if state == "empty" {
					b.ManaPool().Clear()
				}
				if state == "phase" {
					g.PhaseOut(land.ID())
				}
				if state == "returned" {
					g.BouncePermanentToHand(g.FindPermanent(land.ID()))
					b.RemoveFromHand(land.Card.ID())
					g.PutOnBattlefield(land.Card, b.PlayerID())
				}
				obj := g.StackPeek()
				if state == "copy" {
					obj = g.CopyStackObjectDirect(obj, a.PlayerID(), false)
					if obj == nil {
						t.Fatal("copy failed")
					}
					obj.Targets[0] = a.PlayerID()
				}
				mana := b.ManaPool().TotalMana()
				for range 2 {
					if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
						t.Fatal("Mana Short has no pure complete-tree result")
					}
				}
				if b.ManaPool().TotalMana() != mana || a.ManaPool().TotalMana() != 2 || g.FindPermanentIncludingPhased(land.ID()).Tapped {
					t.Fatal("query resolved tap or mana loss")
				}
				if state == "clone" {
					clone := g.Clone()
					clone.ResolveTopOfStack()
					if clone.GetPlayer(b.PlayerID()).ManaPool().TotalMana() != 0 || b.ManaPool().TotalMana() != mana {
						t.Fatal("clone crossed mana state")
					}
				}
				g.ResolveTopOfStack()
				if state == "copy" {
					if a.ManaPool().TotalMana() != 0 || b.ManaPool().TotalMana() != mana || !g.FindPermanent(other.ID()).Tapped || g.FindPermanent(land.ID()).Tapped {
						t.Fatal("copy did not use its new player target")
					}
					g.ResolveStack()
				} else if a.ManaPool().TotalMana() != 2 || g.FindPermanent(other.ID()).Tapped {
					t.Fatal("wrong player's state changed")
				}
				if b.ManaPool().TotalMana() != 0 || b.Life() != 20 || len(b.ManaPool().ProducedThisTurn) != 4 || g.FindPermanent(nonland.ID()).Tapped {
					t.Fatal("wrong pool, life, history or permanent type")
				}
				if g.FindPermanentIncludingPhased(land.ID()).Tapped != (state != "phase") {
					t.Fatal("land visit or phasing selection differs")
				}
			})
		}
	}
}

func TestRONManaLossInstructionScopes(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerAction(Ref_TargetPlayer,LoseUnspentMana)])))`)
	g, a, b := ronGame()
	first := castRONSpell(t, g, a, factory(), b.PlayerID())
	second := castRONSpell(t, g, a, factory(), a.PlayerID())
	subjects := make([]mage.PlayerRef, 0, 2)
	for _, obj := range []*mage.StackObject{first, second} {
		instructions, ok := mage.InspectManaLossInstructions(mage.Sequence(obj.Effects...))
		if !ok || len(instructions) != 1 {
			t.Fatal("RON lost native mana-loss declaration")
		}
		subjects = append(subjects, instructions[0].Subject)
		if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
			t.Fatal("player target scope unknown")
		}
	}
	if subjects[0] == subjects[1] {
		t.Fatal("factories share target declarations")
	}
	a.ManaPool().Add(core.Blue, 2)
	b.ManaPool().Add(core.Red, 3)
	g.ResolveTopOfStack()
	if a.ManaPool().TotalMana() != 0 || b.ManaPool().TotalMana() != 3 {
		t.Fatal("top spell used another instance binding")
	}
	g.ResolveStack()
	if b.ManaPool().TotalMana() != 0 {
		t.Fatal("lower spell lost binding")
	}
}

func TestRONManaLossInstructionInvalid(t *testing.T) {
	for _, body := range []string{`LoseUnspentMana()`, `LoseUnspentMana(You)`, `PlayerAction(Ref_TargetPlayer,LoseUnspentMana)`, `PlayerAction(ThisPermanent,LoseUnspentMana)`, `PlayerAction(ActionPlayer,LoseUnspentMana)`, `PlayerActions(You,[LoseUnspentMana]),PlayerAction(ActionPlayer,LoseUnspentMana)`} {
		if _, err := Compile(nameScopeNode(t, `Card(Name:"Invalid mana loss",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([`+body+`]))])`)); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}
