# ELASTICSEARCH_OPS

## 目的
Elasticsearch 9.2.4 のインデックス設計・運用・検索クエリの規約を定義する。

## 基本方針
- DBは正、ESは検索/集計の投影（projection）
- Mapping はコードと同様にレビュー対象（差分管理）

## Mapping管理
- `configs/es/` 配下に versioned な mapping/settings JSON を置く（例）
- 変更時は reindex 戦略（alias 切り替え）を必ず書く

## ベクトル検索
- `dense_vector` を利用する場合:
  - 次元数、類似度（cosine/dot等）を要件として固定
  - 生成モデル変更時の再計算/再投入手順を用意

## 運用
- index alias を使い、ダウンタイム無しで切り替える
- 大規模再投入はオフピークで実施し、進捗/失敗再開を前提に設計する
