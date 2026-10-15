package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONExactPlayerSacrificeAnimateDead(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			record := chosenDiscardRecord(t, set, "Animate Dead")
			factory, err := Compile(record)
			if err != nil {
				t.Fatal(err)
			}
			for _, mode := range []string{"cast", "noncast", "aura-control", "guest-control", "late-control", "protection", "shroud", "hexproof", "left", "phased-entry", "cycle-entry", "copy-entry", "phased-guest", "returned-guest", "cycle", "copy-delay", "simultaneous", "waiting-clone"} {
				t.Run(mode, func(t *testing.T) {
					g, a, b := ronGame()
					guest := mage.NewCreature("Guest", "", 3, 4)
					guest.SetOwner(b.PlayerID())
					switch mode {
					case "protection":
						mage.WithAbility(mage.ProtectionFromColors(core.Black))(guest)
					case "shroud":
						mage.WithKeyword(core.Shroud)(guest)
					case "hexproof":
						mage.WithKeyword(core.Hexproof)(guest)
					}
					b.AddToGraveyard(guest)
					card := factory()
					card.SetOwner(a.PlayerID())
					var aura *mage.Permanent
					if mode == "cast" {
						a.AddToHand(card)
						a.ManaPool().Add(core.Black, 1)
						a.ManaPool().Add(core.Colorless, 1)
						if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{guest.ID()}, 0); err != nil {
							t.Fatal(err)
						}
						g.ResolveTopOfStack()
						aura = g.FindPermanent(card.ID())
					} else {
						aura = g.PutOnBattlefield(card, a.PlayerID())
					}
					g.PutTriggersOnStack()
					switch mode {
					case "left":
						g.BouncePermanentToHand(aura)
					case "phased-entry":
						g.PhaseOut(aura.ID())
					case "cycle-entry":
						g.PhaseIn(g.PhaseOut(aura.ID()))
					case "copy-entry":
						if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
							t.Fatal("copy failed")
						}
					}
					g.ResolveStack()
					if mode == "left" || mode == "phased-entry" {
						if g.FindPermanent(guest.ID()) != nil || len(g.DelayedTriggers()) != 0 {
							t.Fatal("entry condition failed")
						}
						return
					}
					if mode == "protection" {
						if g.FindPermanent(card.ID()) != nil || g.FindPermanent(guest.ID()) != nil {
							t.Fatal("illegal attachment did not cause delayed sacrifice")
						}
						return
					}
					p := g.FindPermanent(guest.ID())
					if p == nil || g.FindPermanent(card.ID()).AttachedTo != p.ID() || p.CurrentPower(g) != 2 {
						t.Fatal("entry clauses failed")
					}
					switch mode {
					case "aura-control":
						g.AddControlEffect(mage.ControlEffectSpec{TargetID: card.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					case "guest-control":
						g.AddControlEffect(mage.ControlEffectSpec{TargetID: guest.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					case "cycle":
						g.PhaseIn(g.PhaseOut(p.ID()))
					}
					waiting := g.Clone()
					if mode == "simultaneous" {
						g.DestroyPermanents([]*mage.Permanent{g.FindPermanent(card.ID()), g.FindPermanent(guest.ID())})
					} else {
						g.BouncePermanentToHand(g.FindPermanentIncludingPhased(card.ID()))
					}
					g.PutTriggersOnStack()
					wantTriggers := 1
					if mode == "copy-entry" {
						wantTriggers = 2
					}
					if g.StackSize() != wantTriggers {
						t.Fatalf("departure triggers=%d", g.StackSize())
					}
					switch mode {
					case "phased-guest":
						g.PhaseOut(p.ID())
					case "late-control":
						g.AddControlEffect(mage.ControlEffectSpec{TargetID: guest.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					case "returned-guest":
						g.BouncePermanentToHand(p)
						b.RemoveFromHand(guest.ID())
						g.PutOnBattlefield(guest, b.PlayerID())
					case "copy-delay":
						if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
							t.Fatal("copy failed")
						}
					}
					pending := g.Clone()
					for _, game := range []*mage.Game{g, pending} {
						game.ResolveStack()
						stays := mode == "phased-guest" || mode == "returned-guest"
						if (game.FindPermanentIncludingPhased(guest.ID()) != nil) != stays {
							t.Fatal("delayed sacrifice lost exact visit or current player")
						}
					}
					if mode == "waiting-clone" {
						if waiting.FindPermanent(guest.ID()) == nil || len(waiting.DelayedTriggers()) != 1 {
							t.Fatal("original changed waiting clone")
						}
						waiting.BouncePermanentToHand(waiting.FindPermanent(card.ID()))
						waiting.PutTriggersOnStack()
						waiting.ResolveStack()
						if waiting.FindPermanent(guest.ID()) != nil {
							t.Fatal("waiting clone lost result")
						}
					}
				})
			}
		})
	}
}

