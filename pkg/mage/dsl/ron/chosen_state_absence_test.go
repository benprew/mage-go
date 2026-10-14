package ron

import (
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONChosenStateAbsenceJihad(t *testing.T) {
	factory, err := Compile(repositoryRecord(t, "ARN", "Jihad", "ARN.lines.ron"))
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	chooser := &ronColorEntryPlayer{Player: a, answer: core.Red}
	g.SetPlayerAt(0, chooser)
	member := g.PutOnBattlefield(mage.NewArtifact("Red member", "{R}"), b.PlayerID())
	own := g.PutOnBattlefield(mage.NewCreature("White A", "{W}", 1, 4), a.PlayerID())
	other := g.PutOnBattlefield(mage.NewCreature("White B", "{W}", 1, 4), b.PlayerID())
	card := factory()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.White, 3)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if chooser.calls != 0 {
		t.Fatal("announcement made entry choice")
	}
	g.ResolveStack()
	source := g.FindPermanent(card.ID())
	if source == nil || source.ChosenPlayer != b.PlayerID() || source.ChosenColor != core.Red || chooser.calls != 1 {
		t.Fatal("entry choices missing")
	}
	for _, p := range []*mage.Permanent{own, other} {
		if p.CurrentPower(g) != 3 || p.CurrentToughness(g) != 5 {
			t.Fatal("global white bonus missing")
		}
	}
	g.CheckStateTriggers()
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("present member triggered absence")
	}
	token := g.PutOnBattlefield(mage.NewCreature("Red token", "{R}", 1, 4), b.PlayerID())
	token.IsToken = true
	g.RemoveFromBattlefield(member)
	g.CheckStateTriggers()
	g.CheckStateTriggers()
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || own.CurrentPower(g) != 1 {
		t.Fatal("nontoken absence or pending suppression failed")
	}
	g.PutOnBattlefield(mage.NewArtifact("Restored member", "{R}"), b.PlayerID())
	if own.CurrentPower(g) != 3 {
		t.Fatal("static bonus did not return")
	}
	g.ResolveStack()
	if g.FindPermanent(source.ID()) != nil || !slices.ContainsFunc(a.Graveyard(), func(c mage.Card) bool { return c.ID() == card.ID() }) || own.CurrentPower(g) != 1 {
		t.Fatal("queued sacrifice rechecked condition or lost owner")
	}
}

