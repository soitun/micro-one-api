import { Navigate, Outlet } from 'react-router-dom';

export function ProtectedRoute() {
  const token = localStorage.getItem('token');

  if (!token) {
    return <Navigate to="/login" replace />;
  }

  return (
    <div className="min-h-screen bg-background">
      <nav className="border-b">
        <div className="container mx-auto px-4 py-3 flex items-center gap-6">
          <h1 className="text-lg font-semibold">One API</h1>
          <a href="/dashboard" className="text-sm text-muted-foreground hover:text-foreground">
            Dashboard
          </a>
          <a href="/tokens" className="text-sm text-muted-foreground hover:text-foreground">
            Tokens
          </a>
          <button
            onClick={() => {
              localStorage.removeItem('token');
              window.location.href = '/login';
            }}
            className="ml-auto text-sm text-muted-foreground hover:text-foreground"
          >
            Logout
          </button>
        </div>
      </nav>
      <main className="container mx-auto px-4 py-8">
        <Outlet />
      </main>
    </div>
  );
}
