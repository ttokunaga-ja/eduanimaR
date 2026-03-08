# Contributing to pageBench

Thank you for your interest in contributing to pageBench!

## 新しいバックエンドの追加方法

`pagebench/backends/` に新しいファイルを作成し、`RAGBackend` を継承してください。

```python
# pagebench/backends/my_backend.py
from .base import QueryResult, RAGBackend, Source
from pathlib import Path

class MyBackend(RAGBackend):
    def create_collection(self, name: str) -> str:
        # コレクション作成ロジック
        ...

    def upload_document(self, collection_id: str, path: Path) -> str:
        # ドキュメントアップロードロジック
        ...

    def wait_for_ready(self, collection_id: str, timeout: int = 300, poll_interval: int = 5) -> bool:
        # インデックス完了待機ロジック
        ...

    def query(self, collection_id: str, question: str) -> QueryResult:
        # クエリ実行ロジック
        ...
```

次に `pagebench/backends/__init__.py` の `get_backend()` にケースを追加し、  
`pagebench/config.py` の `BackendType` に新しい値を追加します。

## 新しいドメインの追加方法

以下の構造でディレクトリを作成してください:

```
domains/
└── my_domain/
    ├── 0a_registry.csv     # 文書一覧
    ├── 0b_qa_pairs.csv     # Q&Aペア
    └── source_pdfs/        # 元PDF
```

QA データは手動作成または `scripts/generate_qa.py` で自動生成できます。

## 開発環境のセットアップ

```bash
python3 -m venv .venv
source .venv/bin/activate
pip install -e ".[dev]"
```

## テストの実行

```bash
pytest tests/ -v
```

## Pull Request の手順

1. このリポジトリをフォーク
2. フィーチャーブランチを作成 (`git checkout -b feature/my-feature`)
3. 変更をコミット
4. テストが通ることを確認 (`make test`)
5. Pull Request を作成
