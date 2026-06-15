import { screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { http, HttpResponse } from 'msw';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it } from 'vitest';
import { AdminLogsPage } from './LogsPage';
import { renderWithQuery } from '@/test/render';
import { server } from '@/test/msw/server';

describe('AdminLogsPage', () => {
  it('opens log details from the billing log table', async () => {
    const user = userEvent.setup();

    server.use(
      http.get('/api/log', () =>
        HttpResponse.json({
          success: true,
          data: {
            logs: [
              {
                id: '123',
                userId: '42',
                type: 'consume',
                amount: '-1000000',
                balanceAfter: '4000000',
                referenceId: 'req-123',
                remark: 'summary row',
                createdAt: '1760000000',
              },
            ],
            total: 1,
          },
        }),
      ),
      http.get('/api/log/123', () =>
        HttpResponse.json({
          success: true,
          data: {
            id: '123',
            type: 'consume',
            user_id: 42,
            username: 'alice',
            source: 'relay',
            request_id: 'req-123',
            model_name: 'gpt-4o-mini',
            token_name: 'prod-key',
            channel: 7,
            quota: 1000000,
            prompt_tokens: 11,
            completion_tokens: 13,
            cache_read_tokens: 5,
            elapsed_time: 250,
            is_stream: true,
            created_at: 1760000000,
            message: 'full relay log message',
          },
        }),
      ),
    );

    renderWithQuery(
      <MemoryRouter>
        <AdminLogsPage />
      </MemoryRouter>,
    );

    await user.click(await screen.findByRole('button', { name: 'View log 123' }));

    const dialog = await screen.findByRole('dialog');
    expect(within(dialog).getByRole('heading', { name: 'Log Details' })).toBeInTheDocument();
    expect(within(dialog).getByText('gpt-4o-mini')).toBeInTheDocument();
    expect(within(dialog).getByText('prod-key')).toBeInTheDocument();
    expect(within(dialog).getByText('full relay log message')).toBeInTheDocument();
    expect(within(dialog).getByText('250 ms')).toBeInTheDocument();
  });
});
