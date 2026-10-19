package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONDamageOnlyNativeDomainPlan(t *testing.T) {
	permanent := `CreateReplaceWouldDealDamageUntil(DamageWouldBeDealtByPermanentToRecipient(ThisPermanent, Player(You)), [DealToCreatureOrPlaneswalkerInstead(ThisPermanent)], UntilEndOfTurn)`
	optional := `CreateReplaceWouldDealDamageUntil(DamageWouldBeDealtToARecipient(APermanent(IsCardtype(Creature))), [MayActions([DealToPlayerInstead(You)])], UntilEndOfTurn)`
	spell := `CreateReplaceWouldDealDamageUntil(DamageWouldBeDealtBySpell(Ref_TargetSpell), [DealToPlayerInstead(WouldDealDamage_ControllerOfDamageSource)], UntilEndOfTurn)`
	ctx := scope{resolving: true, permanent: true, spellTarget: &spellTargetPlan{name: "Ref_TargetSpell"}}
	for _, text := range []string{permanent, optional, spell} {
		nodes, err := Parse(strings.NewReader(text), "plan.ron")
		if err != nil {
			t.Fatal(err)
		}
		plan, err := compileTemporaryDamageRedirectPlan(nodes[0], ctx)
		if err != nil {
			t.Fatal(err)
		}
		var previous mage.Effect
		for range 2 {
			target := mage.DeclareSpellTarget("spell")
			bindings := instanceBindings{spells: map[*spellTargetPlan]*mage.SpellTargetBinding{ctx.spellTarget: target}}
			effect := plan.build(bindings)
			if effect == previous {
				t.Fatal("plan shares effect instances")
			}
			previous = effect
			action := mage.NewAction(mage.ActionActivated, mage.WithEffects(effect))
			if text == spell {
				action = mage.Activate(mage.Costs(), effect, mage.WithTargets(target))
			}
			card := mage.NewArtifact("Plan", "", mage.WithAbility(action))
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			if text == spell {
				detached := mage.NewArtifact("Detached", "", mage.WithAbility(mage.Activate(mage.Costs(), effect, mage.WithTargets(mage.DeclareSpellTarget("spell")))))
				if err := mage.ValidateCardDefinition(detached.Name(), detached); err == nil {
					t.Fatal("spell binding lost instance identity")
				}
			}
		}
	}
	for _, text := range []string{`Unknown(A,B,C)`, `CreateReplaceWouldDealDamageUntil()`, strings.Replace(permanent, "UntilEndOfTurn", "UntilEndOfCombat", 1), strings.Replace(permanent, "ThisPermanent", "Ref_TargetPermanent_2", 1), strings.Replace(optional, "[MayActions([DealToPlayerInstead(You)])]", "[]", 1), strings.Replace(spell, "Ref_TargetSpell", "Ref_TargetSpell_2", 1)} {
		nodes, err := Parse(strings.NewReader(text), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := compileTemporaryDamageRedirectPlan(nodes[0], ctx); err == nil {
			t.Fatal("accepted", text)
		}
	}
	for _, text := range []string{permanent, optional, spell} {
		nodes, _ := Parse(strings.NewReader(text), "context.ron")
		for _, bad := range []scope{{}, {resolving: true, enteringSource: true}, {resolving: true, drawReplacementBody: true}} {
			if _, err := compileTemporaryDamageRedirectPlan(nodes[0], bad); err == nil {
				t.Fatal("accepted invalid context", text)
			}
		}
	}
}

func TestRONDamageOnlyNativeDomain(t *testing.T) {
	for _, rule := range []string{"Blood of the Martyr", "Shimian Night Stalker", "Reverberation"} {
		for _, destroy := range []string{"Stone Rain", "Wrath of God"} {
			t.Run(rule+"/"+destroy, func(t *testing.T) {
				a := &ronPlayerRedirectChooser{BasePlayer: mage.NewBasePlayer("A"), accept: true}
				b := mage.NewBasePlayer("B")
				g := mage.NewGame(a, b)
				g.SetStep(core.PrecombatMain)
				g.SetTurn(1)
				subject := g.PutOnBattlefield(mage.NewLand("Subject"), b.PlayerID())
				victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 2, 20), a.PlayerID())
				attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "{R}", 1, 20), b.PlayerID())
				var source *mage.Permanent
				if rule == "Shimian Night Stalker" {
					source = g.PutOnBattlefield(repositoryCard(t, "LEG", rule), a.PlayerID())
					source.RevokeBaseAttr(core.AttrSummonSick)
					attacker.RevokeBaseAttr(core.AttrSummonSick)
					g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{attacker.ID()})
					a.ManaPool().Add(core.Black, 1)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{attacker.ID()}); err != nil {
						t.Fatal(err)
					}
					g.ResolveStack()
				}
				if destroy == "Wrath of God" {
					subject = victim
				}
				var spell *mage.StackObject
				if destroy == "Stone Rain" {
					spell = castRONSpell(t, g, a.BasePlayer, repositoryCard(t, "LEA", destroy), subject.ID())
				} else {
					spell = castRONSpell(t, g, a.BasePlayer, repositoryCard(t, "LEA", destroy))
				}
				switch rule {
				case "Blood of the Martyr":
					castRONSpell(t, g, a.BasePlayer, repositoryCard(t, "DRK", rule))
					g.ResolveTopOfStack()
				case "Reverberation":
					castRONSpell(t, g, a.BasePlayer, repositoryCard(t, "LEG", rule), spell.ID)
					g.ResolveTopOfStack()
				}
				instructions, known := mage.InspectDirectDestruction(spell.Effects[0])
				if !known || len(instructions) != 1 {
					t.Fatal("missing destruction plan")
				}
				instruction := instructions[0]
				var visits []mage.PermanentVisit
				if instruction.Subject != nil {
					query, ok := mage.InspectPermanentTarget(instruction.Subject)
					if !ok {
						t.Fatal("missing target")
					}
					visits, known = query.Select(g, spell)
					if !known {
						t.Fatal("unknown target")
					}
				} else {
					query, ok := mage.InspectPermanentCollection(instruction.Subjects)
					if !ok {
						t.Fatal("missing collection")
					}
					visits = query.Select(g)
				}
				if len(visits) == 0 {
					t.Fatal("no subjects")
				}
				for _, game := range []*mage.Game{g.Clone(), g} {
					for range 2 {
						for _, visit := range visits {
							if got, known := game.InspectDestructionDestinations(visit, instruction.CannotRegenerate); !known || got != mage.DestructionToGraveyard {
								t.Fatalf("destination %v known %v", got, known)
							}
							if proceeds, known := game.InspectDestructionProposal(visit, instruction.CannotRegenerate); !known || !proceeds {
								t.Fatal("unknown proposal")
							}
						}
					}
					if a.offers != 0 || game.StackSize() != 1 {
						t.Fatal("inspection made a choice or changed stack")
					}
					if destroy == "Stone Rain" {
						switch rule {
						case "Blood of the Martyr":
							game.DealDamageToPermanent(game.FindPermanent(victim.ID()), 1, attacker.ID())
							if game.GetPlayer(a.PlayerID()).Life() != 19 || game.FindPermanent(victim.ID()).Damage != 0 {
								t.Fatal("lost optional redirect")
							}
						case "Reverberation":
							game.DealDamageToPlayerFromSource(game.GetPlayer(b.PlayerID()), 1, game.CaptureSpellDamageSource(spell.ID))
							if game.GetPlayer(a.PlayerID()).Life() != 19 || game.GetPlayer(b.PlayerID()).Life() != 20 {
								t.Fatal("lost spell redirect")
							}
						case "Shimian Night Stalker":
							game.DealDamageToPlayer(game.GetPlayer(a.PlayerID()), 1, attacker.ID())
							if game.GetPlayer(a.PlayerID()).Life() != 20 || game.FindPermanent(source.ID()).Damage != 1 {
								t.Fatal("lost saved redirect")
							}
						}
					}
					game.ResolveStack()
					for _, visit := range visits {
						if game.FindPermanent(visit.ID) != nil {
							t.Fatal("native destruction disagrees")
						}
					}
				}
			})
		}
	}
}

