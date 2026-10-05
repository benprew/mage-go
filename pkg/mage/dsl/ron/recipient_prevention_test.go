package ron

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const recipientPrevention = `CreatePreventDamageUntil(DamageWouldBeDealtToRecipient(Permanent(Ref_TargetPermanent)),[PreventThatDamage],UntilEndOfTurn)`

func TestRONRecipientPreventionAura(t *testing.T) {
	for _, state := range []string{"normal", "unpaid", "land", "player", "shroud-before", "shroud-after", "depart-before", "phase-before", "control", "phase-after", "return-after", "copy"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", "Indestructible Aura")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			targetCard := mage.NewCreature("Target", "", 2, 20)
			targetCard.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(targetCard, b.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 20), b.PlayerID())
			id := p.ID()
			if state == "land" {
				id = g.PutOnBattlefield(mage.NewLand("Land"), b.PlayerID()).ID()
			}
			if state == "player" {
				id = b.PlayerID()
			}
			if state == "shroud-before" {
				p.GrantBaseAttr(core.Shroud)
			}
			if state != "unpaid" {
				a.ManaPool().Add(core.White, 1)
			}
			err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{id}, 0)
			if state == "unpaid" || state == "land" || state == "player" || state == "shroud-before" {
				if err == nil || g.StackSize() != 0 || len(a.Hand()) != 1 || (state != "unpaid" && a.ManaPool().TotalMana() != 1) {
					t.Fatal("illegal cast changed state")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			g.DealDamageToPermanent(g.FindPermanent(id), 1, other.ID())
			if g.FindPermanent(id).Damage != 1 {
				t.Fatal("prevention started before resolution")
			}
			active := true
			switch state {
			case "shroud-after":
				g.FindPermanent(id).GrantBaseAttr(core.Shroud)
				active = false
			case "depart-before":
				g.RemoveFromBattlefield(g.FindPermanent(id))
				g.PutOnBattlefield(targetCard, b.PlayerID())
				active = false
			case "phase-before":
				g.PhaseOut(id)
				active = false
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			pending := g.Clone()
			g.ResolveStack()
			if state == "phase-before" {
				g.PhaseIn([]uuid.UUID{id})
			}
			if state == "control" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: id, ControllerID: a.PlayerID(), Duration: core.Indefinite})
			}
			if state == "phase-after" {
				g.PhaseIn(g.PhaseOut(id))
			}
			if state == "return-after" {
				g.RemoveFromBattlefield(g.FindPermanent(id))
				g.PutOnBattlefield(targetCard, b.PlayerID())
				active = false
			}
			before := g.FindPermanent(id).Damage
			source := g.CaptureDamageSource(other.ID())
			g.DealSimultaneousDamage(mage.PermanentDamageAssignment(g.FindPermanent(id), 2, source), mage.PermanentDamageAssignment(g.FindPermanent(id), 3, source), mage.PermanentDamageAssignment(g.FindPermanent(other.ID()), 1, source), mage.PlayerDamageAssignment(b, 1, source))
			want := before + 5
			if active {
				want = before
			}
			if g.FindPermanent(id).Damage != want || g.FindPermanent(other.ID()).Damage != 1 || b.Life() != 19 {
				t.Fatal("prevention scope, batch or identity")
			}
			g.DealDamageToPermanent(g.FindPermanent(id), 2, uuid.New())
			if active && g.FindPermanent(id).Damage != before {
				t.Fatal("nonpermanent source escaped")
			}
			if pending.FindPermanentIncludingPhased(id).Damage > 1 {
				t.Fatal("resolution mutated pending clone")
			}
			pending.ResolveStack()
			g.RunStepWithPriority(core.Cleanup)
			after := g.FindPermanent(id).Damage
			g.DealDamageToPermanent(g.FindPermanent(id), 1, other.ID())
			if g.FindPermanent(id).Damage != after+1 {
				t.Fatal("cleanup did not expire prevention")
			}
		})
	}
}

