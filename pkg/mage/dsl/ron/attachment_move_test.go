package ron

import (
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONAttachmentMoveAlteration(t *testing.T) {
	for _, change := range []string{"creature", "land", "creature-land", "no-host", "enchant-limit", "protected", "target-shroud", "target-return", "target-phase", "host-phase", "cycle", "move-before", "copy", "retarget"} {
		t.Run(change, func(t *testing.T) {
			factory, err := Compile(chosenDiscardRecord(t, "LEG", "Enchantment Alteration"))
			if err != nil {
				t.Fatal(err)
			}
			g, base, b := ronGame()
			a := &ronPermanentChoicePlayer{Player: base}
			g.SetPlayerAt(0, a)
			hostCard := mage.NewCreature("Host", "", 2, 5)
			if change == "land" {
				hostCard = mage.NewLand("Host")
			}
			if change == "creature-land" {
				hostCard.AddType(core.TypeLand)
			}
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			first := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 5, mage.WithKeyword(core.Shroud)), b.PlayerID())
			second := g.PutOnBattlefield(mage.NewCreature("Second", "", 2, 5, mage.WithKeyword(core.Hexproof)), b.PlayerID())
			land := g.PutOnBattlefield(mage.NewLand("Land"), b.PlayerID())
			auraCard := mage.NewAura("Aura", "{U}", mage.WithCastTarget(mage.TargetPermanent()))
			if change == "enchant-limit" {
				auraCard = mage.NewAura("Aura", "{U}", mage.WithCastTarget(mage.TargetPermanentYouControl()))
			}
			aura := g.PutOnBattlefield(auraCard, b.PlayerID())
			g.Attach(aura.ID(), host.ID())
			want := second.ID()
			if change == "land" || change == "creature-land" {
				want = land.ID()
			}
			a.choose = func(candidates []*mage.Permanent) *mage.Permanent {
				if slices.ContainsFunc(candidates, func(p *mage.Permanent) bool { return p.ID() == host.ID() || p.ID() == aura.ID() }) {
					t.Fatal("current host or Aura offered")
				}
				for _, p := range candidates {
					if change == "retarget" && p.HasType(core.TypeLand) {
						return p
					}
					if p.ID() == want {
						return p
					}
				}
				t.Fatal("required destination missing")
				return nil
			}
			spell := factory()
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{aura.ID()}, 0); err == nil || a.calls != 0 || g.StackSize() != 0 {
				t.Fatal("unpaid cast changed state")
			}
			a.ManaPool().Add(core.Blue, 1)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{host.ID()}, 0); err == nil || a.ManaPool().TotalMana() != 1 {
				t.Fatal("invalid target spent mana")
			}
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{aura.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			if a.calls != 0 {
				t.Fatal("host chosen at announcement")
			}
			var retargeted, retargetedHost uuid.UUID
			switch change {
			case "no-host":
				g.RemoveFromBattlefield(first)
				g.RemoveFromBattlefield(second)
				want = host.ID()
			case "enchant-limit":
				g.AddControlEffect(mage.ControlEffectSpec{TargetID: first.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				g.AddControlEffect(mage.ControlEffectSpec{TargetID: second.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				want = host.ID()
			case "protected":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.GrantProtectionColors(mage.This, core.Blue)), second.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				want = first.ID()
			case "target-shroud":
				g.MutablePermanent(aura.ID()).GrantBaseAttr(core.Shroud)
				want = host.ID()
			case "target-return":
				g.RemoveFromBattlefield(aura)
				returned := g.PutOnBattlefield(auraCard, b.PlayerID())
				g.Attach(returned.ID(), host.ID())
				want = host.ID()
			case "target-phase":
				g.PhaseOut(aura.ID())
				want = host.ID()
			case "host-phase":
				g.PhaseOut(host.ID())
				want = host.ID()
			case "cycle":
				g.PhaseIn(g.PhaseOut(host.ID()))
			case "move-before":
				g.Attach(aura.ID(), land.ID())
				newLand := g.PutOnBattlefield(mage.NewLand("New Land"), b.PlayerID())
				a.choose = func(candidates []*mage.Permanent) *mage.Permanent { return candidates[0] }
				want = newLand.ID()
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "retarget":
				other := g.PutOnBattlefield(mage.NewAura("Other Aura", "{U}", mage.WithCastTarget(mage.TargetPermanent())), b.PlayerID())
				g.Attach(other.ID(), land.ID())
				newLand := g.PutOnBattlefield(mage.NewLand("New Land"), b.PlayerID())
				retargeted, retargetedHost = other.ID(), newLand.ID()
				phased := g.PhaseOut(aura.ID())
				copied := g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), true)
				if copied == nil || !slices.Equal(copied.Targets, []uuid.UUID{other.ID()}) {
					t.Fatal("retarget failed")
				}
				g.PhaseIn(phased)
			}
			if change == "copy" {
				a.choose = func(candidates []*mage.Permanent) *mage.Permanent {
					for _, p := range candidates {
						if p.ID() == second.ID() {
							return p
						}
					}
					return first
				}
				want = first.ID()
			}
			clone := g.Clone()
			clonePlayer := &ronPermanentChoicePlayer{Player: clone.GetPlayer(a.PlayerID()), choose: a.choose}
			clone.SetPlayerAt(0, clonePlayer)
			g.ResolveStack()
			if clone.FindPermanentIncludingPhased(aura.ID()).AttachedTo != host.ID() && change != "move-before" {
				t.Fatal("resolution changed clone")
			}
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				if retargeted != uuid.Nil && game.FindPermanent(retargeted).AttachedTo != retargetedHost {
					t.Fatal("copy lost its new target or host comparison")
				}
				if game.FindPermanentIncludingPhased(aura.ID()).AttachedTo != want {
					t.Fatalf("host = %v, want %v", game.FindPermanentIncludingPhased(aura.ID()).AttachedTo, want)
				}
				if game.FindPermanentIncludingPhased(aura.ID()).ControllerID() != b.PlayerID() {
					t.Fatal("movement changed controller")
				}
			}
		})
	}
}

