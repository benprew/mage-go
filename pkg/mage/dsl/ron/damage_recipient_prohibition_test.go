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

const damageProhibitionRON = "DamageDealtToItCantBePreventedOrRedirected"

func TestRONDamageRecipientProhibitionCompleteDeathExile(t *testing.T) {
	record := staticBaseRecord(t, "DRK", "Whippoorwill")
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition("Whippoorwill", factory()); err != nil {
		t.Fatal(err)
	}
	for _, ref := range []string{"Ref_TargetPermanent", "Trigger_ThatPermanent"} {
		src := `Card(Name:"Watch",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateTriggerUntil(WhenAPermanentDies(SinglePermanent(Ref_TargetPermanent)),ActionList([Exile([Permanent(` + ref + `)],[])]),UntilEndOfTurn)])))])`
		if registry, err := LoadFS(fstest.MapFS{"watch.ron": {Data: []byte(src)}}, "*.ron"); err != nil || registry == nil {
			t.Fatalf("death exile %s: %v", ref, err)
		}
	}
}

func whippoorwillDamageRule(t *testing.T) mage.CardFactory {
	t.Helper()
	record := staticBaseRecord(t, "DRK", "Whippoorwill")
	rules := record.Fields["Rules"]
	body := rules.Args[0].Args[1].Args[1]
	actions := body.Args[0]
	actions.Args = actions.Args[:1]
	body.Args[0] = actions
	rules.Args[0].Args[1].Args[1] = body
	record.Fields["Rules"] = rules
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	return factory
}

func TestRONDamageRecipientProhibitionGameplay(t *testing.T) {
	for _, change := range []string{"live", "abilities", "source-left", "control", "phase-cycle", "phase-before", "return-before", "return-after", "late-shroud", "copy", "clone", "cleanup"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronModalGame(0)
			g.SetActivePlayerIndex(0)
			g.SetStep(core.PrecombatMain)
			factory := whippoorwillDamageRule(t)
			card := factory()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid cast succeeded")
			}
			a.ManaPool().Add(core.Green, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			source := g.FindPermanent(card.ID())
			targetCard := mage.NewCreature("Recipient", "", 1, 8)
			targetCard.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(targetCard, b.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 8), b.PlayerID())
			a.ManaPool().Add(core.Green, 2)
			activate := func() error { return g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}) }
			if err := activate(); err == nil || g.FindPermanent(source.ID()).Tapped || a.ManaPool().Count(core.Green) != 2 {
				t.Fatal("sick activation spent resources")
			}
			source.RevokeBaseAttr(core.AttrSummonSick)
			if err := activate(); err != nil {
				t.Fatal(err)
			}
			if target.HasAttr(core.AttrDamageCantBePreventedOrRedirected) || a.ManaPool().Count(core.Green) != 0 || !g.FindPermanent(source.ID()).Tapped {
				t.Fatal("announcement timing")
			}
			switch change {
			case "source-left":
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
			case "phase-before":
				g.PhaseOut(target.ID())
			case "return-before":
				g.BouncePermanentToHand(target)
				b.RemoveFromHand(targetCard.ID())
				g.PutOnBattlefield(targetCard, b.PlayerID())
			case "late-shroud":
				g.MutablePermanent(target.ID()).GrantBaseAttr(core.Shroud)
			case "copy":
				a.targets = []uuid.UUID{other.ID()}
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), true) == nil {
					t.Fatal("copy failed")
				}
			case "clone":
				g = g.Clone()
			}
			g.ResolveStack()
			switch change {
			case "abilities":
				g.AddContinuousEffect(mage.RemoveAllAbilities(target.ID(), core.Indefinite))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: other.ID(), TargetID: target.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(target.ID()))
			case "phase-before":
				g.PhaseIn([]uuid.UUID{target.ID()})
			case "return-after":
				g.BouncePermanentToHand(g.FindPermanent(target.ID()))
				g.GetPlayer(b.PlayerID()).RemoveFromHand(targetCard.ID())
				g.PutOnBattlefield(targetCard, b.PlayerID())
			case "cleanup":
				g.RunStepWithPriority(core.Cleanup)
			}
			active := change != "phase-before" && change != "return-before" && change != "return-after" && change != "late-shroud" && change != "cleanup"
			for _, id := range []uuid.UUID{target.ID(), other.ID()} {
				p := g.FindPermanent(id)
				want := active && id == target.ID() || change == "copy" && id == other.ID()
				if p.HasAttr(core.AttrDamageCantBePreventedOrRedirected) != want || p.HasKeyword(core.CantRegenerate) != want {
					t.Fatal("recipient identity or expiry")
				}
				g.AddPreventionShield(id, 3)
				g.DealDamageToPermanent(p, 2, other.ID())
				damage := 0
				if want {
					damage = 2
				}
				if g.FindPermanent(id).Damage != damage {
					t.Fatal("prevention bypass")
				}
			}
		})
	}
}

