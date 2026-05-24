/* eslint-disable react-refresh/only-export-components */
import { lazy, Suspense } from 'react';
import { createBrowserRouter, Navigate } from 'react-router-dom';
import { ProtectedRoute } from '@/components/ProtectedRoute';
import { PageLoading } from '@/components/PageLoading';

const LoginPage = lazy(() => import('@/pages/LoginPage').then((m) => ({ default: m.LoginPage })));
const DashboardPage = lazy(() => import('@/pages/DashboardPage').then((m) => ({ default: m.DashboardPage })));
const TokensPage = lazy(() => import('@/pages/TokensPage').then((m) => ({ default: m.TokensPage })));
const UsagePage = lazy(() => import('@/pages/UsagePage').then((m) => ({ default: m.UsagePage })));
const OrdersPage = lazy(() => import('@/pages/OrdersPage').then((m) => ({ default: m.OrdersPage })));
const AdminUsersPage = lazy(() => import('@/pages/admin/UsersPage').then((m) => ({ default: m.AdminUsersPage })));
const AdminChannelsPage = lazy(() =>
  import('@/pages/admin/ChannelsPage').then((m) => ({ default: m.AdminChannelsPage }))
);
const AdminLogsPage = lazy(() => import('@/pages/admin/LogsPage').then((m) => ({ default: m.AdminLogsPage })));
const AdminRedemptionsPage = lazy(() =>
  import('@/pages/admin/RedemptionsPage').then((m) => ({ default: m.AdminRedemptionsPage }))
);
const AdminOptionsPage = lazy(() =>
  import('@/pages/admin/OptionsPage').then((m) => ({ default: m.AdminOptionsPage }))
);

function withSuspense(element: React.ReactNode) {
  return <Suspense fallback={<PageLoading />}>{element}</Suspense>;
}

export const router = createBrowserRouter([
  {
    path: '/login',
    element: withSuspense(<LoginPage />),
  },
  {
    path: '/',
    element: <ProtectedRoute />,
    children: [
      {
        index: true,
        element: <Navigate to="/dashboard" replace />,
      },
      {
        path: 'dashboard',
        element: withSuspense(<DashboardPage />),
      },
      {
        path: 'tokens',
        element: withSuspense(<TokensPage />),
      },
      {
        path: 'usage',
        element: withSuspense(<UsagePage />),
      },
      {
        path: 'orders',
        element: withSuspense(<OrdersPage />),
      },
      {
        path: 'admin/users',
        element: withSuspense(<AdminUsersPage />),
      },
      {
        path: 'admin/channels',
        element: withSuspense(<AdminChannelsPage />),
      },
      {
        path: 'admin/logs',
        element: withSuspense(<AdminLogsPage />),
      },
      {
        path: 'admin/redemptions',
        element: withSuspense(<AdminRedemptionsPage />),
      },
      {
        path: 'admin/options',
        element: withSuspense(<AdminOptionsPage />),
      },
    ],
  },
]);
