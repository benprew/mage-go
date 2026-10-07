package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONAbilityTargetCards(t *testing.T) {
	for _, name := range []string{"Rust", "Ayesha Tanaka"} {
		for _, scenario := range []string{"counter", "pay", "decline", "departed", "returned", "phase", "clone", "source-control", "gone", "wrong-color", "mana-source"} {
			t.Run(name+"/"+scenario, func(t *testing.T) {
				a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
				b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: scenario != "decline"}
				g := mage.NewGame(a, b)
				c := repositoryCard(t, "LEG", name)
				c.SetOwner(a.PlayerID())
				if err := mage.ValidateCardDefinition(name, c); err != nil {
					t.Fatal(err)
				}
				artifact := repositoryCard(t, "LEB", "Disrupting Scepter")
				artifact.SetOwner(b.PlayerID())
				source := g.PutOnBattlefield(artifact, b.PlayerID())
				g.SetActivePlayerIndex(1)
				g.SetStep(core.PrecombatMain)
				b.ManaPool().Add(core.Colorless, 3)
				if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, []uuid.UUID{a.PlayerID()}); err != nil {
					t.Fatal(err)
				}
				target := g.StackPeek().ID
				a.AddToHand(mage.NewLand("Keep"))
				if scenario == "pay" || scenario == "decline" || scenario == "source-control" {
					b.ManaPool().Add(core.White, 1)
				}
				if scenario == "wrong-color" {
					b.ManaPool().Add(core.Red, 1)
				}
				if scenario == "mana-source" {
					g.PutOnBattlefield(repositoryCard(t, "LEB", "Plains"), b.PlayerID())
				}
				if name == "Rust" {
					a.AddToHand(c)
					if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{target}, 0); err == nil {
						t.Fatal("unpaid cast")
					}
					a.ManaPool().Add(core.Green, 1)
					if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{target}, 0); err != nil {
						t.Fatal(err)
					}
				} else {
					p := g.PutOnBattlefield(c, a.PlayerID())
					g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{source.ID()}); err == nil {
						t.Fatal("permanent accepted")
					}
					if g.FindPermanent(p.ID()).Tapped {
						t.Fatal("invalid target spent tap")
					}
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{target}); err != nil {
						t.Fatal(err)
					}
					if !g.FindPermanent(p.ID()).Tapped || a.offers != 0 || b.offers != 0 {
						t.Fatal("announcement timing")
					}
				}
				clone := g.Clone()
				switch scenario {
				case "departed", "returned":
					g.BouncePermanentToHand(g.FindPermanent(source.ID()))
					if scenario == "returned" {
						b.RemoveFromHand(artifact.ID())
						g.PutOnBattlefield(artifact, b.PlayerID())
					}
				case "phase":
					g.PhaseOut(source.ID())
				case "source-control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: source.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				case "gone":
					g.CounterStackObject(target)
				case "clone":
					clone.ResolveStack()
				}
				g.ResolveStack()
				want := 1
				if name == "Ayesha Tanaka" && (scenario == "pay" || scenario == "source-control" || scenario == "mana-source") {
					want = 0
				}
				if len(a.Hand()) != want {
					t.Fatalf("hand=%d want %d", len(a.Hand()), want)
				}
				if a.offers != 0 {
					t.Fatal("wrong payer")
				}
				if scenario == "gone" && b.offers != 0 {
					t.Fatal("illegal target offered payment")
				}
				if name == "Ayesha Tanaka" && (scenario == "pay" || scenario == "source-control" || scenario == "mana-source") && (b.offers != 1 || b.ManaPool().TotalMana() != 0) {
					t.Fatal("white payment")
				}
				if name == "Ayesha Tanaka" && scenario == "decline" && (b.offers != 1 || b.ManaPool().TotalMana() != 1) {
					t.Fatal("declined payment spent mana")
				}
				if scenario != "clone" {
					clone.ResolveStack()
				}
				if scenario != "pay" && scenario != "decline" && scenario != "source-control" && scenario != "mana-source" && len(clone.GetPlayer(a.PlayerID()).Hand()) != 1 {
					t.Fatal("clone lost ability target")
				}
			})
		}
	}
}

