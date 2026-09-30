package mage

import (
	"maps"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func setupCopyDivisionRoute(g *Game, a Player, card Card, route string) (func() error, func() bool) {
	if route == "prepared" {
		source := g.PutOnBattlefield(NewCreature("Prepared source", "", 2, 2), a.PlayerID())
		g.SetPrepared(source.ID(), true)
		return func() error {
			return g.CastPreparedSpellCopy(a.PlayerID(), source.ID(), func() Card { return card.Copy() })
		}, func() bool { return g.IsPrepared(source.ID()) }
	}
	card.SetOwner(a.PlayerID())
	g.ExileCard(card, uuid.Nil)
	g.RegisterParadigmExiledCopy(a.PlayerID(), card.Name(), card.ID())
	return func() error { _, err := g.CastParadigmCopyFromExile(a.PlayerID(), card.Name()); return err }, func() bool { return g.FindExiledCard(card.ID()) != nil }
}

func TestCopyCastDamageDivisionAdmission(t *testing.T) {
	for _, route := range []string{"prepared", "paradigm"} {
		for _, state := range []string{"negative", "zero", "insufficient", "absent", "phased", "duplicate", "wrong type", "missing required"} {
			t.Run(route+"/"+state, func(t *testing.T) {
				g, a, b := newActivationDivisionGame()
				total := 3
				switch state {
				case "negative":
					total = -1
				case "zero":
					total = 0
				case "insufficient":
					total = 1
				}
				spec := TargetUpToNCreaturesOrPlayers(2)
				if state == "missing required" {
					spec = TargetCreatureOpponentControls()
				}
				card := NewInstant("Rejected copy division", "", NewMultiTargetSpell([]Target{spec}, DealDividedDamage(Fixed(total))))
				victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 2), b.PlayerID())
				targets := []uuid.UUID{a.PlayerID(), b.PlayerID()}
				switch state {
				case "absent":
					targets = []uuid.UUID{uuid.New()}
				case "phased":
					g.PhaseOut(victim.ID())
					targets = []uuid.UUID{victim.ID()}
				case "duplicate":
					targets = []uuid.UUID{victim.ID(), victim.ID()}
				case "wrong type":
					targets = []uuid.UUID{g.PutOnBattlefield(NewArtifact("Other", ""), b.PlayerID()).ID()}
				case "missing required":
					g.ExilePermanent(victim)
					targets = nil
				}
				a.chooseQueue = [][]uuid.UUID{targets}
				a.choose = func(ids []uuid.UUID, amount int, _ *Game) map[uuid.UUID]int {
					return sanitizeDamageDistribution(nil, ids, amount)
				}
				events := 0
				for _, event := range []EventType{EvtSpellCast, EvtBecomesTarget} {
					g.PutOnBattlefield(NewEnchantment("Observer", "", WithAbility(NewTriggered(event, false, GainLifeFor(You, Fixed(1))).SetCondition(func(_ *GameEvent, _ GameReader, _, _ uuid.UUID) bool { events++; return false }))), a.PlayerID())
				}
				cast, retained := setupCopyDivisionRoute(g, a, card, route)
				if err := cast(); err == nil {
					t.Fatal("accepted invalid copy announcement")
				}
				if !retained() || a.divisions != 0 || g.StackSize() != 0 || events != 0 || a.Life() != 20 || b.Life() != 20 {
					t.Fatal("failed announcement changed state or prompted allocation")
				}
			})
		}
	}
}

func TestCopyCastDamageDivisionIsolation(t *testing.T) {
	for _, route := range []string{"prepared", "paradigm"} {
		t.Run(route, func(t *testing.T) {
			g, a, b := newActivationDivisionGame()
			card := NewInstant("Copy division", "", NewMultiTargetSpell([]Target{TargetUpToNCreaturesOrPlayers(2)}, DealDividedDamage(Fixed(4))))
			cast, retained := setupCopyDivisionRoute(g, a, card, route)
			a.chooseQueue = [][]uuid.UUID{{a.PlayerID(), b.PlayerID()}}
			response := map[uuid.UUID]int{a.PlayerID(): 1, b.PlayerID(): 3}
			a.choose = func(_ []uuid.UUID, total int, game *Game) map[uuid.UUID]int {
				if total != 4 || !retained() || game.StackSize() != 0 {
					t.Error("allocation occurred after cast publication")
				}
				return response
			}
			if err := cast(); err != nil {
				t.Fatal(err)
			}
			obj := g.StackPeek()
			if a.divisions != 1 || !maps.Equal(obj.DamageDistribution, response) || !obj.IsCopy || obj.Controller != a.PlayerID() {
				t.Fatal("copy lost allocation or controller")
			}
			if route == "prepared" && retained() {
				t.Fatal("successful copy did not unprepare source")
			}
			if route == "paradigm" && !retained() {
				t.Fatal("successful copy moved original")
			}
			response[b.PlayerID()] = 99
			clone := g.Clone()
			copied := clone.CopyStackObjectDirect(clone.StackPeek(), a.PlayerID(), false)
			copied.DamageDistribution[b.PlayerID()] = 7
			if obj.DamageDistribution[b.PlayerID()] != 3 || clone.StackObjects()[0].DamageDistribution[b.PlayerID()] != 3 {
				t.Fatal("allocation shares response, clone, or copy state")
			}
			g.ResolveStack()
			if a.Life() != 19 || b.Life() != 17 {
				t.Fatal("copy lost announced shares at resolution")
			}
		})
	}
}

