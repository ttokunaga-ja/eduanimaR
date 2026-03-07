#!/usr/bin/env python3
"""
evaluate_api.py — Professor API エンドポイント性能評価スクリプト

評価フロー:
  1. 0a_registry.csv を読み込み、対象 PDF を Professor API にアップロード
  2. インデックス処理の完了を待機
  3. 0b_qa_pairs.csv の各質問を POST /v1/subjects/:id/chats (SSE) で送信
  4. 回答を収集し ROUGE-L スコアを計算
  5. Gemini LLM-as-Judge で多角的採点（正確性・忠実性・完全性）
  6. 結果を 0c_eval_results.csv に出力

使い方:
  python evaluate_api.py --domain 01_academic_papers
  python evaluate_api.py --domain 02_financial_results --limit 10
  python evaluate_api.py --domain 03_government_policy --no-judge
"""

import argparse
import csv
import json
import os
import shutil
import sys
import time
from datetime import datetime
from pathlib import Path
from typing import Optional

import httpx
from dotenv import load_dotenv
from google import genai
from google.genai import types
from pydantic import BaseModel, Field


# ─────────────────────────────────────────────
# 1. Pydantic スキーマ（LLM-as-Judge 出力）
# ─────────────────────────────────────────────

class JudgeResult(BaseModel):
    accuracy: int = Field(ge=1, le=5, description="正確性 (1-5): 参照解答と意味的に一致しているか")
    faithfulness: int = Field(ge=1, le=5, description="忠実性 (1-5): 根拠テキストから逸脱していないか（ハルシネーション）")
    completeness: int = Field(ge=1, le=5, description="完全性 (1-5): 質問に対して必要な情報が揃っているか")
    overall: int = Field(ge=1, le=5, description="総合評価 (1-5)")
    reasoning: str = Field(description="採点の根拠（簡潔に）")


# ─────────────────────────────────────────────
# 2. 設定ロード
# ─────────────────────────────────────────────

def load_config(root_dir: Path) -> dict:
    """pageBench/.env から設定を読み込む"""
    load_dotenv(root_dir / ".env")
    return {
        "api_url": os.getenv("PROFESSOR_API_URL", "http://localhost:8080"),
        "dev_user": os.getenv("PROFESSOR_DEV_USER", "dev-user"),
        "index_wait_secs": int(os.getenv("PROFESSOR_INDEX_WAIT_SECS", "30")),
        "gemini_api_key": os.getenv("GEMINI_API_KEY"),
        "gemini_judge_model": os.getenv("GEMINI_JUDGE_MODEL", "gemini-3.1-pro-preview"),
        "timeout_ms": int(os.getenv("GEMINI_TIMEOUT_MS", "120000")),
        "rate_limit_sleep": float(os.getenv("GEMINI_RATE_LIMIT_SLEEP_SEC", "3")),
    }


# ─────────────────────────────────────────────
# 3. Professor API ヘルパー
# ─────────────────────────────────────────────

def create_subject(base_url: str, dev_user: str, name: str) -> str:
    """科目を作成して subject_id を返す"""
    resp = httpx.post(
        f"{base_url}/v1/subjects",
        json={"name": name},
        headers={"X-Dev-User": dev_user, "Content-Type": "application/json"},
        timeout=30,
    )
    resp.raise_for_status()
    data = resp.json()
    # subject_id キーが存在しない場合も考慮
    return data.get("subject_id") or data.get("id") or data["subject_id"]


def upload_material(base_url: str, dev_user: str, subject_id: str, pdf_path: Path) -> Optional[str]:
    """PDF をアップロードして file_id を返す（Content-Type は httpx が自動設定）"""
    with open(pdf_path, "rb") as f:
        resp = httpx.post(
            f"{base_url}/v1/subjects/{subject_id}/materials",
            files={"file": (pdf_path.name, f, "application/pdf")},
            headers={"X-Dev-User": dev_user},
            timeout=120,
        )
    resp.raise_for_status()
    data = resp.json()
    return data.get("file_id") or data.get("material_id") or data.get("id")


