# -*- coding: utf-8 -*-
"""Ley 端到端测试：entry → auth/blog → MySQL/Redis/etcd 全链路"""
import json, urllib.request, urllib.error, time, random, string, sys, urllib.parse

BASE = "http://127.0.0.1:8000"
results = []

def call(method, path, body=None, token=None, expect_status=None):
    url = BASE + path
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(url, data=data, method=method)
    req.add_header("Content-Type", "application/json")
    if token:
        req.add_header("Authorization", "Bearer " + token)
    try:
        resp = urllib.request.urlopen(req, timeout=10)
        status, text = resp.status, resp.read().decode()
    except urllib.error.HTTPError as e:
        status, text = e.code, e.read().decode()
    try:
        parsed = json.loads(text)
    except Exception:
        parsed = text[:200]
    ok = (status == expect_status) if expect_status is not None else status in (200, 201)
    return status, parsed, ok

def check(name, status, parsed, ok, extra=""):
    detail = ""
    if isinstance(parsed, dict):
        detail = " code=%s msg=%r" % (parsed.get("code"), parsed.get("msg"))
    results.append(ok)
    print("%s %s: HTTP %s%s %s" % ("PASS" if ok else "FAIL", name, status, detail, extra))

def rand(n=6):
    return "".join(random.choices(string.ascii_lowercase + string.digits, k=n))

username = "e2e_" + rand()
email = username + "@example.com"
password = "E2eTest123"

print("=" * 60, "\n[1] 健康检查（端点已移除 → 404 且无死循环）")
for p in ["/healthz", "/readyz"]:
    s, r, ok = call("GET", p, expect_status=404)
    check(p, s, r, ok)

print("=" * 60, "\n[2] 注册")
for attempt in range(3):
    s, r, ok = call("POST", "/api/v1/auth/register", {"username": username, "email": email, "password": password})
    if ok:
        break
    time.sleep(3)
check("register", s, r, ok)
if not ok:
    print("注册失败，中止:", json.dumps(r, ensure_ascii=False)[:300]); sys.exit(1)
access = r["data"]["token_pair"]["access_token"]
refresh = r["data"]["token_pair"]["refresh_token"]
print("    用户:", username, "| token_pair 字段为 snake_case:", "token_pair" in r["data"])

print("\n[3] 登录")
s, r, ok = call("POST", "/api/v1/auth/login", {"account": username, "password": password})
check("login(用户名)", s, r, ok)
s, r, ok = call("POST", "/api/v1/auth/login", {"account": email, "password": password})
check("login(邮箱)", s, r, ok)
s, r, ok = call("POST", "/api/v1/auth/login", {"account": username, "password": "WrongPass1"}, expect_status=401)
check("login(错误密码→401)", s, r, ok, extra="")

print("\n[4] 用户资料")
s, r, ok = call("GET", "/api/v1/users/me", token=access)
check("get /users/me", s, r, ok)
s, r, ok = call("PUT", "/api/v1/users/me", {"avatar": "https://example.com/a.png", "bio": "端到端测试用户"}, token=access)
check("update profile", s, r, ok)
results.append(ok and r["data"]["user"]["bio"] == "端到端测试用户")
print("    %s 资料更新生效" % ("PASS" if ok and r["data"]["user"]["bio"] == "端到端测试用户" else "FAIL"))

print("\n[5] 无 token 访问受保护接口")
s, r, ok = call("GET", "/api/v1/users/me", expect_status=404)
check("users/me 无 token 被拒(404)", s, r, ok, extra="")

print("\n[6] Token 轮换")
s, r, ok = call("POST", "/api/v1/auth/refresh", {"refresh_token": refresh})
check("refresh", s, r, ok)
if ok:
    new_access = r["data"]["token_pair"]["access_token"]
    s2, r2, ok2 = call("POST", "/api/v1/auth/refresh", {"refresh_token": refresh}, expect_status=401)
    check("旧 refresh 重放→401", s2, r2, ok2, extra="")
    access = new_access

print("\n[7] 登出")
s, r, ok = call("POST", "/api/v1/auth/logout", {"refresh_token": refresh}, token=access)
check("logout", s, r, ok)

