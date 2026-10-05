package ron

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const ronPlayerPlaneswalkerTarget = `TargetPlayerOrPermanent(AnyPlayer,IsCardtype(Planeswalker))`
const ronPlayerPlaneswalkerDamage = `Targeted([` + ronPlayerPlaneswalkerTarget + `],ActionList([SpellDealsDamage(ThisSpell,Integer(2),Ref_TargetPlayerOrPermanent)]))`

func TestRONPlayerPlaneswalkerDeclarations(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(`+ronPlayerPlaneswalkerDamage+`)`)
	first, second := factory(), factory()
	left := first.Abilities()[0].(*mage.ActionDefinition)
	if left.Targets()[0] == second.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
		t.Fatal("factory instances share a declaration")
	}
	for _, card := range []mage.Card{first, second} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	missing := mage.NewInstant("Missing", "", mage.WithAbility(mage.NewAction(mage.ActionSpell, mage.WithEffects(left.Effects()...))))
	if mage.ValidateCardDefinition(missing.Name(), missing) == nil {
		t.Fatal("undeclared recipient accepted")
	}
	for _, target := range []string{
		`TargetPlayerOrPermanent`, `TargetPlayerOrPermanent()`, `TargetPlayerOrPermanent(AnyPlayer)`,
		`TargetPlayerOrPermanent(AnyPlayer,IsCardtype(Planeswalker),AnyPlayer)`,
		`TargetPlayerOrPermanent(AnyPlayer(),IsCardtype(Planeswalker))`,
		`TargetPlayerOrPermanent(Opponent(),IsCardtype(Planeswalker))`,
		`TargetPlayerOrPermanent(Ref_TargetPlayer,IsCardtype(Planeswalker))`,
		`TargetPlayerOrPermanent(AnyPlayer,IsCardtype(Creature))`,
		`TargetPlayerOrPermanent(AnyPlayer,IsCardtype(Planeswalker()))`,
		`TargetPlayerOrPermanent(AnyPlayer,IsCardtype(Planeswalker,Creature))`,
	} {
		t.Run(target, func(t *testing.T) {
			src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([` + target + `],ActionList([])))])`
			nodes, err := Parse(strings.NewReader(src), "invalid.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = Compile(nodes[0]); err == nil {
				t.Fatal("invalid target accepted")
			}
		})
	}
	for _, rule := range []string{
		`SpellActions(ActionList([SpellDealsDamage(ThisSpell,Integer(1),Ref_TargetPlayerOrPermanent)]))`,
		strings.Replace(ronPlayerPlaneswalkerDamage, "Ref_TargetPlayerOrPermanent", "Ref_AnyTarget", 1),
		strings.Replace(ronPlayerPlaneswalkerDamage, "Ref_TargetPlayerOrPermanent", "Ref_TargetPermanent", 1),
		strings.Replace(ronPlayerPlaneswalkerDamage, "Ref_TargetPlayerOrPermanent", "Ref_TargetPlayerOrPermanent()", 1),
		`SpellActions(Targeted([` + ronPlayerPlaneswalkerTarget + `],ActionList([DestroyPermanent(Ref_TargetPlayerOrPermanent)])))`,
		`SpellActions(Targeted([` + ronPlayerPlaneswalkerTarget + `],ActionList([PlayerAction(Ref_TargetPlayerOrPermanent,GainLife(Integer(1)))])))`,
		`SpellActions(Modal_ChooseOne([` + ronPlayerPlaneswalkerDamage + `,ActionList([SpellDealsDamage(ThisSpell,Integer(1),Ref_TargetPlayerOrPermanent)])]))`,
		`Activated(Ref_TargetPlayerOrPermanent,ActionList([]))`,
	} {
		if strings.HasPrefix(rule, "Targeted") {
			rule = `SpellActions(` + rule + `)`
		}
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted incompatible or unbound read: %s", rule)
		}
	}
}