func TestRONAbilityTargetInvalid(t *testing.T) {
	for _, rule := range []string{
		`SpellActions(ActionList([CounterAbility(Ref_TargetAbility)]))`,
		`SpellActions(Targeted([TargetSpell(AnySpell)],ActionList([CounterAbility(Ref_TargetSpell)])))`,
		`SpellActions(Targeted([TargetAbility(And([]))],ActionList([CounterAbility(Ref_TargetAbility)])))`,
		`SpellActions(Targeted([TargetAbility(And([ActivatedAbility(),AbilityOfASource(IsCardtype(Artifact))]))],ActionList([CounterAbility(Ref_TargetAbility)])))`,
		`SpellActions(Targeted([TargetAbility(And([ActivatedAbility,AbilityOfASource(IsCardtype(Artifact)),ActivatedAbility]))],ActionList([CounterAbility(Ref_TargetAbility)])))`,
		`SpellActions(Targeted([TargetAbility(And([ActivatedAbility,AbilityOfASource(IsCardtype(Artifact))]))],ActionList([CounterAbility(Ref_TargetAbility())])))`,
		`Activated(PayMana([]),Targeted([TargetAbility(And([ActivatedAbility,AbilityOfASource(IsCardtype(Artifact))]))],ActionList([]))),Activated(PayMana([]),ActionList([CounterAbility(Ref_TargetAbility)]))`,
	} {
		assertConditionalRejected(t, rule)
	}
}

func TestRONAbilityTargetAyeshaCasting(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEG", "Ayesha Tanaka")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.White, 2)
	a.ManaPool().Add(core.Blue, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("partial casting payment accepted")
	}
	if len(a.Hand()) != 1 || a.ManaPool().TotalMana() != 3 || g.StackSize() != 0 {
		t.Fatal("failed cast changed state")
	}
	a.ManaPool().Add(core.Blue, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(card.ID()) != nil || g.StackSize() != 1 {
		t.Fatal("creature entered before resolution")
	}
	g.ResolveStack()
	p := g.FindPermanent(card.ID())
	if p == nil || p.CurrentPower(g) != 2 || p.CurrentToughness(g) != 2 || !p.HasKeyword(core.Banding) || !p.Card.HasSuperType(core.SuperLegendary) {
		t.Fatal("Ayesha characteristics")
	}
	artifact := g.PutOnBattlefield(repositoryCard(t, "LEB", "Disrupting Scepter"), b.PlayerID())
	g.SetActivePlayerIndex(1)
	g.SetStep(core.PrecombatMain)
	b.ManaPool().Add(core.Colorless, 3)
	if err := g.ActivateAbilityByIndex(b.PlayerID(), artifact.ID(), 0, []uuid.UUID{a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{g.StackPeek().ID}); err == nil || g.FindPermanent(p.ID()).Tapped {
		t.Fatal("summoning sickness or failed tap payment")
	}
}

func TestRONAbilityTargetDeclarations(t *testing.T) {
	for _, name := range []string{"Rust", "Ayesha Tanaka"} {
		first := repositoryCard(t, "LEG", name)
		second := repositoryCard(t, "LEG", name)
		index := 0
		x := first.Abilities()[index].(*mage.ActionDefinition)
		y := second.Abilities()[index].(*mage.ActionDefinition)
		if x.Targets()[0] == y.Targets()[0] {
			t.Fatal("shared factory target")
		}
		detached := mage.NewInstant("Detached", "", mage.WithAbility(mage.NewAction(mage.ActionSpell, mage.WithEffects(x.Effects()...))))
		if mage.ValidateCardDefinition("Detached", detached) == nil {
			t.Fatal("detached ability read accepted")
		}
	}
}

func TestRONAbilityTargetRustCopy(t *testing.T) {
	g, a, b := ronGame()
	artifact := g.PutOnBattlefield(mage.NewArtifact("Source", "", mage.WithAbility(mage.Activate(mage.Costs(), mage.GainLifeFor(mage.You, mage.Fixed(3))))), b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), artifact.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	abilityCopy := g.CopyStackObjectDirect(original, a.PlayerID(), false)
	rust := repositoryCard(t, "LEG", "Rust")
	rust.SetOwner(a.PlayerID())
	a.AddToHand(rust)
	a.ManaPool().Add(core.Green, 1)
	if err := g.CastSpellByID(a.PlayerID(), rust.ID(), []uuid.UUID{abilityCopy.ID}, 0); err != nil {
		t.Fatal(err)
	}
	rustCopy := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
	if rustCopy == nil {
		t.Fatal("Rust copy failed")
	}
	g.ResolveTopOfStack()
	if g.GetStack().FindByID(abilityCopy.ID) != nil || g.GetStack().FindByID(original.ID) == nil {
		t.Fatal("counter used source identity")
	}
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 23 || len(a.Graveyard()) != 1 || g.FindPermanent(artifact.ID()) == nil {
		t.Fatal("Rust resolution and copy lifecycle")
	}
}

func TestRONAbilityTargetBadPlayers(t *testing.T) {
	for _, rule := range []string{
		`SpellActions(ActionList([PlayerMayCost(ControllerOfAbility(Ref_TargetAbility),PayMana([ManaCostW])),Unless(CostWasPaid,[GainLife(Integer(1))])]))`,
		`SpellActions(Targeted([TargetSpell(AnySpell)],ActionList([PlayerMayCost(ControllerOfAbility(Ref_TargetSpell),PayMana([ManaCostW])),Unless(CostWasPaid,[CounterSpell(Ref_TargetSpell)])])))`,
		`SpellActions(Targeted([TargetAbility(And([ActivatedAbility,AbilityOfASource(IsCardtype(Artifact))]))],ActionList([PlayerMayCost(ControllerOfAbility(Ref_TargetAbility()),PayMana([ManaCostW])),Unless(CostWasPaid,[CounterAbility(Ref_TargetAbility)])])))`,
		`SpellActions(Targeted([TargetAbility(And([TriggeredAbility,AbilityOfASource(IsCardtype(Artifact))]))],ActionList([CounterAbility(Ref_TargetAbility)])))`,
		`SpellActions(Targeted([TargetAbility(And([ActivatedAbility,AbilityOfASource(IsColor(Red))]))],ActionList([CounterAbility(Ref_TargetAbility)])))`,
		`SpellActions(Targeted([TargetAbility(And([ActivatedAbility,AbilityOfASource(IsCardtype(Artifact()))]))],ActionList([CounterAbility(Ref_TargetAbility)])))`,
	} {
		assertConditionalRejected(t, rule)
	}
}

func TestRONAbilityTargetAyeshaBand(t *testing.T) {
	a := &ronBandSourcePlayer{BasePlayer: mage.NewBasePlayer("A")}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	source := g.PutOnBattlefield(repositoryCard(t, "LEG", "Ayesha Tanaka"), a.PlayerID())
	member := g.PutOnBattlefield(repositoryCard(t, "LEB", "Grizzly Bears"), a.PlayerID())
	blocker := g.PutOnBattlefield(repositoryCard(t, "LEB", "Hill Giant"), b.PlayerID())
	for _, p := range []*mage.Permanent{source, member} {
		g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
	}
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{source.ID(), member.ID()})
	g.GetCombat().AddBand([]uuid.UUID{source.ID(), member.ID()})
	g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: blocker.ID(), AttackerID: source.ID()}})
	clone := g.Clone()
	g.ExecuteCombatDamage()
	if g.FindPermanent(source.ID()) != nil || g.FindPermanent(member.ID()) == nil || g.FindPermanent(blocker.ID()) != nil || b.Life() != 20 {
		t.Fatal("banding did not control incoming damage")
	}
	if clone.FindPermanent(source.ID()) == nil || clone.FindPermanent(member.ID()) == nil {
		t.Fatal("combat changed clone")
	}
}

