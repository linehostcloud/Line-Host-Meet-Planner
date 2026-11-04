import React, {act} from 'react';
import {createRoot} from 'react-dom/client';
import type {Root} from 'react-dom/client';

import MeetPlannerButton from './meet_planner';

describe('MeetPlannerButton', () => {
    let container: HTMLDivElement;
    let root: Root;

    beforeEach(() => {
        container = document.createElement('div');
        document.body.appendChild(container);
        root = createRoot(container);
    });

    afterEach(async () => {
        await act(async () => root.unmount());
        container.remove();
    });

    test('opens the composer action and shows scheduling controls', async () => {
        await act(async () => {
            root.render(<MeetPlannerButton store={{getState: () => ({})} as never}/>);
        });

        const trigger = container.querySelector('button') as HTMLButtonElement;
        expect(trigger.getAttribute('aria-label')).toBe('Criar ou agendar Google Meet');
        await act(async () => trigger.click());
        expect(document.querySelector('[role="dialog"]')).not.toBeNull();

        const schedule = Array.from(document.querySelectorAll('button')).find((button) => button.textContent === 'Agendar') as HTMLButtonElement;
        await act(async () => schedule.click());

        expect(document.querySelector('#meet-planner-start')).not.toBeNull();
        expect(document.querySelector('#meet-planner-end')).not.toBeNull();
        expect(document.querySelector('#meet-planner-guests')).not.toBeNull();
        expect(document.querySelector('#meet-planner-topic')?.getAttribute('required')).not.toBeNull();

        const dayReminder = Array.from(document.querySelectorAll('.meet-planner-reminder-options label')).find((label) => label.textContent === '1 dia antes')?.querySelector('input') as HTMLInputElement;
        const twoHourReminder = Array.from(document.querySelectorAll('.meet-planner-reminder-options label')).find((label) => label.textContent === '2 horas antes')?.querySelector('input') as HTMLInputElement;
        await act(async () => dayReminder.click());
        await act(async () => twoHourReminder.click());
        expect(dayReminder.checked).toBe(true);
        expect(twoHourReminder.checked).toBe(true);
        expect((Array.from(document.querySelectorAll('.meet-planner-reminder-options label')).find((label) => label.textContent === '5 minutos antes')?.querySelector('input') as HTMLInputElement).checked).toBe(true);
    });
});
