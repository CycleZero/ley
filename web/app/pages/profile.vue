<!--
  用户资料页 /profile

  展示当前用户信息，支持编辑资料。
  未登录用户自动跳转到登录页。
-->
<template>
  <div class="max-w-xl mx-auto px-6 py-16">
    <!-- 页面标题 -->
    <FadeIn direction="up" :delay="0">
      <div class="text-center mb-10">
        <h1 class="text-3xl font-serif-jp font-semibold text-heading tracking-widest">
          个人资料
        </h1>
        <div class="mt-4 flex items-center justify-center gap-3">
          <div class="w-12 h-px bg-subtle" />
          <div class="w-1.5 h-1.5 rounded-full bg-accent" />
          <div class="w-12 h-px bg-subtle" />
        </div>
      </div>
    </FadeIn>

    <!-- 未登录提示 -->
    <div v-if="!auth.isLoggedIn" class="text-center py-12">
      <p class="text-muted mb-4">请先登录</p>
      <WasButton variant="primary" @click="navigateTo('/login')">
        去登录
      </WasButton>
    </div>

    <template v-else>
      <!-- 用户卡片 -->
      <FadeIn direction="up" :delay="100">
        <div class="surface-card p-8 mb-8">
          <!-- 头像与基本信息 -->
          <div class="flex items-center gap-4 mb-8 pb-8 border-b border-subtle">
            <div class="w-16 h-16 rounded-full bg-accent-subtle flex items-center justify-center text-2xl font-serif-jp font-medium text-accent flex-shrink-0">
              {{ displayName.charAt(0).toUpperCase() }}
            </div>
            <div>
              <h2 class="text-lg font-serif-jp font-semibold text-heading">
                {{ displayName }}
              </h2>
              <p class="text-sm text-muted mt-1">{{ auth.user?.email }}</p>
              <span class="inline-block mt-2 text-xs px-2 py-0.5 border border-default rounded-sm text-muted">
                {{ roleText }}
              </span>
            </div>
          </div>

          <!-- 编辑表单 -->
          <form class="space-y-6" @submit.prevent="handleSave">
            <WasInput
              v-model="form.avatar"
              label="头像 URL"
              placeholder="https://example.com/avatar.png"
              :error="errors.avatar"
            />

            <WasTextarea
              v-model="form.bio"
              label="个人简介"
              placeholder="介绍一下你自己..."
              rows="3"
              :error="errors.bio"
            />

            <div class="flex items-center justify-between pt-2">
              <span class="text-xs text-placeholder">
                注册于 {{ joinDate }}
              </span>
              <WasButton
                type="submit"
                variant="primary"
                :loading="saving"
              >
                保存修改
              </WasButton>
            </div>
          </form>
        </div>
      </FadeIn>

      <!-- 账户安全 -->
      <FadeIn direction="up" :delay="200">
        <div class="surface-card p-8">
          <h3 class="text-sm font-serif-jp font-semibold text-heading tracking-wider mb-4">
            账户安全
          </h3>
          <div class="space-y-4">
            <div class="flex items-center justify-between py-3 border-b border-subtle">
              <div>
                <p class="text-sm text-body">密码</p>
                <p class="text-xs text-placeholder mt-0.5">定期更换密码可提高安全性</p>
              </div>
              <WasButton variant="secondary" size="sm" disabled>
                修改密码
              </WasButton>
            </div>
            <div class="flex items-center justify-between py-3">
              <div>
                <p class="text-sm text-body">登录状态</p>
                <p class="text-xs text-placeholder mt-0.5">当前已登录</p>
              </div>
              <WasButton variant="outline" size="sm" @click="handleLogout">
                退出登录
              </WasButton>
            </div>
          </div>
        </div>
      </FadeIn>
    </template>
  </div>
</template>

<script setup lang="ts">
definePageMeta({
  middleware: 'auth',
})

const auth = useAuthStore()
const ui = useUiStore()

// ---------- 表单状态 ----------

const form = reactive({
  avatar: auth.user?.avatar || '',
  bio: auth.user?.bio || '',
})

const errors = reactive({
  avatar: '',
  bio: '',
})

const saving = ref(false)

// ---------- 计算属性 ----------

const displayName = computed(() => auth.user?.username || '未知用户')

const roleText = computed(() => {
  switch (auth.user?.role) {
    case 'admin': return '管理员'
    case 'author': return '作者'
    case 'reader': return '读者'
    default: return '用户'
  }
})

const joinDate = computed(() => {
  if (!auth.user?.createdAt) return ''
  const d = new Date(auth.user.createdAt)
  return `${d.getFullYear()}年${d.getMonth() + 1}月${d.getDate()}日`
})

// 监听 user 变化同步表单（刷新资料后）
watch(() => auth.user, (u) => {
  if (u) {
    form.avatar = u.avatar || ''
    form.bio = u.bio || ''
  }
}, { immediate: true })

// ---------- 方法 ----------

function validate(): boolean {
  errors.avatar = ''
  errors.bio = ''
  let valid = true

  if (form.avatar && !form.avatar.startsWith('http')) {
    errors.avatar = '头像 URL 必须以 http:// 或 https:// 开头'
    valid = false
  }

  if (form.bio && form.bio.length > 200) {
    errors.bio = '简介不能超过 200 字'
    valid = false
  }

  return valid
}

async function handleSave() {
  if (!validate()) return

  saving.value = true
  try {
    await auth.updateProfile({
      avatar: form.avatar || undefined,
      bio: form.bio || undefined,
    })
    ui.toast('资料已保存', 'success')
  }
  catch (e: any) {
    ui.toast(e?.message || '保存失败', 'error')
  }
  finally {
    saving.value = false
  }
}

async function handleLogout() {
  await auth.logout()
  await navigateTo('/')
}

// ---------- 页面标题 ----------

useHead({
  title: '个人资料',
})
</script>
