package jwttoken

import (
	"errors"
	"time"

	dErrors "id-gateway/pkg/domain-errors"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims represents the JWT claims for our access tokens
type Claims struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
	ClientID  string `json:"client_id"`
	jwt.RegisteredClaims
}

// JWTService handles JWT creation and validation
type JWTService struct {
	signingKey []byte
	issuer     string
	audience   string
}

func NewJWTService(signingKey string, issuer string, audience string) *JWTService {
	return &JWTService{
		signingKey: []byte(signingKey),
		issuer:     issuer,
		audience:   audience,
	}
}

func (s *JWTService) GenerateAccessToken(
	userID uuid.UUID,
	sessionID uuid.UUID,
	clientID string,
	expiresIn time.Duration) (string, error) {
	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		UserID:    userID.String(),
		SessionID: sessionID.String(),
		ClientID:  clientID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    s.issuer,
			Audience:  []string{s.audience},
			ID:        uuid.NewString(),
		},
	})

	signedToken, err := newToken.SignedString(s.signingKey)
	if err != nil {
		return "", err
	}
	return signedToken, nil
}

func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	var err error
	parsed, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrTokenUnverifiable
		}
		return s.signingKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, dErrors.New(dErrors.CodeUnauthorized, "token has expired")
		}
		return nil, dErrors.New(dErrors.CodeUnauthorized, "invalid token")
	}

	if !parsed.Valid {
		return nil, dErrors.New(dErrors.CodeUnauthorized, "invalid token")
	}

	claims, ok := parsed.Claims.(*Claims)
	if !ok {
		return nil, dErrors.New(dErrors.CodeUnauthorized, "invalid token claims")
	}

	return claims, nil
}

func (s *JWTService) ExtractUserIDFromToken(tokenString string) (uuid.UUID, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return uuid.Nil, err
	}
	return uuid.Parse(claims.UserID)
}

func (s *JWTService) ExtractSessionIDFromAuthHeader(authHeader string) (uuid.UUID, error) {
	claims, err := s.ValidateToken(authHeader)
	if err != nil {
		return uuid.Nil, err
	}
	return uuid.Parse(claims.SessionID)
}
