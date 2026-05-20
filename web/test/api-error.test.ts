import assert from 'node:assert/strict';
import { getApiErrorMessage } from '../src/lib/api-error';

assert.equal(getApiErrorMessage({ response: { data: { message: 'bad token' } } }), 'bad token');
assert.equal(getApiErrorMessage({ response: { data: { error: 'missing option' } } }), 'missing option');
assert.equal(getApiErrorMessage({ message: 'Network Error' }), 'Network Error');
assert.equal(getApiErrorMessage(null, 'Fallback'), 'Fallback');
