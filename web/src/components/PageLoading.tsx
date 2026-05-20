import { Loader2 } from 'lucide-react';

export function PageLoading() {
  return (
    <div className="flex min-h-80 items-center justify-center text-muted-foreground">
      <Loader2 className="mr-2 size-4 animate-spin" />
      Loading...
    </div>
  );
}
