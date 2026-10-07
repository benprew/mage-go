package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

type ronActionPaymentPlayer struct {
	*ronPaymentPlayer
	target uuid.UUID
}

func (p *ronActionPaymentPlayer) ChooseTargets(possible []uuid.UUID, minimum, maximum int, g *mage.Game) []uuid.UUID {
	for _, id := range possible {
		if id == p.target {
			return []uuid.UUID{id}
		}
	}
	return p.BasePlayer.ChooseTargets(possible, minimum, maximum, g)
}

func TestRONActionPaymentRealCards(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"FEM", "Delif's Cone"}, {"FEM", "Farrel's Zealot"}, {"LEG", "Floral Spuzzem"}} {
		for _, state := range []string{"accept", "decline", "prevent", "regenerate", "indestructible", "shroud", "returned", "phased", "phase-cycle", "source-left", "control"} {
			t.Run(tc.name+"/"+state, func(t *testing.T) {
				a := &ronActionPaymentPlayer{ronPaymentPlayer: &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: state != "decline"}}
				b := mage.NewBasePlayer("B")
				g := mage.NewGame(a, b)
				card := repositoryCard(t, tc.set, tc.name)
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				attacker := source
				var targetCard mage.Card = mage.NewCreature("Recipient", "", 1, 8)
				if tc.name == "Floral Spuzzem" {
					targetCard = mage.NewArtifact("Recipient", "")
				}
				targetCard.SetOwner(b.PlayerID())
				target := g.PutOnBattlefield(targetCard, b.PlayerID())
				a.target = target.ID()
				if tc.name == "Delif's Cone" {
					attacker = g.PutOnBattlefield(mage.NewCreature("Attacker", "", 4, 8), a.PlayerID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err == nil || g.FindPermanent(source.ID()) == nil || g.FindPermanent(source.ID()).Tapped {
						t.Fatal("illegal cost changed Cone")
					}
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{attacker.ID()}); err != nil {
						t.Fatal(err)
					}
					if g.FindPermanent(source.ID()) != nil {
						t.Fatal("Cone not sacrificed")
					}
					g.ResolveStack()
				}
				g.GetCombat().AddAttacker(attacker.ID(), b.PlayerID())
				g.FireEvent(core.GameEvent{Type: core.EvtAttacksUnblocked, SourceID: attacker.ID(), PlayerID: a.PlayerID(), TargetID: b.PlayerID()})
				g.PutTriggersOnStack()
				if g.StackSize() != 1 || a.offers != 0 {
					t.Fatalf("timing: stack=%d offers=%d", g.StackSize(), a.offers)
				}
				if tc.name == "Delif's Cone" {
					target = attacker
					targetCard = attacker.Card
				}
				switch state {
				case "prevent":
					g.AddPreventionShield(target.ID(), 3)
				case "regenerate":
					g.AddRegenerationShield(target.ID())
				case "indestructible":
					g.MutablePermanent(target.ID()).GrantBaseAttr(core.Indestructible)
				case "shroud":
					g.MutablePermanent(target.ID()).GrantBaseAttr(core.Shroud)
				case "returned":
					g.BouncePermanentToHand(target)
					g.GetPlayer(targetCard.Owner()).RemoveFromHand(targetCard.ID())
					g.PutOnBattlefield(targetCard, target.ControllerID())
				case "phased":
					g.PhaseOut(target.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(target.ID()))
				case "source-left":
					g.BouncePermanentToHand(attacker)
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: attacker.ID(), ControllerID: b.PlayerID(), Duration: core.EndOfTurn})
				}
				clone := g.Clone()
				g.ResolveStack()
				legal := tc.name == "Delif's Cone" || state != "shroud" && state != "returned" && state != "phased"
				paid := legal && state != "decline" && (tc.name != "Floral Spuzzem" || state != "indestructible")
				wantLife := 20
				if paid && tc.name == "Delif's Cone" {
					wantLife = 24
				}
				if a.Life() != wantLife {
					t.Fatalf("life=%d want=%d", a.Life(), wantLife)
				}
				if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.StackSize() != 1 {
					t.Fatal("clone changed")
				}
				if tc.name == "Farrel's Zealot" && legal {
					want := 0
					if paid && state != "prevent" {
						want = 3
					}
					if g.FindPermanent(target.ID()).Damage != want {
						t.Fatal("damage payment")
					}
				}
				if tc.name == "Floral Spuzzem" && legal {
					survives := !paid || state == "regenerate"
					if (g.FindPermanent(target.ID()) != nil) != survives {
						t.Fatal("destruction payment")
					}
				}
				if state != "source-left" && state != "returned" && state != "phased" {
					g.ExecuteCombatDamage()
					want := 20
					if !paid {
						want -= attacker.CurrentPower(g)
					}
					if b.Life() != want {
						t.Fatalf("combat life=%d want=%d", b.Life(), want)
					}
				}
			})
		}
	}
}

