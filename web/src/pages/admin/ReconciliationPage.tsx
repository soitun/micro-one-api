import { useQuery } from '@tanstack/react-query';
import { ScaleIcon } from 'lucide-react';
import { useMemo, useState } from 'react';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { EmptyState } from '@/components/EmptyState';
import { TableSkeleton } from '@/components/LoadingStates';
import { AdminPagination } from '@/components/admin/AdminPagination';
import { adminApiClient } from '@/lib/api';
import { unwrapApiData } from '@/lib/api-response';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';

interface ReconciliationDiscrepancy {
  user_id: string;
  expected_quota: number;
  actual_quota: number;
  ledger_net_amount: number;
  frozen_quota: number;
}

interface ReconciliationRun {
  run_id: number;
  run_at: number;
  expired_cleaned: number;
  total_accounts: number;
  total_reservations: number;
  discrepancy_count: number;
  discrepancies?: ReconciliationDiscrepancy[];
}

interface ReconciliationRunsPayload {
  runs?: ReconciliationRun[];
  total?: number;
}

function formatDate(seconds: number) {
  if (!seconds) return '-';
  return new Date(seconds * 1000).toLocaleString();
}

function formatQuota(value: number) {
  const parsed = Number(value ?? 0);
  return Number.isFinite(parsed) ? parsed.toLocaleString() : '0';
}

export function AdminReconciliationPage() {
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [selectedRunId, setSelectedRunId] = useState<number | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ['admin-reconciliation', page, pageSize],
    queryFn: async () => {
      const res = await adminApiClient.get(`/reconciliation?page=${page}&page_size=${pageSize}`);
      return unwrapApiData<ReconciliationRunsPayload>(res.data);
    },
  });

  const runs = useMemo(() => data?.runs ?? [], [data?.runs]);

  const { data: selectedRun, isLoading: isDetailLoading } = useQuery({
    queryKey: ['admin-reconciliation-run', selectedRunId],
    enabled: selectedRunId !== null,
    queryFn: async () => {
      const res = await adminApiClient.get(`/reconciliation/${selectedRunId}`);
      return unwrapApiData<ReconciliationRun>(res.data);
    },
  });

  const discrepancies = selectedRun?.discrepancies ?? [];

  return (
    <div className="space-y-5">
      <div>
        <h2 className="flex items-center gap-2 text-2xl font-black tracking-normal text-slate-950 dark:text-white">
          <ScaleIcon className="size-6" />
          账务对账
        </h2>
        <p className="mt-1 text-sm font-medium text-slate-500 dark:text-slate-400">
          查看历史对账运行记录、清理的过期预留数，以及账户额度与账目不一致的差异明细。
        </p>
      </div>

      {isLoading ? (
        <TableSkeleton columns={['运行 ID', '运行时间', '账户数', '预留数', '清理过期', '差异数', '操作']} rows={8} />
      ) : runs.length === 0 ? (
        <EmptyState title="暂无对账记录" description="触发对账任务后，运行记录会显示在这里。" />
      ) : (
        <>
          <div className="overflow-x-auto rounded-lg bg-white shadow-sm ring-1 ring-slate-200 dark:bg-card dark:ring-white/10">
            <Table className="min-w-[880px]">
              <TableHeader>
                <TableRow className="bg-slate-50 hover:bg-slate-50 dark:bg-white/5">
                  <TableHead>运行 ID</TableHead>
                  <TableHead>运行时间</TableHead>
                  <TableHead>账户数</TableHead>
                  <TableHead>预留数</TableHead>
                  <TableHead>清理过期</TableHead>
                  <TableHead>差异数</TableHead>
                  <TableHead>操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {runs.map((run) => (
                  <TableRow key={run.run_id}>
                    <TableCell className="font-mono text-xs">{run.run_id}</TableCell>
                    <TableCell>{formatDate(run.run_at)}</TableCell>
                    <TableCell>{run.total_accounts}</TableCell>
                    <TableCell>{run.total_reservations}</TableCell>
                    <TableCell>{run.expired_cleaned}</TableCell>
                    <TableCell>
                      <span
                        className={
                          run.discrepancy_count > 0
                            ? 'inline-flex rounded-md bg-red-100 px-2 py-1 text-xs font-bold text-red-700 dark:bg-red-500/15 dark:text-red-300'
                            : 'inline-flex rounded-md bg-emerald-100 px-2 py-1 text-xs font-bold text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300'
                        }
                      >
                        {run.discrepancy_count}
                      </span>
                    </TableCell>
                    <TableCell>
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        aria-label={`查看对账 ${run.run_id}`}
                        onClick={() => setSelectedRunId(run.run_id)}
                      >
                        查看
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>

          <AdminPagination
            page={page}
            pageSize={pageSize}
            hasNextPage={runs.length >= pageSize}
            onPageChange={setPage}
            onPageSizeChange={(size) => {
              setPageSize(size);
              setPage(1);
            }}
          />
        </>
      )}

      <Dialog open={selectedRunId !== null} onOpenChange={(open) => !open && setSelectedRunId(null)}>
        <DialogContent className="sm:max-w-3xl">
          <DialogHeader>
            <DialogTitle>对账差异详情</DialogTitle>
            <DialogDescription>
              账户额度（实际）与账目净额（应有）不一致的记录。差异通常意味着漏记账目或额度被直接改动。
            </DialogDescription>
          </DialogHeader>

          {isDetailLoading ? (
            <TableSkeleton columns={['用户 ID', '应有额度', '实际额度', '账目净额', '冻结额度']} rows={4} />
          ) : discrepancies.length === 0 ? (
            <EmptyState title="无差异" description="本次对账未发现账户额度与账目不一致。" />
          ) : (
            <div className="overflow-x-auto rounded-lg ring-1 ring-slate-200 dark:ring-white/10">
              <Table className="min-w-[640px]">
                <TableHeader>
                  <TableRow className="bg-slate-50 hover:bg-slate-50 dark:bg-white/5">
                    <TableHead>用户 ID</TableHead>
                    <TableHead>应有额度</TableHead>
                    <TableHead>实际额度</TableHead>
                    <TableHead>账目净额</TableHead>
                    <TableHead>冻结额度</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {discrepancies.map((item) => (
                    <TableRow key={item.user_id}>
                      <TableCell className="font-mono text-xs">{item.user_id}</TableCell>
                      <TableCell>{formatQuota(item.expected_quota)}</TableCell>
                      <TableCell className="font-semibold text-red-600 dark:text-red-400">
                        {formatQuota(item.actual_quota)}
                      </TableCell>
                      <TableCell>{formatQuota(item.ledger_net_amount)}</TableCell>
                      <TableCell>{formatQuota(item.frozen_quota)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
