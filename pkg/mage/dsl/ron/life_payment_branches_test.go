package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type lifeBranchClonePlayer struct {
	mage.Player
	accept bool
}

func (p *lifeBranchClonePlayer) ChooseMayAbility(string) bool { return p.accept }

func lifeBranchNode(t *testing.T, source string) Node {
	t.Helper()
	nodes, err := Parse(strings.NewReader(source), "life-branches.ron")
	if err != nil {
		t.Fatal(err)
	}
	return nodes[0]
}

func TestRONLifePaymentBranches(t *testing.T) {
	for _, payer := range []string{"MayCost(PayLife(Integer(%d)))", "PlayerMayCost(You,PayLife(Integer(%d)))", "PlayerMayCost(Ref_TargetPlayer,PayLife(Integer(%d)))"} {
		for _, tc := range []struct {
			name         string
			life, cost   int
			accept, paid bool
			offers       int
		}{
			{"paid", 20, 10, true, true, 1},
			{"declined", 20, 10, false, false, 1},
			{"short", 9, 10, true, false, 0},
			{"exact", 10, 10, true, true, 1},
			{"zero", 0, 0, true, true, 1},
			{"zero-declined", -2, 0, false, false, 1},
		} {
			for _, condition := range []string{"CostWasPaid", "CostWasntPaid"} {
				t.Run(payer+"/"+tc.name+"/"+condition, func(t *testing.T) {
					a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: tc.accept}
					b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: tc.accept}
					g := mage.NewGame(a, b)
					paying := a
					if strings.Contains(payer, "Ref_TargetPlayer") {
						paying = b
					}
					paying.SetLife(tc.life)
					paid, unpaid := `[GainLife(Integer(3))]`, `[GainLife(Integer(7))]`
					if condition == "CostWasntPaid" {
						paid, unpaid = unpaid, paid
					}
					body := `ActionList([` + fmt.Sprintf(payer, tc.cost) + `,IfElse(` + condition + `,` + paid + `,` + unpaid + `)])`
					var targets []uuid.UUID
					if strings.Contains(payer, "Ref_TargetPlayer") {
						body = `Targeted([TargetPlayer(AnyPlayer)],` + body + `)`
						targets = []uuid.UUID{b.PlayerID()}
					}
					card := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),`+body+`)`)()
					p := g.PutOnBattlefield(card, a.PlayerID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, targets); err != nil {
						t.Fatal(err)
					}
					if a.offers+b.offers != 0 || paying.Life() != tc.life {
						t.Fatal("payment occurred before resolution")
					}
					clone := g.Clone()
					clone.SetPlayerAt(0, &lifeBranchClonePlayer{Player: clone.GetPlayer(a.PlayerID()), accept: tc.accept})
					clone.SetPlayerAt(1, &lifeBranchClonePlayer{Player: clone.GetPlayer(b.PlayerID()), accept: tc.accept})
					g.ResolveStack()
					if clone.GetPlayer(paying.PlayerID()).Life() != tc.life {
						t.Fatal("payment changed pending clone")
					}
					clone.ResolveStack()
					wantA, wantB := 20, 20
					if paying == a {
						wantA = tc.life
					} else {
						wantB = tc.life
					}
					if tc.paid {
						if paying == a {
							wantA -= tc.cost
						} else {
							wantB -= tc.cost
						}
						wantA += 3
					} else {
						wantA += 7
					}
					for _, game := range []*mage.Game{g, clone} {
						if game.GetPlayer(a.PlayerID()).Life() != wantA || game.GetPlayer(b.PlayerID()).Life() != wantB {
							t.Fatalf("life %d/%d, want %d/%d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life(), wantA, wantB)
						}
					}
					if a.offers+b.offers != tc.offers {
						t.Fatalf("offers = %d, want %d", a.offers+b.offers, tc.offers)
					}
				})
			}
		}
	}
}

func TestRONLifePaymentBranchesNested(t *testing.T) {
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	g := mage.NewGame(a, mage.NewBasePlayer("B"))
	body := `MayCost(PayLife(Integer(10))),IfElse(CostWasPaid,[MayCost(PayLife(Integer(11))),IfElse(CostWasntPaid,[GainLife(Integer(3))],[GainLife(Integer(100))])],[GainLife(Integer(100))]),MayCost(PayLife(Integer(10))),IfElse(CostWasPaid,[GainLife(Integer(2))],[GainLife(Integer(100))])`
	p := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([`+body+`]))`)(), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 5 || a.offers != 2 {
		t.Fatal("nested or adjacent payment result escaped", a.Life(), a.offers)
	}
}

