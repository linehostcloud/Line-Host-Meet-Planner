// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// Modified by Line Host® Cloud Solutions, 2026.
// See LICENSE.txt for license information.

import manifest from 'manifest';

test('Plugin manifest and id are defined', () => {
    expect(manifest).toBeDefined();
    expect(manifest.id).toBeDefined();
    expect(manifest.id).toBe('com.linehost.meet-planner');
});
