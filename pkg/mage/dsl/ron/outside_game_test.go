package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONOutsideGameRing(t *testing.T) {
	card := repositoryCard(t, "ARN", "Ring of Ma'rûf")
	g, a, b := ronGame()
	g.SetTurn(3)
	g.SetStep(core.PrecombatMain)
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("unpaid cast")
	}
	a.ManaPool().Add(core.Colorless, 5)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	p := g.FindPermanent(card.ID())
	if p == nil {
		t.Fatal("cast did not resolve")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
		t.Fatal("unpaid activation")
	}
	if g.FindPermanent(p.ID()) == nil || p.Tapped || len(g.GetExile()) != 0 {
		t.Fatal("failed cost changed source")
	}
	outside := mage.NewCreature("Outside", "", 2, 3)
	outside.SetOwner(a.PlayerID())
	if err := g.SetOutsideGameCards(a.PlayerID(), []mage.Card{outside}); err != nil {
		t.Fatal(err)
	}
	seedRONDrawLibrary(a)
	seedRONDrawLibrary(b)
	a.ManaPool().Add(core.Colorless, 5)
	g.PhaseOut(p.ID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
		t.Fatal("phased source activated")
	}
	if a.ManaPool().TotalMana() != 5 || len(g.GetExile()) != 0 {
		t.Fatal("phased activation paid costs")
	}
	g.PhaseIn([]uuid.UUID{p.ID()})
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(p.ID()) != nil || len(g.GetExile()) != 1 || a.ManaPool().TotalMana() != 0 {
		t.Fatal("activation cost incomplete")
	}
	g.PlayerDrawCard(a)
	if len(a.Hand()) != 1 {
		t.Fatal("replacement installed before resolution")
	}
	g.ResolveStack()
	clone := g.Clone()
	g.PlayerDrawCard(b)
	if len(b.Hand()) != 1 || len(g.OutsideGameCards(a.PlayerID())) != 1 {
		t.Fatal("wrong player replaced")
	}
	g.PlayerDrawCard(a)
	if len(a.Hand()) != 2 || a.Hand()[1].ID() != outside.ID() || len(g.OutsideGameCards(a.PlayerID())) != 0 {
		t.Fatal("complete Ring did not replace draw")
	}
	if a.LastDrawnCardID() == outside.ID() {
		t.Fatal("outside entry counted as draw")
	}
	g.PlayerDrawCard(a)
	if len(a.Hand()) != 3 {
		t.Fatal("replacement used twice")
	}
	clone.PlayerDrawCard(clone.GetPlayer(a.PlayerID()))
	if len(clone.GetPlayer(a.PlayerID()).Hand()) != 2 {
		t.Fatal("clone replacement lost")
	}
}
func TestRONOutsideGameRingExpiryAndEmpty(t *testing.T) {
	for _, state := range []string{"expiry", "empty", "late-pool"} {
		t.Run(state, func(t *testing.T) {
			g, a, _ := ronGame()
			g.SetTurn(3)
			c := repositoryCard(t, "ARN", "Ring of Ma'rûf")
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			a.ManaPool().Add(core.Colorless, 5)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if state == "expiry" {
				g.RunStepWithPriority(core.Cleanup)
			}
			if state != "empty" {
				out := mage.NewArtifact("Late", "")
				out.SetOwner(a.PlayerID())
				if err := g.SetOutsideGameCards(a.PlayerID(), []mage.Card{out}); err != nil {
					t.Fatal(err)
				}
			}
			if state == "expiry" {
				seedRONDrawLibrary(a)
			}
			g.PlayerDrawCard(a)
			if state == "empty" {
				if len(a.Hand()) != 0 || a.DrewFromEmpty() {
					t.Fatal("empty pool restored replaced draw")
				}
			} else if len(a.Hand()) != 1 {
				t.Fatal("missing card")
			}
			if state == "expiry" && len(g.OutsideGameCards(a.PlayerID())) != 1 {
				t.Fatal("expired rule entered card")
			}
		})
	}
}
func TestRONOutsideGameInvalid(t *testing.T) {
	for _, action := range []string{`PutACardFromOutsideGameInHand`, `PutACardFromOutsideGameInHand()`, `PutACardFromOutsideGameInHand(AnyCard())`, `PutACardFromOutsideGameInHand(AnyCard,AnyCard)`, `PutACardFromOutsideGameInHand(AnyPermanent)`, `PutACardFromOutsideGameInHand(IsCardtype(Creature))`} {
		source := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([` + action + `]))])`
		if registry, err := LoadFS(fstest.MapFS{"invalid.ron": {Data: []byte(source)}}, "*.ron"); err == nil || registry != nil {
			t.Fatal("invalid declaration accepted", action)
		}
	}
	n := nameScopeNode(t, `PutACardFromOutsideGameInHand(AnyCard)`)
	for _, ctx := range []scope{{}, {resolving: true, enteringSource: true}} {
		if _, err := compileOutsideGameEntry(n, ctx); err == nil {
			t.Fatal("invalid context")
		}
	}
}
func TestRONOutsideGameInventory(t *testing.T) {
	n, err := Parse(strings.NewReader(`PutACardFromOutsideGameInHand(AnyCard)`), "inventory.ron")
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string]*InventorySignature{}
	walkInventory(n[0], "Rule", "action", "action", "action", true, entries)
	for _, e := range entries {
		if e.Name == "AnyCard" && e.Role == "card_filter" {
			return
		}
	}
	t.Fatal("missing card filter role")
}

func TestRONOutsideGameExplicitPlayerAndPendingClone(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerAction(Ref_TargetPlayer,PutACardFromOutsideGameInHand(AnyCard))])))`)
	g, a, b := ronGame()
	g.SetTurn(3)
	c := factory()
	c.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(c, a.PlayerID())
	outside := mage.NewArtifact("Outside", "")
	outside.SetOwner(b.PlayerID())
	if err := g.SetOutsideGameCards(b.PlayerID(), []mage.Card{outside}); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	g.ResolveStack()
	clone.ResolveStack()
	if len(a.Hand()) != 0 || len(b.Hand()) != 1 || len(clone.GetPlayer(b.PlayerID()).Hand()) != 1 {
		t.Fatal("explicit player or pending clone")
	}
	b.Hand()[0].SetOwner(a.PlayerID())
	if clone.GetPlayer(b.PlayerID()).Hand()[0].Owner() != b.PlayerID() {
		t.Fatal("clones shared mutable card ownership")
	}
}
