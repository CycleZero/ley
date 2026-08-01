import ReactMarkdown from "react-markdown";
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
 */
export function ArticleContent({ content }: { content: string }) {
  return (
    <div className="markdown-body">
      <ReactMarkdown
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
      </ReactMarkdown>
    </div>
  );
}
