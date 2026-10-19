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

type ronAttackTappingPayer struct {
	*mage.BasePlayer
	pay     bool
	choices int
}

func (p *ronAttackTappingPayer) ChooseMayAbility(string) bool { p.choices++; return p.pay }

func TestRONAttackTappingPaymentJohan(t *testing.T) {
	for _, state := range []string{"paid", "declined", "tapped-before", "left-before", "return-before", "phase-before", "control-before", "copy", "clone", "countered"} {
		t.Run(state, func(t *testing.T) {
			a := &ronAttackTappingPayer{BasePlayer: mage.NewBasePlayer("A"), pay: state != "declined"}
			b := &ronAttackTappingPayer{BasePlayer: mage.NewBasePlayer("B"), pay: true}
			g := mage.NewGame(a, b)
			g.SetActivePlayerIndex(0)
			g.SetTurn(2)
			g.SetStep(core.PrecombatMain)
			c := repositoryCard(t, "LEG", "Johan")
			addRONHandCard(a.BasePlayer, c)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err == nil {
				t.Fatal("unpaid cast")
			}
			a.ManaPool().Add(core.Colorless, 3)
			a.ManaPool().Add(core.Red, 1)
			a.ManaPool().Add(core.Green, 1)
			a.ManaPool().Add(core.White, 1)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			source := g.FindPermanent(c.ID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			if source.CurrentPower(g) != 5 || source.CurrentToughness(g) != 4 {
				t.Fatal("metadata")
			}
			attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 2, 4), a.PlayerID())
			attacker.RevokeBaseAttr(core.AttrSummonSick)
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 4), b.PlayerID())
			other.RevokeBaseAttr(core.AttrSummonSick)
			g.FireEvent(core.GameEvent{Type: core.EvtBeginCombat, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("opponent combat")
			}
			g.SetStep(core.BeginCombat)
			g.FireEvent(core.GameEvent{Type: core.EvtBeginCombat, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || a.choices != 0 || !source.CanDeclareAsAttacker(g) {
				t.Fatal("trigger timing")
			}
			switch state {
			case "tapped-before":
				g.TapPermanent(source)
			case "left-before", "return-before":
				g.RemoveFromBattlefield(source)
				if state == "return-before" {
					source = g.PutOnBattlefield(c, a.PlayerID())
					source.RevokeBaseAttr(core.AttrSummonSick)
				}
			case "phase-before":
				g.PhaseOut(source.ID())
			case "control-before":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.EndOfTurn})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy")
				}
			case "countered":
				g.CounterStackObject(g.StackPeek().ID)
			}
			clone := g.Clone()
			g.ResolveStack()
			if state == "clone" {
				clone.ResolveStack()
				if !clone.FindPermanent(source.ID()).CanDeclareAsAttacker(clone) {
					t.Fatal("clone choice changed")
				}
			}
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{attacker.ID()})
			wantTap := state == "declined" || state == "tapped-before" || state == "left-before" || state == "return-before" || state == "phase-before" || state == "countered"
			if !g.GetCombat().IsAttacking(attacker.ID()) || g.FindPermanent(attacker.ID()).Tapped != wantTap {
				t.Fatal("wrong tapping", g.FindPermanent(attacker.ID()).Tapped, wantTap)
			}
			g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{other.ID()})
			if g.FindPermanent(other.ID()).Tapped != (state != "copy") {
				t.Fatal("player scope")
			}
			if state == "paid" || state == "control-before" || state == "copy" || state == "clone" {
				if g.FindPermanent(source.ID()).CanDeclareAsAttacker(g) {
					t.Fatal("source can attack")
				}
			}
			if state == "return-before" && !source.CanDeclareAsAttacker(g) {
				t.Fatal("returned source inherited restriction")
			}
			g.GetEffects().RemoveEndOfCombat()
			g.ApplyContinuousEffects()
			if state == "paid" && !g.FindPermanent(source.ID()).CanDeclareAsAttacker(g) {
				t.Fatal("restriction duration")
			}
		})
	}
}

