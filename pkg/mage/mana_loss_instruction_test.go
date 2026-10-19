package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestManaLossInstructionDeclaration(t *testing.T) {
	target := DeclarePlayerTarget("player")
	effect := (ManaLossInstruction{Subject: target}).Effect()
	got, ok := InspectManaLossInstructions(Sequence(effect, CompositeEffects("nested", effect)))
	if !ok || len(got) != 2 || got[0].Subject != target || got[1].Subject != target {
		t.Fatal("lost native declaration or order")
	}
	got[0].Subject = nil
	again, _ := InspectManaLossInstructions(effect)
	if again[0].Subject != target {
		t.Fatal("inspection shares storage")
	}
	var missing *PlayerTargetBinding
	for _, subject := range []PlayerRef{nil, missing} {
		if (ManaLossInstruction{Subject: subject}).Effect() != nil {
			t.Fatal("missing subject built an effect")
		}
	}
	var absent *loseUnspentManaEffect
	var sequence *sequenceEffect
	var composite *compositeEffect
	for _, tree := range []Effect{nil, absent, sequence, composite, LoseUnspentManaFor(nil), LoseUnspentManaFor(missing), Sequence(effect, destructionInspectionTrap{}), LoseUnspentManaInto(You, BindLostMana("result"))} {
		if _, ok := InspectManaLossInstructions(tree); ok {
			t.Fatal("incomplete or unknown tree accepted")
		}
	}
	for _, tree := range []Effect{Sequence(), CompositeEffects("empty"), LoseUnspentManaFor(You)} {
		if _, ok := InspectManaLossInstructions(tree); !ok {
			t.Fatal("native tree rejected")
		}
	}
}

func TestManaLossInstructionSpell(t *testing.T) {
	for _, state := range []string{"only", "prefix", "empty prefix", "suffix", "controller", "active", "opponent", "illegal", "all illegal", "missing", "missing suffix", "missing all illegal", "foreign", "duplicate", "saved", "opaque subject", "opaque suffix", "opaque all illegal", "history", "phase", "phase cycle", "return"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, land := targetQueryFixture(t)
			target := DeclarePlayerTarget("player")
			player := g.GetOpponent(obj.Controller)
			player.ManaPool().Add(core.Red, 3)
			player.ManaPool().AddRestricted(core.Colorless, 2, core.ArtifactSpellsOnly{})
			obj.TargetSpecs = append(obj.TargetSpecs, target.targetSpec())
			obj.Targets = append(obj.Targets, player.PlayerID())
			subject := PlayerRef(target)
			want, known := false, true
			switch state {
			case "controller":
				subject = You
			case "active":
				subject = ActivePlayer
			case "opponent":
				target = DeclarePlayerTarget("opponent", TargetOpponentOnly())
				subject = target
				obj.TargetSpecs[1] = target.targetSpec()
			case "empty prefix":
				player.ManaPool().Clear()
			case "illegal", "all illegal":
				obj.Targets[1] = uuid.New()
				if state == "all illegal" {
					p.GrantBaseAttr(core.Shroud)
				}
			case "missing", "missing suffix", "missing all illegal":
				obj.TargetSpecs = obj.TargetSpecs[:1]
				obj.Targets = obj.Targets[:1]
				known = false
				if state == "missing all illegal" {
					p.GrantBaseAttr(core.Shroud)
				}
			case "foreign":
				subject = DeclarePlayerTarget("player")
				known = false
			case "duplicate":
				obj.TargetSpecs = append(obj.TargetSpecs, target.targetSpec())
				obj.Targets = append(obj.Targets, player.PlayerID())
				known = false
			case "saved":
				subject = FromPlayer(BindPlayer("saved"))
				known = false
			case "opaque subject", "opaque all illegal":
				subject = lifeInspectionPlayerTrap{}
				known = false
				if state == "opaque all illegal" {
					obj.Targets[1] = uuid.New()
					p.GrantBaseAttr(core.Shroud)
				}
			case "history":
				target = DeclarePlayerTarget("history", TargetPlayerAttackedThisTurn())
				subject = target
				obj.TargetSpecs[1] = target.targetSpec()
			case "phase", "phase cycle":
				g.PhaseOut(p.ID())
				if state == "phase cycle" {
					g.PhaseIn([]uuid.UUID{p.ID()})
				}
			case "return":
				g.BouncePermanentToHand(p)
				g.GetPlayer(p.ControllerID()).RemoveFromHand(p.Card.ID())
				g.PutOnBattlefield(p.Card, p.ControllerID())
			}
			loss := (ManaLossInstruction{Subject: subject}).Effect()
			obj.Effects = []Effect{loss}
			switch state {
			case "prefix", "empty prefix":
				obj.Effects = []Effect{Sequence(loss, Destroy(land))}
				known = false
			case "suffix", "phase cycle":
				obj.Effects = []Effect{Sequence(Destroy(land), loss)}
				want = true
			case "illegal":
				obj.Effects = []Effect{Sequence(loss, Destroy(land))}
				want = true
			case "all illegal", "phase", "return":
				obj.Effects = []Effect{Sequence(Destroy(land), loss)}
			case "opaque suffix":
				obj.Effects = []Effect{Sequence(Destroy(land), loss, destructionInspectionTrap{})}
				known = false
			case "missing suffix", "missing all illegal":
				obj.Effects = []Effect{Sequence(Destroy(land), loss)}
			}
			before := g.Clone()
			for _, game := range []*Game{g, g.Clone()} {
				for range 2 {
					certain, k := game.InspectSpellDestruction(game.StackPeek(), []PermanentVisit{{p.ID(), p.incarnationID}})
					if certain != want || k != known {
						t.Fatalf("proof=%v/%v want=%v/%v", certain, k, want, known)
					}
				}
			}
			if !reflect.DeepEqual(player.ManaPool(), before.GetPlayer(player.PlayerID()).ManaPool()) || !reflect.DeepEqual(g.random, before.random) || g.StackSize() != before.StackSize() || player.Life() != 20 {
				t.Fatal("query changed mana, history, RNG, stack or life")
			}
		})
	}
}

