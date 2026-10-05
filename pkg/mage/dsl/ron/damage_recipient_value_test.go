package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONDamageRecipientValueCards(t *testing.T) {
	for _, name := range []string{"Storm Seeker", "Typhoon"} {
		for _, self := range []bool{false, true} {
			t.Run(name+map[bool]string{false: "/opponent", true: "/self"}[self], func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, "LEG", name)
				if err := mage.ValidateCardDefinition(name, card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				a.ManaPool().Add(core.Green, 1)
				a.ManaPool().Add(core.Colorless, 3)
				b.AddToHand(mage.NewInstant("First", ""))
				target := b.PlayerID()
				if self {
					target = a.PlayerID()
				}
				targets := []uuid.UUID{target}
				if name == "Typhoon" {
					targets = nil
				}
				island := g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), b.PlayerID())
				g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), a.PlayerID())
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, 0); err != nil {
					t.Fatal(err)
				}
				if a.Life() != 20 || b.Life() != 20 {
					t.Fatal("announcement dealt damage")
				}
				b.AddToHand(mage.NewInstant("Second", ""))
				a.AddToHand(mage.NewInstant("Caster card", ""))
				g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), b.PlayerID())
				phased := g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), b.PlayerID())
				g.PhaseOut(phased.ID())
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: island.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				clone := g.Clone()
				clone.ResolveStack()
				if a.Life() != 20 || b.Life() != 20 {
					t.Fatal("clone changed live game")
				}
				g.ResolveStack()
				wantA, wantB := 20, 18
				if name == "Typhoon" {
					wantB = 19
				} else if self {
					wantA, wantB = 19, 20
				}
				for _, game := range []*mage.Game{g, clone} {
					if game.GetPlayer(a.PlayerID()).Life() != wantA || game.GetPlayer(b.PlayerID()).Life() != wantB {
						t.Fatalf("life %d/%d want %d/%d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life(), wantA, wantB)
					}
				}
			})
		}
	}
}

func TestRONDamageRecipientValueCopies(t *testing.T) {
	for _, name := range []string{"Storm Seeker", "Typhoon"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			for range 3 {
				b.AddToHand(mage.NewInstant("Opponent card", ""))
			}
			a.AddToHand(mage.NewInstant("Caster card", ""))
			g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), a.PlayerID())
			for range 2 {
				g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), b.PlayerID())
			}
			card := repositoryCard(t, "LEG", name)
			targets := []uuid.UUID{b.PlayerID()}
			if name == "Typhoon" {
				targets = nil
			}
			castRONSpell(t, g, a, card, targets...)
			clone := g.Clone()
			copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), name == "Storm Seeker")
			if copied == nil {
				t.Fatal("missing copy")
			}
			if name == "Storm Seeker" && (len(copied.Targets) != 1 || copied.Targets[0] != a.PlayerID()) {
				t.Fatal("copy not retargeted")
			}
			g.ResolveStack()
			clone.ResolveStack()
			wantA, wantB := 19, 17
			if name == "Typhoon" {
				wantB = 18
			}
			if a.Life() != wantA || b.Life() != wantB {
				t.Fatalf("copy life %d/%d", a.Life(), b.Life())
			}
			if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != wantB {
				t.Fatal("clone target or controller")
			}
		})
	}
}

func TestRONDamageRecipientValueRollbackAndPrevention(t *testing.T) {
	for _, name := range []string{"Storm Seeker", "Typhoon"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", name)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			targets := []uuid.UUID{b.PlayerID()}
			if name == "Typhoon" {
				targets = nil
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, 0); err == nil {
				t.Fatal("unpaid cast")
			}
			if len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("payment rollback")
			}
			a.ManaPool().Add(core.Green, 1)
			a.ManaPool().Add(core.Colorless, 3)
			if name == "Storm Seeker" {
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{uuid.New()}, 0); err == nil {
					t.Fatal("invalid target")
				}
				if a.ManaPool().TotalMana() != 4 || len(a.Hand()) != 1 {
					t.Fatal("target rollback")
				}
			} else {
				g.SetStep(core.DeclareAttackers)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
					t.Fatal("sorcery timing")
				}
				if a.ManaPool().TotalMana() != 4 || len(a.Hand()) != 1 {
					t.Fatal("timing rollback")
				}
				g.SetStep(core.PrecombatMain)
			}
			for range 3 {
				b.AddToHand(mage.NewInstant("Hand card", ""))
				g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), b.PlayerID())
			}
			g.AddPreventionShield(b.PlayerID(), 2)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if a.Life() != 20 || b.Life() != 19 {
				t.Fatal("damage did not use prevention")
			}
		})
	}
}

