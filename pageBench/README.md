# pageBench

**pageBench** は、PDF ドキュメントを RAG システムに投入し、Gemini LLM-as-Judge で性能を自動評価する OSS ツールです。

## 特徴

- 🚀 **単一バイナリ** — Go 製。`brew install` または `go install` で即利用可能
- 🔌 **OpenAI 互換 API に対応** — OpenAI / Ollama / LM Studio / Azure OpenAI / Dify など
- 📄 **2 モード** — `chat_completions`（RAG 組み込み済みエンドポイント）と `assistants`（Vector Store + File Search）
- 🤖 **LLM-as-Judge** — Gemini で正確性・忠実性・完全性の 3 軸を自動採点
- 💡 **Thinking 対応** — Gemini 3 の ThinkingBudget で評価精度を段階調整
- 📊 **自動レポート** — ROUGE-L / Exact Match + Markdown レポート生成
- ♻️ **再開可能** — チェックポイントで評価を中断・再開

## クイックスタート

```bash
# 1. バイナリをインストール
go install github.com/ttokunaga-ja/pagebench/cmd/pagebench@latest

# または clone してビルド
git clone https://github.com/ttokunaga-ja/pagebench
cd pagebench
make build   # → ./bin/pagebench

# 2. 設定ファイルを準備
cp .env.example .env
# .env を編集: OPENAI_COMPAT_API_KEY と GEMINI_API_KEY を設定

# 3. ドメインディレクトリを準備
mkdir -p domains/my_domain/docs
cp your_documents/*.pdf domains/my_domain/docs/

# corpus.csv を作成 (ファイル一覧)
echo "file_name,title" > domains/my_domain/corpus.csv
echo "document1.pdf,Document 1" >> domains/my_domain/corpus.csv

# 4. QA テストセットを自動生成 (Gemini File API 使用)
pagebench generate --domain domains/my_domain

# 5. 評価実行
pagebench eval --domain domains/my_domain

# 6. レポートを再生成 (オプション)
pagebench report --domain domains/my_domain
```

## ディレクトリ構造

```
domains/
└── my_domain/
    ├── corpus.csv          # ドキュメント一覧 (file_name, title, description)
    ├── testset.csv         # QA テストセット (generate コマンドで自動生成)
    ├── results.csv         # 評価結果 (eval コマンドで生成)
    ├── report.md           # Markdown レポート (eval/report コマンドで生成)
    ├── .checkpoint.json    # 中断再開用チェックポイント (自動管理)
    └── docs/               # PDF ドキュメント置き場
        ├── document1.pdf
        └── document2.pdf
```

## コマンド一覧

| コマンド | 説明 |
|---------|------|
| `pagebench generate` | Gemini File API で QA テストセット生成 |
| `pagebench eval` | RAG システムの評価を実行 |
| `pagebench report` | results.csv からレポートを再生成 |
| `pagebench upload` | ドキュメントのアップロードのみ実行 (assistants モード) |
| `pagebench check` | 設定と接続を確認 |

### `generate` オプション

```
pagebench generate --domain <path> [flags]

Flags:
  -d, --domain string     ドメインディレクトリパス (必須)
      --qa-per-doc int    1 ドキュメントあたりの QA 生成件数 (default 10)
      --thinking string   thinking レベル: shallow|medium|deep
      --model string      Gemini モデル名
```

### `eval` オプション

```
pagebench eval --domain <path> [flags]

Flags:
  -d, --domain string   ドメインディレクトリパス (必須)
      --limit int       評価する最大 QA 件数 (0 = 全件)
      --skip-judge      LLM-as-Judge をスキップ
      --resume          チェックポイントから再開
      --no-cleanup      評価後にコレクションを削除しない
```

## 設定 (.env)

```dotenv
# OpenAI 互換バックエンド
OPENAI_COMPAT_MODE=chat_completions   # または assistants
OPENAI_COMPAT_API_BASE=https://api.openai.com/v1
OPENAI_COMPAT_API_KEY=sk-...
OPENAI_COMPAT_MODEL=gpt-5-mini        # gpt-5-nano (最安) / gpt-5 / o3-mini も可

# Gemini (Judge + Generate)
GEMINI_API_KEY=AIza...
GEMINI_JUDGE_MODEL=gemini-3-flash-preview   # gemini-3.1-pro (高精度) / gemini-3-flash-preview-lite (最安) も可
GEMINI_GENERATE_MODEL=gemini-3-flash-preview
GEMINI_THINKING_LEVEL=shallow         # shallow | medium | deep
```

詳細は [.env.example](.env.example) を参照してください。

## Thinking レベル

Gemini 3 の拡張思考機能で Judge/Generate の精度を調整できます。

| レベル | ThinkingBudget | 特徴 |
|--------|---------------|------|
| `shallow` | 0 (無効) | 高速・低コスト（デフォルト） |
| `medium` | 5,000 | バランス型 |
| `deep` | 24,576 | 最高精度（コスト高） |

```bash
# deep thinking で QA 生成 + 評価
pagebench generate --domain domains/my_domain --thinking deep
pagebench eval --domain domains/my_domain
```

## 対応バックエンド

### chat_completions モード（デフォルト）

RAG が組み込まれた OpenAI 互換エンドポイントへ直接クエリを送ります。ドキュメントのアップロードは外部で行います。

```dotenv
OPENAI_COMPAT_MODE=chat_completions
OPENAI_COMPAT_API_BASE=http://localhost:11434/v1  # Ollama
OPENAI_COMPAT_MODEL=llama3.2
OPENAI_COMPAT_API_KEY=dummy
```

### assistants モード

OpenAI Assistants API v2 (Vector Store + File Search) を使用します。`pagebench upload` でドキュメントを自動アップロードします。

```dotenv
OPENAI_COMPAT_MODE=assistants
OPENAI_COMPAT_API_BASE=https://api.openai.com/v1
OPENAI_COMPAT_API_KEY=sk-...
OPENAI_COMPAT_MODEL=gpt-5             # gpt-5-mini (低コスト) も可
```

## 評価指標

| 指標 | 説明 |
|------|------|
| **ROUGE-L** | 最長共通部分列ベースの F1 スコア（日英対応） |
| **Exact Match** | 完全一致率（正規化後） |
| **Judge Accuracy** | Gemini による正確性スコア (1-5) |
| **Judge Faithfulness** | Gemini による忠実性スコア (1-5) |
| **Judge Completeness** | Gemini による完全性スコア (1-5) |
| **Judge Overall** | 上記の総合評価 (1-5) |
| **Latency** | クエリの応答時間 (ms) |

## 開発

```bash
# テスト実行
make test

# ビルド
make build

# すべて (tidy → lint → test → build)
make all
```

## ライセンス

MIT License — 詳細は [LICENSE](LICENSE) を参照してください。
