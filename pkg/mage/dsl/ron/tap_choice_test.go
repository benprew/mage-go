package ron

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronTapChoicePlayer struct {
	*mage.BasePlayer
	accept  bool
	prompts []string
}

func (p *ronTapChoicePlayer) ChooseMayAbility(prompt string) bool {
	p.prompts = append(p.prompts, prompt)
	return p.accept
}

func TestRONTapChoiceRealCards(t *testing.T) {
	for _, name := range []string{"Twiddle", "Hyperion Blacksmith"} {
		for _, scenario := range []string{"tap", "untap", "decline-tap", "decline-untap", "state-changed", "stun", "decline-stun", "returned", "phased", "phase-cycle", "control", "source-left", "source-returned", "source-control", "source-phased", "shroud"} {
			t.Run(name+"/"+scenario, func(t *testing.T) {
				a := &ronTapChoicePlayer{BasePlayer: mage.NewBasePlayer("A"), accept: !strings.HasPrefix(scenario, "decline")}
				b := &ronTapChoicePlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
				g := mage.NewGame(a, b)
				g.SetStep(core.PrecombatMain)
				g.SetTurn(1)
				targetCard := mage.NewArtifact("Target", "")
				targetCard.SetOwner(a.PlayerID())
				target := g.PutOnBattlefield(targetCard, b.PlayerID())
				initiallyTapped := scenario == "untap" || scenario == "decline-untap" || scenario == "stun" || scenario == "decline-stun"
				target.Tapped = initiallyTapped
				if strings.Contains(scenario, "stun") {
					target.AddCounter(core.Stun, 1)
				}
				set := "LEB"
				if name == "Hyperion Blacksmith" {
					set = "LEG"
				}
				card := repositoryCard(t, set, name)
				if err := mage.ValidateCardDefinition(name, card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				var source *mage.Permanent
				if name == "Twiddle" {
					a.AddToHand(card)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target.ID()}, 0); err == nil {
						t.Fatal("unpaid spell cast")
					}
					if g.StackSize() != 0 || len(a.Hand()) != 1 || target.Tapped != initiallyTapped {
						t.Fatal("failed cast changed state")
					}
					a.ManaPool().Add(core.Blue, 1)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
						t.Fatal(err)
					}
				} else {
					source = g.PutOnBattlefield(card, a.PlayerID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err == nil {
						t.Fatal("summoning sick activation accepted")
					}
					if g.StackSize() != 0 || source.Tapped {
						t.Fatal("failed activation paid tap cost")
					}
					source.RevokeBaseAttr(core.AttrSummonSick)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
						t.Fatal(err)
					}
					if !g.FindPermanent(source.ID()).Tapped {
						t.Fatal("tap cost missing")
					}
				}
				if len(a.prompts) != 0 || len(b.prompts) != 0 || target.Tapped != initiallyTapped {
					t.Fatal("choice or action happened at announcement")
				}
				legal := true
				switch scenario {
				case "state-changed":
					g.TapPermanent(target)
				case "returned":
					g.RemoveFromBattlefield(target)
					target = g.PutOnBattlefield(targetCard, b.PlayerID())
					legal = false
				case "phased":
					g.PhaseOut(target.ID())
					legal = false
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(target.ID()))
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: target.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
					legal = name == "Twiddle"
				case "shroud":
					target.GrantBaseAttr(core.Shroud)
					legal = false
				case "source-left", "source-returned":
					if source != nil {
						g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
						if scenario == "source-returned" {
							g.PutOnBattlefield(card, b.PlayerID())
						}
					}
				case "source-control":
					if source != nil {
						g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					}
				case "source-phased":
					if source != nil {
						g.PhaseOut(source.ID())
					}
				}
				before := g.FindPermanentIncludingPhased(target.ID()).Tapped
				clone := g.Clone()
				clone.ResolveTopOfStack()
				if g.FindPermanentIncludingPhased(target.ID()).Tapped != before || len(a.prompts) != 0 {
					t.Fatal("clone changed live state or decisions")
				}
				g.ResolveTopOfStack()
				want := before
				if legal && a.accept && scenario != "stun" {
					want = !before
				}
				for _, game := range []*mage.Game{g, clone} {
					p := game.FindPermanentIncludingPhased(target.ID())
					expected := want
					if game == clone {
						expected = before
					}
					if p.Tapped != expected {
						t.Fatalf("tapped=%t want=%t", p.Tapped, expected)
					}
					wantStun := uint8(0)
					if scenario == "decline-stun" || (game == clone && scenario == "stun") {
						wantStun = 1
					}
					if p.Counters[core.Stun] != wantStun {
						t.Fatal("untap replacement or decline consumed wrong counter")
					}
				}
				offers := 0
				if legal {
					offers = 1
				}
				if len(a.prompts) != offers || len(b.prompts) != 0 {
					t.Fatalf("wrong chooser: A=%v B=%v", a.prompts, b.prompts)
				}
				if offers == 1 {
					verb := "tap "
					if before {
						verb = "untap "
					}
					if a.prompts[0] != verb+"Target" {
						t.Fatalf("wrong resolution state: %q", a.prompts[0])
					}
				}
			})
		}
	}
}

