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

func TestRONPermanentDamageRealCards(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		amount    int
	}{
		{"LEG", "Crimson Manticore", 1}, {"LEG", "D'Avenant Archer", 1},
		{"ATQ", "Grapeshot Catapult", 1}, {"LEG", "Lady Caleria", 3}, {"LEG", "Tor Wauki", 2},
	} {
		for _, scenario := range []string{"attacking", "blocking", "idle", "player", "land", "sick", "unpaid", "return", "phase", "phase-cycle", "combat-removal", "flying-removal", "control", "shroud", "source-return", "source-phase", "prevention", "copy"} {
			t.Run(tc.name+"/"+scenario, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, tc.set, tc.name)
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				if tc.name == "Crimson Manticore" && !source.HasKeyword(core.Flying) {
					t.Fatal("missing printed flying")
				}
				if scenario != "sick" {
					source.RevokeBaseAttr(core.AttrSummonSick)
				}
				if scenario == "source-return" {
					source.GrantBaseAttr(core.Lifelink)
				}
				victimCard := repositoryCard(t, "LEA", "Wall of Stone")
				victimCard.SetOwner(b.PlayerID())
				victim := g.PutOnBattlefield(victimCard, b.PlayerID())
				current := victim.ID()
				if tc.name == "Grapeshot Catapult" {
					if scenario != "idle" {
						victim.GrantBaseAttr(core.Flying)
					}
				} else if scenario == "blocking" {
					g.GetCombat().AddAttacker(source.ID(), b.PlayerID())
					g.GetCombat().AddBlocker(victim.ID(), source.ID())
				} else if scenario != "idle" {
					g.GetCombat().AddAttacker(victim.ID(), a.PlayerID())
				}
				target := current
				if scenario == "player" {
					target = b.PlayerID()
				}
				if scenario == "land" {
					target = g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID()).ID()
				}
				needsMana := tc.name == "Crimson Manticore"
				if needsMana && scenario != "unpaid" {
					a.ManaPool().Add(core.Red, 1)
				}
				beforeMana := a.ManaPool().TotalMana()
				err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target})
				reject := scenario == "idle" || scenario == "player" || scenario == "land" || scenario == "sick" || (needsMana && scenario == "unpaid")
				if reject {
					if err == nil {
						t.Fatal("illegal activation accepted")
					}
					if g.StackSize() != 0 || g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != beforeMana {
						t.Fatal("failed activation changed state")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if !g.FindPermanent(source.ID()).Tapped || victim.Damage != 0 || a.ManaPool().TotalMana() != 0 {
					t.Fatal("cost or resolution timing")
				}
				want := tc.amount
				switch scenario {
				case "return":
					g.BouncePermanentToHand(g.FindPermanent(current))
					b.RemoveFromHand(victimCard.ID())
					current = g.PutOnBattlefield(victimCard, b.PlayerID()).ID()
					g.GetCombat().AddAttacker(current, a.PlayerID())
					g.FindPermanent(current).GrantBaseAttr(core.Flying)
					want = 0
				case "phase", "phase-cycle":
					phased := g.PhaseOut(current)
					want = 0
					if scenario == "phase-cycle" {
						g.PhaseIn(phased)
						g.GetCombat().AddAttacker(current, a.PlayerID())
						want = tc.amount
					}
				case "combat-removal":
					g.RemoveFromCombat(current)
					if tc.name != "Grapeshot Catapult" {
						want = 0
					}
				case "flying-removal":
					g.FindPermanent(current).RevokeBaseAttr(core.Flying)
					if tc.name == "Grapeshot Catapult" {
						want = 0
					}
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: current, ControllerID: a.PlayerID(), Duration: core.Indefinite})
					g.GetCombat().AddAttacker(current, b.PlayerID())
				case "shroud":
					g.FindPermanent(current).GrantBaseAttr(core.Shroud)
					want = 0
				case "source-return":
					g.BouncePermanentToHand(g.FindPermanent(source.ID()))
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
				case "source-phase":
					g.PhaseOut(source.ID())
				case "prevention":
					g.AddPreventionShield(current, 1)
					want--
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
					want *= 2
				}
				clone := g.Clone()
				g.ResolveStack()
				if g.FindPermanentIncludingPhased(current).Damage != want || clone.FindPermanentIncludingPhased(current).Damage != 0 {
					t.Fatalf("damage or clone isolation: got %d want %d", g.FindPermanentIncludingPhased(current).Damage, want)
				}
				wantLife := 20
				if scenario == "source-return" {
					wantLife += want
				}
				if a.Life() != wantLife || b.Life() != 20 {
					t.Fatal("damage redirected to a player")
				}
				clone.ResolveStack()
				if clone.FindPermanentIncludingPhased(current).Damage != want {
					t.Fatal("clone lost recipient")
				}
			})
		}
	}
}