func TestRONChosenStateAbsenceLifecycle(t *testing.T) {
	for _, state := range []string{"counter", "member-control", "member-color", "member-phase", "source-phase-before", "removed", "source-control-before", "source-control-after", "source-phase-after", "returned", "left", "clone", "copy", "two", "pending-phase-cycle", "pending-control", "pending-return"} {
		t.Run(state, func(t *testing.T) {
			factory, err := Compile(repositoryRecord(t, "ARN", "Jihad", "ARN.lines.ron"))
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			chooser := &ronColorEntryPlayer{Player: a, answer: core.Red}
			g.SetPlayerAt(0, chooser)
			member := g.PutOnBattlefield(mage.NewArtifact("Member", "{R}"), b.PlayerID())
			card := factory()
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			switch state {
			case "member-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: member.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "member-color":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetColors(mage.This, core.Blue)), member.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "member-phase":
				g.PhaseOut(member.ID())
			case "source-phase-before":
				g.PhaseOut(source.ID())
				g.RemoveFromBattlefield(member)
			case "removed":
				removal := mage.RemoveAllAbilities(source.ID(), core.WhileOnBattlefield)
				removal.SetSourceID(member.ID())
				g.AddContinuousEffect(removal)
				g.PhaseOut(member.ID())
			case "source-control-before":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: member.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				g.PhaseOut(member.ID())
			default:
				g.RemoveFromBattlefield(member)
			}
			g.ApplyContinuousEffects()
			g.CheckStateTriggers()
			g.CheckStateTriggers()
			if state == "source-phase-before" || state == "removed" {
				g.PutTriggersOnStack()
				if g.StackSize() != 0 {
					t.Fatal("inactive source detected state")
				}
				if state == "source-phase-before" {
					g.PhaseIn([]uuid.UUID{source.ID()})
				} else {
					g.RemoveFromBattlefield(g.FindPermanentIncludingPhased(member.ID()))
				}
				g.ApplyContinuousEffects()
				g.CheckStateTriggers()
			}
			if state == "pending-phase-cycle" {
				g.PhaseIn(g.PhaseOut(source.ID()))
				g.CheckStateTriggers()
			}
			if state == "pending-control" {
				g.PutOnBattlefield(mage.NewArtifact("Restored member", "{R}"), b.PlayerID())
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			if state == "pending-return" {
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(mage.NewArtifact("Blue member", "{U}"), b.PlayerID())
				chooser.answer = core.Blue
				source = g.PutOnBattlefield(card, a.PlayerID())
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatalf("trigger count %d", g.StackSize())
			}
			if state == "source-control-before" && g.StackPeek().Controller != b.PlayerID() {
				t.Fatal("trigger controller not captured")
			}
			if state == "pending-control" && g.StackPeek().Controller != a.PlayerID() {
				t.Fatal("pending trigger followed new controller")
			}
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("active trigger repeated")
			}
			switch state {
			case "counter":
				g.CounterStackObject(g.StackPeek().ID)
				g.CheckStateTriggers()
				g.PutTriggersOnStack()
				if g.StackSize() != 1 {
					t.Fatal("counter did not rearm")
				}
			case "source-control-after":
				g.PutOnBattlefield(mage.NewArtifact("Restored member", "{R}"), b.PlayerID())
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "source-phase-after":
				g.PhaseOut(source.ID())
				g.ResolveStack()
				if g.FindPermanentIncludingPhased(source.ID()) == nil {
					t.Fatal("phased source sacrificed")
				}
				return
			case "returned":
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(mage.NewArtifact("Blue member", "{U}"), b.PlayerID())
				chooser.answer = core.Blue
				returned := g.PutOnBattlefield(card, a.PlayerID())
				g.ResolveStack()
				if g.FindPermanent(returned.ID()) == nil || returned.ChosenColor != core.Blue {
					t.Fatal("old trigger followed returned source")
				}
				return
			case "left":
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				g.ResolveStack()
				if len(a.Hand()) != 1 {
					t.Fatal("departed source moved again")
				}
				return
			case "clone":
				clone := g.Clone()
				clone.MutablePermanent(source.ID()).ChosenColor = core.Green
				clone.ResolveStack()
				if clone.FindPermanent(source.ID()) != nil || g.FindPermanent(source.ID()) == nil {
					t.Fatal("clone shared resolution")
				}
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				g.ResolveTopOfStack()
				if g.FindPermanent(source.ID()) == nil {
					t.Fatal("copy sacrificed another player's source")
				}
			case "two":
				chooser.answer = core.Green
				green := g.PutOnBattlefield(mage.NewArtifact("Green member", "{G}"), b.PlayerID())
				second := g.PutOnBattlefield(factory(), a.PlayerID())
				g.CheckStateTriggers()
				g.PutTriggersOnStack()
				if g.StackSize() != 1 || second.ChosenColor != core.Green || source.ChosenColor != core.Red {
					t.Fatal("source entry choices shared storage")
				}
				g.RemoveFromBattlefield(green)
				g.CheckStateTriggers()
				g.PutTriggersOnStack()
				if g.StackSize() != 2 {
					t.Fatal("factory declarations collided")
				}
				g.ResolveTopOfStack()
				if g.FindPermanent(second.ID()) != nil || g.FindPermanent(source.ID()) == nil {
					t.Fatal("independent source identity lost")
				}
			}
			g.ResolveStack()
			if state == "source-control-after" || state == "pending-control" || state == "pending-return" {
				if g.FindPermanent(source.ID()) == nil {
					t.Fatal("old trigger sacrificed a stolen or returned source")
				}
			} else if g.FindPermanent(source.ID()) != nil {
				t.Fatal("source not sacrificed")
			}
		})
	}
}

func TestRONChosenStateAbsenceSpellCopy(t *testing.T) {
	factory, err := Compile(repositoryRecord(t, "ARN", "Jihad", "ARN.lines.ron"))
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	first := &ronColorEntryPlayer{Player: a, answer: core.Red}
	second := &ronColorEntryPlayer{Player: b, answer: core.Green}
	g.SetPlayerAt(0, first)
	g.SetPlayerAt(1, second)
	g.PutOnBattlefield(mage.NewArtifact("Red member", "{R}"), b.PlayerID())
	green := g.PutOnBattlefield(mage.NewArtifact("Green member", "{G}"), a.PlayerID())
	card := factory()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.White, 3)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("spell copy failed")
	}
	g.ResolveStack()
	var copied *mage.Permanent
	for _, p := range g.AllBattlefield() {
		if p.Name() == "Jihad" && p.IsToken {
			copied = p
		}
	}
	if copied == nil || copied.ChosenColor != core.Green || copied.ChosenPlayer != a.PlayerID() || first.calls != 1 || second.calls != 1 {
		t.Fatal("spell copy lost fresh entry choices")
	}
	g.RemoveFromBattlefield(green)
	g.CheckStateTriggers()
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || g.StackPeek().Controller != b.PlayerID() {
		t.Fatal("spell copy state binding lost its player")
	}
	g.ResolveStack()
	if g.FindPermanent(copied.ID()) != nil || g.FindPermanent(card.ID()) == nil {
		t.Fatal("spell copy state rule used the original choices")
	}
}

