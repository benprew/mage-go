package ron

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func combatDefenderProbe(t *testing.T, name string) mage.CardFactory {
	t.Helper()
	// A probe is needed because the complete card has other unsupported actions.
	set := "LEB"
	if name == "Floral Spuzzem" {
		set = "LEG"
	}
	n := hostUntapRecord(t, set, name)
	n.Fields = maps.Clone(n.Fields)
	n.Fields["Name"] = Node{Kind: String, String: "Defender target probe"}
	rules := n.Fields["Rules"]
	rules.Args = slices.Clone(rules.Args)
	probe, err := Parse(strings.NewReader(`ActionList([TapPermanent(Ref_TargetPermanent),GainLife(Integer(1))])`), "probe.ron")
	if err != nil {
		t.Fatal(err)
	}
	for i, rule := range rules.Args {
		if rule.Name != "SpellActions" && rule.Name != "TriggerA" {
			continue
		}
		rule.Args = slices.Clone(rule.Args)
		index := len(rule.Args) - 1
		targeted := rule.Args[index]
		targeted.Args = slices.Clone(targeted.Args)
		targeted.Args[1] = probe[0]
		rule.Args[index] = targeted
		rules.Args[i] = rule
	}
	n.Fields["Rules"] = rules
	factory, err := Compile(n)
	if err != nil {
		t.Fatal(err)
	}
	if card := factory(); mage.ValidateCardDefinition(card.Name(), card) != nil {
		t.Fatal("invalid probe definition")
	}
	return factory
}

func TestRONCombatDefenderSpellTargets(t *testing.T) {
	for _, name := range []string{"Blaze of Glory", "False Orders"} {
		for _, defendingCaster := range []bool{false, true} {
			for _, change := range []string{"none", "control", "return", "phase", "cycle", "shroud", "copy"} {
				t.Run(fmt.Sprintf("%s/%v/%s", name, defendingCaster, change), func(t *testing.T) {
					g, a, b := ronGame()
					caster := a
					if defendingCaster {
						caster = b
					}
					factory := combatDefenderProbe(t, name)
					card := factory()
					card.SetOwner(caster.PlayerID())
					caster.AddToHand(card)
					own := g.PutOnBattlefield(mage.NewCreature("Active creature", "", 2, 4), a.PlayerID())
					victimCard := mage.NewCreature("Defender creature", "", 2, 4)
					victimCard.SetOwner(a.PlayerID())
					victim := g.PutOnBattlefield(victimCard, b.PlayerID())
					color := core.White
					step := core.DeclareAttackers
					if name == "False Orders" {
						color, step = core.Red, core.DeclareBlockers
					}
					caster.ManaPool().Add(color, 1)
					announce := func(id uuid.UUID) error {
						return g.CastSpellByID(caster.PlayerID(), card.ID(), []uuid.UUID{id}, 0)
					}
					if announce(victim.ID()) == nil || len(caster.Hand()) != 1 || caster.ManaPool().TotalMana() != 1 {
						t.Fatal("casting outside the record's combat window or rollback")
					}
					g.SetStep(step)
					if announce(own.ID()) == nil || g.StackSize() != 0 || caster.ManaPool().TotalMana() != 1 {
						t.Fatal("active player's creature accepted or payment changed")
					}
					caster.ManaPool().Clear()
					if announce(victim.ID()) == nil || len(caster.Hand()) != 1 || g.StackSize() != 0 {
						t.Fatal("unpaid spell accepted")
					}
					caster.ManaPool().Add(color, 1)
					if err := announce(victim.ID()); err != nil {
						t.Fatal(err)
					}
					if victim.Tapped || caster.Life() != 20 || caster.ManaPool().TotalMana() != 0 {
						t.Fatal("wrong effect timing or casting cost")
					}
					switch change {
					case "control":
						g.AddControlEffect(mage.ControlEffectSpec{SourceID: own.ID(), TargetID: victim.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
					case "return":
						g.RemoveFromBattlefield(victim)
						g.PutOnBattlefield(victimCard, b.PlayerID())
					case "phase":
						g.PhaseOut(victim.ID())
					case "cycle":
						g.PhaseIn(g.PhaseOut(victim.ID()))
					case "shroud":
						g.MutablePermanent(victim.ID()).GrantBaseAttr(core.Shroud)
					case "copy":
						other := a
						if caster == a {
							other = b
						}
						if g.CopyStackObjectDirect(g.StackPeek(), other.PlayerID(), true) == nil {
							t.Fatal("copy failed")
						}
					}
					clone := g.Clone()
					g.ResolveStack()
					if clone.FindPermanentIncludingPhased(victim.ID()).Tapped || clone.GetPlayer(caster.PlayerID()).Life() != 20 {
						t.Fatal("resolution changed clone")
					}
					clone.ResolveStack()
					legal := change == "none" || change == "cycle" || change == "copy"
					for _, game := range []*mage.Game{g, clone} {
						if game.FindPermanentIncludingPhased(victim.ID()).Tapped != legal || game.FindPermanent(own.ID()).Tapped {
							t.Fatal("target legality, incarnation or copy defender")
						}
						want := 20
						if legal {
							want++
						}
						if game.GetPlayer(caster.PlayerID()).Life() != want {
							t.Fatal("illegal target resolved", game.GetPlayer(caster.PlayerID()).Life())
						}
						other := a
						if caster == a {
							other = b
						}
						otherLife := 20
						if change == "copy" {
							otherLife++
						}
						if game.GetPlayer(other.PlayerID()).Life() != otherLife {
							t.Fatal("copy lost its controller")
						}
					}
				})
			}
		}
	}
}

func TestRONCombatDefenderCapturedTarget(t *testing.T) {
	for _, change := range []string{"none", "control", "source-return", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(combatDefenderProbe(t, "Floral Spuzzem")(), a.PlayerID())
			victim := g.PutOnBattlefield(mage.NewArtifact("Defender artifact", ""), b.PlayerID())
			own := g.PutOnBattlefield(mage.NewArtifact("Active artifact", ""), a.PlayerID())
			g.FireEvent(core.GameEvent{Type: core.EvtAttacksUnblocked, SourceID: source.ID(), PlayerID: a.PlayerID(), TargetID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || !slices.Equal(g.StackPeek().Targets, []uuid.UUID{victim.ID()}) {
				t.Fatal("saved defender unavailable at trigger placement")
			}
			switch change {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: victim.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "source-return":
				g.RemoveFromBattlefield(source)
				g.PutOnBattlefield(source.Card, b.PlayerID())
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), true) == nil {
					t.Fatal("copy failed")
				}
			}
			g.SetActivePlayerIndex(1)
			g.SetStep(core.PostcombatMain)
			g.ResolveStack()
			if g.FindPermanent(victim.ID()).Tapped != (change != "control") || g.FindPermanent(own.ID()).Tapped {
				t.Fatal("event defender followed current combat or source")
			}
		})
	}
}

