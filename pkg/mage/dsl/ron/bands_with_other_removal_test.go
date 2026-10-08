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

func TestRONBandsWithOtherRemovalRecords(t *testing.T) {
	for _, name := range []string{"Shelkin Brownie", "Tolaria"} {
		for _, change := range []string{"none", "returned", "phased", "phase-in", "installed-return", "installed-phase", "control", "shroud"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, "LEG", name)
				if err := mage.ValidateCardDefinition(name, card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				index := 0
				if name == "Tolaria" {
					index = 1
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					if a.ManaPool().Count(core.Blue) != 1 || !g.Stack().IsEmpty() {
						t.Fatal("mana ability did not resolve immediately")
					}
					g.FindPermanent(source.ID()).Tapped = false
				}
				g.PutOnBattlefield(repositoryCard(t, "LEG", "Adventurers' Guildhouse"), b.PlayerID())
				fixture := mage.NewCreature("Holder", "", 1, 20, mage.WithSuperTypes(core.SuperLegendary), mage.WithKeyword(core.Banding), mage.WithKeyword(core.Flying))
				fixture.SetColorOverride([]core.Color{core.Green})
				fixture.SetOwner(b.PlayerID())
				target := g.PutOnBattlefield(fixture, b.PlayerID())
				partners := make([]uuid.UUID, 0, 2)
				for i := range 2 {
					p := g.PutOnBattlefield(mage.NewCreature(fmt.Sprintf("Partner %d", i), "", 1, 20, mage.WithSuperTypes(core.SuperLegendary)), b.PlayerID())
					g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
					partners = append(partners, p.ID())
				}
				otherCard := mage.NewCreature("Other", "", 1, 20, mage.WithSuperTypes(core.SuperLegendary))
				otherCard.SetColorOverride([]core.Color{core.Green})
				other := g.PutOnBattlefield(otherCard, b.PlayerID())
				if name == "Shelkin Brownie" {
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), index, []uuid.UUID{target.ID()}); err == nil {
						t.Fatal("tap symbol ignored summoning sickness")
					}
					g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrSummonSick)
				} else {
					g.SetStep(core.PrecombatMain)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), index, []uuid.UUID{target.ID()}); err == nil {
						t.Fatal("upkeep restriction ignored")
					}
					if g.FindPermanent(source.ID()).Tapped || !g.Stack().IsEmpty() {
						t.Fatal("failed timing paid cost")
					}
					g.SetStep(core.Upkeep)
					g.SetActivePlayerIndex(1)
				}
				land := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), index, []uuid.UUID{land.ID()}); err == nil {
					t.Fatal("land target accepted")
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), index, []uuid.UUID{target.ID()}); err != nil {
					t.Fatal(err)
				}
				if !g.FindPermanent(source.ID()).Tapped || !g.FindPermanent(target.ID()).HasKeyword(core.Banding) {
					t.Fatal("payment or resolution timing")
				}
				clone := g.Clone()
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.SetStep(core.PrecombatMain)
				current := target.ID()
				returnTarget := func() {
					g.BouncePermanentToHand(g.FindPermanent(current))
					b.RemoveFromHand(fixture.ID())
					current = g.PutOnBattlefield(fixture, b.PlayerID()).ID()
				}
				switch change {
				case "returned":
					returnTarget()
				case "phased":
					g.PhaseOut(current)
				case "phase-in":
					g.PhaseIn(g.PhaseOut(current))
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: current, TargetID: current, ControllerID: a.PlayerID(), Duration: core.Indefinite})
				case "shroud":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.GrantKeywordModification(mage.This, core.Shroud)), current, b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				}
				g.ResolveStack()
				if change == "installed-return" {
					returnTarget()
				}
				if change == "installed-phase" {
					g.PhaseIn(g.PhaseOut(current))
				}
				g.PhaseIn([]uuid.UUID{current})
				removed := change == "none" || change == "phase-in" || change == "installed-phase" || change == "control"
				if g.FindPermanent(current).HasKeyword(core.Banding) != (name == "Shelkin Brownie" || !removed) || !g.FindPermanent(current).HasKeyword(core.Flying) {
					t.Fatal("wrong ordinary keyword removal")
				}
				clone.ResolveStack()
				checkQuality := func(game *mage.Game, id uuid.UUID, want bool) {
					t.Helper()
					probe := game.Clone()
					probe.MutablePermanent(id).RevokeBaseAttr(core.Banding)
					probe.MutablePermanent(id).RevokeBaseAttr(core.AttrSummonSick)
					ids := append([]uuid.UUID{id}, partners...)
					probe.ExecuteAttackers(b.PlayerID(), ids)
					if (probe.FormAttackingBand(ids) == nil) != want {
						t.Fatalf("quality formation = %v, want %v", !want, want)
					}
				}
				if change != "control" {
					checkQuality(g, current, !removed)
				}
				checkQuality(clone, target.ID(), false)
				checkQuality(g, other.ID(), true)
				g.GetEffects().RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				if !g.FindPermanent(current).HasKeyword(core.Banding) {
					t.Fatal("ordinary banding did not return")
				}
				if change != "control" {
					g.MutablePermanent(current).RevokeBaseAttr(core.AttrSummonSick)
					ids := append([]uuid.UUID{current}, partners...)
					g.ExecuteAttackers(b.PlayerID(), ids)
					if err := g.FormAttackingBand(ids); err != nil {
						t.Fatal("cleanup did not restore quality", err)
					}
				}
			})
		}
	}
}

