package josejwt

import (
	"fmt"
	"strings"
	"time"

	"github.com/SermoDigital/jose/crypto"
	"github.com/SermoDigital/jose/jws"
	"github.com/hashicorp/vault/logical"
)

// TokenCreateEntry is the exposed structure for creating a token
type TokenCreateEntry struct {
	TTL int `json:"ttl" structs:"ttl" mapstructure:"ttl"`

	Claims map[string]string `json:"claims" structs:"claims" mapstructure:"claims"`

	RoleName string `json:"role_name" structs:"role_name" mapstructure:"role_name"`

	RoleID string `json:"role_id" structs:"role_id" mapstructure:"role_id"`

	KeyName string `json:"key_name" structs:"key_name" mapstructure:"key_name"`

	TokenType string `json:"token_type" structs:"token_type" mapstructure:"token_type"`
}

func createJwtToken(backend *JwtBackend, storage logical.Storage, createEntry TokenCreateEntry, roleEntry *RoleStorageEntry) (map[string]interface{}, error) {
	claims := jws.Claims{}

	if roleEntry.AllowCustomClaims {
		// merge the custom claims onto the role claims
		for k, v := range createEntry.Claims {
			roleEntry.Claims[k] = v
		}
	}

	for k, v := range roleEntry.Claims {
		claims.Set(k, v)
	}

	claims.SetExpiration(time.Now().UTC().Add(time.Duration(createEntry.TTL) * time.Second))

	token := jws.NewJWT(claims, crypto.SigningMethodHS256)

	// read the secret for this role
	secret, err := backend.readSecret(storage, roleEntry.RoleID, roleEntry.SecretID)
	if err != nil {
		return nil, err
	} else if secret == nil {
		secret, err = backend.rotateSecret(storage, roleEntry.RoleID, roleEntry.SecretID, roleEntry.SecretTTL)
		if err != nil {
			return nil, err
		}
	}

	serializedToken, _ := token.Serialize([]byte(secret.Key))
	tokenOutput := map[string]interface{}{"ClientToken": string(serializedToken[:])}

	return tokenOutput, nil
}

func (backend *JwtBackend) createTokenEntry(storage logical.Storage, createEntry TokenCreateEntry, roleEntry *RoleStorageEntry) (map[string]interface{}, error) {
	createEntry.TokenType = strings.ToLower(createEntry.TokenType)

	switch createEntry.TokenType {
	case "jws":
		return nil, nil
	case "jwt":
		return createJwtToken(backend, storage, createEntry, roleEntry)
	default:
		// throw an error
		return nil, fmt.Errorf("unsupported token type %s", createEntry.TokenType)
	}
}