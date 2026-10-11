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

func TestRONBoundedDelayedStepGlyphOfDoom(t *testing.T) {
	for _, change := range []string{"none", "after", "skipped", "wall-departure", "wall-return", "wall-phase", "attacker-return", "attacker-phase", "phase-cycle", "control", "regenerate", "copy", "clone", "illegal", "counter"} {
		t.Run(change, func(t *testing.T) {
			factory, err := Compile(hostUntapRecord(t, "LEG", "Glyph of Doom"))
			if err != nil {
				t.Fatal(err)
			}
			if err := mage.ValidateCardDefinition("Glyph of Doom", factory()); err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			wallCard := repositoryCard(t, "LEB", "Wall of Wood")
			wallCard.SetOwner(b.PlayerID())
			wall := g.PutOnBattlefield(wallCard, b.PlayerID())
			attackerCard := repositoryCard(t, "LEB", "Llanowar Elves")
			attackerCard.SetOwner(a.PlayerID())
			attacker := g.PutOnBattlefield(attackerCard, a.PlayerID())
			spared := g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), a.PlayerID())
			g.GetCombat().AddAttacker(attacker.ID(), b.PlayerID())
			g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: wall.ID(), AttackerID: attacker.ID()}})
			if change == "after" {
				g.RunStepWithPriority(core.EndCombat)
			}
			spell := factory()
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{wall.ID()}, 0); err == nil {
				t.Fatal("unpaid spell cast")
			}
			if len(a.Hand()) != 1 || len(g.DelayedTriggers()) != 0 {
				t.Fatal("failed payment changed state")
			}
			a.ManaPool().Add(core.Black, 1)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{wall.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			if change == "copy" && g.CopySpellOnStack(spell.ID(), a.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			if change == "illegal" {
				g.PhaseOut(wall.ID())
			}
			g.ResolveStack()
			wantSchedules := 1
			if change == "copy" {
				wantSchedules = 2
			}
			if change == "illegal" {
				wantSchedules = 0
			}
			if len(g.DelayedTriggers()) != wantSchedules || g.FindPermanent(attacker.ID()) == nil {
				t.Fatal("wrong schedule count or early destruction")
			}
			clone := g.Clone()
			switch change {
			case "wall-departure":
				g.BouncePermanentToHand(g.FindPermanent(wall.ID()))
			case "wall-return":
				g.BouncePermanentToHand(g.FindPermanent(wall.ID()))
				b.RemoveFromHand(wallCard.ID())
				g.PutOnBattlefield(wallCard, b.PlayerID())
			case "wall-phase":
				g.PhaseOut(wall.ID())
			case "attacker-return":
				g.BouncePermanentToHand(g.FindPermanent(attacker.ID()))
				a.RemoveFromHand(attackerCard.ID())
				g.PutOnBattlefield(attackerCard, a.PlayerID())
			case "attacker-phase":
				g.PhaseOut(attacker.ID())
			case "phase-cycle":
				p := g.PhaseOut(attacker.ID())
				g.PhaseIn(p)
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: wall.ID(), TargetID: attacker.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "regenerate":
				g.AddRegenerationShield(attacker.ID())
			}
			if change == "after" || change == "skipped" {
				g.RunStepWithPriority(core.Cleanup)
			}
			if change == "counter" {
				g.FireEvent(core.GameEvent{Type: core.EvtEndOfCombat, PlayerID: a.PlayerID()})
				g.PutTriggersOnStack()
				g.GetStack().RemoveByID(g.StackPeek().ID)
				g.ResolveStack()
			} else {
				g.RunStepWithPriority(core.EndCombat)
			}
			destroyed := change != "after" && change != "skipped" && change != "attacker-return" && change != "attacker-phase" && change != "regenerate" && change != "illegal" && change != "counter"
			if (g.FindPermanentIncludingPhased(attacker.ID()) == nil) != destroyed || g.FindPermanent(spared.ID()) == nil || len(g.DelayedTriggers()) != 0 {
				t.Fatal("wrong delayed history outcome")
			}
			g.RunStepWithPriority(core.EndCombat)
			if (g.FindPermanentIncludingPhased(attacker.ID()) == nil) != destroyed {
				t.Fatal("schedule repeated")
			}
			clone.RunStepWithPriority(core.EndCombat)
			if (clone.FindPermanent(attacker.ID()) == nil) != (change != "illegal") {
				t.Fatal("clone lost original captured history")
			}
		})
	}
}

