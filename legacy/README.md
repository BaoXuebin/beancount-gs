# legacy — beancount-gs v1 后端

此目录存放 beancount-gs **v1** 的 Go 后端与旧前端构建产物，仅供历史参考与数据迁移对照，
不再参与 V2 的构建与运行。

- `server.go` + `script/` + `service/`：v1 单机版 REST 后端（gin + bean-query 终端文本解析）
- `tests/`：v1 占位测试
- `public/`：旧 beancount-web 前端构建产物
- `.github/`：v1 的 Docker 发布工作流
- `go.mod` / `go.sum`：v1 Go 模块（根目录已不再维护 Go module，V2 模块在 `apps/api`）
- `Dockerfile` / `docker-compose.yml` / `var.env`：v1 部署配置

V2 对应实现：
- 后端：[apps/api](../apps/api)（契约驱动，OpenAPI 见 [packages/contracts/openapi.yaml](../packages/contracts/openapi.yaml)）
- 前端：[apps/web](../apps/web)（React + shadcn/ui）
- 重构方案与命名对照：[docs/v2-refactor.md](../docs/v2-refactor.md)
