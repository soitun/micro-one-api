import { useQuery } from '@tanstack/react-query';
import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { EmptyState } from '@/components/EmptyState';
import { TableSkeleton } from '@/components/LoadingStates';
import { apiClient } from '@/lib/api';
import { unwrapApiData } from '@/lib/api-response';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';

interface UsageLog {
  id?: string;
  user_id?: string;
  type: string;
  amount: number;
  balance_after: number;
  reference_id?: string;
  remark?: string;
  created_at: number;
}

interface UsageLogData {
  items?: UsageLog[];
  logs?: UsageLog[];
  total?: number;
}

const LOG_TYPE_NAMES: Record<string, string> = {
  redeem: 'Redeem',
  recharge: 'Recharge',
  consume: 'Consume',
  refund: 'Refund',
};

function normalizeLogs(data: UsageLog[] | UsageLogData): UsageLog[] {
  if (Array.isArray(data)) {
    return data;
  }
  if (Array.isArray(data?.items)) {
    return data.items;
  }
  if (Array.isArray(data?.logs)) {
    return data.logs;
  }
  return [];
}

function formatQuota(value: number) {
  return (value / 500000).toFixed(4);
}

function formatDate(value: number) {
  if (!value) {
    return '-';
  }
  return new Date(value * 1000).toLocaleString();
}

export function UsagePage() {
  const [page, setPage] = useState(1);
  const [type, setType] = useState('');
  const pageSize = 20;

  const { data, isLoading } = useQuery({
    queryKey: ['user-usage-logs', page, type],
    queryFn: async () => {
      const params = new URLSearchParams({
        page: String(page),
        page_size: String(pageSize),
      });
      if (type) {
        params.set('type', type);
      }
      const res = await apiClient.get(`/user/logs?${params}`);
      const payload = unwrapApiData<UsageLog[] | UsageLogData>(res.data);
      return {
        logs: normalizeLogs(payload),
        total: Array.isArray(payload) ? payload.length : payload.total,
      };
    },
  });

  const logs = data?.logs ?? [];

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <h2 className="text-2xl font-semibold">Usage</h2>
        <div className="flex items-center gap-2">
          <select
            value={type}
            onChange={(event) => {
              setType(event.target.value);
              setPage(1);
            }}
            className="h-9 rounded-md border bg-background px-3 text-sm"
          >
            <option value="">All Types</option>
            <option value="consume">Consume</option>
            <option value="recharge">Recharge</option>
            <option value="redeem">Redeem</option>
            <option value="refund">Refund</option>
          </select>
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              setType('');
              setPage(1);
            }}
          >
            Clear
          </Button>
        </div>
      </div>

      {isLoading ? (
        <TableSkeleton columns={['Type', 'Amount', 'Balance', 'Reference', 'Remark', 'Time']} rows={8} />
      ) : logs.length === 0 ? (
        <EmptyState title="No usage records" description="Token usage will appear after API requests are processed." />
      ) : (
        <>
          <div className="overflow-x-auto rounded-lg border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Type</TableHead>
                  <TableHead>Amount</TableHead>
                  <TableHead>Balance</TableHead>
                  <TableHead className="hidden md:table-cell">Reference</TableHead>
                  <TableHead className="hidden lg:table-cell">Remark</TableHead>
                  <TableHead>Time</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {logs.map((log, index) => (
                  <TableRow key={`${log.id || log.reference_id || log.created_at}-${index}`}>
                    <TableCell>
                      <span className="inline-flex items-center rounded-full bg-blue-100 px-2 py-1 text-xs font-medium text-blue-800 dark:bg-blue-900 dark:text-blue-200">
                        {LOG_TYPE_NAMES[log.type] || log.type || '-'}
                      </span>
                    </TableCell>
                    <TableCell>{formatQuota(log.amount ?? 0)}</TableCell>
                    <TableCell>{formatQuota(log.balance_after ?? 0)}</TableCell>
                    <TableCell className="hidden font-mono text-xs md:table-cell">{log.reference_id || '-'}</TableCell>
                    <TableCell className="hidden max-w-sm truncate lg:table-cell">{log.remark || '-'}</TableCell>
                    <TableCell>{formatDate(log.created_at)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>

          <div className="flex items-center justify-end gap-2">
            <Button variant="outline" size="sm" onClick={() => setPage((value) => Math.max(1, value - 1))} disabled={page === 1}>
              Previous
            </Button>
            <span className="min-w-14 text-center text-sm text-muted-foreground">Page {page}</span>
            <Button variant="outline" size="sm" onClick={() => setPage((value) => value + 1)} disabled={logs.length < pageSize}>
              Next
            </Button>
          </div>
        </>
      )}
    </div>
  );
}
