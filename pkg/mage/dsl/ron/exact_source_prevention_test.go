package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONExactSourcePreventionLowering(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"LEA", "Circle of Protection: Blue"}, {"LEA", "Circle of Protection: Green"}, {"LEA", "Circle of Protection: Red"}, {"LEA", "Circle of Protection: White"}, {"LEB", "Circle of Protection: Black"}, {"ATQ", "Circle of Protection: Artifacts"}, {"LEG", "Greater Realm of Preservation"}} {
		t.Run(tc.name, func(t *testing.T) {
			record := staticBaseRecord(t, tc.set, tc.name)
			actions := record.Fields["Rules"].Args[0].Args[1].Args[0].Args
			filter, err := compileChosenDamageSourceFilter(actions[0].Args[0])
			if err != nil {
				t.Fatal(err)
			}
			plan := &chosenDamageSourcePlan{filter: filter}
			build, err := compileExactSourcePrevention(actions[1], scope{chosenDamageSource: plan})
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			binding := mage.BindDamageSource("selected")
			bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}
			shield := build(bindings)
			sourceCard := repositoryCard(t, "LEA", "Prodigal Sorcerer")
			if tc.name == "Circle of Protection: Green" {
				sourceCard = mage.NewCreature("Green source", "{G}", 1, 10)
			}
			if tc.name == "Circle of Protection: Red" {
				sourceCard = repositoryCard(t, "LEA", "Orcish Artillery")
			}
			if tc.name == "Circle of Protection: White" {
				sourceCard = repositoryCard(t, "LEA", "Samite Healer")
			}
			if tc.name == "Circle of Protection: Black" {
				sourceCard = mage.NewCreature("Black source", "{B}", 1, 10)
			}
			if tc.name == "Circle of Protection: Artifacts" {
				sourceCard = repositoryCard(t, "ATQ", "Grapeshot Catapult")
			}
			if tc.name == "Greater Realm of Preservation" {
				sourceCard = repositoryCard(t, "LEA", "Orcish Artillery")
			}
			p := g.PutOnBattlefield(sourceCard, a.PlayerID())
			body := mage.Sequence(mage.SnapshotDamageSource(mage.This, binding), shield)
			if err := mage.ValidateCardDefinition("Fixture", mage.NewArtifact("Fixture", "", mage.WithAbility(mage.Activate(mage.Costs(), body)))); err != nil {
				t.Fatal(err)
			}
			if err := mage.ApplyEffect(g, body, p.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				recipient := game.GetPlayer(b.PlayerID())
				game.DealDamageToPlayer(recipient, 2, p.ID())
				if recipient.Life() != 20 {
					t.Fatal("lowered shield did not prevent damage")
				}
				game.DealDamageToPlayer(recipient, 1, p.ID())
				if recipient.Life() != 19 {
					t.Fatal("shield did not expire")
				}
			}
			if _, err := Compile(record); err == nil || !strings.Contains(err.Error(), "ChooseADamageSource") {
				t.Fatalf("incomplete enumeration was published: %v", err)
			}
		})
	}
}

