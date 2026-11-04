// Modified by Line Host® Cloud Solutions, 2026.
import {getConfigStatus} from 'client/client';
import manifest from 'manifest';
import React from 'react';
import type {Store} from 'redux';
import {getAuthUserPreference} from 'utils/auth_user';
import {handleMeetingStarted} from 'websocket/meeting_started';

import type {GlobalState} from '@mattermost/types/store';

import CallbackURL from 'components/admin_callback_url';
import MeetPlannerButton from 'components/meet_planner';
import {PostTypeRecording, PostTypeSmartNote, PostTypeTranscript} from 'components/post_type_artifact';
import PostTypeMeeting from 'components/post_type_meeting';

import type {PluginRegistry} from 'types/mattermost-webapp';

export default class Plugin {
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    public async initialize(registry: PluginRegistry, store: Store<GlobalState>) {
        let configured = false;
        let isAdmin = false;

        try {
            const status = await getConfigStatus();
            configured = status.configured;
            isAdmin = status.is_admin;
        } catch {
            // Keep the safe default when config status cannot be determined.
        }

        registry.registerPostTypeComponent('custom_google_meet', PostTypeMeeting);
        registry.registerAdminConsoleCustomSetting('CallbackURL', CallbackURL, {showTitle: true});
        registry.registerPostTypeComponent('custom_gmeet_conference', PostTypeMeeting);
        registry.registerPostTypeComponent('custom_gmeet_transcript', PostTypeTranscript);
        registry.registerPostTypeComponent('custom_gmeet_recording', PostTypeRecording);
        registry.registerPostTypeComponent('custom_gmeet_smartnote', PostTypeSmartNote);
        registry.registerWebSocketEventHandler<{meeting_url?: string}>(
            `custom_${manifest.id}_meeting_started`,
            (msg) => handleMeetingStarted(msg, getAuthUserPreference(store.getState())),
        );

        if (!configured && !isAdmin) {
            return;
        }

        registry.registerPostEditorActionComponent(() => <MeetPlannerButton store={store}/>);
    }
}

declare global {
    interface Window {
        registerPlugin(pluginId: string, plugin: Plugin): void;
    }
}

window.registerPlugin(manifest.id, new Plugin());
