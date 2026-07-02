import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/lib/api';
import { EmptyState } from '@/components/EmptyState';
import { Skeleton } from '@/components/ui/skeleton';
import {
  SubscriptionProgressCard,
  type SubscriptionProgressData,
} from '@/components/SubscriptionProgress';

// The progress endpoint (/api/v1/subscriptions/progress) returns the current
// user's single active subscription, or success:false when there is none.
export function SubscriptionsPage() {
  const userId = localStorage.getItem('userId');

  const { data, isLoading } = useQuery({
    queryKey: ['my-subscription-progress', userId],
    enabled: !!userId && userId !== '0',
    queryFn: async () => {
      const res = await apiClient.get(`/v1/subscriptions/progress?user_id=${userId}`);
      if (res.data?.success === false) return null;
      return (res.data?.data as SubscriptionProgressData | null) ?? null;
    },
  });

  return (
    <div className="space-y-4">
      <div>
        <h2 className="text-2xl font-semibold">我的订阅</h2>
        <p className="mt-1 text-sm text-muted-foreground">查看当前订阅的配额用量与到期时间。</p>
      </div>

      {isLoading ? (
        <div className="rounded-xl border bg-card p-4">
          <Skeleton className="mb-3 h-5 w-40" />
          <div className="space-y-2">
            <Skeleton className="h-2 w-full" />
            <Skeleton className="h-2 w-full" />
            <Skeleton className="h-2 w-full" />
          </div>
        </div>
      ) : !data ? (
        <EmptyState
          title="暂无活跃订阅"
          description="你当前没有生效中的订阅。如需订阅，请联系管理员分配。"
        />
      ) : (
        <SubscriptionProgressCard progress={data} title="当前订阅" />
      )}
    </div>
  );
}
