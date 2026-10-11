package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONSavedActivePlayerScopes(t *testing.T) {
	const body = `ActionList([PlayerActions(TheActivePlayer,[CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([GainLife(Integer(2)),PlayerActions(You,[GainLife(Integer(3))]),GainLife(Integer(5)),TapEachPermanent(ControlledByPlayer(ActionPlayer)),UntapEachPermanent(ControlledByPlayer(TheActivePlayer))]))]),GainLife(Integer(11))])`
	factory, err := Compile(activePlayerControlRecord(t, body))
	if err != nil {
		t.Fatal(err)
	}
	for _, copied := range []bool{false, true} {
		t.Run(map[bool]string{false: "original", true: "copy"}[copied], func(t *testing.T) {
			g, a, b := ronGame()
			own := g.PutOnBattlefield(mage.NewCreature("Own", "", 2, 3), a.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 3), b.PlayerID())
			own.Tapped = true
			g.SetActivePlayerIndex(1)
			c := factory()
			c.SetOwner(a.PlayerID())
			a.AddToHand(c)
			a.ManaPool().Add(core.Blue, 1)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if copied && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			g.ResolveStack()
			if a.Life() != 31 || b.Life() != map[bool]int{false: 20, true: 31}[copied] {
				t.Fatal("scope escaped into sibling")
			}
			g.SetActivePlayerIndex(0)
			g.SetTurn(2)
			clone := g.Clone()
			g.RunStepWithPriority(core.EndStep)
			if clone.FindPermanent(other.ID()).Tapped || len(clone.DelayedTriggers()) != map[bool]int{false: 1, true: 2}[copied] {
				t.Fatal("live execution changed clone")
			}
			clone.RunStepWithPriority(core.EndStep)
			for _, game := range []*mage.Game{g, clone} {
				wantB := map[bool]int{false: 27, true: 48}[copied]
				if game.GetPlayer(a.PlayerID()).Life() != 34 || game.GetPlayer(b.PlayerID()).Life() != wantB || game.FindPermanent(own.ID()).Tapped || !game.FindPermanent(other.ID()).Tapped || len(game.DelayedTriggers()) != 0 {
					t.Fatalf("saved and live player scopes: life %d/%d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
				}
			}
		})
	}
}

