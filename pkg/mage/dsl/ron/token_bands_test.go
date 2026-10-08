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

func TestRONTokenBandsMaster(t *testing.T) {
	for _, change := range []string{"none", "leave", "phase", "control"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", "Master of the Hunt")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Green, 2)
			a.ManaPool().Add(core.Colorless, 2)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			source := g.FindPermanent(card.ID())
			if source == nil || source.CurrentPower(g) != 2 || source.CurrentToughness(g) != 2 {
				t.Fatal("wrong source")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
				t.Fatal("unpaid activation accepted")
			}
			if g.StackSize() != 0 || len(g.FilterBattlefield(mage.IsToken)) != 0 {
				t.Fatal("failed payment changed game")
			}
			for range 2 {
				a.ManaPool().Add(core.Green, 2)
				a.ManaPool().Add(core.Colorless, 2)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
			}
			if len(g.FilterBattlefield(mage.IsToken)) != 0 || a.ManaPool().Count(core.Green) != 0 || a.ManaPool().Count(core.Colorless) != 0 {
				t.Fatal("payment or timing")
			}
			if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			switch change {
			case "leave":
				g.BouncePermanentToHand(source)
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
			case "phase":
				g.PhaseOut(source.ID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			clone := g.Clone()
			g.ResolveStack()
			if len(clone.FilterBattlefield(mage.IsToken)) != 0 {
				t.Fatal("clone changed")
			}
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				tokens := game.FilterBattlefield(mage.IsToken)
				if len(tokens) != 3 {
					t.Fatal("wrong count")
				}
				owners := map[uuid.UUID]int{}
				abilities := map[uuid.UUID]bool{}
				for _, p := range tokens {
					if p.Name() != "Wolves of the Hunt" || !p.HasSubType("Wolf") || p.CurrentPower(game) != 1 || p.CurrentToughness(game) != 1 || !slices.Contains(p.Colors(), core.Green) || p.HasKeyword(core.Banding) || p.Card.ManaCost().CMC() != 0 || p.Card.Owner() != p.ControllerID() {
						t.Fatal("wrong token")
					}
					if len(p.RuntimeAbilities) != 1 || p.RuntimeAbilities[0].Source() != p.ID() || abilities[p.RuntimeAbilities[0].AbilityID()] {
						t.Fatal("shared or wrong ability")
					}
					abilities[p.RuntimeAbilities[0].AbilityID()] = true
					owners[p.ControllerID()]++
				}
				if owners[a.PlayerID()] != 2 || owners[b.PlayerID()] != 1 {
					t.Fatal("resolution followed changed source controller")
				}
			}
		})
	}
}

func TestRONTokenBandsCombat(t *testing.T) {
	for _, side := range []string{"attack", "block"} {
		t.Run(side, func(t *testing.T) {
			a := &otherBandPlayer{BasePlayer: mage.NewBasePlayer("A")}
			b := &otherBandPlayer{BasePlayer: mage.NewBasePlayer("B")}
			g := mage.NewGame(a, b)
			controller := a.PlayerID()
			if side == "block" {
				controller = b.PlayerID()
			}
			source := g.PutOnBattlefield(repositoryCard(t, "LEG", "Master of the Hunt"), controller)
			for range 2 {
				g.GetPlayer(controller).ManaPool().Add(core.Green, 2)
				g.GetPlayer(controller).ManaPool().Add(core.Colorless, 2)
				if err := g.ActivateAbilityByIndex(controller, source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
			}
			g.BouncePermanentToHand(source)
			members := g.FilterBattlefield(mage.IsToken)
			enemyController := b.PlayerID()
			if side == "block" {
				enemyController = a.PlayerID()
			}
			enemy := g.PutOnBattlefield(mage.NewCreature("Enemy", "", 1, 20), enemyController)
			for _, p := range append(members, enemy) {
				p.RevokeBaseAttr(core.AttrSummonSick)
			}
			ids := []uuid.UUID{members[0].ID(), members[1].ID()}
			if side == "attack" {
				g.ExecuteAttackers(a.PlayerID(), ids)
				if err := g.FormAttackingBand(ids); err != nil {
					t.Fatal(err)
				}
				g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: enemy.ID(), AttackerID: ids[0]}})
			} else {
				g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{enemy.ID()})
				g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: ids[0], AttackerID: enemy.ID()}, {BlockerID: ids[1], AttackerID: enemy.ID()}})
			}
			g.AddPreventionShield(ids[0], 1)
			g.ExecuteCombatDamage()
			if g.FindPermanent(enemy.ID()).Damage != 2 || g.FindPermanent(ids[0]).Damage != 0 || g.FindPermanent(ids[1]).Damage != 0 {
				t.Fatal("wrong combat damage")
			}
			if (side == "attack" && (a.calls[enemy.ID()] != 1 || b.calls[enemy.ID()] != 0)) || (side == "block" && (b.calls[enemy.ID()] != 1 || a.calls[enemy.ID()] != 0)) {
				t.Fatal("wrong damage chooser")
			}
		})
	}
}

