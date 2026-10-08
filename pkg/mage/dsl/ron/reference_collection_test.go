package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONReferenceCollectionTargets(t *testing.T) {
	for _, action := range []string{"TapEachPermanent", "UntapEachPermanent", "PutEachPermanentIntoItsOwnersHand", "DestroyEachPermanent"} {
		for _, change := range []string{"none", "phase", "phase-cycle", "return", "shroud", "control"} {
			t.Run(action+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				rule := `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature)),TargetPermanent(IsCardtype(Creature))],ActionList([` + action + `(Or([SinglePermanent(Ref_TargetPermanent2),SinglePermanent(Ref_TargetPermanent1),SinglePermanent(Ref_TargetPermanent2)]))])))`
				card := permanentScopeFactory(t, "Instant", rule)()
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				victim := mage.NewCreature("First", "", 2, 4)
				victim.SetOwner(b.PlayerID())
				first := g.PutOnBattlefield(victim, b.PlayerID())
				second := g.PutOnBattlefield(mage.NewCreature("Second", "", 2, 4), b.PlayerID())
				if action == "UntapEachPermanent" {
					first.Tapped, second.Tapped = true, true
				}
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{first.ID(), second.ID()}, 0); err != nil {
					t.Fatal(err)
				}
				pending := g.Clone()
				switch change {
				case "phase":
					g.PhaseOut(first.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(first.ID()))
				case "return":
					g.BouncePermanentToHand(g.FindPermanent(first.ID()))
					b.RemoveFromHand(victim.ID())
					g.PutOnBattlefield(victim, b.PlayerID())
				case "shroud":
					g.MutablePermanent(first.ID()).GrantBaseAttr(core.Shroud)
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: second.ID(), TargetID: first.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				}
				g.ResolveStack()
				g.PhaseIn([]uuid.UUID{first.ID()})
				legal := change == "none" || change == "phase-cycle" || change == "control"
				for i, id := range []uuid.UUID{first.ID(), second.ID()} {
					applied := i == 1 || legal
					p := g.FindPermanent(id)
					if action == "TapEachPermanent" || action == "UntapEachPermanent" {
						want := applied
						if action == "UntapEachPermanent" {
							want = !applied
							if change == "return" && i == 0 {
								want = false
							}
						}
						if p == nil || p.Tapped != want {
							t.Fatal("wrong tap state", i, applied)
						}
					} else if (p == nil) != applied {
						t.Fatal("wrong zone action", i, applied)
					}
				}
				if pending.FindPermanent(first.ID()) == nil || pending.FindPermanent(first.ID()).Tapped != (action == "UntapEachPermanent") {
					t.Fatal("clone changed with original")
				}
				pending.ResolveStack()
				if action == "TapEachPermanent" && !pending.FindPermanent(first.ID()).Tapped {
					t.Fatal("clone lost target binding")
				}
			})
		}
	}
}

