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

func TestRONDelayedDeathCaptureReincarnation(t *testing.T) {
	for _, change := range []string{"normal", "copy", "clone", "phase", "phase-cycle", "return", "shroud", "exile-replacement", "cleanup"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := mage.NewCreature("Watched", "", 2, 3)
			card.SetOwner(b.PlayerID())
			watched := g.PutOnBattlefield(card, a.PlayerID())
			spell := repositoryCard(t, "LEG", "Reincarnation")
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(core.Green, 2)
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{watched.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "phase":
				g.PhaseOut(watched.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(watched.ID()))
			case "return":
				g.BouncePermanentToHand(g.FindPermanent(watched.ID()))
				b.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
			case "shroud":
				g.MutablePermanent(watched.ID()).GrantBaseAttr(core.Shroud)
			case "copy":
				if g.CopySpellOnStack(spell.ID(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			g.ResolveStack()
			if change == "phase" || change == "return" || change == "shroud" {
				if len(g.DelayedTriggers()) != 0 {
					t.Fatal("illegal target installed delay")
				}
				return
			}
			if change == "cleanup" {
				g.RunStepWithPriority(core.Cleanup)
			}
			if change == "exile-replacement" {
				g.AddExileIfWouldGoToGraveyardThisTurn(watched.ID(), spell.ID())
			}
			clone := g.Clone()
			g.DestroyPermanent(g.FindPermanent(watched.ID()))
			if g.StackSize() != 0 {
				t.Fatal("death body ran before trigger placement")
			}
			g.PutTriggersOnStack()
			if change == "exile-replacement" || change == "cleanup" {
				if g.StackSize() != 0 {
					t.Fatal("expired or replaced death triggered")
				}
				return
			}
			g.RunStepWithPriority(core.Cleanup)
			g.ResolveStack()
			returned := g.FindPermanent(card.ID())
			if returned == nil || returned.ControllerID() != b.PlayerID() || returned.Card.Owner() != b.PlayerID() {
				t.Fatal("dead creature owner did not receive return")
			}
			if change == "clone" {
				if clone.FindPermanent(card.ID()) == nil || clone.FindPermanent(card.ID()).ControllerID() != a.PlayerID() {
					t.Fatal("original changed clone")
				}
				clone.DestroyPermanent(clone.FindPermanent(card.ID()))
				clone.PutTriggersOnStack()
				clone.ResolveStack()
				if p := clone.FindPermanent(card.ID()); p == nil || p.ControllerID() != b.PlayerID() {
					t.Fatal("clone lost death owner")
				}
			}
			g.DestroyPermanent(returned)
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("subscription followed returned creature")
			}
		})
	}
}

func deathCaptureAction(event, body string) string {
	return `CreateTriggerUntil(` + event + `(SinglePermanent(Ref_TargetPermanent)),ActionList([` + body + `]),UntilEndOfTurn)`
}

func TestRONDelayedDeathCaptureContextsAndInvalid(t *testing.T) {
	body := `PlayerAction(OwnerOfPermanent(Trigger_ThatPermanent),GainLife(Integer(3))),PlayerAction(ControllerOfPermanent(Trigger_ThatPermanent),GainLife(Integer(5)))`
	action := deathCaptureAction("WhenAPermanentDies", body)
	for _, context := range []string{"spell", "activation", "nested"} {
		t.Run(context, func(t *testing.T) {
			rule := `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([` + action + `])))`
			types := "Instant"
			if context == "activation" {
				types = "Artifact"
				rule = `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([` + action + `])))`
			}
			if context == "nested" {
				rule = `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([` + action + `]))])))`
			}
			factory := permanentScopeFactory(t, types, rule)
			g, a, b := ronGame()
			watchedCard := mage.NewCreature("Watched", "", 2, 3)
			watchedCard.SetOwner(b.PlayerID())
			watched := g.PutOnBattlefield(watchedCard, a.PlayerID())
			if context == "activation" {
				source := g.PutOnBattlefield(factory(), a.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{watched.ID()}); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				g.ExilePermanent(source)
			} else {
				spell := factory()
				spell.SetOwner(a.PlayerID())
				a.AddToHand(spell)
				if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{watched.ID()}, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				if context == "nested" {
					g.RunStepWithPriority(core.EndStep)
					g.ResolveStack()
				}
			}
			g.PutPermanentIntoGraveyard(g.FindPermanent(watched.ID()))
			g.PutTriggersOnStack()
			g.ResolveStack()
			if a.Life() != 25 || b.Life() != 23 {
				t.Fatalf("owner/controller facts changed: %d/%d", a.Life(), b.Life())
			}
		})
	}
	for _, invalid := range []string{
		body,
		deathCaptureAction("WhenAPermanentLeavesTheBattlefield", body),
		strings.Replace(action, "OwnerOfPermanent(Trigger_ThatPermanent)", "OwnerOfPermanent()", 1),
		strings.Replace(action, "OwnerOfPermanent(Trigger_ThatPermanent)", "OwnerOfPermanent(ThisPermanent)", 1),
		strings.Replace(action, "Trigger_ThatPermanent)", "Trigger_ThatPermanent())", 1),
		action + `,PlayerAction(OwnerOfPermanent(Trigger_ThatPermanent),GainLife(Integer(1)))`,
		strings.Replace(action, "UntilEndOfTurn", "UntilEndOfNextTurn", 1),
		strings.Replace(action, "SinglePermanent(Ref_TargetPermanent)", "EachPermanent(IsCardtype(Creature))", 1),
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([`+invalid+`])))])`), "invalid-death.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted invalid death context", invalid)
		}
	}
}

func TestRONDelayedDeathCaptureNativeAndInventory(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	record := hostUntapRecord(t, "LEG", "Reincarnation")
	line := strings.Split(string(data), "\n")[record.Position.Line-1]
	files := fstest.MapFS{"card.ron": {Data: []byte(line)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	first, err := registry.CreateCard("Reincarnation")
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.CreateCard("Reincarnation")
	if err != nil {
		t.Fatal(err)
	}
	if first.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == second.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
		t.Fatal("shared target declarations")
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[UnknownRule])`)}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("atomic collection rejection failed")
	}
	entries := map[string]*InventorySignature{}
	walkInventory(record, "Reincarnation", "Card", "card", "collection", false, entries)
	found := 0
	for _, entry := range entries {
		if entry.Name == "OwnerOfPermanent" && entry.Role == "player_reference" {
			found += entry.Occurrences
		}
	}
	if found != 2 {
		t.Fatal("death owner syntax lost in semantic inventory")
	}
}