func TestRONSavedActivePlayerInvalid(t *testing.T) {
	for _, body := range []string{
		`ActionList([PlayerActions(TheActivePlayer(),[])])`,
		`ActionList([PlayerActions(ActionPlayer,[GainLife(Integer(1))])])`,
		`ActionList([PlayerActions(TheActivePlayer,[]),TapEachPermanent(ControlledByPlayer(ActionPlayer))])`,
		`ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([TapEachPermanent(ControlledByPlayer(ActionPlayer))]))])`,
		`ActionList([PlayerActions(TheActivePlayer,[CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([]))])])`,
		`ActionList([PlayerActions(TheActivePlayer,[CreateFutureTrigger(AtTheBeginningOfTheNextEndStep(),ActionList([GainLife(Integer(1))]))])])`,
		`ActionList([PlayerActions(TheActivePlayer,[CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([TapEachPermanent(ControlledByPlayer(ActionPlayer()))]))])])`,
	} {
		if _, err := Compile(activePlayerControlRecord(t, body)); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRONSavedActivePlayerResolution(t *testing.T) {
	factory, err := Compile(activePlayerControlRecord(t, `ActionList([PlayerActions(TheActivePlayer,[CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([GainLife(Integer(2))]))])])`))
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	for _, active := range []int{1, 0} {
		g.SetActivePlayerIndex(1 - active)
		c := factory()
		c.SetOwner(a.PlayerID())
		a.AddToHand(c)
		a.ManaPool().Add(core.Blue, 1)
		if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		g.SetActivePlayerIndex(active)
		g.ResolveStack()
	}
	if len(g.DelayedTriggers()) != 2 {
		t.Fatal("independent factories lost a delay")
	}
	clone := g.Clone()
	g.RunStepWithPriority(core.EndStep)
	if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 20 {
		t.Fatal("resolution changed cloned frame")
	}
	clone.SetActivePlayerIndex(1)
	clone.RunStepWithPriority(core.EndStep)
	for _, game := range []*mage.Game{g, clone} {
		if game.GetPlayer(a.PlayerID()).Life() != 22 || game.GetPlayer(b.PlayerID()).Life() != 22 || len(game.DelayedTriggers()) != 0 {
			t.Fatal("snapshot used announcement player or another instance")
		}
	}
}

func TestRONSavedActivePlayerSiren(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, change := range []string{"normal", "copy", "phase", "cycle", "return", "regenerate", "control", "payment"} {
			t.Run(set+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				oldCard := mage.NewCreature("Old", "", 2, 3)
				oldCard.SetOwner(b.PlayerID())
				old := g.PutOnBattlefield(oldCard, b.PlayerID())
				own := g.PutOnBattlefield(mage.NewCreature("Own", "", 2, 3), a.PlayerID())
				wall := g.PutOnBattlefield(mage.NewCreature("Wall", "", 0, 3, mage.WithSubTypes("Wall")), b.PlayerID())
				g.SetActivePlayerIndex(1)
				g.SetTurn(2)
				c := repositoryCard(t, set, "Siren's Call")
				c.SetOwner(a.PlayerID())
				a.AddToHand(c)
				if change != "payment" {
					a.ManaPool().Add(core.Blue, 1)
				}
				err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0)
				if change == "payment" {
					if err == nil || g.StackSize() != 0 || len(g.DelayedTriggers()) != 0 || len(a.Hand()) != 1 {
						t.Fatal("failed payment installed a delay")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				g.ResolveStack()
				if len(a.Graveyard()) != 1 {
					t.Fatal("source spell did not depart")
				}
				g.RunStepWithPriority(core.Cleanup)
				g.SetActivePlayerIndex(0)
				g.SetTurn(3)
				fresh := g.PutOnBattlefield(mage.NewCreature("Fresh", "", 2, 3), b.PlayerID())
				dies := true
				switch change {
				case "phase":
					g.PhaseOut(old.ID())
					dies = false
				case "cycle":
					g.PhaseIn(g.PhaseOut(old.ID()))
				case "return":
					g.RemoveFromBattlefield(old)
					old = g.PutOnBattlefield(oldCard, b.PlayerID())
					dies = false
				case "regenerate":
					g.AddRegenerationShield(old.ID())
					dies = false
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: old.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
					dies = false
				}
				clone := g.Clone()
				for _, game := range []*mage.Game{g, clone} {
					game.RunStepWithPriority(core.EndStep)
					if (game.FindPermanentIncludingPhased(old.ID()) == nil) != dies || game.FindPermanent(own.ID()) == nil || game.FindPermanent(wall.ID()) == nil || game.FindPermanent(fresh.ID()) == nil || len(game.DelayedTriggers()) != 0 {
						t.Fatal("delay lost saved player or live collection history")
					}
				}
			})
		}
	}
}

func TestRONSavedActivePlayerNative(t *testing.T) {
	const valid = `Card(Name:"Saved",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([PlayerActions(TheActivePlayer,[CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([TapEachPermanent(ControlledByPlayer(ActionPlayer))]))])]))])`
	registry := mage.NewRegistry()
	if err := LoadInto(registry, fstest.MapFS{"saved.ron": {Data: []byte(valid)}}, "*.ron"); err != nil {
		t.Fatal(err)
	}
	invalid := strings.Replace(strings.Replace(valid, "Saved", "Invalid", 1), "ActionPlayer", "ActionPlayer()", 1)
	if err := LoadInto(registry, fstest.MapFS{"saved.ron": {Data: []byte(valid + invalid)}}, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("invalid collection replaced registry")
	}
}
