package ron

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONPlayerLifeStreamOfLife(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, x := range []int{0, 4} {
			t.Run(fmt.Sprintf("%s/%d", set, x), func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, set, "Stream of Life")
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				a.ManaPool().Add(core.Green, 1)
				if x > 0 {
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, x); err == nil {
						t.Fatal("unpaid X accepted")
					}
					if len(a.Hand()) != 1 || a.ManaPool().TotalMana() != 1 || !g.Stack().IsEmpty() {
						t.Fatal("failed payment changed state")
					}
				}
				a.ManaPool().Add(core.Colorless, x)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{uuid.New()}, x); err == nil {
					t.Fatal("invalid player accepted")
				}
				if a.ManaPool().TotalMana() != x+1 || len(a.Hand()) != 1 {
					t.Fatal("invalid target paid cost")
				}
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, x); err != nil {
					t.Fatal(err)
				}
				if a.Life() != 20 || b.Life() != 20 || a.ManaPool().TotalMana() != 0 {
					t.Fatal("life changed at announcement")
				}
				clone := g.Clone()
				g.ResolveStack()
				if a.Life() != 20 || b.Life() != 20+x || len(a.Graveyard()) != 1 {
					t.Fatal("wrong recipient or X")
				}
				if clone.GetPlayer(b.PlayerID()).Life() != 20 {
					t.Fatal("clone shared life")
				}
				clone.ResolveStack()
				if clone.GetPlayer(b.PlayerID()).Life() != 20+x || b.Life() != 20+x {
					t.Fatal("clone lost target or X")
				}
			})
		}
	}
}

func TestRONPlayerLifeScopesAndCopies(t *testing.T) {
	for _, action := range []string{"GainLife", "LoseLife"} {
		t.Run(action, func(t *testing.T) {
			body := fmt.Sprintf(`SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerAction(Ref_TargetPlayer,%s(Integer(3))),PlayerAction(You,%s(Integer(1)))])))`, action, action)
			factory := permanentScopeFactory(t, "Instant", body)
			first, second := factory(), factory()
			if first.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == second.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
				t.Fatal("shared declarations")
			}
			if err := mage.ValidateCardDefinition(first.Name(), first); err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			first.SetOwner(a.PlayerID())
			second.SetOwner(a.PlayerID())
			a.AddToHand(first)
			a.AddToHand(second)
			if err := g.CastSpellByID(a.PlayerID(), first.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
				t.Fatal(err)
			}
			if err := g.CastSpellByID(a.PlayerID(), second.ID(), []uuid.UUID{a.PlayerID()}, 0); err != nil {
				t.Fatal(err)
			}
			g.AddPreventionShield(a.PlayerID(), 100)
			g.AddPreventionShield(b.PlayerID(), 100)
			clone := g.Clone()
			cp := g.CopySpellOnStack(first.ID(), b.PlayerID(), true)
			if cp == nil || len(cp.Targets) != 1 || cp.Targets[0] != a.PlayerID() {
				t.Fatal("copy did not retarget")
			}
			g.ResolveStack()
			sign := 1
			if action == "LoseLife" {
				sign = -1
			}
			if a.Life() != 20+8*sign || b.Life() != 20+4*sign {
				t.Fatalf("scope/copy: %d/%d", a.Life(), b.Life())
			}
			clone.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != 20+5*sign || clone.GetPlayer(b.PlayerID()).Life() != 20+3*sign {
				t.Fatal("clone binding")
			}
		})
	}
}

func TestRONPlayerLifeTargetRecheckAndReplacement(t *testing.T) {
	for _, scenario := range []string{"normal", "illegal", "replacement", "phased-source", "returned-source", "control"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			factory := permanentScopeFactory(t, "Artifact", `Activated(TapPermanent(ThisPermanent),Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerAction(Ref_TargetPlayer,GainLife(Integer(3)))])))`)
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{uuid.New()}); err == nil {
				t.Fatal("invalid target accepted")
			}
			if g.FindPermanent(source.ID()).Tapped || !g.Stack().IsEmpty() {
				t.Fatal("invalid activation paid tap")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "phased-source":
				g.PhaseOut(source.ID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			default:
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				if scenario == "returned-source" {
					g.PutOnBattlefield(source.Card, a.PlayerID())
				}
			}
			if scenario == "illegal" {
				g.StackPeek().Targets = []uuid.UUID{uuid.New()}
			}
			if scenario == "replacement" {
				g.PutOnBattlefield(mage.NewEnchantment("Double", "", mage.WithLifeGainReplacement(mage.DoubleLifeGainForController)), b.PlayerID())
			}
			g.ResolveStack()
			want := 23
			if scenario == "illegal" {
				want = 20
			}
			if scenario == "replacement" {
				want = 26
			}
			if a.Life() != 20 || b.Life() != want {
				t.Fatalf("life %d/%d", a.Life(), b.Life())
			}
		})
	}
}