func TestRONCombatDefenderIndependentTargets(t *testing.T) {
	for _, wrapped := range []bool{false, true} {
		t.Run(fmt.Sprint(wrapped), func(t *testing.T) {
			player := "SpellDefendingPlayer"
			if wrapped {
				player = "SinglePlayer(" + player + ")"
			}
			src := `Card(Name:"Independent",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(And([IsCardtype(Creature),ControlledByAPlayer(` + player + `)]))],ActionList([TapPermanent(Ref_TargetPermanent)])))])`
			nodes, err := Parse(strings.NewReader(src), "independent-defender.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			g.SetStep(core.BeginCombat)
			first := g.PutOnBattlefield(mage.NewCreature("First target", "", 2, 4), b.PlayerID())
			second := g.PutOnBattlefield(mage.NewCreature("Second target", "", 2, 4), b.PlayerID())
			for i, caster := range []*mage.BasePlayer{a, b} {
				card := factory()
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(caster.PlayerID())
				caster.AddToHand(card)
				id := first.ID()
				if i == 1 {
					id = second.ID()
				}
				if err := g.CastSpellByID(caster.PlayerID(), card.ID(), []uuid.UUID{id}, 0); err != nil {
					t.Fatal(err)
				}
			}
			g.ResolveTopOfStack()
			if g.FindPermanent(first.ID()).Tapped || !g.FindPermanent(second.ID()).Tapped {
				t.Fatal("pending instances shared target state")
			}
			g.ResolveStack()
			if !g.FindPermanent(first.ID()).Tapped {
				t.Fatal("original target changed")
			}
		})
	}
}

func TestRONCombatDefenderModeAndGroups(t *testing.T) {
	for _, context := range []string{"mode", "groups"} {
		t.Run(context, func(t *testing.T) {
			g, a, b := ronModalGame(0)
			g.SetStep(core.BeginCombat)
			victim := g.PutOnBattlefield(mage.NewCreature("Defender creature", "", 2, 4), a.PlayerID())
			other := g.PutOnBattlefield(mage.NewArtifact("Other", ""), b.PlayerID())
			filter := `And([IsCardtype(Creature),And([ControlledByAPlayer(SpellDefendingPlayer)])])`
			body := `Modal_ChooseOne([Targeted([TargetPermanent(` + filter + `)],ActionList([TapPermanent(Ref_TargetPermanent)])),ActionList([GainLife(Integer(5))])])`
			ids := []uuid.UUID{victim.ID()}
			if context == "groups" {
				body = `Targeted([TargetPermanent(` + filter + `),TargetPermanent(IsCardtype(Artifact))],ActionList([TapPermanent(Ref_TargetPermanent1),TapPermanent(Ref_TargetPermanent2)]))`
				ids = append(ids, other.ID())
			}
			card := singularGroupsFactory(t, body, false)()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Blue, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), ids, 0); err != nil {
				t.Fatal(err)
			}
			if context == "groups" {
				g.MutablePermanent(victim.ID()).GrantBaseAttr(core.Shroud)
			}
			g.ResolveStack()
			if g.FindPermanent(victim.ID()).Tapped != (context == "mode") || g.FindPermanent(other.ID()).Tapped != (context == "groups") || a.Life() != 20 {
				t.Fatal("mode or group target identity and partial legality")
			}
		})
	}
}

