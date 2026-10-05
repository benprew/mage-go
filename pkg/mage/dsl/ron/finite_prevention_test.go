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

const finitePrevention = `CreateFuturePreventDamage(NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(Integer(3), Ref_AnyTarget), [PreventThatDamage])`

func TestRONFinitePreventionModes(t *testing.T) {
	for _, tc := range []struct {
		set, name       string
		amount, x, mana int
	}{
		{"LEB", "Healing Salve", 3, 0, 1}, {"LEG", "Alabaster Potion", 4, 4, 6}, {"LEG", "Alabaster Potion", 0, 0, 2},
	} {
		for mode := range 2 {
			for _, permanent := range []bool{false, true} {
				if mode == 0 && permanent {
					continue
				}
				t.Run(fmt.Sprintf("%s/X%d/mode%d/permanent%v", tc.name, tc.x, mode, permanent), func(t *testing.T) {
					g, a, b := ronModalGame(mode)
					victim := g.PutOnBattlefield(mage.NewCreature("Recipient", "", 1, 20), b.PlayerID())
					id := b.PlayerID()
					if permanent {
						id = victim.ID()
					}
					card := repositoryCard(t, tc.set, tc.name)
					card.SetOwner(a.PlayerID())
					a.AddToHand(card)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{id}, tc.x); err == nil || g.StackSize() != 0 || len(a.Hand()) != 1 {
						t.Fatal("unpaid cast changed state")
					}
					a.ManaPool().Add(core.White, tc.mana)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{id}, tc.x); err != nil {
						t.Fatal(err)
					}
					if a.ManaPool().TotalMana() != 0 {
						t.Fatal("wrong casting payment")
					}
					g.DealDamageToPlayer(b, 1, uuid.New())
					if b.Life() != 19 {
						t.Fatal("shield applied before resolution")
					}
					clone := g.Clone()
					g.ResolveStack()
					if mode == 0 {
						if b.Life() != 19+tc.amount || a.Life() != 20 {
							t.Fatal("life mode used wrong player or amount")
						}
						g.DealDamageToPlayer(b, tc.amount+1, uuid.New())
						if b.Life() != 18 {
							t.Fatal("life mode installed prevention")
						}
					} else if permanent {
						g.DealDamageToPermanent(g.FindPermanent(id), tc.amount+2, uuid.New())
						if g.FindPermanent(id).Damage != 2 || b.Life() != 19 {
							t.Fatal("permanent shield amount or recipient")
						}
					} else {
						g.DealDamageToPlayer(b, tc.amount+2, uuid.New())
						if b.Life() != 17 {
							t.Fatal("player shield amount")
						}
					}
					if clone.GetPlayer(b.PlayerID()).Life() != 19 || clone.FindPermanent(victim.ID()).Damage != 0 {
						t.Fatal("pending clone changed")
					}
					clone.ResolveStack()
					if mode == 1 {
						clone.DealDamageToPlayer(clone.GetPlayer(b.PlayerID()), tc.amount+2, uuid.New())
						want := 17
						if permanent {
							want = 17 - tc.amount
						}
						if clone.GetPlayer(b.PlayerID()).Life() != want {
							t.Fatal("clone lost mode or recipient")
						}
					}
				})
			}
		}
	}
}

type finiteActivationCase struct {
	set, name                         string
	amount, mana                      int
	tap, creature, self, artifactOnly bool
}

var finiteActivations = []finiteActivationCase{
	{"LEB", "Samite Healer", 1, 0, true, true, false, false},
	{"LEB", "Conservator", 2, 3, true, false, true, false},
	{"ARN", "Oasis", 1, 0, true, false, false, false},
	{"ATQ", "Amulet of Kroog", 1, 2, true, false, false, false},
	{"ATQ", "Argivian Blacksmith", 2, 0, true, true, false, true},
	{"FEM", "Combat Medic", 1, 2, false, true, false, false},
	{"LEG", "Kei Takahashi", 2, 0, true, true, false, false},
	{"ATQ", "Rakalite", 1, 2, false, false, false, false},
}