func TestRONExactSourcePreventionInvalid(t *testing.T) {
	for _, text := range []string{
		`CreateFuturePreventDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(TheChosenDamageSource,Player(You)),[PreventThatDamage])`,
		`CreateFuturePreventDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(TheChosenDamageSource(),Player(You)),[PreventThatDamage])`,
		`CreateFuturePreventDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(ThisPermanent,Player(You)),[PreventThatDamage])`,
		`CreateFuturePreventDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(TheChosenDamageSource,Permanent(ThisPermanent)),[PreventThatDamage])`,
		`CreateFuturePreventDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(TheChosenDamageSource,Player(Ref_TargetPlayer)),[PreventThatDamage])`,
		`CreateFuturePreventDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(TheChosenDamageSource,Player(You)),[PreventThatDamage()])`,
		`CreateFuturePreventDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(TheChosenDamageSource,Player(You)),[PreventThatDamage,GainLife(Integer(1))])`,
	} {
		nodes, err := Parse(strings.NewReader(text), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		ctx := scope{chosenDamageSource: &chosenDamageSourcePlan{}}
		if text == `CreateFuturePreventDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(TheChosenDamageSource,Player(You)),[PreventThatDamage])` {
			ctx.chosenDamageSource = nil
		}
		if _, err := compileExactSourcePrevention(nodes[0], ctx); err == nil {
			t.Fatalf("accepted %s", text)
		}
	}
}

func TestRONExactSourcePreventionGameplay(t *testing.T) {
	for _, name := range []string{"Lightning Bolt", "Prodigal Sorcerer", "Orcish Artillery"} {
		for _, state := range []string{"live", "depart", "return", "phase", "copy"} {
			t.Run(name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				record := staticBaseRecord(t, "LEA", "Circle of Protection: Red")
				if name == "Prodigal Sorcerer" {
					record = staticBaseRecord(t, "LEA", "Circle of Protection: Blue")
				}
				actions := record.Fields["Rules"].Args[0].Args[1].Args[0].Args
				filter, err := compileChosenDamageSourceFilter(actions[0].Args[0])
				if err != nil {
					t.Fatal(err)
				}
				plan := &chosenDamageSourcePlan{filter: filter}
				build, err := compileExactSourcePrevention(actions[1], scope{chosenDamageSource: plan})
				if err != nil {
					t.Fatal(err)
				}
				card := repositoryCard(t, "LEA", name)
				card.SetOwner(a.PlayerID())
				binding := mage.BindDamageSource("chosen")
				bindings := instanceBindings{chosenDamageSources: map[*chosenDamageSourcePlan]*mage.DamageSourceBinding{plan: binding}}
				var sourceID uuid.UUID
				var save mage.ResolvingEffect
				if name == "Lightning Bolt" {
					castRONSpell(t, g, a, card, b.PlayerID())
					save = mage.ChooseDamageSourceFromLiveObjects(mage.You, mage.DamageSourceHasType(core.TypeInstant), binding)
				} else {
					p := g.PutOnBattlefield(card, a.PlayerID())
					sourceID = p.ID()
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err == nil || p.Tapped || g.StackSize() != 0 {
						t.Fatal("sick activation changed state")
					}
					g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
						t.Fatal(err)
					}
					save = mage.SnapshotDamageSource(mage.This, binding)
				}
				body := mage.Sequence(save, build(bindings))
				if err := mage.ApplyEffect(g, body, sourceID, b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				if name != "Lightning Bolt" {
					switch state {
					case "depart", "return":
						g.BouncePermanentToHand(g.FindPermanent(sourceID))
						if state == "return" {
							a.RemoveFromHand(card.ID())
							g.PutOnBattlefield(card, a.PlayerID())
							g.DealDamageToPlayer(b, 1, sourceID)
						}
					case "phase":
						g.PhaseOut(sourceID)
					}
				}
				clone := g.Clone()
				for _, game := range []*mage.Game{g, clone} {
					before := game.GetPlayer(b.PlayerID()).Life()
					game.ResolveStack()
					want := before
					if state == "copy" {
						amount := 1
						if name == "Orcish Artillery" {
							amount = 2
						}
						if name == "Lightning Bolt" {
							amount = 3
						}
						want -= amount
					}
					if game.GetPlayer(b.PlayerID()).Life() != want {
						t.Fatalf("life=%d want=%d", game.GetPlayer(b.PlayerID()).Life(), want)
					}
					own := 20
					if name == "Orcish Artillery" {
						own = 17
						if state == "copy" {
							own = 14
						}
					}
					if game.GetPlayer(a.PlayerID()).Life() != own {
						t.Fatal("shield changed the other recipient")
					}
				}
			})
		}
	}
}