func TestRONChosenStateAbsenceFilters(t *testing.T) {
	for _, filter := range []string{`IsColor(Red)`, `Or([IsColor(Red),IsColor(Green)])`, `And([IsNonToken,And([IsCardtype(Artifact),IsColor(TheChosenColor)])])`} {
		t.Run(filter, func(t *testing.T) {
			rules := combinedEntryRule + `,TriggerA(WhenAPlayerControlsNoPermanents(SinglePlayer(TheChosenPlayer),` + filter + `),ActionList([SacrificePermanent(ThisPermanent)]))`
			registry, err := LoadFS(fstest.MapFS{"choice.ron": {Data: []byte(colorEntryFixture(rules))}}, "*.ron")
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			g.SetPlayerAt(0, &ronColorEntryPlayer{Player: a, answer: core.Red})
			member := g.PutOnBattlefield(mage.NewArtifact("Member", "{R}"), b.PlayerID())
			card, err := registry.CreateCard("Color entry")
			if err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("matching member ignored")
			}
			g.RemoveFromBattlefield(member)
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			g.ResolveStack()
			if g.FindPermanent(source.ID()) != nil {
				t.Fatal("absence not detected")
			}
		})
	}
}

func TestRONChosenStateAbsenceInvalid(t *testing.T) {
	event := `TriggerA(WhenAPlayerControlsNoPermanents(SinglePlayer(TheChosenPlayer),And([IsNonToken,IsColor(TheChosenColor)])),ActionList([SacrificePermanent(ThisPermanent)]))`
	for _, rule := range []string{event, `AsPermanentEnters(ThisPermanent,[ChooseAPlayer(Opponent)]),` + event, strings.Replace(combinedEntryRule+","+event, "SinglePlayer(TheChosenPlayer)", "SinglePlayer(TheChosenPlayer())", 1), strings.Replace(combinedEntryRule+","+event, "SinglePlayer(TheChosenPlayer)", "SinglePlayer(ActionPlayer)", 1), strings.Replace(combinedEntryRule+","+event, "IsColor(TheChosenColor)", "IsColor(TheChosenColor())", 1), strings.Replace(combinedEntryRule+","+event, "And([IsNonToken,IsColor(TheChosenColor)])", "Or([IsNonToken,IsColor(TheChosenColor)])", 1), strings.Replace(combinedEntryRule+","+event, "And([IsNonToken,IsColor(TheChosenColor)])", "And([])", 1), strings.Replace(combinedEntryRule+","+event, "SacrificePermanent(ThisPermanent)", "PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))", 1)} {
		nodes, err := Parse(strings.NewReader(colorEntryFixture(rule)), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Errorf("invalid declaration accepted: %s", rule)
		}
	}
	registry := mage.NewRegistry()
	registry.Register("Retained", func() mage.Card { return mage.NewArtifact("Retained", "") })
	if err := LoadInto(registry, fstest.MapFS{"invalid.ron": {Data: []byte(colorEntryFixture(event))}}, "*.ron"); err == nil {
		t.Fatal("invalid load accepted")
	}
	if _, err := registry.CreateCard("Retained"); err != nil {
		t.Fatal("rejected load changed registry")
	}
}

func TestRONChosenStateAbsenceExistingCards(t *testing.T) {
	for _, tc := range []struct{ set, name string }{
		{"LEB", "Pirate Ship"}, {"LEB", "Sea Serpent"}, {"ARN", "Dandân"},
		{"ARN", "Island Fish Jasconius"}, {"ARN", "Merchant Ship"},
		{"ARN", "Serendib Djinn"}, {"DRK", "Giant Shark"},
		{"FEM", "Seasinger"}, {"FEM", "Vodalian Knights"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			record := repositoryRecord(t, tc.set, tc.name, tc.set+".lines.ron")
			factory, err := Compile(record)
			blocker := map[string]string{
				"Giant Shark": "expected 1 positional arguments (WasDealtDamageThisTurn)",
			}[tc.name]
			if blocker != "" {
				if err == nil || !strings.Contains(err.Error(), blocker) {
					t.Fatalf("complete record blocker: %v", err)
				}
				var stateRules []Node
				for _, rule := range record.Fields["Rules"].Args {
					if rule.Name == "TriggerA" && rule.Args[0].Name == "WhenAPlayerControlsNoPermanents" {
						stateRules = append(stateRules, rule)
					}
				}
				if len(stateRules) != 1 {
					t.Fatal("expected one exact absence subrule")
				}
				record.Fields["Rules"] = Node{Kind: List, Args: stateRules}
				factory, err = Compile(record)
			}
			if err != nil {
				t.Fatal(err)
			}
			g, a, _ := ronGame()
			land := g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), a.PlayerID())
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("controller land ignored")
			}
			phased := g.PhaseOut(land.ID())
			g.CheckStateTriggers()
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("controller absence or pending suppression changed")
			}
			g.PhaseIn(phased)
			g.ResolveStack()
			if g.FindPermanent(source.ID()) != nil {
				t.Fatal("queued sacrifice changed after land returned")
			}
		})
	}
}