func TestRONPlayerPlaneswalkerWitchHunter(t *testing.T) {
	for _, scenario := range []string{"opponent", "self", "planeswalker", "creature", "land", "sick", "tapped", "phase-source", "depart-source", "return-source", "control-source", "prevent", "copy"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "DRK", "Witch Hunter")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			if scenario != "sick" {
				source.RevokeBaseAttr(core.AttrSummonSick)
			}
			target := b.PlayerID()
			if scenario == "self" {
				target = a.PlayerID()
			}
			if scenario == "planeswalker" {
				target = g.PutOnBattlefield(mage.NewPlaneswalker("Walker", "", 5), b.PlayerID()).ID()
			}
			if scenario == "creature" {
				target = g.PutOnBattlefield(mage.NewCreature("Creature", "", 1, 5), b.PlayerID()).ID()
			}
			if scenario == "land" {
				target = g.PutOnBattlefield(mage.NewLand("Land"), b.PlayerID()).ID()
			}
			if scenario == "tapped" {
				g.TapPermanent(source)
			}
			err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target})
			if scenario == "creature" || scenario == "land" || scenario == "sick" || scenario == "tapped" {
				if err == nil || g.StackSize() != 0 || g.FindPermanent(source.ID()).Tapped != (scenario == "tapped") {
					t.Fatal("invalid activation changed state")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !g.FindPermanent(source.ID()).Tapped || a.Life() != 20 || b.Life() != 20 {
				t.Fatal("wrong cost or resolution timing")
			}
			switch scenario {
			case "phase-source":
				g.PhaseOut(source.ID())
			case "depart-source", "return-source":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				if scenario == "return-source" {
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "control-source":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "prevent":
				g.AddPreventionShield(target, 1)
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			g.ResolveStack()
			if clone.GetPlayer(b.PlayerID()).Life() != 20 {
				t.Fatal("resolution changed clone")
			}
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				wantA, wantB := 20, 19
				if scenario == "self" {
					wantA, wantB = 19, 20
				}
				if scenario == "prevent" || scenario == "planeswalker" {
					wantB = 20
				}
				if scenario == "copy" {
					wantB = 18
				}
				if game.GetPlayer(a.PlayerID()).Life() != wantA || game.GetPlayer(b.PlayerID()).Life() != wantB {
					t.Fatal("wrong player or lost source information")
				}
				if scenario == "planeswalker" && game.FindPermanent(target).Counters[core.Loyalty] != 4 {
					t.Fatal("damage did not remove loyalty")
				}
			}
		})
	}
}

func TestRONPlayerPlaneswalkerWitchHunterBounceAndCast(t *testing.T) {
	for _, scenario := range []string{"cast", "bounce", "own", "unpaid", "control", "return", "phase", "shroud", "source-depart"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "DRK", "Witch Hunter")
			card.SetOwner(a.PlayerID())
			if scenario == "cast" {
				a.AddToHand(card)
				a.ManaPool().Add(core.White, 1)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || a.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
					t.Fatal("unpaid cast changed state")
				}
				a.ManaPool().Add(core.White, 1)
				a.ManaPool().Add(core.Colorless, 2)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				if g.FindPermanent(card.ID()) != nil {
					t.Fatal("entered before resolution")
				}
				g.ResolveStack()
				p := g.FindPermanent(card.ID())
				if p == nil || p.CurrentPower(g) != 1 || p.CurrentToughness(g) != 1 || !p.HasAttr(core.AttrSummonSick) || len(p.Card.Abilities()) != 2 {
					t.Fatal("incomplete creature definition")
				}
				return
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			victimCard := mage.NewCreature("Victim", "", 2, 4)
			victimCard.SetOwner(a.PlayerID())
			controller := b.PlayerID()
			if scenario == "own" {
				controller = a.PlayerID()
			}
			victim := g.PutOnBattlefield(victimCard, controller)
			a.ManaPool().Add(core.White, 2)
			if scenario != "unpaid" {
				a.ManaPool().Add(core.Colorless, 1)
			}
			err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{victim.ID()})
			if scenario == "own" || scenario == "unpaid" {
				wantMana := 3
				if scenario == "unpaid" {
					wantMana = 2
				}
				if err == nil || g.StackSize() != 0 || source.Tapped || a.ManaPool().TotalMana() != wantMana || g.FindPermanent(victim.ID()) == nil {
					t.Fatal("failed bounce changed resources")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != 0 || g.FindPermanent(victim.ID()) == nil {
				t.Fatal("bounce timing or costs")
			}
			switch scenario {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: victim.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "return":
				g.RemoveFromBattlefield(victim)
				g.PutOnBattlefield(victimCard, b.PlayerID())
			case "phase":
				g.PhaseOut(victim.ID())
			case "shroud":
				g.MutablePermanent(victim.ID()).GrantBaseAttr(core.Shroud)
			case "source-depart":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			}
			g.ResolveStack()
			moved := scenario == "bounce" || scenario == "source-depart"
			if (g.FindPermanentIncludingPhased(victim.ID()) == nil) != moved || (len(a.Hand()) == 1) != moved || len(b.Hand()) != 0 {
				t.Fatal("bounce did not use current legality and owner")
			}
		})
	}
}

