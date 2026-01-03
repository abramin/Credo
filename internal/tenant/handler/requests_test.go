package handler

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"credo/pkg/platform/validation"
)

// CreateClientRequestSuite tests CreateClientRequest validation and normalization.
type CreateClientRequestSuite struct {
	suite.Suite
}

func TestCreateClientRequestSuite(t *testing.T) {
	suite.Run(t, new(CreateClientRequestSuite))
}

func (s *CreateClientRequestSuite) validRequest() *CreateClientRequest {
	return &CreateClientRequest{
		TenantID:      "550e8400-e29b-41d4-a716-446655440000",
		Name:          "Test Client",
		RedirectURIs:  []string{"https://example.com/callback"},
		AllowedGrants: []string{"authorization_code"},
		AllowedScopes: []string{"openid", "profile"},
	}
}

// TestValidation verifies size limit enforcement on CreateClientRequest.
func (s *CreateClientRequestSuite) TestValidation() {
	s.Run("valid request passes", func() {
		req := s.validRequest()
		err := req.Validate()
		s.NoError(err)
	})

	s.Run("too many redirect URIs rejected", func() {
		req := s.validRequest()
		req.RedirectURIs = make([]string, validation.MaxRedirectURIs+1)
		for i := range req.RedirectURIs {
			req.RedirectURIs[i] = "https://example.com/callback"
		}

		err := req.Validate()
		s.Require().Error(err)
		s.Contains(err.Error(), "too many redirect URIs")
	})

	s.Run("max redirect URIs allowed", func() {
		req := s.validRequest()
		req.RedirectURIs = make([]string, validation.MaxRedirectURIs)
		for i := range req.RedirectURIs {
			req.RedirectURIs[i] = "https://example.com/callback"
		}

		err := req.Validate()
		s.NoError(err)
	})

	s.Run("too many grant types rejected", func() {
		req := s.validRequest()
		req.AllowedGrants = make([]string, validation.MaxGrants+1)
		for i := range req.AllowedGrants {
			req.AllowedGrants[i] = "authorization_code"
		}

		err := req.Validate()
		s.Require().Error(err)
		s.Contains(err.Error(), "too many grant types")
	})

	s.Run("max grant types allowed", func() {
		req := s.validRequest()
		req.AllowedGrants = make([]string, validation.MaxGrants)
		for i := range req.AllowedGrants {
			req.AllowedGrants[i] = "authorization_code"
		}

		err := req.Validate()
		s.NoError(err)
	})

	s.Run("too many scopes rejected", func() {
		req := s.validRequest()
		req.AllowedScopes = make([]string, validation.MaxScopes+1)
		for i := range req.AllowedScopes {
			req.AllowedScopes[i] = "scope"
		}

		err := req.Validate()
		s.Require().Error(err)
		s.Contains(err.Error(), "too many scopes")
	})

	s.Run("max scopes allowed", func() {
		req := s.validRequest()
		req.AllowedScopes = make([]string, validation.MaxScopes)
		for i := range req.AllowedScopes {
			req.AllowedScopes[i] = "scope"
		}

		err := req.Validate()
		s.NoError(err)
	})

	s.Run("redirect URI exceeds max length rejected", func() {
		req := s.validRequest()
		req.RedirectURIs = []string{"https://example.com/" + strings.Repeat("a", validation.MaxRedirectURILength)}

		err := req.Validate()
		s.Require().Error(err)
		s.Contains(err.Error(), "redirect URI exceeds max length")
	})

	s.Run("redirect URI at max length allowed", func() {
		req := s.validRequest()
		baseURL := "https://example.com/"
		padding := strings.Repeat("a", validation.MaxRedirectURILength-len(baseURL))
		req.RedirectURIs = []string{baseURL + padding}

		err := req.Validate()
		s.NoError(err)
	})

	s.Run("scope exceeds max length rejected", func() {
		req := s.validRequest()
		req.AllowedScopes = []string{strings.Repeat("a", validation.MaxScopeLength+1)}

		err := req.Validate()
		s.Require().Error(err)
		s.Contains(err.Error(), "scope exceeds max length")
	})

	s.Run("scope at max length allowed", func() {
		req := s.validRequest()
		req.AllowedScopes = []string{strings.Repeat("a", validation.MaxScopeLength)}

		err := req.Validate()
		s.NoError(err)
	})
}

