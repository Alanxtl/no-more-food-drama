# 让你选你又不选 MVP Design

Date: 2026-05-20
Project: no-more-food-drama

## Product Definition

“让你选你又不选”是一个给情侣用的附近餐厅协商工具：地图负责找候选餐厅，LLM 负责理解餐厅风格，两个人各自筛掉不想吃的类型或餐厅，系统最后给出双方都能接受的最佳选择。

MVP 要做成真实可部署版本，而不是静态原型。第一版部署到 Vercel，支持双人房间、附近餐厅搜索、LLM 标签增强、类型卡片筛选和共同 Top 5 推荐。

## Confirmed Decisions

- 部署形态：Vercel 全栈轻量 MVP。
- 前端：Next.js App Router、TypeScript、Tailwind CSS。
- 后端：Go API，运行在 Vercel Functions。
- 房间同步：Upstash Redis / Vercel KV，加前端轮询。
- 房间 TTL：1 小时。创建、加入、搜索、投票、单店剔除、重新计算时续期；普通轮询不续期。
- 地图：第一版中国大陆优先，使用高德地图服务端 Provider。
- 高德密钥：服务端环境变量 `AMAP_API_KEY`，不暴露给前端。
- LLM：OpenAI-compatible 通用接口，用户输入 `API Key / Base URL / Model`。服务端只转发，不保存 Key。
- 餐厅标签：先规则兜底，后端后台用 LLM 增强。
- 筛选交互：一次看一个“饮食类型”卡片，不是一次只看一家餐厅。
- 类型态度：`可以吃 / 无所谓 / 今天不吃`。`今天不吃` 是软降权，不是硬删除。
- 入房方式：一人创建房间，分享链接或二维码给另一位。
- 位置方式：自动定位优先，失败或拒绝时输入地址、商圈或地标兜底。

## User Flow

用户打开首页后先输入 LLM 配置。配置只默认存在 `sessionStorage`，不长期保存。用户也可以选择暂不填写，用规则标签继续。

创建者创建房间后，系统生成房间码、分享链接和二维码。另一位打开链接后加入同一房间，系统为每个浏览器生成独立 `participantId`。

创建者设置搜索条件：自动定位优先，失败时输入地址或商圈。半径支持 `1km / 3km / 5km / 10km`，返回数量支持 `20 / 40 / 60`。后端调用高德搜索附近餐厅，立刻写入原始候选餐厅。

餐厅返回后，后端先基于高德分类、店名关键词、距离、价格、营业状态生成规则类型和基础标签。前端马上进入类型卡片流。后台随后调用 LLM 增强餐厅类型和生活化标签，结果写回房间后，前端轮询自动更新。

两个人各自在手机上刷类型卡片。每张卡代表一个饮食类型，例如日料、火锅、粤菜、粉面、烧烤。卡片内预览该类型下的代表餐厅、最近距离、平均评分、价格范围和标签。用户对类型选择 `可以吃 / 无所谓 / 今天不吃`。

双方筛完主要类型后进入结果页。系统输出 Top 5 推荐，解释每个推荐为什么合适，也提示明显风险。结果页支持单店剔除和重新计算。单店 `remove` 是硬删除，类型 `avoid` 只是软降权。

## Architecture

前端和 Go API 同仓库部署到 Vercel。Next.js 负责页面、移动端交互、房间轮询和用户本地配置。Go 负责外部服务调用、房间状态、标签处理和推荐排序。

Suggested structure:

```txt
app/
  page.tsx
  room/[roomId]/page.tsx
  components/
  lib/
api/
  rooms.go
internal/
  amap/
  domain/
  httpapi/
  llm/
  recommend/
  roomstore/
  tagging/
```

Important backend boundaries:

- `internal/amap`: 高德地理编码和周边餐厅搜索 Provider。
- `internal/llm`: OpenAI-compatible chat completions client。
- `internal/tagging`: 规则标签、LLM 输出校验、标签合并。
- `internal/roomstore`: Redis 读写、TTL、版本控制。
- `internal/recommend`: 推荐打分、排序、解释生成。
- `internal/httpapi`: 路由、请求校验、统一响应和错误码。

