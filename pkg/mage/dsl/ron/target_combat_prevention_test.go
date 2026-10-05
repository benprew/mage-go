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

const targetCombatPrevention = `CreatePreventDamageUntil(Or([CombatDamageWouldBeDealtToRecipient(Permanent(Ref_TargetPermanent)), CombatDamageWouldBeDealtByCreature(Ref_TargetPermanent)]), [PreventThatDamage], UntilEndOfTurn)`

func TestRONTargetCombatPreventionCards(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		mana      int
		color     core.Color
	}{
		{"DRK", "Maze of Ith", 0, core.Colorless}, {"ARN", "Ebony Horse", 2, core.Colorless}, {"3ED", "Ebony Horse", 2, core.Colorless}, {"FEM", "Elvish Scout", 1, core.Green},
	} {
		for _, change := range []string{"normal", "unpaid", "sick", "not-attacking", "opponent", "player", "land", "shroud", "return-before", "phase-before", "phase-cycle-before", "removed-before", "source-return", "source-phase", "source-control", "control-before", "control-after", "phase-cycle-after", "return-after", "stun", "copy"} {
			t.Run(tc.set+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, tc.set, tc.name)
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				if change != "sick" {
					source.RevokeBaseAttr(core.AttrSummonSick)
				}
				targetCard := mage.NewCreature("Target", "", 4, 20, mage.WithKeyword(core.DoubleStrike), mage.WithKeyword(core.Trample), mage.WithKeyword(core.Lifelink))
				targetCard.SetOwner(b.PlayerID())
				controller := a.PlayerID()
				if change == "opponent" {
					controller = b.PlayerID()
				}
				defender := b.PlayerID()
				if controller == b.PlayerID() {
					defender = a.PlayerID()
				}
				target := g.PutOnBattlefield(targetCard, controller)
				g.TapPermanent(target)
				if change != "not-attacking" {
					g.GetCombat().AddAttacker(target.ID(), defender)
				}
				id := target.ID()
				if change == "player" {
					id = b.PlayerID()
				}
				if change == "land" {
					id = g.PutOnBattlefield(mage.NewLand("Plain Land"), a.PlayerID()).ID()
				}
				if change != "unpaid" {
					a.ManaPool().Add(tc.color, tc.mana)
				}
				mana := a.ManaPool().TotalMana()
				err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{id})
				reject := change == "not-attacking" || change == "player" || change == "land" || (change == "unpaid" && tc.mana > 0) || (change == "sick" && tc.name == "Elvish Scout") || (change == "opponent" && tc.name != "Maze of Ith")
				if reject {
					if err == nil || g.StackSize() != 0 || g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != mana || !g.FindPermanent(target.ID()).Tapped {
						t.Fatal("illegal activation changed state")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if !g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != 0 || !g.FindPermanent(target.ID()).Tapped {
					t.Fatal("cost or resolution timing")
				}
				active := true
				switch change {
				case "shroud":
					g.FindPermanent(id).GrantBaseAttr(core.Shroud)
					active = false
				case "return-before":
					g.RemoveFromBattlefield(target)
					g.PutOnBattlefield(targetCard, controller)
					active = false
				case "phase-before":
					g.PhaseOut(id)
					active = false
				case "phase-cycle-before":
					g.PhaseIn(g.PhaseOut(id))
					active = false
				case "removed-before":
					g.RemoveFromCombat(id)
					active = false
				case "source-return":
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
					g.PutOnBattlefield(card, a.PlayerID())
				case "source-phase":
					g.PhaseOut(source.ID())
				case "control-before":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: id, ControllerID: b.PlayerID(), Duration: core.Indefinite})
					active = false
				case "source-control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "stun":
					g.FindPermanent(id).Counters[core.Stun] = 1
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				clone := g.Clone()
				pendingTapped := clone.FindPermanentIncludingPhased(id).Tapped
				g.ResolveStack()
				if clone.FindPermanentIncludingPhased(id).Tapped != pendingTapped {
					t.Fatal("resolution changed pending clone")
				}
				clone.ResolveStack()
				if change == "phase-before" {
					g.PhaseIn([]uuid.UUID{id})
					clone.PhaseIn([]uuid.UUID{id})
				}
				if change == "stun" && (!g.FindPermanent(id).Tapped || g.FindPermanent(id).Counters[core.Stun] != 0) {
					t.Fatal("stun did not replace only the untap")
				}
				if active && change != "stun" && g.FindPermanent(id).Tapped {
					t.Fatal("target was not untapped")
				}
				if !active && (change == "shroud" || change == "removed-before") && !g.FindPermanent(id).Tapped {
					t.Fatal("illegal target was untapped")
				}
				if change == "normal" || change == "stun" {
					if !g.GetCombat().IsAttacking(id) {
						t.Fatal("untap removed attacker")
					}
					blocker := g.PutOnBattlefield(mage.NewCreature("Blocker", "", 3, 20), b.PlayerID())
					other := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 20), a.PlayerID())
					g.GetCombat().AddAttacker(other.ID(), b.PlayerID())
					g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: blocker.ID(), AttackerID: id}})
					g.ExecuteCombatDamage()
					if a.Life() != 20 || b.Life() != 19 || g.FindPermanent(id).Damage != 0 || g.FindPermanent(blocker.ID()).Damage != 0 {
						t.Fatal("combat damage to or by target escaped prevention")
					}
					g.GetEffects().RemoveEndOfCombat()
					g.GetCombat().Reset()
					g.GetCombat().AddAttacker(id, b.PlayerID())
					g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: blocker.ID(), AttackerID: id}})
					g.ExecuteCombatDamage()
					if a.Life() != 20 || b.Life() != 19 || g.FindPermanent(id).Damage != 0 || g.FindPermanent(blocker.ID()).Damage != 0 {
						t.Fatal("later combat lost prevention")
					}
					g.DealDamageToPermanent(g.FindPermanent(id), 1, blocker.ID())
					g.DealDamageToPlayer(b, 1, id)
					if g.FindPermanent(id).Damage != 1 || b.Life() != 18 || a.Life() != 21 {
						t.Fatal("noncombat damage or lifelink prevented")
					}
				}
				switch change {
				case "control-after":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: id, ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "phase-cycle-after":
					g.PhaseIn(g.PhaseOut(id))
				case "return-after":
					g.RemoveFromBattlefield(g.FindPermanent(id))
					g.PutOnBattlefield(targetCard, controller)
					active = false
				}
				if sourceCombatPrevented(g, id, defender) != active {
					t.Fatal("wrong target identity or legality")
				}
				g.RunStepWithPriority(core.Cleanup)
				if sourceCombatPrevented(g, id, defender) {
					t.Fatal("prevention survived cleanup")
				}
				if sourceCombatPrevented(clone, id, defender) != (active || change == "return-after") {
					t.Fatal("cleanup or target change affected clone")
				}
			})
		}
	}
}