func TestRONAttackTappingPaymentDeclarations(t *testing.T) {
	valid := `MayCost(CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbility([PermanentRuleEffect(ThisPermanent,[CantAttack])])],UntilEndOfCombat)),If(CostWasPaid,[CreatePlayerEffectUntil(You,[AttackingDoesntCauseCreaturesToTapIf(IsCardtype(Creature),PermanentPassesFilter(ThisPermanent,IsUntapped))],UntilEndOfCombat)])`
	for _, body := range []string{
		`MayCost(CreatePermanentLayerEffectUntil(ThisPermanent,[],UntilEndOfCombat)),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MayCost(CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbility([PermanentRuleEffect(ThisPermanent,[CantAttack()])])],UntilEndOfCombat)),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MayCost(CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbility([PermanentRuleEffect(Ref_TargetPermanent,[CantAttack])])],UntilEndOfCombat)),If(CostWasPaid,[GainLife(Integer(1))])`,
		`CreatePlayerEffectUntil(You,[AttackingDoesntCauseCreaturesToTapIf(IsCardtype(Creature),PermanentPassesFilter(ThisPermanent,IsUntapped))],UntilEndOfCombat())`,
		`CreatePlayerEffectUntil(You,[AttackingDoesntCauseCreaturesToTapIf(IsCardtype(Creature),PermanentPassesFilter(Ref_TargetPermanent,IsUntapped))],UntilEndOfCombat)`,
		`If(CostWasPaid,[GainLife(Integer(1))])`,
	} {
		if _, err := compileAttackTappingSource(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[Activated(PayMana([]),ActionList([` + body + `]))])`); err == nil {
			t.Fatal("accepted", body)
		}
	}
	_ = permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([`+valid+`]))`)
}

func compileAttackTappingSource(src string) (mage.CardFactory, error) {
	nodes, err := Parse(strings.NewReader(src), "test.ron")
	if err != nil {
		return nil, err
	}
	return Compile(nodes[0])
}

func TestRONAttackTappingPaymentLoadInventory(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Johan"`) {
			record = line
		}
	}
	files := fstest.MapFS{"good.ron": {Data: []byte(record)}, "bad.ron": {Data: []byte(`Card(Name:"Bad",Rules:[UnknownRule])`)}}
	if registry, err := LoadFS(files, "*.ron"); err == nil || registry != nil {
		t.Fatal("partial publication")
	}
	delete(files, "bad.ron")
	registry, err := LoadFS(files, "*.ron")
	if err != nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("strict load", err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	roles := map[string]string{"CreatePermanentLayerEffectUntil": "resolution_cost", "AttackingDoesntCauseCreaturesToTapIf": "temporary_player_operation", "PermanentPassesFilter": "static_condition", "CantAttack": "permanent_rule_operation"}
	for name, role := range roles {
		found := false
		for _, sig := range report.Signatures {
			if sig.Name == name && sig.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing role %s/%s", name, role)
		}
	}
}

func TestRONAttackTappingPaymentForms(t *testing.T) {
	grant := `CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbility([PermanentRuleEffect(ThisPermanent,[CantAttack])])],UntilEndOfCombat)`
	for _, body := range []string{
		grant,
		`PlayerMayCost(You,` + grant + `),IfElse(CostWasPaid,[GainLife(Integer(1))],[GainLife(Integer(2))])`,
		`MayCost(` + grant + `),Unless(CostWasntPaid,[GainLife(Integer(1))])`,
	} {
		a := &ronAttackTappingPayer{BasePlayer: mage.NewBasePlayer("A"), pay: true}
		g := mage.NewGame(a, mage.NewBasePlayer("B"))
		factory, err := compileAttackTappingSource(`Card(Name:"Forms",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:4),Rules:[Activated(PayMana([]),ActionList([` + body + `]))])`)
		if err != nil {
			t.Fatal(err)
		}
		first, second := factory(), factory()
		if first.Abilities()[0].AbilityID() == second.Abilities()[0].AbilityID() {
			t.Fatal("shared factory ability")
		}
		p := g.PutOnBattlefield(first, a.PlayerID())
		p.RevokeBaseAttr(core.AttrSummonSick)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if g.FindPermanent(p.ID()).CanDeclareAsAttacker(g) {
			t.Fatal("grant did not restrict attacks")
		}
		if body != grant && a.Life() != 21 {
			t.Fatal("payment branch")
		}
	}
}
