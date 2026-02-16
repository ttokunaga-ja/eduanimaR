# API_CONTRACT_WORKFLOW

## 目的
Professor（Go）↔ Librarian（Python）の API を **gRPC/Proto を正（SSOT）**として管理し、契約逸脱を防ぐ。

## 前提（Librarian）
本サービスは以下の構成を前提とする:

```
eduanima-professor (Go)  ↔  eduanima-librarian (Python)
         [gRPC/Proto]           [gRPC Server]
         [双方向ストリーミング]
         [契約SSOT: eduanimaR_Professor/proto/librarian/v1/librarian.proto]
```

## 原則（MUST）
- 変更は必ず Proto 定義から始める（Contract First）
- 生成物は手編集しない（再生成で消える）
- Librarian の gRPC サーバーは Proto と整合するように実装する
- 契約の正は Professor 側の `proto/librarian/v1/librarian.proto`

## フロー（推奨）
### 1) 契約定義（SSOT）
1. Proto（`eduanimaR_Professor/proto/librarian/v1/librarian.proto`）を更新する（破壊的変更かどうかを明記）
2. `ERROR_HANDLING.md` の共通形式に沿ってエラーも定義する

### 2) コード生成
1. Professor（Go）側: `buf generate` で Go のサーバー/クライアントコードを生成
2. Librarian（Python）側: `buf generate` または grpcio-tools で Python のサーバー/クライアントコードを生成

### 3) Librarian（Python）側
1. 生成された gRPC サーバースタブを実装する
2. リクエスト/レスポンスメッセージは生成物を使用する

### 4) Professor（Go）側
1. 生成された gRPC クライアントを使用して Librarian を呼び出す
2. Professor から Librarian を呼び出す呼び出し点をユースケースとして統一する

## レビュー観点
- 互換性: 既存クライアントに影響する変更か（必須/任意、型変更、フィールド削除等）
- Proto のベストプラクティス: フィールド番号の再利用禁止、後方互換性の維持
- エラーハンドリング: gRPC ステータスコードの適切な使用
- ストリーミング: 双方向ストリーミングの適切な実装（キャンセル伝播、バックプレッシャ）

## APIライフサイクル（推奨）
- 廃止（deprecation）の方針（期間、告知、削除手順）を決める
- 破壊的変更は原則避け、必要なら新しい RPC メソッドを追加する
- Proto のバージョニング: パッケージ名に `v1`, `v2` を含める

> セキュリティ観点の詳細は `05_operations/API_SECURITY.md` を参照。
