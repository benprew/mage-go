package ron

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func activationWindowSource(modifier string) string {
	return `Card(Name:"Window",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[ActivatedWithModifiers(And([PayMana([ManaCostGeneric(1)]),TapPermanent(ThisPermanent)]),ActionList([GainLife(Integer(1))]),` + modifier + `)])`
}

func activationWindowFactory(t *testing.T, modifier string) mage.CardFactory {
	t.Helper()
	nodes, err := Parse(strings.NewReader(activationWindowSource(modifier)), "window.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	return factory
}

func TestRONActivationWindowTiming(t *testing.T) {
	for _, player := range []string{"You", "Opponent"} {
		for _, point := range []struct {
			name string
			step core.PhaseStep
		}{
			{"IsBeforeAttackersDeclared", core.DeclareAttackers},
			{"IsBeforeBlockersDeclared", core.DeclareBlockers},
			{"IsBeforeCombatDamageStep", core.FirstStrikeDamage},
		} {
			for _, active := range []int{0, 1} {
				for step := core.Untap; step <= core.Cleanup; step++ {
					t.Run(fmt.Sprintf("%s/%s/%d/%s", player, point.name, active, step), func(t *testing.T) {
						factory := activationWindowFactory(t, `And([ActivateOnlyIf(IsAPlayersTurn(`+player+`)),And([ActivateOnlyIf(`+point.name+`)])])`)
						g, a, _ := ronGame()
						card := factory()
						card.SetOwner(a.PlayerID())
						p := g.PutOnBattlefield(card, a.PlayerID())
						g.SetActivePlayerIndex(active)
						g.SetStep(step)
						a.ManaPool().Add(core.Colorless, 1)
						want := (active == 0) == (player == "You") && step < point.step
						ability := p.Card.Abilities()[0].(mage.ActivatedAbility)
						for range 2 {
							if ability.CanActivate(a.PlayerID(), g) != want {
								t.Fatal("query ignored turn or window")
							}
						}
						err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil)
						if (err == nil) != want {
							t.Fatalf("activation=%v, want=%v", err, want)
						}
						if !want {
							if p.Tapped || a.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
								t.Fatal("rejection spent resources")
							}
							return
						}
						if !p.Tapped || a.ManaPool().TotalMana() != 0 || g.StackSize() != 1 {
							t.Fatal("activation did not pay")
						}
						clone := g.Clone()
						g.SetStep(core.Cleanup)
						g.SetActivePlayerIndex(1 - active)
						for _, game := range []*mage.Game{g, clone} {
							game.ResolveStack()
							if game.GetPlayer(a.PlayerID()).Life() != 21 {
								t.Fatal("pending action repeated timing check")
							}
						}
					})
				}
			}
		}
	}
}

func TestRONActivationWindowConjunctionHistory(t *testing.T) {
	for _, modifier := range []string{
		`ActivateOnlyIf(And([IsBeforeBlockersDeclared,And([IsDuringCombat,IsPlayersTurn(You)])]))`,
		`And([ActivateOnlyIf(IsBeforeBlockersDeclared),And([ActivateOnlyIf(IsDuringCombat),ActivateOnlyIf(IsAPlayersTurn(You))])])`,
		`And([ActivateOnlyIf(IsDuringCombat),ActivateOnlyIf(IsBeforeBlockersDeclared)])`,
	} {
		factory := activationWindowFactory(t, modifier)
		g, a, _ := ronGame()
		p := g.PutOnBattlefield(factory(), a.PlayerID())
		a.ManaPool().Add(core.Colorless, 1)
		ability := p.Card.Abilities()[0].(mage.ActivatedAbility)
		for _, state := range []struct {
			step core.PhaseStep
			want bool
		}{
			{core.PrecombatMain, false}, {core.BeginCombat, true}, {core.DeclareAttackers, true}, {core.DeclareBlockers, false}, {core.EndCombat, false}, {core.PostcombatMain, false}, {core.BeginCombat, true},
		} {
			g.SetStep(state.step)
			if ability.CanActivate(a.PlayerID(), g) != state.want {
				t.Fatalf("%s at %s", modifier, state.step)
			}
		}
		clone := g.Clone()
		g.SetStep(core.DeclareBlockers)
		if err := clone.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		clone.ResolveStack()
		if a.Life() != 20 || clone.GetPlayer(a.PlayerID()).Life() != 21 {
			t.Fatal("clone history or resources shared")
		}
	}
	for _, steps := range [][]core.PhaseStep{{core.PrecombatMain, core.PostcombatMain}, {core.PrecombatMain, core.BeginCombat, core.EndCombat}} {
		factory := activationWindowFactory(t, `ActivateOnlyIf(IsBeforeAttackersDeclared)`)
		g, a, _ := ronGame()
		p := g.PutOnBattlefield(factory(), a.PlayerID())
		a.ManaPool().Add(core.Colorless, 1)
		for _, step := range steps {
			g.SetStep(step)
		}
		g.SetStep(core.BeginCombat)
		if p.Card.Abilities()[0].(mage.ActivatedAbility).CanActivate(a.PlayerID(), g) {
			t.Fatal("extra combat restored first window")
		}
		g.SetTurn(g.CurrentTurn() + 1)
		g.SetStep(core.PrecombatMain)
		if !p.Card.Abilities()[0].(mage.ActivatedAbility).CanActivate(a.PlayerID(), g) {
			t.Fatal("new turn did not reset window")
		}
	}
}

