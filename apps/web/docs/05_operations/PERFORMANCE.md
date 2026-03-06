# Performance（本番前チェックと日常運用）

このドキュメントは、性能劣化を “後から気づく” のを防ぎ、
SSR/Hydration + RSC 時代の最小ベストプラクティスを固定します。

---

## 結論（Must）

- "use client" 境界を下げ、Hydration コストを最小化する
- 画像/フォント/3rd-party script は Next の標準最適化を優先する
- バンドルサイズと Web Vitals を継続計測する

---

## 本番前チェック（推奨）

- Lighthouse を実行（シミュレーション）
- Field data（可能なら）と合わせて判断
- バンドル分析（依存追加の影響確認）

---

## 実装上の注意

- Client Components で “初回表示に必須データ” を取りに行って白画面を作らない
- 可能なら RSC で表示を完結し、操作が必要な部分だけ Client 化

関連：
- SSR/Hydration の前提： [../02_tech_stack/SSR_HYDRATION.md](../02_tech_stack/SSR_HYDRATION.md)
- キャッシュ戦略： [../01_architecture/CACHING_STRATEGY.md](../01_architecture/CACHING_STRATEGY.md)
