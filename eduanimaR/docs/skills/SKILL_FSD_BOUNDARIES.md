# SKILL: FSD Boundaries（Feature-Sliced Design境界強制）

対象：FSD のレイヤー/スライス境界を "人の善意" ではなくツールで強制する。

変化に敏感な領域：
- ESLint の設定形式（flat config等）
- import alias / path 解決

関連：
- `../01_architecture/FSD_LAYERS.md`
- `../01_architecture/FSD_OVERVIEW.md`
- `../05_operations/CI_CD.md`
- `SKILL_ESLINT_BOUNDARIES.md`

---

## Versions（2026-02-15）

- `eslint`: `10.0.0`
- `eslint-plugin-boundaries`: `5.4.0`
- `@typescript-eslint/parser`: `8.55.0`

---

## Must

- 境界違反はCIで落とす（例外を増やさない）
- deep import を禁止し、Public API を守る
- レイヤー間の依存は単方向（`app→pages→widgets→features→entities→shared`）

### FSD 境界規則（契約）

1. **レイヤー境界**
   - 上位レイヤーは下位レイヤーに依存できる
   - 下位レイヤーは上位レイヤーに依存できない（逆流禁止）
   - 同一レイヤー内の横断依存を最小化する

2. **スライス境界**
   - スライスは独立した機能単位
   - 他のスライスへの直接依存を避ける（shared を経由）

3. **Public API**
   - 各スライスは `index.ts` で Public API を公開
   - 外部からは `index.ts` 経由でのみ import
   - deep import（`@/features/chat/ui/ChatInput`）は禁止

### 実装メモ（テンプレの形）

```typescript
// eslint.config.mjs
import boundaries from 'eslint-plugin-boundaries';

export default [
  {
    plugins: { boundaries },
    settings: {
      'boundaries/elements': [
        { type: 'app', pattern: 'src/app/**' },
        { type: 'pages', pattern: 'src/pages/**' },
        { type: 'widgets', pattern: 'src/widgets/**' },
        { type: 'features', pattern: 'src/features/**' },
        { type: 'entities', pattern: 'src/entities/**' },
        { type: 'shared', pattern: 'src/shared/**' },
      ],
    },
    rules: {
      'boundaries/element-types': [
        'error',
        {
          default: 'disallow',
          rules: [
            // app → すべて
            { from: 'app', allow: ['pages', 'widgets', 'features', 'entities', 'shared'] },
            // pages → widgets/features/entities/shared
            { from: 'pages', allow: ['widgets', 'features', 'entities', 'shared'] },
            // widgets → features/entities/shared
            { from: 'widgets', allow: ['features', 'entities', 'shared'] },
            // features → entities/shared
            { from: 'features', allow: ['entities', 'shared'] },
            // entities → shared
            { from: 'entities', allow: ['shared'] },
            // shared → なし（最下層）
            { from: 'shared', allow: [] },
          ],
        },
      ],
      'boundaries/no-private': 'error', // deep import 禁止
    },
  },
];
```

---

## 禁止

- "一時しのぎ" の例外ルール追加
- テストだけ境界を緩める（抜け道になる）
- deep import（`@/features/chat/ui/ChatInput`）を使う
- 同一レイヤー内の横断依存を無秩序に増やす

---

## チェックリスト

- [ ] import 方向は単方向（`app→...→shared`）か？
- [ ] 同一レイヤー横断（`features→features`）が増えていないか？
- [ ] すべての import が Public API（`index.ts`）経由か？
- [ ] 境界違反を設定で無効化していないか？
- [ ] CI で ESLint の境界チェックが実行されているか？

---

## 実装例

### 正しい import（Public API 経由）

```typescript
// ✅ Good: Public API 経由
import { ChatInput } from '@/features/chat';
import { Subject } from '@/entities/subject';
import { Button } from '@/shared/ui';
```

### 間違った import（deep import）

```typescript
// ❌ Bad: deep import
import { ChatInput } from '@/features/chat/ui/ChatInput';
import { SubjectCard } from '@/entities/subject/ui/SubjectCard';
```

### 境界違反の例

```typescript
// ❌ Bad: shared → features（逆流）
// src/shared/lib/utils.ts
import { useChatHistory } from '@/features/chat';

// ❌ Bad: entities → features（逆流）
// src/entities/subject/lib/utils.ts
import { useSubjectSelector } from '@/features/subjectSelector';
```

---

## トラブルシューティング

### 境界違反が検出された場合

1. **依存方向を確認**
   - 逆流していないか？（下位→上位）
   - 同一レイヤー横断が適切か？

2. **構造を見直す**
   - 共通ロジックは `shared` に移動
   - スライス間の依存は最小化

3. **例外を追加しない**
   - 境界違反を設定で無効化するのではなく、構造を修正する

---

## CI での実行

```yaml
# .github/workflows/ci.yml
- name: Lint with ESLint
  run: npm run lint

- name: Check FSD boundaries
  run: npm run lint -- --rule 'boundaries/element-types: error'
```

---

## 参考

- Feature-Sliced Design: https://feature-sliced.design/
- eslint-plugin-boundaries: https://github.com/javierbrea/eslint-plugin-boundaries
