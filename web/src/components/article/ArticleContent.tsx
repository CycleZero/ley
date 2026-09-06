import { MarkdownHooks } from "react-markdown";
import remarkGfm from "remark-gfm";
import rehypeShiki from "@shikijs/rehype";

// 仅加载常用语言，避免打包全部语言（其余语言按需懒加载）
const LANGUAGES = [
  "go", "typescript", "javascript", "tsx", "jsx", "python", "java", "c", "cpp",
  "csharp", "bash", "shell", "json", "yaml", "markdown", "mdx", "sql", "html",
  "css", "rust", "php", "vue", "ruby", "kotlin", "swift", "dart", "plaintext",
  "text", "diff", "dockerfile", "ini", "toml", "xml",
];

/**
 * 文章正文渲染：Markdown + GFM + Shiki 代码高亮（双主题适配亮/暗）
 * 样式见 globals.css 的 .markdown-body
 *
 * 注意：必须用 MarkdownHooks（异步渲染）——默认导出的 Markdown 是同步的，
 * 无法处理 @shikijs/rehype 这类异步 rehype 插件（会抛 runSync finished async）。
 */
export function ArticleContent({ content }: { content: string }) {
  return (
    <div className="markdown-body">
      <MarkdownHooks
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[
          [
            rehypeShiki,
            {
              themes: { light: "github-light", dark: "github-dark" },
              defaultColor: "light",
              cssVariablePrefix: "--shiki-",
              langs: LANGUAGES,
            },
          ],
        ]}
      >
        {content}
      </MarkdownHooks>
    </div>
  );
}
