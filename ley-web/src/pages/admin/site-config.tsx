import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Save } from "lucide-react";
import { useSiteConfig, useUpdateSiteConfig, type SiteConfig } from "@/hooks/use-site";
import { Button } from "@/components/ui/Button";
import { Card, CardHeader, CardTitle } from "@/components/ui/Card";
import { Field, Input, Textarea } from "@/components/ui/Input";
import { Skeleton } from "@/components/ui/Skeleton";

const empty: SiteConfig = {
  site_title: "",
  site_subtitle: "",
  site_description: "",
  site_logo: "",
  site_favicon: "",
  seo_keywords: "",
  seo_description: "",
  social_github: "",
  social_twitter: "",
  social_email: "",
  footer_text: "",
  icp_number: "",
  enable_likes: false,
};

function Toggle({ checked, onChange }: { checked: boolean; onChange: (v: boolean) => void }) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      onClick={() => onChange(!checked)}
      className={`relative h-6 w-11 rounded-full transition-colors ${checked ? "bg-[var(--accent)]" : "bg-[var(--border)]"}`}
    >
      <span
        className={`absolute top-0.5 size-5 rounded-full bg-white shadow-sm transition-all ${checked ? "left-[22px]" : "left-0.5"}`}
      />
    </button>
  );
}

/** 后台站点配置：基础信息 / SEO / 社交链接 / 开关项 */
export default function SiteConfigPage() {
  const { data, isLoading } = useSiteConfig();
  const update = useUpdateSiteConfig();
  const [form, setForm] = useState<SiteConfig>(empty);

  useEffect(() => {
    if (data?.config) setForm({ ...empty, ...data.config });
  }, [data]);

  const set = <K extends keyof SiteConfig>(key: K, value: SiteConfig[K]) =>
    setForm((f) => ({ ...f, [key]: value }));

  const onSave = async () => {
    try {
      await update.mutateAsync(form);
      toast.success("站点配置已保存");
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "保存失败");
    }
  };

  if (isLoading) {
    return (
      <div className="max-w-2xl space-y-4">
        <Skeleton className="h-10 w-48" />
        <Skeleton className="h-96" />
      </div>
    );
  }

  return (
    <div className="max-w-2xl">
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold text-[var(--text-primary)]">站点配置</h1>
        <Button onClick={onSave} disabled={update.isPending}>
          <Save className="size-4" /> {update.isPending ? "保存中…" : "保存配置"}
        </Button>
      </div>

      {/* 基础信息 */}
      <Card className="p-6">
        <CardHeader>
          <CardTitle>基础信息</CardTitle>
        </CardHeader>
        <div className="space-y-5">
          <Field label="站点标题">
            <Input value={form.site_title} onChange={(e) => set("site_title", e.target.value)} placeholder="Ley" />
          </Field>
          <Field label="站点副标题">
            <Input value={form.site_subtitle} onChange={(e) => set("site_subtitle", e.target.value)} />
          </Field>
          <Field label="站点描述">
            <Textarea value={form.site_description} onChange={(e) => set("site_description", e.target.value)} rows={3} />
          </Field>
          <div className="grid gap-4 sm:grid-cols-2">
            <Field label="Logo URL">
              <Input value={form.site_logo} onChange={(e) => set("site_logo", e.target.value)} placeholder="https://…" />
            </Field>
            <Field label="Favicon URL">
              <Input value={form.site_favicon} onChange={(e) => set("site_favicon", e.target.value)} placeholder="https://…" />
            </Field>
          </div>
        </div>
      </Card>

      {/* SEO */}
      <Card className="mt-6 p-6">
        <CardHeader>
          <CardTitle>SEO</CardTitle>
        </CardHeader>
        <div className="space-y-5">
          <Field label="关键词" hint="逗号分隔">
            <Input value={form.seo_keywords} onChange={(e) => set("seo_keywords", e.target.value)} placeholder="博客, Go, React" />
          </Field>
          <Field label="搜索描述">
            <Textarea value={form.seo_description} onChange={(e) => set("seo_description", e.target.value)} rows={2} />
          </Field>
        </div>
      </Card>

      {/* 社交链接 */}
      <Card className="mt-6 p-6">
        <CardHeader>
          <CardTitle>社交链接</CardTitle>
        </CardHeader>
        <div className="space-y-5">
          <Field label="GitHub" hint="用户名或完整链接">
            <Input value={form.social_github} onChange={(e) => set("social_github", e.target.value)} placeholder="username" />
          </Field>
          <Field label="Twitter / X" hint="用户名或完整链接">
            <Input value={form.social_twitter} onChange={(e) => set("social_twitter", e.target.value)} placeholder="username" />
          </Field>
          <Field label="联系邮箱">
            <Input value={form.social_email} onChange={(e) => set("social_email", e.target.value)} placeholder="you@example.com" />
          </Field>
        </div>
      </Card>

      {/* 页脚与开关 */}
      <Card className="mt-6 p-6">
        <CardHeader>
          <CardTitle>页脚与开关</CardTitle>
        </CardHeader>
        <div className="space-y-5">
          <Field label="页脚文案">
            <Input value={form.footer_text} onChange={(e) => set("footer_text", e.target.value)} placeholder="© 2026 Ley" />
          </Field>
          <Field label="ICP 备案号">
            <Input value={form.icp_number} onChange={(e) => set("icp_number", e.target.value)} placeholder="京ICP备00000000号" />
          </Field>
          <div className="flex items-center justify-between rounded-xl bg-[var(--bg-hover)] px-4 py-3">
            <div>
              <p className="text-sm font-medium text-[var(--text-primary)]">启用点赞</p>
              <p className="text-xs text-[var(--text-tertiary)]">关闭后文章页不显示点赞按钮</p>
            </div>
            <Toggle checked={form.enable_likes} onChange={(v) => set("enable_likes", v)} />
          </div>
        </div>
      </Card>
    </div>
  );
}
