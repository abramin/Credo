package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	consentpb "github.com/credo/gateway/api/proto/consent"
	commonpb "github.com/credo/gateway/api/proto/common"
	"github.com/credo/gateway/internal/consent"
	"github.com/credo/gateway/pkg/errors"
)

// Server is a gRPC adapter that exposes consent.Service over gRPC
// This is the hexagonal architecture "adapter" - it translates between
// protobuf types and domain models, and handles gRPC-specific concerns.
type Server struct {
	consentpb.UnimplementedConsentServiceServer
	service *consent.Service
}

// NewServer creates a new gRPC consent server
func NewServer(service *consent.Service) *Server {
	return &Server{
		service: service,
	}
}

// HasConsent checks if user has valid consent for a purpose
func (s *Server) HasConsent(ctx context.Context, req *consentpb.HasConsentRequest) (*consentpb.HasConsentResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	purpose := mapProtoPurposeToDomain(req.Purpose)
	if purpose == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid purpose")
	}

	// Call domain service
	record, err := s.service.Find(ctx, req.UserId, consent.Purpose(purpose))
	if err != nil {
		if errors.IsNotFound(err) {
			return &consentpb.HasConsentResponse{
				HasConsent: false,
				Status:     consentpb.ConsentStatus_CONSENT_STATUS_EXPIRED,
			}, nil
		}
		return nil, mapDomainErrorToGRPC(err)
	}

	// Check if consent is active
	hasConsent := record.IsActive(s.service.Now())
	status := mapConsentStatusToProto(record)

	return &consentpb.HasConsentResponse{
		HasConsent: hasConsent,
		Status:     status,
		GrantedAt:  timestamppb.New(record.GrantedAt),
		ExpiresAt:  timestamppb.New(*record.ExpiresAt),
	}, nil
}

// RequireConsent enforces consent requirement
func (s *Server) RequireConsent(ctx context.Context, req *consentpb.RequireConsentRequest) (*consentpb.RequireConsentResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	purpose := mapProtoPurposeToDomain(req.Purpose)
	if purpose == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid purpose")
	}

	// Call domain service
	err := s.service.Require(ctx, req.UserId, consent.Purpose(purpose))
	if err != nil {
		// Map consent errors to response
		if errors.IsMissingConsent(err) || errors.IsInvalidConsent(err) {
			return &consentpb.RequireConsentResponse{
				Allowed: false,
				Reason:  err.Error(),
			}, nil
		}
		return nil, mapDomainErrorToGRPC(err)
	}

	return &consentpb.RequireConsentResponse{
		Allowed: true,
		Reason:  "",
	}, nil
}

// GrantConsent grants consent for purposes
func (s *Server) GrantConsent(ctx context.Context, req *consentpb.GrantConsentRequest) (*consentpb.GrantConsentResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if len(req.Purposes) == 0 {
		return nil, status.Error(codes.InvalidArgument, "purposes is required")
	}

	// Map proto purposes to domain
	purposes := make([]consent.Purpose, 0, len(req.Purposes))
	for _, p := range req.Purposes {
		purpose := mapProtoPurposeToDomain(p)
		if purpose == "" {
			return nil, status.Error(codes.InvalidArgument, "invalid purpose")
		}
		purposes = append(purposes, consent.Purpose(purpose))
	}

	// Call domain service
	records, err := s.service.Grant(ctx, req.UserId, purposes)
	if err != nil {
		return nil, mapDomainErrorToGRPC(err)
	}

	// Map domain records to proto
	protoRecords := make([]*consentpb.ConsentRecord, len(records))
	for i, r := range records {
		protoRecords[i] = mapDomainRecordToProto(r)
	}

	return &consentpb.GrantConsentResponse{
		Granted: protoRecords,
	}, nil
}

// RevokeConsent revokes consent for purposes
func (s *Server) RevokeConsent(ctx context.Context, req *consentpb.RevokeConsentRequest) (*consentpb.RevokeConsentResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if len(req.Purposes) == 0 {
		return nil, status.Error(codes.InvalidArgument, "purposes is required")
	}

	// Map proto purposes to domain
	purposes := make([]consent.Purpose, 0, len(req.Purposes))
	for _, p := range req.Purposes {
		purpose := mapProtoPurposeToDomain(p)
		if purpose == "" {
			return nil, status.Error(codes.InvalidArgument, "invalid purpose")
		}
		purposes = append(purposes, consent.Purpose(purpose))
	}

	// Call domain service
	records, err := s.service.Revoke(ctx, req.UserId, purposes)
	if err != nil {
		return nil, mapDomainErrorToGRPC(err)
	}

	// Map domain records to proto
	protoRecords := make([]*consentpb.ConsentRecord, len(records))
	for i, r := range records {
		protoRecords[i] = mapDomainRecordToProto(r)
	}

	return &consentpb.RevokeConsentResponse{
		Revoked: protoRecords,
	}, nil
}