func TestRONDamageRecipientProhibitionInvalid(t *testing.T) {
	for _, ops := range []string{damageProhibitionRON + `()`, damageProhibitionRON + `(You)`, damageProhibitionRON + `,Flying`} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreatePermanentRuleEffectUntil(Ref_TargetPermanent,[` + ops + `],UntilEndOfTurn)])))])`
		if registry, err := LoadFS(fstest.MapFS{"bad.ron": {Data: []byte(src)}}, "*.ron"); err == nil || registry != nil {
			t.Fatal("invalid operation accepted", ops)
		}
	}
	for _, action := range []string{
		`CreatePermanentRuleEffectUntil(Ref_TargetPermanent,[` + damageProhibitionRON + `],UntilEndOfTurn)`,
		`CreatePermanentRuleEffectUntil(You,[` + damageProhibitionRON + `],UntilEndOfTurn)`,
		`CreatePermanentRuleEffectUntil(ThisPermanent,[` + damageProhibitionRON + `],UntilEndOfCombat)`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([` + action + `]))])`
		if registry, err := LoadFS(fstest.MapFS{"bad.ron": {Data: []byte(src)}}, "*.ron"); err == nil || registry != nil {
			t.Fatal("invalid declaration accepted", action)
		}
	}
}

func TestRONDamageRecipientProhibitionInventory(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/DRK.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Whippoorwill"`) {
			record = line
		}
	}
	report, err := InspectFS(fstest.MapFS{"card.ron": {Data: []byte(record)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, signature := range report.Signatures {
		if signature.Name == damageProhibitionRON && signature.Role == "permanent_rule_operation" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing operation role")
	}
}

func TestRONDamageRecipientProhibitionContexts(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`CreatePermanentRuleEffectUntil(ThisPermanent,[`+damageProhibitionRON+`],UntilEndOfTurn)`), "context.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, ctx := range []scope{{permanent: true}, {permanent: true, resolving: true, enteringSource: true}, {permanent: true, resolving: true, drawReplacementBody: true}} {
		if _, err := compileEffect(nodes[0], ctx); err == nil {
			t.Fatal("forbidden context accepted")
		}
	}
	if _, err := compileEffect(nodes[0], scope{permanent: true, resolving: true}); err != nil {
		t.Fatal(err)
	}
}

func TestRONDamageRecipientProhibitionLegality(t *testing.T) {
	for _, reason := range []string{"unpaid", "shroud", "phased", "artifact"} {
		t.Run(reason, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(whippoorwillDamageRule(t)(), a.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			target := g.PutOnBattlefield(mage.NewCreature("Recipient", "", 1, 8), b.PlayerID())
			if reason != "unpaid" {
				a.ManaPool().Add(core.Green, 2)
			}
			switch reason {
			case "shroud":
				target.GrantBaseAttr(core.Shroud)
			case "phased":
				g.PhaseOut(target.ID())
			case "artifact":
				target = g.PutOnBattlefield(mage.NewArtifact("Artifact", ""), b.PlayerID())
			}
			mana := a.ManaPool().Count(core.Green)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err == nil {
				t.Fatal("invalid activation accepted")
			}
			if g.StackSize() != 0 || g.FindPermanent(source.ID()).Tapped || a.ManaPool().Count(core.Green) != mana {
				t.Fatal("failed activation spent resources")
			}
		})
	}
}
