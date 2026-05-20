import { describe, expect, it } from 'vitest';
import { buildAdminListParams } from './admin-table-query';

describe('buildAdminListParams', () => {
  it('serializes pagination search sorting and filters', () => {
    const params = buildAdminListParams({
      page: 2,
      pageSize: 50,
      search: 'alice',
      sortKey: 'username',
      sortDirection: 'asc',
      filters: { status: '1', group: 'default', empty: '' },
    });

    expect(params.toString()).toBe(
      'page=2&page_size=50&keyword=alice&sort=username&order=asc&status=1&group=default',
    );
  });
});
