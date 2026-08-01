import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { useAuthStore } from "@/stores/auth";
import { Button } from "@/components/ui/Button";
import { Field, Input } from "@/components/ui/Input";

const schema = z.object({
  account: z.string().min(1, "请输入用户名或邮箱"),
  password: z.string().min(6, "密码至少 6 位"),
});

type FormValues = z.infer<typeof schema>;

export default function LoginPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const login = useAuthStore((s) => s.login);
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({ resolver: zodResolver(schema) });

  const redirect = (location.state as { from?: string } | null)?.from ?? "/";

  const onSubmit = async (values: FormValues) => {
    try {
      await login(values.account, values.password);
      toast.success("登录成功");
      navigate(redirect, { replace: true });
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "登录失败，请稍后重试");
    }
  };

  return (
    <div>
      <div className="mb-8 text-center">
        <h2 className="text-xl font-bold text-[var(--text-primary)]">欢迎回来</h2>
        <p className="mt-1 text-sm text-[var(--text-secondary)]">登录 Ley 账号</p>
      </div>

      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4" noValidate>
        <Field label="用户名 / 邮箱" error={errors.account?.message}>
          <Input
            placeholder="请输入用户名或邮箱"
            autoComplete="username"
            {...register("account")}
          />
        </Field>
        <Field label="密码" error={errors.password?.message}>
          <Input
            type="password"
            placeholder="请输入密码"
            autoComplete="current-password"
            {...register("password")}
          />
        </Field>
        <Button type="submit" className="w-full" disabled={isSubmitting}>
          {isSubmitting ? "登录中…" : "登 录"}
        </Button>
      </form>

      <div className="mt-6 flex items-center justify-between text-sm">
        <Link to="/" className="text-[var(--text-tertiary)] hover:text-[var(--text-primary)] transition-colors">
          返回首页
        </Link>
        <Link
          to="/register"
          state={{ from: redirect }}
          className="text-[var(--accent)] hover:underline"
        >
          还没有账号？注册
        </Link>
      </div>
    </div>
  );
}