func TestRONDamageOnlyNativeDomainProtection(t *testing.T) {
	for _, tc := range []struct {
		name  string
		color core.Color
	}{{"Black Knight", core.White}, {"White Knight", core.Black}} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			p := g.PutOnBattlefield(repositoryCard(t, "LEA", tc.name), a.PlayerID())
			cost := "{W}"
			if tc.color == core.Black {
				cost = "{B}"
			}
			source := g.PutOnBattlefield(mage.NewCreature("Source", cost, 1, 10), b.PlayerID())
			g.DealDamageToPermanent(p, 1, source.ID())
			if p.Damage != 0 {
				t.Fatal("protection lost native prevention")
			}
			spell := castRONSpell(t, g, a, repositoryCard(t, "LEA", "Wrath of God"))
			instructions, _ := mage.InspectDirectDestruction(spell.Effects[0])
			query, ok := mage.InspectPermanentCollection(instructions[0].Subjects)
			if !ok {
				t.Fatal("missing collection")
			}
			for _, game := range []*mage.Game{g.Clone(), g} {
				for _, visit := range query.Select(game) {
					if got, known := game.InspectDestructionDestinations(visit, true); !known || got != mage.DestructionToGraveyard {
						t.Fatal("protection obscures destruction")
					}
				}
				game.ResolveStack()
				if game.FindPermanent(p.ID()) != nil {
					t.Fatal("protection prevented untargeted destruction")
				}
			}
		})
	}
}
