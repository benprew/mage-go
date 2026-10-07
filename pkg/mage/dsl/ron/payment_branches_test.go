package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONPaymentBranchesSlug(t *testing.T) {
	for _, attacking := range []bool{false, true} {
		for _, choice := range []string{"paid", "declined", "short", "mana-source"} {
			t.Run(fmt.Sprintf("attack=%t/%s", attacking, choice), func(t *testing.T) {
				a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: choice != "declined"}
				b := mage.NewBasePlayer("B")
				g := mage.NewGame(a, b)
				c := repositoryCard(t, "DRK", "Spitting Slug")
				if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
					t.Fatal(err)
				}
				p := g.PutOnBattlefield(c, a.PlayerID())
				p.RevokeBaseAttr(core.AttrSummonSick)
				other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 5), b.PlayerID())
				other.RevokeBaseAttr(core.AttrSummonSick)
				unrelated := g.PutOnBattlefield(mage.NewCreature("Unrelated", "", 2, 5), b.PlayerID())
				a.ManaPool().Add(core.Colorless, 1)
				if choice == "mana-source" {
					g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
				} else if choice != "short" {
					a.ManaPool().Add(core.Green, 1)
				}
				if attacking {
					g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
					g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: other.ID(), AttackerID: p.ID()}})
				} else {
					g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{other.ID()})
					g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: p.ID(), AttackerID: other.ID()}})
				}
				g.ResolveStack()
				paid := choice == "paid" || choice == "mana-source"
				if g.FindPermanent(p.ID()).HasAttr(core.FirstStrike) != paid || g.FindPermanent(other.ID()).HasAttr(core.FirstStrike) == paid || g.FindPermanent(unrelated.ID()).HasAttr(core.FirstStrike) {
					t.Fatal("wrong first strike recipients")
				}
				if a.offers != map[bool]int{true: 0, false: 1}[choice == "short"] {
					t.Fatal("payment offers")
				}
				if paid && g.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 0 {
					t.Fatal("payment not spent")
				}
				if !paid && g.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != map[bool]int{true: 1, false: 2}[choice == "short"] {
					t.Fatal("unpaid payment consumed mana")
				}
				g.RemoveFromCombat(other.ID())
				if g.FindPermanent(other.ID()).HasAttr(core.FirstStrike) == paid {
					t.Fatal("leaving combat ended captured grant")
				}
				g.RunStepWithPriority(core.Cleanup)
				if g.FindPermanent(p.ID()).HasAttr(core.FirstStrike) || g.FindPermanent(other.ID()).HasAttr(core.FirstStrike) {
					t.Fatal("grant did not expire")
				}
			})
		}
	}
}

