<!--
  登录 / 注册页

  使用 blank 布局，全屏居中。
  支持 Tab 切换登录与注册，日式简约表单风格。
-->
<template>
  <div class="w-full max-w-sm mx-auto py-16">
    <!-- 站点标识 -->
    <FadeIn direction="up" :delay="0">
      <div class="text-center mb-10">
        <NuxtLink to="/" class="inline-block">
          <h1 class="text-2xl font-serif-jp font-semibold tracking-widest text-heading">
            {{ siteTitle }}
          </h1>
        </NuxtLink>
        <p class="mt-2 text-sm text-placeholder tracking-wide">
          {{ siteSubtitle }}
        </p>
      </div>
    </FadeIn>

    <!-- Tab 切换 -->
    <FadeIn direction="up" :delay="100">
      <div class="flex items-center justify-center mb-8">
        <button
          class="relative px-4 py-2 text-sm tracking-wider transition-colors duration-300"
          :class="mode === 'login' ? 'text-heading font-medium' : 'text-placeholder hover:text-muted'"
          @click="switchMode('login')"
        >
          登录
          <span
            v-if="mode === 'login'"
            class="absolute bottom-0 left-1/2 -translate-x-1/2 w-6 h-px bg-accent"
          />
        </button>
        <div class="w-px h-4 bg-subtle mx-2" />
        <button
          class="relative px-4 py-2 text-sm tracking-wider transition-colors duration-300"
          :class="mode === 'register' ? 'text-heading font-medium' : 'text-placeholder hover:text-muted'"
          @click="switchMode('register')"
        >
          注册
          <span
            v-if="mode === 'register'"
            class="absolute bottom-0 left-1/2 -translate-x-1/2 w-6 h-px bg-accent"
          />
        </button>
      </div>
    </FadeIn>

    <!-- 表单 -->
    <FadeIn direction="up" :delay="200">
      <form class="space-y-6" @submit.prevent="handleSubmit">
        <!-- 用户名（注册时显示） -->
        <WasInput
          v-if="mode === 'register'"
          v-model="form.username"
          label="用户名"
          placeholder="请输入用户名"
          autocomplete="username"
          :error="errors.username"
        />

        <!-- 账号 / 邮箱 -->
        <WasInput
          v-model="form.account"
          :label="mode === 'login' ? '账号 / 邮箱' : '邮箱'"
          :placeholder="mode === 'login' ? '用户名或邮箱' : 'your@email.com'"
          autocomplete="username"
          :error="errors.account"
        />

        <!-- 密码 -->
        <WasInput
          v-model="form.password"
          label="密码"
          type="password"
          :autocomplete="mode === 'login' ? 'current-password' : 'new-password'"
          placeholder="请输入密码"
          :error="errors.password"
        />

        <!-- 确认密码（注册时显示） -->
        <WasInput
          v-if="mode === 'register'"
          v-model="form.confirmPassword"
          label="确认密码"
          type="password"
          autocomplete="new-password"
          placeholder="再次输入密码"
          :error="errors.confirmPassword"
        />

        <!-- 提交按钮 -->
        <WasButton
          type="submit"
          variant="primary"
          size="lg"
          class="w-full"
          :loading="submitting"
        >
          {{ mode === 'login' ? '登录' : '注册' }}
        </WasButton>
      </form>
    </FadeIn>

    <!-- 返回首页 -->
    <FadeIn direction="up" :delay="300">
      <div class="mt-8 text-center">
        <NuxtLink
          to="/"
          class="text-sm text-placeholder hover:text-accent transition-colors duration-300"
        >
          ← 返回首页
        </NuxtLink>
      </div>
    </FadeIn>
  </div>
</template>

<script setup lang="ts">
// ---------- 页面配置 ----------

definePageMeta({
  layout: 'blank',
})

// ---------- Store ----------

const auth = useAuthStore()
const siteStore = useSiteStore()

// ---------- 状态 ----------

type Mode = 'login' | 'register'
const mode = ref<Mode>('login')
const submitting = ref(false)

const form = reactive({
  username: '',
  account: '',
  password: '',
  confirmPassword: '',
})

const errors = reactive({
  username: '',
  account: '',
  password: '',
  confirmPassword: '',
})

// ---------- 计算属性 ----------

const siteTitle = computed(() => siteStore.siteTitle)
const siteSubtitle = computed(() => siteStore.siteSubtitle || '记录与思考的空间')

// ---------- 方法 ----------

function switchMode(newMode: Mode) {
  mode.value = newMode
  // 清空错误和表单
  errors.username = ''
  errors.account = ''
  errors.password = ''
  errors.confirmPassword = ''
  form.username = ''
  form.account = ''
  form.password = ''
  form.confirmPassword = ''
}

function validate(): boolean {
  let valid = true
  errors.username = ''
  errors.account = ''
  errors.password = ''
  errors.confirmPassword = ''

  if (mode.value === 'register') {
    if (!form.username.trim()) {
      errors.username = '请输入用户名'
      valid = false
    }
    else if (form.username.trim().length < 2) {
      errors.username = '用户名至少2个字符'
      valid = false
    }

    if (!form.confirmPassword) {
      errors.confirmPassword = '请确认密码'
      valid = false
    }
    else if (form.password !== form.confirmPassword) {
      errors.confirmPassword = '两次输入的密码不一致'
      valid = false
    }
  }

  if (!form.account.trim()) {
    errors.account = mode.value === 'login' ? '请输入账号或邮箱' : '请输入邮箱'
    valid = false
  }

  if (!form.password) {
    errors.password = '请输入密码'
    valid = false
  }
  else if (form.password.length < 6) {
    errors.password = '密码至少6位'
    valid = false
  }

  return valid
}

async function handleSubmit() {
  if (!validate()) return

  submitting.value = true
  try {
    if (mode.value === 'login') {
      await auth.login({
        account: form.account.trim(),
        password: form.password,
      })
    }
    else {
      await auth.register({
        username: form.username.trim(),
        email: form.account.trim(),
        password: form.password,
      })
    }
    // 成功后跳转首页
    await navigateTo('/')
  }
  catch (e: any) {
    // 错误提示由 auth store 的 ui.toast 统一处理
    // 这里可以补充表单级错误提示
    errors.account = e?.message || '请求失败，请重试'
  }
  finally {
    submitting.value = false
  }
}

// ---------- 页面标题 ----------

useHead({
  title: '登录',
})
</script>
