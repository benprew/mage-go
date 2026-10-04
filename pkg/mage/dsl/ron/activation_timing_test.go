package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONActivationTimingLegality(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		upkeep    bool
	}{
		{"LEB", "Disrupting Scepter", false}, {"FEM", "Svyelunite Priest", true},
	} {
		for _, scenario := range []string{"legal", "wrong-turn", "wrong-step", "unpaid", "tapped", "phased", "sick", "wrong-controller", "invalid-target", "new-controller"} {
			t.Run(tc.name+"/"+scenario, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, tc.set, tc.name)
				card.SetOwner(a.PlayerID())
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				p := g.PutOnBattlefield(card, a.PlayerID())
				p.RevokeBaseAttr(core.AttrSummonSick)
				subjectCard := mage.NewCreature("Subject", "", 3, 3)
				subjectCard.SetOwner(b.PlayerID())
				subject := g.PutOnBattlefield(subjectCard, b.PlayerID())
				payer := a
				g.SetStep(core.Upkeep)
				if !tc.upkeep {
					g.SetStep(core.DeclareBlockers)
				}
				targets := []uuid.UUID{b.PlayerID()}
				if tc.upkeep {
					targets = []uuid.UUID{subject.ID()}
				}
				switch scenario {
				case "wrong-turn":
					g.SetActivePlayerIndex(1)
				case "wrong-step":
					g.SetStep(core.Draw)
				case "tapped":
					g.TapPermanent(p)
				case "phased":
					g.PhaseOut(p.ID())
				case "sick":
					if tc.upkeep {
						g.MutablePermanent(p.ID()).GrantBaseAttr(core.AttrSummonSick)
					}
				case "wrong-controller":
					payer = b
				case "invalid-target":
					targets = []uuid.UUID{uuid.New()}
				case "new-controller":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: subject.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
					g.SetActivePlayerIndex(1)
					payer = b
				}
				color, mana := core.Colorless, 3
				if tc.upkeep {
					color, mana = core.Blue, 2
				}
				if scenario == "unpaid" {
					mana--
				}
				payer.ManaPool().Add(color, mana)
				allowed := scenario == "legal" || scenario == "new-controller" || (scenario == "wrong-step" && !tc.upkeep) || (scenario == "sick" && !tc.upkeep)
				err := g.ActivateAbilityByIndex(payer.PlayerID(), p.ID(), 0, targets)
				if !allowed {
					if err == nil {
						t.Fatal("illegal activation succeeded")
					}
					if !g.GetStack().IsEmpty() || payer.ManaPool().TotalMana() != mana || g.FindPermanentIncludingPhased(p.ID()).Tapped != (scenario == "tapped") {
						t.Fatal("failed activation changed state")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if !g.FindPermanent(p.ID()).Tapped || payer.ManaPool().TotalMana() != 0 || g.GetStack().IsEmpty() {
					t.Fatal("activation did not pay costs and use the stack")
				}
				g.SetActivePlayerIndex(1 - g.ActivePlayerIndex())
				g.SetStep(core.EndStep)
				g.ResolveStack()
				if tc.upkeep && !g.FindPermanent(subject.ID()).HasAttr(core.Shroud) {
					t.Fatal("timing was checked again at resolution")
				}
			})
		}
	}
}

