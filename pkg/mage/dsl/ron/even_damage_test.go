package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const evenUnionBody = `Targeted([AnyNumberOfAnyTargets],ActionList([SpellDealsDamageDividedAmongRecipientsRoundedDown(ThisSpell,ValueX,Ref_AnyTargets)]))`

func TestRONEvenDamageCatapult(t *testing.T) {
	for _, state := range []string{"normal", "entry", "phase", "phase-cycle", "control", "returned", "prevention", "copy", "empty", "zero", "zero-share"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := distributedDamageGame()
			card := repositoryCard(t, "FEM", "Dwarven Catapult")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			first := g.PutOnBattlefield(mage.NewCreature("First", "", 1, 20), b.PlayerID())
			second := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 20), b.PlayerID())
			own := g.PutOnBattlefield(mage.NewCreature("Own", "", 1, 20), a.PlayerID())
			walker := g.PutOnBattlefield(mage.NewPlaneswalker("Walker", "", 20), b.PlayerID())
			x := 7
			if state == "zero" {
				x = 0
			}
			if state == "zero-share" {
				x = 1
			}
			a.ManaPool().Add(core.Red, 1)
			a.ManaPool().Add(core.Colorless, x)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, x); err != nil {
				t.Fatal(err)
			}
			g.SetXValue(99)
			wantFirst, wantSecond := x/2, x/2
			switch state {
			case "entry":
				g.PutOnBattlefield(mage.NewCreature("Later", "", 1, 20), b.PlayerID())
				wantFirst, wantSecond = x/3, x/3
			case "phase":
				g.PhaseOut(first.ID())
				wantFirst, wantSecond = 0, x
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(first.ID()))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: first.ID(), TargetID: first.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				wantFirst, wantSecond = 0, x
			case "returned":
				g.BouncePermanentToHand(first)
				b.RemoveFromHand(first.Card.ID())
				first = g.PutOnBattlefield(first.Card, b.PlayerID())
			case "prevention":
				g.AddPreventionShield(first.ID(), 2)
				wantFirst = max(0, wantFirst-2)
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				wantFirst *= 2
				wantSecond *= 2
			case "empty":
				g.BouncePermanentToHand(first)
				g.BouncePermanentToHand(second)
				wantFirst, wantSecond = 0, 0
			}
			clone := g.Clone()
			g.ResolveStack()
			if clone.GetPlayer(b.PlayerID()).Life() != 20 || clone.FindPermanent(own.ID()).Damage != 0 {
				t.Fatal("clone shared results")
			}
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				if state != "empty" && (game.FindPermanentIncludingPhased(first.ID()).Damage != wantFirst || game.FindPermanent(second.ID()).Damage != wantSecond) {
					t.Fatalf("damage %d/%d, want %d/%d", game.FindPermanentIncludingPhased(first.ID()).Damage, game.FindPermanent(second.ID()).Damage, wantFirst, wantSecond)
				}
				if game.FindPermanent(own.ID()).Damage != 0 || game.FindPermanent(walker.ID()).Counters[core.Loyalty] != 20 || game.GetPlayer(b.PlayerID()).Life() != 20 {
					t.Fatal("wrong recipients")
				}
			}
			if a.prompts != 0 || a.ManaPool().TotalMana() != 0 {
				t.Fatal("unexpected division choice or payment")
			}
		})
	}
}

