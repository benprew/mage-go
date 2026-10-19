package ron

import (
	"slices"
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONPlayerDamageLimitDomainPlan(t *testing.T) {
	for _, text := range []string{ronStaticDamageCap, `DamageThatWouldReduceLifeTotalToLessThanNumberReducesItToThatNumberInstead(Integer(1))`} {
		nodes, err := Parse(strings.NewReader(text), "plan.ron")
		if err != nil {
			t.Fatal(err)
		}
		var build optionFactory
		if nodes[0].Name == "ReplaceWouldDealDamage" {
			plan, err := compileStaticDamageCapPlan(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			build = plan.build
		} else {
			plan, err := compileDamageLifeFloorPlan(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			build = plan.build
		}
		first := mage.NewArtifact("First", "", build())
		second := mage.NewArtifact("Second", "", build())
		if first.Abilities()[0].AbilityID() == second.Abilities()[0].AbilityID() {
			t.Fatal("plan shared an ability instance")
		}
		for _, card := range []mage.Card{first, second} {
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, text := range []string{`Unknown(Integer(1))`, `DamageThatWouldReduceLifeTotalToLessThanNumberReducesItToThatNumberInstead()`, `DamageThatWouldReduceLifeTotalToLessThanNumberReducesItToThatNumberInstead(Integer(0))`} {
		nodes, err := Parse(strings.NewReader(text), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := compileDamageLifeFloorPlan(nodes[0]); err == nil {
			t.Fatalf("accepted %s", text)
		}
	}
	for _, text := range []string{`Unknown(A,B)`, `ReplaceWouldDealDamage()`, strings.Replace(ronStaticDamageCap, "Player(You)", "Player(Opponent)", 1)} {
		nodes, err := Parse(strings.NewReader(text), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := compileStaticDamageCapPlan(nodes[0]); err == nil {
			t.Fatalf("accepted %s", text)
		}
	}
}

func TestRONPlayerDamageLimitDomain(t *testing.T) {
	for _, limit := range []struct{ name, set string }{{"Ali from Cairo", "ARN"}, {"Forethought Amulet", "LEG"}} {
		for _, name := range []string{"Stone Rain", "Wrath of God"} {
			for _, state := range []string{"live", "source left", "source phase", "phase cycle", "regeneration"} {
				t.Run(limit.name+"/"+name+"/"+state, func(t *testing.T) {
					g, a, b := ronGame()
					source := g.PutOnBattlefield(repositoryCard(t, limit.set, limit.name), a.PlayerID())
					var card mage.Card = mage.NewLand("Subject")
					if name == "Wrath of God" {
						card = mage.NewCreature("Subject", "", 3, 3)
					}
					card.SetOwner(b.PlayerID())
					p := g.PutOnBattlefield(card, b.PlayerID())
					switch state {
					case "source left":
						g.BouncePermanentToHand(source)
					case "source phase":
						g.PhaseOut(source.ID())
					case "phase cycle":
						g.PhaseIn(g.PhaseOut(source.ID()))
					case "regeneration":
						g.AddRegenerationShield(p.ID())
					}
					var spell *mage.StackObject
					if name == "Stone Rain" {
						spell = castRONSpell(t, g, a, repositoryCard(t, "LEA", name), p.ID())
					} else {
						spell = castRONSpell(t, g, a, repositoryCard(t, "LEA", name))
					}
					instructions, known := mage.InspectDirectDestruction(spell.Effects[0])
					if !known || len(instructions) != 1 {
						t.Fatal("missing direct destruction")
					}
					instruction := instructions[0]
					var visits []mage.PermanentVisit
					if instruction.Subject != nil {
						query, ok := mage.InspectPermanentTarget(instruction.Subject)
						if !ok {
							t.Fatal("missing target plan")
						}
						visits, known = query.Select(g, spell)
						if !known {
							t.Fatal("unknown target legality")
						}
					} else {
						query, ok := mage.InspectPermanentCollection(instruction.Subjects)
						if !ok {
							t.Fatal("missing collection plan")
						}
						visits = query.Select(g)
					}
					if !slices.ContainsFunc(visits, func(v mage.PermanentVisit) bool { return v.ID == p.ID() }) {
						t.Fatal("missing subject visit")
					}
					for _, game := range []*mage.Game{g.Clone(), g} {
						for _, visit := range visits {
							want := mage.DestructionToGraveyard
							if state == "regeneration" && !instruction.CannotRegenerate && visit.ID == p.ID() {
								want = mage.DestructionNoMove
							}
							for range 2 {
								if got, known := game.InspectDestructionDestinations(visit, instruction.CannotRegenerate); !known || got != want {
									t.Fatalf("destinations %v known %v, want %v", got, known, want)
								}
								if proceeds, known := game.InspectDestructionProposal(visit, instruction.CannotRegenerate); !known || proceeds != (want == mage.DestructionToGraveyard) {
									t.Fatal("proposal disagrees")
								}
							}
						}
						if game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 20 || game.StackSize() != 1 || game.FindPermanent(p.ID()).Tapped || game.HasRegenerationShield(p.ID()) != (state == "regeneration") {
							t.Fatal("inspection changed state")
						}
						game.ResolveStack()
						if state == "regeneration" && !instruction.CannotRegenerate {
							live := game.FindPermanent(p.ID())
							if live == nil || !live.Tapped || game.HasRegenerationShield(p.ID()) {
								t.Fatal("native regeneration disagrees")
							}
						} else if game.FindPermanent(p.ID()) != nil || !slices.ContainsFunc(game.GetPlayer(b.PlayerID()).Graveyard(), func(c mage.Card) bool { return c.ID() == card.ID() }) {
							t.Fatal("native destruction disagrees")
						}
						if name == "Wrath of God" && limit.name == "Ali from Cairo" && state != "source left" && state != "source phase" && game.FindPermanent(source.ID()) != nil {
							t.Fatal("simultaneous destruction missed the limit source")
						}
					}
				})
			}
		}
	}
}

func TestRONPlayerDamageLimitDomainRetainsDamage(t *testing.T) {
	for _, tc := range []struct {
		name, set  string
		life, gain int
	}{{"Ali from Cairo", "ARN", 1, 25}, {"Forethought Amulet", "LEG", 18, 2}} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
			land := g.PutOnBattlefield(mage.NewLand("Subject"), b.PlayerID())
			spell := castRONSpell(t, g, a, repositoryCard(t, "LEA", "Stone Rain"), land.ID())
			instructions, ok := mage.InspectDirectDestruction(spell.Effects[0])
			if !ok || len(instructions) != 1 {
				t.Fatal("missing destruction instruction")
			}
			query, ok := mage.InspectPermanentTarget(instructions[0].Subject)
			if !ok {
				t.Fatal("missing target plan")
			}
			visits, known := query.Select(g, spell)
			if !known || len(visits) != 1 {
				t.Fatal("missing subject visit")
			}
			for _, game := range []*mage.Game{g.Clone(), g} {
				if got, known := game.InspectDestructionDestinations(visits[0], false); !known || got != mage.DestructionToGraveyard {
					t.Fatal("damage limit obscures destruction")
				}
				game.ResolveStack()
				damage := game.PutOnBattlefield(mage.NewInstant("Damage source", "", mage.WithKeyword(core.Lifelink)), b.PlayerID())
				game.DealDamageToPlayer(game.GetPlayer(a.PlayerID()), 25, damage.ID())
				if game.GetPlayer(a.PlayerID()).Life() != tc.life || game.GetPlayer(b.PlayerID()).Life() != 20+tc.gain {
					t.Fatal("inspection changed damage or its life results")
				}
			}
		})
	}
}
