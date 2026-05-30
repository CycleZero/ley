<!--
  站点配置页 /admin/site
-->
<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-serif-jp font-semibold text-heading tracking-wider">
        站点配置
      </h1>
      <WasButton variant="primary" :loading="saving" @click="save">
        保存配置
      </WasButton>
    </div>

    <div v-if="siteStore.loading" class="py-20 flex justify-center">
      <WasLoading />
    </div>

    <div v-else class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <!-- 基本信息 -->
      <div class="surface-card p-6">
        <h2 class="text-sm font-serif-jp font-semibold text-heading tracking-wider mb-4 pb-2 border-b border-subtle">
          基本信息
        </h2>
        <div class="space-y-4">
          <WasInput
            v-model="form.siteTitle"
            label="站点标题"
            placeholder="Ley"
          />
          <WasInput
            v-model="form.siteSubtitle"
            label="站点副标题"
            placeholder="记录与思考的空间"
          />
          <WasTextarea
            v-model="form.siteDescription"
            label="站点描述"
            placeholder="站点简介，用于 SEO 和社交分享"
            rows="3"
          />
          <WasInput
            v-model="form.siteLogo"
            label="Logo URL"
            placeholder="https://..."
          />
          <WasInput
            v-model="form.siteFavicon"
            label="Favicon URL"
            placeholder="https://..."
          />
        </div>
      </div>

      <!-- SEO -->
      <div class="surface-card p-6">
        <h2 class="text-sm font-serif-jp font-semibold text-heading tracking-wider mb-4 pb-2 border-b border-subtle">
          SEO 设置
        </h2>
        <div class="space-y-4">
          <WasInput
            v-model="form.seoKeywords"
            label="关键词"
            placeholder="博客,技术,生活"
          />
          <WasTextarea
            v-model="form.seoDescription"
            label="SEO 描述"
            placeholder="用于搜索引擎结果页的描述"
            rows="3"
          />
        </div>
      </div>

      <!-- 社交链接 -->
      <div class="surface-card p-6">
        <h2 class="text-sm font-serif-jp font-semibold text-heading tracking-wider mb-4 pb-2 border-b border-subtle">
          社交链接
        </h2>
        <div class="space-y-4">
          <WasInput
            v-model="form.socialGithub"
            label="GitHub"
            placeholder="https://github.com/username"
          />
          <WasInput
            v-model="form.socialTwitter"
            label="Twitter / X"
            placeholder="https://twitter.com/username"
          />
          <WasInput
            v-model="form.socialEmail"
            label="邮箱"
            placeholder="email@example.com"
          />
        </div>
      </div>

      <!-- 页脚 & 备案 -->
      <div class="surface-card p-6">
        <h2 class="text-sm font-serif-jp font-semibold text-heading tracking-wider mb-4 pb-2 border-b border-subtle">
          页脚信息
        </h2>
        <div class="space-y-4">
          <WasTextarea
            v-model="form.footerText"
            label="页脚文字"
            placeholder="© 2026 Ley. All rights reserved."
            rows="2"
          />
          <WasInput
            v-model="form.icpNumber"
            label="备案号"
            placeholder="京ICP备XXXXXXXX号"
          />
        </div>
      </div>

      <!-- 功能开关 -->
      <div class="surface-card p-6 lg:col-span-2">
        <h2 class="text-sm font-serif-jp font-semibold text-heading tracking-wider mb-4 pb-2 border-b border-subtle">
          功能设置
        </h2>
        <div class="flex items-center gap-4">
          <label class="flex items-center gap-2 cursor-pointer">
            <input
              v-model="form.enableLikes"
              type="checkbox"
              class="w-4 h-4 accent-accent rounded border-default"
            >
            <span class="text-sm text-body">启用文章点赞功能</span>
          </label>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
definePageMeta({
  layout: 'admin',
  middleware: 'admin',
  pageTransition: false,
})

const siteStore = useSiteStore()
const ui = useUiStore()

// 加载配置
await siteStore.fetchConfig()

// 表单（深拷贝避免直接修改 store）
const form = reactive<SiteConfig>({
  siteTitle: '',
  siteSubtitle: '',
  siteDescription: '',
  siteLogo: '',
  siteFavicon: '',
  seoKeywords: '',
  seoDescription: '',
  socialGithub: '',
  socialTwitter: '',
  socialEmail: '',
  footerText: '',
  icpNumber: '',
  enableLikes: true,
})

// 同步 store 数据到表单
watch(() => siteStore.config, (config) => {
  if (config) {
    Object.assign(form, config)
  }
}, { immediate: true })

const saving = ref(false)

async function save() {
  saving.value = true
  try {
    await siteStore.updateConfig({ ...form })
    ui.toast('配置已保存', 'success')
  }
  catch (e: any) {
    ui.toast(e?.message || '保存失败', 'error')
  }
  finally {
    saving.value = false
  }
}

useHead({
  title: '站点配置 - 管理后台',
})
</script>
