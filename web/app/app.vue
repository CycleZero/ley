<!--
  Vue 根组件（App.vue）
  
  本文件是 Nuxt 应用的入口组件，定义了页面的顶层结构。
  
  渲染流程：
  1. NuxtLayout: 加载并渲染约定的 layouts/ 布局文件（如 default.vue）。
     若无自定义布局，Nuxt 使用内置默认布局。
  2. NuxtPage: 根据当前路由动态渲染 pages/ 目录下对应的页面组件。
  
  设计原则：
  - 保持根组件极简，所有页面路由和布局逻辑由 Nuxt 框架自动处理
  - 通过 NuxtLayout + NuxtPage 的组合，实现布局复用和页面嵌套渲染
-->
<template>
  <NuxtLayout>
    <NuxtPage />
  </NuxtLayout>
</template>
