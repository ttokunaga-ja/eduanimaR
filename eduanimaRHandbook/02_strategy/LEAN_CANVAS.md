# LEAN CANVAS: eduanima+R

## 1. PROBLEM (課題)
*   LMS上の資料が散在し、復習時に必要な情報を探すのが困難。
*   AIチャットを使っても、具体的な講義資料に基づかない一般的な回答しか得られない（ハルシネーション）。
*   学習の優先順位が分からず、試験前にパニックになる。

## 2. SOLUTION (解決策)
*   **Librarian & Professor:** 自律的な資料検索と、根拠に基づいた教育的解説。
*   **Auto-Ingestion（最重要）:** Chrome拡張機能によるMoodle資料の完全自動収集・解析。学生は何もする必要がない。
*   **Learning Context:** 科目IDによる厳格な検索範囲の制限と、ソースURLの提示。

## 3. UNIQUE VALUE PROPOSITION (独自の価値)
**「あなたのLMS資料を、あなた専用の生きた知識ベースに変える司書と教授」**
*   答えを出すだけでなく、資料の「着眼点」を示し、原典への回帰を促す。

## 4. UNFAIR ADVANTAGE (圧倒的優位性)
*   **Vision Reasoning:** 図表や数式を「意味」で理解する高度な資料解析。
*   **LangGraph Agent:** 複数の検索クエリを自律的に試行する高い資料特定率。
*   **Go/Python Hybrid:** 堅牢なデータ管理と高度なAI推論の両立。

## 5. CHANNELS (販路/接点)
*   ブラウザ拡張機能（LMS利用中の介入）。
*   Webアプリケーション（復習用ダッシュボード）。

## 6. CUSTOMER SEGMENTS (顧客セグメント)
*   大学の学部生（特に資料の多い理系・医歯薬・法学系など）。
*   将来的な拡張：科目内グループ（友人・TA間での資料共有）。

## 7. COST STRUCTURE (コスト構造)
*   Gemini API 使用料（2.0 Flash による低コスト検索）。
*   Google Cloud Run / Cloud SQL (PostgreSQL) 運用費。

## 8. REVENUE STREAMS (収益/価値)
*   個人利用：フリーミアムモデル（将来検討）。
*   学習時間の短縮と理解度向上という「時間的・知的価値」の創出。

