package grpc

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	consentpb "github.com/credo/gateway/api/proto/consent"
	commonpb "github.com/credo/gateway/api/proto/common"
	"github.com/credo/gateway/internal/registry/ports"
	"github.com/credo/gateway/pkg/errors"
)

// ConsentClient is a gRPC adapter that implements ports.ConsentPort
// It wraps the generated gRPC client and translates between domain and protobuf types.
//
// This keeps the registry service independent of gRPC implementation details.
type ConsentClient struct {
	client  consentpb.ConsentServiceClient
	timeout time.Duration
}

// NewConsentClient creates a new gRPC consent client
func NewConsentClient(addr string, timeout time.Duration) (*ConsentClient, error) {
	// TODO: In production, use TLS credentials and service discovery
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to consent service: %w", err)
	}

	return &ConsentClient{
		client:  consentpb.NewConsentServiceClient(conn),
		timeout: timeout,
	}, nil
}

// HasConsent checks if user has active consent for a purpose
// Implements ports.ConsentPort
func (c *ConsentClient) HasConsent(ctx context.Context, userID string, purpose string) (bool, error) {
	// Create request context with timeout
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Add request metadata (request ID, tracing, etc.)
	ctx = c.addMetadata(ctx)

	// Map purpose to proto enum
	protoPurpose := mapDomainPurposeToProto(purpose)
	if protoPurpose == consentpb.Purpose_PURPOSE_UNSPECIFIED {
		return false, errors.NewGatewayError(errors.CodeInvalidArgument,
			fmt.Sprintf("invalid purpose: %s", purpose), nil)
	}

	// Call gRPC
	resp, err := c.client.HasConsent(ctx, &consentpb.HasConsentRequest{
		Metadata: c.buildMetadata(ctx),
		UserId:   userID,
		Purpose:  protoPurpose,
	})
	if err != nil {
		return false, c.mapGRPCError(err)
	}

	return resp.HasConsent, nil
}

// RequireConsent enforces consent requirement
// Implements ports.ConsentPort
func (c *ConsentClient) RequireConsent(ctx context.Context, userID string, purpose string) error {
	// Create request context with timeout
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Add request metadata
	ctx = c.addMetadata(ctx)

	// Map purpose to proto enum
	protoPurpose := mapDomainPurposeToProto(purpose)
	if protoPurpose == consentpb.Purpose_PURPOSE_UNSPECIFIED {
		return errors.NewGatewayError(errors.CodeInvalidArgument,
			fmt.Sprintf("invalid purpose: %s", purpose), nil)
	}

	// Call gRPC
	resp, err := c.client.RequireConsent(ctx, &consentpb.RequireConsentRequest{
		Metadata: c.buildMetadata(ctx),
		UserId:   userID,
		Purpose:  protoPurpose,
	})
	if err != nil {
		return c.mapGRPCError(err)
	}

	// Check if consent is allowed
	if !resp.Allowed {
		return errors.NewGatewayError(errors.CodeMissingConsent,
			resp.Reason, nil)
	}

	return nil
}

// Helper functions

func (c *ConsentClient) addMetadata(ctx context.Context) context.Context {
	// Extract request ID from context if present
	requestID, ok := ctx.Value("request_id").(string)
	if !ok {
		requestID = "unknown"
	}

	// Add metadata to outgoing context
	md := metadata.Pairs(
		"request-id", requestID,
		"timestamp", time.Now().Format(time.RFC3339),
	)
	return metadata.NewOutgoingContext(ctx, md)
}

func (c *ConsentClient) buildMetadata(ctx context.Context) *commonpb.RequestMetadata {
	requestID, _ := ctx.Value("request_id").(string)
	userID, _ := ctx.Value("user_id").(string)
	sessionID, _ := ctx.Value("session_id").(string)
	clientID, _ := ctx.Value("client_id").(string)

	return &commonpb.RequestMetadata{
		RequestId: requestID,
		UserId:    userID,
		SessionId: sessionID,
		ClientId:  clientID,
		Timestamp: timestamppb.Now(),
	}
}

func mapDomainPurposeToProto(purpose string) consentpb.Purpose {
	switch purpose {
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

func (c *ConsentClient) mapGRPCError(err error) error {
	st, ok := status.FromError(err)
	if !ok {
		return errors.NewGatewayError(errors.CodeInternal,
			"internal error", err)
	}

	switch st.Code() {
	case codes.InvalidArgument:
		return errors.NewGatewayError(errors.CodeInvalidArgument,
			st.Message(), err)
	case codes.NotFound:
		return errors.NewGatewayError(errors.CodeNotFound,
			st.Message(), err)
	case codes.PermissionDenied:
		return errors.NewGatewayError(errors.CodeMissingConsent,
			st.Message(), err)
	case codes.DeadlineExceeded:
		return errors.NewGatewayError(errors.CodeInternal,
			"consent service timeout", err)
	case codes.Unavailable:
		return errors.NewGatewayError(errors.CodeInternal,
			"consent service unavailable", err)
	default:
		return errors.NewGatewayError(errors.CodeInternal,
			st.Message(), err)
	}
}

// Ensure ConsentClient implements ports.ConsentPort
var _ ports.ConsentPort = (*ConsentClient)(nil)
