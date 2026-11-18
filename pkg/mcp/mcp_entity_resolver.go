package mcp

import (
	"strings"

	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// mcpEntityResolver resolves an entity from the given candidates by stable ID or by name
type mcpEntityResolver[T any] struct {
	entityName    string
	getId         func(T) int64
	getName       func(T) string
	describe      func(T) string
	maxCandidates int
}

// newMCPEntityResolver returns a new entity resolver
func newMCPEntityResolver[T any](entityName string, getId func(T) int64, getName func(T) string, describe func(T) string) *mcpEntityResolver[T] {
	return &mcpEntityResolver[T]{
		entityName:    entityName,
		getId:         getId,
		getName:       getName,
		describe:      describe,
		maxCandidates: 20,
	}
}

// Resolve returns the only entity which matches the textual id (preferred) or the name.
// If the id is set, the entity must exist in items, and the name (if also set) must match the entity name.
// If only the name is set, the exact matches are preferred, otherwise case-insensitive (and space-trimmed) matches are used.
// Returns a "not found" error if nothing matches, and an "ambiguous name" error with all candidates if more than one entity matches.
func (r *mcpEntityResolver[T]) Resolve(items []T, idField string, id string, nameField string, name string) (T, error) {
	var empty T
	id = strings.TrimSpace(id)

	if id != "" {
		entityId, err := parseMCPOptionalId(idField, id)

		if err != nil {
			return empty, err
		}

		for i := 0; i < len(items); i++ {
			if r.getId(items[i]) == entityId {
				if name != "" && r.getName(items[i]) != name && !strings.EqualFold(strings.TrimSpace(r.getName(items[i])), strings.TrimSpace(name)) {
					return empty, NewMCPInvalidFieldError(nameField, name, "\""+nameField+"\" does not match the name of "+r.entityName+" \""+id+"\" (\""+r.getName(items[i])+"\")")
				}

				return items[i], nil
			}
		}

		return empty, NewMCPNotFoundError(idField, r.entityName, id)
	}

	if name == "" {
		return empty, NewMCPRequiredFieldError(idField + " or " + nameField)
	}

	matches := r.FindAllByName(items, name)

	if len(matches) == 1 {
		return matches[0], nil
	} else if len(matches) > 1 {
		return empty, NewMCPAmbiguousNameError(nameField, r.entityName, name, r.GetCandidates(matches))
	}

	return empty, NewMCPNotFoundError(nameField, r.entityName, name)
}

// FindAllByName returns all entities whose name exactly matches the given name, or (if no exact match) whose name matches case-insensitively after trimming spaces
func (r *mcpEntityResolver[T]) FindAllByName(items []T, name string) []T {
	exactMatches := make([]T, 0)
	looseMatches := make([]T, 0)
	trimmedName := strings.TrimSpace(name)

	for i := 0; i < len(items); i++ {
		itemName := r.getName(items[i])

		if itemName == name {
			exactMatches = append(exactMatches, items[i])
		} else if strings.EqualFold(strings.TrimSpace(itemName), trimmedName) {
			looseMatches = append(looseMatches, items[i])
		}
	}

	if len(exactMatches) > 0 {
		return exactMatches
	}

	return looseMatches
}

// GetCandidates returns the candidate list of the given entities
func (r *mcpEntityResolver[T]) GetCandidates(items []T) []*MCPEntityCandidate {
	count := len(items)

	if r.maxCandidates > 0 && count > r.maxCandidates {
		count = r.maxCandidates
	}

	candidates := make([]*MCPEntityCandidate, 0, count)

	for i := 0; i < count; i++ {
		candidate := &MCPEntityCandidate{
			Id:   utils.Int64ToString(r.getId(items[i])),
			Name: r.getName(items[i]),
		}

		if r.describe != nil {
			candidate.Description = r.describe(items[i])
		}

		candidates = append(candidates, candidate)
	}

	return candidates
}
