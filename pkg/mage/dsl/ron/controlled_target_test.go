package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONControlledTargetBattleGear(t *testing.T) {
	for _, change := range []string{"none", "opponent", "land", "control-before", "source-control", "return-before", "phase-before", "phase-cycle", "shroud", "untap-before", "source-return", "source-phase", "control-after", "untap-after", "phase-after", "return-after", "target-return-after", "target-phase-after", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "ATQ", "Ashnod's Battle Gear")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(b.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			targetCard := mage.NewCreature("Target", "", 3, 5)
			targetCard.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(targetCard, a.PlayerID())
			opponent := g.PutOnBattlefield(mage.NewCreature("Opponent", "", 3, 5), b.PlayerID())
			land := g.PutOnBattlefield(mage.NewLand("Land"), a.PlayerID())
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{target.ID()}); err == nil {
				t.Fatal("unpaid activation accepted")
			}
			if g.StackSize() != 0 || g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != 1 {
				t.Fatal("failed payment changed state")
			}
			a.ManaPool().Add(core.Colorless, 1)
			id := target.ID()
			if change == "opponent" {
				id = opponent.ID()
			}
			if change == "land" {
				id = land.ID()
			}
			err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{id})
			if change == "opponent" || change == "land" {
				if err == nil || g.StackSize() != 0 || g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != 2 {
					t.Fatal("illegal target changed state")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != 0 || target.CurrentPower(g) != 3 {
				t.Fatal("wrong cost or effect timing")
			}
			active := true
			switch change {
			case "control-before":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: opponent.ID(), TargetID: target.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				active = false
			case "source-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: opponent.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "return-before":
				g.RemoveFromBattlefield(target)
				target = g.PutOnBattlefield(targetCard, a.PlayerID())
				active = false
			case "phase-before":
				g.PhaseOut(target.ID())
				active = false
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(target.ID()))
			case "shroud":
				target.GrantBaseAttr(core.Shroud)
				active = false
			case "untap-before":
				g.UntapPermanent(g.FindPermanent(source.ID()))
				active = false
			case "source-return":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.TapPermanent(g.PutOnBattlefield(card, a.PlayerID()))
				active = false
			case "source-phase":
				g.PhaseOut(source.ID())
				active = false
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			check := func(game *mage.Game, on bool) {
				t.Helper()
				p := game.FindPermanentIncludingPhased(target.ID())
				power, toughness := 3, 5
				if on {
					power, toughness = 5, 3
					if change == "copy" {
						power, toughness = 7, 1
					}
				}
				if p.CurrentPower(game) != power || p.CurrentToughness(game) != toughness {
					t.Fatalf("P/T=%d/%d want=%d/%d", p.CurrentPower(game), p.CurrentToughness(game), power, toughness)
				}
			}
			g.ResolveStack()
			check(g, active)
			check(clone, false)
			clone.ResolveStack()
			check(clone, active)
			switch change {
			case "control-after":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: opponent.ID(), TargetID: target.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "untap-after":
				g.UntapPermanent(g.FindPermanent(source.ID()))
				g.TapPermanent(g.FindPermanent(source.ID()))
				active = false
			case "phase-after":
				g.PhaseIn(g.PhaseOut(source.ID()))
				active = false
			case "return-after":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.TapPermanent(g.PutOnBattlefield(card, a.PlayerID()))
				active = false
			case "target-return-after":
				g.RemoveFromBattlefield(g.FindPermanent(target.ID()))
				target = g.PutOnBattlefield(targetCard, a.PlayerID())
				active = false
			case "target-phase-after":
				g.PhaseIn(g.PhaseOut(target.ID()))
			}
			g.PhaseIn([]uuid.UUID{target.ID(), source.ID()})
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			check(g, active)
		})
	}
}

