import { cn } from "@/lib/utils";
import { BentoCard } from "./BentoCard";

const WEEKDAY_LABELS = ["一", "二", "三", "四", "五", "六", "日"] as const;
const pad = (n: number) => String(n).padStart(2, "0");

/**
 * 日历卡：当前月网格（周一开头），mono 星期行，今日蓝色圆环高亮。
 * @param now 可注入固定时间（测试用）；默认客户端当前时间
 */
export function CalendarCard({ now }: { now?: Date }) {
  const today = now ?? new Date();
  const year = today.getFullYear();
  const month = today.getMonth();
  const blanks = (new Date(year, month, 1).getDay() + 6) % 7; // 周一开头的首日偏移
  const daysInMonth = new Date(year, month + 1, 0).getDate();
  const todayDate = today.getDate();
  const isToday = (d: number) => d === todayDate;

  return (
    <BentoCard label="日历" action={`${year}.${pad(month + 1)}`}>
      <div className="font-mono text-[13px] font-semibold text-[var(--ink)]">{month + 1}月</div>
      <div className="mt-2 grid grid-cols-7 gap-[2px] text-center">
        {WEEKDAY_LABELS.map((w) => (
          <span key={w} className="py-[2px] font-mono text-[9px] text-[var(--ink-3)]">
            {w}
          </span>
        ))}
      </div>
      <div className="mt-1 grid grid-cols-7 gap-[2px] text-center">
        {Array.from({ length: blanks }).map((_, i) => (
          <span key={`blank-${i}`} className="invisible" />
        ))}
        {Array.from({ length: daysInMonth }).map((_, i) => {
          const d = i + 1;
          const marking = isToday(d);
          return (
            <span
              key={d}
              data-today={marking ? "true" : undefined}
              className={cn(
                "rounded-[9px] py-[4.5px] font-mono text-[11.5px] text-[var(--ink-2)] transition-colors hover:bg-[rgba(59,130,246,0.1)]",
                marking &&
                  "bg-[var(--accent-subtle)] font-bold text-[var(--accent-blue)] shadow-[0_0_0_2px_var(--accent-blue)]",
              )}
            >
              {d}
            </span>
          );
        })}
      </div>
    </BentoCard>
  );
}