func TestRONPlayerLifeInvalidDeclarations(t *testing.T) {
	for _, body := range []string{
		`SpellActions(ActionList([PlayerAction(Ref_TargetPlayer,GainLife(Integer(3)))]))`,
		`SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([PlayerAction(Ref_TargetPlayer,LoseLife(Integer(3)))])))`,
		`SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerAction(Ref_TargetPlayer(),GainLife(Integer(3)))])))`,
		`SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerAction(Ref_TargetPlayer,GainLife())])))`,
		`SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerAction(Ref_TargetPlayer,LoseLife(Integer(1),Integer(2)))])))`,
		`SpellActions(Modal_ChooseOne([Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerAction(Ref_TargetPlayer,GainLife(Integer(3)))])),ActionList([PlayerAction(Ref_TargetPlayer,GainLife(Integer(3)))])]))`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[` + body + `])`
		nodes, err := Parse(strings.NewReader(src), "invalid-life.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRONPlayerLifeRealModalClauses(t *testing.T) {
	paths, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		records, err := Parse(strings.NewReader(string(data)), filepath.Base(path))
		if err != nil {
			t.Fatal(err)
		}
		for _, record := range records {
			name := record.Fields["Name"].String
			if name != "Healing Salve" && name != "Alabaster Potion" {
				continue
			}
			count++
			t.Run(filepath.Base(path)+"/"+name, func(t *testing.T) {
				// The unsupported prevention mode must still reject the complete record.
				if _, err := Compile(record); err == nil {
					t.Fatal("unsupported complete modal record accepted")
				}
				rules := record.Fields["Rules"]
				spell := rules.Args[0]
				spell.Args = []Node{spell.Args[0].Args[0].Args[0]}
				rules.Args = []Node{spell}
				record.Fields["Rules"] = rules
				factory, err := Compile(record)
				if err != nil {
					t.Fatal(err)
				}
				g, a, b := ronGame()
				card := factory()
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				x, gain := 0, 3
				a.ManaPool().Add(core.White, 2)
				if name == "Alabaster Potion" {
					x, gain = 4, 4
					a.ManaPool().Add(core.Colorless, 4)
				}
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, x); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				if a.Life() != 20 || b.Life() != 20+gain {
					t.Fatal("real mode life recipient")
				}
			})
		}
	}
	if count != 7 {
		t.Fatalf("tested %d real modal clauses", count)
	}
}

func TestRONPlayerLifeModeLocalTargets(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Modal_ChooseOne([Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerAction(Ref_TargetPlayer,GainLife(Integer(3)))])),Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerAction(Ref_TargetPlayer,LoseLife(Integer(2)))]))]))`)
	for mode := range 2 {
		g, a, b := ronModalGame(mode)
		card := factory()
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
			t.Fatal(err)
		}
		obj := g.StackPeek()
		if len(obj.ModalTargets[1-mode]) != 0 || len(obj.ModalTargets[mode]) != 1 {
			t.Fatal("unchosen target group bound")
		}
		g.ResolveStack()
		want := 23
		if mode == 1 {
			want = 18
		}
		if a.Life() != 20 || b.Life() != want || a.prompts != 1 {
			t.Fatal("wrong mode or recipient")
		}
	}
}

func TestRONPlayerLifeSpellController(t *testing.T) {
	for _, action := range []string{"GainLife", "LoseLife"} {
		factory := permanentScopeFactory(t, "Instant", fmt.Sprintf(`SpellActions(Targeted([TargetSpell(AnySpell)],ActionList([PlayerAction(ControllerOfTargetSpell,%s(Integer(3)))])))`, action))
		g, a, b := ronGame()
		victim := mage.NewInstant("Victim", "", mage.WithAbility(mage.Spell(mage.GainLifeFor(mage.You, mage.Fixed(1)))))
		obj := castRONSpell(t, g, b, victim)
		card := factory()
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{obj.ID}, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveTopOfStack()
		want := 23
		if action == "LoseLife" {
			want = 17
		}
		if a.Life() != 20 || b.Life() != want {
			t.Fatal("spell controller fell back to You")
		}
	}
}

func TestRONPlayerLifeTriggerAndImplicitController(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerAction(Ref_TargetPlayer,GainLife(Integer(3))),GainLife(Integer(1))])))`)
	g, a, b := ronModalGame(0)
	a.targets = []uuid.UUID{b.PlayerID()}
	card := factory()
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	source := g.PutOnBattlefield(card, a.PlayerID())
	g.PutTriggersOnStack()
	if g.Stack().Size() != 1 || a.Life() != 20 || b.Life() != 20 {
		t.Fatal("trigger announcement changed life")
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 23 {
		t.Fatal("trigger target or implicit controller")
	}
}
