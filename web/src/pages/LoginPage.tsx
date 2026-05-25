import { useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { apiClient } from '@/lib/api';
import { getApiErrorMessage } from '@/lib/api-error';
import { unwrapApiData } from '@/lib/api-response';

export function LoginPage() {
  const location = useLocation();
  const [mode, setMode] = useState<'login' | 'register'>(location.pathname === '/register' ? 'register' : 'login');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const navigate = useNavigate();

  const signIn = async (nextUsername: string) => {
    const response = await apiClient.post('/user/login', {
      username: nextUsername,
      password,
    });

    const data = unwrapApiData<string | { token?: string }>(response.data, 'Login failed');
    const token = typeof data === 'string' ? data : data?.token;
    if (!token) {
      throw new Error('Login failed');
    }

    localStorage.setItem('token', token);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    const nextUsername = username.trim();

    if (!nextUsername) {
      setError('Username is required');
      return;
    }
    if (mode === 'register' && password !== confirmPassword) {
      setError('Passwords do not match');
      return;
    }

    setLoading(true);

    try {
      if (mode === 'register') {
        const response = await apiClient.post('/user/register', {
          username: nextUsername,
          password,
        });
        unwrapApiData(response.data, 'Registration failed');
        await signIn(nextUsername);
        toast.success('Account created');
      } else {
        await signIn(nextUsername);
        toast.success('Signed in');
      }
      navigate('/dashboard');
    } catch (err: unknown) {
      const message = getApiErrorMessage(err, 'Network error');
      setError(message);
      toast.error(message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-background px-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>{mode === 'login' ? 'Login' : 'Create account'}</CardTitle>
          <CardDescription>
            {mode === 'login' ? 'Sign in to your One API account' : 'Register with a username and password'}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="username">Username</Label>
              <Input
                id="username"
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
                autoFocus
                autoComplete="username"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="password">Password</Label>
              <Input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                minLength={mode === 'register' ? 8 : undefined}
                autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
              />
            </div>
            {mode === 'register' && (
              <div className="space-y-2">
                <Label htmlFor="confirm-password">Confirm password</Label>
                <Input
                  id="confirm-password"
                  type="password"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  required
                  minLength={8}
                  autoComplete="new-password"
                />
              </div>
            )}
            {error && <p className="text-sm text-destructive">{error}</p>}
            <Button type="submit" className="w-full" disabled={loading}>
              {loading ? (mode === 'login' ? 'Signing in...' : 'Creating account...') : mode === 'login' ? 'Sign in' : 'Create account'}
            </Button>
            <Button
              type="button"
              variant="ghost"
              className="w-full"
              disabled={loading}
              onClick={() => {
                setMode((current) => {
                  const nextMode = current === 'login' ? 'register' : 'login';
                  navigate(nextMode === 'register' ? '/register' : '/login', { replace: true });
                  return nextMode;
                });
                setError('');
                setConfirmPassword('');
              }}
            >
              {mode === 'login' ? 'Create a new account' : 'Back to sign in'}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
