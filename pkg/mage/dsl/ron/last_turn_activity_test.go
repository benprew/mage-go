package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONLastTurnActivityArboria(t *testing.T) {
	for _, activity := range []string{"none", "land", "actor-other-controller", "other-actor", "token", "cast-countered", "opponent-turn-cast", "phase-cycle"} {
		t.Run(activity, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", "Arboria")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			g.SetStep(core.PrecombatMain)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("unpaid cast changed state")
			}
			a.ManaPool().Add(core.Green, 4)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().TotalMana() != 0 || len(a.Hand()) != 0 {
				t.Fatal("cast did not pay")
			}
			pending := g.Clone()
			pending.ResolveStack()
			if g.FindPermanent(card.ID()) != nil {
				t.Fatal("clone resolved original spell")
			}
			g.ResolveStack()
			source := g.FindPermanent(card.ID())
			attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 2, 5), b.PlayerID())
			attacker.RevokeBaseAttr(core.AttrSummonSick)
			g.RunStepWithPriority(core.Cleanup)
			g.SetActivePlayerIndex(1)
			g.SetStep(core.DeclareAttackers)
			if !g.FindPermanent(attacker.ID()).CanDeclareAsAttacker(g) {
				t.Fatal("Arboria cast was absent from its controller's history")
			}
			g.SetActivePlayerIndex(0)
			g.SetStep(core.PrecombatMain)
			want := false
			switch activity {
			case "land":
				land := repositoryCard(t, "LEB", "Forest")
				land.SetOwner(a.PlayerID())
				a.AddToHand(land)
				if err := g.PlayLand(a.PlayerID(), land.ID()); err != nil {
					t.Fatal(err)
				}
				g.RemoveFromBattlefield(g.FindPermanent(land.ID()))
				want = true
			case "actor-other-controller":
				p := g.PutOnBattlefieldBy(mage.NewCreature("Gift", "", 1, 2), b.PlayerID(), a.PlayerID())
				g.RemoveFromBattlefield(p)
				want = true
			case "other-actor":
				g.PutOnBattlefieldBy(mage.NewCreature("Gift", "", 1, 2), a.PlayerID(), b.PlayerID())
			case "token":
				g.PutOnBattlefieldBy(mage.NewToken("Token", 1, 1, []core.CardType{core.TypeCreature}, nil), a.PlayerID(), a.PlayerID())
			case "cast-countered":
				spell := mage.NewInstant("Cast", "", mage.WithAbility(mage.Spell(mage.GainLifeFor(mage.You, mage.Fixed(1)))))
				spell.SetOwner(a.PlayerID())
				a.AddToHand(spell)
				if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				g.CounterStackObject(g.StackPeek().ID)
				want = true
			case "opponent-turn-cast":
				g.SetActivePlayerIndex(1)
				spell := mage.NewInstant("Cast", "", mage.WithAbility(mage.Spell(mage.GainLifeFor(mage.You, mage.Fixed(1)))))
				spell.SetOwner(a.PlayerID())
				a.AddToHand(spell)
				if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				g.RunStepWithPriority(core.Cleanup)
				g.SetActivePlayerIndex(0)
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			}
			g.RunStepWithPriority(core.Cleanup)
			g.SetActivePlayerIndex(1)
			g.SetStep(core.DeclareAttackers)
			if got := g.FindPermanent(attacker.ID()).CanDeclareAsAttacker(g); got != want {
				t.Fatalf("attack=%v want %v", got, want)
			}
			g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{attacker.ID()})
			if g.GetCombat().IsAttacking(attacker.ID()) != want || g.FindPermanent(attacker.ID()).Tapped != want {
				t.Fatal("RON attack declaration disagrees with legality")
			}
			g.GetCombat().Reset()
			g.UntapPermanent(g.FindPermanent(attacker.ID()))
			clone := g.Clone()
			changed := g.Clone()
			changed.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			if changed.FindPermanent(attacker.ID()).CanDeclareAsAttacker(changed) != want {
				t.Fatal("source control changed protected player history")
			}
			changed.RemoveFromBattlefield(changed.FindPermanent(source.ID()))
			if !changed.FindPermanent(attacker.ID()).CanDeclareAsAttacker(changed) {
				t.Fatal("departed source retained rule")
			}
			changed.PutOnBattlefield(card, b.PlayerID())
			if changed.FindPermanent(attacker.ID()).CanDeclareAsAttacker(changed) != want {
				t.Fatal("returned source lost player history")
			}
			g.PhaseOut(source.ID())
			if !g.FindPermanent(attacker.ID()).CanDeclareAsAttacker(g) {
				t.Fatal("phased source restricted attack")
			}
			g.PhaseIn([]uuid.UUID{source.ID()})
			g.AddContinuousEffect(mage.RemoveAllAbilities(attacker.ID(), core.EndOfTurn))
			if got := g.FindPermanent(attacker.ID()).CanDeclareAsAttacker(g); got != want {
				t.Fatal("attacker ability removal erased external rule")
			}
			g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.EndOfTurn))
			if !g.FindPermanent(attacker.ID()).CanDeclareAsAttacker(g) {
				t.Fatal("removed source ability retained rule")
			}
			if clone.FindPermanent(attacker.ID()).CanDeclareAsAttacker(clone) != want {
				t.Fatal("clone history changed")
			}
		})
	}
}

