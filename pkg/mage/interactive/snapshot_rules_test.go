package interactive

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSnapshotOmitsRulesText(t *testing.T) {
	game, player, _ := makeGame()
	card := mage.NewCreature("Bird", "{U}", 1, 1, mage.WithKeyword(core.Flying))
	player.AddToHand(card)
	game.AddToBattlefield(mage.NewPermanent(card.NewCopy(), player.PlayerID()))

	data, err := json.Marshal(SnapshotGameState(game, 0))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte("\"RulesText\":")) {
		t.Fatal("snapshot JSON includes RulesText")
	}
}
