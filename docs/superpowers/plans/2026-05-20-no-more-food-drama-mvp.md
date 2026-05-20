# No More Food Drama MVP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the deployable MVP for “让你选你又不选”: a mobile-first Next.js app with Go serverless API, Upstash Redis room state, Amap restaurant search, OpenAI-compatible tag enhancement, two-person type-card filtering, and shared Top 5 recommendations.

**Architecture:** Keep the front end and Go API in one Vercel repository. The Go API owns domain behavior, room persistence, provider calls, tagging, and recommendations; the Next.js app owns session-local LLM configuration, room UI, polling, and mobile interactions. Tests use mock providers by default so the full loop works without paid external API keys.

**Tech Stack:** Next.js App Router, TypeScript, Tailwind CSS, Vitest, Playwright, Go 1.22+, Vercel Go Functions, Upstash Redis REST API, Amap Web Service API, OpenAI-compatible chat completions.

---

## Scope Check

This plan covers one integrated MVP rather than separate product plans because the user-facing loop depends on the API contract, room state, tagging, and UI shipping together. The tasks are still split by file ownership so subagents can work safely: Go domain packages first, HTTP API second, front-end shell third, E2E last.

## External References

- Vercel Go Functions are supported through the Vercel Functions runtime model: https://vercel.com/docs/functions/runtimes
- Upstash Redis REST accepts Redis commands over HTTP with `Authorization: Bearer $TOKEN`; `SET foo bar EX 100` maps to `/set/foo/bar/EX/100`, and JSON request bodies can be used for large values: https://upstash.com/docs/redis/features/restapi

## Target File Structure

```txt
.env.example                         # Required environment variables
package.json                         # Frontend scripts and dependencies
tsconfig.json                        # TypeScript config
next.config.ts                       # Next.js config
postcss.config.mjs                   # Tailwind/PostCSS config
tailwind.config.ts                   # Tailwind theme
vitest.config.ts                     # Component/unit test config
playwright.config.ts                 # E2E config
go.mod                               # Go module
vercel.json                          # Vercel runtime hints

app/layout.tsx                       # App shell metadata
app/page.tsx                         # Home setup screen
app/room/[roomId]/page.tsx           # Room flow screen
app/globals.css                      # Base styles
app/components/HomeSetup.tsx         # LLM config and room creation/join
app/components/RoomLobby.tsx         # Room code, share link, QR state
app/components/SearchSetup.tsx       # Location, radius, limit
app/components/TypeCard.tsx          # One food-type card
app/components/ResultsList.tsx       # Top 5 recommendations
app/lib/api.ts                       # Browser API client
app/lib/session.ts                   # sessionStorage helpers
app/lib/types.ts                     # Frontend API types matching Go JSON
app/tests/*.test.tsx                 # Frontend unit/component tests
e2e/mvp.spec.ts                      # Playwright smoke test

api/rooms.go                         # Vercel Go Function entrypoint
internal/domain/types.go             # Domain structs and constants
internal/domain/room.go              # Room factory and vote mutations
internal/domain/response.go          # API response envelope and error codes
internal/tagging/rules.go            # Rule-based type/tag inference
internal/tagging/llm_merge.go        # Validated LLM merge behavior
internal/recommend/score.go          # Recommendation scoring
internal/roomstore/store.go          # Store interface and errors
internal/roomstore/memory.go         # Test/local store
internal/roomstore/upstash.go        # Upstash REST store
internal/amap/client.go              # Amap geocode/search client
internal/llm/client.go               # OpenAI-compatible chat client
internal/httpapi/server.go           # Router and handler wiring
internal/httpapi/test_fakes.go       # Fake providers for handler tests
internal/httpapi/providers.go        # Runtime provider adapters
internal/**/**_test.go               # Go tests
```

## Environment Variables

Create `.env.example` with:

```bash
AMAP_API_KEY=
UPSTASH_REDIS_REST_URL=
UPSTASH_REDIS_REST_TOKEN=
NEXT_PUBLIC_APP_URL=http://localhost:3000
USE_MOCK_PROVIDERS=true
```

`USE_MOCK_PROVIDERS=true` makes local development and CI use mock Amap/LLM behavior. Production should set it to `false` or omit it.

---

### Task 1: Project Foundation

**Files:**
- Create: `package.json`
- Create: `tsconfig.json`
- Create: `next.config.ts`
- Create: `postcss.config.mjs`
- Create: `tailwind.config.ts`
- Create: `vitest.config.ts`
- Create: `playwright.config.ts`
- Create: `go.mod`
- Create: `vercel.json`
- Create: `.env.example`
- Create: `app/layout.tsx`
- Create: `app/page.tsx`
- Create: `app/globals.css`
- Modify: `.gitignore`

- [ ] **Step 1: Create package metadata and scripts**

Create `package.json`:

```json
{
  "name": "no-more-food-drama",
  "version": "0.1.0",
  "private": true,
  "scripts": {
    "dev": "next dev",
    "build": "next build",
    "start": "next start",
    "lint": "next lint",
    "test": "vitest run",
    "test:watch": "vitest",
    "e2e": "playwright test",
    "go:test": "go test ./...",
    "check": "npm run test && npm run go:test && npm run build"
  },
  "dependencies": {
    "@vitejs/plugin-react": "latest",
    "lucide-react": "latest",
    "next": "latest",
    "qrcode.react": "latest",
    "react": "latest",
    "react-dom": "latest"
  },
  "devDependencies": {
    "@playwright/test": "latest",
    "@testing-library/jest-dom": "latest",
    "@testing-library/react": "latest",
    "@types/node": "latest",
    "@types/react": "latest",
    "@types/react-dom": "latest",
    "autoprefixer": "latest",
    "eslint": "latest",
    "eslint-config-next": "latest",
    "jsdom": "latest",
    "postcss": "latest",
    "tailwindcss": "latest",
    "typescript": "latest",
    "vitest": "latest"
  }
}
```

- [ ] **Step 2: Install dependencies**

Run:

```bash
npm install
```

Expected: `package-lock.json` is created and `npm` exits with code 0.

- [ ] **Step 3: Create TypeScript and build config**

Create `tsconfig.json`:

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "lib": ["dom", "dom.iterable", "es2022"],
    "allowJs": false,
    "skipLibCheck": true,
    "strict": true,
    "noEmit": true,
    "esModuleInterop": true,
    "module": "esnext",
    "moduleResolution": "bundler",
    "resolveJsonModule": true,
    "isolatedModules": true,
    "jsx": "preserve",
    "incremental": true,
    "plugins": [{ "name": "next" }],
    "paths": { "@/*": ["./*"] }
  },
  "include": ["next-env.d.ts", "**/*.ts", "**/*.tsx", ".next/types/**/*.ts"],
  "exclude": ["node_modules"]
}
```

Create `next.config.ts`:

```ts
import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  reactStrictMode: true
};

export default nextConfig;
```

Create `postcss.config.mjs`:

```js
const config = {
  plugins: {
    tailwindcss: {},
    autoprefixer: {}
  }
};

export default config;
```

Create `tailwind.config.ts`:

```ts
import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./app/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        ink: "#25231f",
        paper: "#fbfaf7",
        line: "#e7e2d8",
        accent: "#2f7d6d",
        danger: "#b64b4b"
      }
    }
  },
  plugins: []
};

export default config;
```

- [ ] **Step 4: Create test configs**

Create `vitest.config.ts`:

```ts
import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./app/tests/setup.ts"]
  },
  resolve: {
    alias: {
      "@": new URL(".", import.meta.url).pathname
    }
  }
});
```

Create `playwright.config.ts`:

```ts
import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  timeout: 30_000,
  use: {
    baseURL: "http://127.0.0.1:3000",
    trace: "retain-on-failure"
  },
  webServer: {
    command: "npm run dev",
    url: "http://127.0.0.1:3000",
    reuseExistingServer: true,
    timeout: 120_000,
    env: {
      USE_MOCK_PROVIDERS: "true",
      NEXT_PUBLIC_APP_URL: "http://127.0.0.1:3000"
    }
  },
  projects: [
    {
      name: "mobile-chrome",
      use: { ...devices["Pixel 7"] }
    }
  ]
});
```

Create `app/tests/setup.ts`:

```ts
import "@testing-library/jest-dom/vitest";
```

- [ ] **Step 5: Create Go and Vercel config**

Create `go.mod`:

```go
module github.com/Alanxtl/no-more-food-drama

go 1.22
```

Create `vercel.json`:

```json
{
  "functions": {
    "api/*.go": {
      "runtime": "go1.x"
    }
  }
}
```

Create `.env.example` exactly as listed in the Environment Variables section.

- [ ] **Step 6: Create initial app shell**

Create `app/layout.tsx`:

```tsx
import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "让你选你又不选",
  description: "双人附近餐厅决策工具"
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="zh-CN">
      <body>{children}</body>
    </html>
  );
}
```

Create `app/page.tsx`:

```tsx
export default function HomePage() {
  return (
    <main className="min-h-screen bg-paper text-ink">
      <section className="mx-auto flex min-h-screen w-full max-w-md flex-col justify-center px-5 py-8">
        <p className="text-sm text-neutral-600">no-more-food-drama</p>
        <h1 className="mt-2 text-4xl font-bold leading-tight">让你选你又不选</h1>
        <p className="mt-4 text-base leading-7 text-neutral-700">
          先把附近餐厅找出来，再让两个人各自筛掉今天不想吃的类型。
        </p>
      </section>
    </main>
  );
}
```

Create `app/globals.css`:

```css
@tailwind base;
@tailwind components;
@tailwind utilities;

:root {
  color-scheme: light;
}

* {
  box-sizing: border-box;
}

body {
  margin: 0;
  background: #fbfaf7;
  color: #25231f;
}

button,
input,
select {
  font: inherit;
}
```

Append these lines to `.gitignore` while keeping `.superpowers/`:

```gitignore
.next/
node_modules/
coverage/
test-results/
playwright-report/
.env
```

- [ ] **Step 7: Verify scaffold**

Run:

```bash
npm run test
npm run go:test
npm run build
```

Expected: Vitest exits with no test files or one setup pass, Go reports no packages or passes, and Next builds the home page.

- [ ] **Step 8: Commit**

Run:

```bash
git add package.json package-lock.json tsconfig.json next.config.ts postcss.config.mjs tailwind.config.ts vitest.config.ts playwright.config.ts go.mod vercel.json .env.example app .gitignore
git commit -m "chore: scaffold Next and Go app"
```

---

### Task 2: Go Domain Models and Room Mutations

**Files:**
- Create: `internal/domain/types.go`
- Create: `internal/domain/room.go`
- Create: `internal/domain/response.go`
- Create: `internal/domain/room_test.go`

- [ ] **Step 1: Write failing domain tests**

Create `internal/domain/room_test.go`:

```go
package domain

import (
	"testing"
	"time"
)

func TestNewRoomCreatesCreatorAndOneHourExpiry(t *testing.T) {
	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	room, participantID := NewRoom("ABC123", "https://app.test/room/ABC123", now)

	if room.ID != "ABC123" {
		t.Fatalf("room id = %q", room.ID)
	}
	if participantID == "" {
		t.Fatal("participant id is empty")
	}
	participant := room.Participants[participantID]
	if participant.Role != RoleCreator {
		t.Fatalf("role = %q", participant.Role)
	}
	if got := room.ExpiresAt.Sub(now); got != time.Hour {
		t.Fatalf("expiry duration = %s", got)
	}
	if room.Status != StatusLobby {
		t.Fatalf("status = %q", room.Status)
	}
}

func TestTypeVoteIsSoftPreference(t *testing.T) {
	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	room, participantID := NewRoom("ABC123", "https://app.test/room/ABC123", now)

	err := room.SetTypeVote(participantID, "type-japanese", VoteAvoid, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("SetTypeVote returned error: %v", err)
	}

	if got := room.Participants[participantID].TypeVotes["type-japanese"]; got != VoteAvoid {
		t.Fatalf("vote = %q", got)
	}
	if room.Version != 2 {
		t.Fatalf("version = %d", room.Version)
	}
	if got := room.ExpiresAt.Sub(now.Add(time.Minute)); got != time.Hour {
		t.Fatalf("renewed ttl = %s", got)
	}
}

func TestRestaurantRemoveOverrideIsStored(t *testing.T) {
	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	room, participantID := NewRoom("ABC123", "https://app.test/room/ABC123", now)

	err := room.SetRestaurantOverride(participantID, "restaurant-1", RestaurantRemove, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("SetRestaurantOverride returned error: %v", err)
	}

	if got := room.Participants[participantID].RestaurantOverrides["restaurant-1"]; got != RestaurantRemove {
		t.Fatalf("override = %q", got)
	}
}