func TestRONCombatDefenderCopyRetarget(t *testing.T) {
	g, a, b := opponentTargetGame()
	g.SetStep(core.DeclareAttackers)
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 4), b.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 2, 4), b.PlayerID())
	card := combatDefenderProbe(t, "Blaze of Glory")()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.White, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{first.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	b.requested = []uuid.UUID{second.ID()}
	copied := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), true)
	if copied == nil || !slices.Equal(copied.Targets, []uuid.UUID{second.ID()}) {
		t.Fatal("copy could not retarget to its controller's defending creature")
	}
	if len(b.candidates) != 1 || !slices.Contains(b.candidates[0], first.ID()) || !slices.Contains(b.candidates[0], second.ID()) {
		t.Fatal("copy used the attacking player for target control")
	}
	clone := g.Clone()
	g.ResolveTopOfStack()
	if g.FindPermanent(first.ID()).Tapped || !g.FindPermanent(second.ID()).Tapped || clone.FindPermanent(second.ID()).Tapped {
		t.Fatal("retarget overwrote the original or the clone")
	}
	g.ResolveStack()
	clone.ResolveStack()
	for _, game := range []*mage.Game{g, clone} {
		if !game.FindPermanent(first.ID()).Tapped || !game.FindPermanent(second.ID()).Tapped || game.GetPlayer(a.PlayerID()).Life() != 21 || game.GetPlayer(b.PlayerID()).Life() != 21 {
			t.Fatal("copy target or controller binding lost")
		}
	}
}

func TestRONCombatDefenderDeclarations(t *testing.T) {
	for _, declaration := range []string{
		`ControlledByAPlayer(SpellDefendingPlayer())`, `ControlledByAPlayer(SinglePlayer(SpellDefendingPlayer()))`,
		`ControlledByAPlayer(SinglePlayer())`, `ControlledByAPlayer(SinglePlayer(SpellDefendingPlayer,You))`,
		`ControlledByAPlayer(DefendingPlayer)`, `ControlledByAPlayer(SinglePlayer(DefendingPlayer))`,
		`ControlledByPlayer(SpellDefendingPlayer)`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(` + declaration + `)],ActionList([TapPermanent(Ref_TargetPermanent)])))])`
		nodes, err := Parse(strings.NewReader(src), "invalid-defender.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid defender declaration accepted", declaration)
		}
	}
	for _, event := range []string{"Activated(PayMana([])", "TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You))", "TriggerA(WhenACreatureAttacks(SinglePermanent(ThisPermanent))"} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[` + event + `,Targeted([TargetPermanent(ControlledByAPlayer(SpellDefendingPlayer))],ActionList([TapPermanent(Ref_TargetPermanent)])))])`
		nodes, err := Parse(strings.NewReader(src), "invalid-context.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("spell defender escaped spell scope", event)
		}
	}
}

func TestRONCombatDefenderRecords(t *testing.T) {
	var records []string
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI", "LEG"} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.Contains(line, `Name: "Blaze of Glory"`) || strings.Contains(line, `Name: "False Orders"`) || strings.Contains(line, `Name: "Floral Spuzzem"`) {
				records = append(records, line)
				nodes, err := Parse(strings.NewReader(line), set+".ron")
				if err != nil {
					t.Fatal(err)
				}
				if _, err := Compile(nodes[0]); err == nil || strings.Contains(err.Error(), "permanent target controller") {
					t.Fatal("complete record must reject its remaining operation, after the target", err)
				}
			}
		}
	}
	if len(records) != 11 {
		t.Fatal("record inventory changed", len(records))
	}
	files := fstest.MapFS{"records.ron": {Data: []byte(strings.Join(records, "\n"))}}
	if registry, err := LoadFS(files, "*.ron"); err == nil || registry != nil {
		t.Fatal("unsupported records produced a registry")
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, sig := range report.Signatures {
		if (sig.Name == "SpellDefendingPlayer" || sig.Name == "DefendingPlayer") && sig.Role != "player_reference" {
			t.Fatal("defender role", sig.Name, sig.Role)
		}
	}
}