func TestRONPlayerPlaneswalkerAxelrod(t *testing.T) {
	for _, scenario := range []string{"player", "planeswalker", "undamaged", "other-source", "previous-turn", "prevented-history", "two-deaths", "simultaneous-source", "depart-source", "control-source", "phase-source", "prevent", "illegal-target", "copy", "trample"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronModalGame(0)
			card := repositoryCard(t, "LEG", "Axelrod Gunnarson")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			if source.CurrentPower(g) != 5 || source.CurrentToughness(g) != 5 || !source.HasKeyword(core.Trample) {
				t.Fatal("wrong characteristics")
			}
			target := b.PlayerID()
			if scenario == "planeswalker" || scenario == "illegal-target" {
				target = g.PutOnBattlefield(mage.NewPlaneswalker("Walker", "", 5), b.PlayerID()).ID()
			}
			a.targets = []uuid.UUID{target}
			victimCard := mage.NewCreature("Victim", "", 1, 2)
			victimCard.SetOwner(b.PlayerID())
			victim := g.PutOnBattlefield(victimCard, b.PlayerID())
			if scenario == "trample" {
				source.RevokeBaseAttr(core.AttrSummonSick)
				g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{source.ID()})
				g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: victim.ID(), AttackerID: source.ID()}})
				g.ExecuteCombatDamage()
				if b.Life() != 17 || g.FindPermanent(victim.ID()) != nil {
					t.Fatal("trample and lethal damage")
				}
			} else {
				damageSource := source.ID()
				if scenario == "other-source" {
					damageSource = g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 4), a.PlayerID()).ID()
				}
				if scenario == "prevented-history" {
					g.AddPreventionShield(victim.ID(), 1)
				}
				if scenario != "undamaged" {
					g.DealDamageToPermanent(victim, 1, damageSource)
				}
				if scenario == "previous-turn" {
					g.RunStepWithPriority(core.Cleanup)
				}
				switch scenario {
				case "two-deaths":
					otherCard := mage.NewCreature("Other victim", "", 1, 3)
					otherCard.SetOwner(b.PlayerID())
					other := g.PutOnBattlefield(otherCard, b.PlayerID())
					g.DealDamageToPermanent(other, 1, source.ID())
					g.DestroyPermanents([]*mage.Permanent{victim, other})
				case "simultaneous-source":
					g.DestroyPermanents([]*mage.Permanent{source, victim})
				default:
					g.DestroyPermanent(victim)
				}
			}
			count := 1
			if slices.Contains([]string{"undamaged", "other-source", "previous-turn", "prevented-history"}, scenario) {
				count = 0
			}
			if scenario == "two-deaths" {
				count = 2
			}
			g.PutTriggersOnStack()
			if g.StackSize() != count || a.Life() != 20 || count > 0 && g.StackPeek().Targets[0] != target {
				t.Fatal("wrong event history, target selection or timing")
			}
			switch scenario {
			case "depart-source":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.PutOnBattlefield(card, b.PlayerID())
			case "control-source":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "phase-source":
				g.PhaseOut(source.ID())
			case "prevent":
				g.AddPreventionShield(target, 1)
			case "illegal-target":
				g.PhaseOut(target)
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			g.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("resolution changed clone")
			}
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				wantA, wantB := 20+count, 20-count
				if scenario == "planeswalker" || scenario == "prevent" {
					wantB = 20
				}
				if scenario == "illegal-target" {
					wantA, wantB = 20, 20
				}
				if scenario == "copy" {
					wantB = 19
				}
				if scenario == "trample" {
					wantB = 16
				}
				if game.GetPlayer(a.PlayerID()).Life() != wantA || game.GetPlayer(b.PlayerID()).Life() != wantB {
					t.Fatalf("life=%d/%d want=%d/%d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life(), wantA, wantB)
				}
				if scenario == "planeswalker" && game.FindPermanent(target).Counters[core.Loyalty] != 4 {
					t.Fatal("planeswalker damage")
				}
			}
		})
	}
}

