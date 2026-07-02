import { describe, expect, it } from 'vitest';
import { parseOAuthCallbackInput } from './OAuthBindDialog';

describe('parseOAuthCallbackInput', () => {
  it('accepts a raw code', () => {
    expect(parseOAuthCallbackInput('code-123')).toEqual({ code: 'code-123' });
  });

  it('extracts code and state from a full localhost callback URL', () => {
    expect(
      parseOAuthCallbackInput('http://localhost:1455/auth/callback?code=code-123&state=state-456'),
    ).toEqual({ code: 'code-123', state: 'state-456' });
  });

  it('extracts code and state from a raw query string', () => {
    expect(parseOAuthCallbackInput('code=code-123&state=state-456')).toEqual({
      code: 'code-123',
      state: 'state-456',
    });
  });

  it('accepts a full-width question mark in callback URLs', () => {
    expect(
      parseOAuthCallbackInput('http://localhost:1455/auth/callback？code=code-123&state=state-456'),
    ).toEqual({ code: 'code-123', state: 'state-456' });
  });
});
