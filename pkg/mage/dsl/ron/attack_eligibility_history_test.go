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

func TestRONAttackEligibilitySeason(t *testing.T) {
	for _, endPlayer := range []int{0, 1} {
		g, a, b := ronGame()
		card := repositoryCard(t, "DRK", "Season of the Witch")
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
			t.Fatal("unpaid cast succeeded")
		}
		if len(a.Hand()) != 1 || g.StackSize() != 0 {
			t.Fatal("unpaid cast changed state")
		}
		a.ManaPool().Add(core.Black, 3)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		idle := g.PutOnBattlefield(mage.NewCreature("Idle", "", 1, 8), a.PlayerID())
		secondIdle := g.PutOnBattlefield(mage.NewCreature("Second idle", "", 1, 8), a.PlayerID())
		attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 1, 8, mage.WithKeyword(core.Vigilance)), a.PlayerID())
		tapped := g.PutOnBattlefield(mage.NewCreature("Tapped", "", 1, 8), a.PlayerID())
		sick := g.PutOnBattlefield(mage.NewCreature("Sick", "", 1, 8), a.PlayerID())
		opponent := g.PutOnBattlefield(mage.NewCreature("Opponent", "", 1, 8), b.PlayerID())
		for _, p := range []*mage.Permanent{idle, secondIdle, attacker, tapped, opponent} {
			p.RevokeBaseAttr(core.AttrSummonSick)
		}
		g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{attacker.ID()})
		g.TapPermanent(tapped)
		late := g.PutOnBattlefield(mage.NewCreature("Late", "", 1, 8, mage.WithKeyword(core.Haste)), a.PlayerID())
		g.SetActivePlayerIndex(endPlayer)
		g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: g.ActivePlayerObj().PlayerID()})
		g.PutTriggersOnStack()
		if g.StackSize() != 1 {
			t.Fatal("missing end-step trigger")
		}
		clone := g.Clone()
		g.ResolveStack()
		if g.FindPermanent(idle.ID()) != nil || g.FindPermanent(secondIdle.ID()) != nil {
			t.Fatal("eligible idle creature survived")
		}
		for _, p := range []*mage.Permanent{attacker, tapped, sick, opponent, late} {
			if g.FindPermanent(p.ID()) == nil {
				t.Fatalf("%s destroyed", p.Name())
			}
		}
		if clone.FindPermanent(idle.ID()) == nil {
			t.Fatal("clone changed before resolution")
		}
		clone.ResolveStack()
		if clone.FindPermanent(idle.ID()) != nil {
			t.Fatal("pending clone lost history")
		}
	}
}

func TestRONAttackEligibilityInvalid(t *testing.T) {
	for _, form := range []string{"CouldntAttackThisTurn()", "CouldntAttackThisTurn(Integer(1))"} {
		nodes, err := Parse(strings.NewReader(form), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = compileContextualCollection(nodes[0], scope{resolving: true}); err == nil {
			t.Fatal("invalid history accepted")
		}
	}
	nodes, err := Parse(strings.NewReader("CouldntAttackThisTurn"), "context.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, ctx := range []scope{{}, {resolving: true, enteringSource: true}, {resolving: true, drawReplacementBody: true}} {
		if _, err := compileContextualCollection(nodes[0], ctx); err == nil {
			t.Fatal("unsupported context accepted")
		}
	}
	if _, err := compileFilter(nodes[0]); err == nil {
		t.Fatal("history accepted in generic filters")
	}
}

func TestRONAttackEligibilityLoad(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/DRK.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.HasPrefix(line, `Card(Name: "Season of the Witch"`) {
			record = line
			break
		}
	}
	if record == "" {
		t.Fatal("missing record")
	}
	files := fstest.MapFS{"season.ron": {Data: []byte(record)}}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, signature := range report.Signatures {
		if signature.Name == "CouldntAttackThisTurn" {
			found = true
			if signature.Role != "permanent_collection" || signature.CompiledOccurrences != 1 || signature.Occurrences != 1 {
				t.Fatalf("history collection inventory: %+v", signature)
			}
		}
	}
	if !found {
		t.Fatal("history collection absent from inventory")
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if !registry.CardRegistered("Season of the Witch") {
		t.Fatal("missing complete card")
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(strings.Replace(record, "Season of the Witch", "Bad", 1), "CouldntAttackThisTurn", "CouldntAttackThisTurn()"))}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("partial registry published")
	}
}

type ronSeasonPayer struct {
	*mage.BasePlayer
	accept bool
	offers int
}

func (p *ronSeasonPayer) ChooseMayAbility(string) bool { p.offers++; return p.accept }

func TestRONAttackEligibilityUpkeep(t *testing.T) {
	for _, accept := range []bool{false, true} {
		for _, life := range []int{1, 3} {
			a := &ronSeasonPayer{BasePlayer: mage.NewBasePlayer("A"), accept: accept}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			a.SetLife(life)
			card := repositoryCard(t, "DRK", "Season of the Witch")
			p := g.PutOnBattlefield(card, a.PlayerID())
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("opponent upkeep triggered")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("controller upkeep did not trigger")
			}
			g.ResolveStack()
			paid := accept && life >= 2
			wantLife := life
			if paid {
				wantLife -= 2
			}
			if (g.FindPermanent(p.ID()) != nil) != paid || a.Life() != wantLife || b.Life() != 20 {
				t.Fatal("life payment or sacrifice", accept, life, a.Life())
			}
		}
	}
}

func TestRONAttackEligibilityResolution(t *testing.T) {
	for _, change := range []string{"normal", "tap", "phase", "return", "control", "source-leaves", "regenerate", "indestructible", "exile-replacement", "skipped-combat", "late-source"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "DRK", "Season of the Witch")
			var source *mage.Permanent
			if change != "late-source" {
				source = g.PutOnBattlefield(card, a.PlayerID())
			}
			c := mage.NewCreature("Subject", "", 1, 8)
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			p.RevokeBaseAttr(core.AttrSummonSick)
			if change != "skipped-combat" {
				g.RunStepWithPriority(core.DeclareAttackers)
			}
			if change == "late-source" {
				source = g.PutOnBattlefield(card, a.PlayerID())
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			switch change {
			case "tap":
				g.TapPermanent(p)
			case "phase":
				g.PhaseOut(p.ID())
			case "return":
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(c.ID())
				p = g.PutOnBattlefield(c, a.PlayerID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "source-leaves":
				g.BouncePermanentToHand(source)
			case "regenerate":
				g.AddRegenerationShield(p.ID())
			case "indestructible":
				p.GrantBaseAttr(core.Indestructible)
			case "exile-replacement":
				g.AddExileIfWouldGoToGraveyardThisTurn(p.ID(), source.ID())
			}
			g.ResolveStack()
			survives := change == "tap" || change == "phase" || change == "return" || change == "regenerate" || change == "indestructible" || change == "skipped-combat"
			if (g.FindPermanentIncludingPhased(p.ID()) != nil) != survives {
				t.Fatal("wrong destruction result", change)
			}
			if change == "exile-replacement" && g.FindExiledCard(p.ID()) == nil {
				t.Fatal("replacement did not exile")
			}
			if change == "phase" {
				g.PhaseIn([]uuid.UUID{p.ID()})
				g.RunStepWithPriority(core.EndStep)
				if g.FindPermanent(p.ID()) != nil {
					t.Fatal("phase cycle erased attack opportunity")
				}
			}
		})
	}
}