func TestRONReferenceCollectionPluralIntersection(t *testing.T) {
	g, a, b := ronGame()
	rule := `SpellActions(Targeted([NumberTargetPermanents(Integer(2),IsCardtype(Creature))],ActionList([DestroyEachPermanent(And([Ref_TargetPermanents,HasAbility(Flying)]))])))`
	card := permanentScopeFactory(t, "Instant", rule)()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	first := g.PutOnBattlefield(mage.NewCreature("Flying", "", 2, 4, mage.WithKeyword(core.Flying)), b.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Ground", "", 2, 4), b.PlayerID())
	unselected := g.PutOnBattlefield(mage.NewCreature("Unselected", "", 2, 4, mage.WithKeyword(core.Flying)), b.PlayerID())
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{first.ID(), second.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.MutablePermanent(first.ID()).RevokeBaseAttr(core.Flying)
	g.MutablePermanent(second.ID()).GrantBaseAttr(core.Flying)
	g.ResolveStack()
	if g.FindPermanent(first.ID()) == nil || g.FindPermanent(second.ID()) != nil || g.FindPermanent(unselected.ID()) == nil {
		t.Fatal("intersection lost current membership or target bounds")
	}
}

func TestRONReferenceCollectionInvalid(t *testing.T) {
	for _, subject := range []string{`SinglePermanent`, `SinglePermanent()`, `SinglePermanent(ThisPermanent)`, `SinglePermanent(ThisPermanent,ThisPermanent)`, `SinglePermanent(You)`, `SinglePermanent(Ref_TargetPlayer)`, `SinglePermanent(Trigger_ThatPermanent)`, `SinglePermanent(Ref_TargetPermanent3)`, `Ref_TargetPermanents`, `Ref_TargetPermanents()`, `Or([])`, `And([])`} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([TapEachPermanent(` + subject + `)]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", subject)
		}
	}
}

func TestRONReferenceCollectionScopeIsolation(t *testing.T) {
	for _, rules := range []string{
		`Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([]))),Activated(PayMana([]),ActionList([TapEachPermanent(SinglePermanent(Ref_TargetPermanent))]))`,
		`Activated(PayMana([]),Modal_ChooseOne([Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([])),ActionList([TapEachPermanent(SinglePermanent(Ref_TargetPermanent))])]))`,
		`Activated(PayMana([]),Targeted([NumberTargetPermanents(Integer(1),IsCardtype(Creature))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([TapEachPermanent(Ref_TargetPermanents)]))])))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rules+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("target escaped its supported scope", rules)
		}
	}
}

func TestRONReferenceCollectionWinterBlast(t *testing.T) {
	g, a, b := ronGame()
	g.SetStep(core.PrecombatMain)
	card := repositoryCard(t, "LEG", "Winter Blast")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 8, mage.WithKeyword(core.Flying)), b.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 2, 8, mage.WithKeyword(core.Flying)), b.PlayerID())
	a.ManaPool().Add(core.Green, 1)
	a.ManaPool().Add(core.Colorless, 2)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{first.ID(), second.ID()}, 2); err != nil {
		t.Fatal(err)
	}
	pending := g.Clone()
	g.MutablePermanent(first.ID()).GrantBaseAttr(core.Shroud)
	g.ResolveStack()
	if g.FindPermanent(first.ID()).Tapped || g.FindPermanent(first.ID()).Damage != 0 || !g.FindPermanent(second.ID()).Tapped || g.FindPermanent(second.ID()).Damage != 2 {
		t.Fatal("partial legality diverged between tap and damage")
	}
	pending.ResolveStack()
	if !pending.FindPermanent(first.ID()).Tapped || pending.FindPermanent(first.ID()).Damage != 2 {
		t.Fatal("clone lost original targets")
	}
}

func TestRONReferenceCollectionDelayedCapture(t *testing.T) {
	for _, change := range []string{"none", "return", "phase", "phase-cycle", "shroud"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			rule := `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([TapEachPermanent(SinglePermanent(Ref_TargetPermanent))]))])))`
			card := permanentScopeFactory(t, "Artifact", rule)()
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			targetCard := mage.NewCreature("Target", "", 2, 4)
			target := g.PutOnBattlefield(targetCard, b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			switch change {
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(target.ID()))
				g.PutOnBattlefield(targetCard, b.PlayerID())
			case "phase":
				g.PhaseOut(target.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(target.ID()))
			case "shroud":
				g.MutablePermanent(target.ID()).GrantBaseAttr(core.Shroud)
			}
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				game.RunStepWithPriority(core.EndStep)
				game.ResolveStack()
				game.PhaseIn([]uuid.UUID{target.ID()})
				want := change != "return" && change != "phase"
				if game.FindPermanent(target.ID()).Tapped != want {
					t.Fatal("delayed captured identity", want)
				}
			}
		})
	}
}

func TestRONReferenceCollectionDownstream(t *testing.T) {
	n := hostUntapRecord(t, "PROMOS", "Arena")
	if _, err := Compile(n); err != nil {
		t.Fatal("reference collection consumer rejected", err)
	}
}

func TestRONReferenceCollectionInventory(t *testing.T) {
	src := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([TapEachPermanent(Or([SinglePermanent(Ref_TargetPermanent),IsCardtype(Land)]))])))])`
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 1 {
		t.Fatal(report.Issues)
	}
	for _, signature := range report.Signatures {
		if signature.Name == "SinglePermanent" && signature.Role != "permanent_collection" {
			t.Fatal("wrong singleton role", signature.Role)
		}
		if signature.Name == "Ref_TargetPermanent" && signature.Role != "permanent_reference" {
			t.Fatal("wrong target role", signature.Role)
		}
	}
}

func TestRONReferenceCollectionControllerUnion(t *testing.T) {
	g, a, b := ronGame()
	rule := `Activated(PayMana([]),ActionList([TapEachPermanent(Or([IsCardtype(Land),And([ControlledByAPlayer(SinglePlayer(You)),ControlledByAPlayer(SinglePlayer(You))])]))]))`
	source := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", rule)(), a.PlayerID())
	own := g.PutOnBattlefield(mage.NewCreature("Own", "", 2, 4), a.PlayerID())
	land := g.PutOnBattlefield(mage.NewLand("Land"), b.PlayerID())
	other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 4), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if !g.FindPermanent(source.ID()).Tapped || !g.FindPermanent(own.ID()).Tapped || !g.FindPermanent(land.ID()).Tapped || g.FindPermanent(other.ID()).Tapped {
		t.Fatal("controller union or repeated intersection")
	}
	untapper := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([UntapEachPermanent(ControlledByAPlayer(Opponent))]))`)(), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), untapper.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(land.ID()).Tapped || !g.FindPermanent(own.ID()).Tapped {
		t.Fatal("opponent collection")
	}
}

func TestRONReferenceCollectionCombatIntersection(t *testing.T) {
	for _, filter := range []string{`And([IsCardtype(Land),IsBlockingAttacker(Ref_TargetPermanent)])`, `And([IsCardtype(Creature),IsBlockingAttacker(Ref_TargetPermanent),IsUntapped])`} {
		g, a, b := ronGame()
		attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 2, 8), a.PlayerID())
		blocker := g.PutOnBattlefield(mage.NewCreature("Blocker", "", 2, 8), b.PlayerID())
		attacker.RevokeBaseAttr(core.AttrSummonSick)
		g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{attacker.ID()})
		g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: blocker.ID(), AttackerID: attacker.ID()}})
		card := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([TapEachPermanent(`+filter+`)])))`)()
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{attacker.ID()}, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		want := strings.Contains(filter, "IsUntapped")
		if g.FindPermanent(blocker.ID()).Tapped != want {
			t.Fatal("combat relation ignored intersection", want)
		}
	}
}