func TestRONControlledTargetUntapChoice(t *testing.T) {
	for _, accept := range []bool{false, true} {
		a := &ronStoragePlayer{ronCounterPayer: &ronCounterPayer{BasePlayer: mage.NewBasePlayer("A")}, untap: accept}
		b := mage.NewBasePlayer("B")
		g := mage.NewGame(a, b)
		source := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Ashnod's Battle Gear"), a.PlayerID())
		target := g.PutOnBattlefield(mage.NewCreature("Target", "", 3, 5), a.PlayerID())
		a.ManaPool().Add(core.Colorless, 2)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{target.ID()}); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		g.RunStepWithPriority(core.Untap)
		want := 5
		if accept {
			want = 3
		}
		if a.untapCalls != 1 || g.FindPermanent(source.ID()).Tapped == accept || g.FindPermanent(target.ID()).CurrentPower(g) != want {
			t.Fatal("wrong untap choice or bonus duration")
		}
	}
}

func TestRONControlledTargetDeclarationsAndInstances(t *testing.T) {
	const restriction = `ControlledByAPlayer(SinglePlayer(You))`
	for _, filter := range []string{restriction, `And([IsCardtype(Creature),` + restriction + `])`, `And([Other(ThisPermanent),And([AnyPermanent,` + restriction + `])])`} {
		factory := permanentScopeFactory(t, "Artifact", `Activated(TapPermanent(ThisPermanent),Targeted([TargetPermanent(`+filter+`)],ActionList([TapPermanent(Ref_TargetPermanent)])))`)
		first, second := factory(), factory()
		if err := mage.ValidateCardDefinition(first.Name(), first); err != nil {
			t.Fatal(err)
		}
		if first.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == second.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
			t.Fatal("instances shared target declarations")
		}
		g, a, b := ronGame()
		source := g.PutOnBattlefield(first, a.PlayerID())
		source.RevokeBaseAttr(core.AttrSummonSick)
		secondSource := g.PutOnBattlefield(second, a.PlayerID())
		secondSource.RevokeBaseAttr(core.AttrSummonSick)
		own := g.PutOnBattlefield(mage.NewCreature("Own", "", 2, 3), a.PlayerID())
		other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 3), b.PlayerID())
		ownSecond := g.PutOnBattlefield(mage.NewCreature("Own second", "", 2, 3), a.PlayerID())
		targets := first.Abilities()[0].(*mage.ActionDefinition).Targets()[0].Possible(a.PlayerID(), first, g)
		found := false
		for _, id := range targets {
			if id == other.ID() {
				t.Fatal("opponent was legal")
			}
			if id == own.ID() {
				found = true
			}
			if strings.Contains(filter, "Other(") && id == source.ID() {
				t.Fatal("source exclusion lost")
			}
		}
		if !found {
			t.Fatal("controller target missing")
		}
		for _, pair := range []struct{ source, target uuid.UUID }{{source.ID(), own.ID()}, {secondSource.ID(), ownSecond.ID()}} {
			if err := g.ActivateAbilityByIndex(a.PlayerID(), pair.source, 0, []uuid.UUID{pair.target}); err != nil {
				t.Fatal(err)
			}
		}
		g.ResolveTopOfStack()
		if !g.FindPermanent(ownSecond.ID()).Tapped || g.FindPermanent(own.ID()).Tapped {
			t.Fatal("target bindings shared chosen state")
		}
		g.ResolveTopOfStack()
		if !g.FindPermanent(own.ID()).Tapped {
			t.Fatal("first instance lost target")
		}
	}
	for _, filter := range []string{`ControlledByAPlayer`, `ControlledByAPlayer()`, `ControlledByAPlayer(SinglePlayer(You),Opponent)`, `ControlledByAPlayer(You)`, `ControlledByAPlayer(SinglePlayer(You()))`, `ControlledByAPlayer(SinglePlayer(Ref_TargetPlayer))`, `ControlledByAPlayer(Opponent)`, `Or([IsCardtype(Creature),` + restriction + `])`} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(` + filter + `)],ActionList([TapPermanent(Ref_TargetPermanent)])))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", filter)
		}
	}
}