func TestRONAttachmentMoveInvalid(t *testing.T) {
	for _, body := range []string{`AttachPermanentToAPermanent`, `AttachPermanentToAPermanent(ThisPermanent)`, `AttachPermanentToAPermanent(ThisPermanent,AnyPermanent,AnyPermanent)`, `AttachPermanentToAPermanent(You,AnyPermanent)`, `AttachPermanentToAPermanent(TheChosenPermanent,AnyPermanent)`, `AttachPermanentToAPermanent(ThisPermanent,You)`, `AttachPermanentToAPermanent(ThisPermanent,And([]))`, `AttachPermanentToAPermanent(ThisPermanent,Ref_TargetPermanent)`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+body+`]))])`), "invalid-attach.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration accepted", body)
		}
	}
	for _, ctx := range []scope{{enteringSource: true}, {drawReplacementBody: true}} {
		nodes, err := Parse(strings.NewReader(`AttachPermanentToAPermanent(ThisPermanent,AnyPermanent)`), "boundary.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := compileEffect(nodes[0], ctx); err == nil {
			t.Fatal("foreign context accepted")
		}
	}
	nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid result",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([AttachPermanentToAPermanent(ThisPermanent,AnyPermanent),TapPermanent(TheChosenPermanent)]))])`), "invalid-result.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Compile(nodes[0]); err == nil {
		t.Fatal("private host choice escaped into the action scope")
	}
}

