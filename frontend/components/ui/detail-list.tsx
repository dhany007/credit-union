import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

export function DetailList({ rows, className }: {
  rows: readonly (readonly [string, ReactNode])[];
  className?: string;
}) {
  return (
    <dl className={cn("grid gap-y-2 text-sm", className)}>
      {rows.map(([label, value]) => (
        <div key={label} className="grid min-w-0 grid-cols-[minmax(0,40%)_minmax(0,1fr)] gap-x-2">
          <dt className="break-words text-muted-foreground">{label}</dt>
          <dd className="grid min-w-0 grid-cols-[0.5rem_minmax(0,1fr)] gap-x-2">
            <span aria-hidden="true">:</span>
            <div className="min-w-0 whitespace-pre-wrap break-words font-medium">{value}</div>
          </dd>
        </div>
      ))}
    </dl>
  );
}