func TestRONExactPlayerSacrificeInvalid(t *testing.T) {
	for _, action := range []string{`SacrificePermanent`, `SacrificePermanent()`, `SacrificePermanent(You)`, `SacrificePermanent(ThisPermanent,ThisPermanent)`, `SacrificePermanent(ThisPermanent())`, `SacrificePermanent(ThePermanentPutOnTheBattlefieldThisWay)`} {
		if _, err := Compile(nameScopeNode(t, `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([PlayerAction(You,`+action+`)]))])`)); err == nil {
			t.Fatal("invalid sacrifice accepted", action)
		}
	}
	for _, ctx := range []scope{{}, {enteringSource: true}, {resolving: true, drawReplacementBody: true}} {
		if _, err := compilePlayerEffect(nameScopeNode(t, `SacrificePermanent(ThisPermanent)`), ctx); err == nil {
			t.Fatal("invalid context accepted")
		}
	}
}

func TestRONExactPlayerSacrificePlayerScope(t *testing.T) {
	for _, player := range []string{"You", "ControllerOfPermanent(Ref_TargetPermanent)"} {
		t.Run(player, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([PlayerAction(`+player+`,SacrificePermanent(Ref_TargetPermanent))])))`)
			g, a, b := ronGame()
			guest := g.PutOnBattlefield(mage.NewCreature("Guest", "", 3, 4), b.PlayerID())
			card := factory()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{guest.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if (g.FindPermanent(guest.ID()) != nil) != (player == "You") {
				t.Fatal("explicit player was ignored")
			}
		})
	}
}

func TestRONExactPlayerSacrificeInstances(t *testing.T) {
	factory, err := Compile(chosenDiscardRecord(t, "LEA", "Animate Dead"))
	if err != nil {
		t.Fatal(err)
	}
	g, base, b := ronGame()
	a := &entryAttachmentPlayer{Player: base}
	g.SetPlayerAt(0, a)
	first := mage.NewCreature("First", "", 3, 4)
	first.SetOwner(b.PlayerID())
	b.AddToGraveyard(first)
	one := factory()
	one.SetOwner(a.PlayerID())
	g.PutOnBattlefield(one, a.PlayerID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	second := mage.NewCreature("Second", "", 5, 6)
	second.SetOwner(b.PlayerID())
	b.AddToGraveyard(second)
	two := factory()
	two.SetOwner(a.PlayerID())
	g.PutOnBattlefield(two, a.PlayerID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	clone := g.Clone()
	g.BouncePermanentToHand(g.FindPermanent(one.ID()))
	g.PutTriggersOnStack()
	g.ResolveStack()
	if g.FindPermanent(first.ID()) != nil || g.FindPermanent(second.ID()) == nil {
		t.Fatal("instances shared saved entrant")
	}
	clone.BouncePermanentToHand(clone.FindPermanent(two.ID()))
	clone.PutTriggersOnStack()
	clone.ResolveStack()
	if clone.FindPermanent(first.ID()) == nil || clone.FindPermanent(second.ID()) != nil {
		t.Fatal("clone shared saved entrant")
	}
}
