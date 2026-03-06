# ADR 002: サービスディレクトリを apps/ に統一する

- **日付**: 2026-03-06
- **ステータス**: 採択済み

---

## コンテキスト

リファクタリング前のルート構造:

```
eduanimaR/           # TypeScript フロントエンド
eduanimaR_Professor/ # Go バックエンド
eduanimaR_Librarian/ # Python 推論サービス
eduanimaRHandbook/   # 製品ドキュメント
```

命名に統一性がなく（アンダースコア有無、`_Librarian` 等のサフィックス）、役割（アプリ/ドキュメント）の境界も曖昧だった。

---

## 決定

**アプリケーションサービスを `apps/<name>/` に統一する。製品ドキュメントを `handbook/` に移動する。**

| 旧パス | 新パス |
|---|---|
| `eduanimaR/` | `apps/web/` |
| `eduanimaR_Professor/` | `apps/professor/` |
| `eduanimaR_Librarian/` | `apps/librarian/` |
| `eduanimaRHandbook/` | `handbook/` |

`git mv` で移動したため git 履歴は保持される。

---

## 理由

1. `apps/` プレフィックスにより「これはデプロイ可能なサービス」と一目で判断できる。
2. 命名規則の統一でオートコンプリートやタブ補完が効率化する。
3. `handbook/` の分離により「コード資産」と「ビジネスドキュメント」の境界が明確になる。

---

## 影響

- ルート `Makefile` の全 `cd <service>` パスを更新した。
- `ops/compose/docker-compose.yml` / `ops/compose/docker-compose.prod.yml` の `context:` と `volumes:` を更新した。
- `ops/cloudrun/cloudbuild.yaml` の `context:` を更新した。
- `.github/workflows/infra-smoke.yml` のパスフィルタとマイグレーションパスを更新した。
- フロントエンド内の CI workflow（`apps/web/.github/workflows/`）はルート `.github/workflows/` に昇格させた（ADR 003 参照）。
