package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func singularAttackRuleSource(body string) string {
	return `Card(Name:"Attack rule",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),ManaCost:[ManaCostU],Rules:[SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([` + body + `])))])`
}

func TestRONSingularAttackRuleGameplay(t *testing.T) {
	for _, change := range []string{"normal", "phase", "cycle", "return", "control", "copy", "pending-clone", "cleanup", "tax", "illegal", "payment"} {
		t.Run(change, func(t *testing.T) {
			body := `CreatePermanentRuleEffectUntil(Ref_TargetPermanent,[MustAttack],UntilEndOfTurn)`
			nodes, err := Parse(strings.NewReader(singularAttackRuleSource(body)), "attack.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			first, second := factory(), factory()
			if first.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == second.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
				t.Fatal("shared target declarations")
			}
			g, a, b := ronGame()
			options := []mage.CardOption{}
			if change == "tax" {
				options = append(options, mage.WithStaticAbility(mage.SourceCantAttackUnlessPays(mage.GenericCost(1))))
			}
			card := mage.NewCreature("Subject", "", 2, 3, options...)
			card.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(card, b.PlayerID())
			p.RevokeBaseAttr(core.AttrSummonSick)
			peer := g.PutOnBattlefield(mage.NewCreature("Peer", "", 2, 3), b.PlayerID())
			peer.RevokeBaseAttr(core.AttrSummonSick)
			first.SetOwner(a.PlayerID())
			a.AddToHand(first)
			if change != "payment" {
				a.ManaPool().Add(core.Blue, 1)
			}
			err = g.CastSpellByID(a.PlayerID(), first.ID(), []uuid.UUID{p.ID()}, 0)
			if change == "payment" {
				if err == nil || g.StackSize() != 0 || len(a.Hand()) != 1 {
					t.Fatal("payment rollback")
				}
				g.SetActivePlayerIndex(1)
				g.ExecuteAttackers(b.PlayerID(), nil)
				if g.GetCombat().IsAttacking(p.ID()) {
					t.Fatal("unpaid cast installed rule")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			pending := g.Clone()
			if change == "illegal" {
				g.PhaseOut(p.ID())
			}
			g.ResolveStack()
			g.SetActivePlayerIndex(1)
			actor := b.PlayerID()
			want := change != "illegal"
			switch change {
			case "phase":
				g.PhaseOut(p.ID())
				want = false
			case "cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				p = g.PutOnBattlefield(card, b.PlayerID())
				p.RevokeBaseAttr(core.AttrSummonSick)
				want = false
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
				g.SetActivePlayerIndex(0)
				actor = a.PlayerID()
			case "cleanup":
				g.RunStepWithPriority(core.Cleanup)
				want = false
			case "tax":
				b.ManaPool().Add(core.Colorless, 1)
				want = false
			}
			g.ExecuteAttackers(actor, nil)
			if g.GetCombat().IsAttacking(p.ID()) != want || g.GetCombat().IsAttacking(peer.ID()) {
				t.Fatal("saved rule or membership")
			}
			if change == "tax" && b.ManaPool().TotalMana() != 1 {
				t.Fatal("forced tax")
			}
			if change == "pending-clone" {
				pending.ResolveStack()
				pending.SetActivePlayerIndex(1)
				pending.ExecuteAttackers(b.PlayerID(), nil)
				if !pending.GetCombat().IsAttacking(p.ID()) {
					t.Fatal("pending clone lost binding")
				}
			}
		})
	}
}

func TestRONSingularAttackRuleImpClauses(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, attacks := range []bool{false, true} {
			t.Run(set+map[bool]string{true: "/attack", false: "/no-attack"}[attacks], func(t *testing.T) {
				record := hostUntapRecord(t, set, "Nettling Imp")
				factory, err := Compile(record)
				if err != nil {
					t.Fatal(err)
				}
				g, a, b := ronGame()
				source := g.PutOnBattlefield(factory(), a.PlayerID())
				source.RevokeBaseAttr(core.AttrSummonSick)
				victim := g.PutOnBattlefield(mage.NewCreature("Subject", "", 2, 3), b.PlayerID())
				victim.RevokeBaseAttr(core.AttrSummonSick)
				g.SetTurn(2)
				g.SetActivePlayerIndex(1)
				if !attacks {
					g.TapPermanent(victim)
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
					t.Fatal(err)
				}
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.ResolveStack()
				clone := g.Clone()
				for _, game := range []*mage.Game{g, clone} {
					game.ExecuteAttackers(b.PlayerID(), nil)
					if game.GetCombat().IsAttacking(victim.ID()) != attacks {
						t.Fatal("attack requirement")
					}
					game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
					game.PutTriggersOnStack()
					game.ResolveStack()
					if (game.FindPermanent(victim.ID()) != nil) != attacks {
						t.Fatal("saved delayed target or attack history")
					}
				}
			})
		}
	}
}

func TestRONSingularAttackRuleInvalidAndNative(t *testing.T) {
	valid := singularAttackRuleSource(`CreatePermanentRuleEffectUntil(Ref_TargetPermanent,[MustAttack],UntilEndOfTurn)`)
	for _, expression := range []string{
		`CreatePermanentRuleEffectUntil(Ref_TargetPermanent,[MustAttack()],UntilEndOfTurn)`,
		`CreatePermanentRuleEffectUntil(Ref_TargetPermanent,[MustAttack(You)],UntilEndOfTurn)`,
		`CreatePermanentRuleEffectUntil(Ref_TargetPermanent,[MustAttack],Indefinite)`,
		`CreatePermanentRuleEffectUntil(Ref_TargetPermanent,[MustAttack,CantBlock],UntilEndOfTurn)`,
		`CreatePermanentRuleEffectUntil(Ref_TargetPermanent2,[MustAttack],UntilEndOfTurn)`,
		`CreatePermanentRuleEffectUntil(ActionPermanent,[MustAttack],UntilEndOfTurn)`,
	} {
		nodes, err := Parse(strings.NewReader(singularAttackRuleSource(expression)), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", expression)
		}
	}
	registry := mage.NewRegistry()
	if err := LoadInto(registry, fstest.MapFS{"attack.ron": {Data: []byte(valid)}}, "*.ron"); err != nil {
		t.Fatal(err)
	}
	invalid := strings.Replace(strings.Replace(valid, `Name:"Attack rule"`, `Name:"Invalid"`, 1), `MustAttack]`, `MustAttack()]`, 1)
	if err := LoadInto(registry, fstest.MapFS{"attack.ron": {Data: []byte(valid + invalid)}}, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("invalid collection published")
	}
}

func TestRONSingularAttackRuleSourceAndGroups(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`Card(Name:"Source",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:3),Rules:[Activated(PayMana([]),ActionList([CreatePermanentRuleEffectUntil(ThisPermanent,[MustAttack],UntilEndOfTurn)]))])`), "source.ron")
	if err != nil {
		t.Fatal(err)
	}
	sourceFactory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	g, a, _ := ronGame()
	source := g.PutOnBattlefield(sourceFactory(), a.PlayerID())
	source.RevokeBaseAttr(core.AttrSummonSick)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.ExecuteAttackers(a.PlayerID(), nil)
	if !g.GetCombat().IsAttacking(source.ID()) {
		t.Fatal("source rule")
	}
	body := `Targeted([TargetPermanent(IsCardtype(Creature)),TargetPermanent(IsCardtype(Creature))],ActionList([CreatePermanentRuleEffectUntil(Ref_TargetPermanent1,[MustAttack],UntilEndOfTurn),TapPermanent(Ref_TargetPermanent2)]))`
	factory := permanentScopeFactory(t, "Instant", `SpellActions(`+body+`)`)
	g, a, b := ronGame()
	p := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 3), a.PlayerID())
	q := g.PutOnBattlefield(mage.NewCreature("Second", "", 2, 3), a.PlayerID())
	p.RevokeBaseAttr(core.AttrSummonSick)
	q.RevokeBaseAttr(core.AttrSummonSick)
	for i, subject := range []*mage.Permanent{p, q} {
		card := factory()
		card.SetOwner(b.PlayerID())
		b.AddToHand(card)
		if err := g.CastSpellByID(b.PlayerID(), card.ID(), []uuid.UUID{subject.ID(), source.ID()}, 0); err == nil {
			t.Fatal("absent second group accepted")
		}
		peer := q
		if i == 1 {
			peer = p
		}
		if err := g.CastSpellByID(b.PlayerID(), card.ID(), []uuid.UUID{subject.ID(), peer.ID()}, 0); err != nil {
			t.Fatal(err)
		}
	}
	g.ResolveStack()
	g.UntapPermanent(g.FindPermanent(p.ID()))
	g.UntapPermanent(g.FindPermanent(q.ID()))
	g.ExecuteAttackers(a.PlayerID(), nil)
	if !g.GetCombat().IsAttacking(p.ID()) || !g.GetCombat().IsAttacking(q.ID()) {
		t.Fatal("independent target groups and factory instances")
	}
}
