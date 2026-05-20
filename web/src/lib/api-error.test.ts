import { describe, expect, it } from 'vitest';
import { getApiErrorMessage } from './api-error';

describe('getApiErrorMessage', () => {
  it('returns backend message fields before generic errors', () => {
    expect(getApiErrorMessage({ response: { data: { message: 'bad token' } } })).toBe('bad token');
    expect(getApiErrorMessage({ response: { data: { error: 'missing option' } } })).toBe('missing option');
  });

  it('falls back to error message or explicit fallback', () => {
    expect(getApiErrorMessage({ message: 'Network Error' })).toBe('Network Error');
    expect(getApiErrorMessage(null, 'Fallback')).toBe('Fallback');
  });
});
