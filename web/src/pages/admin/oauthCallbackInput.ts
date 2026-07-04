export interface OAuthCallbackInput {
  code: string;
  state?: string;
}

// Extract the authorization code/state from a raw code, a raw query string, or
// a full callback URL such as `http://localhost:1455/auth/callback?code=xxx&state=yyy`.
export function parseOAuthCallbackInput(input: string): OAuthCallbackInput {
  const trimmed = input.trim().replaceAll('？', '?');
  if (trimmed.includes('code=')) {
    try {
      const rawURL = trimmed.startsWith('http')
        ? trimmed
        : `http://unused/${trimmed.startsWith('?') ? trimmed : `?${trimmed.split('?').pop() || ''}`}`;
      const url = new URL(rawURL);
      const code = url.searchParams.get('code');
      if (code) {
        return { code, state: url.searchParams.get('state') || undefined };
      }
    } catch {
      const match = trimmed.match(/[?&]code=([^&]+)/);
      const stateMatch = trimmed.match(/[?&]state=([^&]+)/);
      if (match) {
        return {
          code: decodeURIComponent(match[1]),
          state: stateMatch ? decodeURIComponent(stateMatch[1]) : undefined,
        };
      }
    }
  }
  return { code: trimmed };
}