func TestRONEvenDamageUnion(t *testing.T) {
	for _, state := range []string{"normal", "empty", "partial", "all-illegal", "returned", "phase", "phase-cycle", "prevention", "copy", "zero", "zero-share"} {
		t.Run(state, func(t *testing.T) {
			factory := evenUnionFactory(t)
			g, a, b := distributedDamageGame()
			card := factory()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			first := g.PutOnBattlefield(mage.NewCreature("First", "", 1, 20), b.PlayerID())
			walker := g.PutOnBattlefield(mage.NewPlaneswalker("Walker", "", 20), b.PlayerID())
			ids := []uuid.UUID{first.ID(), walker.ID(), b.PlayerID()}
			x := 7
			if state == "zero" {
				x = 0
			}
			if state == "zero-share" {
				x = 1
			}
			if state == "empty" {
				ids = nil
			}
			if state == "all-illegal" {
				ids = ids[:2]
			}
			a.ManaPool().Add(core.Colorless, x)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), ids, x); err != nil {
				t.Fatal(err)
			}
			g.SetXValue(99)
			share := x / 3
			wantFirst := share
			switch state {
			case "empty":
				share, wantFirst = 0, 0
			case "partial":
				first.GrantBaseAttr(core.Shroud)
				share, wantFirst = x/2, 0
			case "all-illegal":
				first.GrantBaseAttr(core.Shroud)
				walker.GrantBaseAttr(core.Shroud)

				share, wantFirst = 0, 0
			case "returned":
				g.BouncePermanentToHand(first)
				b.RemoveFromHand(first.Card.ID())
				first = g.PutOnBattlefield(first.Card, b.PlayerID())
				share, wantFirst = x/2, 0
			case "phase":
				g.PhaseOut(first.ID())
				share, wantFirst = x/2, 0
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(first.ID()))
			case "prevention":
				g.AddPreventionShield(first.ID(), 1)
				wantFirst = max(0, share-1)
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				share *= 2
				wantFirst *= 2
			}
			clone := g.Clone()
			g.ResolveStack()
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				if game.FindPermanentIncludingPhased(first.ID()).Damage != wantFirst || int(game.FindPermanent(walker.ID()).Counters[core.Loyalty]) != 20-share || game.GetPlayer(b.PlayerID()).Life() != 20-share {
					t.Fatal("wrong legal-recipient division")
				}
			}
			if a.prompts != 0 {
				t.Fatal("equal division offered an allocation choice")
			}
		})
	}
}