func TestUnknownParticipantReturnsError(t *testing.T) {
	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	room, _ := NewRoom("ABC123", "https://app.test/room/ABC123", now)

	err := room.SetTypeVote("missing", "type-hotpot", VoteWant, now)
	if err != ErrParticipantNotFound {
		t.Fatalf("error = %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/domain
```

Expected: FAIL because `NewRoom`, constants, and types are undefined.

- [ ] **Step 3: Implement domain types**

Create `internal/domain/types.go`:

```go
package domain

import "time"

const RoomTTL = time.Hour

type RoomStatus string

const (
	StatusLobby     RoomStatus = "lobby"
	StatusSearching RoomStatus = "searching"
	StatusTagging   RoomStatus = "tagging"
	StatusFiltering RoomStatus = "filtering"
	StatusResults   RoomStatus = "results"
)

type Role string

const (
	RoleCreator Role = "creator"
	RolePartner Role = "partner"
)

type TypeVote string

const (
	VoteWant    TypeVote = "want"
	VoteNeutral TypeVote = "neutral"
	VoteAvoid   TypeVote = "avoid"
)

type RestaurantOverride string

const (
	RestaurantKeep   RestaurantOverride = "keep"
	RestaurantRemove RestaurantOverride = "remove"
)

type Room struct {
	ID              string                    `json:"id"`
	Version         int                       `json:"version"`
	ShareURL        string                    `json:"shareUrl"`
	CreatedAt       time.Time                 `json:"createdAt"`
	ExpiresAt       time.Time                 `json:"expiresAt"`
	Status          RoomStatus                `json:"status"`
	SearchConfig    *SearchConfig             `json:"searchConfig,omitempty"`
	Participants    map[string]Participant    `json:"participants"`
	Restaurants     []Restaurant              `json:"restaurants"`
	Types           []FoodType                `json:"types"`
	Recommendations []Recommendation          `json:"recommendations"`
}

type SearchConfig struct {
	LocationText string  `json:"locationText,omitempty"`
	Lat          float64 `json:"lat,omitempty"`
	Lng          float64 `json:"lng,omitempty"`
	RadiusKM     int     `json:"radiusKm"`
	Limit        int     `json:"limit"`
}

type Participant struct {
	DisplayName         string                         `json:"displayName"`
	Role                Role                           `json:"role"`
	JoinedAt            time.Time                      `json:"joinedAt"`
	LastSeenAt          time.Time                      `json:"lastSeenAt"`
	TypeVotes           map[string]TypeVote            `json:"typeVotes"`
	RestaurantOverrides map[string]RestaurantOverride  `json:"restaurantOverrides"`
}

type Restaurant struct {
	ID             string   `json:"id"`
	Provider       string   `json:"provider"`
	ProviderID     string   `json:"providerId"`
	Name           string   `json:"name"`
	Address        string   `json:"address"`
	Lat            float64  `json:"lat"`
	Lng            float64  `json:"lng"`
	DistanceMeters int      `json:"distanceMeters"`
	Rating         float64  `json:"rating,omitempty"`
	PriceLevel     string   `json:"priceLevel,omitempty"`
	AvgPriceCNY    int      `json:"avgPriceCny,omitempty"`
	OpenNow        *bool    `json:"openNow,omitempty"`
	Categories     []string `json:"categories"`
	TypeIDs        []string `json:"typeIds"`
	Tags           []string `json:"tags"`
}

type FoodType struct {
	ID            string        `json:"id"`
	Label         string        `json:"label"`
	Source        string        `json:"source"`
	Tags          []string      `json:"tags"`
	RestaurantIDs []string      `json:"restaurantIds"`
	Stats         FoodTypeStats `json:"stats"`
}

type FoodTypeStats struct {
	Count         int     `json:"count"`
	NearestMeters int    `json:"nearestMeters"`
	AvgRating     float64 `json:"avgRating,omitempty"`
	AvgPriceCNY   int     `json:"avgPriceCny,omitempty"`
}

type Recommendation struct {
	RestaurantID string   `json:"restaurantId"`
	Score        float64  `json:"score"`
	Rank         int      `json:"rank"`
	Reasons      []string `json:"reasons"`
	Warnings     []string `json:"warnings"`
}
```

- [ ] **Step 4: Implement room mutations and response envelope**

Create `internal/domain/room.go`:

```go
package domain

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"
)

var ErrParticipantNotFound = errors.New("participant not found")

func NewRoom(id string, shareURL string, now time.Time) (Room, string) {
	participantID := newID("p")
	participant := Participant{
		DisplayName: "我",
		Role: RoleCreator,
		JoinedAt: now,
		LastSeenAt: now,
		TypeVotes: map[string]TypeVote{},
		RestaurantOverrides: map[string]RestaurantOverride{},
	}

	return Room{
		ID: id,
		Version: 1,
		ShareURL: shareURL,
		CreatedAt: now,
		ExpiresAt: now.Add(RoomTTL),
		Status: StatusLobby,
		Participants: map[string]Participant{participantID: participant},
		Restaurants: []Restaurant{},
		Types: []FoodType{},
		Recommendations: []Recommendation{},
	}, participantID
}

func (r *Room) JoinPartner(now time.Time) string {
	participantID := newID("p")
	r.Participants[participantID] = Participant{
		DisplayName: "另一位",
		Role: RolePartner,
		JoinedAt: now,
		LastSeenAt: now,
		TypeVotes: map[string]TypeVote{},
		RestaurantOverrides: map[string]RestaurantOverride{},
	}
	r.touch(now)
	return participantID
}

func (r *Room) Heartbeat(participantID string, now time.Time) error {
	p, ok := r.Participants[participantID]
	if !ok {
		return ErrParticipantNotFound
	}
	p.LastSeenAt = now
	r.Participants[participantID] = p
	r.touch(now)
	return nil
}

func (r *Room) SetTypeVote(participantID string, typeID string, vote TypeVote, now time.Time) error {
	p, ok := r.Participants[participantID]
	if !ok {
		return ErrParticipantNotFound
	}
	p.TypeVotes[typeID] = vote
	p.LastSeenAt = now
	r.Participants[participantID] = p
	r.touch(now)
	return nil
}

func (r *Room) SetRestaurantOverride(participantID string, restaurantID string, override RestaurantOverride, now time.Time) error {
	p, ok := r.Participants[participantID]
	if !ok {
		return ErrParticipantNotFound
	}
	p.RestaurantOverrides[restaurantID] = override
	p.LastSeenAt = now
	r.Participants[participantID] = p
	r.touch(now)
	return nil
}

func (r *Room) touch(now time.Time) {
	r.Version++
	r.ExpiresAt = now.Add(RoomTTL)
}

func newID(prefix string) string {
	var b [6]byte
	_, err := rand.Read(b[:])
	if err != nil {
		panic(err)
	}
	return prefix + "_" + hex.EncodeToString(b[:])
}
```

Create `internal/domain/response.go`:

```go
package domain

type APIResponse struct {
	OK    bool        `json:"ok"`
	Data  any         `json:"data"`
	Error *APIError   `json:"error"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

const (
	ErrorRoomExpired         = "ROOM_EXPIRED"
	ErrorRoomNotFound        = "ROOM_NOT_FOUND"
	ErrorParticipantNotFound = "PARTICIPANT_NOT_FOUND"
	ErrorValidation          = "VALIDATION_ERROR"
	ErrorProvider            = "PROVIDER_ERROR"
)

func Success(data any) APIResponse {
	return APIResponse{OK: true, Data: data, Error: nil}
}

func Failure(code string, message string) APIResponse {
	return APIResponse{OK: false, Data: nil, Error: &APIError{Code: code, Message: message}}
}
```

- [ ] **Step 5: Run tests**

Run:

```bash
go test ./internal/domain
```

Expected: PASS.

- [ ] **Step 6: Commit**

Run:

```bash
git add internal/domain
git commit -m "feat: add room domain model"
```

---

### Task 3: Rule Tagging

**Files:**
- Create: `internal/tagging/rules.go`
- Create: `internal/tagging/rules_test.go`

- [ ] **Step 1: Write failing rule-tagging tests**

Create `internal/tagging/rules_test.go`:

```go
package tagging

import (
	"testing"

	"github.com/Alanxtl/no-more-food-drama/internal/domain"
)

func TestBuildRuleTagsGroupsRestaurantsIntoTypes(t *testing.T) {
	open := true
	restaurants := []domain.Restaurant{
		{ID: "r1", Name: "鮨小野", DistanceMeters: 650, Rating: 4.7, AvgPriceCNY: 128, OpenNow: &open, Categories: []string{"美食", "日本料理"}},
		{ID: "r2", Name: "热辣火锅", DistanceMeters: 900, Rating: 4.5, AvgPriceCNY: 98, OpenNow: &open, Categories: []string{"美食", "火锅"}},
		{ID: "r3", Name: "老街螺蛳粉", DistanceMeters: 300, Rating: 4.2, AvgPriceCNY: 28, OpenNow: &open, Categories: []string{"美食", "小吃快餐"}},
	}

	tagged, types := BuildRuleTags(restaurants)

	assertRestaurantHasType(t, tagged, "r1", "type-japanese")
	assertRestaurantHasType(t, tagged, "r2", "type-hotpot")
	assertRestaurantHasTag(t, tagged, "r3", "快速解决")

	if len(types) != 3 {
		t.Fatalf("types length = %d", len(types))
	}
	if types[0].Stats.NearestMeters != 650 && types[1].Stats.NearestMeters != 650 && types[2].Stats.NearestMeters != 650 {
		t.Fatalf("expected a type with nearest distance 650m: %#v", types)
	}
}

func TestBuildRuleTagsUsesOtherTypeForUnknownFood(t *testing.T) {
	restaurants := []domain.Restaurant{
		{ID: "r1", Name: "好味道", DistanceMeters: 1000, Categories: []string{"美食"}},
	}

	tagged, types := BuildRuleTags(restaurants)

	assertRestaurantHasType(t, tagged, "r1", "type-other")
	if types[0].Label != "其他好吃的" {
		t.Fatalf("label = %q", types[0].Label)
	}
}

func assertRestaurantHasType(t *testing.T, restaurants []domain.Restaurant, restaurantID string, typeID string) {
	t.Helper()
	for _, restaurant := range restaurants {
		if restaurant.ID != restaurantID {
			continue
		}
		for _, got := range restaurant.TypeIDs {
			if got == typeID {
				return
			}
		}
		t.Fatalf("restaurant %s missing type %s: %#v", restaurantID, typeID, restaurant.TypeIDs)
	}
	t.Fatalf("restaurant %s not found", restaurantID)
}

func assertRestaurantHasTag(t *testing.T, restaurants []domain.Restaurant, restaurantID string, tag string) {
	t.Helper()
	for _, restaurant := range restaurants {
		if restaurant.ID != restaurantID {
			continue
		}
		for _, got := range restaurant.Tags {
			if got == tag {
				return
			}
		}
		t.Fatalf("restaurant %s missing tag %s: %#v", restaurantID, tag, restaurant.Tags)
	}
	t.Fatalf("restaurant %s not found", restaurantID)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/tagging
```

Expected: FAIL because `BuildRuleTags` is undefined.

- [ ] **Step 3: Implement deterministic rule tagging**

Create `internal/tagging/rules.go` with this behavior:

```go
package tagging

import (
	"sort"
	"strings"

	"github.com/Alanxtl/no-more-food-drama/internal/domain"
)

type rule struct {
	ID       string
	Label    string
	Keywords []string
	Tags     []string
}

var foodRules = []rule{
	{ID: "type-hotpot", Label: "火锅", Keywords: []string{"火锅", "涮", "锅"}, Tags: []string{"正餐", "重口味"}},
	{ID: "type-japanese", Label: "日料", Keywords: []string{"日本", "日料", "寿司", "鮨", "刺身", "拉面", "居酒屋", "烧鸟"}, Tags: []string{"约会友好", "清淡"}},
	{ID: "type-korean", Label: "韩餐", Keywords: []string{"韩国", "韩餐", "烤肉", "部队锅"}, Tags: []string{"正餐", "重口味"}},
	{ID: "type-yue", Label: "粤菜", Keywords: []string{"粤菜", "广东", "茶餐厅", "烧腊", "点心"}, Tags: []string{"正餐", "清淡"}},
	{ID: "type-sichuan", Label: "川菜", Keywords: []string{"川菜", "四川", "麻辣", "酸菜鱼", "串串"}, Tags: []string{"正餐", "重口味"}},
	{ID: "type-noodles", Label: "粉面", Keywords: []string{"粉", "面", "米线", "螺蛳粉", "拉面", "牛肉面"}, Tags: []string{"快速解决", "小吃"}},
	{ID: "type-bbq", Label: "烧烤", Keywords: []string{"烧烤", "烤串", "烤肉", "烤鱼"}, Tags: []string{"夜宵", "重口味"}},
	{ID: "type-dessert", Label: "咖啡甜品", Keywords: []string{"咖啡", "甜品", "奶茶", "蛋糕", "面包"}, Tags: []string{"适合拍照", "小吃"}},
	{ID: "type-fastfood", Label: "快餐", Keywords: []string{"快餐", "汉堡", "炸鸡", "披萨", "便当"}, Tags: []string{"快速解决"}},
	{ID: "type-snack", Label: "小吃", Keywords: []string{"小吃", "煎饼", "包子", "粥", "饺子"}, Tags: []string{"小吃", "性价比高"}},
}

func BuildRuleTags(restaurants []domain.Restaurant) ([]domain.Restaurant, []domain.FoodType) {
	tagged := make([]domain.Restaurant, len(restaurants))
	copy(tagged, restaurants)

	typeMap := map[string]*domain.FoodType{}

	for i := range tagged {
		text := strings.Join(append([]string{tagged[i].Name}, tagged[i].Categories...), " ")
		matched := false
		for _, rule := range foodRules {
			if containsAny(text, rule.Keywords) {
				applyRule(&tagged[i], rule)
				addToType(typeMap, rule, tagged[i])
				matched = true
			}
		}
		if !matched {
			other := rule{ID: "type-other", Label: "其他好吃的", Tags: []string{"正餐"}}
			applyRule(&tagged[i], other)
			addToType(typeMap, other, tagged[i])
		}
		applyPriceAndDistanceTags(&tagged[i])
	}

	types := flattenTypes(typeMap)
	return tagged, types
}

func containsAny(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func applyRule(restaurant *domain.Restaurant, rule rule) {
	restaurant.TypeIDs = appendUnique(restaurant.TypeIDs, rule.ID)
	for _, tag := range rule.Tags {
		restaurant.Tags = appendUnique(restaurant.Tags, tag)
	}
}

func applyPriceAndDistanceTags(restaurant *domain.Restaurant) {
	if restaurant.AvgPriceCNY > 0 && restaurant.AvgPriceCNY <= 40 {
		restaurant.Tags = appendUnique(restaurant.Tags, "性价比高")
	}
	if restaurant.DistanceMeters <= 800 {
		restaurant.Tags = appendUnique(restaurant.Tags, "离得近")
	}
}

func addToType(typeMap map[string]*domain.FoodType, rule rule, restaurant domain.Restaurant) {
	ft, ok := typeMap[rule.ID]
	if !ok {
		ft = &domain.FoodType{
			ID: rule.ID,
			Label: rule.Label,
			Source: "rules",
			Tags: []string{},
			RestaurantIDs: []string{},
			Stats: domain.FoodTypeStats{NearestMeters: restaurant.DistanceMeters},
		}
		typeMap[rule.ID] = ft
	}
	ft.RestaurantIDs = appendUnique(ft.RestaurantIDs, restaurant.ID)
	for _, tag := range rule.Tags {
		ft.Tags = appendUnique(ft.Tags, tag)
	}
	if restaurant.DistanceMeters > 0 && (ft.Stats.NearestMeters == 0 || restaurant.DistanceMeters < ft.Stats.NearestMeters) {
		ft.Stats.NearestMeters = restaurant.DistanceMeters
	}
	if restaurant.Rating > 0 {
		totalRating := ft.Stats.AvgRating * float64(ft.Stats.Count)
		ft.Stats.AvgRating = (totalRating + restaurant.Rating) / float64(ft.Stats.Count+1)
	}
	if restaurant.AvgPriceCNY > 0 {
		totalPrice := ft.Stats.AvgPriceCNY * ft.Stats.Count
		ft.Stats.AvgPriceCNY = (totalPrice + restaurant.AvgPriceCNY) / (ft.Stats.Count + 1)
	}
	ft.Stats.Count++
}

func flattenTypes(typeMap map[string]*domain.FoodType) []domain.FoodType {
	types := make([]domain.FoodType, 0, len(typeMap))
	for _, ft := range typeMap {
		types = append(types, *ft)
	}
	sort.Slice(types, func(i, j int) bool {
		if types[i].Stats.Count == types[j].Stats.Count {
			return types[i].Label < types[j].Label
		}
		return types[i].Stats.Count > types[j].Stats.Count
	})
	return types
}

func appendUnique(values []string, next string) []string {
	for _, value := range values {
		if value == next {
			return values
		}
	}
	return append(values, next)
}
```

- [ ] **Step 4: Run tests**

Run:

```bash
go test ./internal/tagging
```

Expected: PASS.

- [ ] **Step 5: Commit**

Run:

```bash
git add internal/tagging
git commit -m "feat: add rule-based restaurant tagging"
```

---

### Task 4: Recommendation Engine

**Files:**
- Create: `internal/recommend/score.go`
- Create: `internal/recommend/score_test.go`

- [ ] **Step 1: Write failing scoring tests**

Create `internal/recommend/score_test.go`:

```go
package recommend

import (
	"testing"

	"github.com/Alanxtl/no-more-food-drama/internal/domain"
)

func TestComputeRanksMutualWantAboveSoftAvoid(t *testing.T) {
	open := true
	room := domain.Room{
		Participants: map[string]domain.Participant{
			"p1": {TypeVotes: map[string]domain.TypeVote{"type-japanese": domain.VoteWant, "type-hotpot": domain.VoteAvoid}, RestaurantOverrides: map[string]domain.RestaurantOverride{}},
			"p2": {TypeVotes: map[string]domain.TypeVote{"type-japanese": domain.VoteWant, "type-hotpot": domain.VoteNeutral}, RestaurantOverrides: map[string]domain.RestaurantOverride{}},
		},
		Restaurants: []domain.Restaurant{
			{ID: "sushi", Name: "鮨小野", DistanceMeters: 650, Rating: 4.7, OpenNow: &open, TypeIDs: []string{"type-japanese"}, Tags: []string{"约会友好"}},
			{ID: "hotpot", Name: "热辣火锅", DistanceMeters: 300, Rating: 4.8, OpenNow: &open, TypeIDs: []string{"type-hotpot"}, Tags: []string{"重口味"}},
		},
	}

	recs := Compute(room, 5)

	if len(recs) != 2 {
		t.Fatalf("recommendations length = %d", len(recs))
	}
	if recs[0].RestaurantID != "sushi" {
		t.Fatalf("first recommendation = %s", recs[0].RestaurantID)
	}
	if recs[1].Score >= recs[0].Score {
		t.Fatalf("soft avoid should rank below mutual want: %#v", recs)
	}
}

func TestComputeHardRemovesSingleRestaurant(t *testing.T) {
	open := true
	room := domain.Room{
		Participants: map[string]domain.Participant{
			"p1": {TypeVotes: map[string]domain.TypeVote{}, RestaurantOverrides: map[string]domain.RestaurantOverride{"hotpot": domain.RestaurantRemove}},
			"p2": {TypeVotes: map[string]domain.TypeVote{}, RestaurantOverrides: map[string]domain.RestaurantOverride{}},
		},
		Restaurants: []domain.Restaurant{
			{ID: "hotpot", Name: "热辣火锅", DistanceMeters: 300, Rating: 4.8, OpenNow: &open, TypeIDs: []string{"type-hotpot"}},
			{ID: "noodle", Name: "老街米线", DistanceMeters: 900, Rating: 4.1, OpenNow: &open, TypeIDs: []string{"type-noodles"}},
		},
	}

	recs := Compute(room, 5)

	if len(recs) != 1 {
		t.Fatalf("recommendations length = %d", len(recs))
	}
	if recs[0].RestaurantID != "noodle" {
		t.Fatalf("remaining recommendation = %s", recs[0].RestaurantID)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/recommend
```

Expected: FAIL because `Compute` is undefined.

- [ ] **Step 3: Implement scoring**

Create `internal/recommend/score.go`:

```go
package recommend

import (
	"fmt"
	"sort"

	"github.com/Alanxtl/no-more-food-drama/internal/domain"
)

func Compute(room domain.Room, limit int) []domain.Recommendation {
	scored := make([]domain.Recommendation, 0, len(room.Restaurants))
	for _, restaurant := range room.Restaurants {
		if removedByAnyone(room, restaurant.ID) {
			continue
		}
		score, reasons, warnings := scoreRestaurant(room, restaurant)
		scored = append(scored, domain.Recommendation{
			RestaurantID: restaurant.ID,
			Score: score,
			Reasons: reasons,
			Warnings: warnings,
		})
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].Score == scored[j].Score {
			return scored[i].RestaurantID < scored[j].RestaurantID
		}
		return scored[i].Score > scored[j].Score
	})
	if limit > 0 && len(scored) > limit {
		scored = scored[:limit]
	}
	for i := range scored {
		scored[i].Rank = i + 1
	}
	return scored
}

func scoreRestaurant(room domain.Room, restaurant domain.Restaurant) (float64, []string, []string) {
	score := 0.0
	reasons := []string{}
	warnings := []string{}

	if restaurant.Rating > 0 {
		score += min(restaurant.Rating/5*22, 22)
		reasons = append(reasons, fmt.Sprintf("评分 %.1f", restaurant.Rating))
	}
	if restaurant.OpenNow != nil && *restaurant.OpenNow {
		score += 8
		reasons = append(reasons, "正在营业")
	} else if restaurant.OpenNow != nil && !*restaurant.OpenNow {
		score -= 18
		warnings = append(warnings, "可能已经打烊")
	}
	if restaurant.Address != "" && restaurant.DistanceMeters > 0 {
		score += 5
	}

	if restaurant.DistanceMeters > 0 {
		distanceScore := 20 - float64(restaurant.DistanceMeters)/3000*20
		score += max(0, distanceScore)
		if restaurant.DistanceMeters <= 800 {
			reasons = append(reasons, fmt.Sprintf("离你们 %dm", restaurant.DistanceMeters))
		}
		if restaurant.DistanceMeters > 2500 {
			warnings = append(warnings, "距离偏远")
		}
	}

	preferenceScore, preferenceReason := preference(room, restaurant)
	score += preferenceScore
	if preferenceReason != "" {
		reasons = append(reasons, preferenceReason)
	}

	for _, tag := range restaurant.Tags {
		switch tag {
		case "约会友好", "快速解决", "性价比高", "离得近":
			score += 2.5
		case "可能排队":
			score -= 5
			warnings = append(warnings, "可能要排队")
		}
	}

	return max(0, min(100, score)), reasons, warnings
}

func removedByAnyone(room domain.Room, restaurantID string) bool {
	for _, participant := range room.Participants {
		if participant.RestaurantOverrides[restaurantID] == domain.RestaurantRemove {
			return true
		}
	}
	return false
}

func preference(room domain.Room, restaurant domain.Restaurant) (float64, string) {
	score := 0.0
	wantCount := 0
	avoidCount := 0
	for _, participant := range room.Participants {
		for _, typeID := range restaurant.TypeIDs {
			switch participant.TypeVotes[typeID] {
			case domain.VoteWant:
				score += 15
				wantCount++
			case domain.VoteAvoid:
				score -= 16
				avoidCount++
			}
		}
	}
	if wantCount >= 2 {
		return score, "你们都点了可以吃"
	}
	if wantCount == 1 && avoidCount == 0 {
		return score, "有人点了可以吃，另一位没有排除"
	}
	if avoidCount >= 2 {
		return score, "你们都不太想吃这个类型，作为备选保留"
	}
	if avoidCount == 1 {
		return score, "有一位今天不太想吃这个类型"
	}
	return score, ""
}

func min(a float64, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a float64, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
```

- [ ] **Step 4: Run tests**

Run:

```bash
go test ./internal/recommend
```

Expected: PASS.

- [ ] **Step 5: Commit**

Run:

```bash
git add internal/recommend
git commit -m "feat: add recommendation scoring"
```

---

### Task 5: Room Store With Memory and Upstash Implementations

**Files:**
- Create: `internal/roomstore/store.go`
- Create: `internal/roomstore/memory.go`
- Create: `internal/roomstore/upstash.go`
- Create: `internal/roomstore/memory_test.go`
- Create: `internal/roomstore/upstash_test.go`

- [ ] **Step 1: Write failing memory store tests**

Create `internal/roomstore/memory_test.go`:

```go
package roomstore

import (
	"context"
	"testing"
	"time"

	"github.com/Alanxtl/no-more-food-drama/internal/domain"
)

func TestMemoryStoreDoesNotReturnExpiredRoom(t *testing.T) {
	store := NewMemoryStore()
	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	room, _ := domain.NewRoom("ABC123", "https://app.test/room/ABC123", now)

	if err := store.Save(context.Background(), room, domain.RoomTTL); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	store.SetNow(now.Add(domain.RoomTTL + time.Second))

	_, err := store.Get(context.Background(), "ABC123")
	if err != ErrRoomExpired {
		t.Fatalf("error = %v", err)
	}
}

func TestMemoryStoreUpdateRetriesVersion(t *testing.T) {
	store := NewMemoryStore()
	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	room, _ := domain.NewRoom("ABC123", "https://app.test/room/ABC123", now)
	if err := store.Save(context.Background(), room, domain.RoomTTL); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	updated, err := store.Update(context.Background(), "ABC123", domain.RoomTTL, func(room domain.Room) (domain.Room, error) {
		room.Status = domain.StatusFiltering
		room.Version++
		return room, nil
	})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	if updated.Status != domain.StatusFiltering {
		t.Fatalf("status = %q", updated.Status)
	}
}
```

- [ ] **Step 2: Write failing Upstash client test**

Create `internal/roomstore/upstash_test.go`:

```go
package roomstore

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Alanxtl/no-more-food-drama/internal/domain"
)

func TestUpstashStoreSendsSetWithOneHourExpiry(t *testing.T) {
	var gotAuth string
	var gotCommand []any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotCommand); err != nil {
			t.Fatalf("decode command: %v", err)
		}
		_, _ = w.Write([]byte(`{"result":"OK"}`))
	}))
	defer server.Close()

	store := NewUpstashStore(server.URL, "secret", server.Client())
	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	room, _ := domain.NewRoom("ABC123", "https://app.test/room/ABC123", now)

	err := store.Save(context.Background(), room, domain.RoomTTL)
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	if gotAuth != "Bearer secret" {
		t.Fatalf("authorization = %q", gotAuth)
	}
	if gotCommand[0] != "SET" || gotCommand[1] != "room:ABC123" || gotCommand[3] != "EX" || gotCommand[4] != float64(3600) {
		t.Fatalf("command = %#v", gotCommand)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run:

```bash
go test ./internal/roomstore
```

Expected: FAIL because store types are undefined.

- [ ] **Step 4: Implement store interface and memory store**

Create `internal/roomstore/store.go`:

```go
package roomstore

import (
	"context"
	"errors"
	"time"

	"github.com/Alanxtl/no-more-food-drama/internal/domain"
)

var ErrRoomNotFound = errors.New("room not found")
var ErrRoomExpired = errors.New("room expired")

type Store interface {
	Save(ctx context.Context, room domain.Room, ttl time.Duration) error
	Get(ctx context.Context, roomID string) (domain.Room, error)
	Update(ctx context.Context, roomID string, ttl time.Duration, mutate func(domain.Room) (domain.Room, error)) (domain.Room, error)
}

func roomKey(roomID string) string {
	return "room:" + roomID
}
```

Create `internal/roomstore/memory.go`:

```go
package roomstore

import (
	"context"
	"sync"
	"time"

	"github.com/Alanxtl/no-more-food-drama/internal/domain"
)

type memoryEntry struct {
	room      domain.Room
	expiresAt time.Time
}

type MemoryStore struct {
	mu      sync.Mutex
	rooms   map[string]memoryEntry
	nowFunc func() time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		rooms: map[string]memoryEntry{},
		nowFunc: time.Now,
	}
}

func (s *MemoryStore) SetNow(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nowFunc = func() time.Time { return now }
}

func (s *MemoryStore) Save(ctx context.Context, room domain.Room, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rooms[room.ID] = memoryEntry{room: room, expiresAt: s.nowFunc().Add(ttl)}
	return nil
}

func (s *MemoryStore) Get(ctx context.Context, roomID string) (domain.Room, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.rooms[roomID]
	if !ok {
		return domain.Room{}, ErrRoomNotFound
	}
	if !entry.expiresAt.After(s.nowFunc()) {
		delete(s.rooms, roomID)
		return domain.Room{}, ErrRoomExpired
	}
	return entry.room, nil
}

func (s *MemoryStore) Update(ctx context.Context, roomID string, ttl time.Duration, mutate func(domain.Room) (domain.Room, error)) (domain.Room, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.rooms[roomID]
	if !ok {
		return domain.Room{}, ErrRoomNotFound
	}
	if !entry.expiresAt.After(s.nowFunc()) {
		delete(s.rooms, roomID)
		return domain.Room{}, ErrRoomExpired
	}
	updated, err := mutate(entry.room)
	if err != nil {
		return domain.Room{}, err
	}
	s.rooms[roomID] = memoryEntry{room: updated, expiresAt: s.nowFunc().Add(ttl)}
	return updated, nil
}
```

- [ ] **Step 5: Implement Upstash REST store**

Create `internal/roomstore/upstash.go`:

```go
package roomstore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Alanxtl/no-more-food-drama/internal/domain"
)

type UpstashStore struct {
	baseURL string
	token   string
	client  *http.Client
}

type upstashResponse struct {
	Result json.RawMessage `json:"result"`
	Error  string          `json:"error"`
}

func NewUpstashStore(baseURL string, token string, client *http.Client) *UpstashStore {
	if client == nil {
		client = http.DefaultClient
	}
	return &UpstashStore{baseURL: strings.TrimRight(baseURL, "/"), token: token, client: client}
}

func (s *UpstashStore) Save(ctx context.Context, room domain.Room, ttl time.Duration) error {
	value, err := json.Marshal(room)
	if err != nil {
		return err
	}
	seconds := int(ttl.Seconds())
	_, err = s.command(ctx, []any{"SET", roomKey(room.ID), string(value), "EX", seconds})
	return err
}

func (s *UpstashStore) Get(ctx context.Context, roomID string) (domain.Room, error) {
	raw, err := s.command(ctx, []any{"GET", roomKey(roomID)})
	if err != nil {
		return domain.Room{}, err
	}
	if string(raw) == "null" {
		return domain.Room{}, ErrRoomNotFound
	}
	var encoded string
	if err := json.Unmarshal(raw, &encoded); err != nil {
		return domain.Room{}, err
	}
	var room domain.Room
	if err := json.Unmarshal([]byte(encoded), &room); err != nil {
		return domain.Room{}, err
	}
	if time.Now().After(room.ExpiresAt) {
		return domain.Room{}, ErrRoomExpired
	}
	return room, nil
}

func (s *UpstashStore) Update(ctx context.Context, roomID string, ttl time.Duration, mutate func(domain.Room) (domain.Room, error)) (domain.Room, error) {
	room, err := s.Get(ctx, roomID)
	if err != nil {
		return domain.Room{}, err
	}
	updated, err := mutate(room)
	if err != nil {
		return domain.Room{}, err
	}
	if err := s.Save(ctx, updated, ttl); err != nil {
		return domain.Room{}, err
	}
	return updated, nil
}

func (s *UpstashStore) command(ctx context.Context, command []any) (json.RawMessage, error) {
	body, err := json.Marshal(command)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var decoded upstashResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 || decoded.Error != "" {
		return nil, fmt.Errorf("upstash command failed: %s", decoded.Error)
	}
	return decoded.Result, nil
}
```

- [ ] **Step 6: Run tests**

Run:

```bash
go test ./internal/roomstore
```

Expected: PASS.

- [ ] **Step 7: Commit**

Run:

```bash
git add internal/roomstore
git commit -m "feat: add room stores"
```

---

### Task 6: Amap and LLM Provider Clients

**Files:**
- Create: `internal/amap/client.go`
- Create: `internal/amap/client_test.go`
- Create: `internal/llm/client.go`
- Create: `internal/llm/client_test.go`

- [ ] **Step 1: Write failing Amap client test**

Create `internal/amap/client_test.go`:

```go
package amap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchAroundMapsAmapPois(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v5/place/around" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("key") != "amap-key" {
			t.Fatalf("missing key query")
		}
		_, _ = w.Write([]byte(`{
			"status":"1",
			"pois":[
				{
					"id":"B0FFTEST",
					"name":"鮨小野",
					"address":"海珠区测试路 1 号",
					"location":"113.320000,23.090000",
					"distance":"650",
					"type":"餐饮服务;外国餐厅;日本料理",
					"biz_ext":{"rating":"4.7","cost":"128"}
				}
			]
		}`))
	}))
	defer server.Close()

	client := NewClient("amap-key", server.URL, server.Client())
	restaurants, err := client.SearchAround(context.Background(), SearchRequest{Lat: 23.09, Lng: 113.32, RadiusMeters: 3000, Limit: 20})
	if err != nil {
		t.Fatalf("SearchAround returned error: %v", err)
	}

	if len(restaurants) != 1 {
		t.Fatalf("restaurants length = %d", len(restaurants))
	}
	if restaurants[0].ProviderID != "B0FFTEST" || restaurants[0].AvgPriceCNY != 128 || restaurants[0].Rating != 4.7 {
		t.Fatalf("restaurant = %#v", restaurants[0])
	}
}
```

- [ ] **Step 2: Write failing LLM client test**

Create `internal/llm/client_test.go`:

```go
package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientSendsOpenAICompatibleRequest(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{
			"choices":[
				{"message":{"content":"{\"restaurants\":[{\"id\":\"r1\",\"typeIds\":[\"type-japanese\"],\"tags\":[\"约会友好\"]}]}"}}
			]
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user-key", "deepseek-chat", server.Client())
	result, err := client.EnhanceTags(context.Background(), `{"restaurants":[{"id":"r1","name":"鮨小野"}]}`)
	if err != nil {
		t.Fatalf("EnhanceTags returned error: %v", err)
	}
	if gotAuth != "Bearer user-key" {
		t.Fatalf("authorization = %q", gotAuth)
	}
	if len(result.Restaurants) != 1 || result.Restaurants[0].ID != "r1" {
		t.Fatalf("result = %#v", result)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run:

```bash
go test ./internal/amap ./internal/llm
```

Expected: FAIL because clients are undefined.

- [ ] **Step 4: Implement Amap client**

Create `internal/amap/client.go` with `SearchRequest`, `Client`, `NewClient`, and `SearchAround`. The implementation must:

```go
package amap

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Alanxtl/no-more-food-drama/internal/domain"
)

type SearchRequest struct {
	Lat          float64
	Lng          float64
	RadiusMeters int
	Limit        int
}

type Client struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewClient(apiKey string, baseURL string, client *http.Client) *Client {
	if client == nil {
		client = http.DefaultClient
	}
	return &Client{apiKey: apiKey, baseURL: strings.TrimRight(baseURL, "/"), client: client}
}

func (c *Client) SearchAround(ctx context.Context, request SearchRequest) ([]domain.Restaurant, error) {
	values := url.Values{}
	values.Set("key", c.apiKey)
	values.Set("location", fmt.Sprintf("%f,%f", request.Lng, request.Lat))
	values.Set("radius", strconv.Itoa(request.RadiusMeters))
	values.Set("types", "050000")
	values.Set("page_size", strconv.Itoa(request.Limit))
	values.Set("show_fields", "business")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v5/place/around?"+values.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var decoded amapAroundResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	if decoded.Status != "1" {
		return nil, fmt.Errorf("amap search failed: %s", decoded.Info)
	}
	return mapPOIs(decoded.POIs), nil
}

type amapAroundResponse struct {
	Status string    `json:"status"`
	Info   string    `json:"info"`
	POIs   []amapPOI `json:"pois"`
}

type amapPOI struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Address  string `json:"address"`
	Location string `json:"location"`
	Distance string `json:"distance"`
	Type     string `json:"type"`
	BizExt   struct {
		Rating string `json:"rating"`
		Cost   string `json:"cost"`
	} `json:"biz_ext"`
}

func mapPOIs(pois []amapPOI) []domain.Restaurant {
	restaurants := make([]domain.Restaurant, 0, len(pois))
	for _, poi := range pois {
		lng, lat := parseLocation(poi.Location)
		distance, _ := strconv.Atoi(poi.Distance)
		rating, _ := strconv.ParseFloat(poi.BizExt.Rating, 64)
		cost, _ := strconv.Atoi(poi.BizExt.Cost)
		restaurants = append(restaurants, domain.Restaurant{
			ID: "amap:" + poi.ID,
			Provider: "amap",
			ProviderID: poi.ID,
			Name: poi.Name,
			Address: poi.Address,
			Lat: lat,
			Lng: lng,
			DistanceMeters: distance,
			Rating: rating,
			AvgPriceCNY: cost,
			Categories: strings.Split(poi.Type, ";"),
			Tags: []string{},
			TypeIDs: []string{},
		})
	}
	return restaurants
}

func parseLocation(location string) (float64, float64) {
	parts := strings.Split(location, ",")
	if len(parts) != 2 {
		return 0, 0
	}
	lng, _ := strconv.ParseFloat(parts[0], 64)
	lat, _ := strconv.ParseFloat(parts[1], 64)
	return lng, lat
}
```

- [ ] **Step 5: Implement LLM client**

Create `internal/llm/client.go` with:

```go
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type Client struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

type EnhancementResult struct {
	Restaurants []RestaurantEnhancement `json:"restaurants"`
}

type RestaurantEnhancement struct {
	ID      string   `json:"id"`
	TypeIDs []string `json:"typeIds"`
	Tags    []string `json:"tags"`
}

func NewClient(baseURL string, apiKey string, model string, client *http.Client) *Client {
	if client == nil {
		client = http.DefaultClient
	}
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey, model: model, client: client}
}

func (c *Client) EnhanceTags(ctx context.Context, compactRestaurantJSON string) (EnhancementResult, error) {
	payload := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: "你是餐厅分类助手。只返回 JSON，不要解释。JSON schema: {\"restaurants\":[{\"id\":\"string\",\"typeIds\":[\"string\"],\"tags\":[\"string\"]}]}"},
			{Role: "user", Content: compactRestaurantJSON},
		},
		Temperature: 0.2,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return EnhancementResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return EnhancementResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return EnhancementResult{}, err
	}
	defer resp.Body.Close()

	var decoded chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return EnhancementResult{}, err
	}
	if resp.StatusCode >= 400 {
		return EnhancementResult{}, fmt.Errorf("llm status %d", resp.StatusCode)
	}
	if len(decoded.Choices) == 0 {
		return EnhancementResult{}, fmt.Errorf("llm returned no choices")
	}
	var result EnhancementResult
	if err := json.Unmarshal([]byte(decoded.Choices[0].Message.Content), &result); err != nil {
		return EnhancementResult{}, err
	}
	return result, nil
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}
```

- [ ] **Step 6: Run tests**

Run:

```bash
go test ./internal/amap ./internal/llm
```

Expected: PASS.

- [ ] **Step 7: Commit**

Run:

```bash
git add internal/amap internal/llm
git commit -m "feat: add provider clients"
```

---

### Task 7: LLM Tag Merge

**Files:**
- Create: `internal/tagging/llm_merge.go`
- Create: `internal/tagging/llm_merge_test.go`

- [ ] **Step 1: Write failing merge tests**

Create `internal/tagging/llm_merge_test.go`:

```go
package tagging

import (
	"testing"

	"github.com/Alanxtl/no-more-food-drama/internal/domain"
	"github.com/Alanxtl/no-more-food-drama/internal/llm"
)

func TestMergeLLMEnhancementsUpdatesOnlyTagsAndTypes(t *testing.T) {
	restaurants := []domain.Restaurant{
		{
			ID: "r1",
			Name: "鮨小野",
			Address: "原地址",
			DistanceMeters: 650,
			Rating: 4.7,
			TypeIDs: []string{"type-other"},
			Tags: []string{"正餐"},
		},
	}
	result := llm.EnhancementResult{
		Restaurants: []llm.RestaurantEnhancement{
			{ID: "r1", TypeIDs: []string{"type-japanese"}, Tags: []string{"约会友好", "漂亮饭"}},
		},
	}

	mergedRestaurants, mergedTypes := MergeLLMEnhancements(restaurants, result)

	if mergedRestaurants[0].Address != "原地址" || mergedRestaurants[0].DistanceMeters != 650 || mergedRestaurants[0].Rating != 4.7 {
		t.Fatalf("map facts changed: %#v", mergedRestaurants[0])
	}
	assertRestaurantHasType(t, mergedRestaurants, "r1", "type-japanese")
	assertRestaurantHasTag(t, mergedRestaurants, "r1", "漂亮饭")
	if len(mergedTypes) != 1 || mergedTypes[0].Source != "mixed" {
		t.Fatalf("types = %#v", mergedTypes)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/tagging
```

Expected: FAIL because `MergeLLMEnhancements` is undefined.

- [ ] **Step 3: Implement validated LLM merge**

Create `internal/tagging/llm_merge.go`:

```go
package tagging

import (
	"strings"

	"github.com/Alanxtl/no-more-food-drama/internal/domain"
	"github.com/Alanxtl/no-more-food-drama/internal/llm"
)

func MergeLLMEnhancements(restaurants []domain.Restaurant, result llm.EnhancementResult) ([]domain.Restaurant, []domain.FoodType) {
	byID := map[string]llm.RestaurantEnhancement{}
	for _, enhancement := range result.Restaurants {
		byID[enhancement.ID] = enhancement
	}

	merged := make([]domain.Restaurant, len(restaurants))
	copy(merged, restaurants)
	for i := range merged {
		enhancement, ok := byID[merged[i].ID]
		if !ok {
			continue
		}
		merged[i].TypeIDs = cleanIDs(enhancement.TypeIDs, merged[i].TypeIDs)
		merged[i].Tags = cleanTags(enhancement.Tags, merged[i].Tags)
	}
	return buildTypesFromRestaurants(merged)
}

func buildTypesFromRestaurants(restaurants []domain.Restaurant) ([]domain.Restaurant, []domain.FoodType) {
	typeMap := map[string]*domain.FoodType{}
	for _, restaurant := range restaurants {
		for _, typeID := range restaurant.TypeIDs {
			label := labelForType(typeID)
			addToType(typeMap, rule{ID: typeID, Label: label, Tags: restaurant.Tags}, restaurant)
			typeMap[typeID].Source = "mixed"
		}
	}
	return restaurants, flattenTypes(typeMap)
}

func cleanIDs(next []string, fallback []string) []string {
	cleaned := []string{}
	for _, value := range next {
		value = strings.TrimSpace(value)
		if value != "" {
			cleaned = appendUnique(cleaned, value)
		}
	}
	if len(cleaned) == 0 {
		return fallback
	}
	return cleaned
}

func cleanTags(next []string, fallback []string) []string {
	cleaned := append([]string{}, fallback...)
	for _, value := range next {
		value = strings.TrimSpace(value)
		if value != "" {
			cleaned = appendUnique(cleaned, value)
		}
	}
	return cleaned
}

func labelForType(typeID string) string {
	for _, rule := range foodRules {
		if rule.ID == typeID {
			return rule.Label
		}
	}
	if typeID == "type-other" {
		return "其他好吃的"
	}
	return strings.TrimPrefix(typeID, "type-")
}
```

- [ ] **Step 4: Run tests**

Run:

```bash
go test ./internal/tagging
```

Expected: PASS.

- [ ] **Step 5: Commit**

Run:

```bash
git add internal/tagging
git commit -m "feat: merge LLM restaurant tags"
```

---

### Task 8: HTTP API Router With Mock Providers

**Files:**
- Create: `internal/httpapi/server.go`
- Create: `internal/httpapi/test_fakes.go`
- Create: `internal/httpapi/server_test.go`
- Create: `api/rooms.go`

- [ ] **Step 1: Write failing handler test for create and join**

Create `internal/httpapi/server_test.go`:

```go
package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Alanxtl/no-more-food-drama/internal/roomstore"
)

func TestCreateAndJoinRoom(t *testing.T) {
	server := NewServer(Config{
		AppURL: "https://app.test",
		Store: roomstore.NewMemoryStore(),
		Restaurants: FakeRestaurantProvider{},
	})

	createReq := httptest.NewRequest(http.MethodPost, "/api/rooms", nil)
	createRec := httptest.NewRecorder()
	server.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusOK {
		t.Fatalf("create status = %d body = %s", createRec.Code, createRec.Body.String())
	}
	var createBody envelope
	if err := json.Unmarshal(createRec.Body.Bytes(), &createBody); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	roomID := createBody.Data["roomId"].(string)

	joinReq := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/join", nil)
	joinRec := httptest.NewRecorder()
	server.ServeHTTP(joinRec, joinReq)
	if joinRec.Code != http.StatusOK {
		t.Fatalf("join status = %d body = %s", joinRec.Code, joinRec.Body.String())
	}
}

func TestSearchWritesRuleTaggedRestaurants(t *testing.T) {
	server := NewServer(Config{
		AppURL: "https://app.test",
		Store: roomstore.NewMemoryStore(),
		Restaurants: FakeRestaurantProvider{},
	})
	createRec := httptest.NewRecorder()
	server.ServeHTTP(createRec, httptest.NewRequest(http.MethodPost, "/api/rooms", nil))
	var createBody envelope
	_ = json.Unmarshal(createRec.Body.Bytes(), &createBody)
	roomID := createBody.Data["roomId"].(string)

	body := bytes.NewBufferString(`{"lat":23.09,"lng":113.32,"radiusKm":3,"limit":20}`)
	req := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/search", body)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("search status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("type-japanese")) {
		t.Fatalf("search body missing type-japanese: %s", rec.Body.String())
	}
}

type envelope struct {
	OK bool `json:"ok"`
	Data map[string]any `json:"data"`
	Error any `json:"error"`
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/httpapi
```

Expected: FAIL because `NewServer`, `Config`, and fakes are undefined.

- [ ] **Step 3: Implement fake provider**

Create `internal/httpapi/test_fakes.go`:

```go
package httpapi

import (
	"context"

	"github.com/Alanxtl/no-more-food-drama/internal/domain"
)

type FakeRestaurantProvider struct{}

func (FakeRestaurantProvider) SearchAround(ctx context.Context, lat float64, lng float64, radiusKM int, limit int) ([]domain.Restaurant, error) {
	open := true
	return []domain.Restaurant{
		{ID: "amap:test-sushi", Provider: "amap", ProviderID: "test-sushi", Name: "鮨小野", Address: "测试路 1 号", Lat: lat, Lng: lng, DistanceMeters: 650, Rating: 4.7, AvgPriceCNY: 128, OpenNow: &open, Categories: []string{"餐饮服务", "日本料理"}},
		{ID: "amap:test-hotpot", Provider: "amap", ProviderID: "test-hotpot", Name: "热辣火锅", Address: "测试路 2 号", Lat: lat, Lng: lng, DistanceMeters: 900, Rating: 4.5, AvgPriceCNY: 98, OpenNow: &open, Categories: []string{"餐饮服务", "火锅"}},
	}, nil
}
```

- [ ] **Step 4: Implement router and handlers**

Create `internal/httpapi/server.go` with routes:

```go
package httpapi

import (
	"context"
	"encoding/json"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/Alanxtl/no-more-food-drama/internal/domain"
	"github.com/Alanxtl/no-more-food-drama/internal/recommend"
	"github.com/Alanxtl/no-more-food-drama/internal/roomstore"
	"github.com/Alanxtl/no-more-food-drama/internal/tagging"
)

type RestaurantProvider interface {
	SearchAround(ctx context.Context, lat float64, lng float64, radiusKM int, limit int) ([]domain.Restaurant, error)
}

type Config struct {
	AppURL      string
	Store       roomstore.Store
	Restaurants RestaurantProvider
}

type Server struct {
	config Config
	now    func() time.Time
}

func NewServer(config Config) *Server {
	return &Server{config: config, now: time.Now}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	path := strings.TrimPrefix(r.URL.Path, "/api/rooms")
	if r.Method == http.MethodPost && path == "" {
		s.createRoom(w, r)
		return
	}
	parts := splitPath(path)
	if len(parts) == 1 && r.Method == http.MethodGet {
		s.snapshot(w, r, parts[0])
		return
	}
	if len(parts) == 2 && r.Method == http.MethodPost {
		switch parts[1] {
		case "join":
			s.joinRoom(w, r, parts[0])
		case "search":
			s.search(w, r, parts[0])
		case "recommendations":
			s.recommendations(w, r, parts[0])
		default:
			writeFailure(w, http.StatusNotFound, domain.ErrorValidation, "未知接口")
		}
		return
	}
	writeFailure(w, http.StatusNotFound, domain.ErrorValidation, "未知接口")
}

func (s *Server) createRoom(w http.ResponseWriter, r *http.Request) {
	now := s.now()
	roomID := randomRoomID()
	shareURL := strings.TrimRight(s.config.AppURL, "/") + "/room/" + roomID
	room, participantID := domain.NewRoom(roomID, shareURL, now)
	if err := s.config.Store.Save(r.Context(), room, domain.RoomTTL); err != nil {
		writeFailure(w, http.StatusInternalServerError, domain.ErrorProvider, "房间创建失败")
		return
	}
	writeSuccess(w, map[string]any{"roomId": room.ID, "participantId": participantID, "shareUrl": shareURL, "room": room})
}

func (s *Server) joinRoom(w http.ResponseWriter, r *http.Request, roomID string) {
	var participantID string
	room, err := s.config.Store.Update(r.Context(), roomID, domain.RoomTTL, func(room domain.Room) (domain.Room, error) {
		participantID = room.JoinPartner(s.now())
		return room, nil
	})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeSuccess(w, map[string]any{"participantId": participantID, "room": room})
}

func (s *Server) snapshot(w http.ResponseWriter, r *http.Request, roomID string) {
	room, err := s.config.Store.Get(r.Context(), roomID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeSuccess(w, map[string]any{"room": room})
}

func (s *Server) search(w http.ResponseWriter, r *http.Request, roomID string) {
	var input struct {
		Lat      float64 `json:"lat"`
		Lng      float64 `json:"lng"`
		RadiusKM int     `json:"radiusKm"`
		Limit    int     `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeFailure(w, http.StatusBadRequest, domain.ErrorValidation, "搜索参数无效")
		return
	}
	restaurants, err := s.config.Restaurants.SearchAround(r.Context(), input.Lat, input.Lng, input.RadiusKM, input.Limit)
	if err != nil {
		writeFailure(w, http.StatusBadGateway, domain.ErrorProvider, "附近餐厅搜索失败")
		return
	}
	tagged, types := tagging.BuildRuleTags(restaurants)
	room, err := s.config.Store.Update(r.Context(), roomID, domain.RoomTTL, func(room domain.Room) (domain.Room, error) {
		room.SearchConfig = &domain.SearchConfig{Lat: input.Lat, Lng: input.Lng, RadiusKM: input.RadiusKM, Limit: input.Limit}
		room.Restaurants = tagged
		room.Types = types
		room.Status = domain.StatusFiltering
		room.Version++
		room.ExpiresAt = s.now().Add(domain.RoomTTL)
		return room, nil
	})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeSuccess(w, map[string]any{"room": room})
}

func (s *Server) recommendations(w http.ResponseWriter, r *http.Request, roomID string) {
	room, err := s.config.Store.Update(r.Context(), roomID, domain.RoomTTL, func(room domain.Room) (domain.Room, error) {
		room.Recommendations = recommend.Compute(room, 5)
		room.Status = domain.StatusResults
		room.Version++
		room.ExpiresAt = s.now().Add(domain.RoomTTL)
		return room, nil
	})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeSuccess(w, map[string]any{"room": room})
}

func writeSuccess(w http.ResponseWriter, data any) {
	_ = json.NewEncoder(w).Encode(domain.Success(data))
}

func writeFailure(w http.ResponseWriter, status int, code string, message string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(domain.Failure(code, message))
}

func writeStoreError(w http.ResponseWriter, err error) {
	switch err {
	case roomstore.ErrRoomExpired:
		writeFailure(w, http.StatusGone, domain.ErrorRoomExpired, "房间已过期，请重新创建")
	case roomstore.ErrRoomNotFound:
		writeFailure(w, http.StatusNotFound, domain.ErrorRoomNotFound, "房间不存在")
	default:
		writeFailure(w, http.StatusInternalServerError, domain.ErrorProvider, "房间状态更新失败")
	}
}

func splitPath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}