def ask_question_sse(
    base_url: str,
    dev_user: str,
    subject_id: str,
    question: str,
    timeout: int = 180,
) -> tuple[str, int]:
    """
    SSE チャットエンドポイントに質問を送信し、完全な回答テキストと所要時間(ms)を返す。
    Professor API の SSE 形式: data: <JSON> または data: [DONE]
    """
    start_time = time.time()
    answer_chunks: list[str] = []

    with httpx.stream(
        "POST",
        f"{base_url}/v1/subjects/{subject_id}/chats",
        json={"message": question},
        headers={
            "X-Dev-User": dev_user,
            "Content-Type": "application/json",
            "Accept": "text/event-stream",
        },
        timeout=timeout,
    ) as response:
        response.raise_for_status()
        for line in response.iter_lines():
            if not line:
                continue
            if line.startswith("data:"):
                data_str = line[5:].strip()
                if data_str == "[DONE]":
                    break
                if not data_str:
                    continue
                try:
                    data = json.loads(data_str)
                    # Professor API が返す可能性のある複数のフィールド名に対応
                    content = (
                        data.get("content")
                        or data.get("delta")
                        or data.get("text")
                        or data.get("message")
                        or ""
                    )
                    if content and isinstance(content, str):
                        answer_chunks.append(content)
                except json.JSONDecodeError:
                    # JSON でない場合はテキストとしてそのまま使用
                    if data_str not in ("[DONE]", ""):
                        answer_chunks.append(data_str)

    latency_ms = int((time.time() - start_time) * 1000)
    return "".join(answer_chunks).strip(), latency_ms


# ─────────────────────────────────────────────
# 4. 評価指標
# ─────────────────────────────────────────────

def _is_cjk_dominant(text: str) -> bool:
    """テキストの過半数が CJK 文字か判定（日本語・中国語判定用）"""
    if not text:
        return False
    cjk_count = sum(
        1 for c in text
        if "\u3000" <= c <= "\u9FFF" or "\uF900" <= c <= "\uFAFF"
    )
    return cjk_count / len(text) > 0.3


def _lcs_length(a: list[str], b: list[str]) -> int:
    """LCS（最長共通部分列）の長さを DP で計算する"""
    m, n = len(a), len(b)
    # メモリ節約のため 1D DP を使用
    prev = [0] * (n + 1)
    for i in range(m):
        curr = [0] * (n + 1)
        for j in range(n):
            if a[i] == b[j]:
                curr[j + 1] = prev[j] + 1
            else:
                curr[j + 1] = max(prev[j + 1], curr[j])
        prev = curr
    return prev[n]


def compute_rouge_l(prediction: str, reference: str) -> float:
    """
    ROUGE-L F スコアを純粋 Python で計算する（外部ライブラリ不要）。
    - 英語: 単語レベル（空白分割）
    - 日本語（CJK 主体）: 文字レベル
    """
    if not prediction or not reference:
        return 0.0

    if _is_cjk_dominant(reference):
        pred_tokens: list[str] = list(prediction)
        ref_tokens: list[str] = list(reference)
    else:
        pred_tokens = prediction.lower().split()
        ref_tokens = reference.lower().split()

    if not pred_tokens or not ref_tokens:
        return 0.0

    lcs = _lcs_length(ref_tokens, pred_tokens)
    precision = lcs / len(pred_tokens)
    recall = lcs / len(ref_tokens)
    if precision + recall == 0:
        return 0.0
    # beta=1 の F スコア (F1)
    f_score = 2.0 * precision * recall / (precision + recall)
    return round(f_score, 4)


JUDGE_PROMPT_TEMPLATE = """\
あなたは厳格な RAG システム評価者です。
以下の情報をもとに、システムの回答を多角的に評価してください。

【質問】
{question}

【参照解答（正解）】
{reference_answer}

【根拠テキスト（ソース文書からの抜粋）】
{evidence_text}

【システムの回答】
{system_answer}

以下の基準で各項目を 1〜5 点で採点してください:
- accuracy（正確性）: 参照解答と意味的に一致しているか（1=全く違う, 5=完全に一致）
- faithfulness（忠実性）: 根拠テキストから逸脱していないか（ハルシネーション）（1=大幅に逸脱, 5=完全に根拠に基づく）
- completeness（完全性）: 質問に対して必要な情報が揃っているか（1=不十分, 5=完全）
- overall（総合評価）: 上記を総合した評価
- reasoning: 採点の根拠を日本語で 1〜3 文で説明してください
"""


def judge_with_llm(
    client: genai.Client,
    model: str,
    question: str,
    reference_answer: str,
    system_answer: str,
    evidence_text: str,
    timeout_ms: int,
) -> JudgeResult:
    """Gemini LLM-as-Judge で採点する"""
    prompt = JUDGE_PROMPT_TEMPLATE.format(
        question=question,
        reference_answer=reference_answer[:2000],
        evidence_text=(evidence_text[:1500] if evidence_text else "（なし）"),
        system_answer=(system_answer[:3000] if system_answer else "（回答なし）"),
    )
    response = client.models.generate_content(
        model=model,
        contents=[prompt],
        config=types.GenerateContentConfig(
            response_mime_type="application/json",
            response_schema=JudgeResult,
            temperature=0.1,
            http_options=types.HttpOptions(timeout=timeout_ms),
        ),
    )
    return JudgeResult.model_validate_json(response.text)


# ─────────────────────────────────────────────
# 5. メイン処理
# ─────────────────────────────────────────────

