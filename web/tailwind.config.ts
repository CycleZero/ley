import type { Config } from 'tailwindcss'

export default {
  content: [
    './app/components/**/*.{vue,js,ts}',
    './app/layouts/**/*.vue',
    './app/pages/**/*.vue',
    './app/composables/**/*.{js,ts}',
    './app/*.vue',
    './app/**/*.vue',
  ],
  theme: {
    extend: {
      // ==================== 语义化颜色 ====================
      // 背景层级
      colors: {
        base: 'var(--bg-base)',
        surface: {
          DEFAULT: 'var(--bg-surface)',
          hover: 'var(--bg-surface-hover)',
        },
        elevated: 'var(--bg-elevated)',

        // 文字层级
        heading: 'var(--text-heading)',
        body: 'var(--text-body)',
        muted: 'var(--text-muted)',
        placeholder: 'var(--text-placeholder)',

        // 强调色
        accent: {
          DEFAULT: 'var(--accent)',
          hover: 'var(--accent-hover)',
          subtle: 'var(--accent-bg)',
          border: 'var(--accent-border)',
        },
        success: {
          DEFAULT: 'var(--success)',
          subtle: 'var(--success-bg)',
          border: 'var(--success-border)',
        },
        error: {
          DEFAULT: 'var(--error)',
          subtle: 'var(--error-bg)',
          border: 'var(--error-border)',
        },
        warning: {
          DEFAULT: 'var(--warning)',
          subtle: 'var(--warning-bg)',
        },
      },

      // 边框色
      borderColor: {
        subtle: 'var(--border-subtle)',
        default: 'var(--border-default)',
      },

      // 深色背景 / 浅色文字（分别定义避免冲突）
      backgroundColor: {
        inverted: 'var(--bg-inverted)',
        overlay: 'var(--bg-overlay)',
      },
      textColor: {
        inverted: 'var(--text-inverted)',
      },

      // ==================== 字体 ====================
      fontFamily: {
        'serif-jp': [
          '"Noto Serif SC"',
          '"Source Han Serif SC"',
          '"Songti SC"',
          'serif',
        ],
        'body-jp': [
          '"LXGW WenKai"',
          '"PingFang SC"',
          '"Microsoft YaHei"',
          '-apple-system',
          'BlinkMacSystemFont',
          'sans-serif',
        ],
        'sans': [
          'Inter',
          '"LXGW WenKai"',
          'sans-serif',
        ],
        'mono': [
          '"JetBrains Mono"',
          '"LXGW WenKai"',
          '"PingFang SC"',
          'monospace',
        ],
      },

      // ==================== 排版尺度 ====================
      fontSize: {
        'display': ['2.5rem',  { lineHeight: '1.2', letterSpacing: '0.02em' }],
        'h1':      ['1.75rem', { lineHeight: '1.4', letterSpacing: '0.05em' }],
        'h2':      ['1.375rem',{ lineHeight: '1.4', letterSpacing: '0.05em' }],
        'h3':      ['1.125rem',{ lineHeight: '1.5' }],
        'body':    ['1rem',    { lineHeight: '1.9' }],
        'sm':      ['0.875rem',{ lineHeight: '1.6' }],
        'xs':      ['0.75rem', { lineHeight: '1.5' }],
      },

      // ==================== 间距（間 MA）====================
      spacing: {
        'ma-1': '0.25rem',
        'ma-2': '0.5rem',
        'ma-3': '1rem',
        'ma-4': '1.5rem',
        'ma-5': '2.5rem',
        'ma-6': '4rem',
        'ma-7': '6rem',
        'ma-8': '10rem',
      },

      // ==================== 圆角（极度克制）====================
      borderRadius: {
        'none': '0',
        'sm':   '2px',
        'md':   '4px',
        'lg':   '6px',
      },

      // ==================== 阴影（几乎不用）====================
      boxShadow: {
        'line':   'inset 0 -1px 0 0 var(--border-subtle)',
        'subtle': '0 1px 2px rgba(0,0,0,0.04)',
      },

      // ==================== 动画缓动 ====================
      transitionTimingFunction: {
        'ink': 'cubic-bezier(0.25, 0.1, 0.25, 1)',
      },
    },
  },
  plugins: [],
} satisfies Config
