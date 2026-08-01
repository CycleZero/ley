import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { useAuthStore } from "@/stores/auth";
import { Button } from "@/components/ui/Button";
import { Card, CardHeader, CardTitle } from "@/components/ui/Card";
import { Field, Input, Textarea } from "@/components/ui/Input";
import { Badge } from "@/components/ui/Badge";

/** 个人中心：资料展示 + 编辑（RequireAuth 保护） */
export default function ProfilePage() {
  const user = useAuthStore((s) => s.user);
  const updateProfile = useAuthStore((s) => s.updateProfile);
  const logout = useAuthStore((s) => s.logout);
  const navigate = useNavigate();

  const [avatar, setAvatar] = useState(user?.avatar ?? "");
  const [bio, setBio] = useState(user?.bio ?? "");
  const [saving, setSaving] = useState(false);

  // 用户变化时同步表单
  useEffect(() => {
    setAvatar(user?.avatar ?? "");
    setBio(user?.bio ?? "");
  }, [user]);

  if (!user) return null;

  const onSubmit = async () => {
    setSaving(true);
    try {
      await updateProfile({ avatar, bio });
      toast.success("资料已更新");
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "保存失败");
    } finally {
      setSaving(false);
    }
  };

  const onLogout = async () => {
    await logout();
    toast.success("已退出登录");
    navigate("/");
  };

  return (
    <div className="mx-auto max-w-3xl px-6 py-10">
      <h1 className="mb-8 text-2xl font-bold text-[var(--text-primary)]">个人中心</h1>

      {/* 基本信息 */}
      <Card className="p-8">
        <div className="flex items-center gap-6">
          {user.avatar ? (
            <img src={user.avatar} alt="头像" className="size-20 rounded-2xl object-cover" />
          ) : (
            <div className="flex size-20 items-center justify-center rounded-2xl bg-[var(--accent-subtle)] text-3xl font-bold text-[var(--accent)]">
              {user.username?.charAt(0).toUpperCase()}
            </div>
          )}
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <h2 className="truncate text-xl font-semibold text-[var(--text-primary)]">{user.username}</h2>
              {user.role === "admin" && <Badge variant="accent">管理员</Badge>}
            </div>
            <p className="mt-1 text-sm text-[var(--text-secondary)]">{user.email}</p>
            <p className="mt-0.5 text-xs text-[var(--text-tertiary)]">
              {user.bio || "这个人很懒，还没有写简介"}
            </p>
          </div>
        </div>
      </Card>

      {/* 编辑资料 */}
      <Card className="mt-6 p-8">
        <CardHeader>
          <CardTitle>编辑资料</CardTitle>
        </CardHeader>
        <div className="space-y-5">
          <Field label="头像 URL" hint="支持任意图片地址，留空使用默认头像">
            <Input value={avatar} onChange={(e) => setAvatar(e.target.value)} placeholder="https://…" />
          </Field>
          <Field label="个人简介">
            <Textarea
              value={bio}
              onChange={(e) => setBio(e.target.value)}
              placeholder="介绍一下自己吧（200 字以内）"
              maxLength={200}
              rows={4}
            />
          </Field>
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={onLogout}>
              退出登录
            </Button>
            <Button onClick={onSubmit} disabled={saving}>
              {saving ? "保存中…" : "保存"}
            </Button>
          </div>
        </div>
      </Card>
    </div>
  );
}
