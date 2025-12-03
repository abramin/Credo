package domain

// RegistryOrchestrator coordinates registry calls and caches outcomes. Concrete
// clients are provided by the registry package and injected by services.
type RegistryOrchestrator struct {
	RegulatedMode bool
}

func NewRegistryOrchestrator(regulated bool) *RegistryOrchestrator {
	return &RegistryOrchestrator{RegulatedMode: regulated}
}

type CitizenCheckInput struct {
	NationalID string
}

type CitizenRecord struct {
	NationalID  string // PII
	FullName    string // PII
	DateOfBirth string // PII; retained only when needed
	Valid       bool   // Derived flag
}

type SanctionsCheckInput struct {
	NationalID string
}

type SanctionsRecord struct {
	NationalID string
	Listed     bool
	Source     string
}

type RegistryResult struct {
	Citizen  CitizenRecord
	Sanction SanctionsRecord
}

func (o *RegistryOrchestrator) CheckCitizen(input CitizenCheckInput) (CitizenRecord, error) {
	record := CitizenRecord{
		NationalID:  input.NationalID,
		FullName:    "TODO Citizen",
		DateOfBirth: "1980-01-01",
		Valid:       true,
	}
	if o.RegulatedMode {
		return MinimizeCitizenRecord(record), nil
	}
	return record, nil
}

func (o *RegistryOrchestrator) CheckSanctions(input SanctionsCheckInput) (SanctionsRecord, error) {
	return SanctionsRecord{
		NationalID: input.NationalID,
		Listed:     false,
		Source:     "mock",
	}, nil
}

// MinimizeCitizenRecord strips PII when regulated mode is enabled.
func MinimizeCitizenRecord(record CitizenRecord) CitizenRecord {
	return CitizenRecord{
		NationalID:  "", // drop identifiers
		FullName:    "",
		DateOfBirth: "",
		Valid:       record.Valid,
	}
}