func TestCopyCastDamageDivisionEmptyTargets(t *testing.T) {
	for _, route := range []string{"prepared", "paradigm"} {
		for _, total := range []int{-1, 0, 3} {
			t.Run(route+"/"+Fixed(total).Text(), func(t *testing.T) {
				g, a, _ := newActivationDivisionGame()
				card := NewInstant("Empty copy", "", NewMultiTargetSpell([]Target{TargetUpToNCreaturesOrPlayers(2)}, DealDividedDamage(Fixed(total))))
				cast, _ := setupCopyDivisionRoute(g, a, card, route)
				a.chooseQueue = [][]uuid.UUID{nil}
				if err := cast(); err != nil {
					t.Fatal(err)
				}
				if a.divisions != 0 || len(g.StackPeek().DamageDistribution) != 0 {
					t.Fatal("empty target list prompted allocation")
				}
			})
		}
	}
}

func TestCopyCastDamageDivisionSelectedMode(t *testing.T) {
	for _, route := range []string{"prepared", "paradigm"} {
		t.Run(route, func(t *testing.T) {
			g, a, b := newActivationDivisionGame()
			card := NewInstant("Modal copy", "", WithAbility(NewModalSpell([]Mode{
				{Label: "Divide", Targets: []Target{TargetUpToNCreaturesOrPlayers(1)}, Effects: []Effect{DealDividedDamage(Fixed(3))}},
				{Label: "Inactive", Effects: []Effect{DealDividedDamage(Fixed(-1))}},
			})))
			cast, _ := setupCopyDivisionRoute(g, a, card, route)
			a.chooseQueue = [][]uuid.UUID{{b.PlayerID()}}
			a.choose = func(_ []uuid.UUID, total int, _ *Game) map[uuid.UUID]int {
				return map[uuid.UUID]int{b.PlayerID(): total}
			}
			if err := cast(); err != nil {
				t.Fatal(err)
			}
			if a.divisions != 1 || g.StackPeek().ModeChoice != 0 || g.StackPeek().DamageDistribution[b.PlayerID()] != 3 {
				t.Fatal("wrong selected-mode allocation")
			}
		})
	}
}

func TestCopyCastDamageDivisionPreparedXTargets(t *testing.T) {
	g, a, b := newActivationDivisionGame()
	g.resolution.SetX(2)
	first := g.PutOnBattlefield(NewCreature("First", "", 2, 5), b.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 2, 5), b.PlayerID())
	card := NewInstant("X copy", "", NewMultiTargetSpell([]Target{TargetXCreatures()}, DealDividedDamage(Fixed(4))))
	cast, _ := setupCopyDivisionRoute(g, a, card, "prepared")
	a.chooseQueue = [][]uuid.UUID{{first.ID(), second.ID()}}
	a.choose = func(ids []uuid.UUID, amount int, _ *Game) map[uuid.UUID]int {
		return sanitizeDamageDistribution(nil, ids, amount)
	}
	if err := cast(); err != nil {
		t.Fatal(err)
	}
	if len(a.calls) != 1 || a.calls[0].min != 2 || a.calls[0].max != 2 || g.StackPeek().XValue != 2 || len(g.StackPeek().DamageDistribution) != 2 {
		t.Fatal("copy target prompt did not use stored X")
	}
}

type copyDivisionModePlayer struct {
	*activationDivisionPlayer
}

func (*copyDivisionModePlayer) ChooseMode([]string, string) int { return 1 }

func TestCopyCastDamageDivisionInactiveMode(t *testing.T) {
	for _, route := range []string{"prepared", "paradigm"} {
		t.Run(route, func(t *testing.T) {
			a := &copyDivisionModePlayer{activationDivisionPlayer: &activationDivisionPlayer{recordingPlayer: newRecordingPlayer("A")}}
			b := newRecordingPlayer("B")
			g := NewGame(a, b)
			card := NewInstant("Inactive division copy", "", WithAbility(NewModalSpell([]Mode{
				{Label: "Impossible", Targets: []Target{TargetCreature()}, Effects: []Effect{DealDividedDamage(Fixed(0))}},
				{Label: "Draw", Effects: []Effect{DrawCards(Fixed(1))}},
			})))
			cast, _ := setupCopyDivisionRoute(g, a, card, route)
			if err := cast(); err != nil {
				t.Fatal(err)
			}
			if a.divisions != 0 || g.StackPeek().ModeChoice != 1 || len(g.StackPeek().DamageDistribution) != 0 {
				t.Fatal("inactive mode affected division")
			}
		})
	}
}
