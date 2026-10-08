package ron

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

var otherBandLands = []struct {
	name  string
	color core.Color
}{
	{"Adventurers' Guildhouse", core.Green}, {"Cathedral of Serra", core.White},
	{"Mountain Stronghold", core.Red}, {"Seafarer's Quay", core.Blue}, {"Unholy Citadel", core.Black},
}

type otherBandPlayer struct {
	*mage.BasePlayer
	calls map[uuid.UUID]int
}

func (p *otherBandPlayer) GetBandingDamageAssignment(_ *mage.Game, source *mage.Permanent, recipients []*mage.Permanent, amount int) map[uuid.UUID]int {
	if p.calls == nil {
		p.calls = make(map[uuid.UUID]int)
	}
	p.calls[source.ID()]++
	return map[uuid.UUID]int{recipients[0].ID(): amount}
}

func TestRONBandsWithOtherRecords(t *testing.T) {
	for _, land := range otherBandLands {
		for _, side := range []string{"attack", "block"} {
			t.Run(land.name+"/"+side, func(t *testing.T) {
				a := &otherBandPlayer{BasePlayer: mage.NewBasePlayer("A")}
				b := &otherBandPlayer{BasePlayer: mage.NewBasePlayer("B")}
				g := mage.NewGame(a, b)
				controller := a.PlayerID()
				if side == "block" {
					controller = b.PlayerID()
				}
				card := repositoryCard(t, "LEG", land.name)
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				var source *mage.Permanent
				if side == "attack" {
					g.SetStep(core.PrecombatMain)
					card.SetOwner(controller)
					a.AddToHand(card)
					if err := g.PlayLand(controller, card.ID()); err != nil {
						t.Fatal(err)
					}
					if g.StackSize() != 0 || len(a.Hand()) != 0 {
						t.Fatal("land play used the stack or retained the card")
					}
					source = g.FindPermanent(card.ID())
				} else {
					source = g.PutOnBattlefield(card, controller)
				}
				members := make([]*mage.Permanent, 0, 3)
				for i := range 3 {
					opts := []mage.CardOption{mage.WithSuperTypes(core.SuperLegendary)}
					creature := mage.NewCreature(fmt.Sprintf("Member %d", i), "", 2, 20, opts...)
					creature.SetColorOverride([]core.Color{land.color})
					members = append(members, g.PutOnBattlefield(creature, controller))
				}
				opponent := b.PlayerID()
				if side == "block" {
					opponent = a.PlayerID()
				}
				enemy := g.PutOnBattlefield(mage.NewCreature("Enemy", "", 4, 20), opponent)
				for _, p := range append(members, enemy) {
					p.RevokeBaseAttr(core.AttrSummonSick)
				}
				if side == "attack" {
					ids := []uuid.UUID{members[0].ID(), members[1].ID(), members[2].ID()}
					g.ExecuteAttackers(a.PlayerID(), ids)
					if err := g.FormAttackingBand(ids); err != nil {
						t.Fatal(err)
					}
					g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: enemy.ID(), AttackerID: ids[0]}})
				} else {
					g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{enemy.ID()})
					g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: members[0].ID(), AttackerID: enemy.ID()}, {BlockerID: members[1].ID(), AttackerID: enemy.ID()}, {BlockerID: members[2].ID(), AttackerID: enemy.ID()}})
				}
				clone := g.Clone()
				g.AddPreventionShield(members[0].ID(), 1)
				g.ExecuteCombatDamage()
				for i, member := range members {
					members[i] = g.FindPermanent(member.ID())
				}
				enemy = g.FindPermanent(enemy.ID())
				if members[0].Damage != 3 || members[1].Damage != 0 || enemy.Damage != 6 {
					t.Fatalf("damage=%d/%d enemy=%d", members[0].Damage, members[1].Damage, enemy.Damage)
				}
				if (side == "attack" && (a.calls[enemy.ID()] != 1 || b.calls[enemy.ID()] != 0)) || (side == "block" && (b.calls[enemy.ID()] != 1 || a.calls[enemy.ID()] != 0)) {
					t.Fatal("wrong damage chooser")
				}
				if clone.FindPermanent(members[0].ID()).Damage != 0 {
					t.Fatal("clone changed")
				}
				g.BouncePermanentToHand(source)
				clone.ExecuteCombatDamage()
				if clone.FindPermanent(members[0].ID()).Damage != 4 {
					t.Fatal("clone lost its grant")
				}
			})
		}
	}
}