def run(domain: str, limit: Optional[int], skip_judge: bool, index_wait_override: Optional[int]):
    root_dir = Path(__file__).resolve().parent
    topic_dir = root_dir / domain

    if not topic_dir.exists():
        print(f"[ERROR] ドメインディレクトリが見つかりません: {topic_dir}", file=sys.stderr)
        sys.exit(1)

    cfg = load_config(root_dir)

    if not cfg["gemini_api_key"] and not skip_judge:
        print("[ERROR] GEMINI_API_KEY が .env に設定されていません。--no-judge オプションで Judge をスキップできます。", file=sys.stderr)
        sys.exit(1)

    index_wait = index_wait_override if index_wait_override is not None else cfg["index_wait_secs"]

    # ── ファイルパス ──
    registry_file = topic_dir / "0a_registry.csv"
    qa_file = topic_dir / "0b_qa_pairs.csv"
    output_file = topic_dir / "0c_eval_results.csv"
    pdf_dir = topic_dir / "source_pdfs"

    # ── データ読み込み ──
    with open(registry_file, encoding="utf-8") as f:
        registry = {row["file_name"]: row for row in csv.DictReader(f)}

    with open(qa_file, encoding="utf-8") as f:
        qa_pairs = list(csv.DictReader(f))

    if not qa_pairs:
        print(f"[WARN] QA ペアが見つかりません: {qa_file}")
        return

    if limit:
        qa_pairs = qa_pairs[:limit]

    print(f"\n{'='*60}")
    print(f"  pageBench — Professor API 性能評価")
    print(f"  ドメイン  : {domain}")
    print(f"  API URL   : {cfg['api_url']}")
    print(f"  対象 QA   : {len(qa_pairs)} 件")
    print(f"  LLM Judge : {'スキップ' if skip_judge else cfg['gemini_judge_model']}")
    print(f"{'='*60}\n")

    # ── Gemini クライアント初期化 ──
    gemini_client = None
    if not skip_judge:
        gemini_client = genai.Client(
            api_key=cfg["gemini_api_key"],
            http_options=types.HttpOptions(timeout=cfg["timeout_ms"]),
        )

    base_url = cfg["api_url"]
    dev_user = cfg["dev_user"]

    # ── Step 1: 科目作成 ──
    subject_name = f"[bench] {domain} {datetime.now().strftime('%Y%m%d_%H%M%S')}"
    print(f"[1/4] 科目を作成中: {subject_name}")
    try:
        subject_id = create_subject(base_url, dev_user, subject_name)
        print(f"      subject_id: {subject_id}\n")
    except httpx.ConnectError:
        print(f"[ERROR] Professor API に接続できません。サーバーが起動しているか確認してください: {base_url}", file=sys.stderr)
        sys.exit(1)
    except Exception as e:
        print(f"[ERROR] 科目作成に失敗: {e}", file=sys.stderr)
        sys.exit(1)

    # ── Step 2: PDF アップロード ──
    print(f"[2/4] PDF をアップロード中 ({len(registry)} ファイル)...")
    uploaded = set()
    for file_name, _meta in registry.items():
        pdf_path = pdf_dir / file_name
        if not pdf_path.exists():
            print(f"  [SKIP] PDF が見つかりません: {file_name}")
            continue
        if file_name in uploaded:
            continue

        # 日本語ファイル名によるエンコードエラー回避のため一時ファイルを使用
        safe_name = f"bench_{abs(hash(file_name)) % 0xFFFFFF:06x}.pdf"
        temp_path = pdf_dir / safe_name
        try:
            shutil.copy2(pdf_path, temp_path)
            upload_material(base_url, dev_user, subject_id, temp_path)
            uploaded.add(file_name)
            print(f"  ✓ {file_name}")
        except Exception as e:
            print(f"  ✗ {file_name}: アップロード失敗 ({e})")
        finally:
            if temp_path.exists():
                temp_path.unlink()
        time.sleep(1)

    # ── Step 3: インデックス完了待機 ──
    print(f"\n[3/4] インデックス処理の完了を待機中 ({index_wait} 秒)...")
    time.sleep(index_wait)

    # ── Step 4: 評価ループ ──
    print(f"\n[4/4] 評価開始 ({len(qa_pairs)} 件)...")
    fieldnames = [
        "q_id", "domain", "question", "reference_answer", "system_answer",
        "rouge_l",
        "judge_accuracy", "judge_faithfulness", "judge_completeness",
        "judge_overall", "judge_reasoning",
        "latency_ms", "target_file", "target_page",
    ]

    results: list[dict] = []

    for i, qa in enumerate(qa_pairs, 1):
        q_id = qa.get("q_id", "")
        question = qa.get("question", "")
        reference_answer = qa.get("reference_answer", "")
        evidence_text = qa.get("evidence_text", "")
        target_file = qa.get("target_file", "")
        target_page = qa.get("target_page", "")

        print(f"\n  [{i:03d}/{len(qa_pairs):03d}] {question[:70]}...")

        # 質問送信（SSE）
        system_answer = ""
        latency_ms = -1
        try:
            system_answer, latency_ms = ask_question_sse(
                base_url, dev_user, subject_id, question
            )
            print(f"          回答取得完了 ({latency_ms} ms) | 文字数: {len(system_answer)}")
        except Exception as e:
            print(f"          [ERROR] 回答取得失敗: {e}")

        # ROUGE-L 計算
        rouge_l = compute_rouge_l(system_answer, reference_answer)

        # LLM-as-Judge 採点
        judge_accuracy = judge_faithfulness = judge_completeness = judge_overall = ""
        judge_reasoning = ""

        if not skip_judge and gemini_client and system_answer:
            try:
                judge = judge_with_llm(
                    gemini_client,
                    cfg["gemini_judge_model"],
                    question,
                    reference_answer,
                    system_answer,
                    evidence_text,
                    cfg["timeout_ms"],
                )
                judge_accuracy = judge.accuracy
                judge_faithfulness = judge.faithfulness
                judge_completeness = judge.completeness
                judge_overall = judge.overall
                judge_reasoning = judge.reasoning
                print(
                    f"          Judge: accuracy={judge.accuracy} "
                    f"faithfulness={judge.faithfulness} "
                    f"completeness={judge.completeness} "
                    f"overall={judge.overall}"
                )
            except Exception as e:
                print(f"          [WARN] Judge エラー: {e}")

        results.append({
            "q_id": q_id,
            "domain": domain,
            "question": question,
            "reference_answer": reference_answer,
            "system_answer": system_answer,
            "rouge_l": rouge_l,
            "judge_accuracy": judge_accuracy,
            "judge_faithfulness": judge_faithfulness,
            "judge_completeness": judge_completeness,
            "judge_overall": judge_overall,
            "judge_reasoning": judge_reasoning,
            "latency_ms": latency_ms,
            "target_file": target_file,
            "target_page": target_page,
        })

        time.sleep(cfg["rate_limit_sleep"])

    # ── 結果 CSV 出力 ──
    with open(output_file, mode="w", encoding="utf-8", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=fieldnames)
        writer.writeheader()
        writer.writerows(results)

    # ── サマリー ──
    answered = [r for r in results if r["system_answer"]]
    rouge_scores = [r["rouge_l"] for r in answered]
    judge_overalls = [int(r["judge_overall"]) for r in results if str(r["judge_overall"]).isdigit()]

    avg_rouge = round(sum(rouge_scores) / len(rouge_scores), 4) if rouge_scores else 0.0
    avg_judge = round(sum(judge_overalls) / len(judge_overalls), 2) if judge_overalls else "N/A"
    avg_latency = round(sum(r["latency_ms"] for r in answered) / len(answered)) if answered else 0

    print(f"\n{'='*60}")
    print(f"  📊 評価サマリー: {domain}")
    print(f"  総問題数          : {len(results)}")
    print(f"  回答あり          : {len(answered)}")
    print(f"  平均 ROUGE-L      : {avg_rouge}")
    print(f"  平均 Judge Overall: {avg_judge} / 5")
    print(f"  平均レイテンシ    : {avg_latency} ms")
    print(f"  結果出力先        : {output_file}")
    print(f"{'='*60}\n")