同步方式先用轮询。房间页和筛选页每 2 秒读取一次快照。`heartbeat` 低频更新在线状态，可以每 30 秒一次。写操作使用乐观更新，失败后拉取最新快照。

## Data Model

Room:

```ts
type Room = {
  id: string
  version: number
  createdAt: string
  expiresAt: string
  status: "lobby" | "searching" | "tagging" | "filtering" | "results"
  searchConfig?: SearchConfig
  participants: Record<string, Participant>
  restaurants: Restaurant[]
  types: FoodType[]
  recommendations: Recommendation[]
}
```

Participant:

```ts
type Participant = {
  displayName: string
  role: "creator" | "partner"
  joinedAt: string
  lastSeenAt: string
  typeVotes: Record<string, "want" | "neutral" | "avoid">
  restaurantOverrides: Record<string, "keep" | "remove">
}
```

Restaurant:

```ts
type Restaurant = {
  id: string
  provider: "amap"
  providerId: string
  name: string
  address: string
  lat: number
  lng: number
  distanceMeters: number
  rating?: number
  priceLevel?: string
  avgPriceCny?: number
  openNow?: boolean
  categories: string[]
  typeIds: string[]
  tags: string[]
  raw?: unknown
}
```

FoodType:

```ts
type FoodType = {
  id: string
  label: string
  source: "rules" | "llm" | "mixed"
  tags: string[]
  restaurantIds: string[]
  stats: {
    count: number
    nearestMeters: number
    avgRating?: number
    avgPriceCny?: number
  }
}
```

Recommendation:

```ts
type Recommendation = {
  restaurantId: string
  score: number
  rank: number
  reasons: string[]
  warnings: string[]
}
```

MVP 使用整房间 JSON 存入 Redis key `room:{roomId}`，并维护 `version`。两个人房间的数据量小，整对象读写更简单。后续如果扩展多人、账号或历史记录，再拆成多 key 或数据库表。

## API Design

Responses use a consistent envelope:

```json
{
  "ok": true,
  "data": {},
  "error": null
}
```

Errors use:

```json
{
  "ok": false,
  "data": null,
  "error": {
    "code": "ROOM_EXPIRED",
    "message": "房间已过期，请重新创建"
  }
}
```

Routes:

```txt
POST /api/rooms
Create a room. Returns roomId, creator participantId, shareUrl.

POST /api/rooms/{roomId}/join
Join a room. Returns participantId and current room snapshot.

GET /api/rooms/{roomId}
Read room snapshot for polling. Does not refresh TTL.

POST /api/rooms/{roomId}/heartbeat
Update participant lastSeenAt and refresh room TTL.

POST /api/rooms/{roomId}/search
Submit location, radius and limit. Calls Amap, writes restaurants, rule tags and type cards.

POST /api/rooms/{roomId}/tag
Run background LLM tag enhancement using user-provided API Key, Base URL and Model.

POST /api/rooms/{roomId}/votes/type
Update participant vote for a food type: want, neutral or avoid.

POST /api/rooms/{roomId}/votes/restaurant
Update participant override for one restaurant: keep or remove.

POST /api/rooms/{roomId}/recommendations
Recompute Top 5 recommendations and explanation snapshot.
```

## Tagging Strategy

Rule tagging runs first and always remains available. It uses Amap categories, restaurant names, price, distance and open status to infer food types and basic tags.

Initial type examples:

- 火锅
- 日料
- 韩餐
- 粤菜
- 川菜
- 粉面
- 烧烤
- 咖啡甜品
- 快餐
- 小吃
- 其他好吃的

Initial tag examples:

- 正餐
- 小吃
- 快速解决
- 清淡
- 重口味
- 性价比高
- 约会友好
- 适合拍照
- 可能排队
- 夜宵

