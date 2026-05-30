<!--
  blank.vue — 空白布局

  使用场景：
  - 登录页、注册页等需要全屏居中的页面

  特点：
  - 无导航栏、无页脚
  - 全屏居中显示内容
-->
<template>
  <div class="min-h-screen flex items-center justify-center bg-base px-6">
    <slot />
  </div>
</template>

<script setup lang="ts">
// 空白布局无需任何逻辑
</script>