func TestRONFinitePreventionActivations(t *testing.T) {
	for _, tc := range finiteActivations {
		for _, state := range []string{"normal", "unpaid-or-tapped", "sick", "depart-source", "control-source", "copy"} {
			t.Run(tc.name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, tc.set, tc.name)
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				victimCard := mage.NewCreature("Recipient", "", 1, 20)
				if tc.artifactOnly {
					victimCard.AddType(core.TypeArtifact)
				}
				victim := g.PutOnBattlefield(victimCard, b.PlayerID())
				id := victim.ID()
				var targets []uuid.UUID
				if !tc.self {
					targets = []uuid.UUID{id}
				}
				if state != "sick" {
					source.RevokeBaseAttr(core.AttrSummonSick)
				}
				if state != "unpaid-or-tapped" {
					a.ManaPool().Add(core.White, tc.mana)
				} else if tc.mana == 0 {
					source.Tapped = true
				}
				err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, targets)
				invalid := state == "unpaid-or-tapped" || (state == "sick" && tc.creature && tc.tap)
				if invalid {
					if err == nil || g.StackSize() != 0 || a.ManaPool().TotalMana() != map[bool]int{true: 0, false: tc.mana}[state == "unpaid-or-tapped"] {
						t.Fatal("illegal activation changed resources")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if a.ManaPool().TotalMana() != 0 || (tc.tap && !g.FindPermanent(source.ID()).Tapped) {
					t.Fatal("wrong activation payment")
				}
				if state == "depart-source" {
					g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				}
				if state == "control-source" {
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				clone := g.Clone()
				g.ResolveStack()
				clone.ResolveStack()
				amount := tc.amount
				if state == "copy" && !tc.self {
					amount *= 2
				}
				if tc.self {
					g.DealDamageToPlayer(a, tc.amount+1, uuid.New())
					g.DealDamageToPlayer(b, tc.amount+1, uuid.New())
					want := 19 - tc.amount
					if state == "copy" {
						want = 19
					}
					if a.Life() != 19 || b.Life() != want {
						t.Fatal("controller capture or copied controller")
					}
				} else {
					g.DealDamageToPermanent(g.FindPermanent(id), amount+1, uuid.New())
					if g.FindPermanent(id).Damage != 1 {
						t.Fatal("target shield or source departure")
					}
					if clone.FindPermanent(id).Damage != 0 {
						t.Fatal("clone shared damage")
					}
					clone.DealDamageToPermanent(clone.FindPermanent(id), amount+1, uuid.New())
					if clone.FindPermanent(id).Damage != 1 {
						t.Fatal("clone lost shield")
					}
				}
				g.RunStepWithPriority(core.Cleanup)
				if tc.self {
					before := a.Life()
					g.DealDamageToPlayer(a, 1, uuid.New())
					if a.Life() != before-1 {
						t.Fatal("cleanup kept shield")
					}
				} else {
					before := g.FindPermanent(id).Damage
					g.DealDamageToPermanent(g.FindPermanent(id), 1, uuid.New())
					if g.FindPermanent(id).Damage != before+1 {
						t.Fatal("cleanup kept shield")
					}
				}
			})
		}
	}
}

func TestRONFinitePreventionInvalidAndInventory(t *testing.T) {
	for _, action := range []string{
		`CreateFuturePreventDamage`, `CreateFuturePreventDamage()`,
		`CreateFuturePreventDamage(NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(Integer(3),Ref_AnyTarget))`,
		`CreateFuturePreventDamage(NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(Integer(3),Ref_AnyTarget),[])`,
		`CreateFuturePreventDamage(NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(Integer(3),Ref_AnyTarget),[PreventThatDamage()])`,
		`CreateFuturePreventDamage(NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(Integer(3),Ref_AnyTarget),[PreventThatDamage,GainLife(Integer(1))])`,
		`CreateFuturePreventDamage(NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(ValueX,Ref_AnyTarget),[PreventThatDamage])`,
		`CreateFuturePreventDamage(NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(Integer(-1),Ref_AnyTarget),[PreventThatDamage])`,
		`CreateFuturePreventDamage(NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(Integer(3),Permanent(Ref_TargetPlayer)),[PreventThatDamage])`,
		`CreateFuturePreventDamage(NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(Integer(3),Player(Trigger_ThatPlayer)),[PreventThatDamage])`,
		`CreateFuturePreventDamage(NextAmountOfDamageThatWouldBeDealtThisTurnByPermanent(Integer(1),ThisPermanent),[PreventThatDamage])`,
		`CreateFuturePreventDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(TheChosenDamageSource,Player(You)),[PreventThatDamage])`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([AnyTarget],ActionList([` + action + `])))])`
		nodes, err := Parse(strings.NewReader(src), "invalid-finite.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
	src := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([AnyTarget],ActionList([` + finitePrevention + `])))])`
	report, err := InspectFS(fstest.MapFS{"finite.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 1 {
		t.Fatalf("compile: %v", report.Issues)
	}
	roles := map[string]string{"NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient": "prevention_event", "Integer": "value", "Ref_AnyTarget": "prevention_recipient", "PreventThatDamage": "prevention_operation"}
	for _, entry := range report.Signatures {
		if role, ok := roles[entry.Name]; ok && entry.Role != role {
			t.Errorf("%s role %s, want %s", entry.Name, entry.Role, role)
		}
	}
}