// ListConsents lists all consents for a user
func (s *Server) ListConsents(ctx context.Context, req *consentpb.ListConsentsRequest) (*consentpb.ListConsentsResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	// Call domain service
	records, err := s.service.List(ctx, req.UserId, nil)
	if err != nil {
		return nil, mapDomainErrorToGRPC(err)
	}

	// Map domain records to proto
	protoRecords := make([]*consentpb.ConsentRecord, len(records))
	for i, r := range records {
		protoRecords[i] = mapDomainRecordToProto(r)
	}

	return &consentpb.ListConsentsResponse{
		Consents: protoRecords,
	}, nil
}

// Check implements health check
func (s *Server) Check(ctx context.Context, req *commonpb.HealthCheckRequest) (*commonpb.HealthCheckResponse, error) {
	return &commonpb.HealthCheckResponse{
		Status: commonpb.HealthCheckResponse_SERVING,
	}, nil
}

// Helper functions for mapping between proto and domain types

func mapProtoPurposeToDomain(p consentpb.Purpose) string {
	switch p {
	case consentpb.Purpose_PURPOSE_LOGIN:
		return "login"
	case consentpb.Purpose_PURPOSE_REGISTRY_CHECK:
		return "registry_check"
	case consentpb.Purpose_PURPOSE_VC_ISSUANCE:
		return "vc_issuance"
	case consentpb.Purpose_PURPOSE_DECISION_EVALUATION:
		return "decision_evaluation"
	case consentpb.Purpose_PURPOSE_BIOMETRIC_VERIFICATION:
		return "biometric_verification"
	default:
		return ""
	}
}

func mapDomainPurposeToProto(p consent.Purpose) consentpb.Purpose {
	switch p {
	case "login":
		return consentpb.Purpose_PURPOSE_LOGIN
	case "registry_check":
		return consentpb.Purpose_PURPOSE_REGISTRY_CHECK
	case "vc_issuance":
		return consentpb.Purpose_PURPOSE_VC_ISSUANCE
	case "decision_evaluation":
		return consentpb.Purpose_PURPOSE_DECISION_EVALUATION
	case "biometric_verification":
		return consentpb.Purpose_PURPOSE_BIOMETRIC_VERIFICATION
	default:
		return consentpb.Purpose_PURPOSE_UNSPECIFIED
	}
}

func mapConsentStatusToProto(record *consent.Record) consentpb.ConsentStatus {
	if record.RevokedAt != nil {
		return consentpb.ConsentStatus_CONSENT_STATUS_REVOKED
	}
	// Note: IsActive check would be done by caller
	return consentpb.ConsentStatus_CONSENT_STATUS_ACTIVE
}

func mapDomainRecordToProto(r *consent.Record) *consentpb.ConsentRecord {
	rec := &consentpb.ConsentRecord{
		Id:        r.ID,
		UserId:    r.UserID,
		Purpose:   mapDomainPurposeToProto(r.Purpose),
		GrantedAt: timestamppb.New(r.GrantedAt),
	}

	if r.ExpiresAt != nil {
		rec.ExpiresAt = timestamppb.New(*r.ExpiresAt)
	}
	if r.RevokedAt != nil {
		rec.RevokedAt = timestamppb.New(*r.RevokedAt)
		rec.Status = consentpb.ConsentStatus_CONSENT_STATUS_REVOKED
	} else {
		rec.Status = consentpb.ConsentStatus_CONSENT_STATUS_ACTIVE
	}

	return rec
}

func mapDomainErrorToGRPC(err error) error {
	code := errors.GetCode(err)
	switch code {
	case errors.CodeNotFound:
		return status.Error(codes.NotFound, err.Error())
	case errors.CodeInvalidArgument:
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.CodeMissingConsent, errors.CodeInvalidConsent:
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.CodeInternal:
		return status.Error(codes.Internal, "internal server error")
	default:
		return status.Error(codes.Unknown, err.Error())
	}
}
