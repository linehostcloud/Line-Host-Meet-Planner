// Copyright (c) 2026-present Mattermost, Inc. All Rights Reserved.
// Modified by Line Host® Cloud Solutions, 2026.
// See LICENSE.txt for license information.

package main

import (
	"fmt"

	"github.com/linehostcloud/Line-Host-Meet-Planner/server/store/kvstore"
)

func (p *Plugin) GetConnectURL() string {
	return p.getOAuth2ConnectURL()
}

func (p *Plugin) DisconnectUser(userID string) error {
	store, err := p.getOAuthKVStore()
	if err != nil {
		return err
	}

	if err := store.DeleteOAuth2Token(userID); err != nil {
		return fmt.Errorf("failed to delete user token: %w", err)
	}

	return nil
}

func (p *Plugin) IsUserConnected(userID string) (bool, error) {
	store, err := p.getOAuthKVStore()
	if err != nil {
		return false, err
	}

	token, err := p.getCurrentOAuthToken(store, userID)
	if err != nil {
		return false, err
	}

	return token != nil, nil
}

func (p *Plugin) getCurrentOAuthToken(store kvstore.KVStore, userID string) (*kvstore.OAuth2Token, error) {
	token, err := store.GetOAuth2Token(userID)
	if err != nil || token == nil {
		return token, err
	}
	if token.ClientID == p.getConfiguration().GoogleClientID {
		return token, nil
	}
	// Legacy tokens have no client ID. After changing OAuth clients, they must not
	// appear connected or be used with the new client's credentials.
	if err := store.DeleteOAuth2Token(userID); err != nil {
		return nil, fmt.Errorf("failed to remove token for another OAuth client: %w", err)
	}
	return nil, nil
}

func (p *Plugin) IsUserAdmin(userID string) (bool, error) {
	user, appErr := p.API.GetUser(userID)
	if appErr != nil {
		return false, fmt.Errorf("failed to get user: %w", appErr)
	}

	return user.IsSystemAdmin(), nil
}
