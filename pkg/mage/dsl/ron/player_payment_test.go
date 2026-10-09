package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONPlayerPaymentGuardian(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, set, "Guardian Angel")
			a.AddToHand(c)
			a.ManaPool().Add(core.White, 1)
			a.ManaPool().Add(core.Colorless, 2)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{b.PlayerID()}, 2); err != nil {
				t.Fatal(err)
			}
			if len(g.GetPlayerPayments(a.PlayerID())) != 0 {
				t.Fatal("installed before resolution")
			}
			g.ResolveStack()
			a.ManaPool().Add(core.Colorless, 2)
			offers := g.GetPlayerPayments(a.PlayerID())
			if len(offers) != 1 {
				t.Fatalf("offers=%d", len(offers))
			}
			for range 2 {
				if err := g.TakePlayerPayment(a.PlayerID(), offers[0].ID); err != nil {
					t.Fatal(err)
				}
			}
			if g.StackSize() != 0 || a.ManaPool().TotalMana() != 0 {
				t.Fatal("payment uses stack or incorrect cost")
			}
			g.DealDamageToPlayer(b, 5, c.ID())
			if b.Life() != 19 {
				t.Fatalf("life=%d", b.Life())
			}
		})
	}
}

func TestRONPlayerPaymentInvalid(t *testing.T) {
	valid := `CreatePlayerEffectUntil(You,[MayPayManaAnyTimeTheyCouldCastAnInstant([ManaCostGeneric(1)],[GainLife(Integer(1))])],UntilEndOfTurn)`
	for _, action := range []string{
		strings.Replace(valid, "You", "Ref_TargetPlayer", 1),
		strings.Replace(valid, "UntilEndOfTurn", "UntilEndOfCombat", 1),
		strings.Replace(valid, "UntilEndOfTurn", "UntilEndOfTurn()", 1),
		strings.Replace(valid, "ManaCostGeneric(1)", "ManaCostX", 1),
		strings.Replace(valid, "ManaCostGeneric(1)", "ManaCostGeneric(-1)", 1),
		strings.Replace(valid, "GainLife(Integer(1))", "UnknownAction", 1),
		strings.Replace(valid, "MayPayManaAnyTimeTheyCouldCastAnInstant", "MayActionOnce", 1),
		`CreatePlayerEffectUntil(You,[],UntilEndOfTurn)`,
		`CreatePlayerEffectUntil(You,[MayPayManaAnyTimeTheyCouldCastAnInstant([],[])],UntilEndOfTurn)`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([` + action + `]))])`
		registry, err := LoadFS(fstest.MapFS{"invalid.ron": {Data: []byte(src)}}, "*.ron")
		if err == nil || registry != nil {
			t.Fatalf("invalid declaration accepted: %s", action)
		}
	}
}

func TestRONPlayerPaymentTargetIdentity(t *testing.T) {
	for _, change := range []string{"live", "phase", "phase-cycle", "return", "shroud", "control"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "LEA", "Guardian Angel")
			victim := mage.NewCreature("Victim", "", 8, 8)
			victim.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(victim, b.PlayerID())
			a.AddToHand(c)
			a.ManaPool().Add(core.White, 1)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			a.ManaPool().Add(core.Colorless, 1)
			id := g.GetPlayerPayments(a.PlayerID())[0].ID
			switch change {
			case "phase":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "return":
				g.BouncePermanentToHand(p)
				b.RemoveFromHand(victim.ID())
				p = g.PutOnBattlefield(victim, b.PlayerID())
			case "shroud":
				g.AddContinuousEffect(mage.TemporaryKeyword(p.ID(), core.Shroud))
				g.ApplyContinuousEffects()
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: c.ID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			}
			clone := g.Clone()
			if err := g.TakePlayerPayment(a.PlayerID(), id); err != nil {
				t.Fatal(err)
			}
			if change == "phase" {
				g.PhaseIn([]uuid.UUID{p.ID()})
			}
			g.DealDamageToPermanent(g.FindPermanent(p.ID()), 2, c.ID())
			want := 1
			if change == "return" || change == "phase" {
				want = 2
			}
			if g.FindPermanent(p.ID()).Damage != want {
				t.Fatalf("damage=%d want=%d", g.FindPermanent(p.ID()).Damage, want)
			}
			if clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 1 {
				t.Fatal("clone shared mana")
			}
		})
	}
}

func TestRONPlayerPaymentCopiesAndExpiry(t *testing.T) {
	g, a, b := ronGame()
	c := repositoryCard(t, "LEA", "Guardian Angel")
	a.AddToHand(c)
	a.ManaPool().Add(core.White, 1)
	a.ManaPool().Add(core.Colorless, 2)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{b.PlayerID()}, 2); err != nil {
		t.Fatal(err)
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	pending := g.Clone()
	g.ResolveStack()
	pending.ResolveStack()
	for _, game := range []*mage.Game{g, pending} {
		game.GetPlayer(a.PlayerID()).ManaPool().Add(core.Colorless, 1)
		game.GetPlayer(b.PlayerID()).ManaPool().Add(core.Colorless, 1)
		for _, player := range []uuid.UUID{a.PlayerID(), b.PlayerID()} {
			offers := game.GetPlayerPayments(player)
			if len(offers) != 1 {
				t.Fatal("copy did not retain separate payer")
			}
			if err := game.TakePlayerPayment(player, offers[0].ID); err != nil {
				t.Fatal(err)
			}
		}
		game.DealDamageToPlayer(game.GetPlayer(b.PlayerID()), 1, c.ID())
		if game.GetPlayer(b.PlayerID()).Life() != 20 {
			t.Fatal("copies lost X or recipient")
		}
	}
	g.RunStepWithPriority(core.Cleanup)
	a.ManaPool().Add(core.Colorless, 1)
	if len(g.GetPlayerPayments(a.PlayerID())) != 0 {
		t.Fatal("permission survived cleanup")
	}
	g.DealDamageToPlayer(b, 1, c.ID())
	if b.Life() != 19 {
		t.Fatal("shield survived cleanup")
	}
	pending.GetPlayer(a.PlayerID()).ManaPool().Add(core.Colorless, 1)
	if len(pending.GetPlayerPayments(a.PlayerID())) != 1 {
		t.Fatal("cleanup changed clone")
	}
	pending.DealDamageToPlayer(pending.GetPlayer(b.PlayerID()), 6, c.ID())
	if pending.GetPlayer(b.PlayerID()).Life() != 19 {
		t.Fatal("clone lost unused shields or copied X")
	}
}

func TestRONPlayerPaymentIllegalAndRollback(t *testing.T) {
	g, a, b := ronGame()
	c := repositoryCard(t, "LEA", "Guardian Angel")
	victim := mage.NewCreature("Victim", "", 3, 3)
	p := g.PutOnBattlefield(victim, b.PlayerID())
	a.AddToHand(c)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{p.ID()}, 0); err == nil {
		t.Fatal("unpaid cast accepted")
	}
	if len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("unpaid cast changed zones")
	}
	a.ManaPool().Add(core.White, 1)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(p)
	g.ResolveStack()
	a.ManaPool().Add(core.Colorless, 1)
	if len(g.GetPlayerPayments(a.PlayerID())) != 0 {
		t.Fatal("illegal spell installed permission")
	}
	c = repositoryCard(t, "LEA", "Guardian Angel")
	a.AddToHand(c)
	a.ManaPool().Add(core.White, 1)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	id := g.GetPlayerPayments(a.PlayerID())[0].ID
	a.ManaPool().Clear()
	if err := g.TakePlayerPayment(a.PlayerID(), id); err == nil {
		t.Fatal("unpaid shield accepted")
	}
	g.DealDamageToPlayer(b, 1, c.ID())
	if b.Life() != 19 {
		t.Fatal("failed payment installed shield")
	}
}

func TestRONPlayerPaymentInventory(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`Card(Name:"Inventory",Rules:[SpellActions(ActionList([CreatePlayerEffectUntil(You,[MayPayManaAnyTimeTheyCouldCastAnInstant([ManaCostGeneric(1)],[GainLife(Integer(1))])],UntilEndOfTurn)]))])`), "inventory.ron")
	if err != nil {
		t.Fatal(err)
	}
	uses := map[string]*InventorySignature{}
	walkInventory(nodes[0], "Inventory", "Card", "card", "collection", false, uses)
	want := map[string]string{"You": "player_reference", "MayPayManaAnyTimeTheyCouldCastAnInstant": "temporary_player_operation", "ManaCostGeneric": "metadata", "GainLife": "action", "UntilEndOfTurn": "duration"}
	for name, role := range want {
		found := false
		for _, use := range uses {
			if use.Name == name && use.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s missing role %s", name, role)
		}
	}
}

func TestRONPlayerPaymentResolvingContexts(t *testing.T) {
	action := `CreatePlayerEffectUntil(ControllerOfPermanent(ThisPermanent),[MayPayManaAnyTimeTheyCouldCastAnInstant([ManaCostGeneric(1)],[TapPermanent(ThisPermanent),GainLife(Integer(1))])],UntilEndOfTurn)`
	for _, trigger := range []bool{false, true} {
		rule := `Activated(PayMana([]),ActionList([` + action + `]))`
		if trigger {
			rule = `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([` + action + `]))`
		}
		for _, change := range []string{"live", "return", "phase", "control"} {
			t.Run(rule+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				c := permanentScopeFactory(t, "Artifact", rule)()
				p := g.PutOnBattlefield(c, a.PlayerID())
				if trigger {
					g.PutTriggersOnStack()
				} else if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				a.ManaPool().Add(core.Colorless, 1)
				offers := g.GetPlayerPayments(a.PlayerID())
				if len(offers) != 1 {
					t.Fatal("permission not installed")
				}
				id := offers[0].ID
				switch change {
				case "return":
					g.RemoveFromBattlefield(p)
					p = g.PutOnBattlefield(c, b.PlayerID())
				case "phase":
					g.PhaseOut(p.ID())
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				if err := g.TakePlayerPayment(a.PlayerID(), id); err != nil {
					t.Fatal(err)
				}
				if a.Life() != 21 || b.Life() != 20 {
					t.Fatal("payment followed new controller")
				}
				if live := g.FindPermanent(p.ID()); live != nil && live.Tapped != (change != "return") {
					t.Fatal("source incarnation changed")
				}
			})
		}
	}
}

func TestRONPlayerPaymentMultipleOperations(t *testing.T) {
	g, a, _ := ronGame()
	c := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([CreatePlayerEffectUntil(You,[MayPayManaAnyTimeTheyCouldCastAnInstant([ManaCostGeneric(1)],[GainLife(Integer(2))]),MayPayManaAnyTimeTheyCouldCastAnInstant([ManaCostGeneric(2)],[GainLife(Integer(3))])],UntilEndOfTurn)]))`)()
	if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
		t.Fatal(err)
	}
	body := mage.Sequence(c.Abilities()[0].(*mage.ActionDefinition).Effects()...)
	if err := mage.ApplyEffect(g, body, c.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	a.ManaPool().Add(core.Colorless, 1)
	cheap := g.GetPlayerPayments(a.PlayerID())
	if len(cheap) != 1 {
		t.Fatal("separate costs lost")
	}
	a.ManaPool().Add(core.Colorless, 2)
	both := g.GetPlayerPayments(a.PlayerID())
	if len(both) != 2 || both[0].ID == both[1].ID {
		t.Fatal("operations share identity")
	}
	for _, offer := range both {
		if err := g.TakePlayerPayment(a.PlayerID(), offer.ID); err != nil {
			t.Fatal(err)
		}
	}
	if a.Life() != 25 || a.ManaPool().TotalMana() != 0 {
		t.Fatal("operations share costs or bodies")
	}
}