func TestRONTargetCombatPreventionDeclarations(t *testing.T) {
	for _, action := range []string{
		strings.Replace(targetCombatPrevention, "Or([", "And([", 1),
		strings.Replace(targetCombatPrevention, "CombatDamageWouldBeDealtByCreature(Ref_TargetPermanent)", "CombatDamageWouldBeDealtByCreature(ThisPermanent)", 1),
		strings.Replace(targetCombatPrevention, "Permanent(Ref_TargetPermanent)", "Player(You)", 1),
		strings.Replace(targetCombatPrevention, "Permanent(Ref_TargetPermanent)", "Ref_TargetPermanent", 1),
		strings.ReplaceAll(targetCombatPrevention, "Ref_TargetPermanent", "Ref_TargetPlayer"),
		strings.ReplaceAll(targetCombatPrevention, "Ref_TargetPermanent", "Ref_TargetPermanent_2"),
		strings.Replace(targetCombatPrevention, "Permanent(Ref_TargetPermanent)", "Permanent(Ref_TargetPermanent, ThisPermanent)", 1),
		strings.Replace(targetCombatPrevention, "CombatDamageWouldBeDealtToRecipient(Permanent(Ref_TargetPermanent)), ", "", 1),
		strings.Replace(targetCombatPrevention, "CombatDamageWouldBeDealtByCreature(Ref_TargetPermanent)", "CombatDamageWouldBeDealtToRecipient(Permanent(Ref_TargetPermanent))", 1),
		strings.Replace(targetCombatPrevention, "[PreventThatDamage]", "[]", 1),
		strings.Replace(targetCombatPrevention, "PreventThatDamage]", "PreventThatDamage() ]", 1),
		strings.Replace(targetCombatPrevention, "UntilEndOfTurn", "UntilEndOfCombat", 1),
	} {
		src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(TapPermanent(ThisPermanent),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([%s])))])`, action)
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
	for _, rules := range []string{
		`Activated(TapPermanent(ThisPermanent),ActionList([` + targetCombatPrevention + `]))`,
		`Activated(TapPermanent(ThisPermanent),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([]))),Activated(TapPermanent(ThisPermanent),ActionList([` + targetCombatPrevention + `]))`,
		`PermanentLayerEffect(ThisPermanent,[` + targetCombatPrevention + `])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rules+`])`), "scope.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted scope %s", rules)
		}
	}
	src := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([` + targetCombatPrevention + `])))])`
	report, err := InspectFS(fstest.MapFS{"test.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, sig := range report.Signatures {
		if sig.Name == "Ref_TargetPermanent" && sig.Role != "permanent_reference" {
			t.Fatalf("role=%s", sig.Role)
		}
	}
}

func TestRONTargetCombatPreventionInstancesAndContexts(t *testing.T) {
	for _, rule := range []string{
		`SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([%s])))`,
		`SpellActions(Modal_ChooseOne([Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([%s])),ActionList([GainLife(Integer(1))])]))`,
		`TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([%s])))`,
	} {
		t.Run(rule, func(t *testing.T) {
			typ := "Instant"
			if strings.HasPrefix(rule, "TriggerA") {
				typ = "Artifact"
			}
			src := fmt.Sprintf(`Card(Name:"Probe",Typeline:(Supertypes:[],Cardtypes:[%s],Subtypes:[]),Rules:[%s])`, typ, fmt.Sprintf(rule, targetCombatPrevention))
			nodes, err := Parse(strings.NewReader(src), "probe.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			first := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 10), b.PlayerID())
			second := g.PutOnBattlefield(mage.NewCreature("Second", "", 3, 10), b.PlayerID())
			for _, id := range []uuid.UUID{first.ID(), second.ID()} {
				card := factory()
				card.SetOwner(a.PlayerID())
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				if typ == "Artifact" {
					g.PhaseOut(first.ID())
					if id == first.ID() {
						g.PhaseIn([]uuid.UUID{first.ID()})
						g.PhaseOut(second.ID())
					}
					g.PutOnBattlefield(card, a.PlayerID())
					g.PutTriggersOnStack()
					g.PhaseIn([]uuid.UUID{first.ID(), second.ID()})
				} else {
					a.AddToHand(card)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{id}, 0); err != nil {
						t.Fatal(err)
					}
				}
			}
			clone := g.Clone()
			g.ResolveTopOfStack()
			if sourceCombatPrevented(g, first.ID(), a.PlayerID()) || !sourceCombatPrevented(g, second.ID(), a.PlayerID()) {
				t.Fatal("factory instances shared binding")
			}
			if sourceCombatPrevented(clone, second.ID(), a.PlayerID()) {
				t.Fatal("resolution affected pending clone")
			}
			g.ResolveStack()
			clone.ResolveStack()
			if !sourceCombatPrevented(g, first.ID(), a.PlayerID()) || !sourceCombatPrevented(clone, second.ID(), a.PlayerID()) {
				t.Fatal("pending binding lost identity")
			}
		})
	}
}

func TestRONTargetCombatPreventionReplacementOrder(t *testing.T) {
	for mode := range 2 {
		a := &ronModePlayer{BasePlayer: mage.NewBasePlayer("A"), mode: mode}
		b := mage.NewBasePlayer("B")
		g := mage.NewGame(a, b)
		g.SetTurn(1)
		g.SetStep(core.PrecombatMain)
		maze := g.PutOnBattlefield(repositoryCard(t, "DRK", "Maze of Ith"), a.PlayerID())
		attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 3, 12), b.PlayerID())
		victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 2, 12), a.PlayerID())
		g.GetCombat().AddAttacker(attacker.ID(), a.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), maze.ID(), 0, []uuid.UUID{attacker.ID()}); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		g.SetCreatureDamageRedirect(victim.ID(), a.PlayerID())
		g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: victim.ID(), AttackerID: attacker.ID()}})
		g.ExecuteCombatDamage()
		if a.Life() != 20 || g.FindPermanent(victim.ID()).Damage != 0 || g.FindPermanent(attacker.ID()).Damage != 0 || a.prompts != 1 {
			t.Fatalf("life=%d damage=%d/%d prompts=%d", a.Life(), g.FindPermanent(victim.ID()).Damage, g.FindPermanent(attacker.ID()).Damage, a.prompts)
		}
		g.FindPermanent(victim.ID()).GrantBaseAttr(core.AttrDamageCantBePreventedOrRedirected)
		g.GetCombat().Reset()
		g.GetCombat().AddAttacker(attacker.ID(), a.PlayerID())
		g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: victim.ID(), AttackerID: attacker.ID()}})
		g.ExecuteCombatDamage()
		if g.FindPermanent(victim.ID()).Damage != 3 {
			t.Fatal("unpreventable damage was prevented")
		}
	}
}

func TestRONTargetCombatPreventionCopyRetarget(t *testing.T) {
	g, a, b := ronGame()
	maze := g.PutOnBattlefield(repositoryCard(t, "DRK", "Maze of Ith"), a.PlayerID())
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 10), b.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 3, 10), b.PlayerID())
	g.GetCombat().AddAttacker(first.ID(), a.PlayerID())
	g.GetCombat().AddAttacker(second.ID(), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), maze.ID(), 0, []uuid.UUID{first.ID()}); err != nil {
		t.Fatal(err)
	}
	phased := g.PhaseOut(first.ID())
	cp := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), true)
	if cp == nil || len(cp.Targets) != 1 || cp.Targets[0] != second.ID() {
		t.Fatal("copy did not choose distinct legal target")
	}
	g.PhaseIn(phased)
	g.GetCombat().AddAttacker(first.ID(), a.PlayerID())
	clone := g.Clone()
	g.ResolveTopOfStack()
	if sourceCombatPrevented(g, first.ID(), a.PlayerID()) || !sourceCombatPrevented(g, second.ID(), a.PlayerID()) {
		t.Fatal("copy changed original binding")
	}
	g.ResolveStack()
	clone.ResolveStack()
	if !sourceCombatPrevented(g, first.ID(), a.PlayerID()) || !sourceCombatPrevented(clone, second.ID(), a.PlayerID()) {
		t.Fatal("copy or clone lost binding")
	}
}

func TestRONTargetCombatPreventionReferenceReuse(t *testing.T) {
	for _, ref := range []string{"ThisPermanent", "HostPermanent"} {
		t.Run(ref, func(t *testing.T) {
			action := strings.ReplaceAll(targetCombatPrevention, "Ref_TargetPermanent", ref)
			action = strings.Replace(action, "CombatDamageWouldBeDealtToRecipient(Permanent("+ref+")), CombatDamageWouldBeDealtByCreature("+ref+")", "CombatDamageWouldBeDealtByCreature("+ref+"), CombatDamageWouldBeDealtToRecipient(Permanent("+ref+"))", 1)
			typ := `[Creature]`
			sub := `[]`
			pt := `,CardPT:(Power:2,Toughness:10)`
			rules := `Activated(TapPermanent(ThisPermanent),ActionList([` + action + `]))`
			if ref == "HostPermanent" {
				typ = `[Enchantment]`
				sub = `[Aura]`
				pt = ""
				rules = `EnchantPermanent(IsCardtype(Creature)),` + rules
			}
			src := `Card(Name:"Reference",Typeline:(Supertypes:[],Cardtypes:` + typ + `,Subtypes:` + sub + `),Rules:[` + rules + `]` + pt + `)`
			nodes, err := Parse(strings.NewReader(src), "reference.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			card := factory()
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			subject := source
			if ref == "HostPermanent" {
				subject = g.PutOnBattlefield(mage.NewCreature("Host", "", 3, 10), a.PlayerID())
				g.Attach(source.ID(), subject.ID())
			}
			source.RevokeBaseAttr(core.AttrSummonSick)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if !sourceCombatPrevented(g, subject.ID(), b.PlayerID()) {
				t.Fatal("reference did not select subject")
			}
		})
	}
}

func TestRONTargetCombatPreventionEventReference(t *testing.T) {
	action := strings.ReplaceAll(targetCombatPrevention, "Ref_TargetPermanent", "Trigger_ThatPermanent")
	src := `Card(Name:"Event reference",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),TriggerA(WhenAPermanentBecomesTapped(SinglePermanent(HostPermanent)),ActionList([` + action + `]))])`
	nodes, err := Parse(strings.NewReader(src), "event.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	card := factory()
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 3, 10), a.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 2, 10), a.PlayerID())
	aura := g.PutOnBattlefield(card, a.PlayerID())
	g.Attach(aura.ID(), first.ID())
	g.TapPermanent(first)
	g.PutTriggersOnStack()
	g.Attach(aura.ID(), second.ID())
	clone := g.Clone()
	g.ResolveStack()
	if !sourceCombatPrevented(g, first.ID(), b.PlayerID()) || sourceCombatPrevented(g, second.ID(), b.PlayerID()) {
		t.Fatal("event read followed current attachment")
	}
	if sourceCombatPrevented(clone, first.ID(), b.PlayerID()) {
		t.Fatal("resolution changed pending event clone")
	}
	clone.ResolveStack()
	if !sourceCombatPrevented(clone, first.ID(), b.PlayerID()) {
		t.Fatal("clone lost event identity")
	}
}
