# Component Architecture（UI設計の責務境界）

このドキュメントは、FSD（Feature-Sliced Design）を前提に、UIコンポーネントの責務と配置を固定します。

目的：
- UI部品が pages に閉じ込められる/肥大化するのを防ぐ
- 状態・副作用が UI に漏れて「どこで何が起きるか」を曖昧にしない
- 生成API / DAL / キャッシュ戦略と矛盾しない設計にする

関連：
- レイヤー：`FSD_LAYERS.md`
- DAL：`DATA_ACCESS_LAYER.md`
- SSR/Hydration：`../02_tech_stack/SSR_HYDRATION.md`

---

## 結論（Must）

- “画面合成” は `pages` / `widgets` が責務（features/entities を寄せ集める）
- “ユーザー価値の操作” は `features` が責務（フォーム、mutation、分岐）
- “ドメイン実体の表示” は `entities` が責務（表示と最小ロジック）
- “再利用UIプリミティブ” は `shared/ui`（ビジネスルール禁止）
- API 由来の生データを UI に丸渡ししない（DTO最小化はDAL/adapterで）

---

## 1) コンポーネント種別と置き場所

### shared/ui（UI primitives / wrappers）
- 例：Button、Input、Dialog、Table、Layout primitives
- 特徴：
  - 見た目/アクセシビリティ/テーマに責務を限定
  - ビジネスルール・API呼び出しは禁止

### entities/*/ui（Entity presentation）
- 例：`UserCard`、`ProductPrice`、`OrderStatusBadge`
- 特徴：
  - entity の表示に必要な最小ロジック
  - 他entityやfeatureの合成は禁止

### features/*/ui（Use case UI）
- 例：`LoginForm`、`AddToCartButton`、`UpdateEmailForm`
- 特徴：
  - 入力 → 検証 → mutation → 成功/失敗 の分岐
  - API hooks の利用はここ（もしくは features/model）

### widgets/*/ui（Composite blocks）
- 例：`AppHeader`、`SearchResultsPanel`
- 特徴：
  - 複数features/entitiesの合成
  - ページ固有の事情を持ちにくい“塊”

### pages/*/ui（Page composition）
- 例：`UserProfilePage`
- 特徴：
  - ルーティングに対応する画面（URL）
  - 画面専用の薄い整形（formatting）まで

---

## 2) Props（DTO）設計の原則

- Client Component に渡す props は **最小**（表示に必要なフィールドのみ）
- バックエンドレスポンス型（生成型）を、そのまま UI props にしない
- 画面都合の整形は `pages/widgets` の adapter か `shared/api/dal` で行う

関連：`DATA_ACCESS_LAYER.md`

---

## 3) State の置き場所（目安）

- Server state：TanStack Query（`features` / `entities` の model で）
- UI local state：コンポーネント内（ただし再利用するなら model へ）
- Cross-feature の “なんでもストア”：原則禁止（依存地獄になりやすい）

---

## 4) Server/Client 境界（RSC 時代）

- 可能な限り RSC で表示を完結させ、不要な client 化を避ける
- client 化する理由（操作がある、ブラウザAPIが必要等）を明文化する

---

## 5) レビュー観点（チェックリスト）

- UIが適切な層に置かれている（pages肥大化していない）
- props が過剰でない（DTO最小化できている）
- shared/ui にビジネスルールが入っていない
- features が entities/widgets/pages を直接抱え込んでいない
