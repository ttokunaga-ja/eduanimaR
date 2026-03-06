# TEST_STRATEGY

## 目的
テストを「速い・壊れにくい・意図が明確」な形で整備し、継続的に回せる品質基盤を作る。

本リポジトリは Professor（Go）側が対象であり、他コンポーネント（例: フロントエンド / Librarian 推論サービス）は **通信（契約）と役割境界** の観点でのみ扱う。

## 方針
- 原則: Unit を厚く、Integration を要所、E2Eは必要最小
- 詳細は `TEST_PYRAMID.md` を正とする

## スコープ
### このリポジトリで担保すること（MUST）
- usecase の仕様（境界条件/権限/状態遷移/物理制約）
- DB（sqlc + マイグレーション）の実動作
- 外向きHTTPの契約/エラー形式/認可の整合
- SSE のストリーミング挙動（イベント形、切断・キャンセル、最終結果の整合）
- Librarian（gRPC）呼び出しのリトライ/タイムアウト/エラーマッピング
- Kafka worker の冪等性/再処理可能性（少なくとも at-least-once 前提）

### このリポジトリで“作り込まない”こと（SHOULD）
- フロントエンドのUI/E2E（consumer側で実施）
- Librarian の推論品質のE2E（Professor側は入出力契約とタイムアウトを担保）

## 重点領域
- usecase の仕様（境界条件/権限/状態遷移、user_id/subject_id/is_active などの物理制約を破らない）
- repository のSQL（sqlc生成物の実動作、pgvector/検索クエリの全件スキャン回避）
- handler のエラーマッピング（共通形式、`ERROR_CODES.md` との整合）
- SSE（途中経過/最終回答/引用の整合、キャンセル伝播、接続中断時の後始末）
- gRPC（deadline必須、status→HTTP変換、依存障害時の挙動）
- Kafka worker（重複イベント、DLQ、再実行で壊れない）

## テストデータ
- 可能な限り「固定 seed のテストデータ」を使う
- 大きいPDF等のバイナリをテストに混ぜない（必要なら最小サンプルを用意し、実行時間を管理する）

## 関連
- `03_integration/CONTRACT_TESTING.md`
- `01_architecture/RESILIENCY.md`