func TestRONActivationTimingScepter(t *testing.T) {
	for _, targetSelf := range []bool{false, true} {
		for _, change := range []string{"none", "departed", "returned", "control", "phased"} {
			t.Run(fmt.Sprintf("self=%v/%s", targetSelf, change), func(t *testing.T) {
				a := &ronDiscardPlayer{BasePlayer: mage.NewBasePlayer("A")}
				b := &ronDiscardPlayer{BasePlayer: mage.NewBasePlayer("B")}
				g := mage.NewGame(a, b)
				g.SetTurn(1)
				g.SetStep(core.PrecombatMain)
				for _, player := range []*ronDiscardPlayer{a, b} {
					for range 2 {
						c := mage.NewCreature("Hand", "", 1, 1)
						c.SetOwner(player.PlayerID())
						player.AddToHand(c)
					}
				}
				card := repositoryCard(t, "LEB", "Disrupting Scepter")
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				target := b
				if targetSelf {
					target = a
				}
				a.ManaPool().Add(core.Colorless, 3)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.PlayerID()}); err != nil {
					t.Fatal(err)
				}
				if len(target.Hand()) != 2 || len(target.requests) != 0 {
					t.Fatal("discard occurred before resolution")
				}
				pending := g.Clone()
				switch change {
				case "departed":
					g.DestroyPermanent(g.FindPermanent(source.ID()))
				case "returned":
					g.BouncePermanentToHand(g.FindPermanent(source.ID()))
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, a.PlayerID())
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "phased":
					g.PhaseOut(source.ID())
				}
				g.SetActivePlayerIndex(1)
				g.SetStep(core.Upkeep)
				g.ResolveStack()
				if len(target.Hand()) != 1 || len(target.requests) != 1 || target.requests[0] != 1 {
					t.Fatal("discard used the wrong player or amount")
				}
				other := a
				if targetSelf {
					other = b
				}
				if len(other.Hand()) != 2 || len(other.requests) != 0 {
					t.Fatal("discard affected the other player")
				}
				if len(pending.GetPlayer(target.PlayerID()).Hand()) != 2 {
					t.Fatal("clone shared resolved hand")
				}
				pending.ResolveStack()
				if len(pending.GetPlayer(target.PlayerID()).Hand()) != 1 || len(target.Hand()) != 1 {
					t.Fatal("pending clone lost or repeated discard")
				}
			})
		}
	}
}

func TestRONActivationTimingPriest(t *testing.T) {
	for _, change := range []string{"none", "source-leaves", "source-return", "source-control", "source-phase", "target-leaves", "target-return", "target-phase", "target-phase-in", "target-control", "target-noncreature"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			g.SetStep(core.Upkeep)
			card := repositoryCard(t, "FEM", "Svyelunite Priest")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			victim := repositoryCard(t, "LEB", "Air Elemental")
			victim.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(victim, b.PlayerID())
			id := target.ID()
			a.ManaPool().Add(core.Blue, 2)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{id}); err != nil {
				t.Fatal(err)
			}
			if target.HasAttr(core.Shroud) {
				t.Fatal("grant occurred before resolution")
			}
			pending := g.Clone()
			switch change {
			case "source-leaves":
				g.DestroyPermanent(g.FindPermanent(source.ID()))
			case "source-return":
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
			case "source-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "source-phase":
				g.PhaseOut(source.ID())
			case "target-leaves":
				g.DestroyPermanent(g.FindPermanent(id))
			case "target-return":
				g.BouncePermanentToHand(g.FindPermanent(id))
				b.RemoveFromHand(victim.ID())
				id = g.PutOnBattlefield(victim, b.PlayerID()).ID()
			case "target-phase":
				g.PhaseOut(id)
			case "target-phase-in":
				g.PhaseIn(g.PhaseOut(id))
			case "target-noncreature":
				g.AddContinuousEffect(mage.FuncContinuousEffect(core.LayerType, core.EndOfTurn, func(game *mage.Game, _ uuid.UUID) error {
					game.GetEffects().RevokeAttr(id, core.AttrIsCreature)
					return nil
				}))
				g.ApplyContinuousEffects()
			case "target-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: id, ControllerID: a.PlayerID(), Duration: core.Indefinite})
			}
			g.SetActivePlayerIndex(1)
			g.SetStep(core.Draw)
			g.ResolveStack()
			g.PhaseIn([]uuid.UUID{id})
			affected := change != "target-leaves" && change != "target-return" && change != "target-phase" && change != "target-noncreature"
			if p := g.FindPermanent(id); p != nil {
				if p.HasAttr(core.Shroud) != affected || !p.HasAttr(core.Flying) || p.CurrentPower(g) != 4 {
					t.Fatal("grant lost target identity or characteristics")
				}
			} else if change != "target-leaves" {
				t.Fatal("target is absent")
			}
			pending.ResolveStack()
			if !pending.FindPermanent(target.ID()).HasAttr(core.Shroud) {
				t.Fatal("clone lost pending grant")
			}
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			if p := g.FindPermanent(id); p != nil && p.HasAttr(core.Shroud) {
				t.Fatal("grant did not expire")
			}
			if !pending.FindPermanent(target.ID()).HasAttr(core.Shroud) {
				t.Fatal("cleanup changed clone")
			}
		})
	}
}

