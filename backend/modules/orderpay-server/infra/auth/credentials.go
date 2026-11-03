package auth

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"

	"github.com/fino-io/gokit/session"
	"github.com/fino-io/orderpay/backend/modules/orderpay-server/domain/model"
	"github.com/google/uuid"
)

type Credentials map[[32]byte]model.Principal

type credential struct {
	MerchantID string `json:"merchantId"`
	Admin      bool   `json:"admin"`
}

func Load() (Credentials, error) {
	var entries map[string]credential
	if err := json.Unmarshal([]byte(os.Getenv("ORDERPAY_API_KEYS")), &entries); err != nil || len(entries) == 0 {
		return nil, errors.New("ORDERPAY_API_KEYS must contain configured service credentials")
	}
	result := make(Credentials, len(entries))
	for key, entry := range entries {
		if len(key) < 32 {
			return nil, errors.New("service credentials must have at least 32 characters")
		}
		if !entry.Admin {
			if _, err := uuid.Parse(entry.MerchantID); err != nil {
				return nil, errors.New("merchantId must be a UUID")
			}
		}
		principal := model.Principal{
			MerchantID: entry.MerchantID,
			ActorID:    entry.MerchantID,
			Admin:      entry.Admin,
		}
		if entry.Admin {
			principal.ActorID = "admin"
		}
		result[sha256.Sum256([]byte(key))] = principal
	}
	return result, nil
}

func (c Credentials) Authenticate(ctx context.Context) (model.Principal, bool) {
	token, ok := session.TokenFromContext(ctx)
	if !ok {
		return model.Principal{}, false
	}
	principal, ok := c[sha256.Sum256([]byte(token))]
	return principal, ok
}
