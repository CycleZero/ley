package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/CycleZero/ley/app/entry/internal/common"
	pkgmeta "github.com/CycleZero/ley/pkg/meta"
	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// ===================== 代理转发骨架（handle.go） =====================
//
// 设计约定（全代理 handler 共用，改动前请先读此处）：
//
//  1. 每 API 一个 handler，统一走 callProto 骨架：绑定 JSON 请求体 →
//     注入元数据上下文 → 调 gRPC → 信封响应；handler 自身只负责
//     构造具体请求消息与路径参数合并（见 BindPath* 辅助）。
//  2. 路径参数合并采用「显式逐字段」方式：每个 handler 按自己的路由参数
//     显式调用 BindPathUint64(c, "id", &req.Id) 等辅助（不引入反射，
//     39 个 handler 场景下逐 handler 直调，字段类型与路由参数一一对应、
//     编译期可查）。路径参数合并须在 callProto 之前完成；若客户端同时在
//     body 携带与路径参数同名字段，protojson 合并语义下 body 会覆盖——
//     与 kratos HTTP→gRPC 网关行为一致，REST 契约下资源标识只应经路径传递，
//     约定请求体不得携带路径同名字段。
//  3. 成功响应经 protojson（UseProtoNames）序列化为 snake_case JSON，
//     以 json.RawMessage 塞入信封 data，避免二次编码；失败响应经
//     common.MapGRPCError 映射为 (HTTP 状态码, 业务码, 中文消息) 后写失败信封。

// callProto 通用 gRPC 代理调用骨架：绑定 JSON 请求体 → 构造元数据上下文 →
// 调 RPC → 信封响应。
//
// 参数：
//
//   - req：本次请求对应的 proto 请求消息（handler 分配并传入，body 与路径参数
//     合并后的最终请求体）；
//
//   - invoke：真正的 gRPC 调用闭包（handler 内做具体 client 方法与类型断言）：
//
//     reply, err := invoke(ctx, req)
//
// 各步失败语义：
//   - 请求体绑定失败 → 400 信封「参数解析失败」（不向客户端泄漏 protojson 细节）；
//   - gRPC 调用失败 → common.MapGRPCError 映射 (httpStatus, code, msg) 后写失败信封；
//   - 响应序列化失败 → 500 信封（内部错误，不应发生）。
func callProto(c *gin.Context,
	req proto.Message,
	invoke func(ctx context.Context, req proto.Message) (proto.Message, error),
) {
	// 1. 请求体绑定：protojson 解到 req（DiscardUnknown 容忍前端多余字段）
	if err := bindRequestBody(c, req); err != nil {
		common.Fail(c, http.StatusBadRequest, http.StatusBadRequest, "参数解析失败")
		return
	}

	// 2. 元数据上下文：gin 侧请求元数据（T4 认证中间件经 SetRequestMeta 种入）
	//    转为 pkg/meta.RequestMetaData 后注入 gRPC client 上下文。底层连接挂载了
	//    kratos metadata.Client()（infra/grpc.go），x-md-global-* 会被序列化到
	//    wire header 透传下游 auth/blog——这是 entry 用户上下文透传的唯一出口。
	//    匿名请求（GetRequestMeta 为 nil）时 BuildRequestMeta 仍回填真实客户端 IP。
	ctx := pkgmeta.NewClientCtx(c.Request.Context(), common.BuildRequestMeta(c))

	// 3. 调用下游 RPC：统一记录耗时、日志与业务指标（39 个 handler 共用此出口）
	start := time.Now()
	reply, err := invoke(ctx, req)
	operation := string(req.ProtoReflect().Descriptor().FullName())
	if err != nil {
		httpStatus, code, msg := common.MapGRPCError(err)
		recordProxyCall(ctx, operation, httpStatus, time.Since(start), err)
		common.Fail(c, httpStatus, code, msg)
		return
	}
	recordProxyCall(ctx, operation, http.StatusOK, time.Since(start), nil)

	// 4. 成功：protojson 序列化 reply → snake_case（UseProtoNames，与 auth 服务
	//    kratosjson.MarshalOptions.UseProtoNames = true 的对外契约一致），
	//    data 直接携带原始 JSON（json.RawMessage 内联，避免 map 二次编码漂移）
	//    EmitUnpopulated：输出空 repeated 为 []（如空文章列表 articles:[]），
	//    与 blog/auth 的 kratos 编码器行为对齐——否则前端读 .length 崩溃
	data, err := protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: true}.Marshal(reply)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, http.StatusInternalServerError, "服务内部错误")
		return
	}
	common.OK(c, json.RawMessage(data))
}

