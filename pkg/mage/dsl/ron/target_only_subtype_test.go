package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTargetOnlySubtypeWall(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEG", "Wall of Shadows")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Black, 2)
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	wall := g.FindPermanent(card.ID())
	wall.RevokeBaseAttr(core.AttrSummonSick)
	if wall.CanDeclareAsAttacker(g) {
		t.Fatal("defender attacked")
	}
	aura := repositoryCard(t, "LEB", "Animate Wall")
	aura.SetOwner(b.PlayerID())
	b.AddToHand(aura)
	b.ManaPool().Add(core.White, 1)
	if err := g.CastSpellByID(b.PlayerID(), aura.ID(), []uuid.UUID{wall.ID()}, 0); err == nil || b.ManaPool().Count(core.White) != 1 || g.StackSize() != 0 {
		t.Fatal("wall Aura paid for illegal target")
	}
	spell := repositoryCard(t, "LEB", "Giant Growth")
	spell.SetOwner(b.PlayerID())
	b.AddToHand(spell)
	b.ManaPool().Add(core.Green, 1)
	if err := g.CastSpellByID(b.PlayerID(), spell.ID(), []uuid.UUID{wall.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(wall.ID()).CurrentPower(g) != 3 {
		t.Fatal("broad spell restricted")
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	demolition := g.PutOnBattlefield(repositoryCard(t, "LEB", "Dwarven Demolition Team"), b.PlayerID())
	demolition.RevokeBaseAttr(core.AttrSummonSick)
	if err := g.ActivateAbilityByIndex(b.PlayerID(), demolition.ID(), 0, []uuid.UUID{wall.ID()}); err == nil || g.FindPermanent(demolition.ID()).Tapped {
		t.Fatal("wall ability paid for illegal target")
	}
	attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 4, 5), b.PlayerID())
	other := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 5), b.PlayerID())
	attacker.RevokeBaseAttr(core.AttrSummonSick)
	g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{attacker.ID()})
	g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: wall.ID(), AttackerID: attacker.ID()}})
	clone := g.Clone()
	g.DealDamageToPermanent(g.FindPermanent(wall.ID()), 4, attacker.ID())
	if g.FindPermanent(wall.ID()).Damage != 0 {
		t.Fatal("blocked creature damage not prevented")
	}
	g.DealDamageToPermanent(g.FindPermanent(wall.ID()), 1, other.ID())
	if g.FindPermanent(wall.ID()).Damage != 1 || clone.FindPermanent(wall.ID()).Damage != 0 {
		t.Fatal("unrelated damage or clone isolation")
	}
	clone.DealDamageToPermanent(clone.FindPermanent(wall.ID()), 4, attacker.ID())
	if clone.FindPermanent(wall.ID()).Damage != 0 {
		t.Fatal("cloned prevention failed")
	}
}

func TestRONTargetOnlySubtypeInvalid(t *testing.T) {
	for _, operation := range []string{"CantBeTheTargetOfSpells", "CantBeTheTargetOfAbilities"} {
		for _, filter := range []string{`CanTargetOnly`, `CanTargetOnly()`, `CanTargetOnly(IsCreatureType(Wall),IsCreatureType(Elf))`, `CanTargetOnly(IsCreatureType())`, `CanTargetOnly(IsCreatureType(Wall()))`, `CanTargetOnly(IsCreatureType("Wall"))`, `CanTargetOnly(IsCardtype(Creature))`, `CanTargetOnly(Or([IsCreatureType(Wall),IsCreatureType(Elf)]))`} {
			source := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[Wall]),CardPT:(Power:0,Toughness:1),Rules:[PermanentRuleEffect(ThisPermanent,[` + operation + `(` + filter + `)])])`
			if registry, err := LoadFS(fstest.MapFS{"bad.ron": {Data: []byte(source)}}, "*.ron"); err == nil || registry != nil {
				t.Fatalf("published %s", source)
			}
		}
	}
	for _, subject := range []string{"HostPermanent", "Ref_TargetPermanent", "ThisPermanent()"} {
		source := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[Wall]),Rules:[PermanentRuleEffect(` + subject + `,[CantBeTheTargetOfSpells(CanTargetOnly(IsCreatureType(Wall)))])])`
		nodes, err := Parse(strings.NewReader(source), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("invalid scope accepted")
		}
	}
}

