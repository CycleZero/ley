/**
 * 检测页面滚动方向
 *
 * 用于实现导航栏的智能显隐：
 * - 向上滚动或接近顶部时显示导航栏
 * - 向下滚动时隐藏导航栏
 *
 * @param threshold 滚动阈值（px），低于此值时始终显示
 * @returns isVisible 导航栏是否应该可见
 */
export function useScrollDirection(threshold = 80) {
  const scrollY = ref(0)
  const isVisible = ref(true)

  let lastScrollY = 0
  let ticking = false

  function update() {
    const current = scrollY.value

    // 接近顶部时始终显示
    if (current < threshold) {
      isVisible.value = true
    }
    // 向上滚动显示，向下滚动隐藏
    else {
      isVisible.value = current < lastScrollY
    }

    lastScrollY = current
    ticking = false
  }

  function onScroll() {
    scrollY.value = window.scrollY
    if (!ticking) {
      requestAnimationFrame(update)
      ticking = true
    }
  }

  onMounted(() => {
    window.addEventListener('scroll', onScroll, { passive: true })
  })

  onUnmounted(() => {
    window.removeEventListener('scroll', onScroll)
  })

  return {
    scrollY: readonly(scrollY),
    isVisible: readonly(isVisible),
  }
}