type deathReturnChooser struct {
	*mage.BasePlayer
	pick    mage.Card
	choices int
}

func (p *deathReturnChooser) ChooseCardFromGraveyard(_ []mage.Card, _ string, _ mage.GameReader) mage.Card {
	p.choices++
	return p.pick
}

func TestRONDelayedDeathCaptureChoiceAtResolution(t *testing.T) {
	a := &deathReturnChooser{BasePlayer: mage.NewBasePlayer("A")}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	watchedCard := mage.NewCreature("Watched", "", 2, 3)
	watchedCard.SetOwner(b.PlayerID())
	watched := g.PutOnBattlefield(watchedCard, a.PlayerID())
	spell := repositoryCard(t, "LEG", "Reincarnation")
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(core.Green, 2)
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{watched.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.DestroyPermanent(g.FindPermanent(watched.ID()))
	g.PutTriggersOnStack()
	if a.choices != 0 {
		t.Fatal("graveyard choice occurred before resolution")
	}
	candidate := mage.NewCreature("Later candidate", "", 4, 4)
	candidate.SetOwner(b.PlayerID())
	b.AddToGraveyard(candidate)
	a.pick = candidate
	g.ResolveStack()
	if a.choices != 1 {
		t.Fatal("wrong effect chooser or missing choice")
	}
	returned := g.FindPermanent(candidate.ID())
	if returned == nil || returned.ControllerID() != b.PlayerID() || g.FindPermanent(watchedCard.ID()) != nil {
		t.Fatal("wrong graveyard member or entry controller")
	}
}

func TestRONDelayedDeathCaptureIndependentDeaths(t *testing.T) {
	g, a, b := ronGame()
	for _, owner := range []*mage.BasePlayer{a, b} {
		card := mage.NewCreature("Watched", "", 2, 3)
		card.SetOwner(owner.PlayerID())
		watched := g.PutOnBattlefield(card, a.PlayerID())
		spell := repositoryCard(t, "LEG", "Reincarnation")
		spell.SetOwner(a.PlayerID())
		a.AddToHand(spell)
		a.ManaPool().Add(core.Green, 2)
		a.ManaPool().Add(core.Colorless, 1)
		if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{watched.ID()}, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
	}
	subjects := g.FilterBattlefield(mage.IsCreature)
	g.DestroyPermanents(subjects)
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("simultaneous deaths lost independent triggers")
	}
	clone := g.Clone()
	for _, game := range []*mage.Game{g, clone} {
		game.ResolveStack()
		for _, former := range subjects {
			returned := game.FindPermanent(former.ID())
			if returned == nil || returned.ControllerID() != former.Card.Owner() {
				t.Fatal("death owner overwritten by another trigger")
			}
		}
	}
}

func TestRONDelayedDeathCaptureSiblingDeclarations(t *testing.T) {
	body := `PlayerAction(OwnerOfPermanent(Trigger_ThatPermanent),GainLife(Integer(3)))`
	action := deathCaptureAction("WhenAPermanentDies", body)
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([`+action+`,`+action+`])))`)
	g, a, b := ronGame()
	card := mage.NewCreature("Watched", "", 2, 3)
	card.SetOwner(b.PlayerID())
	watched := g.PutOnBattlefield(card, a.PlayerID())
	spell := factory()
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{watched.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if len(g.DelayedTriggers()) != 2 {
		t.Fatal("sibling declarations lost a watch")
	}
	g.DestroyPermanent(g.FindPermanent(watched.ID()))
	g.PutTriggersOnStack()
	g.ResolveStack()
	if b.Life() != 26 {
		t.Fatal("sibling death facts collided")
	}
}