// bindRequestBody 将 HTTP 请求体按 protojson 解到 req。
//
// GET/HEAD 无请求体语义、以及其余方法请求体为空时直接跳过（protojson 对空输入
// 会报错）；DiscardUnknown 忽略下游 proto 中不存在的未知字段，避免前端携带
// 多余字段时误伤正常请求（与宽松的 HTTP 网关契约一致）。
//
// 注意（T7 修复）：protojson.Unmarshal 会先 proto.Reset 目标消息——若 handler 已
// 在调用 callProto 前把路径参数合并进 req（如 PUT /articles/{id} 的 Id），直接把
// body 解到 req 会把路径参数清空。故先解到临时消息再 proto.Merge 叠加：路径参数
// 与 body 字段共存，body 携带同名字段时仍按「body 覆盖」语义生效（与文件顶部
// 设计约定一致）；req 初始为空时 Merge 等价于直接 Unmarshal，行为不回归。
func bindRequestBody(c *gin.Context, req proto.Message) error {
	if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead {
		return nil
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return nil
	}
	tmp := req.ProtoReflect().New().Interface() // 同类型空消息，作为 body 解码落点
	// 括号包裹复合字面量：if 初始化语句内 literal 后跟方法调用存在解析歧义（gofmt 提示）
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(body, tmp); err != nil {
		return err
	}
	proto.Merge(req, tmp)
	return nil
}

// ===================== 路径参数合并辅助（显式逐字段） =====================

// BindPathString 将路径参数 name 原样写入字符串字段 dst。
//
// 路由未声明该路径参数（c.Param 返回空串）时跳过并返回 true——参数存在性由
// 路由保证，handler 可放心在多个路由复用时调用同一辅助。路径参数不存在
// 解析失败一说，故本辅助永不写失败信封。
func BindPathString(c *gin.Context, name string, dst *string) bool {
	if v := c.Param(name); v != "" {
		*dst = v
	}
	return true
}

// BindPathUint64 读取路径参数 name 并按 uint64 解析写入 dst。
//
// 典型用法：DELETE /api/blog/articles/:id 场景下
// req := &blogv1.DeleteArticleRequest{}; if !BindPathUint64(c, "id", &req.Id) { return }。
// 路由未声明该参数时跳过（同 BindPathString）；解析失败时写出 400 失败信封并
// 返回 false，调用方应立即 return（禁止继续携带零值调用下游）。
func BindPathUint64(c *gin.Context, name string, dst *uint64) bool {
	v := c.Param(name)
	if v == "" {
		return true
	}
	n, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		common.Fail(c, http.StatusBadRequest, http.StatusBadRequest, "路径参数 "+name+" 必须为无符号整数")
		return false
	}
	*dst = n
	return true
}

// BindPathInt64 读取路径参数 name 并按 int64 解析写入 dst（语义同 BindPathUint64）。
func BindPathInt64(c *gin.Context, name string, dst *int64) bool {
	v := c.Param(name)
	if v == "" {
		return true
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		common.Fail(c, http.StatusBadRequest, http.StatusBadRequest, "路径参数 "+name+" 必须为整数")
		return false
	}
	*dst = n
	return true
}

// BindPathInt32 读取路径参数 name 并按 int32 解析写入 dst（语义同 BindPathUint64）。
func BindPathInt32(c *gin.Context, name string, dst *int32) bool {
	v := c.Param(name)
	if v == "" {
		return true
	}
	n, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		common.Fail(c, http.StatusBadRequest, http.StatusBadRequest, "路径参数 "+name+" 必须为整数")
		return false
	}
	*dst = int32(n)
	return true
}