func TestRONEvenDamageInvalid(t *testing.T) {
	for _, body := range []string{
		strings.Replace(evenUnionBody, "AnyNumberOfAnyTargets", "AnyNumberOfAnyTargets()", 1),
		strings.Replace(evenUnionBody, "Ref_AnyTargets", "Ref_AnyTarget", 1),
		strings.Replace(evenUnionBody, "ThisSpell", "ThisPermanent", 1),
		strings.Replace(evenUnionBody, "ValueX", "Unknown", 1),
		strings.Replace(evenUnionBody, "[AnyNumberOfAnyTargets]", "[AnyNumberOfAnyTargets,AnyTarget]", 1),
		`ActionList([SpellDealsDamageDividedAmongRecipientsRoundedDown(ThisSpell,Integer(4),Ref_AnyTargets)])`,
		`Targeted([AnyTarget],ActionList([SpellDealsDamageDividedAmongRecipientsRoundedDown(ThisSpell,Integer(4),Ref_AnyTargets)]))`,
		strings.Replace(evenUnionBody, "ThisSpell,ValueX,Ref_AnyTargets", "ThisSpell,ValueX", 1),
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),ManaCost:[ManaCostX],Rules:[SpellActions(`+body+`)])`), "invalid-even.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
	for _, rules := range []string{`Activated(PayMana([]),` + evenUnionBody + `)`, `TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),` + evenUnionBody + `)`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rules+`])`), "invalid-context.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted unsupported context")
		}
	}
}

func evenUnionFactory(t *testing.T) mage.CardFactory {
	t.Helper()
	nodes, err := Parse(strings.NewReader(`Card(Name:"Equal union",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),ManaCost:[ManaCostX],Rules:[SpellActions(`+evenUnionBody+`)])`), "even-union.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	return factory
}

func TestRONEvenDamagePaymentAndRetarget(t *testing.T) {
	g, a, b := distributedDamageGame()
	card := evenUnionFactory(t)()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 1, 20), b.PlayerID())
	ids := []uuid.UUID{victim.ID(), b.PlayerID()}
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), ids, 5); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("unpaid cast changed state")
	}
	a.ManaPool().Add(core.Colorless, 5)
	for _, bad := range [][]uuid.UUID{{b.PlayerID(), b.PlayerID()}, {uuid.New()}, {g.PutOnBattlefield(mage.NewLand("Land"), b.PlayerID()).ID()}} {
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), bad, 5); err == nil || a.ManaPool().TotalMana() != 5 || len(a.Hand()) != 1 || g.StackSize() != 0 {
			t.Fatal("illegal cast paid mana")
		}
	}
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), ids, 5); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	other := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 20), b.PlayerID())
	a.targets = []uuid.UUID{other.ID(), a.PlayerID()}
	copied := g.CopySpellOnStack(card.ID(), a.PlayerID(), true)
	if copied == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStackObject(copied)
	if other.Damage != 2 || a.Life() != 18 || victim.Damage != 0 || original.Targets[0] != victim.ID() {
		t.Fatal("retargeted copy changed original")
	}
	g.PhaseOut(victim.ID())
	g.ResolveStackObject(original)
	if b.Life() != 15 || a.prompts != 0 || a.ManaPool().TotalMana() != 0 {
		t.Fatal("original lost its total or paid again")
	}
}

func TestRONEvenDamageCatapultPaymentAndInstances(t *testing.T) {
	g, a, b := distributedDamageGame()
	victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 1, 20), b.PlayerID())
	first := repositoryCard(t, "FEM", "Dwarven Catapult")
	second := repositoryCard(t, "FEM", "Dwarven Catapult")
	for i, card := range []mage.Card{first, second} {
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		a.ManaPool().Add(core.Red, 1)
		x := 3 + 2*i
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, x); err == nil || a.ManaPool().Count(core.Red) != 1 || g.StackSize() != i {
			t.Fatal("failed payment changed resources")
		}
		a.ManaPool().Add(core.Colorless, x)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{a.PlayerID()}, x); err == nil || a.ManaPool().TotalMana() != x+1 {
			t.Fatal("own-player target accepted or paid")
		}
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, x); err != nil {
			t.Fatal(err)
		}
	}
	g.ResolveTopOfStack()
	if victim.Damage != 5 {
		t.Fatal("second instance lost X")
	}
	g.ResolveTopOfStack()
	if victim.Damage != 8 {
		t.Fatal("first instance reused second X")
	}
}

func TestRONEvenDamageInventoryAndRemaining(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		record := staticBaseRecord(t, set, "Fireball")
		factory, err := Compile(record)
		if err != nil {
			t.Fatal(err)
		}
		if err = mage.ValidateCardDefinition("Fireball", factory()); err != nil {
			t.Fatal(err)
		}
	}
	nodes, err := Parse(strings.NewReader(`Card(Name:"Inventory",Rules:[SpellActions(`+evenUnionBody+`)])`), "even-inventory.ron")
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string]*InventorySignature{}
	walkInventory(nodes[0], "Inventory", "Card", "card", "collection", false, entries)
	for name, want := range map[string]string{"AnyNumberOfAnyTargets": "target", "ThisSpell": "spell_reference", "ValueX": "value", "Ref_AnyTargets": "recipient_collection_reference"} {
		found := false
		for _, entry := range entries {
			if entry.Name == name && entry.Role == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s lost %s role", name, want)
		}
	}
}

func TestRONEvenDamageDeferredContexts(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`SpellDealsDamageDividedAmongRecipientsRoundedDown(ThisSpell,Integer(4),EachPermanent(IsCardtype(Creature)))`), "deferred-even.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, ctx := range []scope{{spellTargets: true, delayedStep: true}, {spellTargets: true, drawReplacementBody: true}} {
		if _, err = compileEvenSpellDamage(nodes[0], ctx); err == nil {
			t.Fatal("accepted a deferred or replacement spell body")
		}
	}
}