func TestRONAbilityTargetFilterOrder(t *testing.T) {
	for _, filter := range []string{
		`And([ActivatedAbility,AbilityOfASource(IsCardtype(Artifact))])`,
		`And([AbilityOfASource(IsCardtype(Artifact)),ActivatedAbility])`,
	} {
		factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetAbility(`+filter+`)],ActionList([CounterAbility(Ref_TargetAbility)])))`)
		c := factory()
		if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
			t.Fatal(err)
		}
		reads := 0
		var walk func(mage.DefinitionMetadata)
		walk = func(n mage.DefinitionMetadata) {
			if n.Role == "ability-ref" && n.Text == "Ref_TargetAbility" {
				reads++
			}
			for _, child := range n.Children {
				walk(child)
			}
		}
		walk(mage.InspectDefinition(c.Abilities()[0]))
		if reads != 1 {
			t.Fatal("missing typed ability read")
		}
	}
}

func TestRONAbilityTargetAyeshaSourceLifetime(t *testing.T) {
	for _, scenario := range []string{"leaves", "returns", "phased", "copy"} {
		t.Run(scenario, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: false}
			g := mage.NewGame(a, b)
			victim := g.PutOnBattlefield(mage.NewArtifact("Source", "", mage.WithAbility(mage.Activate(mage.Costs(), mage.GainLifeFor(mage.You, mage.Fixed(3))))), b.PlayerID())
			if err := g.ActivateAbilityByIndex(b.PlayerID(), victim.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			card := repositoryCard(t, "LEG", "Ayesha Tanaka")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrSummonSick)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{g.StackPeek().ID}); err != nil {
				t.Fatal(err)
			}
			b.ManaPool().Add(core.White, 1)
			switch scenario {
			case "leaves", "returns":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				if scenario == "returns" {
					g.PutOnBattlefield(card, a.PlayerID())
				}
			case "phased":
				g.PhaseOut(source.ID())
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			g.ResolveStack()
			if b.Life() != 20 || b.offers != 1 || a.offers != 0 || b.ManaPool().TotalMana() != 1 {
				t.Fatal("source lifetime or copied payer")
			}
		})
	}
}
