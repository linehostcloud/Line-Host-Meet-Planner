// Modified by Line Host® Cloud Solutions, 2026.
import manifest from 'manifest';
import {getCsrfToken} from 'utils/csrf';

export interface ConfigStatusResponse {
    configured: boolean;
    is_admin: boolean;
    callback_url?: string;
}

export interface CreateMeetingResponse {
    status?: string;
    meeting_url?: string;
    reason?: string;
    message?: string;
    error?: string;
}

export interface ScheduleMeetingPayload {
    channel_id: string;
    topic: string;
    start_at: string;
    end_at: string;
    attendees: string[];
    time_zone: string;
    reminder_minutes: number[];
}

export interface ScheduleMeetingResponse {
    status?: string;
    meeting_url?: string;
    calendar_url?: string;
    error?: string;
}

const doFetch = async (url: string, options: RequestInit = {}): Promise<Response> => {
    const {headers: optHeaders, ...restOptions} = options;
    return fetch(url, {
        credentials: 'include',
        ...restOptions,
        headers: {
            'Content-Type': 'application/json',
            'X-CSRF-Token': getCsrfToken(),
            ...(optHeaders || {}),
        },
    });
};

export const getConfigStatus = async (): Promise<ConfigStatusResponse> => {
    const statusResp = await doFetch(`/plugins/${manifest.id}/api/v1/config/status`);
    if (!statusResp.ok) {
        throw new Error(`Failed to load config status (HTTP ${statusResp.status}).`);
    }
    return statusResp.json();
};

export const createMeeting = async (channelID: string, connectionId = '', topic = ''): Promise<CreateMeetingResponse> => {
    let resp: Response;
    try {
        resp = await doFetch(`/plugins/${manifest.id}/api/v1/meeting`, {
            method: 'POST',
            body: JSON.stringify({
                channel_id: channelID,
                connection_id: connectionId,
                topic,
            }),
        });
    } catch {
        throw new Error('Unable to reach the server to start a Google Meet meeting. Please try again.');
    }

    const contentType = resp.headers.get('Content-Type') || '';
    if (!contentType.includes('application/json')) {
        const message = resp.ok ?
            'Received an unexpected response from the server while starting a Google Meet meeting.' :
            `Failed to start a Google Meet meeting (HTTP ${resp.status}). Please try again.`;
        throw new Error(message);
    }

    let data: CreateMeetingResponse;
    try {
        data = await resp.json();
    } catch {
        throw new Error('Received an unreadable response from the server while starting a Google Meet meeting.');
    }

    if (!resp.ok) {
        const serverContext = data.reason || data.message || data.error;
        const message = serverContext ?
            `Failed to start a Google Meet meeting (HTTP ${resp.status}): ${serverContext}. Please try again.` :
            `Failed to start a Google Meet meeting (HTTP ${resp.status}). Please try again.`;
        throw new Error(message);
    }

    return data;
};

export const scheduleMeeting = async (payload: ScheduleMeetingPayload): Promise<ScheduleMeetingResponse> => {
    const resp = await doFetch(`/plugins/${manifest.id}/api/v1/meeting/schedule`, {
        method: 'POST',
        body: JSON.stringify(payload),
    });
    if (!(resp.headers.get('Content-Type') || '').includes('application/json')) {
        throw new Error(`Resposta inesperada do servidor (HTTP ${resp.status}).`);
    }
    const data = await resp.json() as ScheduleMeetingResponse;
    if (!resp.ok) {
        const suffix = data.calendar_url ? ` O evento já existe no Google Calendar: ${data.calendar_url}` : '';
        throw new Error((data.error || `Não foi possível agendar (HTTP ${resp.status}).`) + suffix);
    }
    return data;
};