func TestRONPaymentBranchesNested(t *testing.T) {
	for _, condition := range []string{"CostWasPaid", "CostWasntPaid"} {
		a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
		g := mage.NewGame(a, mage.NewBasePlayer("B"))
		g.SetStep(core.PrecombatMain)
		paid := `[MayCost(PayMana([ManaCostW])),IfElse(CostWasPaid,[GainLife(Integer(100))],[GainLife(Integer(3))])]`
		unpaid := `[GainLife(Integer(7))]`
		if condition == "CostWasntPaid" {
			paid, unpaid = unpaid, paid
		}
		actions := `MayCost(PayMana([ManaCostGeneric(1)])),IfElse(` + condition + `,` + paid + `,` + unpaid + `),MayCost(PayMana([ManaCostGeneric(1)])),IfElse(CostWasPaid,[GainLife(Integer(100))],[GainLife(Integer(2))])`
		c := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([`+actions+`]))`)()
		a.ManaPool().Add(core.Colorless, 1)
		c.SetOwner(a.PlayerID())
		a.AddToHand(c)
		if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if g.GetPlayer(a.PlayerID()).Life() != 25 || a.offers != 1 {
			t.Fatal("nested or adjacent result escaped")
		}
	}
}

func TestRONPaymentBranchesInvalid(t *testing.T) {
	pair := `MayCost(PayMana([ManaCostGeneric(1)])),IfElse(CostWasPaid,[GainLife(Integer(1))],[GainLife(Integer(2))])`
	for _, actions := range []string{
		`IfElse(CostWasPaid,[GainLife(Integer(1))],[GainLife(Integer(2))])`,
		strings.Replace(pair, "CostWasPaid", "CostWasPaid()", 1),
		strings.Replace(pair, "CostWasPaid", "You", 1),
		strings.Replace(pair, "[GainLife(Integer(1))]", "[]", 1),
		strings.Replace(pair, "[GainLife(Integer(2))]", "[]", 1),
		strings.Replace(pair, "MayCost", "MustCost", 1),
		strings.Replace(pair, "PayMana([ManaCostGeneric(1)])", "PayMana([ManaCostX])", 1),
		strings.Replace(pair, "PayMana([ManaCostGeneric(1)])", "SacrificePermanent(ThisPermanent)", 1),
		strings.Replace(pair, "IfElse", "GainLife(Integer(1)),IfElse", 1),
		pair + `,If(CostWasPaid,[GainLife(Integer(100))])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([`+actions+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted", actions)
		}
	}
}

func TestRONPaymentBranchesSlugIdentity(t *testing.T) {
	for _, change := range []string{"normal", "source-return", "source-phase", "phase-cycle", "other-return", "other-phase", "control", "copy", "clone", "late-blocker", "leave-combat"} {
		t.Run(change, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: false}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
			g := mage.NewGame(a, b)
			c := repositoryCard(t, "DRK", "Spitting Slug")
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			p.RevokeBaseAttr(core.AttrSummonSick)
			otherCard := mage.NewCreature("Other", "", 2, 5)
			otherCard.SetOwner(b.PlayerID())
			other := g.PutOnBattlefield(otherCard, b.PlayerID())
			second := g.PutOnBattlefield(repositoryCard(t, "DRK", "Spitting Slug"), a.PlayerID())
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
			g.GetCombat().AddBlockerIncarnation(other.ID(), g.CaptureDamageSource(other.ID()).Incarnation(), p.ID())
			g.FireEvent(core.GameEvent{Type: core.EvtCreatureBecomesBlocked, SourceID: p.ID(), PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || a.offers != 0 {
				t.Fatal("pending trigger timing")
			}
			a.ManaPool().Add(core.Green, 1)
			a.ManaPool().Add(core.Colorless, 1)
			switch change {
			case "source-return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				a.RemoveFromHand(c.ID())
				g.PutOnBattlefield(c, a.PlayerID())
			case "source-phase":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "other-return":
				g.BouncePermanentToHand(g.FindPermanent(other.ID()))
				b.RemoveFromHand(otherCard.ID())
				g.PutOnBattlefield(otherCard, b.PlayerID())
			case "other-phase":
				g.PhaseOut(other.ID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				b.ManaPool().Add(core.Green, 1)
				b.ManaPool().Add(core.Colorless, 1)
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "late-blocker":
				late := g.PutOnBattlefield(mage.NewCreature("Late", "", 2, 5), b.PlayerID())
				g.GetCombat().AddBlockerIncarnation(late.ID(), g.CaptureDamageSource(late.ID()).Incarnation(), p.ID())
			case "leave-combat":
				g.RemoveFromCombat(other.ID())
			}
			clone := g.Clone()
			g.ResolveStack()
			expected := change == "normal" || change == "copy" || change == "clone" || change == "late-blocker"
			if g.FindPermanentIncludingPhased(other.ID()).HasAttr(core.FirstStrike) != expected {
				t.Fatal("combat collection restored an absent pair or lost current pair")
			}
			if g.FindPermanent(second.ID()).HasAttr(core.FirstStrike) {
				t.Fatal("other instance changed")
			}
			if clone.FindPermanentIncludingPhased(other.ID()).HasAttr(core.FirstStrike) || clone.StackSize() == 0 {
				t.Fatal("clone state changed")
			}
			if change == "copy" {
				if !g.FindPermanent(p.ID()).HasAttr(core.FirstStrike) || b.offers != 1 || g.GetPlayer(b.PlayerID()).ManaPool().TotalMana() != 0 {
					t.Fatal("copy used wrong payer or source")
				}
			} else if g.FindPermanentIncludingPhased(p.ID()).HasAttr(core.FirstStrike) {
				t.Fatal("unpaid branch granted source keyword")
			}
			if a.offers != 1 || b.offers != map[bool]int{true: 1, false: 0}[change == "copy"] {
				t.Fatal("payer followed changed source control")
			}
			if change == "normal" {
				g.RemoveFromCombat(other.ID())
				late := g.PutOnBattlefield(mage.NewCreature("After", "", 1, 5), b.PlayerID())
				g.GetCombat().AddBlockerIncarnation(late.ID(), g.CaptureDamageSource(late.ID()).Incarnation(), p.ID())
				g.ApplyContinuousEffects()
				if !g.FindPermanent(other.ID()).HasAttr(core.FirstStrike) || g.FindPermanent(late.ID()).HasAttr(core.FirstStrike) {
					t.Fatal("collection was not captured at resolution")
				}
				g.PhaseOut(other.ID())
				g.PhaseIn([]uuid.UUID{other.ID()})
				if !g.FindPermanent(other.ID()).HasAttr(core.FirstStrike) {
					t.Fatal("phase cycle lost captured grant")
				}
				g.BouncePermanentToHand(g.FindPermanent(other.ID()))
				b.RemoveFromHand(otherCard.ID())
				g.PutOnBattlefield(otherCard, b.PlayerID())
				if g.FindPermanent(other.ID()).HasAttr(core.FirstStrike) {
					t.Fatal("returned object inherited grant")
				}
			}
		})
	}
}

func TestRONPaymentBranchesSlugCastingAndDamage(t *testing.T) {
	for _, paid := range []bool{false, true} {
		a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: paid}
		b := mage.NewBasePlayer("B")
		g := mage.NewGame(a, b)
		g.SetStep(core.PrecombatMain)
		c := repositoryCard(t, "DRK", "Spitting Slug")
		c.SetOwner(a.PlayerID())
		a.AddToHand(c)
		a.ManaPool().Add(core.Green, 2)
		if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err == nil {
			t.Fatal("unpaid casting")
		}
		if len(a.Hand()) != 1 || a.ManaPool().TotalMana() != 2 || g.StackSize() != 0 {
			t.Fatal("casting rollback")
		}
		a.ManaPool().Add(core.Colorless, 1)
		if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		if g.FindPermanent(c.ID()) != nil {
			t.Fatal("entry before resolution")
		}
		g.ResolveStack()
		p := g.FindPermanent(c.ID())
		if !p.HasSubType("Slug") || p.CurrentPower(g) != 2 || p.CurrentToughness(g) != 4 || len(c.Abilities()) != 2 {
			t.Fatal("metadata")
		}
		p.RevokeBaseAttr(core.AttrSummonSick)
		other := g.PutOnBattlefield(mage.NewCreature("Victim", "", 4, 2), b.PlayerID())
		a.ManaPool().Add(core.Green, 1)
		a.ManaPool().Add(core.Colorless, 1)
		g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
		g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: other.ID(), AttackerID: p.ID()}})
		g.ExecuteCombatDamage()
		if (g.FindPermanent(p.ID()) != nil) != paid || (g.FindPermanent(other.ID()) == nil) != paid {
			t.Fatal("first strike damage did not follow payment branch")
		}
	}
}

func TestRONPaymentBranchesKeywordDeclarations(t *testing.T) {
	valid := `CreateEachPermanentLayerEffectUntil(And([IsCardtype(Creature),Or([IsBlockingAttacker(ThisPermanent),IsBlockedByDefender(ThisPermanent)])]),[AddAbility([FirstStrike])],UntilEndOfTurn)`
	for _, action := range []string{
		strings.Replace(valid, "ThisPermanent", "Ref_TargetPermanent", 1),
		strings.Replace(valid, "FirstStrike", "FirstStrike()", 1),
		strings.Replace(valid, "[FirstStrike]", "[]", 1),
		strings.Replace(valid, "UntilEndOfTurn", "UntilEndOfCombat", 1),
		strings.Replace(valid, "Or([IsBlockingAttacker(ThisPermanent),IsBlockedByDefender(ThisPermanent)])", "Or([])", 1),
		strings.Replace(valid, "[AddAbility([FirstStrike])]", "[AddAbility([FirstStrike]),AdjustPT(+1,+1)]", 1),
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:4),Rules:[Activated(PayMana([]),ActionList([`+action+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted", action)
		}
	}
}

