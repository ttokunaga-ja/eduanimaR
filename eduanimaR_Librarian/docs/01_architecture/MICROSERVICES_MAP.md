# MICROSERVICES_MAP

## 目的
本ドキュメントは、サービス境界（責務）・依存関係・公開IF（主にHTTP）・ポート/運用単位を一覧化し、機能追加時に「どのサービスを触るべきか」を最短で判断できるようにする。

## 原則
- 1サービス = 1ビジネス能力（Business Capability）
- DBはサービス単位で分離（スキーマ共有やクロスDB参照を避ける）
- 依存関係は可能な限り一方向（循環依存は禁止）

## サービス一覧（テンプレート）
| Service | Responsibility | Owning Data | Exposed APIs (gRPC) | Depends On | Port (dev) |
| --- | --- | --- | --- | --- | --- |
| **api-gateway** | 認証/認可/ルーティング/**gRPC→HTTP変換** | (none) | HTTP/JSON (OpenAPI) | user, product, order | 8080 |
| user | 認証/プロフィール/権限 | users, sessions | `UserService` (gRPC) | (none) | 9001 |
| product | 商品カタログ | products, categories | `ProductService` (gRPC) | user(認可) | 9002 |
| order | 注文/決済連携 | orders, payments | `OrderService` (gRPC) | user, product | 9003 |

> 注: 上表は例。実プロジェクトの実態に合わせて必ず更新すること。
> ポート番号: Gateway=8xxx, 内部gRPCサービス=9xxx(例)

## 依存関係図（記述ルール）
- 図は「通信方向（呼び出し元 → 呼び出し先）」で表す
- 依存は最小限にし、集約が必要な場合はBFF/API Gateway側で行う

## 変更時のチェックリスト
- 新規エンドポイント追加は、責務に最も近いサービスへ配置したか
- 既存サービスに責務を混ぜていないか（境界の劣化）
- 新規データが発生するなら、どのサービスが所有すべきか決めたか
