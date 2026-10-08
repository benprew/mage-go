package mage

import (
	"fmt"
	"slices"

	"github.com/google/uuid"
)

type commandObject struct {
	card     Card
	visit    uuid.UUID
	faceDown bool
}

// AddCommandObject installs an object with a new command-zone visit identity.
// The caller must remove the object from its former zone before installation.
// This service does not grant casting permission or install command abilities.
func (g *Game) AddCommandObject(card Card, faceDown bool) (uuid.UUID, error) {
	if nilDefinition(card) || card.ID() == uuid.Nil {
		return uuid.Nil, fmt.Errorf("command object requires a card with an ID")
	}
	for _, object := range g.zones.commandObjects {
		if object.card.ID() == card.ID() {
			return uuid.Nil, fmt.Errorf("object %s is already in the command zone", card.ID())
		}
	}
	visit := uuid.New()
	copied := card.Copy()
	copied.SetID(card.ID())
	g.zones.commandObjects = append(g.zones.commandObjects, commandObject{card: copied, visit: visit, faceDown: faceDown})
	return visit, nil
}

// RemoveCommandObject removes only the specified command-zone visit.
func (g *Game) RemoveCommandObject(visit uuid.UUID) (Card, bool) {
	for i, object := range g.zones.commandObjects {
		if object.visit == visit {
			if g.zones.departedCommandSources == nil {
				g.zones.departedCommandSources = make(map[uuid.UUID]DamageSource)
			}
			g.zones.departedCommandSources[visit] = damageSourceFromCommandObject(object)
			g.zones.commandObjects = slices.Delete(g.zones.commandObjects, i, i+1)
			return object.card, true
		}
	}
	return nil, false
}

// SetCommandObjectFaceDown changes visibility without changing visit identity.
func (g *Game) SetCommandObjectFaceDown(visit uuid.UUID, faceDown bool) bool {
	for i := range g.zones.commandObjects {
		if g.zones.commandObjects[i].visit == visit {
			g.zones.commandObjects[i].faceDown = faceDown
			return true
		}
	}
	return false
}

func damageSourceFromCommandObject(object commandObject) DamageSource {
	source := DamageSource{id: object.card.ID(), commandVisit: object.visit, owner: object.card.Owner()}
	if object.faceDown {
		return source
	}
	source.card = object.card.Copy()
	source.card.SetID(object.card.ID())
	source.name = object.card.Name()
	source.colors = slices.Clone(cardColors(object.card))
	return source
}

// CommandDamageSources reads face-up command objects in insertion order.
// This category alone cannot supply the complete CR 609.7a source list.
func (g *Game) CommandDamageSources() []DamageSource {
	var sources []DamageSource
	for _, object := range g.zones.commandObjects {
		if !object.faceDown {
			sources = append(sources, damageSourceFromCommandObject(object))
		}
	}
	return sources
}

// ChooseDamageSourceFromCommandObjects chooses from face-up command objects.
// The choice uses a separate result binding for each ability instance.
func ChooseDamageSourceFromCommandObjects(player PlayerRef, filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect {
	return &damageSourceChoice{player: player, filter: filter, binding: binding, commandObjects: true}
}

func (g *Game) commandReferenceInformation(saved DamageSource) DamageSource {
	for _, object := range g.zones.commandObjects {
		if object.visit == saved.CommandVisit() && object.card.ID() == saved.ID() {
			return damageSourceFromCommandObject(object)
		}
	}
	if source, ok := g.zones.departedCommandSources[saved.CommandVisit()]; ok && source.SameObject(saved) {
		return cloneDamageSource(source)
	}
	return cloneDamageSource(saved)
}