func TestRONBandsWithOtherRemovalInvalid(t *testing.T) {
	for _, body := range []string{
		`Activated(PayMana([]),ActionList([CreatePermanentLayerEffectUntil(ThisPermanent,[LosesAbility(AnyBandsWithOthers())],UntilEndOfTurn)]))`,
		`Activated(PayMana([]),ActionList([CreatePermanentLayerEffectUntil(ThisPermanent,[LosesAbility(AnyBandsWithOthers(Flying))],UntilEndOfTurn)]))`,
		`Activated(PayMana([]),ActionList([CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbility([AnyBandsWithOthers])],UntilEndOfTurn)]))`,
		`Activated(PayMana([]),ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent,[LosesAbility(AnyBandsWithOthers)],UntilEndOfTurn)]))`,
		`EachPermanentLayerEffect(IsCardtype(Creature),[LosesAbility(AnyBandsWithOthers)])`,
	} {
		nodes, err := Parse(strings.NewReader(fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[%s])`, body)), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("invalid accepted: %s", body)
		}
	}
}

func TestRONBandsWithOtherRemovalStrictLoad(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var records []string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Shelkin Brownie"`) || strings.Contains(line, `Name: "Tolaria"`) {
			records = append(records, line)
		}
	}
	if len(records) != 2 {
		t.Fatal("missing records")
	}
	files := fstest.MapFS{"removal.ron": {Data: []byte(strings.Join(records, "\n"))}, "reprints.ron": {Data: []byte(strings.Join(records, "\n"))}}
	result, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.RegisteredCardNames()) != 2 {
		t.Fatal("missing complete records")
	}
	for _, name := range []string{"Shelkin Brownie", "Tolaria"} {
		first, err := result.CreateCard(name)
		if err != nil {
			t.Fatal(err)
		}
		second, err := result.CreateCard(name)
		if err != nil {
			t.Fatal(err)
		}
		index := 0
		if name == "Tolaria" {
			index = 1
		}
		if first.ID() == second.ID() || first.Abilities()[index].AbilityID() == second.Abilities()[index].AbilityID() {
			t.Fatal("factory shared identities")
		}
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreatePermanentLayerEffectUntil(ThisPermanent,[LosesAbility(AnyBandsWithOthers())],UntilEndOfTurn)]))])`)}
	if partial, err := LoadFS(files, "*.ron"); err == nil || partial != nil {
		t.Fatal("invalid collection published partial registry")
	}
}

func TestRONBandsWithOtherRemovalCombat(t *testing.T) {
	for _, name := range []string{"Shelkin Brownie", "Tolaria"} {
		for _, remove := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%v", name, remove), func(t *testing.T) {
				a := &otherBandPlayer{BasePlayer: mage.NewBasePlayer("A")}
				b := &otherBandPlayer{BasePlayer: mage.NewBasePlayer("B")}
				g := mage.NewGame(a, b)
				g.PutOnBattlefield(repositoryCard(t, "LEG", "Adventurers' Guildhouse"), a.PlayerID())
				holderCard := mage.NewCreature("Holder", "", 1, 20, mage.WithSuperTypes(core.SuperLegendary))
				holderCard.SetColorOverride([]core.Color{core.Green})
				holder := g.PutOnBattlefield(holderCard, a.PlayerID())
				partner := g.PutOnBattlefield(mage.NewCreature("Partner", "", 1, 20, mage.WithSuperTypes(core.SuperLegendary)), a.PlayerID())
				enemy := g.PutOnBattlefield(mage.NewCreature("Enemy", "", 4, 20), b.PlayerID())
				for _, p := range []*mage.Permanent{holder, partner} {
					g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
				}
				ids := []uuid.UUID{holder.ID(), partner.ID()}
				g.ExecuteAttackers(a.PlayerID(), ids)
				if err := g.FormAttackingBand(ids); err != nil {
					t.Fatal(err)
				}
				source := g.PutOnBattlefield(repositoryCard(t, "LEG", name), b.PlayerID())
				g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrSummonSick)
				if remove {
					index := 0
					if name == "Tolaria" {
						index = 1
						g.SetStep(core.Upkeep)
					}
					if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), index, []uuid.UUID{holder.ID()}); err != nil {
						t.Fatal(err)
					}
					g.ResolveStack()
					if !g.GetCombat().IsBandedWith(ids[0], ids[1]) {
						t.Fatal("removal erased formed band")
					}
				}
				g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: enemy.ID(), AttackerID: holder.ID()}})
				g.ExecuteCombatDamage()
				if (a.calls[enemy.ID()] > 0) == remove {
					t.Fatal("damage assignment used obsolete quality permission")
				}
				if g.FindPermanent(holder.ID()).Damage+g.FindPermanent(partner.ID()).Damage != 4 || g.FindPermanent(enemy.ID()).Damage != 2 {
					t.Fatal("band damage totals changed")
				}
			})
		}
	}
}
