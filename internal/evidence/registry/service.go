package registry

import (
	"context"
	"errors"
)

// Service coordinates registry lookups with caching and optional minimisation.
type Service struct {
	citizens  CitizenClient
	sanctions SanctionsClient
	cache     RegistryCacheStore
	regulated bool
}

func NewService(citizens CitizenClient, sanctions SanctionsClient, cache RegistryCacheStore, regulated bool) *Service {
	return &Service{
		citizens:  citizens,
		sanctions: sanctions,
		cache:     cache,
		regulated: regulated,
	}
}

func (s *Service) Check(ctx context.Context, nationalID string) (RegistryResult, error) {
	citizen, err := s.Citizen(ctx, nationalID)
	if err != nil {
		return RegistryResult{}, err
	}
	sanctions, err := s.Sanctions(ctx, nationalID)
	if err != nil {
		return RegistryResult{}, err
	}
	return RegistryResult{
		Citizen:  citizen,
		Sanction: sanctions,
	}, nil
}

func (s *Service) Citizen(ctx context.Context, nationalID string) (CitizenRecord, error) {
	if s.cache != nil {
		if cached, err := s.cache.FindCitizen(ctx, nationalID); err == nil {
			return cached, nil
		} else if !errors.Is(err, ErrNotFound) {
			return CitizenRecord{}, err
		}
	}
	record, err := s.citizens.Lookup(ctx, nationalID)
	if err != nil {
		return CitizenRecord{}, err
	}
	if s.regulated {
		record = MinimizeCitizenRecord(record)
	}
	if s.cache != nil {
		_ = s.cache.SaveCitizen(ctx, record)
	}
	return record, nil
}

func (s *Service) Sanctions(ctx context.Context, nationalID string) (SanctionsRecord, error) {
	if s.cache != nil {
		if cached, err := s.cache.FindSanction(ctx, nationalID); err == nil {
			return cached, nil
		} else if !errors.Is(err, ErrNotFound) {
			return SanctionsRecord{}, err
		}
	}
	record, err := s.sanctions.Check(ctx, nationalID)
	if err != nil {
		return SanctionsRecord{}, err
	}
	if s.cache != nil {
		_ = s.cache.SaveSanction(ctx, record)
	}
	return record, nil
}