func TestRONTapChoiceInvalid(t *testing.T) {
	for _, action := range []string{
		`MayAction`, `MayAction()`, `MayAction(TapOrUntapPermanent(Ref_TargetPermanent),DrawACard)`,
		`MayAction(TapOrUntapPermanent)`, `MayAction(TapOrUntapPermanent())`,
		`MayAction(TapOrUntapPermanent(Ref_TargetPermanent,Ref_TargetPermanent))`,
		`MayAction(TapOrUntapPermanent(Ref_TargetPermanent()))`,
		`MayAction(TapOrUntapPermanent(Ref_TargetPermanent1))`,
		`MayAction(TapOrUntapPermanent(Ref_TargetPlayer))`,
		`MayAction(TapOrUntapPermanent(ThisPermanent))`,
		`MayAction(TapOrUntapPermanent(HostPermanent))`,
		`MayAction(TapOrUntapPermanent(Trigger_ThatPermanent))`,
		`TapOrUntapPermanent(Ref_TargetPermanent)`, `MayAction(DrawACard())`,
		`MayAction(ActionList([TapOrUntapPermanent(Ref_TargetPermanent)]))`,
		`MayAction(MayAction(TapOrUntapPermanent(Ref_TargetPermanent)))`,
	} {
		t.Run(action, func(t *testing.T) {
			nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([`+action+`])))])`), "invalid.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = Compile(nodes[0]); err == nil {
				t.Fatal("accepted invalid declaration")
			}
		})
	}
	for _, rule := range []string{
		`SpellActions(ActionList([MayAction(TapOrUntapPermanent(Ref_TargetPermanent))]))`,
		`SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([]))),SpellActions(ActionList([MayAction(TapOrUntapPermanent(Ref_TargetPermanent))]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[`+rule+`])`), "scope.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted out-of-scope target")
		}
	}
}

func TestRONTapChoiceInventory(t *testing.T) {
	src := `Card(Name:"Choice",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([MayAction(TapOrUntapPermanent(Ref_TargetPermanent))])))])`
	report, err := InspectFS(fstest.MapFS{"choice.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"MayAction": "action", "TapOrUntapPermanent": "action", "Ref_TargetPermanent": "permanent_reference"} {
		if !slices.ContainsFunc(report.Signatures, func(s InventorySignature) bool { return s.Name == name && s.Role == role }) {
			t.Fatalf("missing %s role for %s", role, name)
		}
	}
}

func TestRONTapChoiceTargetFiltersAndRollback(t *testing.T) {
	for _, name := range []string{"Twiddle", "Hyperion Blacksmith"} {
		for _, typ := range []string{"artifact", "creature", "land", "enchantment", "own-artifact", "shroud", "phased"} {
			t.Run(name+"/"+typ, func(t *testing.T) {
				a := &ronTapChoicePlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
				b := mage.NewBasePlayer("B")
				g := mage.NewGame(a, b)
				g.SetStep(core.PrecombatMain)
				g.SetTurn(1)
				var candidate mage.Card = mage.NewArtifact("Candidate", "")
				switch typ {
				case "creature":
					candidate = mage.NewCreature("Candidate", "", 2, 2)
				case "land":
					candidate = mage.NewLand("Candidate")
				case "enchantment":
					candidate = mage.NewEnchantment("Candidate", "")
				}
				controller := b.PlayerID()
				if typ == "own-artifact" {
					controller = a.PlayerID()
				}
				p := g.PutOnBattlefield(candidate, controller)
				if typ == "shroud" {
					p.GrantBaseAttr(core.Shroud)
				}
				if typ == "phased" {
					g.PhaseOut(p.ID())
				}
				legal := typ == "artifact" || (name == "Twiddle" && (typ == "creature" || typ == "land" || typ == "own-artifact"))
				var err error
				var source *mage.Permanent
				if name == "Twiddle" {
					card := repositoryCard(t, "LEB", name)
					card.SetOwner(a.PlayerID())
					a.AddToHand(card)
					a.ManaPool().Add(core.Blue, 1)
					err = g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{p.ID()}, 0)
				} else {
					card := repositoryCard(t, "LEG", name)
					card.SetOwner(a.PlayerID())
					source = g.PutOnBattlefield(card, a.PlayerID())
					source.RevokeBaseAttr(core.AttrSummonSick)
					err = g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{p.ID()})
				}
				if (err == nil) != legal {
					t.Fatalf("legality=%t want=%t: %v", err == nil, legal, err)
				}
				if !legal {
					if g.StackSize() != 0 || len(a.prompts) != 0 || g.FindPermanentIncludingPhased(p.ID()).Tapped {
						t.Fatal("illegal declaration changed state")
					}
					if name == "Twiddle" && (a.ManaPool().TotalMana() != 1 || len(a.Hand()) != 1) {
						t.Fatal("illegal target consumed spell cost")
					}
					if source != nil && g.FindPermanent(source.ID()).Tapped {
						t.Fatal("illegal target consumed tap cost")
					}
					return
				}
				g.ResolveTopOfStack()
				if !g.FindPermanent(p.ID()).Tapped || len(a.prompts) != 1 {
					t.Fatal("legal target not tapped")
				}
			})
		}
	}
}

func TestRONTapChoiceCopiesAndInstances(t *testing.T) {
	for _, name := range []string{"Twiddle", "Hyperion Blacksmith"} {
		for _, retarget := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%t", name, retarget), func(t *testing.T) {
				a := &ronTapChoicePlayer{BasePlayer: mage.NewBasePlayer("A"), accept: false}
				b := &ronTapChoicePlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
				g := mage.NewGame(a, b)
				g.SetStep(core.PrecombatMain)
				g.SetTurn(1)
				first := g.PutOnBattlefield(mage.NewArtifact("First", ""), b.PlayerID())
				second := g.PutOnBattlefield(mage.NewArtifact("Second", ""), a.PlayerID())
				set := "LEB"
				if name == "Hyperion Blacksmith" {
					set = "LEG"
				}
				card, other := repositoryCard(t, set, name), repositoryCard(t, set, name)
				if card.Abilities()[0].(*mage.ActionDefinition).Effects()[0] == other.Abilities()[0].(*mage.ActionDefinition).Effects()[0] {
					t.Fatal("factory instances share effects")
				}
				if name == "Twiddle" {
					castRONSpell(t, g, a.BasePlayer, card, first.ID())
				} else {
					card.SetOwner(a.PlayerID())
					source := g.PutOnBattlefield(card, a.PlayerID())
					source.RevokeBaseAttr(core.AttrSummonSick)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{first.ID()}); err != nil {
						t.Fatal(err)
					}
				}
				original := g.StackPeek()
				copyObj := g.CopyStackObjectDirect(original, b.PlayerID(), false)
				if copyObj == nil {
					t.Fatal("copy failed")
				}
				target := first
				if retarget {
					target = second
					copyObj.Targets[0] = second.ID()
					copyObj.TargetZones[second.ID()] = core.ZoneBattlefield
				}
				clone := g.Clone()
				g.ResolveTopOfStack()
				copyLegal := name == "Twiddle" || retarget
				if g.FindPermanent(target.ID()).Tapped != copyLegal || len(b.prompts) != map[bool]int{true: 1, false: 0}[copyLegal] {
					t.Fatal("copy chose wrong player or target")
				}
				if original.Targets[0] != first.ID() || len(a.prompts) != 0 {
					t.Fatal("copy overwrote original binding or choice")
				}
				g.ResolveTopOfStack()
				if len(a.prompts) != 1 || g.FindPermanent(first.ID()).Tapped != (copyLegal && !retarget) {
					t.Fatal("original did not decline independently")
				}
				clone.ResolveStack()
				if clone.FindPermanent(first.ID()).Tapped || clone.FindPermanent(second.ID()).Tapped {
					t.Fatal("clone reused live decision or permanent state")
				}
			})
		}
	}
}

func TestRONTapChoiceContexts(t *testing.T) {
	for _, context := range []string{"source", "host", "event", "trigger-target", "mode"} {
		t.Run(context, func(t *testing.T) {
			a := &ronTapChoicePlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
			g := mage.NewGame(a, mage.NewBasePlayer("B"))
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 2), a.PlayerID())
			rule := `Activated(PayMana([]),ActionList([MayAction(TapOrUntapPermanent(ThisPermanent))]))`
			typ := "Artifact"
			switch context {
			case "host":
				typ = "Enchantment"
				rule = `EnchantPermanent(IsCardtype(Creature)),Activated(PayMana([]),ActionList([MayAction(TapOrUntapPermanent(HostPermanent))]))`
			case "event":
				typ = "Enchantment"
				rule = `EnchantPermanent(IsCardtype(Creature)),TriggerA(WhenAPermanentBecomesTapped(SinglePermanent(HostPermanent)),ActionList([MayAction(TapOrUntapPermanent(Trigger_ThatPermanent))]))`
			case "trigger-target":
				rule = `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([MayAction(TapOrUntapPermanent(Ref_TargetPermanent))])))`
			case "mode":
				typ = "Instant"
				rule = `SpellActions(Modal_ChooseOne([Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([MayAction(TapOrUntapPermanent(Ref_TargetPermanent))])),ActionList([GainLife(Integer(1))])]))`
			}
			sourceText := `Card(Name:"Contexts",Typeline:(Supertypes:[],Cardtypes:[` + typ + `],Subtypes:[]),Rules:[` + rule + `])`
			if context == "host" || context == "event" {
				sourceText = strings.Replace(sourceText, "Subtypes:[]", "Subtypes:[Aura]", 1)
			}
			nodes, err := Parse(strings.NewReader(sourceText), "contexts.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			card := factory()
			if err = mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			if context == "mode" {
				castRONSpell(t, g, a.BasePlayer, card, host.ID())
			} else {
				source := g.PutOnBattlefield(card, a.PlayerID())
				if context == "host" || context == "event" {
					g.Attach(source.ID(), host.ID())
				}
				switch context {
				case "event":
					g.TapPermanent(host)
					g.PutTriggersOnStack()
				case "trigger-target":
					g.PutTriggersOnStack()
				default:
					if err = g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
				}
				if context == "source" {
					host = source
				}
			}
			if g.StackSize() != 1 {
				t.Fatalf("stack=%d", g.StackSize())
			}
			g.ResolveTopOfStack()
			if len(a.prompts) != 1 || g.FindPermanent(host.ID()).Tapped == (context == "event") {
				t.Fatal("wrong context reference or chooser")
			}
		})
	}
}

func TestRONTapChoiceSequenceAndEvents(t *testing.T) {
	for _, accept := range []bool{false, true} {
		t.Run(fmt.Sprint(accept), func(t *testing.T) {
			a := &ronTapChoicePlayer{BasePlayer: mage.NewBasePlayer("A"), accept: accept}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 8), b.PlayerID())
			aura := repositoryCard(t, "LEG", "Spirit Shackle")
			aura.SetOwner(b.PlayerID())
			source := g.PutOnBattlefield(aura, b.PlayerID())
			g.Attach(source.ID(), host.ID())
			factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([MayAction(TapOrUntapPermanent(Ref_TargetPermanent)),GainLife(Integer(1)),MayAction(TapOrUntapPermanent(Ref_TargetPermanent))])))`)
			card := factory()
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			castRONSpell(t, g, a.BasePlayer, card, host.ID())
			g.ResolveTopOfStack()
			if g.FindPermanent(host.ID()).Tapped || len(a.prompts) != 2 || a.Life() != 21 {
				t.Fatal("optional choice changed sibling sequencing")
			}
			if accept && (a.prompts[0] != "tap Host" || a.prompts[1] != "untap Host") {
				t.Fatal("second choice used announcement state")
			}
			g.PutTriggersOnStack()
			want := 0
			if accept {
				want = 1
			}
			if g.StackSize() != want {
				t.Fatalf("tap triggers=%d want=%d", g.StackSize(), want)
			}
			g.ResolveStack()
			if int(g.FindPermanent(host.ID()).Counters[core.M0M2]) != want {
				t.Fatal("tap event bypassed RON observer")
			}
		})
	}
}