func TestRONTargetOnlySubtypeAbilityLifecycle(t *testing.T) {
	for _, change := range []string{"present", "source-left", "source-return", "target-return", "target-phase", "phase-cycle", "removed", "control"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			wallCard := repositoryCard(t, "LEG", "Wall of Shadows")
			wall := g.PutOnBattlefield(wallCard, b.PlayerID())
			factory := permanentScopeFactory(t, "Artifact", `Activated(TapPermanent(ThisPermanent),Targeted([TargetPermanent(IsCreatureType(Wall))],ActionList([TapPermanent(Ref_TargetPermanent)]))),Activated(TapPermanent(ThisPermanent),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([TapPermanent(Ref_TargetPermanent)])))`)
			card := factory()
			source := g.PutOnBattlefield(card, a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{wall.ID()}); err == nil || g.FindPermanent(source.ID()).Tapped || g.StackSize() != 0 {
				t.Fatal("other source ability supplied proof or illegal target paid")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{wall.ID()}); err != nil {
				t.Fatal(err)
			}
			if mage.TargetSourceCanTargetOnlySubtype(g.StackPeek().TargetSource, "Wall") {
				t.Fatal("broad selected ability got wall proof")
			}
			switch change {
			case "source-left", "source-return":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				if change == "source-return" {
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "target-return":
				g.RemoveFromBattlefield(g.FindPermanent(wall.ID()))
				g.PutOnBattlefield(wallCard, b.PlayerID())
			case "target-phase":
				g.PhaseOut(wall.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(wall.ID()))
			case "removed":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), wall.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: wall.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			}
			clone := g.Clone()
			copied := g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
			if copied == nil || mage.TargetSourceCanTargetOnlySubtype(copied.TargetSource, "Wall") {
				t.Fatal("copy lost selected declarations")
			}
			g.ResolveStack()
			clone.ResolveStack()
			want := change != "target-return" && change != "target-phase"
			for _, game := range []*mage.Game{g, clone} {
				if game.FindPermanentIncludingPhased(wall.ID()).Tapped != want {
					t.Fatal("resolution visit or source identity")
				}
			}
		})
	}
}

func TestRONTargetOnlySubtypeResolutionRecheck(t *testing.T) {
	for _, ability := range []bool{false, true} {
		g, a, b := ronGame()
		wall := g.PutOnBattlefield(repositoryCard(t, "LEG", "Wall of Shadows"), b.PlayerID())
		if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), wall.ID(), b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if ability {
			source := g.PutOnBattlefield(repositoryCard(t, "LEB", "Dwarven Demolition Team"), a.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{wall.ID()}); err != nil {
				t.Fatal(err)
			}
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
		} else {
			spell := repositoryCard(t, "LEB", "Tunnel")
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(core.Red, 1)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{wall.ID()}, 0); err != nil {
				t.Fatal(err)
			}
		}
		if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
			t.Fatal("missing copy")
		}
		g.GetEffects().RemoveEndOfTurn()
		g.ApplyContinuousEffects()
		clone := g.Clone()
		g.ResolveStack()
		clone.ResolveStack()
		if g.FindPermanent(wall.ID()) == nil || clone.FindPermanent(wall.ID()) == nil {
			t.Fatal("restored restriction did not invalidate targets")
		}
		if !ability && a.ManaPool().TotalMana() != 0 {
			t.Fatal("fizzle refunded mana")
		}
	}
}

func TestRONTargetOnlySubtypeZeroGroups(t *testing.T) {
	for _, kind := range []string{"IsCardtype(Creature)", "IsCreatureType(Wall)"} {
		for _, activated := range []bool{false, true} {
			g, a, _ := ronGame()
			body := `Targeted([NumberTargetPermanents(Integer(0),` + kind + `)],ActionList([TapEachPermanent(Ref_TargetPermanents)]))`
			types, rule := "Instant", `SpellActions(`+body+`)`
			if activated {
				types, rule = "Artifact", `Activated(PayMana([ManaCostGeneric(0)]),`+body+`)`
			}
			card := permanentScopeFactory(t, types, rule)()
			var source mage.Card
			if activated {
				permanent := g.PutOnBattlefield(card, a.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), permanent.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				source = g.StackPeek().TargetSource
			} else {
				source = mage.SpellTargetSource(card)
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
			}
			if len(g.StackPeek().TargetSpecs) != 0 {
				t.Fatal("zero group chose targets")
			}
			want := kind == "IsCreatureType(Wall)"
			if mage.TargetSourceCanTargetOnlySubtype(source, "Wall") != want {
				t.Fatal("zero group declaration lost")
			}
			clone := g.Clone()
			copied := g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), true)
			if activated {
				for _, view := range []mage.Card{clone.StackPeek().TargetSource, copied.TargetSource} {
					if mage.TargetSourceCanTargetOnlySubtype(view, "Wall") != want {
						t.Fatal("zero group copy lost")
					}
				}
			}
			g.ResolveStack()
			clone.ResolveStack()
		}
	}
}

