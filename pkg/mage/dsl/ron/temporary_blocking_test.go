package ron

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTemporaryBlockingRealCards(t *testing.T) {
	for _, entry := range []struct{ set, name string }{
		{"LEA", "Dwarven Warriors"}, {"LEB", "Dwarven Warriors"}, {"2ED", "Dwarven Warriors"},
		{"3ED", "Dwarven Warriors"}, {"CED", "Dwarven Warriors"}, {"CEI", "Dwarven Warriors"}, {"ATQ", "Tawnos's Wand"},
	} {
		t.Run(entry.set+"/"+entry.name, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, entry.set, entry.name)
			if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(c, a.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			target := g.PutOnBattlefield(repositoryCard(t, "LEB", "Grizzly Bears"), b.PlayerID())
			blocker := g.PutOnBattlefield(repositoryCard(t, "LEB", "Grizzly Bears"), a.PlayerID())
			a.ManaPool().Add(core.Colorless, 2)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			if !g.FindPermanent(source.ID()).Tapped || g.StackSize() != 1 || !mage.CanBlock(blocker, target, g) {
				t.Fatal("announcement timing or tap cost")
			}
			if entry.name == "Tawnos's Wand" && a.ManaPool().TotalMana() != 0 {
				t.Fatal("mana cost not paid")
			}
			clone := g.Clone()
			clone.ResolveStack()
			if mage.CanBlock(clone.FindPermanent(blocker.ID()), clone.FindPermanent(target.ID()), clone) || !mage.CanBlock(blocker, target, g) {
				t.Fatal("clone resolution or isolation")
			}
			g.ResolveStack()
			if mage.CanBlock(g.FindPermanent(blocker.ID()), g.FindPermanent(target.ID()), g) {
				t.Fatal("target can still be blocked")
			}
			g.RunStepWithPriority(core.Cleanup)
			if !mage.CanBlock(g.FindPermanent(blocker.ID()), g.FindPermanent(target.ID()), g) {
				t.Fatal("restriction did not expire")
			}
		})
	}
}

func TestRONTemporaryBlockingLegality(t *testing.T) {
	for _, change := range []string{"power-before", "power-after", "return-before", "return-after", "phase-before", "phase-cycle", "source-left", "remove-abilities", "control"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Tawnos's Wand"), a.PlayerID())
			card := repositoryCard(t, "LEB", "Grizzly Bears")
			card.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(card, b.PlayerID())
			blocker := g.PutOnBattlefield(repositoryCard(t, "LEB", "Grizzly Bears"), a.PlayerID())
			a.ManaPool().Add(core.Colorless, 2)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			changeTarget := func() {
				switch change {
				case "power-before", "power-after":
					g.AddContinuousEffect(mage.TemporaryBoost(target.ID(), 1, 1))
				case "return-before", "return-after":
					g.BouncePermanentToHand(g.FindPermanent(target.ID()))
					b.RemoveFromHand(card.ID())
					target = g.PutOnBattlefield(card, b.PlayerID())
				case "phase-before":
					g.PhaseOut(target.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(target.ID()))
				case "source-left":
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				case "remove-abilities":
					g.AddContinuousEffect(mage.RemoveAllAbilities(target.ID(), core.EndOfTurn))
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: target.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				}
			}
			after := change == "power-after" || change == "return-after" || change == "remove-abilities"
			if !after {
				changeTarget()
			}
			g.ResolveStack()
			if after {
				changeTarget()
			}
			if change == "phase-before" {
				g.PhaseIn([]uuid.UUID{target.ID()})
			}
			g.ApplyContinuousEffects()
			wantBlock := change == "power-before" || change == "return-before" || change == "return-after" || change == "phase-before"
			if mage.CanBlock(g.FindPermanent(blocker.ID()), g.FindPermanent(target.ID()), g) != wantBlock {
				t.Fatal("legality or captured lifetime")
			}
		})
	}
}