func castFiniteSalve(t *testing.T, g *mage.Game, a mage.Player, id uuid.UUID) {
	t.Helper()
	card := repositoryCard(t, "LEB", "Healing Salve")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.White, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{id}, 0); err != nil {
		t.Fatal(err)
	}
}

func TestRONFinitePreventionIdentity(t *testing.T) {
	for _, state := range []string{"normal", "shroud-before", "shroud-after", "phase-before", "return-before", "phase-after", "return-after", "control-after", "unpreventable"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronModalGame(1)
			card := mage.NewCreature("Recipient", "", 1, 20)
			card.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(card, b.PlayerID())
			id := p.ID()
			spell := repositoryCard(t, "LEB", "Healing Salve")
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(core.White, 1)
			if state == "shroud-before" {
				p.GrantBaseAttr(core.Shroud)
			}
			err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{id}, 0)
			if state == "shroud-before" {
				if err == nil || g.StackSize() != 0 || a.ManaPool().TotalMana() != 1 {
					t.Fatal("illegal target spent resources")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			bounce := func() {
				g.BouncePermanentToHand(g.FindPermanent(id))
				b.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
			}
			if state == "shroud-after" {
				g.FindPermanent(id).GrantBaseAttr(core.Shroud)
			}
			if state == "phase-before" {
				g.PhaseOut(id)
			}
			if state == "return-before" {
				bounce()
			}
			g.ResolveStack()
			if state == "phase-before" {
				g.PhaseIn([]uuid.UUID{id})
			}
			if state == "phase-after" {
				g.PhaseIn(g.PhaseOut(id))
			}
			if state == "return-after" {
				bounce()
			}
			if state == "control-after" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: id, ControllerID: a.PlayerID(), Duration: core.Indefinite})
			}
			if state == "unpreventable" {
				g.FindPermanent(id).GrantBaseAttr(core.AttrDamageCantBePreventedOrRedirected)
			}
			g.DealDamageToPermanent(g.FindPermanent(id), 2, uuid.New())
			want := 0
			if state == "shroud-after" || state == "phase-before" || state == "return-before" || state == "return-after" || state == "unpreventable" {
				want = 2
			}
			if g.FindPermanent(id).Damage != want {
				t.Fatal("shield identity or legality")
			}
			if state == "unpreventable" {
				g.FindPermanent(id).RevokeBaseAttr(core.AttrDamageCantBePreventedOrRedirected)
			}
			g.DealDamageToPermanent(g.FindPermanent(id), 2, uuid.New())
			next := want + 2
			if want == 0 {
				next = 1
			}
			if state == "unpreventable" {
				next = 2
			}
			if g.FindPermanent(id).Damage != next {
				t.Fatal("shield capacity or unpreventable consumption")
			}
		})
	}
}

func TestRONFinitePreventionRetargetAndInstances(t *testing.T) {
	g, a, b := ronModalGame(1)
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 1, 20), a.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 20), b.PlayerID())
	castFiniteSalve(t, g, a, first.ID())
	original := g.StackPeek()
	clone := g.Clone()
	a.targets = []uuid.UUID{second.ID()}
	copied := g.CopyStackObjectDirect(original, a.PlayerID(), true)
	if copied == nil || copied.Targets[0] != second.ID() || original.Targets[0] != first.ID() {
		t.Fatal("copy target binding")
	}
	g.ResolveStack()
	for _, p := range []*mage.Permanent{first, second} {
		g.DealDamageToPermanent(g.FindPermanent(p.ID()), 4, uuid.New())
		if g.FindPermanent(p.ID()).Damage != 1 {
			t.Fatal("copy and original shared shield")
		}
	}
	clone.ResolveStack()
	clone.DealDamageToPermanent(clone.FindPermanent(second.ID()), 4, uuid.New())
	if clone.FindPermanent(second.ID()).Damage != 4 {
		t.Fatal("clone inherited copy")
	}
	a.targets = nil
	castFiniteSalve(t, g, a, a.PlayerID())
	card := repositoryCard(t, "LEB", "Healing Salve")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.White, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.DealDamageToPlayer(a, 4, uuid.New())
	g.DealDamageToPlayer(b, 4, uuid.New())
	if a.Life() != 19 || b.Life() != 19 {
		t.Fatal("factory instances shared recipient")
	}
}

