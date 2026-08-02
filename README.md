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

首次启动会自动生成 `config.yaml`，编辑其中的
`github_client_id` / `github_client_secret`（回调地址为 `public_url + /api/v2/auth/github/callback`）
与 AI 配置（可选）后重启即可。也可参考 [config.example.yaml](apps/api/config.example.yaml)。

配置文件支持的全部字段：

```yaml
port: 10000                    # 服务端口
db_path: data/beancount-gs.db  # SQLite 路径
data_root: data                # 账本文件根目录
template_dir: ../../template   # 账本模板目录
public_url: http://localhost:10000
frontend_url: http://localhost:5173  # 登录成功后的跳转；开发模式填前端地址，生产同域留空
session_cookie: bgs_session
oauth_state_cookie: bgs_oauth_state
github_client_id: ""           # GitHub OAuth
github_client_secret: ""
ai_provider: ""                # openai | compatible | ollama | deepseek（可选）
ai_api_key: ""
ai_model: ""
ai_base_url: ""
```

DeepSeek 示例：`ai_provider: deepseek`、`ai_api_key: sk-xxx`、`ai_model: deepseek-chat`，
`ai_base_url` 可省略（默认 `https://api.deepseek.com/v1`）。

命令行参数 `-p` / `-db` / `-config` 可覆盖配置文件；`BGS_CONFIG` 环境变量可指定配置文件路径。

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
