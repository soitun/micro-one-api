import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { http, HttpResponse } from 'msw';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, beforeEach, vi } from 'vitest';
import { SubscriptionsPage } from './SubscriptionsPage';
import { renderWithQuery } from '@/test/render';
import { server } from '@/test/msw/server';

describe('SubscriptionsPage', () => {
  beforeEach(() => {
    window.localStorage.setItem('userId', '42');
  });

  it('renders daily/weekly/monthly quota bars for the active subscription', async () => {
    server.use(
      http.get('/api/v1/subscriptions/progress', () =>
        HttpResponse.json({
          success: true,
          data: {
            id: 7,
            status: 'active',
            starts_at: 1700000000,
            expires_at: 1800000000,
            daily_used: { used: 5, limit: 10, remaining: 5 },
            weekly_used: { used: 2, limit: null, remaining: 0 },
            monthly_used: { used: 20, limit: 100, remaining: 80 },
            remaining_seconds: 86400 * 3,
          },
        }),
      ),
    );

    renderWithQuery(
      <MemoryRouter>
        <SubscriptionsPage />
      </MemoryRouter>,
    );

    expect(await screen.findByText('当前订阅')).toBeInTheDocument();
    expect(screen.getByText('$5.00 / $10.00')).toBeInTheDocument();
    expect(screen.getByText('无限制')).toBeInTheDocument(); // weekly limit null
    expect(screen.getByText('剩余 3 天')).toBeInTheDocument();
  });

  it('shows an empty state when there is no active subscription', async () => {
    server.use(
      http.get('/api/v1/subscriptions/progress', () =>
        HttpResponse.json({ success: false, message: 'not found' }),
      ),
    );

    renderWithQuery(
      <MemoryRouter>
        <SubscriptionsPage />
      </MemoryRouter>,
    );

    expect(await screen.findByText('暂无活跃订阅')).toBeInTheDocument();
  });

  it('creates a payment order when buying a subscription plan', async () => {
    const openSpy = vi.spyOn(window, 'open').mockReturnValue(null);
    const captured = { body: null as Record<string, unknown> | null };
    server.use(
      http.get('/api/v1/subscriptions/progress', () =>
        HttpResponse.json({ success: false, message: 'not found' }),
      ),
      http.get('/api/v1/subscriptions/groups', () =>
        HttpResponse.json({
          success: true,
          data: [
            {
              id: 9,
              name: 'codex-pro',
              display_name: 'Codex Pro',
              platform: 'codex',
              daily_limit_usd: 10,
              weekly_limit_usd: null,
              monthly_limit_usd: 100,
              price_quota: 10,
              duration_days: 30,
            },
          ],
        }),
      ),
      http.post('/api/v1/subscriptions/purchase/payment', async ({ request }) => {
        captured.body = (await request.json()) as Record<string, unknown>;
        return HttpResponse.json({
          success: true,
          data: {
            subscription: null,
            payment: { trade_no: 'PAY-1', pay_url: 'https://pay.example/PAY-1' },
          },
        });
      }),
    );

    renderWithQuery(
      <MemoryRouter>
        <SubscriptionsPage />
      </MemoryRouter>,
    );

    expect(await screen.findByText('Codex Pro')).toBeInTheDocument();
    expect(screen.getByText('$10.00')).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: '购买订阅' }));
    await userEvent.click(screen.getByRole('button', { name: '确认购买' }));

    await waitFor(() => expect(captured.body).not.toBeNull());
    expect(captured.body).toMatchObject({ group_id: 9, channel: 'alipay' });
    expect(openSpy).toHaveBeenCalledWith('about:blank', '_blank');
    expect(openSpy).toHaveBeenCalledWith('https://pay.example/PAY-1', '_blank', 'noopener,noreferrer');
  });
});