func TestRONLifePaymentBranchesBronzeTabletFragment(t *testing.T) {
	for _, state := range []string{"paid", "declined", "short", "source-return", "source-phase", "target-return", "target-phase", "target-cycle", "control", "copy"} {
		t.Run(state, func(t *testing.T) {
			record := staticBaseRecord(t, "ATQ", "Bronze Tablet")
			rule := record.Fields["Rules"].Args[2]
			payment := rule.Args[1].Args[1].Args[0].Args[1]
			payment.Args[0] = lifeBranchNode(t, `ControllerOfPermanent(Ref_TargetPermanent)`)
			continuation := rule.Args[1].Args[1].Args[0].Args[2]
			continuation.Args[1] = lifeBranchNode(t, `[GainLife(Integer(1))]`)
			continuation.Args[2] = lifeBranchNode(t, `[GainLife(Integer(2))]`)
			rule.Args[1].Args[1] = lifeBranchNode(t, `ActionList([])`)
			rule.Args[1].Args[1].Args[0].Args = []Node{payment, continuation}
			rules := record.Fields["Rules"]
			rules.Args = []Node{rule}
			record.Fields["Rules"] = rules
			factory, err := Compile(record)
			if err != nil {
				t.Fatal(err)
			}
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: state != "declined"}
			g := mage.NewGame(a, b)
			card := factory()
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			targetCard := mage.NewArtifact("Target", "")
			targetCard.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(targetCard, b.PlayerID())
			a.ManaPool().Add(core.Colorless, 4)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			if a.offers+b.offers != 0 || !source.Tapped || a.ManaPool().TotalMana() != 0 {
				t.Fatal("activation cost or payment timing")
			}
			switch state {
			case "short":
				b.SetLife(9)
			case "source-return":
				g.RemoveFromBattlefield(source)
				g.PutOnBattlefield(card, b.PlayerID())
			case "source-phase":
				g.PhaseOut(source.ID())
			case "target-return":
				g.RemoveFromBattlefield(target)
				g.PutOnBattlefield(targetCard, b.PlayerID())
			case "target-phase":
				g.PhaseOut(target.ID())
			case "target-cycle":
				g.PhaseIn(g.PhaseOut(target.ID()))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: target.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			g.ResolveStack()
			wantA, wantB, offers := 21, 10, 1
			switch state {
			case "declined":
				wantA, wantB = 22, 20
			case "short":
				wantA, wantB, offers = 22, 9, 0
			case "target-return", "target-phase":
				wantA, wantB, offers = 20, 20, 0
			case "copy":
				wantA, wantB, offers = 22, 0, 2
			}
			if a.Life() != wantA || b.Life() != wantB || b.offers != offers || a.offers != 0 {
				t.Fatal("fragment payment, saved controller or legality", a.Life(), b.Life(), a.offers, b.offers)
			}
		})
	}
}

func TestRONLifePaymentBranchesInvalid(t *testing.T) {
	valid := `MayCost(PayLife(Integer(10))),IfElse(CostWasPaid,[GainLife(Integer(1))],[GainLife(Integer(2))])`
	for _, bad := range []string{`PayLife`, `PayLife()`, `PayLife(Integer(-1))`, `PayLife(ValueX)`, `PayLife(Integer())`, `PayLife(10)`, `PayLife(Integer(1),Integer(2))`, `PayLife(LifeTotalOfPlayer(You))`, `Or([PayLife(Integer(1)),PayMana([])])`} {
		source := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([` + strings.Replace(valid, `PayLife(Integer(10))`, bad, 1) + `]))])`
		if _, err := Compile(lifeBranchNode(t, source)); err == nil {
			t.Fatal("accepted", bad)
		}
	}
	for _, bad := range []string{strings.Replace(valid, "MayCost", "MustCost", 1), strings.Replace(valid, "CostWasPaid", "CostWasPaid()", 1), strings.Replace(valid, `[GainLife(Integer(1))]`, `[]`, 1), strings.Replace(valid, `[GainLife(Integer(2))]`, `[]`, 1), valid + `,If(CostWasPaid,[GainLife(Integer(3))])`, strings.Replace(valid, `MayCost(PayLife(Integer(10)))`, `PlayerMayCost(Ref_TargetPlayer,PayLife(Integer(10)))`, 1)} {
		if _, err := Compile(lifeBranchNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+bad+`]))])`)); err == nil {
			t.Fatal("accepted", bad)
		}
	}
}

func TestRONLifePaymentBranchesInstances(t *testing.T) {
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: false}
	g := mage.NewGame(a, b)
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerMayCost(Ref_TargetPlayer,PayLife(Integer(10))),IfElse(CostWasPaid,[GainLife(Integer(1))],[GainLife(Integer(2))])])))`)
	for _, players := range [][2]*ronPaymentPlayer{{a, b}, {b, a}} {
		source := g.PutOnBattlefield(factory(), players[0].PlayerID())
		if err := g.ActivateAbilityByIndex(players[0].PlayerID(), source.ID(), 0, []uuid.UUID{players[1].PlayerID()}); err != nil {
			t.Fatal(err)
		}
	}
	g.ResolveStack()
	if a.Life() != 12 || b.Life() != 21 || a.offers != 1 || b.offers != 1 {
		t.Fatal("factory target or payer binding shared", a.Life(), b.Life(), a.offers, b.offers)
	}
}

func TestRONLifePaymentBranchesSpellCopy(t *testing.T) {
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: false}
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	card := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerMayCost(Ref_TargetPlayer,PayLife(Integer(10))),IfElse(CostWasPaid,[GainLife(Integer(1))],[GainLife(Integer(2))])])))`)()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{a.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	copied := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy failed")
	}
	copied.Targets[0] = b.PlayerID()
	g.ResolveStack()
	if a.Life() != 11 || b.Life() != 22 || a.offers != 1 || b.offers != 1 {
		t.Fatal("copy payer, controller or result shared", a.Life(), b.Life(), a.offers, b.offers)
	}
}

func TestRONLifePaymentBranchesDelayed(t *testing.T) {
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	card := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([MayCost(PayLife(Integer(10))),IfElse(CostWasPaid,[GainLife(Integer(1))],[GainLife(Integer(2))])]))]))`)()
	source := g.PutOnBattlefield(card, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.RemoveFromBattlefield(source)
	for range 2 {
		g.WaitingDelayedPermanentSources()
	}
	if a.offers != 0 || a.Life() != 20 {
		t.Fatal("delay query paid cost")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if a.offers != 0 {
		t.Fatal("payment occurred at trigger detection")
	}
	g.ResolveStack()
	if a.Life() != 11 || a.offers != 1 {
		t.Fatal("delayed payment lost saved player", a.Life(), a.offers)
	}
}
