package ron

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const sourceCombatPrevention = `CreatePreventDamageUntil(CombatDamageWouldBeDealtByCreature(Ref_TargetPermanent), [PreventThatDamage], UntilEndOfTurn)`

func TestRONSourceCombatPreventionCards(t *testing.T) {
	for _, name := range []string{"Horn of Deafening", "Lady Evangela"} {
		for _, scenario := range []string{"normal", "unpaid", "sick", "land", "player", "return", "phase", "phase-cycle", "shroud", "control", "source-return", "source-phase", "source-control", "copy", "target-return-after"} {
			t.Run(name+"/"+scenario, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, "LEG", name)
				if err := mage.ValidateCardDefinition(name, card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				if scenario != "sick" {
					source.RevokeBaseAttr(core.AttrSummonSick)
				}
				targetCard := mage.NewCreature("Attacker", "", 4, 12, mage.WithKeyword(core.DoubleStrike), mage.WithKeyword(core.Trample), mage.WithKeyword(core.Lifelink))
				targetCard.SetOwner(b.PlayerID())
				target := g.PutOnBattlefield(targetCard, b.PlayerID())
				id := target.ID()
				if scenario == "land" {
					id = g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID()).ID()
				}
				if scenario == "player" {
					id = b.PlayerID()
				}
				if scenario != "unpaid" {
					a.ManaPool().Add(core.White, 1)
					a.ManaPool().Add(core.Black, 1)
				}
				mana := a.ManaPool().TotalMana()
				err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{id})
				reject := scenario == "unpaid" || scenario == "land" || scenario == "player" || (scenario == "sick" && name == "Lady Evangela")
				if reject {
					if err == nil || g.StackSize() != 0 || g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != mana {
						t.Fatal("illegal activation changed state")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if !g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != 0 {
					t.Fatal("incorrect costs")
				}
				if sourceCombatPrevented(g, id, a.PlayerID()) {
					t.Fatal("prevention began before resolution")
				}
				prevented := true
				var phased []uuid.UUID
				switch scenario {
				case "return":
					g.RemoveFromBattlefield(target)
					g.PutOnBattlefield(targetCard, b.PlayerID())
					prevented = false
				case "phase":
					phased = g.PhaseOut(id)
					prevented = false
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(id))
				case "shroud":
					g.FindPermanent(id).GrantBaseAttr(core.Shroud)
					prevented = false
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: id, ControllerID: a.PlayerID(), Duration: core.Indefinite})
				case "source-return":
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
					g.PutOnBattlefield(card, a.PlayerID())
				case "source-phase":
					g.PhaseOut(source.ID())
				case "source-control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				clone := g.Clone()
				g.ResolveStack()
				if scenario != "phase" && sourceCombatPrevented(clone, id, a.PlayerID()) {
					t.Fatal("resolution changed pending clone")
				}
				clone.ResolveStack()
				for _, game := range []*mage.Game{g, clone} {
					if scenario == "phase" {
						game.PhaseIn(phased)
					}
					if scenario == "target-return-after" {
						game.RemoveFromBattlefield(game.FindPermanent(id))
						game.PutOnBattlefield(targetCard, b.PlayerID())
						prevented = false
					}
					if sourceCombatPrevented(game, id, a.PlayerID()) != prevented {
						t.Fatal("wrong target identity or legality")
					}
				}
				if scenario != "normal" {
					return
				}
				blocker := g.PutOnBattlefield(mage.NewCreature("Blocker", "", 2, 12), a.PlayerID())
				other := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 12), b.PlayerID())
				for range 2 {
					g.GetCombat().Reset()
					g.GetCombat().AddAttacker(id, a.PlayerID())
					g.GetCombat().AddAttacker(other.ID(), a.PlayerID())
					g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: blocker.ID(), AttackerID: id}})
					g.ExecuteCombatDamage()
					g.GetEffects().RemoveEndOfCombat()
				}
				if a.Life() != 18 || b.Life() != 20 || g.FindPermanent(blocker.ID()).Damage != 0 || g.FindPermanent(id).Damage != 4 {
					t.Fatalf("combat life=%d/%d damage=%d/%d", a.Life(), b.Life(), g.FindPermanent(blocker.ID()).Damage, g.FindPermanent(id).Damage)
				}
				g.DealDamageToPlayer(a, 1, id)
				if a.Life() != 17 || b.Life() != 21 {
					t.Fatal("noncombat damage or lifelink prevented")
				}
				g.PhaseIn(g.PhaseOut(id))
				g.RunStepWithPriority(core.Cleanup)
				g.GetCombat().Reset()
				g.GetCombat().AddAttacker(id, a.PlayerID())
				g.ExecuteCombatDamage()
				if a.Life() != 9 {
					t.Fatalf("cleanup life=%d want=9", a.Life())
				}
			})
		}
	}
}

