# SKILL: Next.js（App Router / RSC / BFF）

対象：Next.js App Router を BFF として運用する。

変化に敏感な領域：
- RSC/Client 境界（`use client`）
- ルートの静的化/動的化（Dynamic API）
- fetch キャッシュ/再検証

関連：
- `../02_tech_stack/STACK.md`
- `../02_tech_stack/SSR_HYDRATION.md`
- `../01_architecture/DATA_ACCESS_LAYER.md`
- `../01_architecture/CACHING_STRATEGY.md`

---

## Versions（2026-02-11 / dist-tag: latest）

- `next`: `16.1.6`
- `react`: `19.2.4`
- `react-dom`: `19.2.4`

（確認：`npm view next version` など）

---

## Must
- `src/app/**/page.tsx` は薄い adapter（基本 `src/pages/**` を描画）
- RSC で取得するデータは DAL 経由（直叩き禁止）
- Dynamic API の使用箇所を把握し、意図せぬ動的化を避ける

### 実装パターン（テンプレの形）

- App Router の page は pages レイヤーを描画するだけ

例：`src/app/(routes)/page.tsx`

```tsx
import { HomePage } from '@/pages/home';

export default function Page() {
	return <HomePage />;
}
```

## 禁止
- RSC が Route Handler を呼ぶ（サーバ内で無駄なHTTP hop）
- なんとなく `use client` を付ける（hydrationコスト増）
- 画面内で手書き `fetch`（生成API/DALに寄せる）

## チェックリスト
- [ ] この画面は RSC で完結できるか？（操作が無いなら client 化しない）
- [ ] 認証/ユーザー依存データで意図せず静的化していないか？
- [ ] 取得は DAL 経由か？DTO 最小化できているか？
