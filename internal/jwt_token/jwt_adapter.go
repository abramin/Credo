package jwttoken

import (
	authmw "credo/pkg/platform/middleware/auth"
)

func ToMiddlewareClaims(claims *AccessTokenClaims) *authmw.JWTClaims {
	return &authmw.JWTClaims{
		UserID:     claims.UserID,
		SessionID:  claims.SessionID,
		ClientID:   claims.ClientID,
		JTI:        claims.ID,                    // JWT ID for revocation tracking
		APIVersion: claims.APIVersion().String(), // API version from token audience
	}
}

type JWTServiceAdapter struct {
	service *JWTService
}

func NewJWTServiceAdapter(service *JWTService) *JWTServiceAdapter {
	return &JWTServiceAdapter{service: service}
}

func (a *JWTServiceAdapter) ValidateToken(tokenString string) (*authmw.JWTClaims, error) {
	claims, err := a.service.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}
	return ToMiddlewareClaims(claims), nil
}
