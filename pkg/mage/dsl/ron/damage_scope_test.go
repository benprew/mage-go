package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONDamageScopeDeclarations(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([AnyTarget],ActionList([SpellDealsDamage(ThisSpell,Integer(3),Ref_AnyTarget)])))`)
	first, second := factory(), factory()
	left, right := first.Abilities()[0].(*mage.ActionDefinition), second.Abilities()[0].(*mage.ActionDefinition)
	if left.Targets()[0] == right.Targets()[0] {
		t.Fatal("shared target declaration")
	}
	reads := 0
	var walk func(mage.DefinitionMetadata)
	walk = func(n mage.DefinitionMetadata) {
		if n.Role == "damage-recipient-ref" && n.Text == "Ref_AnyTarget" {
			reads++
		}
		for _, child := range n.Children {
			walk(child)
		}
	}
	walk(mage.InspectDefinition(first.Abilities()[0]))
	if reads != 1 {
		t.Fatalf("typed recipient reads=%d want=1", reads)
	}
	for _, c := range []mage.Card{first, second} {
		if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
			t.Fatal(err)
		}
	}
	missing := mage.NewInstant("Missing", "", mage.WithAbility(mage.NewAction(mage.ActionSpell, mage.WithEffects(left.Effects()...))))
	if mage.ValidateCardDefinition(missing.Name(), missing) == nil {
		t.Fatal("missing recipient declaration accepted")
	}
}

func TestRONDamageScopeRealCards(t *testing.T) {
	for _, tc := range []struct {
		set, name    string
		amount, mana int
		spell        bool
	}{
		{"LEA", "Lightning Bolt", 3, 1, true}, {"LEB", "Lightning Bolt", 3, 1, true},
		{"2ED", "Lightning Bolt", 3, 1, true}, {"3ED", "Lightning Bolt", 3, 1, true},
		{"CED", "Lightning Bolt", 3, 1, true}, {"CEI", "Lightning Bolt", 3, 1, true},
		{"LEA", "Prodigal Sorcerer", 1, 0, false}, {"LEA", "Rod of Ruin", 1, 3, false},
		{"ARN", "Aladdin's Ring", 4, 8, false}, {"FEM", "Aeolipile", 2, 1, false},
	} {
		for _, kind := range []string{"player", "creature", "planeswalker"} {
			t.Run(tc.set+"/"+tc.name+"/"+kind, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, tc.set, tc.name)
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				target := b.PlayerID()
				if kind == "creature" {
					target = g.PutOnBattlefield(repositoryCard(t, "LEA", "Wall of Stone"), b.PlayerID()).ID()
				}
				if kind == "planeswalker" {
					target = g.PutOnBattlefield(mage.NewPlaneswalker("Fixture", "", 9), b.PlayerID()).ID()
				}
				a.ManaPool().Add(core.Red, tc.mana)
				if tc.spell {
					a.AddToHand(card)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, 0); err != nil {
						t.Fatal(err)
					}
				} else {
					source := g.PutOnBattlefield(card, a.PlayerID())
					source.RevokeBaseAttr(core.AttrSummonSick)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target}); err != nil {
						t.Fatal(err)
					}
				}
				clone := g.Clone()
				g.ResolveStack()
				if a.Life() != 20 {
					t.Fatal("recipient became resolving controller")
				}
				switch kind {
				case "player":
					if b.Life() != 20-tc.amount || clone.GetPlayer(b.PlayerID()).Life() != 20 {
						t.Fatal("player binding or clone isolation")
					}
				case "creature":
					if g.FindPermanent(target).Damage != tc.amount || clone.FindPermanent(target).Damage != 0 {
						t.Fatal("creature binding or clone isolation")
					}
				case "planeswalker":
					if g.FindPermanent(target).Counters[core.Loyalty] != 9-tc.amount || clone.FindPermanent(target).Counters[core.Loyalty] != 9 {
						t.Fatal("planeswalker binding or clone isolation")
					}
				}
				clone.ResolveStack()
				if clone.GetPlayer(b.PlayerID()).Life() != b.Life() {
					t.Fatal("clone lost target")
				}
			})
		}
	}
}

func TestRONDamageScopeIdentity(t *testing.T) {
	for _, scenario := range []string{"control", "return", "phase", "phase-cycle", "shroud"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			victimCard := repositoryCard(t, "LEA", "Wall of Stone")
			victimCard.SetOwner(b.PlayerID())
			victim := g.PutOnBattlefield(victimCard, b.PlayerID())
			current := victim.ID()
			bolt := repositoryCard(t, "LEA", "Lightning Bolt")
			bolt.SetOwner(a.PlayerID())
			a.AddToHand(bolt)
			a.ManaPool().Add(core.Red, 1)
			if err := g.CastSpellByID(a.PlayerID(), bolt.ID(), []uuid.UUID{current}, 0); err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: bolt.ID(), TargetID: current, ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "return":
				g.BouncePermanentToHand(g.FindPermanent(current))
				b.RemoveFromHand(victimCard.ID())
				current = g.PutOnBattlefield(victimCard, b.PlayerID()).ID()
			case "phase", "phase-cycle":
				g.PhaseOut(current)
			case "shroud":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.GrantKeywordModification(mage.This, core.Shroud)), current, b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "phase-cycle" {
				g.PhaseIn([]uuid.UUID{current})
			}
			g.ResolveStack()
			want := 0
			if scenario == "control" || scenario == "phase-cycle" {
				want = 3
			}
			if g.FindPermanentIncludingPhased(current).Damage != want {
				t.Fatal("target identity or resolution legality")
			}
		})
	}
}

func TestRONDamageScopeModesCopies(t *testing.T) {
	for mode := range 2 {
		g, a, b := ronModalGame(mode)
		factory := permanentScopeFactory(t, "Instant", `SpellActions(Modal_ChooseOne([Targeted([AnyTarget],ActionList([SpellDealsDamage(ThisSpell,Integer(2),Ref_AnyTarget)])),Targeted([AnyTarget],ActionList([SpellDealsDamage(ThisSpell,Integer(3),Ref_AnyTarget)]))]))`)
		card := factory()
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
			t.Fatal(err)
		}
		a.targets = []uuid.UUID{a.PlayerID()}
		if cp := g.CopySpellOnStack(card.ID(), a.PlayerID(), true); cp == nil || cp.ModeChoice != mode || cp.Targets[0] != a.PlayerID() {
			t.Fatal("mode or retargeted copy")
		}
		g.ResolveStack()
		if a.Life() != 18-mode || b.Life() != 18-mode || a.prompts != 1 {
			t.Fatal("mode-local recipient")
		}
	}
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([AnyTarget],ActionList([SpellDealsDamage(ThisSpell,Integer(2),Ref_AnyTarget)])))`)
	g, a, b := ronGame()
	for _, target := range []uuid.UUID{a.PlayerID(), b.PlayerID()} {
		c := factory()
		c.SetOwner(a.PlayerID())
		a.AddToHand(c)
		if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{target}, 0); err != nil {
			t.Fatal(err)
		}
	}
	g.ResolveStack()
	if a.Life() != 18 || b.Life() != 18 {
		t.Fatal("instance recipient overwritten")
	}
}

