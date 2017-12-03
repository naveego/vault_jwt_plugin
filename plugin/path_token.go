package josejwt

import (
	"github.com/SermoDigital/jose/crypto"
	"github.com/SermoDigital/jose/jws"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
	"github.com/mitchellh/mapstructure"
)

// basic schema for the validation of the token,
// this will map the fields coming in from the vault request field map
var validateTokenSchema = map[string]*framework.FieldSchema{
	"role_name": {
		Type:        framework.TypeString,
		Description: "The role associated with this token",
	},
	"token": {
		Type:        framework.TypeString,
		Description: "The Token to validate",
	},
}

// basic schema for the creation of the token,
// this will map the fields coming in from the vault request field map
var createTokenSchema = map[string]*framework.FieldSchema{
	"claims": {
		Type:        framework.TypeCommaStringSlice,
		Description: "The custom claims that are aplied to the token",
	},
	"payload": {
		Type:        framework.TypeCommaStringSlice,
		Description: "The custom payload applied to the token",
	},
	"role_name": {
		Type:        framework.TypeString,
		Description: "The name of the role to use in the token",
	},
	"role_id": {
		Type:        framework.TypeString,
		Description: "The unique identifier for the role to use in the token",
	},
	"ttl": {
		Type:        framework.TypeDurationSecond,
		Description: "The duration in seconds after which the token will expire",
		Default:     600, // default of 10 minutes
	},
}

// Provides basic token validation for a provided jwt token
func (backend *JwtBackend) validateToken(req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	byteToken := []byte(data.Get("token").(string))
	token, err := jws.ParseJWT(byteToken)

	if err != nil {
		return logical.ErrorResponse("unable to parse token"), err
	}

	roleName := data.Get("role_name").(string)
	if roleName == "" {
		roleName = token.Claims().Get("roleName").(string)
	}

	role, err := backend.getRoleEntry(req.Storage, roleName)
	if err != nil {
		return logical.ErrorResponse("unable to retrieve role details"), err
	}

	secret, err := backend.readSecret(req.Storage, role.RoleID, role.SecretID)
	if err != nil {
		return logical.ErrorResponse("unable to retrieve role secrets"), err
	} else if secret == nil {
		validation := map[string]interface{}{
			"is_valid": false,
		}

		return &logical.Response{Data: validation}, nil
	}

	//err = token.Verify([]byte(secret.Key), crypto.SigningMethodHS256)
	err = token.Validate([]byte(secret.Key), crypto.SigningMethodHS256)
	if err != nil {
		return logical.ErrorResponse("Invalid Token"), err
	}

	validation := map[string]interface{}{
		"is_valid": true,
	}
	return &logical.Response{Data: validation}, nil
}

// create the basic jwt token with an expiry wihtin the claim
func (backend *JwtBackend) createToken(req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	roleName := data.Get("role_name").(string)
	roleID := data.Get("role_id").(string)

	// get the role by name
	roleEntry, err := backend.getRoleEntry(req.Storage, roleName)
	if err != nil {
		return logical.ErrorResponse("Role note recognised"), nil
	}

	salt, _ := backend.Salt()
	hmac := salt.GetHMAC(roleID)

	if hmac != roleEntry.HMAC {
		return logical.ErrorResponse("unauthorized access"), nil
	}

	var tokenEntry TokenCreateEntry
	if err := mapstructure.Decode(data.Raw, &tokenEntry); err != nil {
		return logical.ErrorResponse("Error decoding role"), err
	}
	tokenEntry.TTL = roleEntry.TokenTTL

	token, err := backend.createTokenEntry(req.Storage, tokenEntry, roleEntry)
	if err != nil {
		return logical.ErrorResponse("Error creating token"), err
	}

	return &logical.Response{Data: token}, nil
}

func pathToken(backend *JwtBackend) []*framework.Path {
	tokenSchema := map[string]*framework.FieldSchema{}
	for k, v := range createTokenSchema {
		tokenSchema[k] = v
	}

	for k, v := range validateTokenSchema {
		tokenSchema[k] = v
	}

	paths := []*framework.Path{
		&framework.Path{
			Pattern: "token/issue",
			Fields:  tokenSchema,
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.ReadOperation: backend.createToken,
			},
		},
		&framework.Path{
			Pattern: "token/validate",
			Fields:  tokenSchema,
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.ReadOperation: backend.validateToken,
			},
		},
	}

	return paths
}