func TestRONFinitePreventionRakaliteDelay(t *testing.T) {
	for _, state := range []string{"normal", "twice", "phase", "return", "control"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "ATQ", "Rakalite")
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			n := 1
			if state == "twice" {
				n = 2
			}
			for range n {
				a.ManaPool().Add(core.Colorless, 2)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
			}
			if len(g.DelayedTriggers()) != n {
				t.Fatal("delays were not independent")
			}
			g.DealDamageToPlayer(b, n+1, uuid.New())
			if b.Life() != 19 {
				t.Fatal("shield amount")
			}
			if state == "phase" {
				g.PhaseOut(p.ID())
			}
			if state == "return" {
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
			}
			if state == "control" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			clone := g.Clone()
			g.RunStepWithPriority(core.EndStep)
			present := state == "phase" || state == "return"
			if (g.FindPermanentIncludingPhased(p.ID()) != nil) != present || len(g.DelayedTriggers()) != 0 {
				t.Fatal("delayed return identity")
			}
			if !present && (len(a.Hand()) != 1 || len(b.Hand()) != 0) {
				t.Fatal("return ignored owner")
			}
			if clone.FindPermanentIncludingPhased(p.ID()) == nil || len(clone.DelayedTriggers()) != n {
				t.Fatal("delay mutated clone")
			}
			g.RunStepWithPriority(core.EndStep)
			if len(a.Hand()) > 1 {
				t.Fatal("return repeated")
			}
		})
	}
}

type finiteChoicePlayer struct {
	*mage.BasePlayer
	redirect bool
	numbers  int
}

func (p *finiteChoicePlayer) ChooseNumber(_, _ int, _ string) int { p.numbers++; return 1 }
func (p *finiteChoicePlayer) ChooseMode(options []string, _ string) int {
	if p.redirect && len(options) > 1 {
		return 1
	}
	return 0
}

func TestRONFinitePreventionBatchAndCombat(t *testing.T) {
	for _, kind := range []string{"batch", "combat", "redirect-first", "prevent-first"} {
		t.Run(kind, func(t *testing.T) {
			a := &ronModePlayer{BasePlayer: mage.NewBasePlayer("A"), mode: 1}
			b := &finiteChoicePlayer{BasePlayer: mage.NewBasePlayer("B"), redirect: kind == "redirect-first"}
			g := mage.NewGame(a, b)
			g.SetTurn(1)
			g.SetStep(core.PrecombatMain)
			first := g.PutOnBattlefield(mage.NewCreature("First", "", 3, 10, mage.WithKeyword(core.Lifelink)), a.PlayerID())
			second := g.PutOnBattlefield(mage.NewCreature("Second", "", 3, 10), a.PlayerID())
			victim := g.PutOnBattlefield(mage.NewCreature("Recipient", "", 1, 20), b.PlayerID())
			id := b.PlayerID()
			redirect := kind == "redirect-first" || kind == "prevent-first"
			if redirect {
				id = victim.ID()
			}
			castFiniteSalve(t, g, a, id)
			g.ResolveStack()
			if kind == "combat" {
				first.RevokeBaseAttr(core.AttrSummonSick)
				second.RevokeBaseAttr(core.AttrSummonSick)
				g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{first.ID(), second.ID()})
				g.ExecuteCombatDamage()
			} else {
				s1, s2 := g.CaptureDamageSource(first.ID()), g.CaptureDamageSource(second.ID())
				if redirect {
					g.SetCreatureDamageRedirect(id, b.PlayerID())
					g.DealSimultaneousDamage(mage.PermanentDamageAssignment(g.FindPermanent(id), 3, s1), mage.PermanentDamageAssignment(g.FindPermanent(id), 3, s2))
				} else {
					g.RemoveFromBattlefield(first)
					g.PutOnBattlefield(first.Card, b.PlayerID())
					g.DealSimultaneousDamage(mage.PlayerDamageAssignment(b, 3, s1), mage.PlayerDamageAssignment(b, 3, s2))
				}
			}
			if !redirect {
				if b.Life() != 17 || a.Life() != 22 || b.numbers != 1 {
					t.Fatalf("batch life %d/%d choices %d", a.Life(), b.Life(), b.numbers)
				}
			} else {
				damage, life := 1, 18
				if kind == "redirect-first" {
					damage, life = 0, 17
				}
				if g.FindPermanent(id).Damage != damage || b.Life() != life {
					t.Fatalf("replacement damage=%d life=%d", g.FindPermanent(id).Damage, b.Life())
				}
			}
		})
	}
}

