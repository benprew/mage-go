package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCombatTimingCasting(t *testing.T) {
	for _, condition := range []string{
		"IsDuringCombat", "IsBeforeAttackersDeclared", "IsBeforeBlockersDeclared", "IsBeforeCombatDamageStep",
		"And([IsDuringCombat,And([IsBeforeBlockersDeclared])])", "And([IsBeforeBlockersDeclared,IsDuringCombat])",
	} {
		factory := permanentScopeFactory(t, "Instant", `CastEffect(CantBeCastUnless(`+condition+`)),SpellActions(ActionList([GainLife(Integer(1))]))`)
		for step := core.Untap; step <= core.Cleanup; step++ {
			g, a, b := ronGame()
			g.SetStep(step)
			want := step < core.DeclareBlockers
			switch condition {
			case "IsDuringCombat":
				want = step >= core.BeginCombat && step <= core.EndCombat
			case "IsBeforeAttackersDeclared":
				want = step < core.DeclareAttackers
			case "IsBeforeCombatDamageStep":
				want = step < core.FirstStrikeDamage
			case "And([IsDuringCombat,And([IsBeforeBlockersDeclared])])", "And([IsBeforeBlockersDeclared,IsDuringCombat])":
				want = step >= core.BeginCombat && step < core.DeclareBlockers
			}
			for _, player := range []*mage.BasePlayer{a, b} {
				card := factory()
				card.SetOwner(player.PlayerID())
				player.AddToHand(card)
				if got := len(g.GetCastableSpells(player.PlayerID())) == 1; got != want {
					t.Fatalf("%s/%s query = %v", condition, step, got)
				}
				err := g.CastSpellByID(player.PlayerID(), card.ID(), nil, 0)
				if (err == nil) != want {
					t.Fatalf("%s/%s cast: %v", condition, step, err)
				}
				if want {
					g.SetStep(core.Cleanup)
					if g.CopyStackObjectDirect(g.StackPeek(), player.PlayerID(), false) == nil {
						t.Fatal("stack copy repeated timing check")
					}
					g.ResolveStack()
					if player.Life() != 22 {
						t.Fatal("resolution repeated timing check")
					}
					g = mage.NewGame(a, b)
					g.SetStep(step)
				} else if len(player.Hand()) != 1 || g.StackSize() != 0 {
					t.Fatal("rejection changed resources")
				}
			}
		}
	}
}

func TestRONCombatTimingInvalid(t *testing.T) {
	for _, condition := range []string{"IsDuringCombat", "IsBeforeAttackersDeclared", "IsBeforeBlockersDeclared", "IsBeforeCombatDamageStep"} {
		for _, invalid := range []string{condition + "()", condition + "(You)"} {
			src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[CastEffect(CantBeCastUnless(%s)),SpellActions(ActionList([GainLife(Integer(1))]))])`, invalid)
			nodes, err := Parse(strings.NewReader(src), "invalid.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = Compile(nodes[0]); err == nil {
				t.Fatalf("accepted %s", invalid)
			}
		}
	}
}

func TestRONCombatTimingExtraCombat(t *testing.T) {
	for _, condition := range []string{"IsBeforeBlockersDeclared", "And([IsDuringCombat,IsBeforeBlockersDeclared])", "And([IsBeforeBlockersDeclared,And([IsDuringCombat])])"} {
		g, a, _ := ronGame()
		for _, step := range []core.PhaseStep{core.BeginCombat, core.DeclareBlockers, core.EndCombat, core.PostcombatMain, core.BeginCombat} {
			g.SetStep(step)
		}
		card := permanentScopeFactory(t, "Instant", `CastEffect(CantBeCastUnless(`+condition+`)),SpellActions(ActionList([GainLife(Integer(1))]))`)()
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0)
		if (err == nil) != (condition != "IsBeforeBlockersDeclared") {
			t.Fatalf("extra combat %s: %v", condition, err)
		}
	}
}

func TestRONCombatTimingAngusLifetime(t *testing.T) {
	for _, change := range []string{"none", "departure", "phase", "control", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", "Angus Mackenzie")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid cast")
			}
			for _, color := range []core.Color{core.Green, core.White, core.Blue} {
				a.ManaPool().Add(color, 1)
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			source := g.FindPermanent(card.ID())
			if source.CurrentPower(g) != 2 || source.CurrentToughness(g) != 2 || !source.Card.HasSuperType(core.SuperLegendary) {
				t.Fatal("metadata")
			}
			for _, color := range []core.Color{core.Green, core.White, core.Blue} {
				a.ManaPool().Add(color, 1)
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
				t.Fatal("sick activation")
			}
			if a.ManaPool().TotalMana() != 3 || source.Tapped {
				t.Fatal("rollback")
			}
			g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrSummonSick)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			pending := g.Clone()
			switch change {
			case "departure":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			case "phase":
				g.PhaseOut(source.ID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy")
				}
			}
			g.SetStep(core.CombatDamage)
			g.ResolveStack()
			if pending.GetEffects().ApplyReplacements(mage.NewDamageToPlayerAction(uuid.Nil, b.PlayerID(), 1, true), pending) == nil {
				t.Fatal("pending clone changed")
			}
			assertRONGlobalCombatPrevention(t, g, a.PlayerID(), b.PlayerID())
			pending.ResolveStack()
			if pending.GetEffects().ApplyReplacements(mage.NewDamageToPlayerAction(uuid.Nil, b.PlayerID(), 1, true), pending) != nil {
				t.Fatal("pending clone lost prevention")
			}
		})
	}
}

func TestRONCombatTimingAngus(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEG", "Angus Mackenzie")
	source := g.PutOnBattlefield(card, a.PlayerID())
	source.RevokeBaseAttr(core.AttrSummonSick)
	a.ManaPool().Add(core.Green, 1)
	a.ManaPool().Add(core.White, 1)
	a.ManaPool().Add(core.Blue, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != 0 {
		t.Fatal("cost payment")
	}
	g.ResolveStack()
	g.SetStep(core.FirstStrikeDamage)
	g.UntapPermanent(g.FindPermanent(source.ID()))
	a.ManaPool().Add(core.Green, 1)
	a.ManaPool().Add(core.White, 1)
	a.ManaPool().Add(core.Blue, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
		t.Fatal("late activation accepted")
	}
	if g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != 3 {
		t.Fatal("timing rejection spent resources")
	}
	attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 3, 3), a.PlayerID())
	attacker.RevokeBaseAttr(core.AttrSummonSick)
	g.SetStep(core.DeclareAttackers)
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{attacker.ID()})
	g.ExecuteCombatDamage()
	if b.Life() != 20 {
		t.Fatal("combat damage not prevented")
	}
	g.DealDamageToPlayer(b, 2, source.ID())
	if b.Life() != 18 {
		t.Fatal("noncombat damage prevented")
	}
}