func TestRONAttachmentMoveScopesAndDelay(t *testing.T) {
	for _, change := range []string{"current", "returned", "phased", "cycle"} {
		t.Run(change, func(t *testing.T) {
			g, base, other := ronGame()
			a := &ronPermanentChoicePlayer{Player: base, choose: func(candidates []*mage.Permanent) *mage.Permanent { return candidates[0] }}
			b := &ronPermanentChoicePlayer{Player: other, choose: func(candidates []*mage.Permanent) *mage.Permanent { return candidates[len(candidates)-1] }}
			g.SetPlayerAt(0, a)
			g.SetPlayerAt(1, b)
			host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 4), b.PlayerID())
			first := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 4), b.PlayerID())
			second := g.PutOnBattlefield(mage.NewCreature("Second", "", 2, 4), b.PlayerID())
			factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsEnchantmentType(Aura))],ActionList([PlayerActions(ControllerOfPermanent(Ref_TargetPermanent),[CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([AttachPermanentToAPermanent(Ref_TargetPermanent,IsCardtype(Creature))]))])])))`)
			auras := make([]*mage.Permanent, 0, 2)
			for range 2 {
				card := mage.NewAura("Aura", "")
				aura := g.PutOnBattlefield(card, b.PlayerID())
				g.Attach(aura.ID(), host.ID())
				auras = append(auras, aura)
				spell := factory()
				spell.SetOwner(a.PlayerID())
				a.AddToHand(spell)
				if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{aura.ID()}, 0); err != nil {
					t.Fatal(err)
				}
			}
			g.ResolveStack()
			if a.calls != 0 || b.calls != 0 {
				t.Fatal("delayed host chosen early")
			}
			switch change {
			case "returned":
				card := auras[0].Card
				g.RemoveFromBattlefield(auras[0])
				returned := g.PutOnBattlefield(card, b.PlayerID())
				g.Attach(returned.ID(), first.ID())
			case "phased":
				g.PhaseOut(auras[0].ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(auras[0].ID()))
			}
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				if game == clone {
					game.SetPlayerAt(1, &ronPermanentChoicePlayer{Player: clone.GetPlayer(b.PlayerID()), choose: b.choose})
				}
				game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				game.ResolveStack()
				want := second.ID()
				if change == "returned" {
					want = first.ID()
				}
				if change == "phased" {
					want = host.ID()
				}
				if game.FindPermanentIncludingPhased(auras[0].ID()).AttachedTo != want || game.FindPermanent(auras[1].ID()).AttachedTo != second.ID() {
					t.Fatal("delayed action lost exact instance or chooser")
				}
			}
			if a.calls != 0 {
				t.Fatal("effect controller replaced scoped chooser")
			}
		})
	}
}

func TestRONAttachmentMoveKudzuPrerequisite(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		record := chosenDiscardRecord(t, set, "Kudzu")
		if _, err := Compile(record); err != nil {
			t.Fatalf("complete Kudzu record: %v", err)
		}
		move := record.Fields["Rules"].Args[1].Args[1].Args[0].Args[1].Args[1]
		if move.Name != "AttachPermanentToAPermanent" {
			t.Fatal("repository action changed")
		}
		build, err := compileAttachmentMove(move, scope{permanent: true, aura: true})
		if err != nil {
			t.Fatal(err)
		}
		g, a, b := ronGame()
		host := g.PutOnBattlefield(mage.NewLand("Old"), b.PlayerID())
		next := g.PutOnBattlefield(mage.NewLand("Next"), b.PlayerID())
		aura := g.PutOnBattlefield(repositoryCard(t, "LEB", "Wild Growth"), a.PlayerID())
		g.Attach(aura.ID(), host.ID())
		if err := mage.ApplyEffect(g, build(instanceBindings{}), aura.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if aura.AttachedTo != next.ID() {
			t.Fatal("repository movement prerequisite failed")
		}
	}
}

func TestRONAttachmentMovePlayerScope(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsEnchantmentType(Aura))],ActionList([PlayerActions(ControllerOfPermanent(Ref_TargetPermanent),[AttachPermanentToAPermanent(Ref_TargetPermanent,IsCardtype(Creature))])])))`)
	g, base, other := ronGame()
	a := &ronPermanentChoicePlayer{Player: base, choose: func(candidates []*mage.Permanent) *mage.Permanent { return candidates[0] }}
	b := &ronPermanentChoicePlayer{Player: other, choose: func(candidates []*mage.Permanent) *mage.Permanent { return candidates[len(candidates)-1] }}
	g.SetPlayerAt(0, a)
	g.SetPlayerAt(1, b)
	host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 4), b.PlayerID())
	g.PutOnBattlefield(mage.NewCreature("First", "", 2, 4), b.PlayerID())
	last := g.PutOnBattlefield(mage.NewCreature("Last", "", 2, 4), b.PlayerID())
	aura := g.PutOnBattlefield(mage.NewAura("Aura", ""), b.PlayerID())
	g.Attach(aura.ID(), host.ID())
	spell := factory()
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{aura.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.calls != 0 || b.calls != 1 || aura.AttachedTo != last.ID() {
		t.Fatal("scoped chooser was not used")
	}
}