func TestManaLossInstructionExecution(t *testing.T) {
	for _, subject := range []string{"controller", "active", "target"} {
		t.Run(subject, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			g.SetActivePlayerIndex(1)
			a.ManaPool().Add(core.Blue, 2)
			b.ManaPool().AddRestricted(core.Red, 3, core.ArtifactSpellsOnly{})
			ref, id := PlayerRef(You), a.PlayerID()
			var targets []TargetBinding
			var ids []uuid.UUID
			switch subject {
			case "active":
				ref, id = ActivePlayer, b.PlayerID()
			case "target":
				target := DeclarePlayerTarget("player")
				ref, id = target, b.PlayerID()
				targets, ids = []TargetBinding{target}, []uuid.UUID{id}
			}
			card := NewInstant("Empty", "", Spell((ManaLossInstruction{Subject: ref}).Effect(), WithTargets(targets...)))
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), ids, 0); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.ManaPool().TotalMana() != 2 || b.ManaPool().TotalMana() != 3 {
				t.Fatal("clone cleared live pools")
			}
			g.ResolveStack()
			for _, game := range []*Game{g, clone} {
				if game.GetPlayer(id).ManaPool().TotalMana() != 0 || len(game.GetPlayer(a.PlayerID()).ManaPool().ProducedThisTurn) != 2 || len(game.GetPlayer(b.PlayerID()).ManaPool().ProducedThisTurn) != 3 || game.GetPlayer(id).Life() != 20 {
					t.Fatal("wrong pool, history or life after resolution")
				}
				other, count := b.PlayerID(), 3
				if id == b.PlayerID() {
					other, count = a.PlayerID(), 2
				}
				if game.GetPlayer(other).ManaPool().TotalMana() != count {
					t.Fatal("another player's pool changed")
				}
			}
		})
	}
}

func TestManaLossInstructionCloneHistory(t *testing.T) {
	for _, custom := range []bool{false, true} {
		a := NewBasePlayer("A")
		var player Player = a
		if custom {
			player = &manaLossWrappedPlayer{Player: a}
		}
		g := NewGame(player, NewBasePlayer("B"))
		a.ManaPool().Add(core.Blue, 3)
		a.ManaPool().DrainGeneric(1)
		clone := g.Clone()
		pool := clone.GetPlayer(a.PlayerID()).ManaPool()
		if !reflect.DeepEqual(pool, a.ManaPool()) {
			t.Fatal("clone lost mana history")
		}
		pool.ProducedThisTurn[0].Color = core.Red
		pool.LastDrainedColors[core.Blue] = 99
		pool.Clear()
		if a.ManaPool().ProducedThisTurn[0].Color != core.Blue || a.ManaPool().LastDrainedColors[core.Blue] != 1 || a.ManaPool().TotalMana() != 2 {
			t.Fatal("clone shares mana history or pool storage")
		}
	}
}

type manaLossWrappedPlayer struct{ Player }
