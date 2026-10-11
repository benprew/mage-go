package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestLiveDamageHistoryRules(t *testing.T) {
	for _, mode := range []string{"both", "regeneration"} {
		t.Run(mode, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := g.PutOnBattlefield(NewCreature("Source", "", 1, 5), a.PlayerID())
			victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 5), b.PlayerID())
			victim.Card.SetOwner(b.PlayerID())
			if mode != "exile" {
				if err := ApplyEffect(g, CantRegenerateDealtDamageByThisTurn(This, AnyPermanent), source.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			if mode != "regeneration" {
				if err := ApplyEffect(g, ExileDealtDamageByThisTurn(This, AnyPermanent), source.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			g.AddRegenerationShield(victim.ID())
			g.DealDamageToPermanent(victim, 1, source.ID())
			g.DestroyPermanent(victim)
			if g.FindPermanent(victim.ID()) != nil {
				t.Fatal("later damage did not prevent regeneration")
			}
			if (g.FindExiledCard(victim.Card.ID()) != nil) != (mode == "both") {
				t.Fatal("wrong death destination")
			}
		})
	}
}

func TestLiveDamageHistoryInvalid(t *testing.T) {
	for _, effect := range []ResolvingEffect{CantRegenerateDealtDamageByThisTurn(nil, AnyPermanent), ExileDealtDamageByThisTurn(nil, AnyPermanent)} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("nil source accepted")
		}
	}
}

func TestLiveDamageHistoryLifecycle(t *testing.T) {
	for _, change := range []string{"before", "prevented", "zero", "other-source", "source-return", "victim-return", "source-phase", "victim-phase-cycle", "source-departure", "cleanup", "clone", "abilities", "sacrifice", "zero-toughness"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := g.PutOnBattlefield(NewCreature("Source", "", 1, 5), a.PlayerID())
			victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 5), b.PlayerID())
			source.Card.SetOwner(a.PlayerID())
			victim.Card.SetOwner(b.PlayerID())
			saved := g.CaptureDamageSource(source.ID())
			if change == "before" {
				g.DealDamageToPermanent(victim, 1, source.ID())
			}
			for _, effect := range []ResolvingEffect{CantRegenerateDealtDamageByThisTurn(This, AnyPermanent), ExileDealtDamageByThisTurn(This, AnyPermanent)} {
				if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			amount := 1
			switch change {
			case "prevented":
				g.AddPreventionShield(victim.ID(), 1)
			case "zero":
				amount = 0
			case "other-source":
				saved = g.CaptureDamageSource(g.PutOnBattlefield(NewCreature("Other", "", 1, 5), a.PlayerID()).ID())
			case "source-return":
				g.RemoveFromBattlefield(source)
				source = g.PutOnBattlefield(source.Card, a.PlayerID())
				saved = g.CaptureDamageSource(source.ID())
			case "source-departure":
				g.RemoveFromBattlefield(source)
				g.zones.ClearLKI()
			case "source-phase":
				g.PhaseOut(source.ID())
			case "victim-phase-cycle":
				g.PhaseIn(g.PhaseOut(victim.ID()))
			case "abilities":
				g.AddContinuousEffect(RemoveAllAbilities(victim.ID(), core.EndOfTurn))
				g.ApplyContinuousEffects()
			}
			if change != "before" {
				g.DealDamageToPermanentFromSource(victim, amount, saved)
			}
			switch change {
			case "victim-return":
				g.RemoveFromBattlefield(victim)
				victim = g.PutOnBattlefield(victim.Card, b.PlayerID())
			case "cleanup":
				g.RunStepWithPriority(core.Cleanup)
			case "clone":
				g = g.Clone()
			}
			victim = g.FindPermanent(victim.ID())
			g.AddRegenerationShield(victim.ID())
			switch change {
			case "sacrifice":
				g.DoSacrifice(victim)
			case "zero-toughness":
				g.AddCountersWithReplacement(victim, core.M1M1, 5, source.ID(), false)
				g.CheckStateBasedActions()
			default:
				g.DestroyPermanent(victim)
			}
			wantExile := change != "prevented" && change != "zero" && change != "other-source" && change != "source-return" && change != "victim-return" && change != "cleanup"
			if (g.FindExiledCard(victim.Card.ID()) != nil) != wantExile {
				t.Fatalf("exile=%v want=%v", g.FindExiledCard(victim.Card.ID()) != nil, wantExile)
			}
			if wantExile && g.CreatureDeaths() != 0 {
				t.Fatal("exile counted as death")
			}
		})
	}
}

func TestLiveDamageHistoryReplacementOrder(t *testing.T) {
	for _, index := range []int{0, 1} {
		a := NewBasePlayer("A")
		b := &deathExileOrderPlayer{BasePlayer: NewBasePlayer("B"), index: index}
		g := NewGame(a, b)
		source := g.PutOnBattlefield(NewCreature("Source", "", 1, 5), a.PlayerID())
		victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 5), b.PlayerID())
		victim.Card.SetOwner(b.PlayerID())
		g.AddRegenerationShield(victim.ID())
		if err := ApplyEffect(g, ExileDealtDamageByThisTurn(This, IsCreature), source.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		g.DealDamageToPermanent(victim, 1, source.ID())
		g.DestroyPermanent(victim)
		if b.choices != 1 || (g.FindPermanent(victim.ID()) != nil) != (index == 0) || (g.FindExiledCard(victim.Card.ID()) != nil) != (index == 1) {
			t.Fatal("replacement order lost affected controller choice")
		}
	}
}