func TestRONTemporaryBlockingRollback(t *testing.T) {
	for _, reason := range []string{"power", "noncreature", "phased", "unpaid", "sick", "shroud"} {
		t.Run(reason, func(t *testing.T) {
			g, a, b := ronGame()
			set, name := "ATQ", "Tawnos's Wand"
			if reason == "sick" {
				set, name = "LEB", "Dwarven Warriors"
			}
			source := g.PutOnBattlefield(repositoryCard(t, set, name), a.PlayerID())
			targetCard := repositoryCard(t, "LEB", "Grizzly Bears")
			if reason == "noncreature" {
				targetCard = repositoryCard(t, "LEB", "Forest")
			}
			target := g.PutOnBattlefield(targetCard, b.PlayerID())
			amount := 2
			if reason == "unpaid" {
				amount = 1
			}
			a.ManaPool().Add(core.Colorless, amount)
			switch reason {
			case "power":
				g.AddContinuousEffect(mage.TemporaryBoost(target.ID(), 1, 1))
			case "phased":
				g.PhaseOut(target.ID())
			case "shroud":
				g.AddContinuousEffect(mage.TemporaryKeyword(target.ID(), core.Shroud))
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err == nil {
				t.Fatal("illegal activation succeeded")
			}
			if g.StackSize() != 0 || g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != amount {
				t.Fatal("failed activation changed state")
			}
		})
	}
}

func TestRONTemporaryBlockingCopiesAndBindings(t *testing.T) {
	g, a, b := ronModalGame(0)
	c := repositoryCard(t, "ATQ", "Tawnos's Wand")
	otherCard := repositoryCard(t, "ATQ", "Tawnos's Wand")
	firstTarget := c.Abilities()[0].(*mage.ActionDefinition).Targets()[0]
	if firstTarget == otherCard.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
		t.Fatal("shared target declaration")
	}
	action := c.Abilities()[0].(*mage.ActionDefinition)
	invalid := mage.NewInstant("Undeclared", "", mage.WithAbility(mage.NewAction(mage.ActionSpell, mage.WithEffects(action.Effects()...))))
	if mage.ValidateCardDefinition(invalid.Name(), invalid) == nil {
		t.Fatal("missing typed declaration accepted")
	}
	source := g.PutOnBattlefield(c, a.PlayerID())
	first := g.PutOnBattlefield(repositoryCard(t, "LEB", "Grizzly Bears"), b.PlayerID())
	second := g.PutOnBattlefield(repositoryCard(t, "LEB", "Grizzly Bears"), b.PlayerID())
	blocker := g.PutOnBattlefield(repositoryCard(t, "LEB", "Grizzly Bears"), a.PlayerID())
	a.ManaPool().Add(core.Colorless, 2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{first.ID()}); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	a.targets = []uuid.UUID{second.ID()}
	if g.CopyStackObjectDirect(original, a.PlayerID(), true) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveTopOfStack()
	if mage.CanBlock(g.FindPermanent(blocker.ID()), g.FindPermanent(second.ID()), g) || !mage.CanBlock(g.FindPermanent(blocker.ID()), g.FindPermanent(first.ID()), g) {
		t.Fatal("copy target isolation")
	}
	g.ResolveStack()
	if mage.CanBlock(g.FindPermanent(blocker.ID()), g.FindPermanent(first.ID()), g) {
		t.Fatal("original lost target")
	}
}