func randomRoomID() string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	out := make([]byte, 6)
	for i := range out {
		out[i] = alphabet[rand.Intn(len(alphabet))]
	}
	return string(out)
}
```

- [ ] **Step 5: Add Vercel Go entrypoint**

Create `api/rooms.go`:

```go
package handler

import (
	"net/http"
	"os"

	"github.com/Alanxtl/no-more-food-drama/internal/httpapi"
	"github.com/Alanxtl/no-more-food-drama/internal/roomstore"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	store := roomstore.NewMemoryStore()
	server := httpapi.NewServer(httpapi.Config{
		AppURL: env("NEXT_PUBLIC_APP_URL", "http://localhost:3000"),
		Store: store,
		Restaurants: httpapi.FakeRestaurantProvider{},
	})
	server.ServeHTTP(w, r)
}

func env(name string, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}
```

This entrypoint intentionally uses the fake provider and memory store first so the route compiles. A later task replaces runtime wiring with Upstash and Amap based on environment variables.

- [ ] **Step 6: Run tests**

Run:

```bash
go test ./internal/httpapi ./...
```

Expected: PASS.

- [ ] **Step 7: Commit**

Run:

```bash
git add internal/httpapi api/rooms.go
git commit -m "feat: add room HTTP API"
```

---

### Task 9: Type Votes and Restaurant Overrides API

**Files:**
- Modify: `internal/httpapi/server.go`
- Modify: `internal/httpapi/server_test.go`

- [ ] **Step 1: Add failing tests for type and restaurant votes**

Append to `internal/httpapi/server_test.go`:

```go
func TestTypeVoteEndpointUpdatesParticipantVote(t *testing.T) {
	server := NewServer(Config{AppURL: "https://app.test", Store: roomstore.NewMemoryStore(), Restaurants: FakeRestaurantProvider{}})
	createRec := httptest.NewRecorder()
	server.ServeHTTP(createRec, httptest.NewRequest(http.MethodPost, "/api/rooms", nil))
	var createBody envelope
	_ = json.Unmarshal(createRec.Body.Bytes(), &createBody)
	roomID := createBody.Data["roomId"].(string)
	participantID := createBody.Data["participantId"].(string)

	body := bytes.NewBufferString(`{"participantId":"` + participantID + `","typeId":"type-hotpot","vote":"avoid"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/votes/type", body)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("vote status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"type-hotpot":"avoid"`)) {
		t.Fatalf("vote body missing avoid: %s", rec.Body.String())
	}
}