func TestRONRecipientPreventionArtifacts(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		cost      int
	}{{"DRK", "Scarecrow", 6}, {"LEG", "Al-abara's Carpet", 5}} {
		for _, state := range []string{"normal", "unpaid", "sick", "depart", "control", "copy"} {
			t.Run(tc.name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, tc.set, tc.name)
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				if tc.name == "Scarecrow" && (p.CurrentPower(g) != 2 || p.CurrentToughness(g) != 2 || !p.HasType(core.TypeCreature) || !p.HasType(core.TypeArtifact)) {
					t.Fatal("printed characteristics")
				}
				if state != "sick" {
					p.RevokeBaseAttr(core.AttrSummonSick)
				}
				if state != "unpaid" {
					a.ManaPool().Add(core.Colorless, tc.cost)
				}
				err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil)
				if state == "unpaid" || (state == "sick" && tc.name == "Scarecrow") {
					if err == nil || g.FindPermanent(p.ID()).Tapped || g.StackSize() != 0 || a.ManaPool().TotalMana() != map[bool]int{true: 0, false: tc.cost}[state == "unpaid"] {
						t.Fatal("cost rollback")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if !g.FindPermanent(p.ID()).Tapped || a.ManaPool().TotalMana() != 0 {
					t.Fatal("payment")
				}
				attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 1, 20, mage.WithKeyword(core.Flying)), b.PlayerID())
				g.DealDamageToPlayer(a, 1, attacker.ID())
				if a.Life() != 19 {
					t.Fatal("premature prevention")
				}
				if state == "depart" {
					g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				}
				if state == "control" {
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				if state == "copy" {
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy")
					}
				}
				clone := g.Clone()
				g.ResolveStack()
				clone.ResolveStack()
				for _, flying := range []bool{false, true} {
					for _, attacking := range []bool{false, true} {
						src := g.FindPermanent(attacker.ID())
						src.RevokeBaseAttr(core.AttrSummonSick)
						src.Tapped = false
						src.RevokeBaseAttr(core.Flying)
						if flying {
							src.GrantBaseAttr(core.Flying)
						}
						g.GetCombat().Reset()
						if attacking {
							g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{src.ID()})
						}
						blocked := (tc.name == "Scarecrow" && flying) || (tc.name != "Scarecrow" && attacking && !flying)
						before := a.Life()
						g.DealDamageToPlayer(a, 2, src.ID())
						want := before - 2
						if blocked {
							want = before
						}
						if a.Life() != want {
							t.Fatalf("flying=%v attacking=%v life=%d want=%d", flying, attacking, a.Life(), want)
						}
						before = b.Life()
						g.DealDamageToPlayer(b, 1, src.ID())
						want = before - 1
						if state == "copy" && blocked {
							want = before
						}
						if b.Life() != want {
							t.Fatal("wrong original or copy recipient")
						}
					}
				}
				before := a.Life()
				g.DealDamageToPlayer(a, 1, uuid.New())
				if a.Life() != before-1 {
					t.Fatal("noncreature source prevented")
				}
				g.GetEffects().RemoveEndOfCombat()
				if tc.name != "Scarecrow" {
					g.FindPermanent(attacker.ID()).RevokeBaseAttr(core.Flying)
					g.FindPermanent(attacker.ID()).Tapped = false
					g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{attacker.ID()})
				}
				before = a.Life()
				g.DealDamageToPlayer(a, 1, attacker.ID())
				if a.Life() != before {
					t.Fatal("end of combat expired prevention")
				}
				g.RunStepWithPriority(core.Cleanup)
				g.DealDamageToPlayer(a, 1, attacker.ID())
				if a.Life() != before-1 {
					t.Fatal("cleanup")
				}
			})
		}
	}
}

