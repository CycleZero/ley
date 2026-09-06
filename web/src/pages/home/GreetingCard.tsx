import { useEffect, useState } from "react";
import { Moon, Sun } from "lucide-react";
import { BentoCard } from "./BentoCard";

const WEEKDAYS = ["日", "一", "二", "三", "四", "五", "六"] as const;

const pad = (n: number) => String(n).padStart(2, "0");

/** 按小时问候：6-11 早上好 / 12-17 下午好 / 其余晚上好；白天太阳・夜晚月亮 */
function greetingFor(hour: number): { text: string; Icon: typeof Sun } {
  if (hour >= 6 && hour < 12) return { text: "早上好", Icon: Sun };
  if (hour >= 12 && hour < 18) return { text: "下午好", Icon: Sun };
  return { text: "晚上好", Icon: Moon };
}

/**
 * 问候卡：按小时问候 + mono 日期/时钟（客户端实时，YYYY.MM.DD 星期 + HH:MM）。
 * 唯一允许的 1 处角标装饰：左上角 L 形括号（参照 v4 .corners，只保留一枚）。
 * @param now 可注入固定时间（测试用）；默认客户端实时时钟
 */
export function GreetingCard({ now }: { now?: Date }) {
  const [time, setTime] = useState<Date>(() => now ?? new Date());

  useEffect(() => {
    if (now) return; // 固定时间模式（测试）跳过实时 tick
    const timer = setInterval(() => setTime(new Date()), 1000);
    return () => clearInterval(timer);
  }, [now]);

  const { text, Icon } = greetingFor(time.getHours());
  const dateLine = `${time.getFullYear()}.${pad(time.getMonth() + 1)}.${pad(time.getDate())} 星期${WEEKDAYS[time.getDay()]}`;
  const clock = `${pad(time.getHours())}:${pad(time.getMinutes())}`;

  return (
    <BentoCard label="问候" className="relative">
      {/* 角标装饰：左上 L 形括号（全局唯一一枚） */}
      <i
        aria-hidden
        className="pointer-events-none absolute left-3 top-3 size-[14px] rounded-tl-[6px] border-l-2 border-t-2 border-[var(--accent-blue)] opacity-60"
      />
      <div className="relative">
        <h2 className="flex items-center gap-2 text-[23px] font-bold tracking-[0.02em] text-[var(--ink)]">
          {text}
          <Icon className="size-[18px] text-[var(--amber)]" aria-hidden />
        </h2>
        <p className="mt-1.5 font-mono text-[12px] tracking-[0.06em] text-[var(--ink-2)]">{dateLine}</p>
        <div className="mt-3 flex items-baseline gap-[17px]">
          <span className="bg-gradient-to-b from-[var(--ink)] to-[var(--accent-blue-hover)] bg-clip-text font-mono text-[36px] font-semibold tracking-[0.02em] text-transparent">
            {clock}
          </span>
          <span className="font-mono text-[12px] tracking-[0.2em] text-[var(--accent-blue)]">
            {time.getHours() < 12 ? "AM" : "PM"}
          </span>
        </div>
      </div>
    </BentoCard>
  );
}
