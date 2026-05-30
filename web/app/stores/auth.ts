/**
 * 认证状态管理
 *
 * 包含：用户信息、Token 管理、登录/注册/登出/刷新。
 * 认证相关 API 直接内联调用 $fetch，避免与 useApiClient 形成循环依赖。
 */
export const useAuthStore = defineStore('auth', () => {
  const config = useRuntimeConfig()
  const ui = useUiStore()

  // ---------- State ----------

  /** 访问令牌（15 分钟有效） */
  const accessToken = useCookie<string | null>('ley_at', {
    default: () => null,
    maxAge: 15 * 60, // 15 分钟
  })

  /** 刷新令牌（7 天有效） */
  const refreshToken = useCookie<string | null>('ley_rt', {
    default: () => null,
    maxAge: 7 * 24 * 60 * 60, // 7 天
  })

  /** 当前用户信息 */
  const user = ref<UserInfo | null>(null)

  /** 是否已登录 */
  const isLoggedIn = computed(() => !!accessToken.value && !!user.value)

  /** 是否为管理员 */
  const isAdmin = computed(() => user.value?.role === 'admin')

  // ---------- Actions ----------

  /**
   * 登录
   * 成功后自动设置 Token 和用户信息
   */
  async function login(data: LoginRequest) {
    const res = await $fetch<GatewayResponse<LoginReply>>(`${config.public.apiBase}/api/v1/auth/login`, {
      method: 'POST',
      body: data,
    })
    const payload = unwrap(res)
    accessToken.value = payload.tokenPair.accessToken
    refreshToken.value = payload.tokenPair.refreshToken
    user.value = payload.user
    ui.toast('登录成功', 'success')
    return payload
  }

  /**
   * 注册
   * 成功后自动登录
   */
  async function register(data: RegisterRequest) {
    const res = await $fetch<GatewayResponse<RegisterReply>>(`${config.public.apiBase}/api/v1/auth/register`, {
      method: 'POST',
      body: data,
    })
    const payload = unwrap(res)
    accessToken.value = payload.tokenPair.accessToken
    refreshToken.value = payload.tokenPair.refreshToken
    user.value = payload.user
    ui.toast('注册成功', 'success')
    return payload
  }

  /**
   * 刷新 Token
   * 被 useApiClient 拦截器调用，也支持手动调用
   */
  async function refresh() {
    if (!refreshToken.value) {
      throw new Error('无刷新令牌')
    }
    const res = await $fetch<GatewayResponse<RefreshTokenReply>>(`${config.public.apiBase}/api/v1/auth/refresh`, {
      method: 'POST',
      body: { refreshToken: refreshToken.value } satisfies RefreshTokenRequest,
    })
    const payload = unwrap(res)
    accessToken.value = payload.tokenPair.accessToken
    refreshToken.value = payload.tokenPair.refreshToken
    user.value = payload.user
    return payload
  }

  /**
   * 登出
   * 将当前 Token 加入后端黑名单，并清空本地状态
   */
  async function logout() {
    if (accessToken.value && refreshToken.value) {
      await $fetch<GatewayResponse<LogoutReply>>(`${config.public.apiBase}/api/v1/auth/logout`, {
        method: 'POST',
        body: {
          refreshToken: refreshToken.value,
        } satisfies LogoutRequest,
      }).catch(() => {
        // 忽略网络错误，强制清空本地状态
      })
    }
    accessToken.value = null
    refreshToken.value = null
    user.value = null
    ui.toast('已登出', 'info')
  }

  /**
   * 获取当前用户资料
   */
  async function fetchProfile() {
    const res = await $fetch<GatewayResponse<GetProfileReply>>(`${config.public.apiBase}/api/v1/auth/profile`, {
      method: 'GET',
      headers: {
        Authorization: `Bearer ${accessToken.value}`,
      },
    })
    const payload = unwrap(res)
    user.value = payload.user
    return payload
  }

  /**
   * 更新用户资料
   */
  async function updateProfile(data: UpdateProfileRequest) {
    const res = await $fetch<GatewayResponse<UpdateProfileReply>>(`${config.public.apiBase}/api/v1/auth/profile`, {
      method: 'PUT',
      headers: {
        Authorization: `Bearer ${accessToken.value}`,
      },
      body: data,
    })
    const payload = unwrap(res)
    user.value = payload.user
    ui.toast('资料已更新', 'success')
    return payload
  }

  /**
   * 初始化：如果有 Token，尝试恢复登录态
   * 通常在 app.vue 的 onMounted 中调用
   */
  async function init() {
    if (accessToken.value && !user.value) {
      try {
        await fetchProfile()
      }
      catch {
        // 获取资料失败，Token 可能已过期，尝试刷新
        if (refreshToken.value) {
          try {
            await refresh()
          }
          catch {
            logout()
          }
        }
        else {
          logout()
        }
      }
    }
  }

  return {
    // State
    accessToken,
    refreshToken,
    user,
    isLoggedIn,
    isAdmin,
    // Actions
    login,
    register,
    refresh,
    logout,
    fetchProfile,
    updateProfile,
    init,
  }
})

// ---------- 网关响应解包 ----------

interface GatewayResponse<T> {
  code: number
  msg: string
  data: T
}

function unwrap<T>(res: GatewayResponse<T>): T {
  if (res.code !== 0) {
    throw new Error(res.msg || `请求失败: code=${res.code}`)
  }
  return res.data
}