func TestRONFixedPowerFilters(t *testing.T) {
	for _, comparison := range []string{"LessThanOrEqualTo", "GreaterThanOrEqualTo", "EqualTo"} {
		for _, limit := range []int{-1, 2, 3} {
			nodes, err := Parse(strings.NewReader(fmt.Sprintf(`PowerIs(%s(Integer(%d)))`, comparison, limit)), "filter.ron")
			if err != nil {
				t.Fatal(err)
			}
			filter, err := compileFilter(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			g, a, _ := ronGame()
			p := g.PutOnBattlefield(repositoryCard(t, "LEB", "Grizzly Bears"), a.PlayerID())
			for _, power := range []int{2, 3} {
				want := power == limit
				if comparison == "LessThanOrEqualTo" {
					want = power <= limit
				}
				if comparison == "GreaterThanOrEqualTo" {
					want = power >= limit
				}
				if filter.Match(p, g) != want {
					t.Fatalf("%d %s %d", power, comparison, limit)
				}
				g.AddContinuousEffect(mage.TemporaryBoost(p.ID(), 1, 0))
			}
		}
	}
}

func TestRONTemporaryBlockingInvalid(t *testing.T) {
	for _, body := range []string{
		`CreatePermanentRuleEffectUntil(Ref_TargetPermanent, [], UntilEndOfTurn)`,
		`CreatePermanentRuleEffectUntil(Ref_TargetPermanent, [CantBeBlocked()], UntilEndOfTurn)`,
		`CreatePermanentRuleEffectUntil(Ref_TargetPermanent, [CantBeBlocked, Flying], UntilEndOfTurn)`,
		`CreatePermanentRuleEffectUntil(Ref_TargetPermanent, [CantBeBlocked], UntilEndOfCombat)`,
		`CreatePermanentRuleEffectUntil(Ref_TargetPermanent(), [CantBeBlocked], UntilEndOfTurn)`,
		`CreatePermanentRuleEffectUntil(Ref_TargetPlayer, [CantBeBlocked], UntilEndOfTurn)`,
		`CreatePermanentRuleEffectUntil(ThisPermanent, [CantBeBlocked], UntilEndOfTurn)`,
		`CreatePermanentRuleEffectUntil(Ref_TargetPermanent, [CantBeBlocked])`,
	} {
		source := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([` + body + `])))])`
		nodes, err := Parse(strings.NewReader(source), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
	for _, source := range []string{
		`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([CreatePermanentRuleEffectUntil(Ref_TargetPermanent,[CantBeBlocked],UntilEndOfTurn)]))])`,
		`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Modal_ChooseOne([Targeted([TargetPermanent(AnyPermanent)],ActionList([])),ActionList([CreatePermanentRuleEffectUntil(Ref_TargetPermanent,[CantBeBlocked],UntilEndOfTurn)])]))])`,
	} {
		nodes, err := Parse(strings.NewReader(source), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("out-of-scope reference accepted")
		}
	}
	for _, expression := range []string{`PowerIs`, `PowerIs()`, `PowerIs(LessThanOrEqualTo(2))`, `PowerIs(LessThanOrEqualTo(ValueX))`, `PowerIs(LessThanOrEqualTo(Integer(2),Integer(3)))`, `PowerIs(LessThan(Integer(2)))`, `PowerIs(LessThanOrEqualTo(Integer(2,3)))`, `PowerIs(LessThanOrEqualTo(PowerOfPermanent(ThisPermanent)))`} {
		nodes, err := Parse(strings.NewReader(expression), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = compileFilter(nodes[0]); err == nil {
			t.Fatalf("accepted %s", expression)
		}
	}
}

func TestRONTemporaryBlockingPhaseAndSourceLifetime(t *testing.T) {
	g, a, b := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Tawnos's Wand"), a.PlayerID())
	target := g.PutOnBattlefield(repositoryCard(t, "LEB", "Grizzly Bears"), b.PlayerID())
	blocker := g.PutOnBattlefield(repositoryCard(t, "LEB", "Grizzly Bears"), a.PlayerID())
	a.ManaPool().Add(core.Colorless, 2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PhaseIn(g.PhaseOut(target.ID()))
	late := g.PutOnBattlefield(repositoryCard(t, "LEB", "Grizzly Bears"), a.PlayerID())
	if mage.CanBlock(g.FindPermanent(blocker.ID()), g.FindPermanent(target.ID()), g) || mage.CanBlock(late, g.FindPermanent(target.ID()), g) {
		t.Fatal("source departure or phase cycle ended restriction")
	}
	g.PhaseOut(target.ID())
	g.RunStepWithPriority(core.Cleanup)
	g.PhaseIn([]uuid.UUID{target.ID()})
	if !mage.CanBlock(g.FindPermanent(blocker.ID()), g.FindPermanent(target.ID()), g) {
		t.Fatal("restriction did not expire while phased")
	}
}

func TestRONTemporaryBlockingSpellAndTriggerContexts(t *testing.T) {
	for _, context := range []string{"spell", "mode", "trigger"} {
		t.Run(context, func(t *testing.T) {
			body := `Targeted([TargetPermanent(And([IsCardtype(Creature),PowerIs(LessThanOrEqualTo(Integer(2)))]))],ActionList([CreatePermanentRuleEffectUntil(Ref_TargetPermanent,[CantBeBlocked],UntilEndOfTurn)]))`
			kind, rule := "Instant", `SpellActions(`+body+`)`
			if context == "mode" {
				rule = `SpellActions(Modal_ChooseOne([` + body + `,Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreatePermanentRuleEffectUntil(Ref_TargetPermanent,[CantBeBlocked],UntilEndOfTurn)]))]))`
			}
			if context == "trigger" {
				kind, rule = "Artifact", `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),`+body+`)`
			}
			factory := permanentScopeFactory(t, kind, rule)
			g, a, b := ronModalGame(0)
			target := g.PutOnBattlefield(repositoryCard(t, "LEB", "Grizzly Bears"), b.PlayerID())
			blocker := g.PutOnBattlefield(repositoryCard(t, "LEB", "Grizzly Bears"), a.PlayerID())
			c := factory()
			c.SetOwner(a.PlayerID())
			if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			if context == "trigger" {
				a.targets = []uuid.UUID{target.ID()}
				g.PutOnBattlefield(c, a.PlayerID())
				g.PutTriggersOnStack()
			} else {
				a.AddToHand(c)
				if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
					t.Fatal(err)
				}
			}
			if g.StackSize() != 1 || !mage.CanBlock(blocker, target, g) {
				t.Fatal("restriction started before resolution")
			}
			g.ResolveStack()
			if mage.CanBlock(g.FindPermanent(blocker.ID()), g.FindPermanent(target.ID()), g) {
				t.Fatal("context lost target binding")
			}
		})
	}
}

func TestRONTemporaryBlockingInventory(t *testing.T) {
	report, err := InspectFS(os.DirFS("../../../../cards/mtgish"), "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	wanted := map[string]string{"LessThanOrEqualTo": "power_comparison", "CantBeBlocked": "permanent_rule_operation"}
	for name, role := range wanted {
		found := false
		for _, s := range report.Signatures {
			if s.Name != name || s.Role != role {
				continue
			}
			for _, card := range s.Cards {
				if card == "Dwarven Warriors" {
					found = true
					if s.CompiledOccurrences != 7 {
						t.Fatalf("%s compiled records=%d", name, s.CompiledOccurrences)
					}
				}
			}
		}
		if !found {
			t.Fatalf("missing %s role %s", name, role)
		}
	}
}