func TestLiveDamageHistorySimultaneousAndWaiting(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 5), a.PlayerID())
	saved := g.CaptureDamageSource(source.ID())
	first := g.PutOnBattlefield(NewCreature("Lethal", "", 1, 3), b.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Zero", "", 1, 3), b.PlayerID())
	for _, effect := range []ResolvingEffect{CantRegenerateDealtDamageByThisTurn(This, IsCreature), ExileDealtDamageByThisTurn(This, IsCreature)} {
		if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	refs := g.WaitingExileSources()
	if len(refs) != 1 || !refs[0].SameObject(saved) {
		t.Fatal("missing exact history source before damage")
	}
	g.DealSimultaneousDamage(PermanentDamageAssignment(first, 1, saved), PermanentDamageAssignment(second, 1, saved))
	g.RemoveFromBattlefield(source)
	returned := g.PutOnBattlefield(source.Card, a.PlayerID())
	if g.WaitingExileSources()[0].Incarnation() == returned.incarnationID {
		t.Fatal("waiting reference followed return")
	}
	clone := g.Clone()
	probe := &deathExileBatchProbe{replacementBase: replacementBase{duration: core.EndOfTurn}, first: first.ID(), second: second.ID(), t: t}
	g.AddReplacementEffect(probe)
	g.AddRegenerationShield(first.ID())
	first.Damage = 3
	g.AddCountersWithReplacement(second, core.M1M1, 3, source.ID(), false)
	g.CheckStateBasedActions()
	if probe.calls != 2 || g.FindExiledCard(first.Card.ID()) == nil || g.FindExiledCard(second.Card.ID()) == nil || g.CreatureDeaths() != 0 {
		t.Fatal("mixed death batch failed")
	}
	g.RunStepWithPriority(core.Cleanup)
	if len(g.WaitingExileSources()) != 0 || len(clone.WaitingExileSources()) != 1 || clone.FindPermanent(first.ID()) == nil {
		t.Fatal("cleanup or clone shared rule state")
	}
}

func TestLiveDamageHistoryCreatureAtDamage(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 5), a.PlayerID())
	victimCard := NewCreature("Changing", "", 1, 5)
	victimCard.SetOwner(b.PlayerID())
	victim := g.PutOnBattlefield(victimCard, b.PlayerID())
	if err := ApplyEffect(g, CantRegenerateCreaturesDealtDamageByThisTurn(This), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.DealDamageToPermanent(victim, 1, source.ID())
	victimCard.types = []core.CardType{core.TypeArtifact}
	g.ApplyContinuousEffects()
	g.AddRegenerationShield(victim.ID())
	g.DestroyPermanent(victim)
	if g.FindPermanent(victim.ID()) != nil {
		t.Fatal("type change erased the no-regeneration rule")
	}
}

func TestLiveDamageHistoryCloneWrites(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 5), a.PlayerID())
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 5), b.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 5), b.PlayerID())
	if err := ApplyEffect(g, CantRegenerateCreaturesDealtDamageByThisTurn(This), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.DealDamageToPermanent(first, 1, source.ID())
	clone := g.Clone()
	clone.DealDamageToPermanent(clone.FindPermanent(second.ID()), 1, source.ID())
	if !g.CanRegeneratePermanent(second) || clone.CanRegeneratePermanent(clone.FindPermanent(second.ID())) {
		t.Fatal("clone shared recipient history writes")
	}
	if g.CanRegeneratePermanent(first) || clone.CanRegeneratePermanent(clone.FindPermanent(first.ID())) {
		t.Fatal("clone lost initial history")
	}
}

func TestLiveDamageHistoryWaitingDeclarations(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	source.Card.SetOwner(a.PlayerID())
	body := Sequence(CantRegenerateCreaturesDealtDamageByThisTurn(This), ExileDealtDamageByThisTurn(This, AnyPermanent), TapEach(PermanentsDealtDamageByThisTurn(This, MatchingPermanents(AnyPermanent))))
	if err := ApplyEffect(g, AtNext(EachEndStep, body), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	references := g.WaitingDelayedPermanentSources()
	if len(references) != 1 {
		t.Fatal("missing source declaration")
	}
	g.RemoveFromBattlefield(source)
	returned := g.PutOnBattlefield(source.Card, a.PlayerID())
	refs := g.WaitingDelayedPermanentSources()
	if len(refs) != 1 || !refs[0].SameObject(references[0]) || refs[0].Incarnation() == returned.incarnationID || len(g.WaitingExileSources()) != 0 {
		t.Fatal("waiting query executed the rule or followed return")
	}
}
