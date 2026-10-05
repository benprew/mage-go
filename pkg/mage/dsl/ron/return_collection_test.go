package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONReturnCollectionRecall(t *testing.T) {
	for _, change := range []string{"none", "control", "phase", "phase-cycle", "return", "late-entry", "type-change", "shroud", "token", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronModalGame(0)
			card := repositoryCard(t, "ATQ", "Hurkyl's Recall")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			owned := mage.NewArtifact("Owned", "")
			owned.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(owned, a.PlayerID())
			other := mage.NewArtifact("Other", "")
			other.SetOwner(a.PlayerID())
			q := g.PutOnBattlefield(other, b.PlayerID())
			land := mage.NewLand("Land")
			land.SetOwner(b.PlayerID())
			r := g.PutOnBattlefield(land, b.PlayerID())
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("unpaid casting changed state")
			}
			a.ManaPool().Add(core.Blue, 1)
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{r.ID()}, 0); err == nil || a.ManaPool().TotalMana() != 2 || g.StackSize() != 0 {
				t.Fatal("invalid player target spent mana")
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(p.ID()) == nil || len(b.Hand()) != 0 || a.ManaPool().TotalMana() != 0 {
				t.Fatal("wrong announcement timing")
			}
			clone := g.Clone()
			want := 1
			tokenID := uuid.Nil
			switch change {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: q.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "phase":
				g.PhaseOut(p.ID())
				want = 0
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "return":
				g.BouncePermanentToHand(p)
				b.RemoveFromHand(owned.ID())
				g.PutOnBattlefield(owned, b.PlayerID())
			case "late-entry":
				c := mage.NewArtifact("Later", "")
				c.SetOwner(b.PlayerID())
				g.PutOnBattlefield(c, a.PlayerID())
				want = 2
			case "type-change":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.AddCardType(mage.This, core.TypeArtifact)), r.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				want = 2
			case "shroud":
				g.FindPermanent(p.ID()).GrantBaseAttr(core.Shroud)
				g.FindPermanent(p.ID()).GrantBaseAttr(core.Indestructible)
				g.AddRegenerationShield(p.ID())
			case "token":
				token := mage.NewToken("Artifact token", 1, 1, []core.CardType{core.TypeArtifact, core.TypeCreature}, nil)
				tokenID = token.ID()
				token.SetOwner(b.PlayerID())
				g.PutOnBattlefield(token, a.PlayerID())
			case "copy":
				a.targets = []uuid.UUID{a.PlayerID()}
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), true) == nil {
					t.Fatal("copy failed")
				}
				g.ResolveTopOfStack()
				if g.FindPermanent(q.ID()) != nil || g.FindPermanent(p.ID()) == nil || len(a.Hand()) != 1 {
					t.Fatal("copy used original owner")
				}
			}
			g.ResolveStack()
			if tokenID != uuid.Nil && g.FindPermanent(tokenID) != nil {
				t.Fatal("artifact token stayed on battlefield")
			}
			if len(a.Graveyard()) != 1 || a.Graveyard()[0].ID() != card.ID() {
				t.Fatal("spell did not reach its owner graveyard")
			}
			if len(b.Hand()) != want || (change != "type-change" && g.FindPermanent(r.ID()) == nil) {
				t.Fatal("wrong resolution collection")
			}
			if change != "copy" && g.FindPermanent(q.ID()) == nil {
				t.Fatal("controller replaced owner")
			}
			if change == "phase" {
				g.PhaseIn([]uuid.UUID{p.ID()})
				if g.FindPermanent(p.ID()) == nil {
					t.Fatal("phased member was returned")
				}
			}
			if clone.FindPermanent(p.ID()) == nil || len(clone.GetPlayer(b.PlayerID()).Hand()) != 0 {
				t.Fatal("resolution changed clone")
			}
			clone.ResolveStack()
			if len(clone.GetPlayer(b.PlayerID()).Hand()) != 1 || clone.FindPermanent(q.ID()) == nil {
				t.Fatal("clone lost owner binding")
			}
		})
	}
}

