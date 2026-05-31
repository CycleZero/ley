// ============================================================
// Ley API 类型定义
// 基于后端 OpenAPI (openapi.yaml) 生成
// ============================================================

// ---------- 通用类型 ----------

export interface UserInfo {
  id: string
  username: string
  email: string
  avatar: string
  bio: string
  role: string
  createdAt: string
  updatedAt: string
}

export interface AuthorInfo {
  id: string
  username: string
  avatar: string
}

export interface TokenPair {
  accessToken: string
  refreshToken: string
  expiresIn: string
}

// ---------- 认证 ----------

export interface LoginRequest {
  account: string
  password: string
}

export interface LoginReply {
  user: UserInfo
  tokenPair: TokenPair
}

export interface LogoutRequest {
  refreshToken: string
}

export interface LogoutReply {
  // 空对象
}

export interface RefreshTokenRequest {
  refreshToken: string
}

export interface RefreshTokenReply {
  user: UserInfo
  tokenPair: TokenPair
}

export interface RegisterRequest {
  username: string
  email: string
  password: string
}

export interface RegisterReply {
  user: UserInfo
  tokenPair: TokenPair
}

export interface GetProfileReply {
  user: UserInfo
}

export interface UpdateProfileRequest {
  avatar?: string
  bio?: string
}

export interface UpdateProfileReply {
  user: UserInfo
}

// ---------- 分类 ----------

export interface CategoryInfo {
  id: string
  name: string
  slug: string
  description: string
  parentId: string
  sortOrder: number
  articleCount: string
  children: CategoryInfo[]
}

export interface ListCategoriesReply {
  categories: CategoryInfo[]
}

export interface CreateCategoryRequest {
  name: string
  slug: string
  description?: string
  parentId?: string
  sortOrder?: number
}

export interface CreateCategoryReply {
  category: CategoryInfo
}

export interface UpdateCategoryRequest {
  id?: string
  name?: string
  slug?: string
  description?: string
  parentId?: string
  sortOrder?: number
}

export interface UpdateCategoryReply {
  category: CategoryInfo
}

export interface DeleteCategoryReply {
  // 空对象
}

// ---------- 标签 ----------

export interface TagInfo {
  id: string
  name: string
  slug: string
  articleCount: string
}

export interface ListTagsReply {
  tags: TagInfo[]
}

export interface CreateTagRequest {
  name: string
}

export interface CreateTagReply {
  tag: TagInfo
}

export interface DeleteTagReply {
  // 空对象
}

// ---------- 文章 ----------

export interface ArticleInfo {
  id: string
  title: string
  slug: string
  content: string
  excerpt: string
  coverImage: string
  status: string
  authorId: string
  author: AuthorInfo
  categoryId: string
  category: CategoryInfo
  tags: TagInfo[]
  viewCount: string
  likeCount: string
  isTop: boolean
  isLiked: boolean
  publishedAt: string
  createdAt: string
  updatedAt: string
}

export interface ListArticlesReply {
  articles: ArticleInfo[]
  total: string
  page: number
  pageSize: number
}

export interface CreateArticleRequest {
  title: string
  content: string
  excerpt?: string
  coverImage?: string
  categoryId?: string
  tagNames?: string[]
}

export interface CreateArticleReply {
  article: ArticleInfo
}

export interface GetArticleReply {
  article: ArticleInfo
}

export interface UpdateArticleRequest {
  id?: string
  title?: string
  content?: string
  excerpt?: string
  coverImage?: string
  categoryId?: string
  tagNames?: string[]
}

export interface UpdateArticleReply {
  article: ArticleInfo
}

export interface DeleteArticleReply {
  // 空对象
}

export interface ArchiveArticleRequest {
  id: string
}

export interface ArchiveArticleReply {
  article: ArticleInfo
}

export interface PublishArticleRequest {
  id: string
}

export interface PublishArticleReply {
  article: ArticleInfo
}

export interface LikeArticleRequest {
  id: string
}

export interface LikeArticleReply {
  // 空对象
}

export interface UnlikeArticleReply {
  // 空对象
}

export interface ViewArticleReply {
  counted: boolean
}

export interface SearchArticlesReply {
  articles: ArticleInfo[]
  total: string
}

// ---------- 文件 ----------

export interface FileInfo {
  id: string
  filename: string
  mimeType: string
  size: string
  url: string
  createdAt: string
}

export interface ListFilesReply {
  files: FileInfo[]
  total: string
}

export interface GetPresignedPutURLReply {
  url: string
  objectKey: string
}

export interface UploadFileRequest {
  filename: string
  mimeType: string
  content: string // base64 编码的字节
}

export interface UploadFileReply {
  file: FileInfo
}

export interface GetFileReply {
  file: FileInfo
}

export interface DeleteFileReply {
  // 空对象
}

// ---------- 站点配置 ----------

export interface SiteConfig {
  siteTitle: string
  siteSubtitle: string
  siteDescription: string
  siteLogo: string
  siteFavicon: string
  seoKeywords: string
  seoDescription: string
  socialGithub: string
  socialTwitter: string
  socialEmail: string
  footerText: string
  icpNumber: string
  enableLikes: boolean
}

export interface SiteBackground {
  id: string
  filename: string
  url: string
  isActive: boolean
  sortOrder: number
  createdAt: string
}

export interface MusicTrack {
  title: string
  artist: string
  url: string
  coverUrl: string
}

export interface MusicPlaylist {
  tracks: MusicTrack[]
}

export interface GetSiteConfigReply {
  config: SiteConfig
}

export interface UpdateSiteConfigRequest {
  config: SiteConfig
}

export interface UpdateSiteConfigReply {
  config: SiteConfig
}

export interface ListBackgroundsReply {
  backgrounds: SiteBackground[]
}

export interface UploadBackgroundRequest {
  filename: string
  content: string // base64 编码的字节
}

export interface UploadBackgroundReply {
  background: SiteBackground
}

export interface DeleteBackgroundReply {
  // 空对象
}

export interface SetActiveBackgroundRequest {
  id: string
}

export interface SetActiveBackgroundReply {
  // 空对象
}

export interface GetMusicPlaylistReply {
  playlist: MusicPlaylist
}

export interface UpdateMusicPlaylistRequest {
  playlist: MusicPlaylist
}

export interface UpdateMusicPlaylistReply {
  playlist: MusicPlaylist
}

// ============================================================
// 预留：评论系统类型（暂不实现，保留扩展空间）
// ============================================================

/** 评论信息（预留） */
export interface CommentInfo {
  id: string
  articleId: string
  parentId: string
  authorId: string
  author: AuthorInfo
  content: string
  likeCount: string
  isLiked: boolean
  status: string
  createdAt: string
  updatedAt: string
  children?: CommentInfo[]
}

/** 创建评论请求（预留） */
export interface CreateCommentRequest {
  articleId: string
  parentId?: string
  content: string
}

/** 评论列表响应（预留） */
export interface ListCommentsReply {
  comments: CommentInfo[]
  total: string
}