func TestRestaurantOverrideEndpointUpdatesHardRemove(t *testing.T) {
	server := NewServer(Config{AppURL: "https://app.test", Store: roomstore.NewMemoryStore(), Restaurants: FakeRestaurantProvider{}})
	createRec := httptest.NewRecorder()
	server.ServeHTTP(createRec, httptest.NewRequest(http.MethodPost, "/api/rooms", nil))
	var createBody envelope
	_ = json.Unmarshal(createRec.Body.Bytes(), &createBody)
	roomID := createBody.Data["roomId"].(string)
	participantID := createBody.Data["participantId"].(string)

	body := bytes.NewBufferString(`{"participantId":"` + participantID + `","restaurantId":"amap:test-hotpot","override":"remove"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/votes/restaurant", body)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("override status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"amap:test-hotpot":"remove"`)) {
		t.Fatalf("override body missing remove: %s", rec.Body.String())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/httpapi
```

Expected: FAIL with 404 for `/votes/type` and `/votes/restaurant`.

- [ ] **Step 3: Add vote routes and handlers**

Modify the route switch in `internal/httpapi/server.go` so paths with three segments are handled:

```go
if len(parts) == 3 && r.Method == http.MethodPost && parts[1] == "votes" {
	switch parts[2] {
	case "type":
		s.typeVote(w, r, parts[0])
	case "restaurant":
		s.restaurantOverride(w, r, parts[0])
	default:
		writeFailure(w, http.StatusNotFound, domain.ErrorValidation, "未知接口")
	}
	return
}
```

Add these handlers:

```go
func (s *Server) typeVote(w http.ResponseWriter, r *http.Request, roomID string) {
	var input struct {
		ParticipantID string          `json:"participantId"`
		TypeID        string          `json:"typeId"`
		Vote          domain.TypeVote `json:"vote"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.ParticipantID == "" || input.TypeID == "" {
		writeFailure(w, http.StatusBadRequest, domain.ErrorValidation, "类型选择参数无效")
		return
	}
	room, err := s.config.Store.Update(r.Context(), roomID, domain.RoomTTL, func(room domain.Room) (domain.Room, error) {
		if err := room.SetTypeVote(input.ParticipantID, input.TypeID, input.Vote, s.now()); err != nil {
			return domain.Room{}, err
		}
		return room, nil
	})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeSuccess(w, map[string]any{"room": room})
}

func (s *Server) restaurantOverride(w http.ResponseWriter, r *http.Request, roomID string) {
	var input struct {
		ParticipantID string                    `json:"participantId"`
		RestaurantID  string                    `json:"restaurantId"`
		Override      domain.RestaurantOverride `json:"override"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.ParticipantID == "" || input.RestaurantID == "" {
		writeFailure(w, http.StatusBadRequest, domain.ErrorValidation, "餐厅选择参数无效")
		return
	}
	room, err := s.config.Store.Update(r.Context(), roomID, domain.RoomTTL, func(room domain.Room) (domain.Room, error) {
		if err := room.SetRestaurantOverride(input.ParticipantID, input.RestaurantID, input.Override, s.now()); err != nil {
			return domain.Room{}, err
		}
		return room, nil
	})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeSuccess(w, map[string]any{"room": room})
}
```

- [ ] **Step 4: Run tests**

Run:

```bash
go test ./internal/httpapi ./...
```

Expected: PASS.

- [ ] **Step 5: Commit**

Run:

```bash
git add internal/httpapi
git commit -m "feat: add room voting endpoints"
```

---

### Task 10: LLM Tag Enhancement API

**Files:**
- Modify: `internal/httpapi/server.go`
- Modify: `internal/httpapi/server_test.go`
- Modify: `internal/httpapi/test_fakes.go`
- Create: `internal/httpapi/providers.go`

- [ ] **Step 1: Add failing tag endpoint test**

Append to `internal/httpapi/server_test.go`:

```go
func TestTagEndpointMergesLLMEnhancements(t *testing.T) {
	server := NewServer(Config{AppURL: "https://app.test", Store: roomstore.NewMemoryStore(), Restaurants: FakeRestaurantProvider{}, Tagger: FakeTagger{}})
	createRec := httptest.NewRecorder()
	server.ServeHTTP(createRec, httptest.NewRequest(http.MethodPost, "/api/rooms", nil))
	var createBody envelope
	_ = json.Unmarshal(createRec.Body.Bytes(), &createBody)
	roomID := createBody.Data["roomId"].(string)

	searchBody := bytes.NewBufferString(`{"lat":23.09,"lng":113.32,"radiusKm":3,"limit":20}`)
	searchReq := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/search", searchBody)
	server.ServeHTTP(httptest.NewRecorder(), searchReq)

	tagBody := bytes.NewBufferString(`{"apiKey":"sk-test","baseUrl":"https://api.example.com/v1","model":"deepseek-chat"}`)
	tagReq := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/tag", tagBody)
	tagRec := httptest.NewRecorder()
	server.ServeHTTP(tagRec, tagReq)

	if tagRec.Code != http.StatusOK {
		t.Fatalf("tag status = %d body = %s", tagRec.Code, tagRec.Body.String())
	}
	if !bytes.Contains(tagRec.Body.Bytes(), []byte("漂亮饭")) {
		t.Fatalf("tag body missing LLM tag: %s", tagRec.Body.String())
	}
}
```

Add to `internal/httpapi/test_fakes.go`:

```go
type FakeTagger struct{}

func (FakeTagger) Enhance(ctx context.Context, restaurants []domain.Restaurant, apiKey string, baseURL string, model string) (llm.EnhancementResult, error) {
	return llm.EnhancementResult{
		Restaurants: []llm.RestaurantEnhancement{
			{ID: "amap:test-sushi", TypeIDs: []string{"type-japanese"}, Tags: []string{"漂亮饭", "约会友好"}},
		},
	}, nil
}
```

The fake file needs this import:

```go
import "github.com/Alanxtl/no-more-food-drama/internal/llm"
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/httpapi
```

Expected: FAIL because `Config.Tagger` and `/tag` are undefined.

- [ ] **Step 3: Add Tagger interface and route**

Add to `internal/httpapi/server.go`:

```go
type Tagger interface {
	Enhance(ctx context.Context, restaurants []domain.Restaurant, apiKey string, baseURL string, model string) (llm.EnhancementResult, error)
}
```

Add `Tagger Tagger` to `Config`.

Add route handling for `POST /api/rooms/{roomId}/tag` in the existing two-segment switch:

```go
case "tag":
	s.tag(w, r, parts[0])
```

Add handler:

```go
func (s *Server) tag(w http.ResponseWriter, r *http.Request, roomID string) {
	var input struct {
		APIKey  string `json:"apiKey"`
		BaseURL string `json:"baseUrl"`
		Model   string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.APIKey == "" || input.BaseURL == "" || input.Model == "" {
		writeFailure(w, http.StatusBadRequest, domain.ErrorValidation, "LLM 配置无效")
		return
	}
	current, err := s.config.Store.Get(r.Context(), roomID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	result, err := s.config.Tagger.Enhance(r.Context(), current.Restaurants, input.APIKey, input.BaseURL, input.Model)
	if err != nil {
		writeFailure(w, http.StatusBadGateway, domain.ErrorProvider, "LLM 标签生成失败，已保留规则标签")
		return
	}
	room, err := s.config.Store.Update(r.Context(), roomID, domain.RoomTTL, func(room domain.Room) (domain.Room, error) {
		restaurants, types := tagging.MergeLLMEnhancements(room.Restaurants, result)
		room.Restaurants = restaurants
		room.Types = types
		room.Status = domain.StatusFiltering
		room.Version++
		room.ExpiresAt = s.now().Add(domain.RoomTTL)
		return room, nil
	})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeSuccess(w, map[string]any{"room": room})
}
```

Add this import where the `Tagger` interface is defined:

```go
import "github.com/Alanxtl/no-more-food-drama/internal/llm"
```

- [ ] **Step 4: Add production LLM tagger**

Create `internal/httpapi/providers.go`:

```go
package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"os"

	"github.com/Alanxtl/no-more-food-drama/internal/domain"
	"github.com/Alanxtl/no-more-food-drama/internal/llm"
)

func useMockProviders() bool {
	return os.Getenv("USE_MOCK_PROVIDERS") == "true"
}

type LLMTagger struct {
	HTTPClient *http.Client
}

func (t LLMTagger) Enhance(ctx context.Context, restaurants []domain.Restaurant, apiKey string, baseURL string, model string) (llm.EnhancementResult, error) {
	payload, err := compactRestaurants(restaurants)
	if err != nil {
		return llm.EnhancementResult{}, err
	}
	client := llm.NewClient(baseURL, apiKey, model, t.HTTPClient)
	return client.EnhanceTags(ctx, payload)
}

func compactRestaurants(restaurants []domain.Restaurant) (string, error) {
	type compact struct {
		ID         string   `json:"id"`
		Name       string   `json:"name"`
		Categories []string `json:"categories"`
		Tags       []string `json:"tags"`
	}
	items := make([]compact, 0, len(restaurants))
	for _, restaurant := range restaurants {
		items = append(items, compact{ID: restaurant.ID, Name: restaurant.Name, Categories: restaurant.Categories, Tags: restaurant.Tags})
	}
	out, err := json.Marshal(map[string]any{"restaurants": items})
	if err != nil {
		return "", err
	}
	return string(out), nil
}
```

Wire a `Tagger` in every `NewServer(Config{...})` call in tests and production. Tests use `FakeTagger{}`. Production uses `LLMTagger{}`.

- [ ] **Step 5: Run tests**

Run:

```bash
go test ./internal/httpapi ./internal/tagging ./...
```

Expected: PASS.

- [ ] **Step 6: Commit**

Run:

```bash
git add internal/httpapi internal/tagging api/rooms.go
git commit -m "feat: add LLM tag endpoint"
```

---

### Task 11: Frontend API Client and Session Helpers

**Files:**
- Create: `app/lib/types.ts`
- Create: `app/lib/session.ts`
- Create: `app/lib/api.ts`
- Create: `app/tests/session.test.ts`
- Create: `app/tests/api.test.ts`

- [ ] **Step 1: Write failing session tests**

Create `app/tests/session.test.ts`:

```ts
import { describe, expect, it, beforeEach } from "vitest";
import { loadLlmConfig, saveLlmConfig } from "@/app/lib/session";

describe("LLM config session storage", () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

  it("stores LLM config only in sessionStorage", () => {
    saveLlmConfig({
      apiKey: "sk-test",
      baseUrl: "https://api.example.com/v1",
      model: "deepseek-chat"
    });

    expect(loadLlmConfig()).toEqual({
      apiKey: "sk-test",
      baseUrl: "https://api.example.com/v1",
      model: "deepseek-chat"
    });
    expect(localStorage.getItem("llmConfig")).toBeNull();
  });
});
```

- [ ] **Step 2: Write failing API client test**

Create `app/tests/api.test.ts`:

```ts
import { describe, expect, it, vi } from "vitest";
import { createRoom } from "@/app/lib/api";

describe("api client", () => {
  it("unwraps successful API responses", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => ({
      ok: true,
      json: async () => ({ ok: true, data: { roomId: "ABC123" }, error: null })
    })));

    await expect(createRoom()).resolves.toEqual({ roomId: "ABC123" });
  });
});
```

- [ ] **Step 3: Run tests to verify they fail**

Run:

```bash
npm run test -- app/tests/session.test.ts app/tests/api.test.ts
```

Expected: FAIL because `app/lib/session` and `app/lib/api` are missing.

- [ ] **Step 4: Implement frontend types**

Create `app/lib/types.ts` with JSON-compatible types:

```ts
export type RoomStatus = "lobby" | "searching" | "tagging" | "filtering" | "results";
export type Role = "creator" | "partner";
export type TypeVote = "want" | "neutral" | "avoid";
export type RestaurantOverride = "keep" | "remove";

