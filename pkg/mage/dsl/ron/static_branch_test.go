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

func TestRONStaticBranchMob(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "DRK", "Angry Mob")
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(repositoryCard(t, "LEB", "Swamp"), a.PlayerID())
	land := g.PutOnBattlefield(repositoryCard(t, "LEB", "Swamp"), b.PlayerID())
	g.PutOnBattlefield(repositoryCard(t, "LEB", "Badlands"), b.PlayerID())
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.White, 1)
	a.ManaPool().Add(core.Colorless, 2)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || a.ManaPool().TotalMana() != 3 || g.StackSize() != 0 {
		t.Fatal("unpaid cast did not roll back")
	}
	a.ManaPool().Add(core.White, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(card.ID()) != nil {
		t.Fatal("entered before resolution")
	}
	g.ResolveStack()
	checkConditionalPT(t, g, card.ID(), 4, 4)
	if !g.FindPermanent(card.ID()).HasKeyword(core.Trample) {
		t.Fatal("missing trample")
	}
	clone := g.Clone()
	g.SetActivePlayerIndex(1)
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, card.ID(), 2, 2)
	checkConditionalPT(t, clone, card.ID(), 4, 4)
	g.SetActivePlayerIndex(0)
	g.PhaseOut(land.ID())
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, card.ID(), 3, 3)
	g.PhaseIn([]uuid.UUID{land.ID()})
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, card.ID(), 4, 4)
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: card.ID(), TargetID: card.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	checkConditionalPT(t, g, card.ID(), 2, 2)
	g.SetActivePlayerIndex(1)
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, card.ID(), 3, 3)
	g.MutablePermanent(card.ID()).AddCounter(core.P1P1, 1)
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn,
		mage.ModifyPT(mage.This, mage.Fixed(2), mage.Fixed(4)), mage.SwitchPowerToughness(mage.This)), card.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, card.ID(), 8, 6)
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, card.ID(), 4, 4)
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), card.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, card.ID(), 3, 3)
	if g.FindPermanent(card.ID()).HasKeyword(core.Trample) {
		t.Fatal("removed ability still grants trample")
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, card.ID(), 4, 4)
}

func TestRONStaticBranchLurker(t *testing.T) {
	for _, action := range []string{"attack", "block"} {
		t.Run(action, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "DRK", "Lurker")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			castRONSpell(t, g, a, card)
			g.ResolveStack()
			p := g.FindPermanent(card.ID())
			bolt := repositoryCard(t, "LEB", "Lightning Bolt")
			if p.CanBeTargetedBy(mage.SpellTargetSource(bolt), b.PlayerID(), g) {
				t.Fatal("spell can target before combat history")
			}
			if !p.CanBeTargetedBy(bolt, b.PlayerID(), g) {
				t.Fatal("ability cannot target")
			}
			bolt.SetOwner(b.PlayerID())
			b.AddToHand(bolt)
			b.ManaPool().Add(core.Red, 1)
			if err := g.CastSpellByID(b.PlayerID(), bolt.ID(), []uuid.UUID{p.ID()}, 0); err == nil || len(b.Hand()) != 1 || b.ManaPool().Count(core.Red) != 1 || g.StackSize() != 0 {
				t.Fatal("forbidden target did not roll back")
			}
			g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
			g.SetStep(core.DeclareAttackers)
			if action == "attack" {
				g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
			} else {
				g.SetActivePlayerIndex(1)
				attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 1, 5), b.PlayerID())
				g.MutablePermanent(attacker.ID()).RevokeBaseAttr(core.AttrSummonSick)
				g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{attacker.ID()})
				g.SetStep(core.DeclareBlockers)
				g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: p.ID(), AttackerID: attacker.ID()}})
			}
			if !g.FindPermanent(p.ID()).CanBeTargetedBy(mage.SpellTargetSource(bolt), b.PlayerID(), g) {
				t.Fatal("combat declaration did not open targeting")
			}
			g.PhaseIn(g.PhaseOut(p.ID()))
			if !g.FindPermanent(p.ID()).CanBeTargetedBy(mage.SpellTargetSource(bolt), b.PlayerID(), g) {
				t.Fatal("phase cycle lost turn history")
			}
			clone := g.Clone()
			if err := g.CastSpellByID(b.PlayerID(), bolt.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
				t.Fatal("spell copy failed")
			}
			g.BouncePermanentToHand(g.FindPermanent(p.ID()))
			a.RemoveFromHand(card.ID())
			returned := g.PutOnBattlefield(card, a.PlayerID())
			if returned.CanBeTargetedBy(mage.SpellTargetSource(bolt), b.PlayerID(), g) {
				t.Fatal("returned object inherited combat history")
			}
			g.ResolveStack()
			if g.FindPermanent(returned.ID()).Damage != 0 {
				t.Fatal("spell or copy followed returned target")
			}
			clone.RunStepWithPriority(core.Cleanup)
			if clone.FindPermanent(p.ID()).CanBeTargetedBy(mage.SpellTargetSource(bolt), b.PlayerID(), clone) {
				t.Fatal("targeting did not close at final cleanup")
			}
		})
	}
}

