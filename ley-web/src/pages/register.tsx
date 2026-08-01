import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { useAuthStore } from "@/stores/auth";
import { Button } from "@/components/ui/Button";
import { Field, Input } from "@/components/ui/Input";

const schema = z
  .object({
    username: z
      .string()
      .min(3, "用户名至少 3 个字符")
      .max(32, "用户名最长 32 个字符")
      .regex(/^[a-zA-Z0-9_-]+$/, "用户名只能包含字母、数字、下划线和连字符"),
    email: z.string().email("邮箱格式不正确"),
    password: z
      .string()
      .min(8, "密码至少 8 位")
      .regex(/[a-z]/, "密码必须包含小写字母")
      .regex(/[A-Z]/, "密码必须包含大写字母")
      .regex(/[0-9]/, "密码必须包含数字"),
    confirm: z.string(),
  })
  .refine((v) => v.password === v.confirm, {
    message: "两次输入的密码不一致",
    path: ["confirm"],
  });

type FormValues = z.infer<typeof schema>;

export default function RegisterPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const register = useAuthStore((s) => s.register);
  const {
    register: registerField,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({ resolver: zodResolver(schema) });

  const redirect = (location.state as { from?: string } | null)?.from ?? "/";

  const onSubmit = async (values: FormValues) => {
    try {
      await register(values.username, values.email, values.password);
      toast.success("注册成功，已自动登录");
      navigate(redirect, { replace: true });
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "注册失败，请稍后重试");
    }
  };

  return (
    <div>
      <div className="mb-8 text-center">
        <h2 className="text-xl font-bold text-[var(--text-primary)]">创建账号</h2>
        <p className="mt-1 text-sm text-[var(--text-secondary)]">加入 Ley，开始记录</p>
      </div>

      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4" noValidate>
        <Field label="用户名" error={errors.username?.message}>
          <Input placeholder="3-32 位字母、数字或下划线" autoComplete="username" {...registerField("username")} />
        </Field>
        <Field label="邮箱" error={errors.email?.message}>
          <Input type="email" placeholder="you@example.com" autoComplete="email" {...registerField("email")} />
        </Field>
        <Field label="密码" error={errors.password?.message} hint="至少 8 位，需包含大写字母、小写字母和数字">
          <Input type="password" placeholder="请输入密码" autoComplete="new-password" {...registerField("password")} />
        </Field>
        <Field label="确认密码" error={errors.confirm?.message}>
          <Input type="password" placeholder="再次输入密码" autoComplete="new-password" {...registerField("confirm")} />
        </Field>
        <Button type="submit" className="w-full" disabled={isSubmitting}>
          {isSubmitting ? "注册中…" : "注 册"}
        </Button>
      </form>

      <div className="mt-6 flex items-center justify-between text-sm">
        <Link to="/" className="text-[var(--text-tertiary)] hover:text-[var(--text-primary)] transition-colors">
          返回首页
        </Link>
        <Link to="/login" state={{ from: redirect }} className="text-[var(--accent)] hover:underline">
          已有账号？登录
        </Link>
      </div>
    </div>
  );
}
