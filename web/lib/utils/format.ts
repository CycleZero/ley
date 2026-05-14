/**
 * format.ts — 通用格式化工具函数
 */

/**
 * 将 ISO 8601 时间字符串格式化为中文日期
 * 例如 "2026-05-13T10:30:00Z" → "2026年5月13日"
 */
export function formatDate(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (isNaN(d.getTime())) return iso
  return `${d.getFullYear()}年${d.getMonth() + 1}月${d.getDate()}日`
}

/**
 * 格式化阅读时长估算（中文文章约 300 字/分钟）
 */
export function formatReadingTime(content: string): string {
  const len = content.length
  const minutes = Math.max(1, Math.round(len / 300))
  return `${minutes} 分钟`
}

/**
 * 数字缩写（1.2k / 3.5w）
 */
export function formatCount(n: number): string {
  if (n >= 10000) return `${(n / 10000).toFixed(1)}w`
  if (n >= 1000) return `${(n / 1000).toFixed(1)}k`
  return String(n)
}
