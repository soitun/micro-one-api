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

interface LedgerLog {
  id?: string;
  type: string;
  amount: number;
  balance_after: number;
  reference_id?: string;
  remark?: string;
  created_at: number;
}

interface LedgerLogData {
  items?: LedgerLog[];
  logs?: LedgerLog[];
  total?: number;
}

const ORDER_TYPES = new Set(['recharge', 'redeem', 'refund']);

const TYPE_NAMES: Record<string, string> = {
  recharge: '充值',
  redeem: '兑换',
  refund: '退款',
};

function normalizeLogs(data: LedgerLog[] | LedgerLogData): LedgerLog[] {
  if (Array.isArray(data)) return data;
  if (Array.isArray(data?.items)) return data.items;
  if (Array.isArray(data?.logs)) return data.logs;
  return [];
}

function formatQuota(value: number) {
  return (value / 500000).toFixed(4);
}

function formatDate(value: number) {
  if (!value) return '-';
  return new Date(value * 1000).toLocaleString();
}

export function OrdersPage() {
  const [page, setPage] = useState(1);
  const [type, setType] = useState('');
  const pageSize = 20;

  const { data, isLoading } = useQuery({
    queryKey: ['user-orders', page, type],
    queryFn: async () => {
      const params = new URLSearchParams({
        page: String(page),
        page_size: String(pageSize),
      });
      if (type) params.set('type', type);
      const res = await apiClient.get(`/user/logs?${params}`);
      const payload = unwrapApiData<LedgerLog[] | LedgerLogData>(res.data);
      const logs = normalizeLogs(payload).filter((log) => ORDER_TYPES.has(log.type));
      return {
        logs,
        total: Array.isArray(payload) ? logs.length : payload.total,
      };
    },
  });

  const logs = data?.logs ?? [];

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 className="text-2xl font-black tracking-normal text-slate-950 dark:text-white">我的订单</h2>
          <p className="mt-1 text-sm font-medium text-slate-500 dark:text-slate-400">
            展示充值、兑换和退款相关记录。
          </p>
        </div>
        <select
          value={type}
          onChange={(event) => {
            setType(event.target.value);
            setPage(1);
          }}
          className="h-10 rounded-lg border border-slate-200 bg-white px-3 text-sm font-semibold dark:border-white/10 dark:bg-background"
        >
          <option value="">全部订单</option>
          <option value="recharge">充值记录</option>
          <option value="redeem">兑换记录</option>
          <option value="refund">退款记录</option>
        </select>
      </div>

      {isLoading ? (
        <TableSkeleton columns={['类型', '金额', '余额', '关联 ID', '备注', '时间']} rows={8} />
      ) : logs.length === 0 ? (
        <EmptyState title="暂无订单记录" description="充值、兑换或退款后会显示在这里。" />
      ) : (
        <>
          <div className="overflow-x-auto rounded-lg bg-white shadow-sm ring-1 ring-slate-200 dark:bg-card dark:ring-white/10">
            <Table>
              <TableHeader>
                <TableRow className="bg-slate-50 hover:bg-slate-50 dark:bg-white/5">
                  <TableHead>类型</TableHead>
                  <TableHead>金额</TableHead>
                  <TableHead>余额</TableHead>
                  <TableHead>关联 ID</TableHead>
                  <TableHead>备注</TableHead>
                  <TableHead>时间</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {logs.map((log, index) => (
                  <TableRow key={`${log.id || log.reference_id || log.created_at}-${index}`}>
                    <TableCell>
                      <span className="inline-flex rounded-md bg-emerald-100 px-2 py-1 text-xs font-bold text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300">
                        {TYPE_NAMES[log.type] || log.type || '-'}
                      </span>
                    </TableCell>
                    <TableCell className="font-semibold">{formatQuota(log.amount ?? 0)}</TableCell>
                    <TableCell>{formatQuota(log.balance_after ?? 0)}</TableCell>
                    <TableCell className="font-mono text-xs">{log.reference_id || '-'}</TableCell>
                    <TableCell className="max-w-sm truncate">{log.remark || '-'}</TableCell>
                    <TableCell>{formatDate(log.created_at)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>

          <div className="flex items-center justify-end gap-2">
            <Button variant="outline" size="sm" onClick={() => setPage((value) => Math.max(1, value - 1))} disabled={page === 1}>
              上一页
            </Button>
            <span className="min-w-14 text-center text-sm text-muted-foreground">第 {page} 页</span>
            <Button variant="outline" size="sm" onClick={() => setPage((value) => value + 1)} disabled={logs.length < pageSize}>
              下一页
            </Button>
          </div>
        </>
      )}
    </div>
  );
}
