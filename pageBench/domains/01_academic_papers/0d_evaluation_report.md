# pageBench Evaluation Report

| Item | Value |
|------|-------|
| **Domain** | `01_academic_papers` |
| **Backend** | `agent @ http://localhost:8080/api/v1/subjects/7cbb30a3-476c-4e9b-bdd7-8fa85850d8d7` |
| **Date** | 2026-03-12 18:12:39 |
| **Total QA** | 102 |
| **Answered (answerable)** | 81 |
| **Unanswerable** | 21 |

## 🔖 Executive Summary

> 📝 Semantic Similarity 平均 `0.9040` (良好 ✅) ／ Judge Overall `3.95/5` (標準的 ⚡) ／ 幻覚拒否率 `85.7%` (注意が必要 ⚠️) ／ 最弱難易度: `reasoning` (SemanticSim: 0.8952)

## 📊 Score Summary

> 📌 以下の指標は **answerable（回答可能）** 問題のみを対象に集計しています。

| Metric | Score |
|--------|-------|
| **File Hit Rate** | 70.37% |
| **Page Hit Rate** | 60.49% |
| **Ref File Found Rate** | 70.37% |
| **Ref Page Found Rate** | 61.73% |
| **Ref File+Page Found Rate** | 60.49% |
| **Semantic Similarity** (avg) | 0.9040 |
| **Judge Accuracy** (avg) | 3.95 / 5 |
| **Judge Faithfulness** (avg) | 4.00 / 5 |
| **Judge Completeness** (avg) | 3.96 / 5 |
| **Judge Overall** (avg) | 3.95 / 5 |
| **Latency** (avg) | 26347 ms |

## ⏱️ Phase Latency Breakdown

> 📌 answerable 回答済みレコードのみ対象

| Metric | Value |
|--------|-------|
| **Loop Count** (avg) | 2.79 |
| **Librarian Phase** (avg) | 12184 ms |
| **Answer Generation** (avg) | 5494 ms |

## 📋 Difficulty Breakdown

> answerable 回答済みレコードのみ集計。難易度別の弱点を把握できます。

| Difficulty | N | SemanticSim | Judge | FileHitRate | LoopCount(avg) | Latency(avg) |
|:-----------|--:|------------:|------:|------------:|---------------:|-------------:|
| simple      | 29 | 0.9095 | 4.17 | 79.31% | 2.69 | 25732 ms |
| reasoning   | 31 | 0.8952 | 3.74 | 64.52% | 2.87 | 27141 ms |
| multi_chunk | 21 | 0.9095 | 3.95 | 66.67% | 2.81 | 26025 ms |

## 🔄 Loop Count Distribution

> answerable 回答済みレコードのみ対象。多ループは検索の苦戦を示す可能性があります。

| Loops | Count | Bar |
|------:|------:|-----|
| 1     | 7 | █ |
| 2     | 3 |  |
| 3+    | 71 | █████████████████ |

## 📈 Semantic Similarity Distribution

> gemini-embedding-001 (SEMANTIC_SIMILARITY) によるコサイン類似度分布

| Range | Count | Bar |
|-------|------:|-----|
| 0.9–1.0 | 54 | █████████████ |
| 0.7–0.9 | 27 | ██████ |
| 0.5–0.7 | 0 |  |
| 0.3–0.5 | 0 |  |
| 0.0–0.3 | 0 |  |

## ⚖️ Judge Overall Distribution

| Score | Count | Bar |
|------:|------:|-----|
| 1 / 5 | 20 | ████ |
| 2 / 5 | 1 |  |
| 3 / 5 | 1 |  |
| 4 / 5 | 0 |  |
| 5 / 5 | 59 | ██████████████ |

## 🧪 Hallucination Check (Unanswerable Questions)

> 回答不能問題 (21 件) に対する RAG の幻覚抑制チェック

| Metric | Value |
|--------|-------|
| **Unanswerable Count** | 21 |
| **Hallucination Refusal Rate** | 85.71% |
| **Avg Hallucination Score** (1=OK, 5=worst) | 1.00 / 5 |
| **Avg Latency (unanswerable)** | 25800 ms |
| **Avg Loop Count (unanswerable)** | 2.81 |

### Hallucination Score Distribution

| Score | Meaning | Count | Bar |
|------:|---------|------:|-----|
| 1 | 正確に拒否 | 20 | ████████████████████ |
| 2 | ほぼ拒否 | 0 |  |
| 3 | 部分的幻覚 | 0 |  |
| 4 | 重度の幻覚 | 0 |  |
| 5 | 完全な幻覚 | 0 |  |

### ⚠️ Hallucination Cases (RAG failed to refuse)