func TestRONPlayerPlaneswalkerSpellModesInstancesAndRetarget(t *testing.T) {
	for mode := range 2 {
		g, a, b := ronModalGame(mode)
		factory := permanentScopeFactory(t, "Instant", `SpellActions(Modal_ChooseOne([`+ronPlayerPlaneswalkerDamage+`,`+ronPlayerPlaneswalkerDamage+`]))`)
		for _, target := range []uuid.UUID{a.PlayerID(), b.PlayerID()} {
			card := factory()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, 0); err != nil {
				t.Fatal(err)
			}
		}
		a.targets = []uuid.UUID{a.PlayerID()}
		cp := g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), true)
		if cp == nil || cp.ModeChoice != mode || cp.Targets[0] != a.PlayerID() {
			t.Fatal("copy lost mode or new recipient")
		}
		g.ResolveTopOfStack()
		if a.Life() != 18 || b.Life() != 20 {
			t.Fatal("retarget changed original")
		}
		g.ResolveTopOfStack()
		if b.Life() != 18 {
			t.Fatal("second instance lost its target")
		}
		g.ResolveTopOfStack()
		if a.Life() != 16 {
			t.Fatal("first instance lost its target")
		}
	}
}

func TestRONPlayerPlaneswalkerTargetIdentity(t *testing.T) {
	for _, change := range []string{"control", "return", "phase", "phase-cycle", "shroud", "type"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "DRK", "Witch Hunter")
			source := g.PutOnBattlefield(card, a.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			walkerCard := mage.NewPlaneswalker("Walker", "", 5)
			walker := g.PutOnBattlefield(walkerCard, b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{walker.ID()}); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: walker.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "return":
				g.RemoveFromBattlefield(walker)
				walker = g.PutOnBattlefield(walkerCard, b.PlayerID())
			case "phase", "phase-cycle":
				phased := g.PhaseOut(walker.ID())
				if change == "phase-cycle" {
					g.PhaseIn(phased)
				}
			case "shroud":
				g.MutablePermanent(walker.ID()).GrantBaseAttr(core.Shroud)
			case "type":
				g.MutablePermanent(walker.ID()).Card.Types()[0] = core.TypeArtifact
			}
			g.ResolveStack()
			want := uint8(5)
			if change == "control" || change == "phase-cycle" {
				want = 4
			}
			if g.FindPermanentIncludingPhased(walker.ID()).Counters[core.Loyalty] != want {
				t.Fatal("target legality or incarnation")
			}
		})
	}
}

