package ai

import (
	"testing"
	"uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/interactive"
)

func TestRespondToChoiceUsesMayPolicy(t *testing.T) {
	player := mage.NewBasePlayer("Human")
	for _, tc := range []struct {
		reason string
		want   bool
	}{
		{"draw a card", true}, {"pay {2} to mill yourself", false},
	} {
		got := RespondToChoice(player, nil, interactive.ChoiceRequest{Type: interactive.ChoiceMay, Reason: tc.reason})
		if got.Accepted != tc.want {
			t.Errorf("%s: got %v, want %v", tc.reason, got.Accepted, tc.want)
		}
	}
}

func TestRespondToChoiceUsesLivePlayerAndLegalOptions(t *testing.T) {
	player := mage.NewBasePlayer("Human")
	game := mage.NewGame(player, mage.NewBasePlayer("Opponent"))
	red := mage.NewCreature("Red", "{R}{R}", 2, 2)
	red.SetOwner(player.PlayerID())
	player.AddToHand(red)
	req := interactive.ChoiceRequest{Type: interactive.ChoiceManaColor, Options: []interactive.ChoiceOption{{Color: core.White}, {Color: core.Red}}}
	if got := RespondToChoice(player, game, req); got.SelectedColor != core.Red {
		t.Fatalf("mana choice = %v", got)
	}
	req.Options = req.Options[:1]
	if got := RespondToChoice(player, game, req); got.SelectedColor != core.White {
		t.Fatalf("illegal mana choice = %v", got)
	}
	req = interactive.ChoiceRequest{Type: interactive.ChoiceNumber, Reason: "deal damage", Options: []interactive.ChoiceOption{{Label: "3"}, {Label: "4"}, {Label: "5"}}}
	if got := RespondToChoice(player, game, req); got.SelectedIndex != 2 {
		t.Fatalf("number choice = %v", got)
	}
	req.Reason = "pay life"
	if got := RespondToChoice(player, game, req); got.SelectedIndex != 0 {
		t.Fatalf("cost choice = %v", got)
	}
}

func TestRespondToChoiceRanksOnlyOfferedCards(t *testing.T) {
	player := mage.NewBasePlayer("Human")
	game := mage.NewGame(player, mage.NewBasePlayer("Opponent"))
	land := mage.NewLand("Forest")
	spell := mage.NewCreature("Bear", "{1}{G}", 2, 2)
	hidden := mage.NewCreature("Cheap", "{0}", 0, 1)
	for _, card := range []*mage.Card{land, spell, hidden} {
		card.SetOwner(player.PlayerID())
		player.AddToHand(card)
	}
	req := interactive.ChoiceRequest{Type: interactive.ChoiceCardsFromHand, Reason: "discard", Amount: 2, Options: []interactive.ChoiceOption{{ID: land.ID()}, {ID: spell.ID()}}}
	got := RespondToChoice(player, game, req)
	if len(got.SelectedIDs) != 2 || got.SelectedIDs[0] != spell.ID() || got.SelectedIDs[1] != land.ID() {
		t.Fatalf("discard = %v", got)
	}
	if len(player.Hand()) != 3 {
		t.Fatal("choice changed the hand")
	}
}

func TestLondonBottomPreservesTwoLands(t *testing.T) {
	hand := make([]*mage.Card, 0, 7)
	hand = append(hand, mage.NewLand("Forest"), mage.NewLand("Forest"))
	for range 5 {
		hand = append(hand, mage.NewCreature("Bear", "{1}{G}", 2, 2))
	}
	for _, card := range LondonBottomCards(hand, 2) {
		if card.HasType(core.TypeLand) {
			t.Fatal("bottomed a needed land")
		}
	}
}

func TestMulliganLondonConservesCards(t *testing.T) {
	for _, lands := range []bool{false, true} {
		player := mage.NewBasePlayer("Bot")
		for range 20 {
			card := mage.NewCreature("Bear", "{1}{G}", 2, 2)
			if lands {
				card = mage.NewLand("Forest")
			}
			card.SetOwner(player.PlayerID())
			player.AddToLibrary(card)
		}
		for range 7 {
			player.DrawCard()
		}
		MulliganLondon(player, 0)
		if len(player.Hand()) != 5 || len(player.Library()) != 15 {
			t.Fatalf("hand=%d library=%d", len(player.Hand()), len(player.Library()))
		}
		seen := map[uuid.UUID]bool{}
		for _, cards := range [][]*mage.Card{player.Hand(), player.Library()} {
			for _, card := range cards {
				if seen[card.ID()] || card.Owner() != player.PlayerID() {
					t.Fatal("card duplicated or owner changed")
				}
				seen[card.ID()] = true
			}
		}
	}
}