func TestRONRecipientPreventionInvalidAndInventory(t *testing.T) {
	for _, action := range []string{
		strings.Replace(recipientPrevention, "DamageWouldBeDealtToRecipient(Permanent(Ref_TargetPermanent))", "DamageWouldBeDealtToRecipient", 1),
		strings.Replace(recipientPrevention, "Permanent(Ref_TargetPermanent)", "Permanent(Ref_TargetPlayer)", 1),
		strings.Replace(recipientPrevention, "Permanent(Ref_TargetPermanent)", "Player(HostController)", 1),
		strings.Replace(recipientPrevention, "Permanent(Ref_TargetPermanent)", "Permanent(HostPermanent)", 1),
		strings.Replace(recipientPrevention, "[PreventThatDamage]", "[]", 1),
		strings.Replace(recipientPrevention, "[PreventThatDamage]", "[PreventThatDamage()]", 1),
		strings.Replace(recipientPrevention, "UntilEndOfTurn", "UntilEndOfCombat", 1),
		strings.Replace(recipientPrevention, "DamageWouldBeDealtToRecipient(Permanent(Ref_TargetPermanent))", "DamageWouldBeDealtByAPermanentToRecipient(And([]),Player(You))", 1),
	} {
		src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([%s])))])`, action)
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
	src := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(TapPermanent(ThisPermanent),ActionList([CreatePreventDamageUntil(DamageWouldBeDealtByAPermanentToRecipient(And([IsCardtype(Creature),HasAbility(Flying)]),Player(You)),[PreventThatDamage],UntilEndOfTurn)]))])`
	report, err := InspectFS(fstest.MapFS{"probe.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 1 {
		t.Fatalf("probe rejected: %v", report.Issues)
	}
	for _, sig := range report.Signatures {
		if sig.Name == "You" && sig.Role != "player_reference" {
			t.Fatalf("player role=%s", sig.Role)
		}
	}
}

func TestRONRecipientPreventionInstances(t *testing.T) {
	for _, target := range []string{"TargetPermanent(IsCardtype(Creature))", "TargetPlayer(AnyPlayer)", "AnyTarget"} {
		t.Run(target, func(t *testing.T) {
			ref := `Permanent(Ref_TargetPermanent)`
			if target == "TargetPlayer(AnyPlayer)" {
				ref = `Player(Ref_TargetPlayer)`
			}
			if target == "AnyTarget" {
				ref = `Ref_AnyTarget`
			}
			action := strings.Replace(recipientPrevention, `Permanent(Ref_TargetPermanent)`, ref, 1)
			nodes, err := Parse(strings.NewReader(fmt.Sprintf(`Card(Name:"Probe",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([%s],ActionList([%s])))])`, target, action)), "instances.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			first := g.PutOnBattlefield(mage.NewCreature("First", "", 1, 20), a.PlayerID())
			second := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 20), b.PlayerID())
			ids := []uuid.UUID{first.ID(), second.ID()}
			if target == "TargetPlayer(AnyPlayer)" {
				ids = []uuid.UUID{a.PlayerID(), b.PlayerID()}
			}
			for i, id := range ids {
				player := a
				if i == 1 {
					player = b
				}
				card := factory()
				card.SetOwner(player.PlayerID())
				player.AddToHand(card)
				if err := g.CastSpellByID(player.PlayerID(), card.ID(), []uuid.UUID{id}, 0); err != nil {
					t.Fatal(err)
				}
			}
			clone := g.Clone()
			g.ResolveTopOfStack()
			before := b.Life()
			g.DealDamageToPermanent(g.FindPermanent(second.ID()), 2, uuid.New())
			g.DealDamageToPlayer(b, 2, uuid.New())
			if target == "TargetPlayer(AnyPlayer)" {
				if b.Life() != before || g.FindPermanent(second.ID()).Damage != 2 {
					t.Fatal("player binding")
				}
			} else if g.FindPermanent(second.ID()).Damage != 0 || b.Life() != before-2 {
				t.Fatal("permanent binding")
			}
			if clone.FindPermanent(second.ID()).Damage != 0 || clone.GetPlayer(b.PlayerID()).Life() != 20 {
				t.Fatal("clone isolation")
			}
			before = a.Life()
			g.DealDamageToPermanent(g.FindPermanent(first.ID()), 1, uuid.New())
			g.DealDamageToPlayer(a, 1, uuid.New())
			if a.Life() != before-1 || g.FindPermanent(first.ID()).Damage != 1 {
				t.Fatal("factory bindings shared")
			}
			g.ResolveStack()
			clone.ResolveStack()
			before = a.Life()
			damage := g.FindPermanent(first.ID()).Damage
			g.DealDamageToPermanent(g.FindPermanent(first.ID()), 1, uuid.New())
			g.DealDamageToPlayer(a, 1, uuid.New())
			if target == "TargetPlayer(AnyPlayer)" {
				if a.Life() != before {
					t.Fatal("pending player binding")
				}
			} else if g.FindPermanent(first.ID()).Damage != damage {
				t.Fatal("pending permanent binding")
			}
		})
	}
}

