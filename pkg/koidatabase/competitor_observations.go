package koidatabase

import (
	"net/url"
	"slices"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/koidomain"
)

// observeCompetitors runs under the brand lock after the first conditional cell save.
func observeCompetitors(tx *gorm.DB, run koidomain.VisibilityRun, brand koidomain.Brand, result koidomain.VisibilityResult) error {
	if result.Collector != koidomain.CollectorCloro || result.Outcome == nil || *result.Outcome == koidomain.OutcomeError {
		return nil
	}

	var competitors []koidomain.Competitor
	if err := tx.Where("brand_id = ?", run.BrandID).Order("created_at, id").Find(&competitors).Error; err != nil {
		return err
	}
	var states []koidomain.CompetitorObservation
	if err := tx.Where("brand_id = ?", run.BrandID).Order("created_at, id").Find(&states).Error; err != nil {
		return err
	}

	seen := map[uuid.UUID]bool{}
	for _, observation := range result.Observations {
		if koidomain.ObservationIsBrand(observation, run.BrandSnapshot) || koidomain.ObservationIsBrand(observation, brand.Snapshot()) {
			continue
		}
		index, ambiguous := koidomain.ResolveObservation(observation, competitors)
		if ambiguous {
			continue
		}

		stateIndex := -1
		identities := make([]koidomain.Competitor, len(states))
		for i, state := range states {
			identities[i] = state.Identity
			if state.Key == observation.Key {
				stateIndex = i
			}
		}
		if stateIndex < 0 {
			var tied bool
			stateIndex, tied = koidomain.ResolveObservation(observation, identities)
			if tied {
				continue
			}
		}
		if stateIndex < 0 && index >= 0 {
			continue
		}

		var state koidomain.CompetitorObservation
		if stateIndex >= 0 {
			state = states[stateIndex]
			if seen[state.ID] {
				continue
			}
		} else {
			identity := koidomain.Competitor{
				BrandID:        run.BrandID,
				Name:           observation.Name,
				Source:         koidomain.CompetitorSourceObserved,
				AdditionalURLs: observation.URLs,
			}
			identity.Normalize()
			state = koidomain.CompetitorObservation{BrandID: run.BrandID, Key: observation.Key, Identity: identity}
			if err := tx.Create(&state).Error; err != nil {
				return err
			}
			stateIndex = len(states)
			states = append(states, state)
		}
		seen[state.ID] = true
		state.Count++
		state.Identity.Aliases = append(state.Identity.Aliases, observation.Name)
		state.Identity.AdditionalURLs = slices.Concat(state.Identity.AdditionalURLs, observation.URLs)
		state.Identity.Normalize()
		// Keep observation growth within the same bounds as editable identities.
		state.Identity.Aliases = state.Identity.Aliases[:min(20, len(state.Identity.Aliases))]
		state.Identity.AdditionalURLs = state.Identity.AdditionalURLs[:min(20, len(state.Identity.AdditionalURLs))]
		setObservedPrimaryIdentity(&state.Identity)

		if index >= 0 {
			state.Identity = competitors[index]
		} else if !state.Dismissed && state.Count >= 2 {
			competitor := state.Identity
			competitor.BaseEntity = koidomain.BaseEntity{}
			if err := competitor.Validate(); err != nil {
				return err
			}
			if err := checkCompetitorURLs(tx, competitor); err != nil {
				return err
			}
			if err := tx.Create(&competitor).Error; err != nil {
				return err
			}
			state.Identity = competitor
			competitors = append(competitors, competitor)
		}

		if err := tx.Select("count", "identity").Updates(&state).Error; err != nil {
			return err
		}
		states[stateIndex] = state
	}
	return nil
}

func setObservedPrimaryIdentity(competitor *koidomain.Competitor) {
	for _, value := range competitor.AdditionalURLs {
		u, err := url.Parse(value)
		if err != nil {
			continue
		}
		if u.Host == "github.com" && u.Path != "" {
			if competitor.GitHubRepo == "" {
				competitor.GitHubRepo = value
			}
		} else if competitor.Domain == "" {
			competitor.Domain = u.Host
		}
	}
}