func TestRONStaticBranchInvalid(t *testing.T) {
	for _, rule := range []string{
		`IfElse(IsPlayersTurn(You), [], [])`,
		`IfElse(IsPlayersTurn(You), [Flying])`,
		`IfElse(IsPlayersTurn(You), [Flying], [Flying])`,
		`Unless(IsPlayersTurn(You,You), [PermanentLayerEffect(ThisPermanent,[AdjustPT(1,1)])])`,
		`Unless(IsPlayersTurn(Ref_TargetPlayer), [PermanentLayerEffect(ThisPermanent,[AdjustPT(1,1)])])`,
		`Unless(PermanentPassesFilter(HostPermanent,Or([AttackedThisTurn,BlockedThisTurn])), [PermanentRuleEffect(ThisPermanent,[CantBeTheTargetOfSpells(AnySpell)])])`,
		`Unless(IsPlayersTurn(You),[PermanentRuleEffect(ThisPermanent,[CantBeTheTargetOfSpells(AnySpell),CantBeTheTargetOfAbilities(AnyAbility)])])`,
		`Unless(PermanentPassesFilter(ThisPermanent,Or([AttackedThisTurn,AttackedThisTurn])),[PermanentRuleEffect(ThisPermanent,[CantBeTheTargetOfSpells(AnySpell)])])`,
		`Unless(PermanentPassesFilter(ThisPermanent,Or([AttackedThisTurn(),BlockedThisTurn])),[PermanentRuleEffect(ThisPermanent,[CantBeTheTargetOfSpells(AnySpell)])])`,
		`Unless(IsPlayersTurn(You),[PermanentRuleEffect(ThisPermanent,[CantBeTheTargetOfSpells(AnySpell())])])`,
		`IfElse(IsPlayersTurn(You),[PermanentLayerEffect(ThisPermanent,[SetPowerAndToughnessBoth(Integer(2))])],[PermanentLayerEffect(Ref_TargetPermanent,[SetPowerAndToughnessBoth(Integer(3))])])`,
		`IfElse(IsPlayersTurn(You),[PermanentLayerEffect(ThisPermanent,[SetPowerAndToughnessBoth(ValueX)])],[PermanentLayerEffect(ThisPermanent,[SetPowerAndToughnessBoth(Integer(3))])])`,
	} {
		assertConditionalRejected(t, rule)
	}
}

func TestRONStaticBranchStrictInventory(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/DRK.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	files := fstest.MapFS{}
	for line := range strings.SplitSeq(string(data), "\n") {
		for _, name := range []string{"Angry Mob", "Lurker"} {
			if strings.Contains(line, `Name: "`+name+`"`) {
				files[name+".ron"] = &fstest.MapFile{Data: []byte(line)}
			}
		}
	}
	if len(files) != 2 {
		t.Fatal("missing complete records")
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 2 || len(report.Issues) != 0 {
		t.Fatalf("inventory: %v", err)
	}
	for _, s := range report.Signatures {
		if s.Role == "unknown" {
			t.Fatalf("unknown descendant %s in %s", s.Name, s.Context)
		}
	}
	first, err := registry.CreateCard("Angry Mob")
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.CreateCard("Angry Mob")
	if err != nil || first.ID() == second.ID() || first.Abilities()[0] == second.Abilities()[0] {
		t.Fatal("factory shares instance state")
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(string(files["Angry Mob.ron"].Data), "IsPlayersTurn(You)", "IsPlayersTurn(Trigger_ThatPlayer)", 1))}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 2 {
		t.Fatal("invalid collection changed registry")
	}
}

func TestRONStaticBranchMobCombat(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "DRK", "Angry Mob")
	p := g.PutOnBattlefield(card, a.PlayerID())
	g.PutOnBattlefield(repositoryCard(t, "LEB", "Swamp"), b.PlayerID())
	g.PutOnBattlefield(repositoryCard(t, "LEB", "Swamp"), b.PlayerID())
	other := g.PutOnBattlefield(repositoryCard(t, "DRK", "Angry Mob"), b.PlayerID())
	checkConditionalPT(t, g, p.ID(), 4, 4)
	checkConditionalPT(t, g, other.ID(), 2, 2)
	g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
	g.SetStep(core.DeclareAttackers)
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
	g.SetStep(core.DeclareBlockers)
	g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: other.ID(), AttackerID: p.ID()}})
	g.SetStep(core.CombatDamage)
	g.ExecuteCombatDamage()
	if b.Life() != 18 || g.FindPermanent(other.ID()) != nil || g.FindPermanent(p.ID()) == nil {
		t.Fatal("combat did not use branch values and trample")
	}
}

func TestRONStaticBranchLurkerResolution(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "DRK", "Lurker")
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.ApplyContinuousEffects()
	spell := repositoryCard(t, "LEB", "Giant Growth")
	castRONSpell(t, g, a, spell, p.ID())
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	clone := g.Clone()
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	g.ResolveStack()
	checkConditionalPT(t, g, p.ID(), 2, 3)
	clone.ResolveStack()
	checkConditionalPT(t, clone, p.ID(), 8, 9)
	abilityCard := repositoryCard(t, "LEB", "Prodigal Sorcerer")
	abilityCard.SetOwner(b.PlayerID())
	source := g.PutOnBattlefield(abilityCard, b.PlayerID())
	g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrSummonSick)
	if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, []uuid.UUID{p.ID()}); err != nil {
		t.Fatal(err)
	}
	g.DestroyPermanent(g.FindPermanent(source.ID()))
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("ability copy failed")
	}
	g.ResolveStack()
	if g.FindPermanent(p.ID()).Damage != 2 {
		t.Fatal("departed ability source or copy was treated as a spell")
	}
}