LLM tagging runs after the restaurant list is already visible. The Go backend calls an OpenAI-compatible `/chat/completions` endpoint with a compressed restaurant list. The model must return JSON only. The returned data can update `typeIds`, `tags` and optional short reasons, but cannot overwrite map facts such as distance, rating, address or open status.

If LLM fails, times out or returns invalid JSON, the app keeps rule tags and shows a non-blocking message: “已用规则标签继续”。

## Recommendation Scoring

Recommendations use a 100-point score:

```txt
基础质量 35
- 评分、评价数量、是否营业、地址和距离完整度

便利性 20
- 距离越近越高，接近半径边缘降分

双方偏好 30
- 双方都 want 的类型加分
- 一方 avoid 软降权
- 双方 avoid 大幅降权，但不绝对删除
- 单店 remove 硬删除

场景匹配 10
- 约会友好、快速解决、清淡、重口等标签按当前时间和双方选择加减分

风险惩罚 5
- 可能排队、太贵、已打烊、距离偏远等
```

Result explanations should be conversational and concrete:

- “离你们 650m，正在营业，评分 4.7。”
- “你们都没有排除日料，对方还点了可以吃。”
- “可能要排队，所以没有排第一。”

When Top 5 is sparse, the results page can show a backup area by relaxing avoid penalties. If every restaurant is removed by single-restaurant overrides, the page asks users to reopen some choices or search again.

## Frontend UX

The first screen is the actual tool, not a marketing page. The UI should feel like a useful mobile-first decision helper: clear, fast, slightly playful, but not childish.

Primary states:

- `HomeSetup`: LLM config, create room, join room.
- `RoomLobby`: room code, share link, QR code, online status.
- `SearchSetup`: geolocation, manual address, radius, result count.
- `LoadingCandidates`: nearby restaurant search and initial display.
- `TypeSwipe`: one food type per card, representative restaurants and tags.
- `Results`: Top 5, reasons, warnings, navigation, single-restaurant remove, recompute.

The type card is the core interaction. Each card presents one food type and a compact preview:

- type label
- count of restaurants
- nearest distance
- average rating and price if available
- key tags
- three representative restaurants
- actions: `今天不吃 / 无所谓 / 可以吃`

## Error Handling

- Missing `AMAP_API_KEY`: show deployment configuration error.
- Geolocation denied or failed: switch to manual address input.
- Address geocoding failed: ask user to try a landmark or business district.
- Amap returns no results: suggest expanding radius or changing location.
- LLM config invalid: keep rule tags and show “已用规则标签继续”。
- Room expired: show “房间已过期，请重新创建”。
- Redis conflict: retry once, then pull latest room snapshot.
- Too many avoid votes: show backup recommendations and explain that shared choices are limited.

## Testing Strategy

Go unit tests:

- rule tagging from Amap categories and names
- LLM JSON validation and merge behavior
- recommendation scoring and hard/soft exclusions
- room TTL and version updates
- API error code mapping

Go handler tests:

- create room
- join room
- search with mock Amap Provider
- type vote
- restaurant override
- expired room

Frontend tests:

- LLM config form stores only in session
- room lobby displays share URL and QR state
- type card vote buttons update local optimistic state
- result page renders reasons and warnings

E2E smoke test:

- create room
- join as second participant
- search mock restaurants
- apply type votes from both participants
- compute and display Top 5 recommendations

External API calls should be mocked in CI. Development should include Mock Amap and Mock LLM providers so the full product loop can run without paid API keys.

## Out Of Scope For MVP

- User accounts
- Long-term preference memory
- WebSocket realtime sync
- Payment or billing
- Multi-room history
- Restaurant photos
- Real queue time estimation
- Weather-aware recommendations
- “今天不吃上次吃过的”
- Roulette or drawing mode

These are intentionally left for later so the first version can validate the core loop: nearby restaurants, LLM understanding, dual-person type filtering and shared recommendation.