func TestSuggestedCombatDamageAndOrder(t *testing.T) {
	blockers := []interactive.PermanentState{{ID: uuid.New(), Toughness: 3, Damage: 1}, {ID: uuid.New(), Toughness: 2}, {ID: uuid.New(), Toughness: 4}}
	got := SuggestedCombatDamage(blockers, 5)
	if got[blockers[0].ID] != 2 || got[blockers[1].ID] != 2 || got[blockers[2].ID] != 1 {
		t.Fatalf("damage=%v", got)
	}
	order := CombatDamageOrder(blockers, map[uuid.UUID]int{blockers[0].ID: 1, blockers[1].ID: 2, blockers[2].ID: 2})
	if order[0] != blockers[1].ID {
		t.Fatal("lethal blocker must come first")
	}
	if blockers[0].Toughness != 3 {
		t.Fatal("order changed caller's slice")
	}
}

func TestRespondToChoiceModesPermanentsTargetsAndLibrary(t *testing.T) {
	game, bot, opponent := aiGame()
	bot.SetLife(20)
	mode := interactive.ChoiceRequest{Type: interactive.ChoiceMode, Options: []interactive.ChoiceOption{{Label: "Gain 3 life"}, {Label: "Prevent 3 damage"}}}
	if got := RespondToChoice(bot.BasePlayer, game, mode); got.SelectedIndex != 1 {
		t.Fatalf("mode=%v", got)
	}
	strong := battlefieldCreature(game, bot.PlayerID(), "Dragon", 5, 5)
	weak := battlefieldCreature(game, bot.PlayerID(), "Bird", 1, 1)
	request := interactive.ChoiceRequest{Type: interactive.ChoicePermanent, Reason: "sacrifice a creature", Options: []interactive.ChoiceOption{{ID: strong.ID()}, {ID: weak.ID()}}}
	if got := RespondToChoice(bot.BasePlayer, game, request); len(got.SelectedIDs) != 1 || got.SelectedIDs[0] != weak.ID() {
		t.Fatalf("sacrifice=%v", got)
	}
	request.Reason = "Choose target"
	request.Options = append(request.Options, interactive.ChoiceOption{ID: opponent.PlayerID()})
	if got := RespondToChoice(bot.BasePlayer, game, request); len(got.SelectedIDs) != 1 || got.SelectedIDs[0] != opponent.PlayerID() {
		t.Fatalf("target=%v", got)
	}
	land := mage.NewLand("Forest")
	spell := mage.NewCreature("Dragon", "{4}{R}{R}", 5, 5)
	for _, card := range []*mage.Card{land, spell} {
		card.SetOwner(bot.PlayerID())
		bot.AddToLibrary(card)
	}
	request = interactive.ChoiceRequest{Type: interactive.ChoiceCardFromLibrary, Options: []interactive.ChoiceOption{{ID: land.ID()}, {ID: spell.ID()}, {Label: "Done"}}}
	if got := RespondToChoice(bot.BasePlayer, game, request); len(got.SelectedIDs) != 1 || got.SelectedIDs[0] != spell.ID() {
		t.Fatalf("library=%v", got)
	}
	request.Options = request.Options[2:]
	if got := RespondToChoice(bot.BasePlayer, game, request); len(got.SelectedIDs) != 0 {
		t.Fatalf("done=%v", got)
	}
}

func TestMulliganLondonKeepsGoodHandAndBottomsPriorMulligans(t *testing.T) {
	for _, prior := range []int{0, 1, 2, 7} {
		player := mage.NewBasePlayer("Bot")
		for i := range 7 {
			card := mage.NewCreature("Bear", "{1}{G}", 2, 2)
			if i < 2 {
				card = mage.NewLand("Forest")
			}
			card.SetOwner(player.PlayerID())
			player.AddToHand(card)
		}
		MulliganLondon(player, prior)
		if len(player.Hand()) != 7-prior || len(player.Library()) != prior {
			t.Fatalf("prior=%d hand=%d library=%d", prior, len(player.Hand()), len(player.Library()))
		}
		if prior <= 2 {
			lands := 0
			for _, card := range player.Hand() {
				if card.HasType(core.TypeLand) {
					lands++
				}
			}
			if lands != 2 {
				t.Fatalf("prior=%d kept %d lands", prior, lands)
			}
		}
	}
}
