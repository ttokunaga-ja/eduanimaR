# Page Requirements（Index）

このフォルダは、各ページ（route）の要件を管理します。

- テンプレ：`PAGE_REQUIREMENTS_TEMPLATE.md`
- 新規作成時はテンプレを複製して埋めます

推奨命名：`P_<id>_<PageName>_REQUIREMENTS.md`

---

## ページ要件マトリクス（Phase 1-5別）

### Phase別の提供機能

eduanimaRは段階的にリリースします。各Phaseでの提供機能を以下に示します：

| Phase | 認証 | チャット（Q&A） | 資料管理 | 拡張機能 | Web版固有 | 備考 |
|:---:|:---:|:---:|:---:|:---:|:---:|:---|
| **Phase 1** | ❌ dev-user固定 | ✅ 基本Q&A、Librarian推論ループ、プログレスバー、フィードバックボタン | ✅ 拡張機能で自動アップロード実装 | ✅ 実装（ローカル読み込み） | ✅ 資料一覧・会話履歴・科目選択UI | ローカル開発のみ、Web版curlテスト |
| **Phase 2** | ✅ SSO（Google/Meta/MS/LINE） | ✅ SSE配信、エビデンス表示、お問い合わせフォーム | ✅ 拡張機能自動アップロード本番適用 | ✅ ZIP配布、Web版リンク | ✅ 維持 | 本番デプロイ、新規登録は拡張のみ |
| **Phase 3** | ✅ | ✅ | ✅ | ✅ Chrome Web Store公開 | ✅ 維持 | ストア審査対応 |
| **Phase 4** | ✅ | ✅ 画面解説機能 | ✅ | ✅ HTML・画像取得、解説生成 | ✅ 維持 | 小テスト支援 |
| **Phase 5** | ✅ | ✅ 学習計画生成 | ✅ | ✅ | ✅ 維持 | 構想段階 |

### Phase 1-5の一貫した制約

**全Phaseで禁止**:
- **Web版での新規登録**: 拡張機能でのみユーザー登録可能
- **Web版でのファイルアップロード**: 拡張機能でのみアップロード可能

**Phase 1のみの制約**:
- 開発環境のみ（本番デプロイなし）
- 固定dev-user認証（SSO未実装）

**Phase 2以降の前提**:
- SSO認証必須（Google/Meta/Microsoft/LINE）
- Web版未登録ユーザーへの拡張機能ダウンロード誘導UI必須
- 本番環境デプロイ

### Phase別の共通機能詳細

**すべてのPhaseで提供される共通機能**:

1. **Q&A（チャット）**
   - 質問の入力欄
   - AI Agentの進捗状況を**プログレスバー**で表示
   - 回答の表示
   - 根拠として使用した資料を確認可能（クリッカブルリンク）
   - ヒアリング判断となった場合に選択肢が提示される
   - 回答の最後に**Good/Badのフィードバックボタン**
   - **お問い合わせフォーム（Googleフォーム）へのリンク**

2. **認証**
   - Phase 1: dev-user固定
   - Phase 2以降: SSO（Google/Meta/Microsoft/LINE）

### Web版固有機能（全Phase）

1. **資料一覧閲覧**（Phase 1）
   - 大画面を活かした表示
   - トップメニューバー中央のプルダウンで科目を選択
   - 選択科目の資料一覧を表示
   - プルダウンで「全て」を選択すると科目ごとに一覧表示

2. **会話履歴確認**（Phase 1）
   - トップメニューバー中央のプルダウンで科目を選択
   - 選択科目の会話履歴を表示
   - プルダウンで「全て」を選択すると科目ごとに一覧表示

3. **科目選択UI**（Phase 1）
   - トップメニューバー中央にプルダウン配置
   - どの科目に関して質問・履歴・ファイルを確認しているかを選択可能
   - 選択肢: 「科目A」「科目B」...「全て」

### 拡張機能固有機能

1. **SSO経由のユーザー登録**（Phase 2）
   - Google/Meta/Microsoft/LINE による新規ユーザー登録
   - **Web版からの新規登録は禁止**

2. **Moodle資料の自動検知・自動アップロード**（Phase 1実装 → Phase 2本番適用）
   - LMS上で資料を自動検知
   - Professor APIへ自動送信

3. **Web版へのリンク**（Phase 2）
   - 拡張機能からWeb版へ遷移可能

4. **コース判別による検索精度向上**（Phase 1）
   - QA時に閲覧中のMoodleコースを判別
   - 資料の検索に物理制限をかける（subject_id絞り込み）

5. **閲覧中画面の解説機能**（Phase 4）
   - 現在閲覧中の画面のHTML取得
   - 画面内に表示されている画像ファイル取得（図・グラフ等）
   - 取得したHTML・画像をProfessor APIへ送信
   - LLMによる解説生成（資料を根拠に表示）

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
