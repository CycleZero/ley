<script setup lang="ts">
/**
 * PublicFooter — 公开页面底部栏
 *
 * 数据来源：GET /api/v1/site/config（与 Header 共享 key，Nuxt 自动去重）
 *
 * 三行：版权/备案号/社交链接（全从站点配置读取）
 */
const { data: siteData } = useFetch('/api/v1/site/config', {
  key: 'site-config',
  transform: (res: {
    config: {
      site_title: string
      footer_text: string
      icp_number: string
      social_github: string
      social_twitter: string
      social_email: string
    }
  }) => res.config,
})

const title = computed(() => siteData.value?.site_title || 'Ley Blog')
const footerText = computed(() => siteData.value?.footer_text || '')
const icp = computed(() => siteData.value?.icp_number || '')
const github = computed(() => siteData.value?.social_github || '')
const twitter = computed(() => siteData.value?.social_twitter || '')
const email = computed(() => siteData.value?.social_email || '')

const year = new Date().getFullYear()
</script>

<template>
  <footer class="border-t border-gray-200 bg-gray-50">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <div class="flex flex-col items-center gap-3 text-sm text-gray-500">
        <p v-if="footerText">{{ footerText }}</p>
        <p v-else>&copy; {{ year }} {{ title }}</p>
        <p v-if="icp">
          <a href="https://beian.miit.gov.cn" target="_blank" rel="noopener noreferrer" class="hover:text-gray-700 transition-colors">{{ icp }}</a>
        </p>
        <div v-if="github || twitter || email" class="flex items-center gap-4">
          <a v-if="github" :href="github" target="_blank" rel="noopener noreferrer" class="hover:text-gray-700 transition-colors">GitHub</a>
          <a v-if="twitter" :href="twitter" target="_blank" rel="noopener noreferrer" class="hover:text-gray-700 transition-colors">Twitter</a>
          <a v-if="email" :href="`mailto:${email}`" class="hover:text-gray-700 transition-colors">Email</a>
        </div>
        <p class="text-xs text-gray-400 mt-2">
          Powered by <a href="https://github.com/CycleZero/ley" target="_blank" class="hover:text-gray-600">Ley</a>
        </p>
      </div>
    </div>
  </footer>
</template>