func TestRONDamageRecipientValueLocalScopes(t *testing.T) {
	rule := `SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([SpellDealsDamage(ThisSpell,TheNumberOfCardsInPlayersHand(DealsDamage_ThatPlayer),Player(Ref_TargetPlayer)),SpellDealsDamage(ThisSpell,TheNumberOfCardsInPlayersHand(DealsDamage_ThatPlayer),Player(You))])))`
	factory := permanentScopeFactory(t, "Instant", rule)
	first, second := factory(), factory()
	if first.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == second.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
		t.Fatal("shared factories")
	}
	g, a, b := ronGame()
	a.AddToHand(mage.NewInstant("Caster card", ""))
	for range 2 {
		b.AddToHand(mage.NewInstant("Opponent card", ""))
	}
	castRONSpell(t, g, a, first, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 19 || b.Life() != 18 {
		t.Fatal("adjacent damage amounts share recipient")
	}
	for _, rule := range []string{
		`SpellActions(ActionList([SpellDealsDamage(ThisSpell,TheNumberOfCardsInPlayersHand(DealsDamage_ThatPlayer),EachPlayer(AnyPlayer)),SpellDealsDamage(ThisSpell,TheNumberOfCardsInPlayersHand(DealsDamage_ThatPlayer),EachPlayer(Opponent))]))`,
		`SpellActions(Modal_ChooseOne([ActionList([SpellDealsDamage(ThisSpell,TheNumberOfCardsInPlayersHand(DealsDamage_ThatPlayer),EachPlayer(AnyPlayer))]),ActionList([SpellDealsDamage(ThisSpell,Integer(1),Player(You))])]))`,
	} {
		card := permanentScopeFactory(t, "Instant", rule)()
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRONDamageRecipientValueInvalid(t *testing.T) {
	for _, body := range []string{
		`SpellDealsDamage(ThisSpell,TheNumberOfCardsInPlayersHand(DealsDamage_ThatPlayer),Permanent(Ref_TargetPermanent))`,
		`SpellDealsDamage(ThisSpell,TheNumberOfCardsInPlayersHand(DealsDamage_ThatPlayer),MultipleRecipients([EachPlayer(AnyPlayer),EachPermanent(IsCardtype(Creature))]))`,
		`SpellDealsDamage(ThisSpell,Integer(1),EachPlayer)`,
		`SpellDealsDamage(ThisSpell,Integer(1),EachPlayer())`,
		`SpellDealsDamage(ThisSpell,Integer(1),EachPlayer(AnyPlayer()))`,
		`SpellDealsDamage(ThisSpell,Integer(1),EachPlayer(You))`,
		`SpellDealsDamage(ThisSpell,Integer(1),EachPlayer(SinglePlayer(You)))`,
		`SpellDealsDamage(ThisSpell,Integer(1),EachPlayer(AnyPlayer,Opponent))`,
		`SpellDealsDamage(ThisSpell,Integer(1),Player(DealsDamage_ThatPlayer))`,
		`SpellDealsDamage(ThisSpell,TheNumberOfCardsInPlayersHand(DealsDamage_ThatPlayer()),Player(You))`,
		`SpellDealsDamage(ThisSpell,TheNumberOfCardsInPlayersHand(DealsDamage_ThatPlayer),Player(You)),GainLife(TheNumberOfCardsInPlayersHand(DealsDamage_ThatPlayer))`,
		`GainLife(TheNumberOfCardsInPlayersHand(DealsDamage_ThatPlayer)),SpellDealsDamage(ThisSpell,Integer(1),EachPlayer(AnyPlayer))`,
		`SpellDealsDamage(ThisSpell,Integer(1),Player(You)),CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([GainLife(TheNumberOfCardsInPlayersHand(DealsDamage_ThatPlayer))]))`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([` + body + `])))])`
		nodes, err := Parse(strings.NewReader(src), "invalid-damage.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRONDamageRecipientValueAllPlayers(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([SpellDealsDamage(ThisSpell,TheNumberOfCardsInPlayersHand(DealsDamage_ThatPlayer),EachPlayer(AnyPlayer))]))`)
	g, a, b := ronGame()
	a.AddToHand(mage.NewInstant("A hand", ""))
	for range 2 {
		b.AddToHand(mage.NewInstant("B hand", ""))
	}
	castRONSpell(t, g, a, factory())
	events := 0
	g.SetOnDamageDealt(func(_, _ string, _ int, _ bool) {
		events++
		if a.Life() != 19 || b.Life() != 18 {
			t.Error("partial player batch")
		}
	})
	g.ResolveStack()
	if events != 2 {
		t.Fatal("player-specific amounts")
	}
}

func TestRONDamageRecipientValueActivation(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(TapPermanent(ThisPermanent),Targeted([TargetPlayer(AnyPlayer)],ActionList([PermanentDealsDamage(ThisPermanent,TheNumberOfCardsInPlayersHand(DealsDamage_ThatPlayer),Player(Ref_TargetPlayer))])))`)
	g, a, b := ronGame()
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	g.SetTurn(2)
	b.AddToHand(mage.NewInstant("B hand", ""))
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(source)
	clone := g.Clone()
	clone.ResolveStack()
	g.ResolveStack()
	if b.Life() != 19 || clone.GetPlayer(b.PlayerID()).Life() != 19 || a.Life() != 20 {
		t.Fatal("departed damage source")
	}
}

func TestRONDamageRecipientValueInventory(t *testing.T) {
	src := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([UnknownAction,SpellDealsDamage(ThisSpell,TheNumberOfCardsInPlayersHand(DealsDamage_ThatPlayer),EachPlayer(Opponent))]))])`
	nodes, err := Parse(strings.NewReader(src), "inventory.ron")
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string]*InventorySignature{}
	walkInventory(nodes[0], "Inventory", "Card", "card", "collection", true, entries)
	for name, role := range map[string]string{"DealsDamage_ThatPlayer": "player_reference", "Opponent": "player_filter", "EachPlayer": "recipient"} {
		found := false
		for _, entry := range entries {
			if entry.Name == name && entry.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing %s/%s", name, role)
		}
	}
	if _, err := Compile(nodes[0]); err == nil {
		t.Fatal("accepted unsupported earlier action")
	}
}