func TestRONActionPaymentInvalid(t *testing.T) {
	for _, actions := range []string{
		`MayCost(GainLife()),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MayCost(HavePermanentDealDamage(ThisPermanent,Integer(3),Permanent(Ref_TargetPlayer))),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MayCost(DestroyPermanent(Trigger_ThatCreature)),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MayCost(GainLife(Integer(1))),GainLife(Integer(2)),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MayCost(GainLife(Integer(1))),If(CostWasPaid(),[GainLife(Integer(1))])`,
		`MayCost(GainLife(Integer(1))),If(CostWasPaid,[])`,
		`MustCost(GainLife(Integer(1))),If(CostWasPaid,[GainLife(Integer(1))])`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([` + actions + `]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid-action-payment.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted", actions)
		}
	}
}

func TestRONActionPaymentStrictLoad(t *testing.T) {
	files := fstest.MapFS{}
	for _, tc := range []struct{ set, name string }{{"FEM", "Delif's Cone"}, {"FEM", "Farrel's Zealot"}, {"LEG", "Floral Spuzzem"}} {
		n := hostUntapRecord(t, tc.set, tc.name)
		factory, err := Compile(n)
		if err != nil {
			t.Fatal(err)
		}
		first, second := factory(), factory()
		if first.ID() == second.ID() || first.Abilities()[0] == second.Abilities()[0] {
			t.Fatal("shared factory declarations")
		}
		raw, err := os.ReadFile("../../../../cards/mtgish/" + tc.set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		var data string
		for line := range strings.SplitSeq(string(raw), "\n") {
			if strings.Contains(line, `Name: "`+tc.name+`"`) {
				data = line
			}
		}
		files[tc.name+".ron"] = &fstest.MapFile{Data: []byte(data)}
		files[tc.name+"-reprint.ron"] = &fstest.MapFile{Data: []byte(data)}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil || len(registry.RegisteredCardNames()) != 3 {
		t.Fatal(err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 6 || report.UniqueCards != 3 {
		t.Fatal("reprint inventory", err)
	}
	found := false
	for _, signature := range report.Signatures {
		if signature.Name == "ThisPermanent" && signature.Role == "permanent_reference" && strings.Contains(signature.Context, "HavePermanentDealDamage") {
			found = true
		}
	}
	if !found {
		t.Fatal("missing payment source role")
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([MayCost(GainLife(Integer(1))),If(CostWasPaid,[])]))])`)}
	if LoadInto(registry, files, "*.ron") == nil || len(registry.RegisteredCardNames()) != 3 {
		t.Fatal("partial registry installed")
	}
}

func TestRONActionPaymentBranchesAndChooser(t *testing.T) {
	for _, continuation := range []struct {
		text     string
		gainPaid bool
	}{
		{`If(CostWasPaid,[GainLife(Integer(1))])`, true},
		{`Unless(CostWasntPaid,[GainLife(Integer(1))])`, true},
		{`If(CostWasntPaid,[GainLife(Integer(1))])`, false},
		{`Unless(CostWasPaid,[GainLife(Integer(1))])`, false},
		{`IfElse(CostWasPaid,[GainLife(Integer(1))],[LoseLife(Integer(1))])`, true},
		{`IfElse(CostWasntPaid,[GainLife(Integer(1))],[LoseLife(Integer(1))])`, false},
	} {
		for _, accept := range []bool{false, true} {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: !accept}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: accept}
			g := mage.NewGame(a, b)
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerMayCost(Ref_TargetPlayer,GainLife(Integer(4))),`+continuation.text+`])))`)
			card := factory()
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			want := 20
			if accept {
				want += 4
			}
			if accept == continuation.gainPaid {
				want++
			} else if strings.Contains(continuation.text, "IfElse") {
				want--
			}
			if a.Life() != want || b.Life() != 20 || a.offers != 0 || b.offers != 1 {
				t.Fatal("branch polarity or explicit payer", continuation.text, accept)
			}
		}
	}
}