func TestRONPlayerPlaneswalkerAxelrodCasting(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEG", "Axelrod Gunnarson")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Black, 2)
	a.ManaPool().Add(core.Red, 1)
	a.ManaPool().Add(core.Colorless, 4)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || a.ManaPool().TotalMana() != 7 || g.StackSize() != 0 || len(a.Hand()) != 1 {
		t.Fatal("missing red mana changed resources")
	}
	a.ManaPool().Add(core.Red, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().TotalMana() != 0 || g.FindPermanent(card.ID()) != nil || len(g.StackPeek().Targets) != 0 {
		t.Fatal("creature casting chose a trigger target or entered early")
	}
	g.ResolveStack()
	source := g.FindPermanent(card.ID())
	if source == nil || !source.HasAttr(core.AttrSummonSick) || source.CurrentPower(g) != 5 || source.CurrentToughness(g) != 5 || !source.HasKeyword(core.Trample) || !source.Card.HasSuperType(core.SuperLegendary) {
		t.Fatal("incomplete entered creature")
	}
	if a.Life() != 20 || b.Life() != 20 || g.StackSize() != 0 {
		t.Fatal("death trigger fired on entry")
	}
}

func TestRONPlayerPlaneswalkerPacketsAndReplacementOrder(t *testing.T) {
	for mode := range 2 {
		a := mage.NewBasePlayer("A")
		b := &ronModePlayer{BasePlayer: mage.NewBasePlayer("B"), mode: mode}
		g := mage.NewGame(a, b)
		g.SetStep(core.PrecombatMain)
		g.SetTurn(1)
		factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([`+ronPlayerPlaneswalkerTarget+`],ActionList([SpellDealsMultipleDamage(ThisSpell,[DamageToRecipients(Integer(4),Ref_TargetPlayerOrPermanent),DamageToRecipients(Integer(2),Player(You))])])))`)
		card := factory()
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
			t.Fatal(err)
		}
		g.AddHalfDamageFromSourcePreventionShield(b.PlayerID(), card.ID())
		g.AddPreventionShield(b.PlayerID(), 1)
		g.ResolveStack()
		if a.Life() != 18 || b.Life() != 19-mode || b.prompts != 1 {
			t.Fatal("typed union bypassed damage batches or recipient replacement order")
		}
	}
}

func TestRONPlayerPlaneswalkerPreventionRecipient(t *testing.T) {
	for _, kind := range []string{"player", "planeswalker"} {
		t.Run(kind, func(t *testing.T) {
			g, a, b := ronGame()
			factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([`+ronPlayerPlaneswalkerTarget+`],ActionList([CreateFuturePreventDamage(NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(Integer(1),Ref_TargetPlayerOrPermanent),[PreventThatDamage])])))`)
			card := factory()
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			target := b.PlayerID()
			if kind == "planeswalker" {
				target = g.PutOnBattlefield(mage.NewPlaneswalker("Walker", "", 5), b.PlayerID()).ID()
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if kind == "player" {
				g.DealDamageToPlayer(b, 3, card.ID())
				if b.Life() != 18 || a.Life() != 20 {
					t.Fatal("shield lost its player recipient")
				}
			} else {
				g.DealDamageToPermanent(g.FindPermanent(target), 3, card.ID())
				if g.FindPermanent(target).Counters[core.Loyalty] != 3 {
					t.Fatal("shield lost its planeswalker recipient")
				}
			}
		})
	}
}

func TestRONPlayerPlaneswalkerStrictLoadAndInventory(t *testing.T) {
	files := fstest.MapFS{}
	for _, set := range []string{"LEG", "DRK"} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			for _, name := range []string{"Axelrod Gunnarson", "Witch Hunter"} {
				if strings.Contains(line, `Name: "`+name+`"`) {
					files[name+".ron"] = &fstest.MapFile{Data: []byte(line)}
				}
			}
		}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil || len(registry.RegisteredCardNames()) != 2 {
		t.Fatalf("complete records: %v", err)
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Invalid",Rules:[Unknown])`)}
	if registry, err = LoadFS(files, "*.ron"); err == nil || registry != nil {
		t.Fatal("partial collection published")
	}
	files["nested.ron"] = &fstest.MapFile{Data: []byte(fmt.Sprintf(`Card(Name:"Nested",Rules:[Unknown(Targeted([%s],ActionList([])))])`, ronPlayerPlaneswalkerTarget))}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	roles := map[string]string{"AnyPlayer": "player_filter", "IsCardtype": "permanent_filter", "Planeswalker": "card_type"}
	for name, role := range roles {
		found := false
		for _, sig := range report.Signatures {
			if sig.Name == name && strings.Contains(sig.Context, "TargetPlayerOrPermanent") && sig.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing target descendant role %s/%s", name, role)
		}
	}
	unknown := false
	for _, sig := range report.Signatures {
		if sig.Name == "Planeswalker" && strings.Contains(sig.Context, "Unknown") {
			unknown = true
			if sig.Role != "unknown" {
				t.Fatal("unsupported parent invented a child contract")
			}
		}
	}
	if !unknown {
		t.Fatal("inventory skipped an unsupported descendant")
	}
}
