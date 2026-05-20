# 让你选你又不选

给情侣用的附近餐厅协商工具：地图找候选餐厅，LLM 归纳餐厅风格，两个人各自筛掉今天不想吃的类型，系统给出共同可接受的 Top 5。

## MVP 行为

- 一个人创建房间，另一位通过房间链接或二维码加入。
- 创建者搜索附近餐厅，后端统一整理餐厅信息。
- 规则标签会立即生成，用户提供 OpenAI 兼容 API Key 后可继续用 LLM 增强类型和标签。
- 两个人各自看同一组饮食类型卡片，而不是一次只看一家店。
- 每张类型卡可以选“今天不吃”“无所谓”“可以吃”。
- 类型级“今天不吃”是软惩罚，具体餐厅“剔除”是硬排除。
- 两个人都完成类型筛选后，系统输出共同可接受的 Top 5 推荐。
- 房间状态保存在后端，TTL 为 1 小时。

## 技术栈

- Next.js App Router + TypeScript + Tailwind CSS
- Go serverless API
- Upstash Redis REST room store
- Amap Web Service 餐厅搜索
- OpenAI 兼容 chat completions 标签增强
- Vitest + Playwright

## 本地开发

安装依赖：

```bash
npm install
```

启动 Next.js：

```bash
npm run dev
```

本地默认建议使用 mock provider：

```bash
USE_MOCK_PROVIDERS=true npm run dev
```

Windows 如果当前 shell 还没读到新安装的 Go，可以先刷新 PATH：

```powershell
$env:Path = [Environment]::GetEnvironmentVariable('Path','Machine') + ';' + [Environment]::GetEnvironmentVariable('Path','User')
```

## 环境变量

复制 `.env.example` 并按环境填写：

```bash
AMAP_API_KEY=
UPSTASH_REDIS_REST_URL=
UPSTASH_REDIS_REST_TOKEN=
NEXT_PUBLIC_APP_URL=http://localhost:3000
USE_MOCK_PROVIDERS=true
```

`USE_MOCK_PROVIDERS=true` 会使用内存房间、假餐厅和假标签，适合本地开发与 CI。生产环境应设置为 `false` 或不设置，并提供 Amap 与 Upstash 配置。

LLM API Key 由用户在网页输入，前端只放在 `sessionStorage`，不写死在服务端，也不默认长期保存。

## 检查

```bash
npm run test
npm run go:test
npm run build
npm run lint
npm run e2e
```

也可以跑一键检查：

```bash
npm run check
```

`npm run e2e` 会启动一个本地 Go API 测试服务和 Next.js dev server，完整跑通双人创建房间、搜索、按类型筛选、生成推荐的 smoke flow。

## Vercel 部署

推荐部署到 Vercel。仓库内的 `api/rooms.go` 会作为 Go Function 处理房间、餐厅搜索、标签生成和推荐排序。

生产环境至少需要配置：

```bash
AMAP_API_KEY=your-amap-key
UPSTASH_REDIS_REST_URL=https://...
UPSTASH_REDIS_REST_TOKEN=...
NEXT_PUBLIC_APP_URL=https://your-app.vercel.app
```

如果需要临时演示无外部依赖版本，可以在 Vercel 上设置：

```bash
USE_MOCK_PROVIDERS=true
```

但 mock 模式使用内存状态，不适合作为正式多人使用环境。
