import type { SortDirection } from './table-utils';

interface BuildAdminListParamsOptions {
  page: number;
  pageSize: number;
  search?: string;
  sortKey?: string | null;
  sortDirection?: SortDirection;
  filters?: Record<string, string | number | null | undefined>;
}

export function buildAdminListParams({
  page,
  pageSize,
  search,
  sortKey,
  sortDirection,
  filters = {},
}: BuildAdminListParamsOptions) {
  const params = new URLSearchParams();
  params.set('page', String(page));
  params.set('page_size', String(pageSize));
  if (search?.trim()) params.set('keyword', search.trim());
  if (sortKey && sortDirection) {
    params.set('sort', sortKey);
    params.set('order', sortDirection);
  }
  for (const [key, value] of Object.entries(filters)) {
    if (value !== null && value !== undefined && value !== '') {
      params.set(key, String(value));
    }
  }
  return params;
}
