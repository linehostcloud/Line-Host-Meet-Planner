import {getConfigStatus} from 'client/client';
import React, {act} from 'react';
import {createRoot} from 'react-dom/client';
import type {Root} from 'react-dom/client';

import CallbackURL from './admin_callback_url';

jest.mock('client/client', () => ({getConfigStatus: jest.fn()}));

test('shows the exact callback URL and saved connection status without exposing credentials', async () => {
    const url = 'https://mattermost.example/plugins/com.linehost.meet-planner/api/v1/oauth/callback';
    (getConfigStatus as jest.Mock).mockResolvedValue({configured: true, is_admin: true, callback_url: url});
    const container = document.createElement('div');
    const root: Root = createRoot(container);

    await act(async () => root.render(<CallbackURL/>));

    expect(container.querySelector('input')?.value).toBe(url);
    expect(container.textContent).toContain('Conexão do plugin: configurada');
    expect(container.textContent).toContain('URIs de redirecionamento autorizados');

    await act(async () => root.unmount());
});
