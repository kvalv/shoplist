package auth

import (
	"encoding/json"
	"testing"
)

const sampleIdentityJSON = `{
	"id": "100000000000000000000",
	"name": "John Doe",
	"email": "john.doe@example.com",
	"oidc_fields": {
		"picture": "https://lh3.googleusercontent.com/a-/example=s96-c",
		"name": "John Doe"
	},
	"idp": {"id": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", "type": "google"},
	"geo": {"country": "NO"},
	"user_uuid": "11111111-2222-3333-4444-555555555555",
	"account_id": "abcdef1234567890abcdef1234567890",
	"iat": 1771453742,
	"ip": "192.0.2.1"
}`

func TestParseIdentity(t *testing.T) {
	var identity cfIdentity
	if err := json.Unmarshal([]byte(sampleIdentityJSON), &identity); err != nil {
		t.Fatal(err)
	}

	if identity.Name != "John Doe" {
		t.Errorf("name = %q, want %q", identity.Name, "John Doe")
	}
	if identity.Email != "john.doe@example.com" {
		t.Errorf("email = %q, want %q", identity.Email, "john.doe@example.com")
	}
	if identity.picture() == "" {
		t.Error("picture is empty")
	}
	if identity.OIDCFields.Name != "John Doe" {
		t.Errorf("oidc_fields.name = %q, want %q", identity.OIDCFields.Name, "John Doe")
	}
}
