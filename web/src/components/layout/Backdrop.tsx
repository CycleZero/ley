import { useMemo } from "react";
import type { CSSProperties } from "react";
import { cn } from "@/lib/utils";

/**
 * 全屏背景层：近白微渐变 + 双层科技网格线（视觉基准 v4 的 .bg/.grid-overlay）。
 * - fixed inset-0 / aria-hidden / pointer-events-none：纯装饰，不干扰交互与读屏
 * - 粒子（可选）：白/蓝/青小点漂浮，≤16 个；prefers-reduced-motion 时禁用动画
 */

const PARTICLE_COLORS = ["#ffffff", "#bcd8f7", "#bcd8f7", "#c4f0e8", "#ffffff"] as const;

const GRID_STYLE: CSSProperties = {
  backgroundImage: [
    "repeating-linear-gradient(0deg, rgba(59,130,246,0.04) 0 1px, transparent 1px 46px)",
    "repeating-linear-gradient(90deg, rgba(59,130,246,0.04) 0 1px, transparent 1px 46px)",
  ].join(","),
  WebkitMaskImage: "radial-gradient(125% 105% at 50% 36%, #000 55%, transparent 98%)",
  maskImage: "radial-gradient(125% 105% at 50% 36%, #000 55%, transparent 98%)",
};

const FLOAT_KEYFRAMES = `@keyframes ley-backdrop-float {
  0% { transform: translateY(12vh); opacity: 0 }
  12% { opacity: var(--ley-pc, 0.8) }
  88% { opacity: var(--ley-pc, 0.8) }
  100% { transform: translateY(-92vh); opacity: 0 }
}`;

interface Particle {
  key: number;
  color: string;
  width: number;
  left: number;
  top: number;
  duration: number;
  delay: number;
  opacity: number;
}

function makeParticles(count: number): Particle[] {
  return Array.from({ length: count }, (_, i) => ({
    key: i,
    color: PARTICLE_COLORS[i % PARTICLE_COLORS.length],
    width: Math.round((Math.random() * 1.8 + 1.2) * 10) / 10,
    left: Math.random() * 100,
    top: 100 + Math.random() * 6,
    duration: Math.round((Math.random() * 20 + 16) * 10) / 10,
    delay: -Math.round(Math.random() * 36 * 10) / 10,
    opacity: Math.round((Math.random() * 0.3 + 0.2) * 100) / 100,
  }));
}

export interface BackdropProps {
  /** 是否渲染漂浮粒子（默认 true，≤16 个小点） */
  particles?: boolean;
  className?: string;
}

export function Backdrop({ particles = true, className }: BackdropProps) {
  // 尊重减少动画：粒子仍渲染（静态点），仅禁用漂浮动画
  const reducedMotion =
    typeof window !== "undefined" &&
    window.matchMedia?.("(prefers-reduced-motion: reduce)").matches === true;

  // 视口从宽取值，默认取 1280×800 同一公式（上限 16）
  const particleCount = useMemo(() => {
    if (!particles) return 0;
    const width = Math.max(window.innerWidth, 1280);
    const height = Math.max(window.innerHeight, 800);
    return Math.min(16, Math.round((width * height) / 90_000));
  }, [particles]);

  const dots = useMemo(() => makeParticles(particleCount), [particleCount]);

  return (
    <div
      aria-hidden="true"
      className={cn(
        "pointer-events-none fixed inset-0 -z-10 overflow-hidden",
        "bg-[linear-gradient(165deg,var(--bg-page)_0%,var(--bg-card)_55%,var(--bg-page)_100%)]",
        className,
      )}
    >
      <div className="absolute inset-0" style={GRID_STYLE} />
      {dots.length > 0 && (
        <div className="absolute inset-0">
          <style>{FLOAT_KEYFRAMES}</style>
          {dots.map((p) => (
            <i
              key={p.key}
              className="ley-particle absolute rounded-full"
              style={
                {
                  width: `${p.width}px`,
                  height: `${p.width}px`,
                  left: `${p.left}%`,
                  top: `${p.top}%`,
                  background: p.color,
                  boxShadow:
                    p.color === "#ffffff"
                      ? "0 0 6px 1px rgba(255,255,255,0.7)"
                      : p.color === "#c4f0e8"
                        ? "0 0 7px 1px rgba(94,234,212,0.45)"
                        : "0 0 7px 1px rgba(59,130,246,0.4)",
                  "--ley-pc": p.opacity,
                  animation: reducedMotion ? "none" : "ley-backdrop-float linear infinite",
                  animationDuration: reducedMotion ? undefined : `${p.duration}s`,
                  animationDelay: reducedMotion ? undefined : `${p.delay}s`,
                } as CSSProperties
              }
            />
          ))}
        </div>
      )}
    </div>
  );
}

export default Backdrop;
