package test

import "github.com/xaionaro-go/mediamtx/pkg/auth"

// AuthManager is a test auth manager.
type AuthManager struct {
	fnc func(req *auth.Request) error
}

// Authenticate replicates auth.Manager.Replicate
func (m *AuthManager) Authenticate(req *auth.Request) error {
	return m.fnc(req)
}

// NilAuthManager is an auth manager that accepts everything.
var NilAuthManager = &AuthManager{
	fnc: func(_ *auth.Request) error {
		return nil
	},
}