func TestRONTokenBandsInvalid(t *testing.T) {
	for _, ability := range []string{`BandsWithOthers`, `BandsWithOthers()`, `BandsWithOthers(IsCardtype(Creature),IsColor(Green))`, `BandsWithOthers(And([]))`, `BandsWithOthers(Other(ThisPermanent))`, `BandsWithOthers(ControlledByAPlayer(SinglePlayer(You)))`, `BandsWithOthers(IsNamed(NamedCard("")))`, `BandsWithOthers(SinglePermanent(Ref_TargetPermanent))`} {
		token := strings.Replace(saprolingRON, `SubtypesList([Saproling]), [])`, `SubtypesList([Saproling]), [`+ability+`])`, 1)
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([CreateTokens([` + token + `],[])]))])`
		if registry, err := LoadFS(fstest.MapFS{"bad.ron": {Data: []byte(src)}}, "*.ron"); err == nil || registry != nil {
			t.Fatalf("accepted %s", ability)
		}
	}
}

func TestRONTokenBandsLifecycle(t *testing.T) {
	for _, change := range []string{"wrong-name", "wrong-type", "remove-banding", "remove-abilities", "phase", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, _ := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, "LEG", "Master of the Hunt"), a.PlayerID())
			a.ManaPool().Add(core.Green, 2)
			a.ManaPool().Add(core.Colorless, 2)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			holder := g.FilterBattlefield(mage.IsToken)[0]
			var partner *mage.Permanent
			switch change {
			case "wrong-name":
				partner = g.PutOnBattlefield(mage.NewCreature("Wolf", "", 1, 20), a.PlayerID())
			case "wrong-type":
				partner = g.PutOnBattlefield(mage.NewArtifact("Wolves of the Hunt", ""), a.PlayerID())
			case "copy":
				partner = g.PutOnBattlefield(holder.Card.Copy(), a.PlayerID())
			default:
				partner = g.PutOnBattlefield(mage.NewCreature("Wolves of the Hunt", "", 1, 20), a.PlayerID())
			}
			switch change {
			case "remove-banding":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveKeyword(mage.This, core.Banding)), holder.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "remove-abilities":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), holder.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "phase":
				g.PhaseOut(holder.ID())
			}
			for _, p := range []*mage.Permanent{g.FindPermanentIncludingPhased(holder.ID()), partner} {
				p.RevokeBaseAttr(core.AttrSummonSick)
			}
			ids := []uuid.UUID{holder.ID(), partner.ID()}
			g.ExecuteAttackers(a.PlayerID(), ids)
			err := g.FormAttackingBand(ids)
			if (change == "copy") != (err == nil) {
				t.Fatalf("formation after %s: %v", change, err)
			}
			if change == "remove-banding" || change == "remove-abilities" {
				g.GetEffects().RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				if err := g.FormAttackingBand(ids); err != nil {
					t.Fatal("cleanup did not restore quality", err)
				}
			}
		})
	}
}

func TestRONTokenBandsSpellBatch(t *testing.T) {
	src := `Card(Name:"Maker",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([CreateTokens([NumberTokens(Integer(2),NamedTokenWithPT("Member",PT(1,20),Colors([Green]),[],[Creature],SubtypesList([Wolf]),[BandsWithOthers(IsNamed(NamedCard("Member"))),BandsWithOthers(IsColor(Green))]))],[])]))])`
	registry, err := LoadFS(fstest.MapFS{"maker.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	g, a, _ := ronGame()
	card, err := registry.CreateCard("Maker")
	if err != nil {
		t.Fatal(err)
	}
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	tokens := g.FilterBattlefield(mage.IsToken)
	if len(tokens) != 2 {
		t.Fatal("wrong batch")
	}
	partnerCard := mage.NewCreature("Other", "", 1, 20)
	partnerCard.SetColorOverride([]core.Color{core.Green})
	partner := g.PutOnBattlefield(partnerCard, a.PlayerID())
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), tokens[0].ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for _, p := range []*mage.Permanent{tokens[0], tokens[1], partner} {
		g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
	}
	ids := []uuid.UUID{tokens[1].ID(), partner.ID()}
	g.ExecuteAttackers(a.PlayerID(), ids)
	if err := g.FormAttackingBand(ids); err != nil {
		t.Fatal("independent second quality lost", err)
	}
}
