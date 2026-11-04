/* eslint-disable react/jsx-no-literals */
import {createMeeting, scheduleMeeting} from 'client/client';
import manifest from 'manifest';
import React, {useEffect, useRef, useState} from 'react';
import {createPortal} from 'react-dom';
import type {Store} from 'redux';

import type {GlobalState} from '@mattermost/types/store';

import {GoogleMeetIcon} from 'components/icons';

import './meet_planner.scss';

type Props = {store: Store<GlobalState>};
type Mode = 'now' | 'schedule';
const reminderOptions = [
    {minutes: 1440, label: '1 dia antes'},
    {minutes: 120, label: '2 horas antes'},
    {minutes: 60, label: '1 hora antes'},
    {minutes: 30, label: '30 minutos antes'},
    {minutes: 15, label: '15 minutos antes'},
    {minutes: 10, label: '10 minutos antes'},
    {minutes: 5, label: '5 minutos antes'},
];

const localDateTime = (date: Date) => {
    const pad = (value: number) => String(value).padStart(2, '0');
    return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
};

export default function MeetPlannerButton({store}: Props) {
    const [open, setOpen] = useState(false);
    const [mode, setMode] = useState<Mode>('now');
    const [topic, setTopic] = useState('');
    const [startAt, setStartAt] = useState(() => localDateTime(new Date(Date.now() + (60 * 60 * 1000))));
    const [endAt, setEndAt] = useState(() => localDateTime(new Date(Date.now() + (2 * 60 * 60 * 1000))));
    const [guests, setGuests] = useState('');
    const [reminderMinutes, setReminderMinutes] = useState<number[]>([5]);
    const [busy, setBusy] = useState(false);
    const [error, setError] = useState('');
    const [needsConnect, setNeedsConnect] = useState(false);
    const titleRef = useRef<HTMLInputElement>(null);

    useEffect(() => {
        if (!open) {
            return undefined;
        }
        titleRef.current?.focus();
        const onKeyDown = (event: KeyboardEvent) => {
            if (event.key === 'Escape' && !busy) {
                setOpen(false);
            }
        };
        document.addEventListener('keydown', onKeyDown);
        return () => document.removeEventListener('keydown', onKeyDown);
    }, [open, busy]);

    const submit = async (event: React.FormEvent) => {
        event.preventDefault();
        setError('');
        setNeedsConnect(false);
        const channelID = store.getState().entities.channels.currentChannelId;
        if (!channelID) {
            setError('Abra um canal ou conversa antes de criar a reunião.');
            return;
        }
        if (mode === 'schedule') {
            const start = new Date(startAt);
            const end = new Date(endAt);
            if (!topic.trim() || !Number.isFinite(start.getTime()) || !Number.isFinite(end.getTime()) || start.getTime() <= Date.now() + 60_000 || end <= start) {
                setError('Informe um título e um horário futuro de início e fim.');
                return;
            }
            if (reminderMinutes.some((minutes) => start.getTime() - (minutes * 60_000) <= Date.now() + 60_000)) {
                setError('Um aviso escolhido já está próximo ou passou. Escolha um aviso mais curto ou um início mais distante.');
                return;
            }
        }
        setBusy(true);
        try {
            if (mode === 'now') {
                const connectionID = store.getState().websocket?.connectionId || '';
                const response = await createMeeting(channelID, connectionID, topic.trim());
                if (response.status !== 'ok') {
                    if (response.reason === 'not_connected' || response.reason === 'needs_reconnect') {
                        setNeedsConnect(true);
                        throw new Error('Conecte sua conta Google para iniciar a reunião.');
                    }
                    if (response.reason === 'not_configured') {
                        throw new Error('O administrador precisa configurar o Google Meet neste servidor.');
                    }
                    if (response.reason === 'permission_denied' || response.reason === 'public_channel_restricted') {
                        throw new Error('Você não tem permissão para iniciar uma reunião nesta conversa.');
                    }
                    throw new Error('Não foi possível iniciar a reunião. Tente novamente.');
                }
            } else {
                await scheduleMeeting({
                    channel_id: channelID,
                    topic: topic.trim(),
                    start_at: new Date(startAt).toISOString(),
                    end_at: new Date(endAt).toISOString(),
                    attendees: guests.split(/[,;\n]+/).map((value) => value.trim()).filter(Boolean),
                    time_zone: new Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC',
                    reminder_minutes: reminderMinutes,
                });
            }
            setOpen(false);
            setTopic('');
            setGuests('');
            setReminderMinutes([5]);
        } catch (caught) {
            const message = caught instanceof Error ? caught.message : 'Não foi possível concluir a reunião.';
            if (message.includes('/meet connect')) {
                setNeedsConnect(true);
            }
            setError(message);
        } finally {
            setBusy(false);
        }
    };

    let submitLabel = mode === 'now' ? 'Iniciar reunião' : 'Agendar reunião';
    if (busy) {
        submitLabel = 'Aguarde…';
    }

    return (<>
        <button
            type='button'
            className='meet-planner-trigger'
            title='Criar ou agendar Google Meet'
            aria-label='Criar ou agendar Google Meet'
            onClick={() => {
                setError('');
                setOpen(true);
            }}
        >
            <GoogleMeetIcon/>
            <span>Meet</span>
        </button>
        {open && createPortal(
            <div
                className='meet-planner-overlay'
                onMouseDown={(event) => {
                    if (event.target === event.currentTarget && !busy) {
                        setOpen(false);
                    }
                }}
            >
                <section
                    className='meet-planner-dialog'
                    role='dialog'
                    aria-modal='true'
                    aria-labelledby='meet-planner-title'
                >
                    <div className='meet-planner-heading'>
                        <h2 id='meet-planner-title'>Google Meet</h2>
                        <button
                            type='button'
                            className='meet-planner-close'
                            aria-label='Fechar'
                            disabled={busy}
                            onClick={() => setOpen(false)}
                        >×</button>
                    </div>
                    <div
                        className='meet-planner-modes'
                        role='group'
                        aria-label='Tipo de reunião'
                    >
                        <button
                            type='button'
                            className={mode === 'now' ? 'selected' : ''}
                            aria-pressed={mode === 'now'}
                            onClick={() => setMode('now')}
                        >Iniciar agora</button>
                        <button
                            type='button'
                            className={mode === 'schedule' ? 'selected' : ''}
                            aria-pressed={mode === 'schedule'}
                            onClick={() => setMode('schedule')}
                        >Agendar</button>
                    </div>
                    <form onSubmit={submit}>
                        <label htmlFor='meet-planner-topic'>Título {mode === 'now' ? '(opcional)' : ''}</label>
                        <input
                            id='meet-planner-topic'
                            ref={titleRef}
                            value={topic}
                            maxLength={200}
                            onChange={(event) => setTopic(event.target.value)}
                            required={mode === 'schedule'}
                            placeholder='Assunto da reunião'
                        />
                        {mode === 'schedule' && <>
                            <div className='meet-planner-times'>
                                <div>
                                    <label htmlFor='meet-planner-start'>Início</label>
                                    <input
                                        id='meet-planner-start'
                                        type='datetime-local'
                                        value={startAt}
                                        onChange={(event) => setStartAt(event.target.value)}
                                        required={true}
                                    />
                                </div>
                                <div>
                                    <label htmlFor='meet-planner-end'>Fim</label>
                                    <input
                                        id='meet-planner-end'
                                        type='datetime-local'
                                        value={endAt}
                                        onChange={(event) => setEndAt(event.target.value)}
                                        required={true}
                                    />
                                </div>
                            </div>
                            <label htmlFor='meet-planner-guests'>Convidados por e-mail (opcional)</label>
                            <textarea
                                id='meet-planner-guests'
                                value={guests}
                                onChange={(event) => setGuests(event.target.value)}
                                placeholder='nome@empresa.com, outra@empresa.com'
                                rows={2}
                            />
                            <fieldset className='meet-planner-reminders'>
                                <legend>Avisar neste canal</legend>
                                <div className='meet-planner-reminder-options'>
                                    {reminderOptions.map(({minutes, label}) => (
                                        <label key={minutes}>
                                            <input
                                                type='checkbox'
                                                checked={reminderMinutes.includes(minutes)}
                                                onChange={(event) => setReminderMinutes((current) => {
                                                    return event.target.checked ? [...current, minutes] : current.filter((value) => value !== minutes);
                                                })}
                                            />
                                            {label}
                                        </label>
                                    ))}
                                </div>
                            </fieldset>
                            <p className='meet-planner-help'>O evento será criado no calendário da sua conta Google. Os convidados receberão um convite, o link será publicado nesta conversa e os avisos selecionados aparecerão no canal. Sem seleção, não haverá aviso no Mattermost.</p>
                        </>}
                        {error && (
                            <p
                                className='meet-planner-error'
                                role='alert'
                            >
                                {error}
                            </p>
                        )}
                        {needsConnect && <a href={`/plugins/${manifest.id}/api/v1/oauth/connect`}>Conectar conta Google</a>}
                        <div className='meet-planner-actions'>
                            <button
                                type='button'
                                disabled={busy}
                                onClick={() => setOpen(false)}
                            >Cancelar</button>
                            <button
                                type='submit'
                                className='primary'
                                disabled={busy}
                            >{submitLabel}</button>
                        </div>
                    </form>
                </section>
            </div>,
            document.body,
        )}
    </>);
}
