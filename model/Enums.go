package model

// 1. Define the custom type (underlying type is string)
type CredentialType string

// 2. Define the exact allowed values
const (
	CredTypePassword CredentialType = "password"
	CredTypeGoogle   CredentialType = "google"
	CredTypeFacebook CredentialType = "facebook"
	CredTypeGithub   CredentialType = "github"
	CredTypeZalo     CredentialType = "zalo" // easy to add new ones here
)

// Optional: Helper to validate if a string is a valid enum
func (ct CredentialType) IsValid() bool {
	switch ct {
	case CredTypePassword, CredTypeGoogle, CredTypeFacebook, CredTypeGithub, CredTypeZalo:
		return true
	}
	return false
}