type ronActionNativePlayer struct {
	*gametest.TestPlayer
	target uuid.UUID
	accept bool
}

func (p *ronActionNativePlayer) ChooseMayAbility(string) bool { return p.accept }
func (p *ronActionNativePlayer) ChooseTargets(possible []uuid.UUID, minimum, maximum int, g *mage.Game) []uuid.UUID {
	for _, id := range possible {
		if id == p.target {
			return []uuid.UUID{id}
		}
	}
	return p.TestPlayer.ChooseTargets(possible, minimum, maximum, g)
}

func TestRONActionPaymentNativeCombat(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"FEM", "Delif's Cone"}, {"FEM", "Farrel's Zealot"}, {"LEG", "Floral Spuzzem"}} {
		for _, state := range []string{"accept", "decline", "blocked"} {
			t.Run(tc.name+"/"+state, func(t *testing.T) {
				a := &ronActionNativePlayer{TestPlayer: gametest.NewTestPlayer("A"), accept: state != "decline"}
				b := gametest.NewTestPlayer("B")
				g := mage.NewGame(a, b)
				g.SetTurn(1)
				g.SetStep(core.PrecombatMain)
				card := repositoryCard(t, tc.set, tc.name)
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				if tc.name != "Delif's Cone" && g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0) == nil {
					t.Fatal("unpaid cast")
				}
				a.ManaPool().Add(core.Colorless, 10)
				a.ManaPool().Add(core.White, 2)
				a.ManaPool().Add(core.Green, 1)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				attacker := g.FindPermanent(card.ID())
				if tc.name == "Delif's Cone" {
					attacker = g.PutOnBattlefield(mage.NewCreature("Attacker", "", 4, 8), a.PlayerID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, []uuid.UUID{attacker.ID()}); err != nil {
						t.Fatal(err)
					}
					g.ResolveStack()
				}
				attacker.RevokeBaseAttr(core.AttrSummonSick)
				var target *mage.Permanent
				if tc.name == "Floral Spuzzem" {
					target = g.PutOnBattlefield(mage.NewArtifact("Recipient", ""), b.PlayerID())
				} else {
					target = g.PutOnBattlefield(mage.NewCreature("Recipient", "", 0, 8), b.PlayerID())
				}
				a.target = target.ID()
				g.PutOnBattlefield(mage.NewCreature("Blocker", "", 0, 8), b.PlayerID())
				a.SetAttackers(1, []string{attacker.Name()})
				if state == "blocked" {
					b.SetBlockers(1, map[string]string{"Blocker": attacker.Name()})
				}
				g.RunStepWithPriority(core.DeclareAttackers)
				if a.Life() != 20 {
					t.Fatal("early payment")
				}
				g.RunStepWithPriority(core.DeclareBlockers)
				clone := g.Clone()
				for _, game := range []*mage.Game{g, clone} {
					game.RunStepWithPriority(core.CombatDamage)
					wantB := 20
					if state == "decline" {
						wantB -= attacker.CurrentPower(game)
					}
					wantA := 20
					if state == "accept" && tc.name == "Delif's Cone" {
						wantA = 24
					}
					if game.GetPlayer(a.PlayerID()).Life() != wantA || game.GetPlayer(b.PlayerID()).Life() != wantB {
						t.Fatal("native combat result")
					}
				}
				g.RunStepWithPriority(core.EndCombat)
				g.RunStepWithPriority(core.Cleanup)
				g.GetCombat().AddAttacker(attacker.ID(), b.PlayerID())
				g.ExecuteCombatDamage()
				if b.Life() != clone.GetPlayer(b.PlayerID()).Life()-attacker.CurrentPower(g) {
					t.Fatal("payment rule survived cleanup")
				}
			})
		}
	}
}
