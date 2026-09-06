import type { ReactNode } from "react";
import { cn } from "@/lib/utils";
import { HomeHero } from "./home/HomeHero";
import { GreetingCard } from "./home/GreetingCard";
import { StatsCard } from "./home/StatsCard";
import { SocialCard } from "./home/SocialCard";
import { RecentPostsCard } from "./home/RecentPostsCard";
import { CalendarCard } from "./home/CalendarCard";
import { TagCloudCard } from "./home/TagCloudCard";
import { AboutCard } from "./home/AboutCard";

/**
 * Bento 网格单元格（Wave D T13）：统一承载 span 断点策略 ——
 * 默认 12 列中的指定 span；<1120px 收为 6 列；<768px 单列。
 * 各面板组件不感知 span，由本页统一配置。
 */
function Cell({ className, children }: { className?: string; children: ReactNode }) {
  return (
    <div className={cn(className, "max-[1120px]:col-span-6 max-[767px]:col-span-12")}>{children}</div>
  );
}

/**
 * 首页（Wave D T13 收官）：Hero + Bento 信息面板，纯组装式。
 * 数据由各面板内部 hooks 拉取（T12），本页不重复拉取；
 * 旧的文章无限流/分类过滤已迁至 /articles 路由。
 * 布局对齐 /tmp/opencode/mock-d1-blue-tech/index-v4.html .bento：
 *   行1: 问候(4) + 站点统计(4) + 找到我(4)
 *   行2: 最近文章(6) + 日历(3) + 标签云(3)
 *   行3: 关于我(4) 居中收尾
 */
export default function HomePage() {
  return (
    <div className="pb-4 px-2.5">
      <HomeHero />

      <div className="grid grid-cols-12 gap-3.5">
        <Cell className="col-span-4">
          <GreetingCard />
        </Cell>
        <Cell className="col-span-4">
          <StatsCard />
        </Cell>
        <Cell className="col-span-4">
          <SocialCard />
        </Cell>
        <Cell className="col-span-6">
          <RecentPostsCard />
        </Cell>
        <Cell className="col-span-3">
          <CalendarCard />
        </Cell>
        <Cell className="col-span-3">
          <TagCloudCard />
        </Cell>
        <Cell className="col-span-4 col-start-5 max-[1120px]:col-start-auto">
          <AboutCard />
        </Cell>
      </div>
    </div>
  );
}
