# pageBench Evaluation Report

| Item | Value |
|------|-------|
| **Domain** | `04_test` |
| **Backend** | `agent @ http://localhost:8080/api/v1/subjects/487c66fb-6515-434c-9487-cf8251946205` |
| **Date** | 2026-03-10 10:25:22 |
| **Total QA** | 17 |
| **Answered** | 12 |
| **Unanswerable** | 5 |

## 📊 Score Summary

| Metric | Score |
|--------|-------|
| **File Hit Rate** | 0.00% |
| **Page Hit Rate** | 0.00% |
| **Ref File Found Rate** | 0.00% |
| **Ref Page Found Rate** | 0.00% |
| **Ref File+Page Found Rate** | 0.00% |
| **ROUGE-L** (avg) | 0.1688 |
| **Exact Match** (avg) | 0.0000 |
| **Judge Overall** (avg) | 1.00 / 5 |
| **Latency** (avg) | 5212 ms |

## 📈 ROUGE-L Distribution

| Range | Count | Bar |
|-------|-------|-----|
| 0.8–1.0 | 0 |  |
| 0.6–0.8 | 0 |  |
| 0.4–0.6 | 0 |  |
| 0.2–0.4 | 1 | █ |
| 0.0–0.2 | 11 | ██████████████████ |

## ⚖️ Judge Overall Distribution

| Score | Count | Bar |
|-------|-------|-----|
| 1 / 5 | 12 | ████████████████████ |
| 2 / 5 | 0 |  |
| 3 / 5 | 0 |  |
| 4 / 5 | 0 |  |
| 5 / 5 | 0 |  |

## 🧪 Hallucination Check (Unanswerable Questions)

> 回答不能問題 (5 件) に対する RAG の幻覚抑制チェック

| Metric | Value |
|--------|-------|
| **Unanswerable Count** | 5 |
| **Hallucination Refusal Rate** | 60.00% |
| **Avg Hallucination Score** (1=OK, 5=worst) | 1.00 / 5 |

### Hallucination Score Distribution

| Score | Meaning | Count | Bar |
|-------|---------|-------|-----|
| 1 | 正確に拒否 | 5 | ████████████████████ |
| 2 | ほぼ拒否 | 0 |  |
| 3 | 部分的幻覚 | 0 |  |
| 4 | 重度の幻覚 | 0 |  |
| 5 | 完全な幻覚 | 0 |  |

### ⚠️ Hallucination Cases (RAG failed to refuse)

**1. [Q0006]**
- Question: 競合他社である大成建設と比較して、インフロニア・ホールディングスのPBR（株価純資産倍率）の推移はどうなっていますか？
- rag_answer: ご提供いただいた資料の中には、インフロニア・ホールディングスおよび大成建設のPBR（株価純資産倍率）の推移に関する情報は含まれておりません。

そのため、ご質問にお答えすることができません。資料をご提供いただければ、その内容に基づいて改めて...
- judge_hallucination: 1
- reasoning: システムは提供された資料に情報がないことを正しく認識し、架空の情報を生成することなく適切に回答を拒否しています。

**2. [Q0016]**
- Question: Reddit以外のプラットフォーム（例：TwitterやFacebook）におけるコミュニティ・アイデンティティの影響について、この文書ではどのような数値データ...
- rag_answer: 提供された教材（Course Materials）には内容が含まれていないため、Reddit以外のプラットフォーム（TwitterやFacebookなど）におけるコミュニティ・アイデンティティの影響に関する数値データを確認することができませ...
- judge_hallucination: 1
- reasoning: システムは文書内に情報がないことを正しく認識し、架空の数値を捏造することなく適切に回答を拒否しています。

## 🏆 Top 3 by ROUGE-L

**1. ROUGE-L=0.3140**
- Question: 想定事例1において、AI開発・提供事業者Aが責任を負わないための重要な条件として何が挙げられていますか。
- ref_answer: AIの限界等について適切に警告を行っていることが、原則として責任を負わないための重要な条件として挙げられています。
- rag_answer: 提供された教材に「想定事例1」に関する具体的な内容が含まれていないため、ご質問に回答することができません。

教材の内容を共有していただければ、それに基づき、AI開発・提供事業者Aが責任を負わないための条件について解説いたします。

**2. ROUGE-L=0.1913**
- Question: 株式給付信託（J-ESOP）の追加費用が発生する仕組みについて説明してください。
- ref_answer: 配布株式数が当期利益に連動し、一般管理費として費用計上されるため、当期利益の向上に合わせて追加費用が発生します。
- rag_answer: 提供された資料に「株式給付信託（J-ESOP）」に関する内容が含まれていないため、ご質問にお答えすることができません。

**3. ROUGE-L=0.1881**
- Question: Redditのデータセットを用いた分析において、最終的にいくつのコミュニティとコミュニティ・月（community-months）が対象となりましたか？
- ref_answer: 283のコミュニティと、合計4,872のコミュニティ・月が対象となりました。
- rag_answer: 提供されたコース資料（Course Materials）のセクションに具体的な内容が含まれていないため、ご質問にあるRedditのデータセット分析に関するコミュニティ数およびコミュニティ・月（community-months）の数値を確認す...

