# Component Requirements（Index）

このフォルダは、再利用するコンポーネントの要件を管理します。

- テンプレ：`COMPONENT_REQUIREMENTS_TEMPLATE.md`
- 新規作成時はテンプレを複製して埋めます

推奨命名：`C_<id>_<ComponentName>_REQUIREMENTS.md`

---

## コンポーネント一覧

### C_2: ChatMessageList
- **Purpose**: Q&Aスレッド内のメッセージ一覧表示
- **FSD placement**: `features/chat/ui`
- **Key features**:
  - ユーザー/AI メッセージの区別
  - Source（根拠）の表示
  - SSEストリーミング中の表示
- **Document**: [`C_2_ChatMessageList_REQUIREMENTS.md`](./C_2_ChatMessageList_REQUIREMENTS.md)

### C_3: SourceCitation
- **Purpose**: AI回答の根拠（Source）を表示するクリック可能なリンクカード
- **FSD placement**: `features/chat/ui`
- **Key features**:
  - ファイル名、引用箇所、アイコン表示
  - クリック可能
  - アクセシビリティ対応
- **Document**: [`C_3_SourceCitation_REQUIREMENTS.md`](./C_3_SourceCitation_REQUIREMENTS.md)

### C_4: FileTreeExplorer
- **Purpose**: 科目ごとに保存済み資料をツリー形式で表示
- **FSD placement**: `features/fileManagement/ui`
- **Key features**:
  - ファイル操作（クリックで開く、アップロード）
  - 検索/フィルタ機能
  - 解析状態の表示
- **Document**: [`C_4_FileTreeExplorer_REQUIREMENTS.md`](./C_4_FileTreeExplorer_REQUIREMENTS.md)

### C_5: StreamingIndicator
- **Purpose**: SSEストリーミング中のインジケータ表示
- **FSD placement**: `features/chat/ui`
- **Key features**:
  - アニメーション付きインジケータ
  - アクセシビリティ対応
  - サイズバリアント対応
- **Document**: [`C_5_StreamingIndicator_REQUIREMENTS.md`](./C_5_StreamingIndicator_REQUIREMENTS.md)
