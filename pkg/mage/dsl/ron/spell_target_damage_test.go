package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONSpellTargetDamageBronzeHorse(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEG", "Bronze Horse")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("unpaid cast")
	}
	a.ManaPool().Add(core.Blue, 7)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	horse := g.FindPermanent(card.ID())
	if horse == nil || horse.CurrentPower(g) != 4 || horse.CurrentToughness(g) != 4 || !horse.HasKeyword(core.Trample) || a.ManaPool().TotalMana() != 0 {
		t.Fatal("horse metadata or casting")
	}
	cast := func(targeted bool) {
		t.Helper()
		rules := `SpellActions(ActionList([SpellDealsDamage(ThisSpell,Integer(1),EachPermanent(IsCardtype(Creature)))]))`
		var targets []uuid.UUID
		if targeted {
			rules = `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([SpellDealsDamage(ThisSpell,Integer(1),Permanent(Ref_TargetPermanent))])))`
			targets = []uuid.UUID{horse.ID()}
		}
		spell := permanentScopeFactory(t, "Instant", rules)()
		spell.SetOwner(b.PlayerID())
		b.AddToHand(spell)
		if err := g.CastSpellByID(b.PlayerID(), spell.ID(), targets, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
	}
	cast(true)
	if g.FindPermanent(horse.ID()).Damage != 1 {
		t.Fatal("horse counted itself as another creature")
	}
	other := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 10), a.PlayerID())
	cast(true)
	if g.FindPermanent(horse.ID()).Damage != 1 {
		t.Fatal("targeted spell damage was not prevented")
	}
	cast(false)
	if g.FindPermanent(horse.ID()).Damage != 2 {
		t.Fatal("untargeted spell damage was prevented")
	}
	g.DealDamageToPermanent(g.FindPermanent(horse.ID()), 1, other.ID())
	if g.FindPermanent(horse.ID()).Damage != 3 {
		t.Fatal("permanent damage was prevented")
	}
}

func TestRONSpellTargetDamageInvalid(t *testing.T) {
	valid := `PreventDamage(DamageWouldBeDealtByASpellToRecipient(TargetsPermanent(ThisPermanent),Permanent(ThisPermanent)),[PreventThatDamage])`
	for _, rule := range []string{
		strings.ReplaceAll(valid, "TargetsPermanent(ThisPermanent)", "TargetsPermanent()"),
		strings.ReplaceAll(valid, "TargetsPermanent(ThisPermanent)", "TargetsPermanent(ThisPermanent,ThisPermanent)"),
		strings.ReplaceAll(valid, "TargetsPermanent(ThisPermanent)", "TargetsPermanent(ThisPermanent())"),
		strings.ReplaceAll(valid, "TargetsPermanent(ThisPermanent)", "TargetsPermanent(HostPermanent)"),
		strings.ReplaceAll(valid, "TargetsPermanent(ThisPermanent)", "TargetsPermanent(Ref_TargetPermanent)"),
		strings.ReplaceAll(valid, "TargetsPermanent(ThisPermanent)", "IsCardtype(Instant)"),
		strings.ReplaceAll(valid, ",Permanent(ThisPermanent))", ",Player(You))"),
		strings.ReplaceAll(valid, ",Permanent(ThisPermanent))", ",Permanent(HostPermanent))"),
		strings.ReplaceAll(valid, "[PreventThatDamage]", "[]"),
		strings.ReplaceAll(valid, "[PreventThatDamage]", "[PreventThatDamage()]"),
		`Activated(PayMana([]),ActionList([` + valid + `]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted", rule)
		}
	}
}

func TestRONSpellTargetDamageLifecycle(t *testing.T) {
	for _, change := range []string{"copy", "clone", "other-leaves", "other-control", "horse-control", "phase-cycle", "remove-abilities", "return"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", "Bronze Horse")
			horse := g.PutOnBattlefield(card, a.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 10), a.PlayerID())
			spell := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([SpellDealsDamage(ThisSpell,Integer(1),Permanent(Ref_TargetPermanent))])))`)()
			spell.SetOwner(b.PlayerID())
			b.AddToHand(spell)
			if err := g.CastSpellByID(b.PlayerID(), spell.ID(), []uuid.UUID{horse.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			want := 0
			switch change {
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy")
				}
			case "clone":
				clone := g.Clone()
				clone.RemoveFromBattlefield(clone.FindPermanent(other.ID()))
				clone.ResolveStack()
				if clone.FindPermanent(horse.ID()).Damage != 1 || g.FindPermanent(horse.ID()).Damage != 0 {
					t.Fatal("clone isolation")
				}
			case "other-leaves":
				g.RemoveFromBattlefield(g.FindPermanent(other.ID()))
				want = 1
			case "other-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: horse.ID(), TargetID: other.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				want = 1
			case "horse-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: other.ID(), TargetID: horse.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				want = 1
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(horse.ID()))
			case "remove-abilities":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), horse.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				want = 1
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(horse.ID()))
				g.PutOnBattlefield(card, a.PlayerID())
			}
			g.ResolveStack()
			if g.FindPermanent(horse.ID()).Damage != want {
				t.Fatal("live condition or source lifecycle")
			}
		})
	}
}

func TestRONSpellTargetDamageIllegalTargetFacts(t *testing.T) {
	g, a, b := ronGame()
	horse := g.PutOnBattlefield(repositoryCard(t, "LEG", "Bronze Horse"), a.PlayerID())
	other := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 10), a.PlayerID())
	spell := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature)),TargetPermanent(IsCardtype(Creature))],ActionList([SpellDealsDamage(ThisSpell,Integer(1),EachPermanent(IsCardtype(Creature)))])))`)()
	spell.SetOwner(b.PlayerID())
	b.AddToHand(spell)
	if err := g.CastSpellByID(b.PlayerID(), spell.ID(), []uuid.UUID{horse.ID(), other.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.MutablePermanent(horse.ID()).GrantBaseAttr(core.Shroud)
	g.ResolveStack()
	if g.FindPermanent(horse.ID()).Damage != 0 || g.FindPermanent(other.ID()).Damage != 1 {
		t.Fatal("illegal target erased the spell's announced target facts")
	}
}

func TestRONSpellTargetDamageCombat(t *testing.T) {
	g, a, b := ronGame()
	horse := g.PutOnBattlefield(repositoryCard(t, "LEG", "Bronze Horse"), a.PlayerID())
	horse.RevokeBaseAttr(core.AttrSummonSick)
	blocker := g.PutOnBattlefield(mage.NewCreature("Blocker", "", 1, 1), b.PlayerID())
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{horse.ID()})
	g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: blocker.ID(), AttackerID: horse.ID()}})
	g.ExecuteCombatDamage()
	if b.Life() != 17 || g.FindPermanent(horse.ID()).Damage != 1 {
		t.Fatal("trample or combat exclusion")
	}
}

func TestRONSpellTargetDamageActivatedAbility(t *testing.T) {
	g, a, b := ronGame()
	horse := g.PutOnBattlefield(repositoryCard(t, "LEG", "Bronze Horse"), a.PlayerID())
	g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 10), a.PlayerID())
	pinger := g.PutOnBattlefield(repositoryCard(t, "LEA", "Prodigal Sorcerer"), b.PlayerID())
	pinger.RevokeBaseAttr(core.AttrSummonSick)
	if err := g.ActivateAbilityByIndex(b.PlayerID(), pinger.ID(), 0, []uuid.UUID{horse.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(horse.ID()).Damage != 1 {
		t.Fatal("targeting ability was treated as a targeting spell")
	}
}
