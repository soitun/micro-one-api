import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it } from 'vitest';
import { AppNavigation } from './AppNavigation';

function renderNavigation() {
  return render(
    <MemoryRouter initialEntries={['/dashboard']}>
      <AppNavigation />
    </MemoryRouter>
  );
}

describe('AppNavigation', () => {
  it('renders user navigation links', () => {
    renderNavigation();

    expect(screen.getByRole('link', { name: 'Dashboard' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Tokens' })).toBeInTheDocument();
  });

  it('renders admin links when an admin token exists', () => {
    window.localStorage.setItem('adminToken', 'admin-token');

    renderNavigation();

    expect(screen.getByRole('link', { name: 'Users' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Options' })).toBeInTheDocument();
  });

  it('opens and closes the mobile menu', async () => {
    const user = userEvent.setup();
    renderNavigation();

    await user.click(screen.getByRole('button', { name: /open navigation/i }));
    expect(screen.getByRole('dialog')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /close/i }));
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('logout clears both tokens and redirects to login', async () => {
    const user = userEvent.setup();
    window.localStorage.setItem('token', 'user-token');
    window.localStorage.setItem('adminToken', 'admin-token');
    renderNavigation();

    await user.click(screen.getByRole('button', { name: 'Logout' }));

    expect(window.localStorage.getItem('token')).toBeNull();
    expect(window.localStorage.getItem('adminToken')).toBeNull();
  });
});
