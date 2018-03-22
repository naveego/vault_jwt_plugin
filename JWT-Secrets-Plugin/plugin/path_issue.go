package josejwt

import (
	"context"
	"fmt"
	"strings"

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

	"role_name": {
		Type:        framework.TypeString,
		Description: "The name of the role to use to create the token",
	},
	"claims": {
		Type:        framework.TypeKVPairs,
		Description: "The claims to add to the token.",
	},
	"token_ttl": {
		Type:        framework.TypeDurationSecond,
		Description: "The duration in seconds after which the token will expire",
	},
}

// create the basic jwt token with an expiry within the claim
func (backend *JwtBackend) issueToken(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {

	roleName := data.Get("role_name").(string)

	// get the role by name
	roleEntry, err := backend.getRoleEntry(ctx, req.Storage, roleName)
	if roleEntry == nil || err != nil {
		err = fmt.Errorf("Role name %q not recognised", roleName)
		return logical.ErrorResponse(err.Error()), err
	}

	keyEntry, err := backend.getKeyEntry(ctx, req.Storage, roleEntry.Key)
	if keyEntry == nil || err != nil {
		err = fmt.Errorf(fmt.Sprintf("Key name %q for role name %q not recognized", roleEntry.Key, roleName))
		return logical.ErrorResponse(err.Error()), err
	}

	var tokenEntry TokenCreateEntry
	if err := mapstructure.Decode(data.Raw, &tokenEntry); err != nil {
		return logical.ErrorResponse("Error decoding token"), err
	}

	if tokenEntry.TTL == 0 {
		// no TTL so use the default from the role
		tokenEntry.TTL = roleEntry.TokenTTL
	}

	if tokenEntry.TTL > roleEntry.MaxTokenTTL {
		// requested TTL exceeds max, so clip it
		tokenEntry.TTL = roleEntry.MaxTokenTTL
	}

	token, err := backend.createToken(tokenEntry, roleEntry, keyEntry)

	if err != nil {
		return logical.ErrorResponse(fmt.Sprintf("Error creating token, %#v", err)), err
	}

	response := map[string]interface{}{
		"token": string(token),
	}

	return &logical.Response{Data: response}, nil
}

// Provides basic token validation for a provided jwt token
func (backend *JwtBackend) validateToken(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {

	// TODO: implement validation
	// 0. Parse token into JWT.
	// 1. Get key for role.
	// 2. Validate JWT using key.

	validation := map[string]interface{}{
		"is_valid": true,
	}
	return &logical.Response{Data: validation}, nil
}

// refresh the provided token so that it can live on...
func (backend *JwtBackend) refreshToken(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {

	// TODO: implement refresh
	// 0. Parse token
	// 1. Check expiration
	// 2. If not expired, get original request
	// 3. If original request not expired, re-issue token

	// byteToken := []byte(data.Get("token").(string))
	// token, err := jws.ParseJWT(byteToken)

	// if err != nil {
	// 	return logical.ErrorResponse("unable to parse token"), err
	// }

	// roleName := data.Get("role_name").(string)
	// if roleName == "" {
	// 	roleName = token.Claims().Get("role-name").(string)
	// }

	// role, err := backend.getRoleEntry(ctx, req.Storage, roleName)
	// if err != nil {
	// 	return logical.ErrorResponse("unable to retrieve role details"), err
	// }
	// secretID := role.SecretID
	// tokenID := token.Claims().Get("id").(string)
	// if tokenID != "" {
	// 	secretID = tokenID
	// }

	// secret, err := backend.readSecret(ctx, req.Storage, role.RoleID, secretID)
	// if secret == nil {
	// 	// secret has probably expired so we will make a new one
	// 	secret, err = backend.createSecret(ctx, req.Storage, role.RoleID, role.TTL)
	// }
	// if err != nil {
	// 	return logical.ErrorResponse("Unable to regnerate the secret"), err
	// }

	// err = token.Validate([]byte(secret.Key), crypto.SigningMethodHS256)
	// if err != nil {
	// 	return logical.ErrorResponse("Invalid Token"), err
	// }

	// expiry := time.Now().Add(time.Duration(role.TokenTTL) * time.Second).UTC()
	// token.Claims().SetExpiration(expiry)

	// // make sure we update the expiry on the secret
	// secret.Expiration = expiry
	// backend.setSecretEntry(ctx, req.Storage, secret)

	// tokenData, _ := token.Serialize([]byte(secret.Key))
	// tokenOutput := map[string]interface{}{
	// 	"ClientToken": string(tokenData[:]),
	// }

	return &logical.Response{Data: nil}, nil
}

// split the display name, taking everything after the first dash '-'
func getRoleName(displayName string) string {
	index := strings.Index(displayName, "-")
	if index != -1 {
		return displayName[index+1:]
	}

	return displayName
}

func contains(array []string, value string) bool {
	for _, v := range array {
		if v == value {
			return true
		}
	}

	return false
}
func pathToken(backend *JwtBackend) []*framework.Path {
	paths := []*framework.Path{
		&framework.Path{
			Pattern: fmt.Sprintf("token/issue/%s", framework.GenericNameRegex("role_name")),
			Fields:  createTokenSchema,
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.UpdateOperation: backend.issueToken,
			},
		},
		&framework.Path{
			Pattern: fmt.Sprintf("token/validate/%s", framework.GenericNameRegex("role_name")),
			Fields:  validateTokenSchema,
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.UpdateOperation: backend.validateToken,
			},
		},
	}

	return paths
}