func TestRONPaymentBranchesTargetAndCopies(t *testing.T) {
	for _, change := range []string{"normal", "copy", "return", "phase", "shroud"} {
		t.Run(change, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: false}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([PlayerMayCost(You,PayMana([])),IfElse(CostWasPaid,[TapPermanent(Ref_TargetPermanent)],[CreatePermanentLayerEffectUntil(Ref_TargetPermanent,[AddAbility([FirstStrike])],UntilEndOfTurn)])])))`)
			c := factory()
			targetCard := mage.NewCreature("Target", "", 2, 4)
			targetCard.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(targetCard, a.PlayerID())
			c.SetOwner(a.PlayerID())
			a.AddToHand(c)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			second := g.PutOnBattlefield(mage.NewCreature("Second", "", 2, 4), b.PlayerID())
			switch change {
			case "copy":
				copied := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
				if copied == nil {
					t.Fatal("copy")
				}
				copied.Targets[0] = second.ID()
				copied.TargetZones[second.ID()] = core.ZoneBattlefield
			case "return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				a.RemoveFromHand(targetCard.ID())
				g.PutOnBattlefield(targetCard, a.PlayerID())
			case "phase":
				g.PhaseOut(p.ID())
			case "shroud":
				g.FindPermanent(p.ID()).GrantBaseAttr(core.Shroud)
				g.ApplyContinuousEffects()
			}
			clone := g.Clone()
			g.ResolveStack()
			legal := change == "normal" || change == "copy"
			if g.FindPermanentIncludingPhased(p.ID()).HasAttr(core.FirstStrike) != legal {
				t.Fatal("target binding or legality")
			}
			if a.offers != map[bool]int{true: 1, false: 0}[legal] {
				t.Fatal("payment offered for illegal target")
			}
			if change == "copy" && (!g.FindPermanent(second.ID()).Tapped || b.offers != 1) {
				t.Fatal("copy branch or controller")
			}
			if clone.FindPermanent(p.ID()) != nil && clone.FindPermanent(p.ID()).HasAttr(core.FirstStrike) {
				t.Fatal("clone changed")
			}
		})
	}
}

func TestRONPaymentBranchesSlugMultipleBlockers(t *testing.T) {
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: false}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	p := g.PutOnBattlefield(repositoryCard(t, "DRK", "Spitting Slug"), a.PlayerID())
	p.RevokeBaseAttr(core.AttrSummonSick)
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 1, 5), b.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 5), b.PlayerID())
	a.ManaPool().Add(core.Green, 1)
	a.ManaPool().Add(core.Colorless, 1)
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
	assignments := []mage.BlockAssignment{{BlockerID: first.ID(), AttackerID: p.ID()}, {BlockerID: second.ID(), AttackerID: p.ID()}}
	g.ExecuteBlockers(assignments)
	if a.offers != 1 || !g.FindPermanent(first.ID()).HasAttr(core.FirstStrike) || !g.FindPermanent(second.ID()).HasAttr(core.FirstStrike) {
		t.Fatal("blocked transition was not one payment for all current blockers")
	}
	g.ExecuteBlockers(assignments)
	if a.offers != 1 {
		t.Fatal("repeated blocker assignments repeated trigger")
	}
}

func TestRONPaymentBranchesCollectionKeywords(t *testing.T) {
	g, a, b := ronGame()
	early := g.PutOnBattlefield(mage.NewCreature("Early", "", 1, 4), a.PlayerID())
	opponent := g.PutOnBattlefield(mage.NewCreature("Opponent", "", 1, 4), b.PlayerID())
	artifact := g.PutOnBattlefield(mage.NewArtifact("Artifact", ""), a.PlayerID())
	c := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([CreateEachPermanentLayerEffectUntil(IsCardtype(Creature),[AddAbility([Flying,FirstStrike])],UntilEndOfTurn)]))`)()
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	late := g.PutOnBattlefield(mage.NewCreature("Late", "", 1, 4), a.PlayerID())
	for _, p := range []*mage.Permanent{early, opponent} {
		if !g.FindPermanent(p.ID()).HasAttr(core.Flying) || !g.FindPermanent(p.ID()).HasAttr(core.FirstStrike) {
			t.Fatal("missing captured keyword")
		}
	}
	if g.FindPermanent(late.ID()).HasAttr(core.Flying) || g.FindPermanent(artifact.ID()).HasAttr(core.Flying) {
		t.Fatal("grant membership expanded")
	}
	g.RunStepWithPriority(core.Cleanup)
	if g.FindPermanent(early.ID()).HasAttr(core.Flying) || g.FindPermanent(opponent.ID()).HasAttr(core.FirstStrike) {
		t.Fatal("keyword cleanup")
	}
}
