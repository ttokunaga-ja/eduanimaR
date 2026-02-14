# TEST_STRATEGY

## 目的
テストを「速い・壊れにくい・意図が明確」な形で整備し、継続的に回せる品質基盤を作る。

## 方針
- 原則: Unit を厚く、Integration を要所、E2Eは必要最小
- 詳細は `TEST_PYRAMID.md` を正とする

## 重点領域
- usecase / graph の仕様（境界条件/停止判断/状態遷移）
- outbound client（Professor / Gemini）のリトライ・タイムアウト・エラーマッピング
- handler の入出力・エラーマッピング（共通形式）

## 本サービスで扱わないもの
- DB リポジトリの実検証（Librarian は DB-less）
- CDC/Indexer/イベント基盤を伴う E2E（Professor 側の責務）