func TestRONTargetOnlySubtypeTriggered(t *testing.T) {
	for _, kind := range []string{"IsCreatureType(Wall)", "IsCardtype(Creature)"} {
		g, a, b := ronGame()
		wall := g.PutOnBattlefield(repositoryCard(t, "LEG", "Wall of Shadows"), b.PlayerID())
		card := permanentScopeFactory(t, "Artifact", `TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),Targeted([TargetPermanent(`+kind+`)],ActionList([TapPermanent(Ref_TargetPermanent)])))`)()
		source := g.PutOnBattlefield(card, a.PlayerID())
		g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
		g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
		g.PutOnBattlefield(card, b.PlayerID())
		clone := g.Clone()
		for _, game := range []*mage.Game{g, clone} {
			game.PutTriggersOnStack()
			if kind == "IsCreatureType(Wall)" {
				if game.StackSize() != 0 {
					t.Fatal("wall-only trigger found target")
				}
				continue
			}
			if game.StackSize() != 1 || mage.TargetSourceCanTargetOnlySubtype(game.StackPeek().TargetSource, "Wall") {
				t.Fatal("pending trigger source changed declarations")
			}
			game.ResolveStack()
			if !game.FindPermanent(wall.ID()).Tapped {
				t.Fatal("broad trigger restricted")
			}
		}
	}
}

func TestRONTargetOnlySubtypeHost(t *testing.T) {
	data := `Card(Name:"Host restriction",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),PermanentRuleEffect(HostPermanent,[CantBeTheTargetOfSpells(CanTargetOnly(IsCreatureType(Wall))),CantBeTheTargetOfAbilities(CanTargetOnly(IsCreatureType(Wall)))])])`
	registry, err := LoadFS(fstest.MapFS{"host.ron": {Data: []byte(data)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	host := g.PutOnBattlefield(mage.NewCreature("Host", "", 0, 5, mage.WithSubTypes("Wall")), a.PlayerID())
	other := g.PutOnBattlefield(mage.NewCreature("Other", "", 0, 5, mage.WithSubTypes("Wall")), b.PlayerID())
	card, err := registry.CreateCard("Host restriction")
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.CreateCard("Host restriction")
	if err != nil || card.Abilities()[0] == second.Abilities()[0] {
		t.Fatal("factory shared declarations")
	}
	aura := g.PutOnBattlefield(card, a.PlayerID())
	g.Attach(aura.ID(), host.ID())
	spell := repositoryCard(t, "LEB", "Tunnel")
	blocked := func(game *mage.Game, id uuid.UUID) bool {
		return !game.FindPermanent(id).CanBeTargetedBy(mage.SpellTargetSource(spell), b.PlayerID(), game)
	}
	if !blocked(g, host.ID()) || blocked(g, other.ID()) {
		t.Fatal("wrong attachment subject")
	}
	clone := g.Clone()
	g.Attach(aura.ID(), other.ID())
	if blocked(g, host.ID()) || !blocked(g, other.ID()) || !blocked(clone, host.ID()) {
		t.Fatal("reattachment or clone isolation")
	}
	g.PhaseOut(aura.ID())
	if blocked(g, other.ID()) {
		t.Fatal("phased source rule active")
	}
	g.PhaseIn([]uuid.UUID{aura.ID()})
	if !blocked(g, other.ID()) {
		t.Fatal("phase cycle lost rule")
	}
	g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
	if blocked(g, other.ID()) {
		t.Fatal("departed source rule active")
	}
	report, err := InspectFS(fstest.MapFS{"host.ron": {Data: []byte(data)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, entry := range report.Signatures {
		if entry.Name == "IsCreatureType" && entry.Role == "permanent_filter" {
			count++
		}
	}
	if count == 0 {
		t.Fatal("nested targeting predicate role missing")
	}
}

func TestRONTargetOnlySubtypeModes(t *testing.T) {
	for _, activated := range []bool{false, true} {
		for _, choice := range []int{0, 1} {
			g, a, b := ronModalGame(choice)
			wall := g.PutOnBattlefield(repositoryCard(t, "LEG", "Wall of Shadows"), b.PlayerID())
			g.PutOnBattlefield(mage.NewCreature("Other Wall", "", 0, 5, mage.WithSubTypes("Wall")), b.PlayerID())
			body := `Modal_ChooseOne([Targeted([TargetPermanent(IsCreatureType(Wall))],ActionList([TapPermanent(Ref_TargetPermanent)])),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([TapPermanent(Ref_TargetPermanent)]))])`
			types, rule := "Instant", `SpellActions(`+body+`)`
			if activated {
				types, rule = "Artifact", `Activated(TapPermanent(ThisPermanent),`+body+`)`
			}
			card := permanentScopeFactory(t, types, rule)()
			a.targets = []uuid.UUID{wall.ID()}
			var err error
			var source *mage.Permanent
			if activated {
				source = g.PutOnBattlefield(card, a.PlayerID())
				err = g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil)
			} else {
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				err = g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{wall.ID()}, 0)
			}
			if choice == 0 {
				if err == nil || g.StackSize() != 0 || (activated && g.FindPermanent(source.ID()).Tapped) || (!activated && len(a.Hand()) != 1) {
					t.Fatal("wall mode announcement did not roll back")
				}
				continue
			}
			if err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			copied := g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), true)
			if copied == nil {
				t.Fatal("copy failed")
			}
			g.ResolveStack()
			clone.ResolveStack()
			if !g.FindPermanent(wall.ID()).Tapped || !clone.FindPermanent(wall.ID()).Tapped {
				t.Fatal("unselected wall mode supplied restriction")
			}
		}
	}
}