export type LlmConfig = {
  apiKey: string;
  baseUrl: string;
  model: string;
};

export type Room = {
  id: string;
  version: number;
  shareUrl: string;
  createdAt: string;
  expiresAt: string;
  status: RoomStatus;
  searchConfig?: SearchConfig;
  participants: Record<string, Participant>;
  restaurants: Restaurant[];
  types: FoodType[];
  recommendations: Recommendation[];
};

export type SearchConfig = {
  locationText?: string;
  lat?: number;
  lng?: number;
  radiusKm: number;
  limit: number;
};

export type Participant = {
  displayName: string;
  role: Role;
  joinedAt: string;
  lastSeenAt: string;
  typeVotes: Record<string, TypeVote>;
  restaurantOverrides: Record<string, RestaurantOverride>;
};

export type Restaurant = {
  id: string;
  provider: "amap";
  providerId: string;
  name: string;
  address: string;
  lat: number;
  lng: number;
  distanceMeters: number;
  rating?: number;
  priceLevel?: string;
  avgPriceCny?: number;
  openNow?: boolean;
  categories: string[];
  typeIds: string[];
  tags: string[];
};

export type FoodType = {
  id: string;
  label: string;
  source: "rules" | "llm" | "mixed";
  tags: string[];
  restaurantIds: string[];
  stats: {
    count: number;
    nearestMeters: number;
    avgRating?: number;
    avgPriceCny?: number;
  };
};