func TestRONLastTurnActivityDeclarations(t *testing.T) {
	valid := `EachPermanentRuleEffect(IsCardtype(Creature),[CantAttackAPlayer(ExceptFor(Or([CastASpellDuringTheirLastTurn(AnySpell),PutAPermanentOnBattleDuringTheirLastTurn(IsNonToken)])))])`
	for _, bad := range []string{
		strings.Replace(valid, "AnySpell", "AnySpell()", 1),
		strings.Replace(valid, "AnySpell", "IsCardtype(Instant)", 1),
		strings.Replace(valid, "IsNonToken", "IsNonToken()", 1),
		strings.Replace(valid, "IsNonToken", "AnyPermanent", 1),
		strings.Replace(valid, "ExceptFor(", "ExceptFor(AnyPlayer,", 1),
		strings.Replace(valid, "Or([CastASpellDuringTheirLastTurn(AnySpell),PutAPermanentOnBattleDuringTheirLastTurn(IsNonToken)])", "Or([])", 1),
		strings.Replace(valid, "ExceptFor(Or([CastASpellDuringTheirLastTurn(AnySpell),PutAPermanentOnBattleDuringTheirLastTurn(IsNonToken)]))", "AnyPlayer", 1),
		strings.Replace(valid, "CastASpellDuringTheirLastTurn(AnySpell)", "CastASpellDuringTheirLastTurn(AnySpell,You)", 1),
		strings.Replace(valid, "IsCardtype(Creature)", "ControlledByPlayer(Ref_TargetPlayer)", 1),
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[` + bad + `])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration accepted", bad)
		}
	}
	record := staticBaseRecord(t, "LEG", "Arboria")
	types := record.Fields["Typeline"].Fields["Cardtypes"]
	types.Args = []Node{{Kind: Symbol, Name: "Instant"}}
	record.Fields["Typeline"].Fields["Cardtypes"] = types
	if _, err := Compile(record); err == nil {
		t.Fatal("spell static restriction accepted")
	}
}

func TestRONLastTurnActivityLoad(t *testing.T) {
	record := staticBaseRecord(t, "LEG", "Arboria")
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition("Arboria", factory()); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	data := ""
	for line := range strings.SplitSeq(string(raw), "\n") {
		if strings.Contains(line, `Name: "Arboria"`) {
			data = line
		}
	}
	if data == "" {
		t.Fatal("missing record")
	}
	files := fstest.MapFS{"cards.ron": {Data: []byte(data)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil || registry == nil {
		t.Fatal(err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("inspection", err)
	}
	roles := map[string]bool{"last_turn_attack_player_filter/CastASpellDuringTheirLastTurn": false, "last_turn_attack_player_filter/PutAPermanentOnBattleDuringTheirLastTurn": false, "spell_filter/AnySpell": false, "permanent_filter/IsNonToken": false}
	for _, signature := range report.Signatures {
		key := signature.Role + "/" + signature.Name
		if _, ok := roles[key]; ok {
			roles[key] = true
		}
	}
	for key, found := range roles {
		if !found {
			t.Fatal("missing inventory role", key)
		}
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(data, "AnySpell", "AnySpell()", 1))}
	if registry, err := LoadFS(files, "*.ron"); err == nil || registry != nil {
		t.Fatal("partial registry published")
	}
}