func TestRONReturnCollectionBatchAndInstances(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(TapPermanent(ThisPermanent),ActionList([PutEachPermanentIntoItsOwnersHand(And([IsCardtype(Artifact),OwnedByAPlayer(SinglePlayer(You))]))]))`)
	g, a, b := ronGame()
	first, second := factory(), factory()
	first.SetOwner(a.PlayerID())
	second.SetOwner(b.PlayerID())
	p, q := g.PutOnBattlefield(first, a.PlayerID()), g.PutOnBattlefield(second, b.PlayerID())
	if first.Abilities()[0] == second.Abilities()[0] {
		t.Fatal("factories shared ability")
	}
	observer := mage.NewLand("Observer", mage.WithAbility(mage.NewStateTriggered(false, mage.Sacrifice(mage.This)).SetCondition(func(_ *core.GameEvent, game mage.GameReader, _, _ uuid.UUID) bool {
		return game.FindPermanent(p.ID()) == nil && game.FindPermanent(q.ID()) != nil
	})))
	observer.SetOwner(a.PlayerID())
	watcher := g.PutOnBattlefield(observer, a.PlayerID())
	for _, v := range []struct {
		source *mage.Permanent
		player uuid.UUID
	}{{p, a.PlayerID()}, {q, b.PlayerID()}} {
		if err := g.ActivateAbilityByIndex(v.player, v.source.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
	}
	g.ResolveStack()
	if len(a.Hand()) != 1 || len(b.Hand()) != 1 || g.FindPermanent(watcher.ID()) == nil {
		t.Fatal("instance owner bindings crossed")
	}
	batch := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([PutEachPermanentIntoItsOwnersHand(IsCardtype(Artifact))]))`)()
	a.RemoveFromHand(first.ID())
	b.RemoveFromHand(second.ID())
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	batch.SetOwner(a.PlayerID())
	a.AddToHand(batch)
	if err := g.CastSpellByID(a.PlayerID(), batch.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if len(a.Hand()) != 1 || len(b.Hand()) != 1 || g.FindPermanent(watcher.ID()) == nil {
		t.Fatal("return exposed intermediate state")
	}
}

func TestRONReturnCollectionInvalid(t *testing.T) {
	for _, body := range []string{
		`PutEachPermanentIntoItsOwnersHand`, `PutEachPermanentIntoItsOwnersHand()`, `PutEachPermanentIntoItsOwnersHand(AnyPermanent,AnyPermanent)`,
		`PutEachPermanentIntoItsOwnersHand(OwnedByAPlayer(SinglePlayer(Ref_TargetPlayer)))`,
		`PutEachPermanentIntoItsOwnersHand(OwnedByAPlayer(SinglePlayer(Ref_TargetPermanent)))`,
		`PutEachPermanentIntoItsOwnersHand(OwnedByAPlayer(You))`,
		`PutEachPermanentIntoItsOwnersHand(OwnedByAPlayer(SinglePlayer(You())))`,
		`PutEachPermanentIntoItsOwnersHand(And([]))`,
		`PutEachPermanentIntoItsOwnersHand(And([OwnedByAPlayer(SinglePlayer(You)),OwnedByAPlayer(SinglePlayer(You))]))`,
		`PutEachPermanentIntoItsOwnersHand(Or([OwnedByAPlayer(SinglePlayer(You)),IsCardtype(Artifact)]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([`+body+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRONReturnCollectionInventory(t *testing.T) {
	source := `Card(Name:"Return",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([PutEachPermanentIntoItsOwnersHand(And([IsCardtype(Artifact),OwnedByAPlayer(SinglePlayer(Ref_TargetPlayer))]))])))])`
	report, err := InspectFS(fstest.MapFS{"return.ron": {Data: []byte(source)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 1 {
		t.Fatal("collection not compiled")
	}
	found := false
	for _, s := range report.Signatures {
		if s.Name == "Ref_TargetPlayer" {
			found = true
			if s.Role != "player_reference" {
				t.Fatal("owner has wrong role")
			}
		}
	}
	if !found {
		t.Fatal("owner read missing")
	}
	if err := mage.ValidateCardDefinition("Return", permanentScopeFactory(t, "Instant", `SpellActions(ActionList([PutEachPermanentIntoItsOwnersHand(And([And([OwnedByAPlayer(SinglePlayer(You))]),Or([IsCardtype(Artifact),IsCardtype(Enchantment)])]))]))`)()); err != nil {
		t.Fatal(err)
	}
	repositoryCard(t, "3ED", "Hurkyl's Recall")
}
