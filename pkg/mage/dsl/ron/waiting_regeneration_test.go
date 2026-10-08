package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingRegeneration(t *testing.T) {
	for _, name := range []string{"Drudge Skeletons", "Death Ward", "Living Wall"} {
		for _, state := range []string{"live", "depart", "return", "phase", "cycle", "consume", "cleanup", "illegal"} {
			t.Run(name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, "LEB", name)
				card.SetOwner(a.PlayerID())
				subjectCard := card
				if name == "Death Ward" {
					subjectCard = mage.NewCreature("Subject", "{R}", 1, 20)
				}
				subject := g.PutOnBattlefield(subjectCard, a.PlayerID())
				saved := g.CaptureDamageSource(subject.ID())
				if name == "Death Ward" {
					a.AddToHand(card)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{subject.ID()}, 0); err == nil {
						t.Fatal("unpaid cast")
					}
					a.ManaPool().Add(core.White, 1)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{subject.ID()}, 0); err != nil {
						t.Fatal(err)
					}
				} else {
					if err := g.ActivateAbilityByIndex(a.PlayerID(), subject.ID(), 0, nil); err == nil {
						t.Fatal("unpaid activation")
					}
					a.ManaPool().Add(core.Black, 1)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), subject.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
				}
				if len(g.WaitingRegenerationPermanentSources()) != 0 {
					t.Fatal("shield before resolution")
				}
				if state == "illegal" {
					g.PhaseOut(subject.ID())
				}
				g.ResolveStack()
				switch state {
				case "depart", "return":
					g.RemoveFromBattlefield(g.FindPermanent(subject.ID()))
					if state == "return" {
						g.PutOnBattlefield(subjectCard, b.PlayerID())
					}
				case "phase":
					g.PhaseOut(subject.ID())
				case "cycle":
					g.PhaseIn(g.PhaseOut(subject.ID()))
				case "consume":
					g.DestroyPermanent(g.FindPermanent(subject.ID()))
				case "cleanup":
					g.GetEffects().ClearReplacementsEndOfTurn()
				}
				for _, game := range []*mage.Game{g, g.Clone()} {
					refs := game.WaitingRegenerationPermanentSources()
					want := 1
					if state == "illegal" || state == "consume" || state == "cleanup" {
						want = 0
					}
					if len(refs) != want {
						t.Fatalf("sources=%d want=%d", len(refs), want)
					}
					if want == 0 {
						continue
					}
					if !refs[0].SameObject(saved) {
						t.Fatal("wrong incarnation")
					}
					binding := mage.BindDamageSource("regeneration subject")
					body := mage.Sequence(mage.ChooseDamageSourceFromWaitingRegenerationReferences(mage.You, mage.DamageSourceFilter{}, binding), mage.DealDamageFrom(mage.FromDamageSource(binding), mage.You, mage.Fixed(1)))
					if err := mage.ApplyEffect(game, body, uuid.Nil, b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					if game.GetPlayer(b.PlayerID()).Life() != 19 {
						t.Fatal("retained source unavailable")
					}
				}
			})
		}
	}
}

func TestRONWaitingRegenerationProtection(t *testing.T) {
	for _, tc := range []struct{ set, name string }{
		{"LEB", "Circle of Protection: Black"}, {"LEA", "Circle of Protection: Blue"}, {"LEA", "Circle of Protection: Green"},
		{"LEA", "Circle of Protection: Red"}, {"LEA", "Circle of Protection: White"}, {"ATQ", "Circle of Protection: Artifacts"}, {"LEG", "Greater Realm of Preservation"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			record := staticBaseRecord(t, tc.set, tc.name)
			actions := record.Fields["Rules"].Args[0].Args[1].Args[0].Args
			plan, err := compileChosenDamageSourceChoice(actions[0])
			if err != nil {
				t.Fatal(err)
			}
			shield, err := compileExactSourcePrevention(actions[1], scope{chosenDamageSource: plan})
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, "LEB", "Living Wall"), a.PlayerID())
			if tc.name != "Circle of Protection: Artifacts" {
				color := core.Black
				switch tc.name {
				case "Circle of Protection: Blue":
					color = core.Blue
				case "Circle of Protection: Green":
					color = core.Green
				case "Circle of Protection: Red":
					color = core.Red
				case "Circle of Protection: White":
					color = core.White
				}
				if err := mage.ApplyEffect(g, mage.Until(core.Indefinite, mage.SetColors(mage.This, color)), source.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			a.ManaPool().Add(core.Black, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			saved := g.CaptureDamageSource(source.ID())
			g.RemoveFromBattlefield(source)
			for _, game := range []*mage.Game{g, g.Clone()} {
				binding := mage.BindDamageSource("chosen")
				bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}
				body := mage.Sequence(plan.waitingRegenerationChoice(bindings), shield(bindings), mage.DealDamageFrom(mage.FromDamageSource(binding), mage.You, mage.Fixed(2)))
				if err := mage.ValidateCardDefinition("Consumer", mage.NewArtifact("Consumer", "", mage.WithAbility(mage.Activate(mage.Costs(), body)))); err != nil {
					t.Fatal(err)
				}
				if err := mage.ApplyEffect(game, body, uuid.Nil, b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				if game.GetPlayer(b.PlayerID()).Life() != 20 {
					t.Fatal("choice or prevention failed")
				}
				game.DealDamageToPlayerFromSource(game.GetPlayer(b.PlayerID()), 1, saved)
				if game.GetPlayer(b.PlayerID()).Life() != 19 {
					t.Fatal("shield did not expire")
				}
			}
			if _, err := Compile(record); err == nil || !strings.Contains(err.Error(), "unsupported action (ChooseADamageSource)") {
				t.Fatal("incomplete source enumeration accepted", err)
			}
			invalid := plan.waitingRegenerationChoice(instanceBindings{})
			if mage.ValidateCardDefinition("Invalid", mage.NewArtifact("Invalid", "", mage.WithAbility(mage.Activate(mage.Costs(), invalid)))) == nil {
				t.Fatal("missing binding accepted")
			}
		})
	}
}

func TestRONWaitingRegenerationInvalid(t *testing.T) {
	for _, expression := range []string{`ChooseADamageSource`, `ChooseADamageSource()`, `ChooseADamageSource(IsColor(Red),You)`, `ChooseADamageSource(IsColor(Colorless))`, `ChooseADamageSource(And([]))`, `ChooseADamageSource(IsColor(Red()))`, `ChooseADamageSource(IsCardtype(Unknown))`} {
		nodes, err := Parse(strings.NewReader(expression), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := compileChosenDamageSourceChoice(nodes[0]); err == nil {
			t.Fatal("invalid choice", expression)
		}
	}
}