func TestRONPermanentDamageSpellAndTriggerBindings(t *testing.T) {
	for _, context := range []string{"spell", "mode", "trigger"} {
		t.Run(context, func(t *testing.T) {
			body := `Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([SpellDealsDamage(ThisSpell,Integer(2),Permanent(Ref_TargetPermanent))]))`
			rule, types := `SpellActions(`+body+`)`, "Instant"
			if context == "mode" {
				rule = `SpellActions(Modal_ChooseOne([` + body + `,` + strings.ReplaceAll(body, "Integer(2)", "Integer(3)") + `]))`
			}
			if context == "trigger" {
				types = "Artifact"
				rule = `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),` + strings.ReplaceAll(strings.ReplaceAll(body, "SpellDealsDamage", "PermanentDealsDamage"), "ThisSpell", "ThisPermanent") + `)`
			}
			factory := permanentScopeFactory(t, types, rule)
			g, a, b := ronModalGame(0)
			left := g.PutOnBattlefield(repositoryCard(t, "LEA", "Wall of Stone"), a.PlayerID())
			right := g.PutOnBattlefield(repositoryCard(t, "LEA", "Wall of Stone"), b.PlayerID())
			card := factory()
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			if context == "trigger" {
				a.targets = []uuid.UUID{left.ID()}
				source := g.PutOnBattlefield(card, a.PlayerID())
				g.PutTriggersOnStack()
				g.RemoveFromBattlefield(source)
			} else {
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{left.ID()}, 0); err != nil {
					t.Fatal(err)
				}
			}
			a.targets = []uuid.UUID{right.ID()}
			copied := g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), true)
			if copied == nil || len(copied.Targets) != 1 || copied.Targets[0] != right.ID() {
				t.Fatal("retargeted copy failed")
			}
			g.ResolveStack()
			if left.Damage != 2 || right.Damage != 2 {
				t.Fatal("copied binding did not select its own recipient")
			}
			if context != "trigger" {
				for _, target := range []uuid.UUID{left.ID(), right.ID()} {
					next := factory()
					next.SetOwner(a.PlayerID())
					a.AddToHand(next)
					if err := g.CastSpellByID(a.PlayerID(), next.ID(), []uuid.UUID{target}, 0); err != nil {
						t.Fatal(err)
					}
				}
				g.ResolveStack()
				if left.Damage != 4 || right.Damage != 4 {
					t.Fatal("factory instances shared chosen state")
				}
			}
		})
	}
}

func TestRONPermanentDamageDeclarations(t *testing.T) {
	rule := `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([SpellDealsDamage(ThisSpell,Integer(2),Permanent(Ref_TargetPermanent))])))`
	factory := permanentScopeFactory(t, "Instant", rule)
	first, second := factory(), factory()
	left, right := first.Abilities()[0].(*mage.ActionDefinition), second.Abilities()[0].(*mage.ActionDefinition)
	if left.Targets()[0] == right.Targets()[0] {
		t.Fatal("shared target declaration")
	}
	for _, card := range []mage.Card{first, second} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	missing := mage.NewInstant("Missing", "", mage.WithAbility(mage.NewAction(mage.ActionSpell, mage.WithEffects(left.Effects()...))))
	if mage.ValidateCardDefinition(missing.Name(), missing) == nil {
		t.Fatal("undeclared typed read accepted")
	}
	src := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[` + rule + `])`
	report, err := InspectFS(fstest.MapFS{"damage.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	seen := false
	for _, entry := range report.Signatures {
		if entry.Name == "Ref_TargetPermanent" {
			seen = true
			if entry.Role != "permanent_reference" {
				t.Fatalf("recipient operand role = %s", entry.Role)
			}
		}
	}
	if !seen {
		t.Fatal("missing nested recipient reference")
	}
}

func TestRONPermanentDamageReplacementOrder(t *testing.T) {
	for mode := range 2 {
		a := mage.NewBasePlayer("A")
		b := &ronModePlayer{BasePlayer: mage.NewBasePlayer("B"), mode: mode}
		g := mage.NewGame(a, b)
		g.SetStep(core.PrecombatMain)
		g.SetTurn(1)
		victim := g.PutOnBattlefield(repositoryCard(t, "LEA", "Wall of Stone"), b.PlayerID())
		card := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([SpellDealsDamage(ThisSpell,Integer(4),Permanent(Ref_TargetPermanent))])))`)()
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{victim.ID()}, 0); err != nil {
			t.Fatal(err)
		}
		g.SetCreatureDamageRedirect(victim.ID(), b.PlayerID())
		g.AddPreventionShield(victim.ID(), 1)
		g.ResolveStack()
		if victim.Damage != 0 || b.Life() != 16+mode || b.prompts != 1 {
			t.Fatalf("replacement order: damage=%d life=%d prompts=%d", victim.Damage, b.Life(), b.prompts)
		}
	}
}

func TestRONPermanentDamageInvalid(t *testing.T) {
	for _, recipient := range []string{`Permanent`, `Permanent()`, `Permanent(Ref_TargetPermanent,Ref_TargetPermanent)`, `Permanent(Ref_TargetPermanent())`, `Permanent(Ref_TargetPlayer)`, `Permanent(Ref_AnyTarget)`, `Permanent(Ref_TargetPermanent1)`, `Permanent(ActionPermanent)`, `Permanent(ThisPermanent)`, `Permanent(HostPermanent)`, `SinglePermanent(Ref_TargetPermanent)`} {
		src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([SpellDealsDamage(ThisSpell,Integer(1),%s)])))])`, recipient)
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", recipient)
		}
	}
	for _, target := range []string{"", "TargetPlayer(AnyPlayer)", "AnyTarget"} {
		src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([%s],ActionList([SpellDealsDamage(ThisSpell,Integer(1),Permanent(Ref_TargetPermanent))])))])`, target)
		nodes, err := Parse(strings.NewReader(src), "unbound.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted unbound %s", target)
		}
	}
	for _, rule := range []string{
		`SpellActions(Modal_ChooseOne([Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([SpellDealsDamage(ThisSpell,Integer(1),Permanent(Ref_TargetPermanent))])),ActionList([SpellDealsDamage(ThisSpell,Integer(1),Permanent(Ref_TargetPermanent))])]))`,
		`PermanentLayerEffect(ThisPermanent,[SpellDealsDamage(ThisSpell,Integer(1),Permanent(Ref_TargetPermanent))])`,
		`Activated(Permanent(Ref_TargetPermanent),ActionList([]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid context",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "context.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}
