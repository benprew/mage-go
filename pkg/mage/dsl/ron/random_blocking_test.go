package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronPilePlayer struct {
	*mage.BasePlayer
	choices []int
}

func (p *ronPilePlayer) ChooseMode(_ []string, _ string) int {
	if len(p.choices) == 0 {
		return 0
	}
	n := p.choices[0]
	p.choices = p.choices[1:]
	return n
}
func TestRONRandomBlockingPiles(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			card := repositoryCard(t, set, "Camouflage")
			a := mage.NewBasePlayer("A")
			b := &ronPilePlayer{BasePlayer: mage.NewBasePlayer("B"), choices: []int{1}}
			g := mage.NewGame(a, b)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("cast outside attackers step")
			}
			g.SetStep(core.DeclareAttackers)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("cast without mana")
			}
			a.ManaPool().Add(core.Green, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if b.Life() != 20 || a.ManaPool().TotalMana() != 0 {
				t.Fatal("bad cast payment")
			}
			g.ResolveStack()
			attacker := g.PutOnBattlefield(mage.NewCreature("Late attacker", "", 3, 8), a.PlayerID())
			attacker.RevokeBaseAttr(core.AttrSummonSick)
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{attacker.ID()})
			blocker := g.PutOnBattlefield(mage.NewCreature("Late blocker", "", 1, 8), b.PlayerID())
			g.SetStep(core.DeclareBlockers)
			g.ExecuteBlockers(nil)
			if !g.GetCombat().IsBlocking(blocker.ID()) || len(a.Graveyard()) != 1 {
				t.Fatal("complete RON did not replace blocking")
			}
			g.ExecuteCombatDamage()
			if b.Life() != 20 {
				t.Fatal("random block did not prevent player damage")
			}
		})
	}
}
func TestRONRandomBlockingPilesInvalid(t *testing.T) {
	record := randomBlockingRecord(t)
	valid := `CreateGameEffect(UntilEndOfTurn, DefendingPlayersChooseCreaturesToDefendAttackersAtRandom)`
	for _, replacement := range []string{`CreateGameEffect(UntilEndOfTurn)`, `CreateGameEffect(UntilEndOfCombat,DefendingPlayersChooseCreaturesToDefendAttackersAtRandom)`, `CreateGameEffect(UntilEndOfTurn(),DefendingPlayersChooseCreaturesToDefendAttackersAtRandom)`, `CreateGameEffect(UntilEndOfTurn,DefendingPlayersChooseCreaturesToDefendAttackersAtRandom())`, `CreateGameEffect(UntilEndOfTurn,Unknown)`} {
		nodes, err := Parse(strings.NewReader(strings.Replace(record, valid, replacement, 1)), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted", replacement)
		}
	}
	nodes, err := Parse(strings.NewReader(valid), "context.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, ctx := range []scope{{}, {resolving: true, enteringSource: true}, {resolving: true, drawReplacementBody: true}} {
		if _, err := compileRandomBlockingPiles(nodes[0], ctx); err == nil {
			t.Fatal("accepted invalid context")
		}
	}
}
func TestRONRandomBlockingPilesInventory(t *testing.T) {
	files := fstest.MapFS{"card.ron": {Data: []byte(randomBlockingRecord(t))}}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal(report, err)
	}
	for name, role := range map[string]string{"UntilEndOfTurn": "duration", "DefendingPlayersChooseCreaturesToDefendAttackersAtRandom": "temporary_game_operation"} {
		found := false
		for _, signature := range report.Signatures {
			found = found || signature.Name == name && signature.Role == role
		}
		if !found {
			t.Fatal("missing semantic role", name)
		}
	}
}

func randomBlockingRecord(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../../../cards/mtgish/LEA.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Camouflage"`) {
			return line
		}
	}
	t.Fatal("missing complete record")
	return ""
}

func TestRONRandomBlockingPilesTimingAndCopies(t *testing.T) {
	for _, step := range []core.PhaseStep{core.Upkeep, core.PrecombatMain, core.DeclareAttackers, core.DeclareBlockers, core.EndCombat} {
		for _, opponent := range []bool{false, true} {
			g, a, b := ronGame()
			g.SetStep(step)
			player := a
			if opponent {
				player = b
			}
			card := repositoryCard(t, "LEA", "Camouflage")
			card.SetOwner(player.PlayerID())
			player.AddToHand(card)
			player.ManaPool().Add(core.Green, 1)
			legal := step == core.DeclareAttackers && !opponent
			if err := g.CastSpellByID(player.PlayerID(), card.ID(), nil, 0); (err == nil) != legal {
				t.Fatal("wrong timing", step, opponent, err)
			}
			if !legal && (len(player.Hand()) != 1 || player.ManaPool().TotalMana() != 1 || g.StackSize() != 0) {
				t.Fatal("failed casting changed resources")
			}
		}
	}
	for _, counter := range []bool{false, true} {
		a := mage.NewBasePlayer("A")
		b := &ronPilePlayer{BasePlayer: mage.NewBasePlayer("B"), choices: []int{1}}
		g := mage.NewGame(a, b)
		card := repositoryCard(t, "LEA", "Camouflage")
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		a.ManaPool().Add(core.Green, 1)
		g.SetStep(core.DeclareAttackers)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		if counter {
			g.CounterStackObject(g.StackPeek().ID)
		} else if g.CopySpellOnStack(card.ID(), a.PlayerID(), false) == nil {
			t.Fatal("copy failed")
		}
		clone := g.Clone()
		clone.ResolveStack()
		g.ResolveStack()
		attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 1, 8), a.PlayerID())
		attacker.RevokeBaseAttr(core.AttrSummonSick)
		blocker := g.PutOnBattlefield(mage.NewCreature("Blocker", "", 1, 8), b.PlayerID())
		g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{attacker.ID()})
		g.ExecuteBlockers(nil)
		if g.GetCombat().IsBlocking(blocker.ID()) == counter {
			t.Fatal("counter or duplicate spell installed the wrong procedure")
		}
		if a.ManaPool().TotalMana() != 0 || len(a.Graveyard()) != 1 {
			t.Fatal("copy changed payment or zones")
		}
	}
}