// TestRequiredFields verifies required field enforcement.
func (s *CreateClientRequestSuite) TestRequiredFields() {
	s.Run("missing name rejected", func() {
		req := &CreateClientRequest{
			TenantID:      "550e8400-e29b-41d4-a716-446655440000",
			RedirectURIs:  []string{"https://example.com/callback"},
			AllowedGrants: []string{"authorization_code"},
			AllowedScopes: []string{"openid"},
		}

		err := req.Validate()
		s.Require().Error(err)
		s.Contains(err.Error(), "name is required")
	})

	s.Run("missing redirect_uris rejected", func() {
		req := &CreateClientRequest{
			TenantID:      "550e8400-e29b-41d4-a716-446655440000",
			Name:          "Test Client",
			AllowedGrants: []string{"authorization_code"},
			AllowedScopes: []string{"openid"},
		}

		err := req.Validate()
		s.Require().Error(err)
		s.Contains(err.Error(), "at least one redirect_uri is required")
	})

	s.Run("nil request rejected", func() {
		var req *CreateClientRequest
		err := req.Validate()
		s.Require().Error(err)
		s.Contains(err.Error(), "request is required")
	})
}

// TestNormalize verifies input normalization.
func (s *CreateClientRequestSuite) TestNormalize() {
	s.Run("trims whitespace and deduplicates", func() {
		req := &CreateClientRequest{
			Name:          "  Test Client  ",
			RedirectURIs:  []string{"  https://example.com/callback  ", "https://example.com/callback"},
			AllowedGrants: []string{"  AUTHORIZATION_CODE  ", "authorization_code"},
			AllowedScopes: []string{"  openid  ", "openid"},
		}

		req.Normalize()

		s.Equal("Test Client", req.Name)
		s.Len(req.RedirectURIs, 1)
		s.Equal("https://example.com/callback", req.RedirectURIs[0])
		s.Len(req.AllowedGrants, 1)
		s.Equal("authorization_code", req.AllowedGrants[0])
		s.Len(req.AllowedScopes, 1)
		s.Equal("openid", req.AllowedScopes[0])
	})

	s.Run("nil request does not panic", func() {
		var req *CreateClientRequest
		s.NotPanics(func() { req.Normalize() })
	})
}

// UpdateClientRequestSuite tests UpdateClientRequest validation and normalization.
type UpdateClientRequestSuite struct {
	suite.Suite
}

func TestUpdateClientRequestSuite(t *testing.T) {
	suite.Run(t, new(UpdateClientRequestSuite))
}

