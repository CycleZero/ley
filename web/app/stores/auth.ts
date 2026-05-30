/**
 * 认证状态管理
 *
 * 关键设计：Token 存储在 cookie 中，Pinia 只存 user 和内存中的 token ref。
 * 避免 Pinia hydration 序列化覆盖 cookie：
 *   - login/logout/refresh 时手动写 cookie
 *   - init() 时手动读 cookie，不从 Pinia state 恢复 token
 */
export const useAuthStore = defineStore('auth', () => {
  const config = useRuntimeConfig()
  const ui = useUiStore()

  // ---------- Cookie 读写工具（隔离在 Pinia 外部逻辑） ----------

  const _atCookie = useCookie<string | null>('ley_at', { default: () => null })
  const _rtCookie = useCookie<string | null>('ley_rt', { default: () => null })

  function readTokenFromCookie(): string | null {
    // 直接读 cookie，绕过 Pinia state
    return _atCookie.value
  }

  function readRefreshFromCookie(): string | null {
    return _rtCookie.value
  }

  function writeTokens(at: string | null, rt: string | null) {
    _atCookie.value = at
    _rtCookie.value = rt
  }

  // ---------- State ----------

  /** 当前用户信息 */
  const user = ref<UserInfo | null>(null)

  /** 是否已登录（以 cookie 为准，避免 Pinia state 干扰） */
  const isLoggedIn = computed(() => !!readTokenFromCookie() && !!user.value)

  /** 是否为管理员 */
  const isAdmin = computed(() => user.value?.role === 'admin')

  // ---------- Actions ----------

  /**
   * 登录
   */
  async function login(data: LoginRequest) {
    const res = await $fetch<GatewayResponse<LoginReply>>(`${config.public.apiBase}/api/v1/auth/login`, {
      method: 'POST',
      body: data,
    })
    const payload = unwrap(res)
    writeTokens(payload.tokenPair.accessToken, payload.tokenPair.refreshToken)
    user.value = payload.user
    ui.toast('登录成功', 'success')
    return payload
  }

  /**
   * 注册
   */
  async function register(data: RegisterRequest) {
    const res = await $fetch<GatewayResponse<RegisterReply>>(`${config.public.apiBase}/api/v1/auth/register`, {
      method: 'POST',
      body: data,
    })
    const payload = unwrap(res)
    writeTokens(payload.tokenPair.accessToken, payload.tokenPair.refreshToken)
    user.value = payload.user
    ui.toast('注册成功', 'success')
    return payload
  }

  /**
   * 刷新 Token
   */
  async function refresh() {
    const rt = readRefreshFromCookie()
    if (!rt) {
      throw new Error('无刷新令牌')
    }
    const res = await $fetch<GatewayResponse<RefreshTokenReply>>(`${config.public.apiBase}/api/v1/auth/refresh`, {
      method: 'POST',
      body: { refreshToken: rt } satisfies RefreshTokenRequest,
    })
    const payload = unwrap(res)
    writeTokens(payload.tokenPair.accessToken, payload.tokenPair.refreshToken)
    user.value = payload.user
    return payload
  }

  /**
   * 登出
   */
  async function logout() {
    const at = readTokenFromCookie()
    const rt = readRefreshFromCookie()
    if (at && rt) {
      await $fetch<GatewayResponse<LogoutReply>>(`${config.public.apiBase}/api/v1/auth/logout`, {
        method: 'POST',
        body: { refreshToken: rt } satisfies LogoutRequest,
      }).catch(() => {})
    }
    writeTokens(null, null)
    user.value = null
    ui.toast('已登出', 'info')
  }

  /**
   * 获取当前用户资料
   */
  async function fetchProfile() {
    const at = readTokenFromCookie()
    if (!at) throw new Error('无访问令牌')
    const res = await $fetch<GatewayResponse<GetProfileReply>>(`${config.public.apiBase}/api/v1/users/me`, {
      method: 'GET',
      headers: {
        Authorization: `Bearer ${at}`,
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
    const at = readTokenFromCookie()
    if (!at) throw new Error('无访问令牌')
    const res = await $fetch<GatewayResponse<UpdateProfileReply>>(`${config.public.apiBase}/api/v1/users/me`, {
      method: 'PUT',
      headers: {
        Authorization: `Bearer ${at}`,
      },
      body: data,
    })
    const payload = unwrap(res)
    user.value = payload.user
    ui.toast('资料已更新', 'success')
    return payload
  }

  /**
   * 初始化：从 cookie 读取 token，尝试恢复登录态
   */
  async function init() {
    if (user.value) return

    const at = readTokenFromCookie()
    if (at) {
      try {
        await fetchProfile()
        return
      }
      catch (e: any) {
        const status = e?.statusCode || e?.response?.status || e?.status
        if (status !== 401) {
          return
        }
        // 401 时尝试刷新
      }
    }

    const rt = readRefreshFromCookie()
    if (rt) {
      try {
        await refresh()
      }
      catch {
        logout()
      }
    }
  }

  return {
    // State
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
    // 供外部读取 token（从 cookie，非 Pinia state）
    get accessToken() { return readTokenFromCookie() },
    get refreshToken() { return readRefreshFromCookie() },
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
