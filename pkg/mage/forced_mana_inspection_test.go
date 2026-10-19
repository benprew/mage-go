package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestForcedManaInspectionDeclaration(t *testing.T) {
	target := DeclarePlayerTarget("player")
	result := BindLostMana("lost")
	instruction := ForcedManaLossInstruction{Subject: target, Filter: IsLand, Result: result}
	effect := instruction.Effect()
	got, ok := InspectForcedManaLossInstruction(effect)
	if !ok || got.Subject != target || got.Result != result {
		t.Fatal("lost native operands")
	}
	if _, ok := InspectForcedManaLossInstruction(ActivateManaAbilityOfEach(target, IsLand)); ok {
		t.Fatal("activation alone is a different instruction")
	}
	for _, bad := range []ForcedManaLossInstruction{{}, {Subject: You}, {Subject: You, Filter: IsLand, Result: BindLostMana("")}} {
		if bad.Effect() != nil {
			t.Fatal("invalid declaration built an effect")
		}
	}
	var missing *PlayerTargetBinding
	if (ForcedManaLossInstruction{Subject: missing, Filter: IsLand}).Effect() != nil {
		t.Fatal("typed nil player accepted")
	}
	for _, restore := range []bool{false, true} {
		instruction := LostManaInstruction{Subject: target, Result: result, Restore: restore}
		got, ok := InspectLostManaInstruction(instruction.Effect())
		if !ok || got != instruction {
			t.Fatal("lost transfer declaration")
		}
	}
	for _, bad := range []LostManaInstruction{{}, {Subject: You}, {Subject: You, Result: BindLostMana("")}, {Subject: missing, Result: result}} {
		if bad.Effect() != nil {
			t.Fatal("invalid transfer built an effect")
		}
	}
}

func TestForcedManaInspectionSpell(t *testing.T) {
	for _, state := range []string{"only", "prefix", "suffix", "empty prefix", "illegal prefix", "all illegal", "missing", "missing suffix", "missing all illegal", "foreign", "forward", "duplicate", "foreign result", "foreign result suffix", "foreign result all illegal", "empty result", "opaque player", "opaque filter", "opaque suffix", "opaque all illegal", "loss transfer", "transfer only", "same player", "active", "no result", "clone"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, land := targetQueryFixture(t)
			player := g.GetOpponent(obj.Controller)
			player.ManaPool().AddRestricted(core.Colorless, 2, core.ArtifactSpellsOnly{})
			target := DeclarePlayerTarget("player")
			obj.TargetSpecs = append(obj.TargetSpecs, target.targetSpec())
			obj.Targets = append(obj.Targets, player.PlayerID())
			result := BindLostMana("lost")
			subject := PlayerRef(target)
			filter := IsLand
			want, known := false, true
			switch state {
			case "empty prefix":
				player.ManaPool().Clear()
			case "illegal prefix", "all illegal":
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
			case "forward", "duplicate", "foreign result", "foreign result suffix", "foreign result all illegal", "empty result", "opaque player", "opaque filter", "opaque suffix", "opaque all illegal", "transfer only":
				known = false
			case "same player":
				subject = You
			case "active":
				subject = ActivePlayer
			}
			if state == "opaque player" {
				subject = lifeInspectionPlayerTrap{}
			}
			if state == "opaque filter" {
				filter = NewPermanentFilter("opaque", func(*Permanent, *Game) bool { t.Fatal("filter callback ran"); return true })
			}
			if state == "empty result" {
				result = BindLostMana("")
			}
			loss := (ForcedManaLossInstruction{Subject: subject, Filter: filter, Result: result}).Effect()
			transfer := AddLostManaFor(You, result)
			tree := Sequence(loss, transfer)
			switch state {
			case "prefix", "empty prefix":
				tree = Sequence(loss, transfer, Destroy(land))
				known = false
			case "illegal prefix":
				tree = Sequence(loss, Destroy(land))
				want = true
			case "suffix", "missing suffix", "missing all illegal":
				tree = Sequence(Destroy(land), loss, transfer)
				want = known
			case "forward":
				tree = Sequence(transfer, loss)
			case "duplicate":
				tree = Sequence(loss, loss, transfer)
			case "foreign result", "foreign result suffix", "foreign result all illegal":
				tree = Sequence(loss, AddLostManaFor(You, BindLostMana("lost")))
				if state != "foreign result" {
					tree = Sequence(Destroy(land), tree)
				}
				if state == "foreign result all illegal" {
					p.GrantBaseAttr(core.Shroud)
					obj.Targets[1] = uuid.New()
				}
			case "opaque suffix":
				tree = Sequence(Destroy(land), loss, transfer, destructionInspectionTrap{})
			case "opaque all illegal":
				p.GrantBaseAttr(core.Shroud)
				obj.Targets[1] = uuid.New()
				tree = Sequence(loss, transfer, destructionInspectionTrap{})
			case "loss transfer":
				tree = Sequence(LoseUnspentManaInto(target, result), transfer)
			case "transfer only":
				tree = Sequence(transfer)
			case "no result":
				tree = Sequence((ForcedManaLossInstruction{Subject: subject, Filter: IsLand}).Effect())
			}
			obj.Effects = []Effect{tree}
			before := g.Clone()
			for _, game := range []*Game{g, g.Clone()} {
				for range 2 {
					certain, k := game.InspectSpellDestruction(game.StackPeek(), []PermanentVisit{{p.ID(), p.incarnationID}})
					if certain != want || k != known {
						t.Fatalf("proof=%v/%v want=%v/%v", certain, k, want, known)
					}
				}
			}
			if !reflect.DeepEqual(player.ManaPool(), before.GetPlayer(player.PlayerID()).ManaPool()) || !reflect.DeepEqual(g.random, before.random) || g.StackSize() != before.StackSize() || len(g.resolution.lostManaBindings) != 0 {
				t.Fatal("inspection changed state")
			}
		})
	}
}