# ─────────────────────────────────────────────
# CLI エントリポイント
# ─────────────────────────────────────────────

def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="pageBench — Professor API 性能評価スクリプト",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
例:
  python evaluate_api.py --domain 01_academic_papers
  python evaluate_api.py --domain 02_financial_results --limit 5
  python evaluate_api.py --domain 03_government_policy --no-judge
  python evaluate_api.py --domain 01_academic_papers --index-wait 60
        """,
    )
    parser.add_argument(
        "--domain",
        required=True,
        choices=["00_sample", "01_academic_papers", "02_financial_results", "03_government_policy"],
        help="評価するドメインディレクトリ名",
    )
    parser.add_argument(
        "--limit",
        type=int,
        default=None,
        help="評価する最大 QA ペア数（動作確認用）",
    )
    parser.add_argument(
        "--no-judge",
        action="store_true",
        help="LLM-as-Judge をスキップする（コスト削減・疎通確認用）",
    )
    parser.add_argument(
        "--index-wait",
        type=int,
        default=None,
        help="インデックス完了待機秒数（.env の PROFESSOR_INDEX_WAIT_SECS を上書き）",
    )
    return parser.parse_args()


if __name__ == "__main__":
    args = parse_args()
    run(
        domain=args.domain,
        limit=args.limit,
        skip_judge=args.no_judge,
        index_wait_override=args.index_wait,
    )