func TestRONBoundedDelayedStepInvalid(t *testing.T) {
	for _, event := range []string{`AtNextEndOfCombatThisTurn()`, `AtNextEndOfCombatThisTurn(You)`, `AtNextEndOfCombatThisTurn(Integer(1))`} {
		src := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([CreateFutureTrigger(` + event + `,ActionList([GainLife(Integer(1))]))]))])`
		nodes, err := Parse(strings.NewReader(src), "bad-bounded.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted invalid event", event)
		}
	}
	for _, body := range []string{`ActionList([])`, `ActionList([DestroyPermanent(Ref_TargetPermanent)])`, `Targeted([TargetPermanent(AnyPermanent)],ActionList([DestroyPermanent(Ref_TargetPermanent)]))`} {
		src := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([CreateFutureTrigger(AtNextEndOfCombatThisTurn,` + body + `)]))])`
		nodes, err := Parse(strings.NewReader(src), "bad-bounded.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted invalid body", body)
		}
	}
}

func TestRONBoundedDelayedStepNativeAndBindings(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Glyph of Doom"`) {
			record = line
		}
	}
	files := fstest.MapFS{"LEG.ron": &fstest.MapFile{Data: []byte(record)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	first, err := registry.CreateCard("Glyph of Doom")
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.CreateCard("Glyph of Doom")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID() == second.ID() {
		t.Fatal("factories share identity")
	}
	g, a, b := ronGame()
	walls := make([]*mage.Permanent, 2)
	attackers := make([]*mage.Permanent, 2)
	for i := range 2 {
		walls[i] = g.PutOnBattlefield(repositoryCard(t, "LEB", "Wall of Wood"), b.PlayerID())
		attackers[i] = g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), a.PlayerID())
		attackers[i].Card.SetOwner(a.PlayerID())
		g.GetCombat().AddAttacker(attackers[i].ID(), b.PlayerID())
		g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: walls[i].ID(), AttackerID: attackers[i].ID()}})
	}
	for i, spell := range []mage.Card{first, second} {
		spell.SetOwner(a.PlayerID())
		a.AddToHand(spell)
		a.ManaPool().Add(core.Black, 1)
		if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{walls[i].ID()}, 0); err != nil {
			t.Fatal(err)
		}
	}
	g.ResolveStack()
	if len(g.DelayedTriggers()) != 2 {
		t.Fatal("queued casts lost separate schedules")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndOfCombat, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("schedules did not both trigger")
	}
	g.ResolveTopOfStack()
	count := 0
	for _, p := range attackers {
		if g.FindPermanent(p.ID()) == nil {
			count++
		}
	}
	if count != 1 {
		t.Fatal("one schedule used both target groups")
	}
	g.ResolveStack()
	for _, p := range attackers {
		if g.FindPermanent(p.ID()) != nil {
			t.Fatal("independent schedule lost target")
		}
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[UnknownRule])`)}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("invalid collection replaced registry")
	}
	report, err := InspectFS(fstest.MapFS{"LEG.ron": files["LEG.ron"]}, "*.ron")
	if err != nil || report.Compiled != 1 || len(report.Issues) != 0 {
		t.Fatal("semantic report rejected bounded schedule", err)
	}
	found := false
	for _, entry := range report.Signatures {
		if entry.Name == "AtNextEndOfCombatThisTurn" && entry.Role == "delayed_step_event" {
			found = true
		}
	}
	if !found {
		t.Fatal("bounded event inventory role missing")
	}
}

func TestRONBoundedDelayedStepLiveHistory(t *testing.T) {
	g, a, b := ronGame()
	wall := g.PutOnBattlefield(mage.NewCreature("Wall", "", 0, 5, mage.WithSubTypes("Wall"), mage.WithKeyword(core.CanBlockAny)), b.PlayerID())
	spell := repositoryCard(t, "LEG", "Glyph of Doom")
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(core.Black, 1)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{wall.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.FireEvent(core.GameEvent{Type: core.EvtEndOfCombat, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	victims := make([]*mage.Permanent, 3)
	assignments := make([]mage.BlockAssignment, 3)
	for i := range 3 {
		card := mage.NewCreature("Victim", "", 1, 5)
		if i == 2 {
			card = mage.NewCreature("Protected", "", 1, 5, mage.WithKeyword(core.Indestructible))
		}
		victims[i] = g.PutOnBattlefield(card, a.PlayerID())
		victims[i].Card.SetOwner(a.PlayerID())
		g.GetCombat().AddAttacker(victims[i].ID(), b.PlayerID())
		assignments[i] = mage.BlockAssignment{BlockerID: wall.ID(), AttackerID: victims[i].ID()}
	}
	g.ExecuteBlockers(assignments)
	g.ResolveStack()
	if g.FindPermanent(victims[0].ID()) != nil || g.FindPermanent(victims[1].ID()) != nil || g.FindPermanent(victims[2].ID()) == nil {
		t.Fatalf("history outcome: %v %v %v", g.FindPermanent(victims[0].ID()) != nil, g.FindPermanent(victims[1].ID()) != nil, g.FindPermanent(victims[2].ID()) != nil)
	}
}