func TestRONActivationWindowInvalidAndNative(t *testing.T) {
	valid := activationWindowSource(`ActivateOnlyIf(IsAPlayersTurn(Opponent))`)
	for _, modifier := range []string{
		`ActivateOnlyIf(IsAPlayersTurn())`, `ActivateOnlyIf(IsAPlayersTurn(You,Opponent))`, `ActivateOnlyIf(IsPlayersTurn(You()))`,
		`ActivateOnlyIf(IsAPlayersTurn(Ref_TargetPlayer))`, `ActivateOnlyIf(IsAPlayersTurn(TheActivePlayer))`,
		`ActivateOnlyIf(IsBeforeAttackersDeclared())`, `ActivateOnlyIf(IsBeforeBlockersDeclared(You))`,
		`ActivateOnlyIf(And([]))`, `ActivateOnlyIf(And([IsAPlayersTurn(You),Unknown]))`,
		`ActivateOnlyIf(And([IsDuringCombat,IsDuringUpkeep]))`, `ActivateOnlyIf(Or([IsAPlayersTurn(You),IsAPlayersTurn(Opponent)]))`,
	} {
		nodes, err := Parse(strings.NewReader(activationWindowSource(modifier)), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", modifier)
		}
	}
	registry := mage.NewRegistry()
	if err := LoadInto(registry, fstest.MapFS{"window.ron": {Data: []byte(valid)}}, "*.ron"); err != nil {
		t.Fatal(err)
	}
	invalid := strings.Replace(activationWindowSource(`ActivateOnlyIf(IsAPlayersTurn(ActionPlayer))`), `Name:"Window"`, `Name:"Invalid"`, 1)
	if err := LoadInto(registry, fstest.MapFS{"window.ron": {Data: []byte(valid + invalid)}}, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("invalid collection changed registry")
	}
}

func TestRONActivationWindowImpGameplay(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, change := range []string{"attack", "tapped", "phase", "cycle", "return", "control", "wall", "fresh", "own-turn", "late", "regenerate", "copy", "source-phase"} {
			t.Run(set+"/"+change, func(t *testing.T) {
				factory, err := Compile(hostUntapRecord(t, set, "Nettling Imp"))
				if err != nil {
					t.Fatal(err)
				}
				g, a, b := ronGame()
				source := g.PutOnBattlefield(factory(), a.PlayerID())
				source.RevokeBaseAttr(core.AttrSummonSick)
				options := []mage.CardOption{}
				if change == "wall" {
					options = append(options, mage.WithSubTypes("Wall"))
				}
				card := mage.NewCreature("Subject", "", 2, 3, options...)
				card.SetOwner(b.PlayerID())
				victim := g.PutOnBattlefield(card, b.PlayerID())
				victim.RevokeBaseAttr(core.AttrSummonSick)
				peer := g.PutOnBattlefield(mage.NewCreature("Peer", "", 2, 3), b.PlayerID())
				peer.RevokeBaseAttr(core.AttrSummonSick)
				g.SetTurn(2)
				g.SetActivePlayerIndex(1)
				g.SetStep(core.PrecombatMain)
				if change == "fresh" {
					g.RemoveFromBattlefield(victim)
					victim = g.PutOnBattlefield(card, b.PlayerID())
				}
				if change == "own-turn" {
					g.SetActivePlayerIndex(0)
				}
				if change == "late" {
					g.SetStep(core.DeclareAttackers)
				}
				if change == "tapped" || change == "regenerate" {
					g.TapPermanent(victim)
				}
				err = g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()})
				reject := change == "wall" || change == "fresh" || change == "own-turn" || change == "late"
				if reject {
					if err == nil || source.Tapped || g.StackSize() != 0 {
						t.Fatal("illegal activation changed state")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if !source.Tapped {
					t.Fatal("tap payment missing")
				}
				if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				switch change {
				case "phase":
					g.PhaseOut(victim.ID())
				case "cycle":
					g.PhaseIn(g.PhaseOut(victim.ID()))
				case "return":
					g.RemoveFromBattlefield(victim)
					victim = g.PutOnBattlefield(card, b.PlayerID())
					victim.RevokeBaseAttr(core.AttrSummonSick)
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: victim.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				}
				if change == "source-phase" {
					g.PhaseOut(source.ID())
				} else {
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				}
				g.ResolveStack()
				if change == "regenerate" {
					g.AddRegenerationShield(victim.ID())
				}
				clone := g.Clone()
				for _, game := range []*mage.Game{g, clone} {
					game.ExecuteAttackers(b.PlayerID(), nil)
					attacks := change == "attack" || change == "cycle" || change == "copy" || change == "source-phase"
					if game.GetCombat().IsAttacking(victim.ID()) != attacks || game.GetCombat().IsAttacking(peer.ID()) {
						t.Fatal("attack membership")
					}
					game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
					game.PutTriggersOnStack()
					game.ResolveStack()
					survives := change != "tapped"
					if (game.FindPermanentIncludingPhased(victim.ID()) != nil) != survives {
						t.Fatal("delayed destruction or saved identity")
					}
					if game.FindPermanent(peer.ID()) == nil {
						t.Fatal("unselected peer destroyed")
					}
				}
			})
		}
	}
}