export type Recommendation = {
  restaurantId: string;
  score: number;
  rank: number;
  reasons: string[];
  warnings: string[];
};
```

- [ ] **Step 5: Implement session and API helpers**

Create `app/lib/session.ts`:

```ts
import type { LlmConfig } from "./types";

const LLM_CONFIG_KEY = "llmConfig";
const PARTICIPANT_KEY_PREFIX = "participant:";

export function saveLlmConfig(config: LlmConfig) {
  sessionStorage.setItem(LLM_CONFIG_KEY, JSON.stringify(config));
}

export function loadLlmConfig(): LlmConfig | null {
  const raw = sessionStorage.getItem(LLM_CONFIG_KEY);
  if (!raw) {
    return null;
  }
  return JSON.parse(raw) as LlmConfig;
}

export function saveParticipant(roomId: string, participantId: string) {
  sessionStorage.setItem(PARTICIPANT_KEY_PREFIX + roomId, participantId);
}

export function loadParticipant(roomId: string): string | null {
  return sessionStorage.getItem(PARTICIPANT_KEY_PREFIX + roomId);
}
```

Create `app/lib/api.ts`:

```ts
import type { Room, TypeVote, RestaurantOverride } from "./types";

type ApiEnvelope<T> = {
  ok: boolean;
  data: T | null;
  error: null | { code: string; message: string };
};

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {})
    }
  });
  const envelope = (await response.json()) as ApiEnvelope<T>;
  if (!response.ok || !envelope.ok || !envelope.data) {
    throw new Error(envelope.error?.message ?? "请求失败");
  }
  return envelope.data;
}

export function createRoom() {
  return request<{ roomId: string; participantId: string; shareUrl: string; room: Room }>("/api/rooms", {
    method: "POST"
  });
}

export function joinRoom(roomId: string) {
  return request<{ participantId: string; room: Room }>(`/api/rooms/${roomId}/join`, {
    method: "POST"
  });
}

export function getRoom(roomId: string) {
  return request<{ room: Room }>(`/api/rooms/${roomId}`);
}