// TestValidation verifies size limit enforcement on UpdateClientRequest.
func (s *UpdateClientRequestSuite) TestValidation() {
	s.Run("empty request is valid", func() {
		req := &UpdateClientRequest{}
		err := req.Validate()
		s.NoError(err)
	})

	s.Run("too many redirect URIs rejected", func() {
		uris := make([]string, validation.MaxRedirectURIs+1)
		for i := range uris {
			uris[i] = "https://example.com/callback"
		}
		req := &UpdateClientRequest{RedirectURIs: &uris}

		err := req.Validate()
		s.Require().Error(err)
		s.Contains(err.Error(), "too many redirect URIs")
	})

	s.Run("max redirect URIs allowed", func() {
		uris := make([]string, validation.MaxRedirectURIs)
		for i := range uris {
			uris[i] = "https://example.com/callback"
		}
		req := &UpdateClientRequest{RedirectURIs: &uris}

		err := req.Validate()
		s.NoError(err)
	})

	s.Run("too many grant types rejected", func() {
		grants := make([]string, validation.MaxGrants+1)
		for i := range grants {
			grants[i] = "authorization_code"
		}
		req := &UpdateClientRequest{AllowedGrants: &grants}

		err := req.Validate()
		s.Require().Error(err)
		s.Contains(err.Error(), "too many grant types")
	})

	s.Run("too many scopes rejected", func() {
		scopes := make([]string, validation.MaxScopes+1)
		for i := range scopes {
			scopes[i] = "scope"
		}
		req := &UpdateClientRequest{AllowedScopes: &scopes}

		err := req.Validate()
		s.Require().Error(err)
		s.Contains(err.Error(), "too many scopes")
	})

	s.Run("redirect URI exceeds max length rejected", func() {
		uris := []string{"https://example.com/" + strings.Repeat("a", validation.MaxRedirectURILength)}
		req := &UpdateClientRequest{RedirectURIs: &uris}

		err := req.Validate()
		s.Require().Error(err)
		s.Contains(err.Error(), "redirect URI exceeds max length")
	})

	s.Run("scope exceeds max length rejected", func() {
		scopes := []string{strings.Repeat("a", validation.MaxScopeLength+1)}
		req := &UpdateClientRequest{AllowedScopes: &scopes}

		err := req.Validate()
		s.Require().Error(err)
		s.Contains(err.Error(), "scope exceeds max length")
	})

	s.Run("nil request rejected", func() {
		var req *UpdateClientRequest
		err := req.Validate()
		s.Require().Error(err)
		s.Contains(err.Error(), "request is required")
	})
}

// TestNormalize verifies input normalization.
func (s *UpdateClientRequestSuite) TestNormalize() {
	s.Run("trims and deduplicates pointer fields", func() {
		name := "  Test Client  "
		uris := []string{"  https://example.com/callback  ", "https://example.com/callback"}
		grants := []string{"  AUTHORIZATION_CODE  ", "authorization_code"}
		scopes := []string{"  openid  ", "openid"}

		req := &UpdateClientRequest{
			Name:          &name,
			RedirectURIs:  &uris,
			AllowedGrants: &grants,
			AllowedScopes: &scopes,
		}

		req.Normalize()

		s.Equal("Test Client", *req.Name)
		s.Len(*req.RedirectURIs, 1)
		s.Equal("https://example.com/callback", (*req.RedirectURIs)[0])
		s.Len(*req.AllowedGrants, 1)
		s.Equal("authorization_code", (*req.AllowedGrants)[0])
		s.Len(*req.AllowedScopes, 1)
		s.Equal("openid", (*req.AllowedScopes)[0])
	})

	s.Run("nil request does not panic", func() {
		var req *UpdateClientRequest
		s.NotPanics(func() { req.Normalize() })
	})

	s.Run("nil fields do not cause panic", func() {
		req := &UpdateClientRequest{}
		s.NotPanics(func() { req.Normalize() })
	})
}

// CreateTenantRequestSuite tests CreateTenantRequest validation and normalization.
type CreateTenantRequestSuite struct {
	suite.Suite
}

func TestCreateTenantRequestSuite(t *testing.T) {
	suite.Run(t, new(CreateTenantRequestSuite))
}

// TestValidation verifies required field enforcement.
func (s *CreateTenantRequestSuite) TestValidation() {
	s.Run("valid request passes", func() {
		req := &CreateTenantRequest{Name: "Test Tenant"}
		err := req.Validate()
		s.NoError(err)
	})

	s.Run("missing name rejected", func() {
		req := &CreateTenantRequest{}
		err := req.Validate()
		s.Require().Error(err)
		s.Contains(err.Error(), "name is required")
	})

	s.Run("nil request rejected", func() {
		var req *CreateTenantRequest
		err := req.Validate()
		s.Require().Error(err)
		s.Contains(err.Error(), "request is required")
	})
}

// TestNormalize verifies input normalization.
func (s *CreateTenantRequestSuite) TestNormalize() {
	s.Run("trims whitespace", func() {
		req := &CreateTenantRequest{Name: "  Test Tenant  "}
		req.Normalize()
		s.Equal("Test Tenant", req.Name)
	})

	s.Run("nil request does not panic", func() {
		var req *CreateTenantRequest
		s.NotPanics(func() { req.Normalize() })
	})
}