func TestRONRecipientPreventionArtifactCasting(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"DRK", "Scarecrow"}, {"LEG", "Al-abara's Carpet"}} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, _ := ronGame()
			card := repositoryCard(t, tc.set, tc.name)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid cast")
			}
			if len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("cast rollback")
			}
			a.ManaPool().Add(core.Colorless, 5)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(card.ID()) != nil || a.ManaPool().TotalMana() != 0 {
				t.Fatal("cast timing or cost")
			}
			g.ResolveStack()
			if g.FindPermanent(card.ID()) == nil {
				t.Fatal("artifact did not enter")
			}
		})
	}
}

func TestRONRecipientPreventionCombat(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		cost      int
	}{{"DRK", "Scarecrow", 6}, {"LEG", "Al-abara's Carpet", 5}, {"LEG", "Indestructible Aura", 1}} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, tc.set, tc.name)
			card.SetOwner(a.PlayerID())
			ground := g.PutOnBattlefield(mage.NewCreature("Ground", "", 3, 20, mage.WithKeyword(core.Lifelink)), b.PlayerID())
			air := g.PutOnBattlefield(mage.NewCreature("Air", "", 3, 20, mage.WithKeyword(core.Flying), mage.WithKeyword(core.Lifelink)), b.PlayerID())
			victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 2, 20), a.PlayerID())
			if tc.name == "Indestructible Aura" {
				a.AddToHand(card)
				a.ManaPool().Add(core.White, 1)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{victim.ID()}, 0); err != nil {
					t.Fatal(err)
				}
			} else {
				p := g.PutOnBattlefield(card, a.PlayerID())
				p.RevokeBaseAttr(core.AttrSummonSick)
				a.ManaPool().Add(core.Colorless, tc.cost)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
			}
			g.ResolveStack()
			ground.RevokeBaseAttr(core.AttrSummonSick)
			air.RevokeBaseAttr(core.AttrSummonSick)
			g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{ground.ID(), air.ID()})
			if tc.name == "Indestructible Aura" {
				g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: victim.ID(), AttackerID: ground.ID()}})
			}
			g.ExecuteCombatDamage()
			if a.Life() != 17 || b.Life() != 23 || g.FindPermanent(victim.ID()).Damage != 0 {
				t.Fatalf("combat life=%d/%d damage=%d", a.Life(), b.Life(), g.FindPermanent(victim.ID()).Damage)
			}
			if tc.name == "Indestructible Aura" && g.FindPermanent(ground.ID()).Damage != 2 {
				t.Fatal("recipient prevention stopped outgoing damage")
			}
		})
	}
}

func TestRONRecipientPreventionCopyRetarget(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEG", "Indestructible Aura")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.White, 1)
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 1, 20), a.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 20), b.PlayerID())
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{first.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	phased := g.PhaseOut(first.ID())
	cp := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), true)
	if cp == nil || len(cp.Targets) != 1 || cp.Targets[0] != second.ID() {
		t.Fatal("copy target")
	}
	g.PhaseIn(phased)
	clone := g.Clone()
	g.ResolveTopOfStack()
	g.DealDamageToPermanent(g.FindPermanent(second.ID()), 2, uuid.New())
	g.DealDamageToPermanent(g.FindPermanent(first.ID()), 2, uuid.New())
	if g.FindPermanent(second.ID()).Damage != 0 || g.FindPermanent(first.ID()).Damage != 2 {
		t.Fatal("copy changed original binding")
	}
	g.ResolveStack()
	clone.ResolveStack()
	damage := g.FindPermanent(first.ID()).Damage
	g.DealDamageToPermanent(g.FindPermanent(first.ID()), 2, uuid.New())
	if g.FindPermanent(first.ID()).Damage != damage || clone.FindPermanent(first.ID()).Damage != 0 {
		t.Fatal("original or clone binding")
	}
}
