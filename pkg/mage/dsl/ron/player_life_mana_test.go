package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONPlayerLifeManaChannel(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, set, "Channel")
			a.AddToHand(c)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err == nil {
				t.Fatal("unpaid Channel cast")
			}
			a.ManaPool().Add(core.Green, 2)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if len(g.GetPlayerManaAbilities(a.PlayerID())) != 0 {
				t.Fatal("permission installed before resolution")
			}
			g.ResolveStack()
			offers := g.GetPlayerManaAbilities(a.PlayerID())
			if len(offers) != 1 || len(g.GetPlayerManaAbilities(b.PlayerID())) != 0 {
				t.Fatal("wrong permission owner")
			}
			if err := g.ActivatePlayerManaAbility(a.PlayerID(), offers[0].ID); err != nil {
				t.Fatal(err)
			}
			if a.Life() != 19 || a.ManaPool().Count(core.Colorless) != 1 || g.StackSize() != 0 {
				t.Fatal("wrong production")
			}
			spell := mage.NewInstant("Payment", "{3}", mage.WithAbility(mage.Spell(mage.GainLifeFor(mage.You, mage.Fixed(1)))))
			a.AddToHand(spell)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{}, 0); err != nil {
				t.Fatal(err)
			}
			if a.Life() != 17 || a.ManaPool().TotalMana() != 0 {
				t.Fatal("Channel did not fund casting")
			}
			g.ResolveStack()
			if a.Life() != 18 {
				t.Fatal("spell did not resolve")
			}
		})
	}
}

func TestRONPlayerLifeManaInvalid(t *testing.T) {
	valid := `CreatePlayerEffectUntil(You,[MayPayLifeToProduceManaAsManaAbility(Integer(1),ManaProduceC)],UntilEndOfTurn)`
	for _, action := range []string{
		strings.Replace(valid, "You", "Ref_TargetPlayer", 1),
		strings.Replace(valid, "You", "ThisPermanent", 1),
		strings.Replace(valid, "Integer(1),ManaProduceC", "Integer(1),ManaProduceC,ManaProduceC", 1),
		strings.Replace(valid, "Integer(1)", "Integer(0)", 1),
		strings.Replace(valid, "Integer(1)", "Integer(-1)", 1),
		strings.Replace(valid, "Integer(1)", "ValueX", 1),
		strings.Replace(valid, "ManaProduceC", "AnyManaColor", 1),
		strings.Replace(valid, "ManaProduceC", "ManaProduceC()", 1),
		strings.Replace(valid, "UntilEndOfTurn", "UntilEndOfTurn()", 1),
		strings.Replace(valid, "UntilEndOfTurn", "UntilEndOfCombat", 1),
		strings.Replace(valid, "Integer(1),ManaProduceC", "Integer(1)", 1),
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([` + action + `]))])`
		_, err := LoadFS(fstest.MapFS{"invalid.ron": {Data: []byte(src)}}, "*.ron")
		if err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
}

func TestRONPlayerLifeManaCopiesAndExpiry(t *testing.T) {
	g, a, b := ronGame()
	c := repositoryCard(t, "LEA", "Channel")
	a.AddToHand(c)
	a.ManaPool().Add(core.Green, 2)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	pending := g.Clone()
	for _, game := range []*mage.Game{g, pending} {
		game.ResolveStack()
		for _, player := range []uuid.UUID{a.PlayerID(), b.PlayerID()} {
			offers := game.GetPlayerManaAbilities(player)
			if len(offers) != 1 {
				t.Fatal("copy lost its controller")
			}
			if err := game.ActivatePlayerManaAbility(player, offers[0].ID); err != nil {
				t.Fatal(err)
			}
			if game.GetPlayer(player).Life() != 19 || game.GetPlayer(player).ManaPool().Count(core.Colorless) != 1 {
				t.Fatal("copy shared payer")
			}
		}
	}
	g.RunStepWithPriority(core.Cleanup)
	if len(g.GetPlayerManaAbilities(a.PlayerID())) != 0 || len(g.GetPlayerManaAbilities(b.PlayerID())) != 0 {
		t.Fatal("permission survived cleanup")
	}
	if len(pending.GetPlayerManaAbilities(a.PlayerID())) != 1 {
		t.Fatal("cleanup changed clone")
	}
}

func TestRONPlayerLifeManaScopes(t *testing.T) {
	action := `CreatePlayerEffectUntil(Ref_TargetPlayer,[MayPayLifeToProduceManaAsManaAbility(Integer(2),ManaProduceG)],UntilEndOfTurn)`
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPlayer(AnyPlayer)],ActionList([`+action+`])))`)
	g, a, b := ronGame()
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	id := g.GetPlayerManaAbilities(b.PlayerID())[0].ID
	g.RemoveFromBattlefield(p)
	if err := g.ActivatePlayerManaAbility(b.PlayerID(), id); err != nil {
		t.Fatal(err)
	}
	if b.Life() != 18 || b.ManaPool().Count(core.Green) != 1 || len(g.GetPlayerManaAbilities(a.PlayerID())) != 0 {
		t.Fatal("wrong scoped payer")
	}
	nodes, err := Parse(strings.NewReader(strings.Replace(action, "Ref_TargetPlayer", "You", 1)), "context.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, ctx := range []scope{{}, {resolving: true, enteringSource: true}, {resolving: true, drawReplacementBody: true}} {
		if _, err := compilePlayerPayment(nodes[0], ctx); err == nil {
			t.Fatal("unsupported installation context accepted")
		}
	}
}

func TestRONPlayerLifeManaInventory(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`Card(Name:"Inventory",Rules:[SpellActions(ActionList([CreatePlayerEffectUntil(You,[MayPayLifeToProduceManaAsManaAbility(Integer(1),ManaProduceC)],UntilEndOfTurn)]))])`), "inventory.ron")
	if err != nil {
		t.Fatal(err)
	}
	uses := map[string]*InventorySignature{}
	walkInventory(nodes[0], "Inventory", "Card", "card", "collection", false, uses)
	for name, role := range map[string]string{"You": "player_reference", "MayPayLifeToProduceManaAsManaAbility": "temporary_player_operation", "Integer": "value", "ManaProduceC": "mana_production", "UntilEndOfTurn": "duration"} {
		found := false
		for _, use := range uses {
			if use.Name == name && use.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s has no %s role", name, role)
		}
	}
}