**1. [Q0035]**
- Question: How does the application of Layer-Wise Relevance Propagation (LRP) to the CNN/Da...
- rag_answer: Based on the provided research paper, the comparison between the application of Layer-Wise Relevance Propagation (LRP) a...
- answerability: answerable
- judge_hallucination: 1
- reasoning: システムは、質問にあるLRPとSee et al. [2017]の性能指標の比較について、論文内に定量的データが存在しないことを正しく指摘し、将来の課題であると言及しているため。

**2. [Q0039]**
- Question: Does the probabilistic framework presented by Joren and Alvarez-Melis include sp...
- rag_answer: Based on the provided PDF, here is the answer to your question:

No, the probabilistic framework presented by Joren and ...
- answerability: answerable
- judge_hallucination: 1
- reasoning: システムは質問された特定の技術（BERTや人種バイアス緩和）が文書に含まれていないことを正しく指摘し、文書の実際の範囲（性別バイアスや静的埋め込み）を正確に説明しています。

**3. [Q0080]**
- Question: この研究で提案されたマルチリンガル・キャプションモデルを、Google Cloud Vision APIと連携させてリアルタイム翻訳を行う際のAPI利用料金はい...
- rag_answer: 
- answerability: 
- judge_hallucination: 
- reasoning: 

## 🏆 Top 3 by Semantic Similarity

