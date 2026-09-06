import { cloneElement, forwardRef, isValidElement, useId, type InputHTMLAttributes, type ReactElement, type SelectHTMLAttributes, type TextareaHTMLAttributes } from "react";
import { cn } from "@/lib/utils";

const baseField =
  "w-full rounded-xl border border-[var(--border)] bg-[var(--bg-input)] px-3 text-sm text-[var(--text-primary)] placeholder:text-[var(--text-tertiary)] transition focus:outline-none focus:border-[var(--accent)] focus:ring-2 focus:ring-[var(--accent)]/30 disabled:opacity-50 disabled:cursor-not-allowed";

export const Input = forwardRef<HTMLInputElement, InputHTMLAttributes<HTMLInputElement>>(
  ({ className, ...props }, ref) => (
    <input ref={ref} className={cn(baseField, "h-9", className)} {...props} />
  ),
);
Input.displayName = "Input";

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaHTMLAttributes<HTMLTextAreaElement>>(
  ({ className, ...props }, ref) => (
    <textarea ref={ref} className={cn(baseField, "min-h-24 py-2 leading-relaxed resize-y", className)} {...props} />
  ),
);
Textarea.displayName = "Textarea";

export const Select = forwardRef<HTMLSelectElement, SelectHTMLAttributes<HTMLSelectElement>>(
  ({ className, children, ...props }, ref) => (
    <select ref={ref} className={cn(baseField, "h-9 pr-8 appearance-none bg-no-repeat bg-[right_0.6rem_center]", className)}
      style={{
        // chevron 颜色 = --ink-tertiary (#8aa8c4)：SVG data URI 无法引用页面 CSS 变量，故用 token 的解析值并注明来源
        backgroundImage:
          "url(\"data:image/svg+xml;charset=utf-8,%3Csvg xmlns='http://www.w3.org/2000/svg' width='16' height='16' viewBox='0 0 24 24' fill='none' stroke='%238aa8c4' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='m6 9 6 6 6-6'/%3E%3C/svg%3E\")",
      }}
      {...props}
    >
      {children}
    </select>
  ),
);
Select.displayName = "Select";

export function Label({ className, children, ...props }: React.LabelHTMLAttributes<HTMLLabelElement>) {
  return (
    <label className={cn("mb-1.5 block text-sm font-medium text-[var(--text-primary)]", className)} {...props}>
      {children}
    </label>
  );
}

/** 带标签 + 错误提示的表单字段包装（Label 通过 useId 与控件关联） */
export function Field({
  label,
  error,
  hint,
  children,
}: {
  label?: string;
  error?: string;
  hint?: string;
  children: React.ReactNode;
}) {
  const fieldId = useId();
  return (
    <div>
      {label && <Label htmlFor={fieldId}>{label}</Label>}
      {/* 注入 id 使 label 可点击聚焦、可被 getByLabelText 查询（已有 id 则保留） */}
      {isValidElement(children)
        ? cloneElement(children as ReactElement<{ id?: string }>, {
            id: (children.props as { id?: string } | undefined)?.id ?? fieldId,
          })
        : children}
      {error ? (
        <p className="mt-1.5 text-xs text-red-500">{error}</p>
      ) : hint ? (
        <p className="mt-1.5 text-xs text-[var(--text-tertiary)]">{hint}</p>
      ) : null}
    </div>
  );
}