## 🔍 Bottom 3 by ROUGE-L (for improvement)

**1. ROUGE-L=0.0870**
- Question: 2026年3月期第3四半期（FY25 3Q）実績において、インフロニア・ホールディングスがHD設立以来で過去最高を達成した項目は何ですか？
- ref_answer: 売上高、事業利益、EBITDA、当期利益の4項目において過去最高を達成しました。
- rag_answer: ご提示いただいた「Course Materials（コース資料）」のセクションが空欄であるため、提供された資料のみに基づいて回答することができません。

また、ご質問にある「2026年3月期第3四半期（FY25 3Q）」は、現在のカレンダー...

**2. ROUGE-L=0.1053**
- Question: 2026年2月9日時点の株価最高値はいくらですか？
- ref_answer: 2,654.5円です。
- rag_answer: ご提供いただいた教材には、2026年2月9日時点の株価に関する情報は含まれていません。そのため、質問にお答えすることができません。

**3. ROUGE-L=0.1279**
- Question: 本研究で提案されているコミュニティのタイポロジーを構成する2つの主要な側面は何ですか？
- ref_answer: 「どれほど独特か（distinctive）」と「どれほど時間的に動的か（temporally dynamic）」という2つの側面です。
- rag_answer: 提供されたコース資料の内容が空であるため、本研究で提案されているコミュニティのタイポロジーを構成する2つの主要な側面を特定することができません。

回答を構成するために必要な資料をご提供いただけますでしょうか。

## 📁 Ref File Miss Cases

参照ファイルを取得できなかったケースを表示します。

**1. [Q0001]**
- Question: 2026年3月期第3四半期（FY25 3Q）実績において、インフロニア・ホールディングスがHD設立以来で過去最高を達成した項目は何ですか？
- ref_file/ref_page: 02_01_インフロニアＨＤ_2026年3月期第３四半期アナリスト向け決算説明会資料　プレゼンテーション資料 .pdf / 3
- retrieved_file_pages: []

**2. [Q0002]**
- Question: FY25通期計画における1株当たり配当金（年間）は、2月10日の修正によりいくらになりましたか？
- ref_file/ref_page: 02_01_インフロニアＨＤ_2026年3月期第３四半期アナリスト向け決算説明会資料　プレゼンテーション資料 .pdf / 3
- retrieved_file_pages: []

**3. [Q0003]**
- Question: インフラ運営セグメントにおいて、前田建設グループの3Q実績が増収増益となった主な要因は何ですか？
- ref_file/ref_page: 02_01_インフロニアＨＤ_2026年3月期第３四半期アナリスト向け決算説明会資料　プレゼンテーション資料 .pdf / 5
- retrieved_file_pages: []

**4. [Q0004]**
- Question: 株式給付信託（J-ESOP）の追加費用が発生する仕組みについて説明してください。
- ref_file/ref_page: 02_01_インフロニアＨＤ_2026年3月期第３四半期アナリスト向け決算説明会資料　プレゼンテーション資料 .pdf / 8
- retrieved_file_pages: []

**5. [Q0005]**
- Question: 2026年2月9日時点の株価最高値はいくらですか？
- ref_file/ref_page: 02_01_インフロニアＨＤ_2026年3月期第３四半期アナリスト向け決算説明会資料　プレゼンテーション資料 .pdf / 9
- retrieved_file_pages: []

**6. [Q0008]**
- Question: AIサービス提供者が負うべき説明義務の範囲を規定する3つの要素は何ですか。
- ref_file/ref_page: 03_01_議事要旨（PDF形式：375KB）.pdf / 5
- retrieved_file_pages: []

**7. [Q0009]**
- Question: 想定事例1において、AI開発・提供事業者Aが責任を負わないための重要な条件として何が挙げられていますか。
- ref_file/ref_page: 03_01_議事要旨（PDF形式：375KB）.pdf / 3
- retrieved_file_pages: []

**8. [Q0011]**
- Question: 本研究で提案されているコミュニティのタイポロジーを構成する2つの主要な側面は何ですか？
- ref_file/ref_page: 1705.09665.pdf / 1
- retrieved_file_pages: []

**9. [Q0012]**
- Question: Redditのデータセットを用いた分析において、最終的にいくつのコミュニティとコミュニティ・月（community-months）が対象となりましたか？
- ref_file/ref_page: 1705.09665.pdf / 4
- retrieved_file_pages: []

**10. [Q0013]**
- Question: 「Seahawks」や「starcraft」のような動的なコミュニティにおけるユーザー保持率（retention rate）の特徴について、文書ではどのように述べられていますか？
- ref_file/ref_page: 1705.09665.pdf / 5
- retrieved_file_pages: []

---
*Generated by [pageBench](https://github.com/ttokunaga-ja/pagebench)*
