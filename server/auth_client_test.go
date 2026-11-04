package main

import (
	"testing"

	"github.com/linehostcloud/Line-Host-Meet-Planner/server/store/kvstore"
)

func TestGetCurrentOAuthTokenRemovesPreviousClientTokens(t *testing.T) {
	for _, tc := range []struct {
		name     string
		clientID string
		want     bool
	}{
		{"legacy token", "", false},
		{"previous client", "previous-client", false},
		{"current client", "current-client", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := newMockKVStore()
			store.tokens["user"] = &kvstore.OAuth2Token{ClientID: tc.clientID, AccessToken: "access"}
			p := &Plugin{}
			p.setConfiguration(&configuration{GoogleClientID: "current-client"})

			token, err := p.getCurrentOAuthToken(store, "user")
			if err != nil {
				t.Fatal(err)
			}
			if (token != nil) != tc.want {
				t.Fatalf("token present = %v, want %v", token != nil, tc.want)
			}
			if (store.tokens["user"] != nil) != tc.want {
				t.Fatalf("stored token present = %v, want %v", store.tokens["user"] != nil, tc.want)
			}
		})
	}
}
