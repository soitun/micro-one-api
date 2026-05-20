import { NavLink, Navigate, Outlet } from 'react-router-dom';
import { useState } from 'react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { ThemeToggle } from '@/components/ThemeToggle';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';

export function ProtectedRoute() {
  const token = localStorage.getItem('token');
  const [adminToken, setAdminToken] = useState(localStorage.getItem('adminToken') || '');
  const [adminDialogOpen, setAdminDialogOpen] = useState(false);
  const [adminInput, setAdminInput] = useState('');

  if (!token) {
    return <Navigate to="/login" replace />;
  }

  const handleSetAdminToken = () => {
    localStorage.setItem('adminToken', adminInput);
    setAdminToken(adminInput);
    setAdminDialogOpen(false);
    toast.success('Admin access enabled');
  };

  const handleClearAdminToken = () => {
    localStorage.removeItem('adminToken');
    setAdminToken('');
    toast.success('Admin access disabled');
  };

  const isAdmin = !!adminToken;
  const navLinkClass = ({ isActive }: { isActive: boolean }) =>
    `text-sm ${isActive ? 'text-foreground' : 'text-muted-foreground hover:text-foreground'}`;

  return (
    <div className="min-h-screen bg-background">
      <nav className="border-b">
        <div className="container mx-auto px-4 py-3 flex items-center gap-6">
          <h1 className="text-lg font-semibold">One API</h1>
          <NavLink to="/dashboard" className={navLinkClass}>
            Dashboard
          </NavLink>
          <NavLink to="/tokens" className={navLinkClass}>
            Tokens
          </NavLink>
          {isAdmin && (
            <>
              <span className="text-muted-foreground">|</span>
              <NavLink to="/admin/users" className={navLinkClass}>
                Users
              </NavLink>
              <NavLink to="/admin/channels" className={navLinkClass}>
                Channels
              </NavLink>
              <NavLink to="/admin/logs" className={navLinkClass}>
                Logs
              </NavLink>
              <NavLink to="/admin/redemptions" className={navLinkClass}>
                Redemptions
              </NavLink>
              <NavLink to="/admin/options" className={navLinkClass}>
                Options
              </NavLink>
            </>
          )}
          <div className="ml-auto flex items-center gap-3">
            <ThemeToggle />
            {isAdmin ? (
              <Button variant="outline" size="sm" onClick={handleClearAdminToken}>
                Exit Admin
              </Button>
            ) : (
              <Dialog open={adminDialogOpen} onOpenChange={setAdminDialogOpen}>
                <DialogTrigger>
                  <Button variant="outline" size="sm">
                    Admin
                  </Button>
                </DialogTrigger>
                <DialogContent>
                  <DialogHeader>
                    <DialogTitle>Admin Access</DialogTitle>
                    <DialogDescription>Enter your admin token to access management features.</DialogDescription>
                  </DialogHeader>
                  <div className="space-y-4 pt-4">
                    <Input
                      type="password"
                      placeholder="Admin Token"
                      value={adminInput}
                      onChange={(e) => setAdminInput(e.target.value)}
                    />
                    <Button onClick={handleSetAdminToken} disabled={!adminInput.trim()} className="w-full">
                      Confirm
                    </Button>
                  </div>
                </DialogContent>
              </Dialog>
            )}
            <button
              onClick={() => {
                localStorage.removeItem('token');
                localStorage.removeItem('adminToken');
                window.location.href = '/login';
              }}
              className="text-sm text-muted-foreground hover:text-foreground"
            >
              Logout
            </button>
          </div>
        </div>
      </nav>
      <main className="container mx-auto px-4 py-8">
        <Outlet />
      </main>
    </div>
  );
}
