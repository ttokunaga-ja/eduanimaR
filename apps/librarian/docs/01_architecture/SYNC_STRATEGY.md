# SYNC_STRATEGY（Librarian 観点）

## 結論
DB↔検索インデックスの同期（CDC/Outbox/再構築など）は **Librarian の責務ではない**。
本サービスは検索の物理実行を行わず、Professor の検索ツールを呼び出すだけである。

## 目的
Librarian が「同期の詳細」に踏み込まず、**同期遅延や検索結果の揺れを前提にした停止判断**を行えるようにする。

## 前提
- Professor は、DB/インデックス/バッチ処理/再インデックスを管理する。
- Librarian は、Professor から返る検索候補（`temp_index` 付きテキスト断片）だけを材料に判断する。
- 取得件数（k）や除外ID（既出断片）など「物理検索の状態」は Professor が保持し、Librarian は指定しない。

## Librarian 側の設計指針
- 停止条件は「検索基盤の完全性」ではなく、**タスクの充足性（target_items が引用付きで満たされたか）**で定義する。
- 検索結果が薄い/揺れる場合は、以下の順で改善を試みる:
  1. クエリ多様化（言い換え/キーワード抽出/制約追加）
  2. 範囲拡大（関連概念/上位概念）
  3. それでも不足なら、MaxRetry 到達時点の最善を返す

## 禁止事項
- Librarian が同期の仕組み（CDC/Outbox/Indexer 等）を直接操作すること
- Librarian が検索結果の「正」を上書きすること

## 関連
- `01_architecture/EDUANIMA_LIBRARIAN_SERVICE_SPEC.md`
- `03_integration/INTER_SERVICE_COMM.md`
