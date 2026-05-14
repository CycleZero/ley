/**
 * API 客户端插件
 *
 * 本插件通过 Nuxt 的依赖注入机制，将 lib/api/client.ts 中导出的
 * api 对象注册为全局可用的 $api 辅助函数。
 *
 * 使用方式：
 * - 在 Vue 组件的 <script setup> 中通过 const { $api } = useNuxtApp() 获取
 * - 或在模板中直接使用 $api
 *
 * 设计决策：
 * - 使用 provide 模式而非全局变量，确保与 Nuxt 生命周期一致
 * - 避免服务端渲染时 localStorage 不可用的问题（api 内部已处理 import.meta.server）
 *
 * 注意：文件名约定为 api.ts，Nuxt 会自动扫描 plugins/ 目录并注册，
 * 不需要在 nuxt.config.ts 中手动声明。插件加载顺序由文件名前缀（数字）决定，
 * 无前缀则按字母顺序加载。
 */

import { api } from '~~/lib/api/client'

export default defineNuxtPlugin(() => {
  /**
   * 返回 provide 对象，键名 'api' 将映射为模板和组合式函数中的 $api。
   * 例如：const { $api } = useNuxtApp(); await $api.get('/api/v1/articles')
   */
  return {
    provide: { api },
  }
})