func TestRONReferenceCollectionSourceAndHost(t *testing.T) {
	for _, subject := range []string{"ThisPermanent", "HostPermanent"} {
		for _, change := range []string{"none", "phase", "return", "reattach", "host-phase", "host-return"} {
			if subject == "ThisPermanent" && strings.HasPrefix(change, "host-") {
				continue
			}
			t.Run(subject+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				rule := `EnchantPermanent(IsCardtype(Creature)),Activated(PayMana([]),ActionList([TapEachPermanent(SinglePermanent(` + subject + `))]))`
				nodes, err := Parse(strings.NewReader(`Card(Name:"Host probe",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[`+rule+`])`), "host.ron")
				if err != nil {
					t.Fatal(err)
				}
				factory, err := Compile(nodes[0])
				if err != nil {
					t.Fatal(err)
				}
				card := factory()
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 4), b.PlayerID())
				other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 4), b.PlayerID())
				g.Attach(source.ID(), host.ID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				switch change {
				case "phase":
					g.PhaseOut(source.ID())
				case "return":
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
					g.PutOnBattlefield(card, a.PlayerID())
					g.Attach(source.ID(), other.ID())
				case "reattach":
					g.Attach(source.ID(), other.ID())
				case "host-phase":
					g.PhaseOut(host.ID())
				case "host-return":
					g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
					g.PutOnBattlefield(host.Card, b.PlayerID())
				}
				g.ResolveStack()
				g.PhaseIn([]uuid.UUID{source.ID(), host.ID()})
				p := g.FindPermanent(source.ID())
				if subject == "HostPermanent" {
					p = g.FindPermanent(host.ID())
					if change == "reattach" {
						p = g.FindPermanent(other.ID())
					}
				}
				want := change == "none" || change == "reattach"
				if subject == "HostPermanent" {
					want = change != "host-phase" && change != "host-return"
				}
				if p == nil || p.Tapped != want {
					t.Fatal("source or live host identity", want)
				}
			})
		}
	}
}

func TestRONReferenceCollectionInstancesAndCopy(t *testing.T) {
	a := &ronModePlayer{BasePlayer: mage.NewBasePlayer("A")}
	b := &ronModePlayer{BasePlayer: mage.NewBasePlayer("B")}
	g := mage.NewGame(a, b)
	g.SetTurn(2)
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([TapEachPermanent(SinglePermanent(Ref_TargetPermanent))])))`)
	left, right := factory(), factory()
	if left.Abilities()[0] == right.Abilities()[0] {
		t.Fatal("shared ability")
	}
	firstSource := g.PutOnBattlefield(left, a.PlayerID())
	secondSource := g.PutOnBattlefield(right, b.PlayerID())
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 4), a.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 2, 4), b.PlayerID())
	third := g.PutOnBattlefield(mage.NewCreature("Third", "", 2, 4), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), firstSource.ID(), 0, []uuid.UUID{first.ID()}); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	b.targets = []uuid.UUID{second.ID()}
	if g.CopyStackObjectDirect(original, b.PlayerID(), true) == nil {
		t.Fatal("copy failed")
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), secondSource.ID(), 0, []uuid.UUID{third.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveTopOfStack()
	if !g.FindPermanent(third.ID()).Tapped || g.FindPermanent(first.ID()).Tapped || g.FindPermanent(second.ID()).Tapped {
		t.Fatal("instance bindings merged")
	}
	g.ResolveTopOfStack()
	if !g.FindPermanent(second.ID()).Tapped || g.FindPermanent(first.ID()).Tapped || original.Targets[0] != first.ID() {
		t.Fatal("copy changed original")
	}
	g.ResolveTopOfStack()
	if !g.FindPermanent(first.ID()).Tapped {
		t.Fatal("original binding lost")
	}
}