func TestRONSourceCombatPreventionDeclarations(t *testing.T) {
	for _, action := range []string{
		strings.Replace(sourceCombatPrevention, "CombatDamageWouldBeDealtByCreature(Ref_TargetPermanent)", "CombatDamageWouldBeDealtByCreature", 1),
		strings.Replace(sourceCombatPrevention, "Ref_TargetPermanent", "Ref_TargetPermanent, ThisPermanent", 1),
		strings.Replace(sourceCombatPrevention, "Ref_TargetPermanent", "Ref_TargetPermanent()", 1),
		strings.Replace(sourceCombatPrevention, "Ref_TargetPermanent", "Ref_TargetPlayer", 1),
		strings.Replace(sourceCombatPrevention, "Ref_TargetPermanent", "Ref_TargetPermanent_2", 1),
		strings.Replace(sourceCombatPrevention, "Ref_TargetPermanent", "HostPermanent", 1),
		strings.Replace(sourceCombatPrevention, "Ref_TargetPermanent", "Trigger_ThatPermanent", 1),
		strings.Replace(sourceCombatPrevention, "[PreventThatDamage]", "[]", 1),
		strings.Replace(sourceCombatPrevention, "[PreventThatDamage]", "[PreventThatDamage, PreventThatDamage]", 1),
		strings.Replace(sourceCombatPrevention, "PreventThatDamage]", "PreventThatDamage() ]", 1),
		strings.Replace(sourceCombatPrevention, "UntilEndOfTurn", "UntilEndOfCombat", 1),
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
	for _, rule := range []string{sourceCombatPrevention, `SpellActions(ActionList([` + sourceCombatPrevention + `]))`, `Activated(` + sourceCombatPrevention + `,ActionList([]))`, `PermanentLayerEffect(HostPermanent,[` + sourceCombatPrevention + `])`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "context.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
	src := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([` + sourceCombatPrevention + `])))])`
	report, err := InspectFS(fstest.MapFS{"test.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, sig := range report.Signatures {
		if sig.Name == "Ref_TargetPermanent" && sig.Role != "permanent_reference" {
			t.Fatalf("role=%s", sig.Role)
		}
	}
}

func sourceCombatPrevented(g *mage.Game, source, recipient uuid.UUID) bool {
	clone := g.Clone()
	if p := clone.FindPermanent(source); p.ControllerID() == recipient {
		recipient = clone.GetOpponent(recipient).PlayerID()
	}
	before := clone.GetPlayer(recipient).Life()
	clone.GetCombat().Reset()
	clone.GetCombat().AddAttacker(source, recipient)
	clone.ExecuteCombatDamage()
	return clone.GetPlayer(recipient).Life() == before
}

func TestRONSourceCombatPreventionSubdue(t *testing.T) {
	for _, change := range []string{"none", "return", "phase", "phase-cycle", "copy", "counter"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			targetCard := repositoryCard(t, "LEB", "Shivan Dragon")
			targetCard.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(targetCard, b.PlayerID())
			other := g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), b.PlayerID())
			spell := repositoryCard(t, "LEG", "Subdue")
			if err := mage.ValidateCardDefinition(spell.Name(), spell); err != nil {
				t.Fatal(err)
			}
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{target.ID()}, 0); err == nil {
				t.Fatal("unpaid cast accepted")
			}
			if len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("failed cost changed state")
			}
			a.ManaPool().Add(core.Green, 1)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			if target.CurrentToughness(g) != 5 || sourceCombatPrevented(g, target.ID(), a.PlayerID()) {
				t.Fatal("effect began before resolution")
			}
			clone := g.Clone()
			switch change {
			case "return":
				g.RemoveFromBattlefield(target)
				g.PutOnBattlefield(targetCard, b.PlayerID())
			case "phase":
				g.PhaseOut(target.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(target.ID()))
			case "copy":
				phased := g.PhaseOut(target.ID())
				cp := g.CopySpellOnStack(spell.ID(), a.PlayerID(), true)
				if cp == nil || len(cp.Targets) != 1 || cp.Targets[0] != other.ID() {
					t.Fatal("copy did not retarget independently")
				}
				g.PhaseIn(phased)
			case "counter":
				g.CounterStackObject(g.StackPeek().ID)
			}
			g.ResolveStack()
			g.PhaseIn([]uuid.UUID{target.ID()})
			want := 5
			prevented := change == "none" || change == "phase-cycle" || change == "copy"
			if prevented {
				want = 11
			}
			if g.FindPermanent(target.ID()).CurrentPower(g) != 5 || g.FindPermanent(target.ID()).CurrentToughness(g) != want || sourceCombatPrevented(g, target.ID(), a.PlayerID()) != prevented {
				t.Fatal("wrong prevention or mana-value bonus")
			}
			if change == "copy" && (g.FindPermanent(other.ID()).CurrentToughness(g) != 2 || !sourceCombatPrevented(g, other.ID(), a.PlayerID())) {
				t.Fatal("copy used original target or mana value")
			}
			if clone.FindPermanent(target.ID()).CurrentToughness(clone) != 5 || sourceCombatPrevented(clone, target.ID(), a.PlayerID()) {
				t.Fatal("resolution changed pending clone")
			}
			clone.ResolveStack()
			if clone.FindPermanent(target.ID()).CurrentToughness(clone) != 11 || !sourceCombatPrevented(clone, target.ID(), a.PlayerID()) {
				t.Fatal("clone lost effect")
			}
			g.RunStepWithPriority(core.Cleanup)
			if g.FindPermanent(target.ID()).CurrentToughness(g) != 5 || sourceCombatPrevented(g, target.ID(), a.PlayerID()) {
				t.Fatal("cleanup did not expire both effects")
			}
			if clone.FindPermanent(target.ID()).CurrentToughness(clone) != 11 {
				t.Fatal("cleanup changed clone")
			}
		})
	}
}

func TestRONSourceCombatPreventionReplacementOrder(t *testing.T) {
	for mode := range 2 {
		a := &ronModePlayer{BasePlayer: mage.NewBasePlayer("A"), mode: mode}
		b := mage.NewBasePlayer("B")
		g := mage.NewGame(a, b)
		g.SetStep(core.PrecombatMain)
		g.SetTurn(1)
		horn := g.PutOnBattlefield(repositoryCard(t, "LEG", "Horn of Deafening"), a.PlayerID())
		attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 3, 12), b.PlayerID())
		victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 1, 12), a.PlayerID())
		a.ManaPool().Add(core.Colorless, 2)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), horn.ID(), 0, []uuid.UUID{attacker.ID()}); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		g.SetCreatureDamageRedirect(victim.ID(), a.PlayerID())
		g.GetCombat().AddAttacker(attacker.ID(), a.PlayerID())
		g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: victim.ID(), AttackerID: attacker.ID()}})
		g.ExecuteCombatDamage()
		if a.Life() != 20 || g.FindPermanent(victim.ID()).Damage != 0 || a.prompts != 1 {
			t.Fatalf("life=%d damage=%d prompts=%d", a.Life(), g.FindPermanent(victim.ID()).Damage, a.prompts)
		}
		g.FindPermanent(victim.ID()).GrantBaseAttr(core.AttrDamageCantBePreventedOrRedirected)
		g.GetCombat().Reset()
		g.GetCombat().AddAttacker(attacker.ID(), a.PlayerID())
		g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: victim.ID(), AttackerID: attacker.ID()}})
		g.ExecuteCombatDamage()
		if g.FindPermanent(victim.ID()).Damage != 3 {
			t.Fatal("unpreventable damage prevented")
		}
	}
}

func TestRONSourceCombatPreventionInstances(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), "LEG.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, node := range nodes {
		if node.Fields["Name"].String != "Horn of Deafening" {
			continue
		}
		factory, err := Compile(node)
		if err != nil {
			t.Fatal(err)
		}
		g, a, b := ronGame()
		first := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 8), b.PlayerID())
		second := g.PutOnBattlefield(mage.NewCreature("Second", "", 3, 8), b.PlayerID())
		for _, target := range []uuid.UUID{first.ID(), second.ID()} {
			horn := g.PutOnBattlefield(factory(), a.PlayerID())
			a.ManaPool().Add(core.Colorless, 2)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), horn.ID(), 0, []uuid.UUID{target}); err != nil {
				t.Fatal(err)
			}
		}
		g.ResolveTopOfStack()
		if sourceCombatPrevented(g, first.ID(), a.PlayerID()) || !sourceCombatPrevented(g, second.ID(), a.PlayerID()) {
			t.Fatal("factory instances shared target state")
		}
		g.ResolveTopOfStack()
		if !sourceCombatPrevented(g, first.ID(), a.PlayerID()) || !sourceCombatPrevented(g, second.ID(), a.PlayerID()) {
			t.Fatal("earlier binding was overwritten")
		}
		return
	}
	t.Fatal("missing Horn of Deafening")
}