func TestRONActivationTimingInvalid(t *testing.T) {
	for _, modifier := range []string{
		`ActivateOnlyIf(PlayerPassesFilter(You, IsTheirTurn()))`,
		`ActivateOnlyIf(PlayerPassesFilter(You, IsTheirUpkeep()))`,
		`ActivateOnlyIf(PlayerPassesFilter(Ref_TargetPlayer, IsTheirTurn))`,
		`ActivateOnlyIf(PlayerPassesFilter(Opponent, IsTheirUpkeep))`,
		`ActivateOnlyIf(PlayerPassesFilter(SinglePlayer(You), IsTheirTurn))`,
		`ActivateOnlyIf(PlayerPassesFilter(You, NumCardsInHandIs(EqualTo(Integer(7)))))`,
		`ActivateOnlyIf(PlayerPassesFilter(You))`, `ActivateOnlyIf(You)`,
		`ActivateOnlyOnceEachTurn`, `And([])`, `And([ActivateOnlyIf(PlayerPassesFilter(You, IsTheirTurn)), ActivateOnlyOnceEachTurn])`,
	} {
		src := fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [ActivatedWithModifiers(PayMana([]), ActionList([GainLife(Integer(1))]), %s)])`, modifier)
		nodes, err := Parse(strings.NewReader(src), "invalid-timing.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", modifier)
		}
	}
}

func TestRONActivationTimingInstancesAndCopy(t *testing.T) {
	g, a, b := ronGame()
	g.SetStep(core.Upkeep)
	sourceCard := repositoryCard(t, "FEM", "Svyelunite Priest")
	sourceCard.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(sourceCard, a.PlayerID())
	source.RevokeBaseAttr(core.AttrSummonSick)
	otherCard := repositoryCard(t, "FEM", "Svyelunite Priest")
	otherCard.SetOwner(a.PlayerID())
	other := g.PutOnBattlefield(otherCard, a.PlayerID())
	other.RevokeBaseAttr(core.AttrSummonSick)
	if sourceCard.Abilities()[0] == otherCard.Abilities()[0] {
		t.Fatal("factories share ability state")
	}
	a.ManaPool().Add(core.Blue, 4)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{source.ID()}); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), other.ID(), 0, []uuid.UUID{other.ID()}); err != nil {
		t.Fatal(err)
	}
	g.SetActivePlayerIndex(1)
	g.SetStep(core.EndStep)
	g.PhaseOut(other.ID())
	copied := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), true)
	if copied == nil || len(copied.Targets) != 1 || copied.Targets[0] != source.ID() {
		t.Fatal("copy did not retain source and retarget")
	}
	g.ResolveTopOfStack()
	if !g.FindPermanent(source.ID()).HasAttr(core.Shroud) {
		t.Fatal("copy lost typed target")
	}
	g.PhaseIn([]uuid.UUID{other.ID()})
	g.ResolveTopOfStack()
	if !g.FindPermanent(other.ID()).HasAttr(core.Shroud) {
		t.Fatal("copy changed original target")
	}
	g.ResolveStack()
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	if g.FindPermanent(source.ID()).HasAttr(core.Shroud) || g.FindPermanent(other.ID()).HasAttr(core.Shroud) {
		t.Fatal("self grants did not expire")
	}
}

func TestRONActivationTimingRepeatAndInventory(t *testing.T) {
	rule := `ActivatedWithModifiers(PayMana([]), ActionList([GainLife(Integer(1))]), ActivateOnlyIf(PlayerPassesFilter(You, IsTheirUpkeep)))`
	factory := permanentScopeFactory(t, "Artifact", rule)
	g, a, _ := ronGame()
	g.SetStep(core.Upkeep)
	card := factory()
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	for range 2 {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
	}
	g.SetStep(core.EndStep)
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatal("timing imposed a use limit or resolution restriction")
	}
	src := fmt.Sprintf(`Card(Name: "Timing inventory", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [%s])`, rule)
	nodes, err := Parse(strings.NewReader(src), "timing.ron")
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string]*InventorySignature{}
	walkInventory(nodes[0], "Timing inventory", "Card", "card", "collection", true, entries)
	for name, role := range map[string]string{"PayMana": "cost", "ActionList": "actions", "ActivateOnlyIf": "activation_modifier", "PlayerPassesFilter": "activation_condition", "You": "player_reference", "IsTheirUpkeep": "activation_player_filter"} {
		found := false
		for _, entry := range entries {
			if entry.Name == name && entry.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s lacks %s role", name, role)
		}
	}
	for _, rule := range []string{
		`ActivatedWithModifiers(PayMana([]), ActionList([]))`,
		`ActivatedWithModifiers(PayMana([]), ActionList([]), ActivateOnlyIf(PlayerPassesFilter(You, IsTheirTurn)), Flying)`,
		`ActivatedWithModifiers(PayMana([ManaCostX]), ActionList([]), ActivateOnlyIf(PlayerPassesFilter(You, IsTheirTurn)))`,
		`ActivatedWithModifiers(PayMana([]), ActionList([TapPermanent(Ref_TargetPermanent)]), ActivateOnlyIf(PlayerPassesFilter(You, IsTheirTurn)))`,
	} {
		nodes, err := Parse(strings.NewReader(fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [%s])`, rule)), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
	nodes, err = Parse(strings.NewReader(fmt.Sprintf(`Card(Name: "Invalid spell", Typeline: (Supertypes: [], Cardtypes: [Instant], Subtypes: []), Rules: [%s])`, rule)), "spell.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Compile(nodes[0]); err == nil {
		t.Fatal("accepted activated modifier on instant")
	}
}