**1. SemanticSim=0.9550** `[Q0088]` `difficulty=multi_chunk`
- Question: Based on the comparisons across the CNN, TIME, and 20 Newsgroups data sets, whic...
- ref_answer: The weGAN model consistently outperformed deGAN in classification accuracy across the CNN (92.36% vs 92.29%), TIME (84.7...
- rag_answer: Based on the data provided in the PDF, the comparison between the **weGAN** and **deGAN** models reveals the following r...

**2. SemanticSim=0.9528** `[Q0095]` `difficulty=reasoning`
- Question: How does the iBLEU score relate to the amount of noise in incomplete text accord...
- ref_answer: The iBLEU score is calculated as 1 - BLEU; a higher iBLEU score indicates a higher level of noise in the incomplete text...
- rag_answer: Based on the provided document, the **inverted BLEU (iBLEU)** score is a metric used to denote the level of noise in inc...

**3. SemanticSim=0.9514** `[Q0057]` `difficulty=reasoning`
- Question: Based on the performance comparison in Table 1, which specific SciSpacy model ac...
- ref_answer: The SciSpacy (BC5CDR) model achieved an F1 score of 68.46 for the 'Disease' entity type. The CORD-NER ('Ours') system ou...
- rag_answer: Based on Table 1 (page 2) of the provided document, the specific SciSpacy model that achieved the highest F1 score for t...

## 🔍 Bottom 3 by Semantic Similarity (for improvement)

**1. SemanticSim=0.7614** `[Q0013]` `difficulty=reasoning`
- Question: Based on the Random Forest regressor analysis in Section 3.1, what was the resul...
- ref_answer: Combining features from both the identity-based typology and the activity-based baseline resulted in a statistically sig...
- rag_answer: Based on the provided course materials, it is not possible to answer this question.

While **Reference 4** mentions that...

**2. SemanticSim=0.7978** `[Q0009]` `difficulty=multi_chunk`
- Question: PO-EMOのドイツ語コーパス（ANTI-K）に含まれる731の連は、何年から何年までのタイムラインに分散していますか？
- ref_answer: 1575年から1936年までの新高ドイツ語のタイムラインに分散しています。
- rag_answer: 提供された資料に基づき回答いたします。

提供された4つの資料（Reference 1〜5）の中には、PO-EMOのドイツ語コーパス（ANTI-K）に含まれる連（stanzas）のタイムラインや、それが何年から何年までのものかという具体的な...

**3. SemanticSim=0.8063** `[Q0085]` `difficulty=reasoning`
- Question: In the weGAN model for cross-corpus word embeddings, what is the specific role o...
- ref_answer: In equation (8), ekm_i is a unit vector where only the km_i-th component is one and all others are zero, representing th...
- rag_answer: Based on the provided course materials, there is no mention of a "weGAN" model or a unit vector $e_{km_i}$.

The materia...

## 🔁 High Loop Count Cases (LoopCount ≥ 2)

> 多ループが発生した問題は RAG の検索に苦戦している可能性があります。

**1. [Q0013] loops=3, SemanticSim=0.7614, difficulty=reasoning**
- Question: Based on the Random Forest regressor analysis in Section 3.1, what was the resulting R2 value when c...
- rag_answer: Based on the provided course materials, it is not possible to answer this question.

While **Reference 4** mentions that...

**2. [Q0009] loops=3, SemanticSim=0.7978, difficulty=multi_chunk**
- Question: PO-EMOのドイツ語コーパス（ANTI-K）に含まれる731の連は、何年から何年までのタイムラインに分散していますか？
- rag_answer: 提供された資料に基づき回答いたします。

提供された4つの資料（Reference 1〜5）の中には、PO-EMOのドイツ語コーパス（ANTI-K）に含まれる連（stanzas）のタイムラインや、それが何年から何年までのものかという具体的な...

**3. [Q0085] loops=3, SemanticSim=0.8063, difficulty=reasoning**
- Question: In the weGAN model for cross-corpus word embeddings, what is the specific role of the unit vector ek...
- rag_answer: Based on the provided course materials, there is no mention of a "weGAN" model or a unit vector $e_{km_i}$.

The materia...

**4. [Q0072] loops=3, SemanticSim=0.8128, difficulty=simple**
- Question: 2D-ConvNetモデルにおいて、log-Melスペクトラムのカラーチャネル間の相互依存性を見つけ出し、各チャネルに重要度を適応的に割り当てるために導入されたネットワークは何ですか？
- rag_answer: 提供された教材に基づき、ご質問にお答えします。

提供された5つの参考文献（Reference 1〜5）を確認しましたが、2D-ConvNetモデルにおけるlog-Melスペクトラムのチャネル間相互依存性や、各チャネルへの適応的な重要度割り...

**5. [Q0099] loops=3, SemanticSim=0.8314, difficulty=reasoning**
- Question: What specific hardware and optimization settings were used to fine-tune the conventional BERT-Base U...
- rag_answer: Based on the provided document, there appears to be a misunderstanding in the student's question regarding the specific ...

## 📁 Ref File Miss Cases

参照ファイルを取得できなかったケースを表示します。

**1. [Q0002]**
- Question: Minimally Supervised Learning of Affective Eventsの研究で提案された手法において、BiGRUエンコーダを使用しAL+CA+COデータセットで訓練した場合...
- ref_file/ref_page: 1909.00694.pdf / 5
- retrieved_file_pages: []

**2. [Q0007]**
- Question: PO-EMOプロジェクトで定義された9つの美的感情ラベルのうち、Schindlerら(2017)の「Amusement」因子に対応し、面白さや笑いを誘う感情を指すラベルは何ですか？
- ref_file/ref_page: 2003.07723.pdf / 4
- retrieved_file_pages: []

**3. [Q0008]**
- Question: PO-EMOの専門家アノテーションにおいて、ドイツ語と英語のそれぞれのコーパスで最も頻繁に出現した感情ラベルの上位2つは何ですか？
- ref_file/ref_page: 2003.07723.pdf / 7
- retrieved_file_pages: []

**4. [Q0009]**
- Question: PO-EMOのドイツ語コーパス（ANTI-K）に含まれる731の連は、何年から何年までのタイムラインに分散していますか？
- ref_file/ref_page: 2003.07723.pdf / 3
- retrieved_file_pages: []

**5. [Q0012]**
- Question: According to Figure 2, how do the 'Seahawks' and 'Cooking' Reddit communities differ in terms of the...
- ref_file/ref_page: 1705.09665.pdf / 3, 5
- retrieved_file_pages: []

**6. [Q0013]**
- Question: Based on the Random Forest regressor analysis in Section 3.1, what was the resulting R2 value when c...
- ref_file/ref_page: 1705.09665.pdf / 5
- retrieved_file_pages: []

**7. [Q0014]**
- Question: What specific linguistic preprocessing step did Justine Zhang and William L. Hamilton implement to p...
- ref_file/ref_page: 1705.09665.pdf / 4
- retrieved_file_pages: []

**8. [Q0018]**
- Question: QA-CTSタスクの提案モデルにおいて、臨床固有表現認識（CNER）モデルから得られた出力タグシーケンス（InqおよびInt）は、最終的な予測を行う前にどのようにBERTモデルの出力と統合されますか？
- ref_file/ref_page: 1908.06606.pdf / 2
- retrieved_file_pages: []

**9. [Q0029]**
- Question: What is the total number of features used by the NMT-Fake review detector for '1/2/3-grams of detail...
- ref_file/ref_page: 1805.02400.pdf / 21
- retrieved_file_pages: []

**10. [Q0037]**
- Question: In the fastText embedding experiments, how does the 'Composite NN + Prob' method compare to the 'Geo...
- ref_file/ref_page: 1910.14497.pdf / 4
- retrieved_file_pages: []

---
*Generated by [pageBench](https://github.com/ttokunaga-ja/pagebench)*
