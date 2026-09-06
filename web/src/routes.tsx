import { createBrowserRouter, Navigate } from "react-router-dom";
import { lazy } from "react";
import DefaultLayout from "@/components/layout/DefaultLayout";
import CleanLayout from "@/components/layout/CleanLayout";
import BlankLayout from "@/components/layout/BlankLayout";
import AdminLayout from "@/components/layout/AdminLayout";
import RequireAuth from "@/components/RequireAuth";
import RequireAdmin from "@/components/RequireAdmin";

const HomePage = lazy(() => import("@/pages/home"));
const ArticleListPage = lazy(() => import("@/pages/article-list"));
const ArticleDetailPage = lazy(() => import("@/pages/article-detail"));
const AboutPage = lazy(() => import("@/pages/about"));
const CategoriesPage = lazy(() => import("@/pages/categories"));
const TagsPage = lazy(() => import("@/pages/tags"));
const TagArticlesPage = lazy(() => import("@/pages/tag-articles"));
const SearchPage = lazy(() => import("@/pages/search"));
const LoginPage = lazy(() => import("@/pages/login"));
const RegisterPage = lazy(() => import("@/pages/register"));
const ProfilePage = lazy(() => import("@/pages/profile"));
const AdminDashboard = lazy(() => import("@/pages/admin/dashboard"));
const ArticleManage = lazy(() => import("@/pages/admin/article-manage"));
const ArticleEditor = lazy(() => import("@/pages/admin/article-editor"));
const CategoryManage = lazy(() => import("@/pages/admin/category-manage"));
const TagManage = lazy(() => import("@/pages/admin/tag-manage"));
const FileManage = lazy(() => import("@/pages/admin/file-manage"));
const SiteConfig = lazy(() => import("@/pages/admin/site-config"));

export const router = createBrowserRouter([
  {
    element: <DefaultLayout />,
    children: [
      { index: true, element: <HomePage /> },
      { path: "articles", element: <ArticleListPage /> },
      { path: "about", element: <AboutPage /> },
      { path: "categories", element: <CategoriesPage /> },
      { path: "tags", element: <TagsPage /> },
      { path: "tags/:name", element: <TagArticlesPage /> },
      { path: "search", element: <SearchPage /> },
    ],
  },
  {
    element: <CleanLayout />,
    children: [{ path: "articles/:slug", element: <ArticleDetailPage /> }],
  },
  {
    element: <BlankLayout />,
    children: [
      { path: "login", element: <LoginPage /> },
      { path: "register", element: <RegisterPage /> },
    ],
  },
  // Auth-required routes
  {
    element: <RequireAuth />,
    children: [
      {
        element: <DefaultLayout />,
        children: [{ path: "profile", element: <ProfilePage /> }],
      },
      // Admin routes
      {
        element: <RequireAdmin />,
        children: [
          {
            element: <AdminLayout />,
            children: [
              { path: "admin", element: <AdminDashboard /> },
              { path: "admin/articles", element: <ArticleManage /> },
              { path: "admin/articles/new", element: <ArticleEditor /> },
              { path: "admin/articles/edit/:id", element: <ArticleEditor /> },
              { path: "admin/categories", element: <CategoryManage /> },
              { path: "admin/tags", element: <TagManage /> },
              { path: "admin/files", element: <FileManage /> },
              { path: "admin/site", element: <SiteConfig /> },
            ],
          },
        ],
      },
    ],
  },
  // Catch-all → redirect to home
  { path: "*", element: <Navigate to="/" replace /> },
]);