func TestRONBandsWithOtherInvalid(t *testing.T) {
	for _, body := range []string{
		`BandsWithOthers`, `BandsWithOthers()`, `BandsWithOthers(IsCardtype(Creature),IsColor(Red))`,
		`BandsWithOthers(ControlledByAPlayer(SinglePlayer(You)))`, `BandsWithOthers(SinglePermanent(ThisPermanent))`,
		`BandsWithOthers(And([]))`, `BandsWithOthers(IsNamed(NamedCard("")))`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Land],Subtypes:[]),Rules:[EachPermanentLayerEffect(IsCardtype(Creature),[AddAbility([` + body + `])])])`
		if registry, err := LoadFS(fstest.MapFS{"bad.ron": {Data: []byte(src)}}, "*.ron"); err == nil || registry != nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRONBandsWithOtherLifecycle(t *testing.T) {
	for _, land := range otherBandLands {
		for _, change := range []string{"other-color-member", "nonlegendary", "source-phase", "source-abilities", "source-control", "recipient-banding-loss", "recipient-abilities", "returned-member"} {
			t.Run(land.name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				source := g.PutOnBattlefield(repositoryCard(t, "LEG", land.name), a.PlayerID())
				firstCard := mage.NewCreature("Holder", "", 1, 20, mage.WithSuperTypes(core.SuperLegendary))
				firstCard.SetColorOverride([]core.Color{land.color})
				first := g.PutOnBattlefield(firstCard, a.PlayerID())
				second := g.PutOnBattlefield(mage.NewCreature("Partner", "", 1, 20, mage.WithSuperTypes(core.SuperLegendary)), a.PlayerID())
				first.RevokeBaseAttr(core.AttrSummonSick)
				second.RevokeBaseAttr(core.AttrSummonSick)
				ids := []uuid.UUID{first.ID(), second.ID()}
				g.ExecuteAttackers(a.PlayerID(), ids)
				switch change {
				case "other-color-member":
				case "nonlegendary":
					g.BouncePermanentToHand(second)
					second = g.PutOnBattlefield(mage.NewCreature("Nonlegendary", "", 1, 20), a.PlayerID())
					second.RevokeBaseAttr(core.AttrSummonSick)
					g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{second.ID()})
					ids[1] = second.ID()
				case "source-phase":
					g.PhaseOut(source.ID())
				case "source-abilities":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), source.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				case "source-control":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ChangeController(mage.This, mage.You)), source.ID(), b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				case "recipient-banding-loss":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveKeyword(mage.This, core.Banding)), first.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				case "recipient-abilities":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), first.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				case "returned-member":
					g.BouncePermanentToHand(first)
					a.RemoveFromHand(first.Card.ID())
					g.PutOnBattlefield(first.Card, a.PlayerID())
				}
				g.ApplyContinuousEffects()
				err := g.FormAttackingBand(ids)
				if (change == "other-color-member") != (err == nil) {
					t.Fatalf("formation after %s: %v", change, err)
				}
				if change == "recipient-banding-loss" || change == "recipient-abilities" || change == "source-abilities" || change == "source-control" {
					g.GetEffects().RemoveEndOfTurn()
					g.ApplyContinuousEffects()
					if err := g.FormAttackingBand(ids); err != nil {
						t.Fatalf("cleanup did not restore formation: %v", err)
					}
				}
				if g.GetPlayer(b.PlayerID()).Life() != 20 {
					t.Fatal("query changed life")
				}
			})
		}
	}
}

func TestRONBandsWithOtherStrictLoad(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var records []string
	for line := range strings.SplitSeq(string(data), "\n") {
		for _, land := range otherBandLands {
			if strings.Contains(line, `Name: "`+land.name+`"`) {
				records = append(records, line)
			}
		}
	}
	if len(records) != 5 {
		t.Fatal("missing complete land records")
	}
	collection := strings.Join(records, "\n")
	files := fstest.MapFS{"a.ron": {Data: []byte(collection)}, "b.ron": {Data: []byte(collection)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, land := range otherBandLands {
		first, err := registry.CreateCard(land.name)
		if err != nil {
			t.Fatal(err)
		}
		second, err := registry.CreateCard(land.name)
		if err != nil {
			t.Fatal(err)
		}
		if first == nil || second == nil || first.ID() == second.ID() || first.Abilities()[0].AbilityID() == second.Abilities()[0].AbilityID() {
			t.Fatal("factory instances share identities")
		}
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Land],Subtypes:[]),Rules:[BandsWithOthers])`)}
	if partial, err := LoadFS(files, "*.ron"); err == nil || partial != nil {
		t.Fatal("invalid collection published a partial registry")
	}
}
