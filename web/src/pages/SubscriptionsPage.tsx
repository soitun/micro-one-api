import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/lib/api';
import { EmptyState } from '@/components/EmptyState';
import { Skeleton } from '@/components/ui/skeleton';
import {
  SubscriptionProgressCard,
  type SubscriptionProgressData,
} from '@/components/SubscriptionProgress';

export function SubscriptionsPage() {
  const userId = localStorage.getItem('userId');

  const { data: progress, isLoading } = useQuery({
    queryKey: ['my-subscription-progress', userId],
    enabled: !!userId && userId !== '0',
    queryFn: async () => {
      const res = await apiClient.get(`/v1/subscriptions/progress?user_id=${userId}`);
      if (res.data?.success === false) return null;
      return (res.data?.data as SubscriptionProgressData | null) ?? null;
    },
  });

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-semibold">我的订阅</h2>
        <p className="mt-1 text-sm text-muted-foreground">查看当前订阅用量与到期时间。购买套餐请前往「充值 / 订阅」。</p>
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
      ) : progress ? (
        <SubscriptionProgressCard progress={progress} title="当前订阅" />
      ) : (
        <EmptyState
          title="暂无活跃订阅"
          description="你当前没有生效中的订阅，可前往「充值 / 订阅」选择套餐购买。"
        />
      )}
    </div>
  );
}