func TestRONActivationTimingManaPlanning(t *testing.T) {
	for _, filter := range []string{"IsTheirTurn", "IsTheirUpkeep"} {
		g, a, _ := ronGame()
		factory := permanentScopeFactory(t, "Land", fmt.Sprintf(`ActivatedWithModifiers(TapPermanent(ThisPermanent), ActionList([AddMana(ManaProduceU)]), ActivateOnlyIf(PlayerPassesFilter(You, %s)))`, filter))
		card := factory()
		card.SetOwner(a.PlayerID())
		source := g.PutOnBattlefield(card, a.PlayerID())
		for _, state := range []struct {
			active  int
			step    core.PhaseStep
			allowed bool
		}{{1, core.Upkeep, false}, {0, core.PrecombatMain, filter == "IsTheirTurn"}, {0, core.Upkeep, true}} {
			g.SetActivePlayerIndex(state.active)
			g.SetStep(state.step)
			if g.CanAfford(a.PlayerID(), core.ParseManaCost("{U}"), nil) != state.allowed {
				t.Fatal("mana solver ignored timing")
			}
			if g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != 0 {
				t.Fatal("mana query changed state")
			}
		}
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		if !g.GetStack().IsEmpty() || a.ManaPool().TotalMana() != 1 {
			t.Fatal("mana ability did not resolve immediately")
		}
	}
}