func TestRONDamageScopeAnnouncedX(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`Card(Name:"X Damage",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),ManaCost:[ManaCostX,ManaCostR],Rules:[SpellActions(Targeted([AnyTarget],ActionList([SpellDealsDamage(ThisSpell,ValueX,Ref_AnyTarget)])))])`), "x.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	card := factory()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Red, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 4); err == nil {
		t.Fatal("unpaid X accepted")
	}
	if len(a.Hand()) != 1 || a.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
		t.Fatal("unpaid X changed state")
	}
	a.ManaPool().Add(core.Colorless, 4)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 4); err != nil {
		t.Fatal(err)
	}
	if g.CopySpellOnStack(card.ID(), b.PlayerID(), true) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if a.Life() != 16 || b.Life() != 16 || a.ManaPool().TotalMana() != 0 {
		t.Fatal("announced X or copied recipient was lost")
	}
}

func TestRONDamageScopeReplacementOrder(t *testing.T) {
	for mode := range 2 {
		a := mage.NewBasePlayer("A")
		b := &ronModePlayer{BasePlayer: mage.NewBasePlayer("B"), mode: mode}
		g := mage.NewGame(a, b)
		g.SetStep(core.PrecombatMain)
		g.SetTurn(1)
		factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([AnyTarget],ActionList([SpellDealsDamage(ThisSpell,Integer(4),Ref_AnyTarget)])))`)
		card := factory()
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
			t.Fatal(err)
		}
		g.AddHalfDamageFromSourcePreventionShield(b.PlayerID(), card.ID())
		g.AddPreventionShield(b.PlayerID(), 1)
		g.ResolveStack()
		if b.Life() != 19-mode || b.prompts != 1 {
			t.Fatalf("recipient replacement choice: life=%d prompts=%d", b.Life(), b.prompts)
		}
	}
}

func TestRONDamageScopeTrigger(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),Targeted([AnyTarget],ActionList([PermanentDealsDamage(ThisPermanent,Integer(1),Ref_AnyTarget)])))`)
	g, a, b := ronGame()
	card := factory()
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	card.SetOwner(b.PlayerID())
	source := g.PutOnBattlefield(card, b.PlayerID())
	g.PutTriggersOnStack()
	if a.Life() != 20 || len(g.StackPeek().Targets) != 1 || g.StackPeek().Targets[0] != a.PlayerID() {
		t.Fatal("trigger announcement")
	}
	g.PhaseOut(source.ID())
	g.ResolveStack()
	if a.Life() != 19 || b.Life() != 20 {
		t.Fatal("trigger lost target or phased source LKI")
	}
}