export function searchRestaurants(roomId: string, input: { lat: number; lng: number; radiusKm: number; limit: number }) {
  return request<{ room: Room }>(`/api/rooms/${roomId}/search`, {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export function voteType(roomId: string, input: { participantId: string; typeId: string; vote: TypeVote }) {
  return request<{ room: Room }>(`/api/rooms/${roomId}/votes/type`, {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export function overrideRestaurant(roomId: string, input: { participantId: string; restaurantId: string; override: RestaurantOverride }) {
  return request<{ room: Room }>(`/api/rooms/${roomId}/votes/restaurant`, {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export function computeRecommendations(roomId: string) {
  return request<{ room: Room }>(`/api/rooms/${roomId}/recommendations`, {
    method: "POST"
  });
}

export function tagRoom(roomId: string, input: { apiKey: string; baseUrl: string; model: string }) {
  return request<{ room: Room }>(`/api/rooms/${roomId}/tag`, {
    method: "POST",
    body: JSON.stringify(input)
  });
}
```

- [ ] **Step 6: Run tests**

Run:

```bash
npm run test -- app/tests/session.test.ts app/tests/api.test.ts
```

Expected: PASS.

- [ ] **Step 7: Commit**

Run:

```bash
git add app/lib app/tests
git commit -m "feat: add frontend API helpers"
```

---

### Task 12: Home Setup and Room Lobby UI

**Files:**
- Create: `app/components/HomeSetup.tsx`
- Create: `app/components/RoomLobby.tsx`
- Modify: `app/page.tsx`
- Create: `app/tests/HomeSetup.test.tsx`
- Create: `app/tests/RoomLobby.test.tsx`

- [ ] **Step 1: Write failing HomeSetup test**

Create `app/tests/HomeSetup.test.tsx`:

```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import HomeSetup from "@/app/components/HomeSetup";

describe("HomeSetup", () => {
  it("saves LLM config and creates a room", async () => {
    const onCreateRoom = vi.fn(async () => {});
    render(<HomeSetup onCreateRoom={onCreateRoom} />);

    await userEvent.type(screen.getByLabelText("API Key"), "sk-test");
    await userEvent.type(screen.getByLabelText("Base URL"), "https://api.example.com/v1");
    await userEvent.type(screen.getByLabelText("Model"), "deepseek-chat");
    await userEvent.click(screen.getByRole("button", { name: "创建双人房间" }));

    expect(onCreateRoom).toHaveBeenCalledWith({
      apiKey: "sk-test",
      baseUrl: "https://api.example.com/v1",
      model: "deepseek-chat"
    });
  });
});
```

- [ ] **Step 2: Write failing RoomLobby test**

Create `app/tests/RoomLobby.test.tsx`:

```tsx
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import RoomLobby from "@/app/components/RoomLobby";

describe("RoomLobby", () => {
  it("shows room code and share url", () => {
    render(<RoomLobby roomId="ABC123" shareUrl="https://app.test/room/ABC123" partnerOnline={false} />);

    expect(screen.getByText("ABC123")).toBeInTheDocument();
    expect(screen.getByText("https://app.test/room/ABC123")).toBeInTheDocument();
    expect(screen.getByText("等待另一位加入")).toBeInTheDocument();
  });
});
```

- [ ] **Step 3: Run tests to verify they fail**

Run:

```bash
npm run test -- app/tests/HomeSetup.test.tsx app/tests/RoomLobby.test.tsx
```

Expected: FAIL because components are missing.

- [ ] **Step 4: Implement HomeSetup**

Create `app/components/HomeSetup.tsx`:

```tsx
"use client";

import { KeyRound, Plus } from "lucide-react";
import type { LlmConfig } from "@/app/lib/types";

type Props = {
  onCreateRoom: (config: LlmConfig | null) => Promise<void> | void;
};

export default function HomeSetup({ onCreateRoom }: Props) {
  async function handleSubmit(formData: FormData) {
    const apiKey = String(formData.get("apiKey") ?? "").trim();
    const baseUrl = String(formData.get("baseUrl") ?? "").trim();
    const model = String(formData.get("model") ?? "").trim();
    const config = apiKey && baseUrl && model ? { apiKey, baseUrl, model } : null;
    await onCreateRoom(config);
  }

  return (
    <form action={handleSubmit} className="mx-auto flex min-h-screen w-full max-w-md flex-col justify-center px-5 py-8">
      <p className="text-sm text-neutral-600">no-more-food-drama</p>
      <h1 className="mt-2 text-4xl font-bold leading-tight">让你选你又不选</h1>
      <p className="mt-4 text-base leading-7 text-neutral-700">先让附近餐厅排好队，再让两个人各自筛掉今天不想吃的类型。</p>

      <div className="mt-8 space-y-3 rounded-lg border border-line bg-white p-4">
        <div className="flex items-center gap-2 text-sm font-semibold">
          <KeyRound size={18} />
          LLM 配置
        </div>
        <label className="block text-sm">
          API Key
          <input name="apiKey" aria-label="API Key" className="mt-1 w-full rounded-md border border-line px-3 py-2" type="password" />
        </label>
        <label className="block text-sm">
          Base URL
          <input name="baseUrl" aria-label="Base URL" className="mt-1 w-full rounded-md border border-line px-3 py-2" placeholder="https://api.openai.com/v1" />
        </label>
        <label className="block text-sm">
          Model
          <input name="model" aria-label="Model" className="mt-1 w-full rounded-md border border-line px-3 py-2" placeholder="gpt-4o-mini" />
        </label>
        <p className="text-xs leading-5 text-neutral-500">不填写也能继续，系统会先用规则标签。</p>
      </div>

      <button className="mt-5 inline-flex h-12 items-center justify-center gap-2 rounded-md bg-accent px-4 font-semibold text-white" type="submit">
        <Plus size={18} />
        创建双人房间
      </button>
    </form>
  );
}
```

- [ ] **Step 5: Implement RoomLobby and wire page**

Create `app/components/RoomLobby.tsx`:

```tsx
"use client";

import { Copy, QrCode } from "lucide-react";
import { QRCodeSVG } from "qrcode.react";

type Props = {
  roomId: string;
  shareUrl: string;
  partnerOnline: boolean;
};

export default function RoomLobby({ roomId, shareUrl, partnerOnline }: Props) {
  return (
    <section className="mx-auto w-full max-w-md px-5 py-8">
      <p className="text-sm text-neutral-600">房间码</p>
      <h1 className="mt-2 text-5xl font-bold tracking-normal">{roomId}</h1>

      <div className="mt-6 rounded-lg border border-line bg-white p-4">
        <div className="flex items-center gap-2 font-semibold">
          <QrCode size={18} />
          分享给另一位
        </div>
        <div className="mt-4 flex justify-center rounded-md bg-paper p-4">
          <QRCodeSVG value={shareUrl} size={160} />
        </div>
        <p className="mt-3 break-all text-sm text-neutral-700">{shareUrl}</p>
        <button className="mt-3 inline-flex h-10 items-center gap-2 rounded-md border border-line px-3 text-sm" type="button">
          <Copy size={16} />
          复制链接
        </button>
      </div>

      <p className="mt-4 text-sm text-neutral-700">{partnerOnline ? "另一位已加入" : "等待另一位加入"}</p>
    </section>
  );
}
```

Modify `app/page.tsx` to render `HomeSetup`:

```tsx
import HomeSetup from "./components/HomeSetup";

export default function HomePage() {
  return (
    <main className="min-h-screen bg-paper text-ink">
      <HomeSetup onCreateRoom={() => {}} />
    </main>
  );
}
```

- [ ] **Step 6: Run tests**

Run:

```bash
npm run test -- app/tests/HomeSetup.test.tsx app/tests/RoomLobby.test.tsx
```

Expected: PASS.

- [ ] **Step 7: Commit**

Run:

```bash
git add app/components app/page.tsx app/tests
git commit -m "feat: add home and lobby UI"
```

---

### Task 13: Room Flow UI With Search, Type Cards, and Results

**Files:**
- Create: `app/components/SearchSetup.tsx`
- Create: `app/components/TypeCard.tsx`
- Create: `app/components/ResultsList.tsx`
- Create: `app/room/[roomId]/page.tsx`
- Create: `app/tests/SearchSetup.test.tsx`
- Create: `app/tests/TypeCard.test.tsx`
- Create: `app/tests/ResultsList.test.tsx`

- [ ] **Step 1: Write failing TypeCard and Results tests**

Create `app/tests/TypeCard.test.tsx`:

```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import TypeCard from "@/app/components/TypeCard";
import type { FoodType, Restaurant } from "@/app/lib/types";

const foodType: FoodType = {
  id: "type-japanese",
  label: "日料",
  source: "rules",
  tags: ["约会友好", "清淡"],
  restaurantIds: ["r1"],
  stats: { count: 1, nearestMeters: 650, avgRating: 4.7, avgPriceCny: 128 }
};

const restaurants: Restaurant[] = [
  { id: "r1", provider: "amap", providerId: "r1", name: "鮨小野", address: "测试路", lat: 1, lng: 1, distanceMeters: 650, rating: 4.7, avgPriceCny: 128, categories: [], typeIds: ["type-japanese"], tags: ["约会友好"] }
];

describe("TypeCard", () => {
  it("votes on a food type", async () => {
    const onVote = vi.fn();
    render(<TypeCard foodType={foodType} restaurants={restaurants} onVote={onVote} />);

    expect(screen.getByText("日料")).toBeInTheDocument();
    expect(screen.getByText("鮨小野")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "今天不吃" }));

    expect(onVote).toHaveBeenCalledWith("type-japanese", "avoid");
  });
});
```

Create `app/tests/ResultsList.test.tsx`:

```tsx
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import ResultsList from "@/app/components/ResultsList";
import type { Recommendation, Restaurant } from "@/app/lib/types";

describe("ResultsList", () => {
  it("shows ranked recommendation reasons and warnings", () => {
    const restaurants: Restaurant[] = [
      { id: "r1", provider: "amap", providerId: "r1", name: "鮨小野", address: "测试路", lat: 1, lng: 1, distanceMeters: 650, rating: 4.7, categories: [], typeIds: [], tags: [] }
    ];
    const recommendations: Recommendation[] = [
      { restaurantId: "r1", rank: 1, score: 92, reasons: ["离你们 650m", "正在营业"], warnings: ["可能要排队"] }
    ];

    render(<ResultsList restaurants={restaurants} recommendations={recommendations} onRemove={() => {}} />);

    expect(screen.getByText("1. 鮨小野")).toBeInTheDocument();
    expect(screen.getByText("离你们 650m")).toBeInTheDocument();
    expect(screen.getByText("可能要排队")).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
npm run test -- app/tests/TypeCard.test.tsx app/tests/ResultsList.test.tsx
```

Expected: FAIL because components are missing.

- [ ] **Step 3: Implement SearchSetup**

Create `app/components/SearchSetup.tsx`:

```tsx
"use client";

import { MapPin, Search } from "lucide-react";

type Props = {
  onSearch: (input: { lat: number; lng: number; radiusKm: number; limit: number }) => void;
};

export default function SearchSetup({ onSearch }: Props) {
  function useDemoLocation() {
    onSearch({ lat: 23.09, lng: 113.32, radiusKm: 3, limit: 20 });
  }

  return (
    <section className="mx-auto w-full max-w-md px-5 py-8">
      <h1 className="text-3xl font-bold">先找附近能吃的</h1>
      <p className="mt-2 text-sm text-neutral-600">定位失败时可以先用当前城市商圈或测试位置继续。</p>
      <button onClick={useDemoLocation} className="mt-6 inline-flex h-12 w-full items-center justify-center gap-2 rounded-md bg-accent px-4 font-semibold text-white" type="button">
        <MapPin size={18} />
        使用测试位置搜索
      </button>
      <button onClick={useDemoLocation} className="mt-3 inline-flex h-11 w-full items-center justify-center gap-2 rounded-md border border-line bg-white px-4 font-semibold" type="button">
        <Search size={18} />
        手动地址兜底
      </button>
    </section>
  );
}
```

- [ ] **Step 4: Implement TypeCard and ResultsList**

Create `app/components/TypeCard.tsx`:

```tsx
"use client";

import type { FoodType, Restaurant, TypeVote } from "@/app/lib/types";

type Props = {
  foodType: FoodType;
  restaurants: Restaurant[];
  onVote: (typeId: string, vote: TypeVote) => void;
};

export default function TypeCard({ foodType, restaurants, onVote }: Props) {
  const preview = restaurants.filter((restaurant) => foodType.restaurantIds.includes(restaurant.id)).slice(0, 3);
  return (
    <section className="mx-auto w-full max-w-md px-5 py-8">
      <div className="rounded-lg border border-line bg-white p-5">
        <p className="text-sm text-neutral-600">{foodType.stats.count} 家候选 · 最近 {foodType.stats.nearestMeters}m</p>
        <h1 className="mt-2 text-5xl font-bold">{foodType.label}</h1>
        <div className="mt-4 flex flex-wrap gap-2">
          {foodType.tags.map((tag) => (
            <span key={tag} className="rounded-full bg-paper px-3 py-1 text-xs">{tag}</span>
          ))}
        </div>
        <div className="mt-5 space-y-2">
          {preview.map((restaurant) => (
            <div key={restaurant.id} className="rounded-md border border-line p-3">
              <p className="font-semibold">{restaurant.name}</p>
              <p className="mt-1 text-sm text-neutral-600">{restaurant.distanceMeters}m · {restaurant.rating ? restaurant.rating.toFixed(1) : "暂无评分"}</p>
            </div>
          ))}
        </div>
      </div>
      <div className="mt-4 grid grid-cols-3 gap-2">
        <button onClick={() => onVote(foodType.id, "avoid")} className="h-12 rounded-md border border-red-200 bg-white text-danger" type="button">今天不吃</button>
        <button onClick={() => onVote(foodType.id, "neutral")} className="h-12 rounded-md border border-line bg-white" type="button">无所谓</button>
        <button onClick={() => onVote(foodType.id, "want")} className="h-12 rounded-md bg-accent font-semibold text-white" type="button">可以吃</button>
      </div>
    </section>
  );
}
```

Create `app/components/ResultsList.tsx`:

```tsx
"use client";

import { X } from "lucide-react";
import type { Recommendation, Restaurant } from "@/app/lib/types";

type Props = {
  restaurants: Restaurant[];
  recommendations: Recommendation[];
  onRemove: (restaurantId: string) => void;
};

export default function ResultsList({ restaurants, recommendations, onRemove }: Props) {
  const byID = new Map(restaurants.map((restaurant) => [restaurant.id, restaurant]));
  return (
    <section className="mx-auto w-full max-w-md px-5 py-8">
      <h1 className="text-3xl font-bold">现在就去这几家</h1>
      <div className="mt-5 space-y-3">
        {recommendations.map((recommendation) => {
          const restaurant = byID.get(recommendation.restaurantId);
          if (!restaurant) {
            return null;
          }
          return (
            <article key={recommendation.restaurantId} className="rounded-lg border border-line bg-white p-4">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <h2 className="text-xl font-bold">{recommendation.rank}. {restaurant.name}</h2>
                  <p className="mt-1 text-sm text-neutral-600">{restaurant.distanceMeters}m · {restaurant.rating ? restaurant.rating.toFixed(1) : "暂无评分"}</p>
                </div>
                <button aria-label={`剔除 ${restaurant.name}`} onClick={() => onRemove(restaurant.id)} className="rounded-md border border-line p-2" type="button">
                  <X size={16} />
                </button>
              </div>
              <div className="mt-3 space-y-1 text-sm">
                {recommendation.reasons.map((reason) => <p key={reason}>{reason}</p>)}
                {recommendation.warnings.map((warning) => <p key={warning} className="text-danger">{warning}</p>)}
              </div>
            </article>
          );
        })}
      </div>
    </section>
  );
}
```

- [ ] **Step 5: Implement room page**

Create `app/room/[roomId]/page.tsx`:

```tsx
"use client";

import { useEffect, useMemo, useState } from "react";
import RoomLobby from "@/app/components/RoomLobby";
import SearchSetup from "@/app/components/SearchSetup";
import TypeCard from "@/app/components/TypeCard";
import ResultsList from "@/app/components/ResultsList";
import { computeRecommendations, getRoom, joinRoom, overrideRestaurant, searchRestaurants, tagRoom, voteType } from "@/app/lib/api";
import { loadLlmConfig, loadParticipant, saveParticipant } from "@/app/lib/session";
import type { Room, TypeVote } from "@/app/lib/types";

export default function RoomPage({ params }: { params: { roomId: string } }) {
  const roomId = params.roomId;
  const [participantId, setParticipantId] = useState<string | null>(null);
  const [room, setRoom] = useState<Room | null>(null);
  const [typeIndex, setTypeIndex] = useState(0);

  useEffect(() => {
    const existing = loadParticipant(roomId);
    if (existing) {
      setParticipantId(existing);
      getRoom(roomId).then((data) => setRoom(data.room));
      return;
    }
    joinRoom(roomId).then((data) => {
      saveParticipant(roomId, data.participantId);
      setParticipantId(data.participantId);
      setRoom(data.room);
    });
  }, [roomId]);

  useEffect(() => {
    const id = window.setInterval(() => {
      getRoom(roomId).then((data) => setRoom(data.room)).catch(() => {});
    }, 2000);
    return () => window.clearInterval(id);
  }, [roomId]);

  const partnerOnline = useMemo(() => room ? Object.keys(room.participants).length > 1 : false, [room]);

  if (!room || !participantId) {
    return <main className="min-h-screen bg-paper text-ink px-5 py-8">加入房间中...</main>;
  }

  if (room.restaurants.length === 0) {
    return (
      <main className="min-h-screen bg-paper text-ink">
        <RoomLobby roomId={room.id} shareUrl={room.shareUrl} partnerOnline={partnerOnline} />
        <SearchSetup onSearch={async (input) => {
          const searched = await searchRestaurants(roomId, input);
          setRoom(searched.room);
          const llmConfig = loadLlmConfig();
          if (llmConfig) {
            tagRoom(roomId, llmConfig).then((tagged) => setRoom(tagged.room)).catch(() => {});
          }
        }} />
      </main>
    );
  }

  if (room.recommendations.length > 0) {
    return (
      <main className="min-h-screen bg-paper text-ink">
        <ResultsList
          restaurants={room.restaurants}
          recommendations={room.recommendations}
          onRemove={async (restaurantId) => {
            const updated = await overrideRestaurant(roomId, { participantId, restaurantId, override: "remove" });
            const recomputed = await computeRecommendations(roomId);
            setRoom(recomputed.room ?? updated.room);
          }}
        />
      </main>
    );
  }

  const foodType = room.types[typeIndex] ?? room.types[0];
  return (
    <main className="min-h-screen bg-paper text-ink">
      {foodType ? (
        <TypeCard
          foodType={foodType}
          restaurants={room.restaurants}
          onVote={async (typeId: string, vote: TypeVote) => {
            const updated = await voteType(roomId, { participantId, typeId, vote });
            setRoom(updated.room);
            const nextIndex = typeIndex + 1;
            if (nextIndex >= updated.room.types.length) {
              setRoom((await computeRecommendations(roomId)).room);
            } else {
              setTypeIndex(nextIndex);
            }
          }}
        />
      ) : null}
    </main>
  );
}
```

- [ ] **Step 6: Run component tests**

Run:

```bash
npm run test -- app/tests/TypeCard.test.tsx app/tests/ResultsList.test.tsx
```

Expected: PASS.

- [ ] **Step 7: Commit**

Run:

```bash
git add app/components app/room app/tests
git commit -m "feat: add room decision UI"
```

---

### Task 14: Production Wiring for Upstash, Amap, and Mock Switch

**Files:**
- Modify: `api/rooms.go`
- Modify: `internal/httpapi/providers.go`

- [ ] **Step 1: Write failing production wiring test**

Create `internal/httpapi/providers_test.go`:

```go
package httpapi

import (
	"context"
	"testing"

	"github.com/Alanxtl/no-more-food-drama/internal/amap"
	"github.com/Alanxtl/no-more-food-drama/internal/domain"
)

type recordingAmapClient struct {
	request amap.SearchRequest
}

func (c *recordingAmapClient) SearchAround(ctx context.Context, request amap.SearchRequest) ([]domain.Restaurant, error) {
	c.request = request
	return []domain.Restaurant{{ID: "amap:test", Name: "测试餐厅"}}, nil
}

func TestAmapRestaurantProviderConvertsRadiusToMeters(t *testing.T) {
	client := &recordingAmapClient{}
	provider := AmapRestaurantProvider{Client: client}

	restaurants, err := provider.SearchAround(context.Background(), 23.09, 113.32, 3, 20)
	if err != nil {
		t.Fatalf("SearchAround returned error: %v", err)
	}

	if len(restaurants) != 1 {
		t.Fatalf("restaurants length = %d", len(restaurants))
	}
	if client.request.RadiusMeters != 3000 || client.request.Limit != 20 {
		t.Fatalf("request = %#v", client.request)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/httpapi
```

Expected: FAIL because `AmapRestaurantProvider` is undefined.

- [ ] **Step 3: Add provider wiring helpers**

Append to `internal/httpapi/providers.go`:

```go
type AmapSearchClient interface {
	SearchAround(ctx context.Context, request amap.SearchRequest) ([]domain.Restaurant, error)
}

type AmapRestaurantProvider struct {
	Client AmapSearchClient
}

func (p AmapRestaurantProvider) SearchAround(ctx context.Context, lat float64, lng float64, radiusKM int, limit int) ([]domain.Restaurant, error) {
	return p.Client.SearchAround(ctx, amap.SearchRequest{
		Lat: lat,
		Lng: lng,
		RadiusMeters: radiusKM * 1000,
		Limit: limit,
	})
}
```

`internal/httpapi/providers.go` needs these imports in addition to the existing imports:

```go
import (
	"context"

	"github.com/Alanxtl/no-more-food-drama/internal/amap"
	"github.com/Alanxtl/no-more-food-drama/internal/domain"
)
```

- [ ] **Step 4: Wire production entrypoint**

Modify `api/rooms.go` to use Upstash, Amap, and LLMTagger when `USE_MOCK_PROVIDERS` is not true:

```go
package handler

import (
	"net/http"
	"os"

	"github.com/Alanxtl/no-more-food-drama/internal/amap"
	"github.com/Alanxtl/no-more-food-drama/internal/httpapi"
	"github.com/Alanxtl/no-more-food-drama/internal/roomstore"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	var store roomstore.Store
	var restaurants httpapi.RestaurantProvider
	var tagger httpapi.Tagger

	if os.Getenv("USE_MOCK_PROVIDERS") == "true" {
		store = roomstore.NewMemoryStore()
		restaurants = httpapi.FakeRestaurantProvider{}
		tagger = httpapi.FakeTagger{}
	} else {
		store = roomstore.NewUpstashStore(requiredEnv("UPSTASH_REDIS_REST_URL"), requiredEnv("UPSTASH_REDIS_REST_TOKEN"), nil)
		amapClient := amap.NewClient(requiredEnv("AMAP_API_KEY"), "https://restapi.amap.com", nil)
		restaurants = httpapi.AmapRestaurantProvider{Client: amapClient}
		tagger = httpapi.LLMTagger{}
	}

	server := httpapi.NewServer(httpapi.Config{
		AppURL: env("NEXT_PUBLIC_APP_URL", "http://localhost:3000"),
		Store: store,
		Restaurants: restaurants,
		Tagger: tagger,
	})
	server.ServeHTTP(w, r)
}

func env(name string, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}

func requiredEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		panic(name + " is required")
	}
	return value
}
```

- [ ] **Step 5: Run Go tests**

Run:

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 6: Commit**

Run:

```bash
git add api/rooms.go internal/httpapi
git commit -m "feat: wire production providers"
```

---

### Task 15: E2E Smoke Test

**Files:**
- Create: `e2e/mvp.spec.ts`
- Modify: `app/page.tsx`
- Modify: `app/components/HomeSetup.tsx`

- [ ] **Step 1: Write failing E2E test**

Create `e2e/mvp.spec.ts`:

```ts
import { expect, test } from "@playwright/test";

test("two people can create a room, search, vote, and see recommendations", async ({ browser }) => {
  const creator = await browser.newPage();
  await creator.goto("/");
  await creator.getByRole("button", { name: "创建双人房间" }).click();
  await expect(creator.getByText("房间码")).toBeVisible();

  const roomUrl = creator.url();
  const partner = await browser.newPage();
  await partner.goto(roomUrl);
  await expect(partner.getByText("加入房间中...")).toBeVisible();

  await creator.getByRole("button", { name: "使用测试位置搜索" }).click();
  await expect(creator.getByText("日料")).toBeVisible();

  await creator.getByRole("button", { name: "可以吃" }).click();
  await partner.reload();
  await expect(partner.getByText("日料")).toBeVisible();
  await partner.getByRole("button", { name: "可以吃" }).click();

  await expect(creator.getByText("现在就去这几家")).toBeVisible({ timeout: 10_000 });
});
```

- [ ] **Step 2: Run E2E to verify it fails**

Run:

```bash
npm run e2e
```

Expected: FAIL because the home page does not call `createRoom` and route to `/room/{roomId}`.

- [ ] **Step 3: Wire home page to API**

Modify `app/page.tsx`:

```tsx
"use client";

import { useRouter } from "next/navigation";
import HomeSetup from "./components/HomeSetup";
import { createRoom } from "./lib/api";
import { saveLlmConfig, saveParticipant } from "./lib/session";
import type { LlmConfig } from "./lib/types";

export default function HomePage() {
  const router = useRouter();

  async function handleCreate(config: LlmConfig | null) {
    if (config) {
      saveLlmConfig(config);
    }
    const data = await createRoom();
    saveParticipant(data.roomId, data.participantId);
    router.push(`/room/${data.roomId}`);
  }

  return (
    <main className="min-h-screen bg-paper text-ink">
      <HomeSetup onCreateRoom={handleCreate} />
    </main>
  );
}
```

Ensure `HomeSetup` keeps accepting `LlmConfig | null`:

```tsx
type Props = {
  onCreateRoom: (config: LlmConfig | null) => Promise<void> | void;
};
```

- [ ] **Step 4: Run E2E**

Run:

```bash
npm run e2e
```

Expected: PASS on mobile-chrome project.

- [ ] **Step 5: Run full checks**

Run:

```bash
npm run test
npm run go:test
npm run build
```

Expected: all commands PASS.

- [ ] **Step 6: Commit**

Run:

```bash
git add app e2e
git commit -m "test: add MVP smoke flow"
```

---

### Task 16: README and Deployment Notes

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Replace README with setup instructions**

Write `README.md`:

```md
# 让你选你又不选

给情侣用的附近餐厅协商工具：地图找候选餐厅，LLM 归纳餐厅风格，两个人各自筛掉今天不想吃的类型，系统给出共同可接受的 Top 5。

## Local Development

```bash
npm install
npm run dev
```

Local mock mode:

```bash
USE_MOCK_PROVIDERS=true npm run dev
```

## Environment Variables

```bash
AMAP_API_KEY=
UPSTASH_REDIS_REST_URL=
UPSTASH_REDIS_REST_TOKEN=
NEXT_PUBLIC_APP_URL=http://localhost:3000
USE_MOCK_PROVIDERS=true
```

## Checks

```bash
npm run test
npm run go:test
npm run build
npm run e2e
```

## MVP Behavior

- One person creates a room and shares the room link or QR code.
- The creator searches nearby restaurants.
- Rule tags appear immediately.
- LLM tags can enhance results through an OpenAI-compatible user-provided key.
- Each person votes on food-type cards: 今天不吃, 无所谓, 可以吃.
- Type-level 今天不吃 is a soft penalty.
- Restaurant-level remove is a hard exclusion.
- Rooms expire after one hour.
```

- [ ] **Step 2: Run formatting and checks**

Run:

```bash
gofmt -w api internal
npm run test
npm run go:test
npm run build
```

Expected: all commands PASS.

- [ ] **Step 3: Commit**

Run:

```bash
git add README.md api internal app e2e
git commit -m "docs: add MVP setup guide"
```

---

## Plan Self-Review

Spec coverage:

- Vercel deployable app: Task 1 and Task 12.
- Go backend: Tasks 2 through 8 and Task 12.
- Upstash Redis one-hour TTL: Task 5 and Task 12.
- Amap provider with server-side key: Task 6 and Task 12.
- OpenAI-compatible LLM client and `/tag` route: Tasks 6, 7, 10, 11, and 13.
- Rule fallback tags: Task 3.
- Type-card flow: Task 11.
- Soft avoid and hard restaurant remove: Tasks 4 and 8.
- Room sharing: Task 10 and Task 13.
- Polling and session-local config: Tasks 9 and 11.
- Top 5 recommendation explanation: Tasks 4 and 11.
- Tests: each implementation task has failing tests and verification commands.

Placeholder scan:

- The plan avoids red-flag placeholder terms and unspecified test steps.
- Each implementation task names exact files and commands.
- Each behavior-changing task starts with a failing test.

Type consistency:

- Backend uses `TypeVote` values `want`, `neutral`, `avoid`.
- Frontend uses matching `TypeVote`.
- Backend uses `RestaurantOverride` values `keep`, `remove`.
- Frontend uses matching `RestaurantOverride`.
- Room IDs, participant IDs, restaurant IDs, and type IDs are passed as strings across the API.
