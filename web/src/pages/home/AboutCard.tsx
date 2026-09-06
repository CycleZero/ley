import { Skeleton } from "@/components/ui/Skeleton";
import { useSiteConfig } from "@/hooks/use-site";
import { BentoCard } from "./BentoCard";

const FALLBACK_NAME = "未命名";
const FALLBACK_ROLE = "个人博客";
const FALLBACK_BIO = "在这里写下你的故事。";

/**
 * 关于我卡：46px 渐变头像（蓝→紫→粉，--lav 仅此一处 ≤46px 允许）、
 * 名字/简介/角色 mono 小字 —— 全部来自 useSiteConfig，无配置时用占位。
 */
export function AboutCard() {
  const { data, isLoading } = useSiteConfig();
  const cfg = data?.config;

  const name = cfg?.site_title?.trim() || FALLBACK_NAME;
  const role = cfg?.site_subtitle?.trim() || FALLBACK_ROLE;
  const bio = cfg?.site_description?.trim() || FALLBACK_BIO;

  return (
    <BentoCard label="关于我">
      {isLoading ? (
        <div className="flex flex-col items-center gap-2 py-2">
          <Skeleton className="size-[46px] rounded-full" />
          <Skeleton className="h-4 w-24" />
          <Skeleton className="h-3 w-40" />
        </div>
      ) : (
        <div className="flex flex-col items-center text-center">
          <div className="mb-2.5 grid size-[46px] place-items-center rounded-full bg-[linear-gradient(140deg,#57a2f2_10%,#9f9bf0_60%,#f2b3c9_100%)] text-[19px] font-extrabold text-white shadow-[0_0_0_3px_rgba(255,255,255,0.95),0_0_22px_rgba(167,139,250,0.45)]">
            {name.trim()[0] ?? "?"}
          </div>
          <div className="text-[16px] font-bold text-[var(--ink)]">{name}</div>
          <div className="mt-[5px] font-mono text-[9.5px] uppercase tracking-[0.18em] text-[var(--ink-3)]">
            {role}
          </div>
          <p className="mt-2.5 text-[12.5px] leading-[1.8] text-[var(--ink-2)]">{bio}</p>
        </div>
      )}
    </BentoCard>
  );
}