print("=" * 60, "\n[8] 分类与标签")
s, r, ok = call("POST", "/api/v1/categories", {"name": "E2E分类" + rand(), "slug": "e2e-cat-" + rand()}, token=access)
check("create category", s, r, ok)
cat_id = r["data"]["category"]["id"] if ok else 0
s, r, ok = call("GET", "/api/v1/categories")
check("list categories", s, r, ok)
s, r, ok = call("POST", "/api/v1/tags", {"name": "e2etag" + rand()}, token=access)
check("create tag", s, r, ok)
s, r, ok = call("GET", "/api/v1/tags")
check("list tags", s, r, ok)

print("\n[9] 文章生命周期")
title = "端到端测试文章 " + rand()
s, r, ok = call("POST", "/api/v1/articles", {
    "title": title,
    "content": "# 端到端测试\n\n这是通过 entry 创建的文章。\n\n```go\nfmt.Println(\"hello e2e\")\n```",
    "category_id": cat_id,
    "tag_names": ["e2etag"],
    "status": "draft",
}, token=access)
check("create article(草稿)", s, r, ok)
if not ok:
    print("创建文章失败，中止:", json.dumps(r, ensure_ascii=False)[:300]); sys.exit(1)
aid = r["data"]["article"]["id"]
slug = r["data"]["article"]["slug"]
print("    文章:", title, "(id=%s, slug=%s)" % (aid, slug))

s, r, ok = call("POST", "/api/v1/articles/%s/publish" % aid, token=access)
check("publish", s, r, ok)
results.append(ok and r["data"]["article"]["status"] == "published")
print("    %s 发布生效" % ("PASS" if ok and r["data"]["article"]["status"] == "published" else "FAIL"))

s, r, ok = call("GET", "/api/v1/articles?page=1&page_size=10")
check("list articles(公开)", s, r, ok)
found = any(a["id"] == aid for a in r["data"]["articles"]) if ok else False
results.append(found)
print("    %s 列表中能找到新文章" % ("PASS" if found else "FAIL"))

s, r, ok = call("GET", "/api/v1/articles/" + slug)
check("get article by slug", s, r, ok)
content_ok = ok and "端到端测试" in r["data"]["article"]["content"]
results.append(content_ok)
print("    %s Markdown 内容完整" % ("PASS" if content_ok else "FAIL"))

s, r, ok = call("GET", "/api/v1/articles/search?keyword=" + urllib.parse.quote("端到端测试") + "&page=1&page_size=5")
check("search", s, r, ok)
found = any(a["id"] == aid for a in r["data"]["articles"]) if ok else False
results.append(found)
print("    %s 搜索结果包含新文章" % ("PASS" if found else "FAIL"))

print("\n[10] 点赞")
s, r, ok = call("POST", "/api/v1/articles/%s/like" % aid, token=access)
check("like", s, r, ok)
s, r, ok = call("GET", "/api/v1/articles/%s" % slug, token=access)
liked = r["data"]["article"]["is_liked"] if ok else False
results.append(liked)
print("    %s is_liked=true" % ("PASS" if liked else "FAIL"))
s, r, ok = call("DELETE", "/api/v1/articles/%s/like" % aid, token=access)
check("unlike", s, r, ok)

print("\n[11] 站点配置")
s, r, ok = call("GET", "/api/v1/site/config")
check("get site config", s, r, ok)

print("\n[12] 归档与删除")
s, r, ok = call("POST", "/api/v1/articles/%s/archive" % aid, token=access)
check("archive", s, r, ok)
s, r, ok = call("DELETE", "/api/v1/articles/%s" % aid, token=access)
check("delete article", s, r, ok)
if ok and cat_id:
    s, r, ok = call("DELETE", "/api/v1/categories/%s" % cat_id, token=access)
    check("delete category", s, r, ok)

print("=" * 60)
passed = sum(1 for x in results if x)
print("E2E RESULT: %d/%d 项通过" % (passed, len(results)))
sys.exit(0 if passed == len(results) else 1)