func TestRONActivationWindowCostsControlAndNativeSkip(t *testing.T) {
	factory := activationWindowFactory(t, `And([ActivateOnlyIf(IsPlayersTurn(You)),ActivateOnlyIf(IsBeforeBlockersDeclared),ActivateOnlyAsASorcery])`)
	g, a, b := ronGame()
	card := factory()
	card.SetOwner(b.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil || p.Tapped || g.StackSize() != 0 {
		t.Fatal("failed payment changed state")
	}
	a.ManaPool().Add(core.Colorless, 1)
	combat := g.Clone()
	combat.SetStep(core.BeginCombat)
	if err := combat.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil || combat.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 1 || combat.FindPermanent(p.ID()).Tapped {
		t.Fatal("window replaced sorcery restriction")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatal("turn read owner instead of actor")
	}
	g.UntapPermanent(p)
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	b.ManaPool().Add(core.Colorless, 1)
	if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 0, nil); err == nil {
		t.Fatal("old controller turn supplied new actor")
	}
	g.SetActivePlayerIndex(1)
	if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if b.Life() != 21 {
		t.Fatal("control change lost activation")
	}
	factory = activationWindowFactory(t, `ActivateOnlyIf(IsBeforeCombatDamageStep)`)
	g, a, _ = ronGame()
	p = g.PutOnBattlefield(factory(), a.PlayerID())
	g.RunStepWithPriority(core.PrecombatMain)
	g.RunStepWithPriority(core.BeginCombat)
	g.SetStep(core.DeclareAttackers)
	a.ManaPool().Add(core.Colorless, 1)
	ability := p.Card.Abilities()[0].(mage.ActivatedAbility)
	if !ability.CanActivate(a.PlayerID(), g) {
		t.Fatal("window closed before empty attackers step ended")
	}
	g.RunStepWithPriority(core.DeclareAttackers)
	a.ManaPool().Add(core.Colorless, 1)
	if ability.CanActivate(a.PlayerID(), g) {
		t.Fatal("native skipped damage window stayed open")
	}
}

func TestRONActivationWindowInventory(t *testing.T) {
	nodes, err := Parse(strings.NewReader(activationWindowSource(`ActivateOnlyIf(And([IsAPlayersTurn(Opponent),IsBeforeAttackersDeclared]))`)), "inventory.ron")
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string]*InventorySignature{}
	walkInventory(nodes[0], "Window", "Card", "card", "collection", true, entries)
	for name, role := range map[string]string{"And": "activation_condition", "IsAPlayersTurn": "activation_condition", "Opponent": "player_reference", "IsBeforeAttackersDeclared": "activation_condition"} {
		found := false
		for _, entry := range entries {
			if entry.Name == name && entry.Role == role {
				found = true
			}
		}
		if !found {
			t.Errorf("%s missing %s", name, role)
		}
	}
}

func TestRONActivationWindowModalPermission(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `ActivatedWithModifiers(PayMana([ManaCostGeneric(1)]),Modal_ChooseOne([ActionList([GainLife(Integer(2))]),ActionList([GainLife(Integer(3))])]),And([OtherPlayersMayActivate(AnyPlayer),ActivateOnlyIf(IsPlayersTurn(You)),ActivateOnlyIf(IsBeforeAttackersDeclared)]))`)
	g, a, b := ronGame()
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	q := g.PutOnBattlefield(factory(), a.PlayerID())
	b.ManaPool().Add(core.Colorless, 2)
	if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 0, nil); err == nil || b.ManaPool().TotalMana() != 2 || g.StackSize() != 0 {
		t.Fatal("permission replaced acting-player turn condition")
	}
	g.SetActivePlayerIndex(1)
	for _, source := range []*mage.Permanent{p, q} {
		if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
	}
	g.SetStep(core.DeclareAttackers)
	g.ResolveStack()
	if b.Life() != 24 || a.Life() != 20 || b.ManaPool().TotalMana() != 0 {
		t.Fatal("modal actor, independent factories or pending window")
	}
}
