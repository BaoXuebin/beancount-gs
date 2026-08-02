# beancount-gs V2

多人协作的 beancount 记账服务（SaaS 模式）：React + shadcn/ui 前端、Go 后端、
SQLite 元数据、GitHub OAuth2 登录、AI 记账助手与 MCP 接入。

## 仓库结构

```text
apps/
  api/        # Go 后端（gin + SQLite + bean-query 适配 + MCP）
  web/        # React + Vite + TypeScript + Tailwind + shadcn/ui
packages/
  contracts/  # OpenAPI 3.0 契约（前后端与文档的单一来源）
prototype/    # 按页面拆分的线框原型（浏览器直接打开）
docs/         # V2 重构方案
template/     # 账本初始化模板（beancount 文件结构）
lib/          # beancount 运行时（Docker 构建用）
legacy/       # v1 后端与旧前端（仅历史参考）
```

## 快速开始

**后端**（需要 GitHub OAuth App 与 bean-query 2.3.x）

```shell
cd apps/api
go run ./cmd/server
```

环境变量：`GITHUB_CLIENT_ID` / `GITHUB_CLIENT_SECRET` / `APP_PUBLIC_URL`（OAuth 回调），
`DB_PATH`（SQLite 路径，默认 `data/beancount-gs.db`），`DATA_ROOT`（账本文件根目录），
`AI_PROVIDER` / `AI_API_KEY` / `AI_MODEL` / `AI_BASE_URL`（AI，可省略）。

**前端**

```shell
cd apps/web
pnpm install
pnpm dev        # http://localhost:5173，/api 代理到后端
```

**API 文档**：后端启动后 `GET /api/v2/openapi.json` 返回契约；原型中的调试页见
`prototype/pages/api-docs.html`。

**Agent 接入（MCP）**：在「设置 → 集成与 MCP」创建 API Key 后，配置到 Claude Code / Cursor：

```json
{
  "mcpServers": {
    "beancount-gs": {
      "type": "http",
      "url": "http://localhost:10000/api/v2/mcp",
      "headers": { "Authorization": "Bearer <bgsk_...>" }
    }
  }
}
```

## 测试

```shell
cd apps/api && go test ./...   # 含真实 bean-query 集成测试
cd apps/web && pnpm build      # TS 严格模式 + 产物构建
```

## 相关链接

- 原型总览：[prototype/index.html](prototype/index.html)
- 重构方案：[docs/v2-refactor.md](docs/v2-refactor.md)
- API 契约：[packages/contracts/openapi.yaml](packages/contracts/openapi.yaml)
- v1 历史代码：[legacy/](legacy/)
