import { MarkdownHooks, type Components } from "react-markdown";
import remarkGfm from "remark-gfm";
import rehypeShiki from "@shikijs/rehype";
import { Children, useRef, useState, type ReactElement } from "react";
import { Check, Copy } from "lucide-react";

// 仅加载常用语言，避免打包全部语言（其余语言按需懒加载）
const LANGUAGES = [
  "go", "typescript", "javascript", "tsx", "jsx", "python", "java", "c", "cpp",
  "csharp", "bash", "shell", "json", "yaml", "markdown", "mdx", "sql", "html",
  "css", "rust", "php", "vue", "ruby", "kotlin", "swift", "dart", "plaintext",
  "text", "diff", "dockerfile", "ini", "toml", "xml",
];

// 代码块：语言标签 + 复制按钮（Shiki 高亮由 rehype 插件完成）
function CodeBlock({ children, ...props }: React.HTMLAttributes<HTMLPreElement>) {
  const [copied, setCopied] = useState(false);
  const preRef = useRef<HTMLPreElement>(null);

  // 语言从 code 子元素的 language-xxx 类提取（rehypeShiki addLanguageClass 注入）
  const codeEl = Children.only(children) as ReactElement<{ className?: string }>;
  const lang = (codeEl?.props?.className ?? "").match(/language-([\w-]+)/)?.[1] ?? "text";

  const copy = async () => {
    const text = preRef.current?.textContent ?? "";
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // 剪贴板不可用（非安全上下文等）时静默失败
    }
  };

  return (
    <div className="code-block">
      <div className="code-block-header">
        <span className="code-block-lang">{lang}</span>
        <button type="button" className="code-block-copy" onClick={copy} aria-label="复制代码">
          {copied ? <Check className="size-3.5" /> : <Copy className="size-3.5" />}
          {copied ? "已复制" : "复制"}
        </button>
      </div>
      <pre ref={preRef} {...props}>{children}</pre>
    </div>
  );
}

const components: Components = {
  pre: CodeBlock,
};

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
              addLanguageClass: true, // 注入 language-xxx 类，供语言标签读取
            },
          ],
        ]}
        components={components}
      >
        {content}
      </MarkdownHooks>
    </div>
  );
}