func TestRONDamageScopeRollbackAndSourceLKI(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "FEM", "Aeolipile")
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err == nil {
		t.Fatal("unpaid activation")
	}
	a.ManaPool().Add(core.Colorless, 1)
	land := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{land.ID()}); err == nil {
		t.Fatal("land accepted as damage target")
	}
	if g.StackSize() != 0 || g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != 1 {
		t.Fatal("rollback")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(source.ID()) != nil || b.Life() != 20 {
		t.Fatal("sacrifice or damage timing")
	}
	g.PutOnBattlefield(card, b.PlayerID())
	clone := g.Clone()
	g.ResolveStack()
	clone.ResolveStack()
	if b.Life() != 18 || clone.GetPlayer(b.PlayerID()).Life() != 18 {
		t.Fatal("damage lost original source LKI")
	}
}

func TestRONDamageScopeInvalid(t *testing.T) {
	for _, rule := range []string{
		`SpellActions(ActionList([SpellDealsDamage(ThisSpell,Integer(1),Ref_AnyTarget)]))`,
		`SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([SpellDealsDamage(ThisSpell,Integer(1),Ref_AnyTarget)])))`,
		`SpellActions(Targeted([AnyTarget],ActionList([SpellDealsDamage(ThisSpell,Integer(1),Ref_AnyTarget())])))`,
		`SpellActions(Modal_ChooseOne([Targeted([AnyTarget],ActionList([SpellDealsDamage(ThisSpell,Integer(1),Ref_AnyTarget)])),ActionList([SpellDealsDamage(ThisSpell,Integer(1),Ref_AnyTarget)])]))`,
		`SpellActions(Targeted([AnyTarget()],ActionList([])))`,
		`SpellActions(Targeted([AnyTarget],ActionList([CreateFuturePreventDamage(NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(Integer(1),Ref_AnyTarget()),[PreventThatDamage])])))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}