func TestRONControlledTargetSpellAndTriggerContexts(t *testing.T) {
	const actions = `Targeted([TargetPermanent(And([IsCardtype(Creature),ControlledByAPlayer(SinglePlayer(You))]))],ActionList([TapPermanent(Ref_TargetPermanent)]))`
	for _, context := range []string{"spell", "trigger"} {
		for _, change := range []string{"none", "source-control", "source-depart", "target-control", "opponent-copy"} {
			t.Run(context+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				own := g.PutOnBattlefield(mage.NewCreature("Own", "", 2, 3), a.PlayerID())
				other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 3), b.PlayerID())
				var card mage.Card
				if context == "spell" {
					card = permanentScopeFactory(t, "Instant", `SpellActions(`+actions+`)`)()
					card.SetOwner(a.PlayerID())
					a.AddToHand(card)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{other.ID()}, 0); err == nil {
						t.Fatal("opponent spell target accepted")
					}
					if len(a.Hand()) != 1 || g.StackSize() != 0 {
						t.Fatal("illegal spell changed state")
					}
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{own.ID()}, 0); err != nil {
						t.Fatal(err)
					}
				} else {
					card = permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),`+actions+`)`)()
					g.PutOnBattlefield(card, a.PlayerID())
					g.PutTriggersOnStack()
					if g.StackSize() != 1 || g.StackPeek().Targets[0] != own.ID() {
						t.Fatal("trigger did not choose controller's creature")
					}
				}
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				switch change {
				case "source-control":
					if context == "trigger" {
						g.AddControlEffect(mage.ControlEffectSpec{SourceID: other.ID(), TargetID: card.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					}
				case "source-depart":
					if context == "trigger" {
						g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
					}
				case "target-control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: other.ID(), TargetID: own.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "opponent-copy":
					cp := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
					if cp == nil {
						t.Fatal("copy failed")
					}
					g.ResolveTopOfStack()
					if g.FindPermanent(own.ID()).Tapped {
						t.Fatal("copy used original controller for legality")
					}
				}
				clone := g.Clone()
				g.ResolveStack()
				if g.FindPermanent(own.ID()).Tapped != (change != "target-control") || g.FindPermanent(other.ID()).Tapped {
					t.Fatal("wrong resolving controller target")
				}
				if clone.FindPermanent(own.ID()).Tapped {
					t.Fatal("resolution changed pending clone")
				}
				clone.ResolveStack()
				if clone.FindPermanent(own.ID()).Tapped != (change != "target-control") {
					t.Fatal("clone lost target context")
				}
			})
		}
	}
}

func TestRONControlledTargetBattleGearLethal(t *testing.T) {
	g, a, b := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Ashnod's Battle Gear"), a.PlayerID())
	target := g.PutOnBattlefield(mage.NewCreature("Target", "", 2, 2), a.PlayerID())
	a.ManaPool().Add(core.Colorless, 2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{target.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(target.ID()) != nil || len(a.Graveyard()) != 1 || len(b.Graveyard()) != 0 {
		t.Fatal("toughness reduction did not use state-based actions")
	}
}

func TestRONControlledTargetModes(t *testing.T) {
	const body = `Targeted([TargetPermanent(And([IsCardtype(Creature),ControlledByAPlayer(SinglePlayer(You))]))],ActionList([TapPermanent(Ref_TargetPermanent)]))`
	for mode := range 2 {
		g, a, b := ronModalGame(mode)
		own := g.PutOnBattlefield(mage.NewCreature("Own", "", 2, 3), a.PlayerID())
		other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 3), b.PlayerID())
		card := permanentScopeFactory(t, "Instant", `SpellActions(Modal_ChooseOne([`+body+`,`+body+`]))`)()
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{other.ID()}, 0); err == nil {
			t.Fatal("opponent modal target accepted")
		}
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{own.ID()}, 0); err != nil {
			t.Fatal(err)
		}
		obj := g.StackPeek()
		if obj.ModeChoice != mode || len(obj.ModalTargets[1-mode]) != 0 {
			t.Fatal("targets escaped their mode")
		}
		g.ResolveStack()
		if !g.FindPermanent(own.ID()).Tapped || g.FindPermanent(other.ID()).Tapped {
			t.Fatal("wrong modal target")
		}
	}
}
