# MatchMark API contract

`openapi.yaml` is the hand-written OpenAPI 3.0 spec for every route in `internal/handler`. `TestOpenAPISpecMatchesRoutes` (`go test ./internal/handler`) fails when a route and the spec drift apart.

## Generate the TypeScript types

```sh
cd source/api
npm install
npm run generate
```

This writes `source/web/src/lib/api/schema.d.ts` (committed) using `openapi-typescript`. `npm run check` regenerates and fails if the committed file is stale.

## Using the client

`source/web/src/lib/api/client.ts` wraps the types with `openapi-fetch`; the web app needs `openapi-fetch` as a dependency.

```ts
import { api, deviceHeaders } from "./lib/api/client";

await api.POST("/api/admin/login", { body: { password } });
const { data } = await api.GET("/api/competitions/{competitionID}/entries", {
  params: { path: { competitionID: 1 } },
});
await api.PUT("/api/scorer/stages/{stage}", {
  params: { path: { stage: 1 } },
  headers: deviceHeaders(token),
  body: { x: 0, "10": 2, "9": 3, "8": 0, "7": 0, "0": 0, miss: 0 },
});
```
