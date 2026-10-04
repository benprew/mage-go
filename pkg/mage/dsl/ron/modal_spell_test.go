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

type ronModePlayer struct {
	*mage.BasePlayer
	mode    int
	targets []uuid.UUID
	prompts int
}

func (p *ronModePlayer) ChooseMode(_ []string, _ string) int { p.prompts++; return p.mode }
func (p *ronModePlayer) ChooseTargets(_ []uuid.UUID, _, _ int, _ *mage.Game) []uuid.UUID {
	return p.targets
}
func ronModalGame(mode int) (*mage.Game, *ronModePlayer, *mage.BasePlayer) {
	a := &ronModePlayer{BasePlayer: mage.NewBasePlayer("A"), mode: mode}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	g.SetActivePlayerIndex(1)
	g.SetTurn(1)
	g.SetStep(core.PrecombatMain)
	return g, a, b
}
func TestRONModalBlastCards(t *testing.T) {
	for _, name := range []string{"Blue Elemental Blast", "Red Elemental Blast"} {
		for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
			for mode := range 2 {
				t.Run(fmt.Sprintf("%s/%s/%d", name, set, mode), func(t *testing.T) {
					g, a, b := ronModalGame(mode)
					victimName := "Air Elemental"
					if name == "Blue Elemental Blast" {
						victimName = "Shivan Dragon"
					}
					victim := repositoryCard(t, "LEB", victimName)
					var target uuid.UUID
					if mode == 0 {
						target = castRONSpell(t, g, b, victim).ID
					} else {
						victim.SetOwner(b.PlayerID())
						target = g.PutOnBattlefield(victim, b.PlayerID()).ID()
					}
					card := repositoryCard(t, set, name)
					if err := mage.ValidateCardDefinition(name, card); err != nil {
						t.Fatal(err)
					}
					card.SetOwner(a.PlayerID())
					a.AddToHand(card)
					a.ManaPool().Add(core.Blue, 1)
					a.ManaPool().Add(core.Red, 1)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, 0); err != nil {
						t.Fatal(err)
					}
					obj := g.StackPeek()
					if obj.ModeChoice != mode || len(obj.ModalTargets) != 2 || len(obj.ModalTargets[1-mode]) != 0 || len(obj.Targets) != 1 || obj.Targets[0] != target {
						t.Fatal("mode targets were not selected at announcement")
					}
					clone := g.Clone()
					clone.ResolveTopOfStack()
					if (mode == 0 && g.GetStack().FindByID(target) == nil) || (mode == 1 && g.FindPermanent(target) == nil) {
						t.Fatal("clone changed original target")
					}
					g.ResolveTopOfStack()
					if g.GetStack().FindByID(target) != nil || g.FindPermanent(target) != nil || len(b.Graveyard()) != 1 {
						t.Fatal("selected mode did not move victim to owner's graveyard")
					}
					if a.prompts != 1 {
						t.Fatal("mode was chosen again at resolution")
					}
				})
			}
		}
	}
}
func TestRONModalInvalid(t *testing.T) {
	good := `Targeted([TargetPermanent(AnyPermanent)], ActionList([DestroyPermanent(Ref_TargetPermanent)]))`
	for _, body := range []string{
		`Modal_ChooseOne`, `Modal_ChooseOne()`, `Modal_ChooseOne(ActionList([]))`, `Modal_ChooseOne([])`, `Modal_ChooseOne([` + good + `])`,
		`Modal_ChooseOne([` + good + `, ActionList([DestroyPermanent(Ref_TargetPermanent)])])`,
		`Modal_ChooseOne([ActionList([CounterSpell(Ref_TargetSpell)]), Targeted([TargetSpell(AnySpell)], ActionList([CounterSpell(Ref_TargetSpell)]))])`,
		`Modal_ChooseOne([` + good + `, Targeted([TargetPlayer(AnyPlayer)], ActionList([DestroyPermanent(Ref_TargetPermanent)]))])`,
		`Modal_ChooseOne([` + good + `, Unknown([])])`,
		`Modal_ChooseOne([` + good + `, Modal_ChooseOne([` + good + `, ` + good + `])])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Instant], Subtypes: []), Rules: [SpellActions(`+body+`)])`), "invalid-modal.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
	for _, rule := range []string{`Activated(PayMana([]), Modal_ChooseOne([` + good + `, ` + good + `]))`, `SpellActions(Modal_ChooseOne([` + good + `, ` + good + `]))`} {
		nodes, err := Parse(strings.NewReader(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [`+rule+`])`), "invalid-context.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted unsupported permanent modal context")
		}
	}
}
func TestRONModalInstances(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Modal_ChooseOne([Targeted([TargetPermanent(AnyPermanent)], ActionList([DestroyPermanent(Ref_TargetPermanent)])), Targeted([TargetPermanent(AnyPermanent)], ActionList([TapPermanent(Ref_TargetPermanent)]))]))`)
	first, second := factory(), factory()
	modes := first.Abilities()[0].(*mage.ModalSpellAbility).Modes()
	other := second.Abilities()[0].(*mage.ModalSpellAbility).Modes()
	for _, mode := range modes {
		if strings.Contains(mode.Label, "Ref_") || strings.Contains(mode.Label, "mode1/") {
			t.Fatal("mode label exposes compiler binding names")
		}
	}
	if modes[0].Targets[0] == modes[1].Targets[0] || modes[0].Targets[0] == other[0].Targets[0] {
		t.Fatal("target declarations are shared")
	}
	for _, card := range []mage.Card{first, second} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRONModalAnnouncementRollback(t *testing.T) {
	for _, reason := range []string{"wrong-mode-target", "wrong-color", "unpaid", "invalid-mode"} {
		t.Run(reason, func(t *testing.T) {
			g, a, b := ronModalGame(1)
			victim := repositoryCard(t, "LEB", "Shivan Dragon")
			victim.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(victim, b.PlayerID()).ID()
			if reason == "wrong-mode-target" {
				a.mode = 0
			}
			if reason == "wrong-color" {
				g.AddContinuousEffect(mage.BecomesColor(target, core.Blue, core.EndOfTurn))
			}
			if reason == "invalid-mode" {
				a.mode = 2
			}
			card := repositoryCard(t, "LEB", "Blue Elemental Blast")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if reason != "unpaid" {
				a.ManaPool().Add(core.Blue, 1)
			}
			before := a.ManaPool().Count(core.Blue)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, 0); err == nil {
				t.Fatal("illegal or unpaid cast succeeded")
			}
			if g.StackSize() != 0 || len(a.Hand()) != 1 || len(a.Graveyard()) != 0 || a.ManaPool().Count(core.Blue) != before || g.FindPermanent(target) == nil {
				t.Fatal("failed announcement changed resources")
			}
		})
	}
}
func TestRONModalResolutionLegality(t *testing.T) {
	for mode := range 2 {
		for _, change := range []string{"color", "depart-return", "phased", "regenerate", "uncounterable"} {
			if (mode == 0 && (change == "phased" || change == "regenerate")) || (mode == 1 && change == "uncounterable") {
				continue
			}
			t.Run(fmt.Sprintf("%d/%s", mode, change), func(t *testing.T) {
				g, a, b := ronModalGame(mode)
				victim := repositoryCard(t, "LEB", "Shivan Dragon")
				victim.SetOwner(b.PlayerID())
				var target uuid.UUID
				if mode == 0 {
					target = castRONSpell(t, g, b, victim).ID
				} else {
					target = g.PutOnBattlefield(victim, b.PlayerID()).ID()
				}
				card := repositoryCard(t, "LEB", "Blue Elemental Blast")
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				a.ManaPool().Add(core.Blue, 1)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, 0); err != nil {
					t.Fatal(err)
				}
				var returned uuid.UUID
				switch change {
				case "color":
					if mode == 0 {
						victim.(*mage.BaseCard).SetColorOverride([]core.Color{core.Blue})
					} else {
						g.AddContinuousEffect(mage.BecomesColor(target, core.Blue, core.EndOfTurn))
					}
				case "depart-return":
					if mode == 0 {
						g.GetStack().RemoveByID(target)
						old := g.StackPeek()
						g.GetStack().RemoveByID(old.ID)
						recast := &mage.StackObject{ID: uuid.New(), SourceID: victim.ID(), Card: victim, Controller: b.PlayerID()}
						returned = recast.ID
						g.PushStack(recast)
						g.PushStack(old)
					} else {
						g.RemoveFromBattlefield(g.FindPermanent(target))
						returned = g.PutOnBattlefield(victim, b.PlayerID()).ID()
					}
				case "phased":
					g.PhaseOut(target)
				case "regenerate":
					g.AddRegenerationShield(target)
				case "uncounterable":
					mage.WithUncounterable()(victim.(*mage.BaseCard))
				}
				g.ResolveTopOfStack()
				if change == "depart-return" {
					if (mode == 0 && g.GetStack().FindByID(returned) == nil) || (mode == 1 && g.FindPermanent(returned) == nil) {
						t.Fatal("old target followed returned object")
					}
				} else if mode == 0 {
					if g.GetStack().FindByID(target) == nil {
						t.Fatal("illegal or uncounterable spell countered")
					}
				} else if change != "phased" {
					p := g.FindPermanent(target)
					if p == nil {
						t.Fatal("illegal or regenerated permanent destroyed")
					}
					if change == "regenerate" && (!p.Tapped || g.HasRegenerationShield(target)) {
						t.Fatal("destruction did not use regeneration service")
					}
				}
				if change == "phased" {
					g.PhaseIn([]uuid.UUID{target})
					if g.FindPermanent(target) == nil {
						t.Fatal("phased target was destroyed")
					}
				}
				if len(b.Graveyard()) != 0 {
					t.Fatal("failed action moved victim")
				}
			})
		}
	}
}
func TestRONModalCopyTargets(t *testing.T) {
	g, a, b := ronModalGame(1)
	firstCard, secondCard := repositoryCard(t, "LEB", "Shivan Dragon"), repositoryCard(t, "LEB", "Shivan Dragon")
	firstCard.SetOwner(b.PlayerID())
	secondCard.SetOwner(b.PlayerID())
	first, second := g.PutOnBattlefield(firstCard, b.PlayerID()), g.PutOnBattlefield(secondCard, b.PlayerID())
	card := repositoryCard(t, "LEB", "Blue Elemental Blast")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Blue, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{first.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	clone := g.Clone()
	a.targets = []uuid.UUID{second.ID()}
	a.mode = 0
	copied := g.CopyStackObjectDirect(original, a.PlayerID(), true)
	if copied == nil || copied.ModeChoice != 1 || copied.Targets[0] != second.ID() || original.Targets[0] != first.ID() || a.prompts != 1 {
		t.Fatal("copy changed mode or original target")
	}
	g.ResolveTopOfStack()
	if g.FindPermanent(second.ID()) != nil || g.FindPermanent(first.ID()) == nil {
		t.Fatal("copy did not use its own target")
	}
	g.ResolveTopOfStack()
	if g.FindPermanent(first.ID()) != nil || len(b.Graveyard()) != 2 || len(a.Graveyard()) != 1 {
		t.Fatal("original did not keep its own mode and target")
	}
	clone.ResolveTopOfStack()
	if clone.FindPermanent(first.ID()) != nil || clone.FindPermanent(second.ID()) == nil {
		t.Fatal("clone shared copied targets")
	}
}

func TestRONModalUntargetedAndPlayerModes(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Modal_ChooseOne([Targeted([TargetPlayer(AnyPlayer)], ActionList([PlayerAction(Ref_TargetPlayer, DrawNumberCards(Integer(3)))])), ActionList([GainLife(Integer(2))])]))`)
	for mode := range 2 {
		g, a, b := ronModalGame(mode)
		fillRONLibrary(t, b, 3)
		card := factory()
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		var targets []uuid.UUID
		if mode == 0 {
			targets = []uuid.UUID{b.PlayerID()}
		}
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, 0); err != nil {
			t.Fatal(err)
		}
		if len(g.StackPeek().Targets) != 1-mode {
			t.Fatal("unchosen mode requested targets")
		}
		g.ResolveTopOfStack()
		if (mode == 0 && (a.Life() != 20 || len(b.Hand()) != 3)) || (mode == 1 && (a.Life() != 22 || len(b.Hand()) != 0)) {
			t.Fatal("wrong player or mode action executed")
		}
	}
}
func TestRONModalInventory(t *testing.T) {
	source := `Card(Name: "Modal", Typeline: (Supertypes: [], Cardtypes: [Instant], Subtypes: []), Rules: [SpellActions(Modal_ChooseOne([Targeted([TargetSpell(AnySpell)], ActionList([CounterSpell(Ref_TargetSpell)])), Targeted([TargetPermanent(AnyPermanent)], ActionList([DestroyPermanent(Ref_TargetPermanent)]))]))])`
	report, err := InspectFS(fstest.MapFS{"modal.ron": {Data: []byte(source)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 1 {
		t.Fatal("supported modal declaration not compiled")
	}
	roles := map[string]string{"Modal_ChooseOne": "actions", "Targeted": "actions", "TargetSpell": "target", "CounterSpell": "action", "Ref_TargetSpell": "spell_reference", "DestroyPermanent": "action", "Ref_TargetPermanent": "permanent_reference"}
	for _, entry := range report.Signatures {
		if role, ok := roles[entry.Name]; ok && entry.Role != role {
			t.Errorf("%s role = %s, want %s", entry.Name, entry.Role, role)
		}
	}
}