func TestForcedManaInspectionExecution(t *testing.T) {
	a := NewBasePlayer("A")
	b := &forcedManaPlayer{BasePlayer: NewBasePlayer("B"), amount: 2}
	g := NewGame(a, b)
	g.SetActivePlayerIndex(1)
	paid := BindNumber("storage payment")
	land := g.PutOnBattlefield(NewLand("Storage", WithAbility(Activate(Costs(TapCost(This), RemoveAnyCountersCost(core.Storage, paid)), AddManaAmount(core.Black, FromNumber(paid))))), b.PlayerID())
	land.AddCounter(core.Storage, 3)
	bad := g.PutOnBattlefield(NewLand("Bad", WithAbility(Activate(Costs(TapCost(This), LifePayCost(11), LifePayCost(11)), AddMana(core.White, 5)))), b.PlayerID())
	if err := ApplyEffect(g, ReplaceTappedManaTypeForPlayerThisTurn(ActivePlayer, IsLand, core.Blue), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	b.ManaPool().AddRestricted(core.Colorless, 2, core.ArtifactSpellsOnly{})
	result := BindLostMana("lost")
	effect := Sequence((ForcedManaLossInstruction{Subject: ActivePlayer, Filter: IsLand, Result: result}).Effect(), (LostManaInstruction{Subject: You, Result: result, Restore: true}).Effect())
	card := NewInstant("Transfer", "", Spell(effect))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	before := b.ManaPool().SnapshotPool()
	for range 2 {
		if certain, known := g.InspectSpellDestruction(g.StackPeek(), nil); certain || !known {
			t.Fatal("native tree unknown")
		}
	}
	if b.calls != 0 || land.Tapped || land.Counters[core.Storage] != 3 || !reflect.DeepEqual(before, b.ManaPool().SnapshotPool()) {
		t.Fatal("inspection activated a mana ability")
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		pool := game.GetPlayer(a.PlayerID()).ManaPool()
		amount := 2
		if game != g {
			amount = 3
		}
		if pool.Count(core.Blue) != amount || pool.Count(core.Colorless) != 2 || pool.TotalProducedThisTurn() != 0 || pool.CanPay(core.ManaCost{Generic: amount + 2}, nil) {
			t.Fatalf("clone=%v blue=%d colorless=%d produced=%d total=%d donor=%d tapped=%v counters=%d saved=%v", game != g, pool.Count(core.Blue), pool.Count(core.Colorless), pool.TotalProducedThisTurn(), pool.TotalMana(), game.GetPlayer(b.PlayerID()).ManaPool().TotalMana(), game.FindPermanent(land.ID()).Tapped, game.FindPermanent(land.ID()).Counters[core.Storage], g.resolution.lostManaBindings)
		}
		if game.GetPlayer(b.PlayerID()).Life() != 20 || game.FindPermanent(bad.ID()).Tapped || !game.FindPermanent(land.ID()).Tapped || game.FindPermanent(land.ID()).Counters[core.Storage] != 3-amount {
			t.Fatal("payment rollback or counter payment changed")
		}
	}
}
