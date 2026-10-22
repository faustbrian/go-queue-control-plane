package server

import (
	authentication "github.com/faustbrian/go-authentication/v2"
	authhttp "github.com/faustbrian/go-authentication/v2/adapters/http"
	"github.com/faustbrian/go-authentication/v2/apikey"
	authorization "github.com/faustbrian/go-authorization"
	"github.com/faustbrian/go-authorization/acl"
	controlauthz "github.com/faustbrian/go-queue-control-plane/v5/authz"
)

const (
	// APIKeyIDHeader carries the non-secret static key identifier.
	// #nosec G101 -- public HTTP header name, not a credential; credential values come from caller-owned authentication
	APIKeyIDHeader = "X-Queue-Control-Key-ID"
	// APIKeySecretHeader carries the static key credential.
	// #nosec G101 -- public HTTP header name, not a credential; credential values come from caller-owned authentication
	APIKeySecretHeader = "X-Queue-Control-Key"
)

// StaticAccess is a coherent authentication and authorization configuration.
type StaticAccess struct {
	// Extractor reads the static key headers; it does not itself authenticate
	// credentials.
	Extractor *authhttp.Extractor
	// Authenticator verifies startup-configured key material; the bundle
	// supplies it to middleware.
	Authenticator authentication.Authenticator
	// Authorizer applies deny-overrides tenant/resource ACL decisions to
	// authenticated actors.
	Authorizer *controlauthz.Authorizer
	// Challenge is the public authentication challenge, not stored credential
	// material.
	Challenge authentication.Challenge
}

// NewStaticAccess builds bounded API-key authentication and deny-overrides ACL
// authorization from immutable startup configuration.
func NewStaticAccess(keys []apikey.Entry, entries []acl.Entry) (*StaticAccess, error) {
	authenticator, err := apikey.NewStatic(keys)
	if err != nil {
		return nil, err
	}
	extractor := must(authhttp.NewExtractor(
		authhttp.APIKeyHeader(APIKeyIDHeader, APIKeySecretHeader),
	))
	evaluator, err := acl.New(entries)
	if err != nil {
		return nil, err
	}
	snapshot := must(authorization.NewSnapshot(
		1,
		authorization.DenyOverrides,
		authorization.PolicyDefinition{
			ID:        "queue-control-plane-acl",
			Evaluator: evaluator,
		},
	))
	engine := must(authorization.NewEngine(snapshot))
	authorizer := must(controlauthz.New(engine, authorization.SubjectAPIKey))

	return &StaticAccess{
		Extractor:     extractor,
		Authenticator: authenticator,
		Authorizer:    authorizer,
		Challenge:     must(authentication.NewChallenge("QueueControlKey", nil)),
	}, nil
}

func must[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}

	return value
}
