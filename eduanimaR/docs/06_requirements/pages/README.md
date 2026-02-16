# Page Requirements（Index）

このフォルダは、各ページ（route）の要件を管理します。

- テンプレ：`PAGE_REQUIREMENTS_TEMPLATE.md`
- 新規作成時はテンプレを複製して埋めます

推奨命名：`P_<id>_<PageName>_REQUIREMENTS.md`

---

## ページ要件マトリクス（Phase 1-4別）

### Phase別の提供機能

eduanimaRは段階的にリリースします。各Phaseでの提供機能を以下に示します：

| Phase | 認証 | チャット（Q&A） | 資料管理 | 共有 | 備考 |
|:---:|:---:|:---:|:---:|:---:|:---|
| **Phase 1** | ❌ 認証スキップ（dev-user固定） | ✅ 基本Q&A、Librarian推論ループ実装・検証 | ✅ 拡張機能で自動アップロード実装 | ❌ | ローカル開発のみ、Web版curlテスト、拡張ローカル読み込み |
| **Phase 2** | ✅ SSO（Google/Meta/MS/LINE） | ✅ SSE配信、エビデンス表示 | ✅ 拡張機能自動アップロード本番適用 | ❌ | Chrome Web Store公開、Web版デプロイ、新規登録は拡張のみ |
| **Phase 3** | ✅ | ✅ 学習計画生成 | ✅ | ❌ | 小テストHTML解析、コンテキスト自動認識 |
| **Phase 4** | ✅ | ✅ | ✅ | ✅ 科目資料セットのみ | 質問履歴・学習ログは非共有 |

**Phase 1-4の一貫した制約**:
- **Web版での新規登録不可**: 拡張機能でのみユーザー登録可能
- **Web版でのファイルアップロード制限**: 拡張機能でのみアップロード可能
- **個人利用のみ**: Phase 1-4では科目内グループ共有は対象外

**参照**: [`../../eduanimaRHandbook/04_product/ROADMAP.md`](../../eduanimaRHandbook/04_product/ROADMAP.md)

---

## カスタマージャーニー対応（現状ペイン → 解決策）

### 現状のペイン（Handbookより）

**忙しい学部生の課題**:
- **探す時間が溶ける**: 「どこに何が書いてあったか」を探す時間が負担
- **重要箇所が分からない**: 資料の着眼点が不明で、理解に時間がかかる
- **手動ファイル管理が面倒**: ファイル収集・整理のオーバーヘッド

**参照**: [`../../eduanimaRHandbook/03_customer/CUSTOMER_JOURNEY.md`](../../eduanimaRHandbook/03_customer/CUSTOMER_JOURNEY.md)、[`../../eduanimaRHandbook/03_customer/PERSONAS.md`](../../eduanimaRHandbook/03_customer/PERSONAS.md)

### eduanimaRによる解決策

| 現状ペイン | eduanimaRの解決策 | 該当ページ |
|:---|:---|:---|
| 探す時間が溶ける | 質問から根拠箇所に1分以内到達（Q&Aページ、エビデンス表示） | `/qa/chat`、`/subjects/[id]/files` |
| 重要箇所が分からない | 資料の着眼点を示し、why_relevantで説明（エビデンスカード） | `/qa/chat` |
| 手動ファイル管理が面倒 | LMS資料の自動収集・アップロード（Chrome拡張機能） | Chrome拡張機能のみ |

**理想の体験（Handbookより）**:
- 質問を投げると、根拠箇所（資料名 + ページ番号）がすぐに表示される
- 「なぜこの箇所が重要か」が明示され、学習者が納得できる
- ファイル収集のオーバーヘッドがゼロ

**参照**: [`../../eduanimaRHandbook/03_customer/CUSTOMER_JOURNEY.md`](../../eduanimaRHandbook/03_customer/CUSTOMER_JOURNEY.md)

---

## ペルソナ要件（Goals/Painsを各ページ要件に紐づけ）

### 主要ペルソナ：忙しい学部生

**Goals（目標）**:
- 「何を勉強すべきか」を1分以内に特定したい → **Q&Aページ**: エビデンス初回表示までの時間を1分以内に
- 資料の該当箇所（ページ番号）をすぐに見つけたい → **エビデンスカード**: クリッカブルURL、ページ番号表示
- 学習計画を自分で立てられるようになりたい → **Phase 3以降**: 学習計画生成機能

**Pains（課題）**:
- 探す時間が負担 → **Q&Aページ**: 質問からエビデンスまでの時間短縮
- 資料が散在 → **資料管理ページ**: 科目別ファイルツリー表示
- 手動ファイル管理が面倒 → **Chrome拡張機能**: 自動アップロード

**参照**: [`../../eduanimaRHandbook/03_customer/PERSONAS.md`](../../eduanimaRHandbook/03_customer/PERSONAS.md)