func TestRONFinitePreventionActivationTargets(t *testing.T) {
	for _, tc := range finiteActivations {
		if tc.self {
			continue
		}
		for _, kind := range []string{"player", "land", "creature", "artifact-creature", "shroud", "phase-at-resolution", "type-at-resolution"} {
			if kind == "type-at-resolution" && !tc.artifactOnly {
				continue
			}
			t.Run(tc.name+"/"+kind, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, tc.set, tc.name)
				card.SetOwner(a.PlayerID())
				src := g.PutOnBattlefield(card, a.PlayerID())
				src.RevokeBaseAttr(core.AttrSummonSick)
				a.ManaPool().Add(core.White, tc.mana)
				victimCard := mage.NewCreature("Recipient", "", 1, 20)
				typed := tc.name == "Oasis" || tc.name == "Kei Takahashi" || tc.artifactOnly
				if kind == "artifact-creature" || tc.artifactOnly && (kind == "shroud" || kind == "phase-at-resolution" || kind == "type-at-resolution") {
					victimCard.AddType(core.TypeArtifact)
				}
				p := g.PutOnBattlefield(victimCard, b.PlayerID())
				id := p.ID()
				if kind == "player" {
					id = b.PlayerID()
				}
				if kind == "land" {
					id = g.PutOnBattlefield(mage.NewLand("Land"), b.PlayerID()).ID()
				}
				if kind == "shroud" {
					p.GrantBaseAttr(core.Shroud)
				}
				err := g.ActivateAbilityByIndex(a.PlayerID(), src.ID(), 0, []uuid.UUID{id})
				invalid := kind == "land" || kind == "shroud" || kind == "player" && typed || kind == "creature" && tc.artifactOnly
				if invalid {
					if err == nil || g.StackSize() != 0 || g.FindPermanent(src.ID()).Tapped || a.ManaPool().TotalMana() != tc.mana {
						t.Fatal("target legality or rollback")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if kind == "phase-at-resolution" {
					g.PhaseOut(id)
				}
				if kind == "type-at-resolution" {
					g.FindPermanent(id).RevokeBaseAttr(core.AttrIsArtifact)
				}
				g.ResolveStack()
				if kind == "phase-at-resolution" {
					g.PhaseIn([]uuid.UUID{id})
				}
				if kind == "player" {
					g.DealDamageToPlayer(b, tc.amount+1, uuid.New())
					if b.Life() != 19 {
						t.Fatal("any-target player shield")
					}
				} else {
					g.DealDamageToPermanent(g.FindPermanent(id), tc.amount+1, uuid.New())
					want := 1
					if kind == "phase-at-resolution" || kind == "type-at-resolution" && typed {
						want = tc.amount + 1
					}
					if g.FindPermanent(id).Damage != want {
						t.Fatal("resolution legality")
					}
				}
			})
		}
	}
}

func TestRONFinitePreventionResolvingContexts(t *testing.T) {
	for _, kind := range []string{"trigger", "delayed"} {
		t.Run(kind, func(t *testing.T) {
			g, a, b := ronGame()
			if kind == "trigger" {
				body := strings.ReplaceAll(finitePrevention, "Ref_AnyTarget", "Player(You)")
				card := permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([`+body+`]))`)()
				card.SetOwner(a.PlayerID())
				g.PutOnBattlefield(card, b.PlayerID())
				g.PutTriggersOnStack()
				g.ResolveStack()
				g.DealDamageToPlayer(b, 4, uuid.New())
				if b.Life() != 19 || a.Life() != 20 {
					t.Fatal("trigger controller binding")
				}
			} else {
				src := `Card(Name:"Delayed Shield",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),ManaCost:[ManaCostX],Rules:[SpellActions(Targeted([AnyTarget],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([` + strings.ReplaceAll(finitePrevention, "Integer(3)", "ValueX") + `]))])))])`
				nodes, err := Parse(strings.NewReader(src), "delayed-shield.ron")
				if err != nil {
					t.Fatal(err)
				}
				factory, err := Compile(nodes[0])
				if err != nil {
					t.Fatal(err)
				}
				for i, player := range []mage.Player{a, b} {
					card := factory()
					card.SetOwner(a.PlayerID())
					a.AddToHand(card)
					a.ManaPool().Add(core.Colorless, i+2)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{player.PlayerID()}, i+2); err != nil {
						t.Fatal(err)
					}
					g.ResolveStack()
				}
				g.DealDamageToPlayer(a, 1, uuid.New())
				if a.Life() != 19 {
					t.Fatal("delay applied early")
				}
				g.RunStepWithPriority(core.EndStep)
				g.DealDamageToPlayer(a, 3, uuid.New())
				g.DealDamageToPlayer(b, 4, uuid.New())
				if a.Life() != 18 || b.Life() != 19 {
					t.Fatal("delays shared target or X")
				}
			}
		})
	}
}
