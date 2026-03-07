# ============================================================
# pageBench Makefile
# Professor API 性能評価ツール
#
# 使い方:
#   make setup          # Python 仮想環境セットアップ
#   make eval-academic  # 01_academic_papers を評価（ROUGE-L + LLM Judge）
#   make eval-financial # 02_financial_results を評価
#   make eval-policy    # 03_government_policy を評価
#   make eval-all       # 全ドメイン評価
#   make smoke-test     # 疎通確認（1問のみ、Judge スキップ）
# ============================================================

PYTHON   := .venv/bin/python3
EVAL     := $(PYTHON) evaluate_api.py

.DEFAULT_GOAL := help

# ─────────────────────────────────────────────
# ヘルプ
# ─────────────────────────────────────────────
.PHONY: help
help:
	@echo ""
	@echo "  pageBench — Professor API 性能評価ツール"
	@echo ""
	@echo "  セットアップ:"
	@echo "    make setup           仮想環境の作成・依存関係インストール・NLTK 取得"
	@echo "    make env             .env.example から .env を生成（未存在の場合のみ）"
	@echo ""
	@echo "  評価実行:"
	@echo "    make eval-academic   01_academic_papers を評価（英語論文）"
	@echo "    make eval-financial  02_financial_results を評価（日本語決算資料）"
	@echo "    make eval-policy     03_government_policy を評価（日本語行政文書）"
	@echo "    make eval-all        全ドメインを順番に評価"
	@echo ""
	@echo "  動作確認:"
	@echo "    make smoke-test      01_academic_papers の先頭 3 問のみ（Judge スキップ）"
	@echo "    make check-api       Professor API の死活確認（healthz）"
	@echo ""
	@echo "  クリーンアップ:"
	@echo "    make clean-results   0c_eval_results.csv を削除"
	@echo "    make clean-venv      仮想環境を削除"
	@echo ""

# ─────────────────────────────────────────────
# セットアップ
# ─────────────────────────────────────────────
.PHONY: setup
setup:
	@echo ">>> Python 仮想環境を作成中..."
	@if [ ! -d .venv ]; then python3 -m venv .venv; fi
	@echo ">>> pip をアップグレード中..."
	.venv/bin/pip install --upgrade pip --quiet
	@echo ">>> 依存関係をインストール中..."
	.venv/bin/pip install -r requirements.txt --quiet
	@echo ">>> NLTK リソースを取得中..."
	$(PYTHON) fix_nltk.py
	@echo ""
	@echo "✅ セットアップ完了！"
	@echo "   次に pageBench/.env を設定してください:"
	@echo "   cp .env.example .env && vi .env"
	@echo ""

.PHONY: env
env:
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "✅ .env を作成しました。API キーと URL を設定してください:"; \
		echo "   GEMINI_API_KEY=..."; \
		echo "   PROFESSOR_API_URL=http://localhost:8080"; \
	else \
		echo "⚠️  .env は既に存在します（上書きしません）"; \
	fi

# ─────────────────────────────────────────────
# 死活確認
# ─────────────────────────────────────────────
.PHONY: check-api
check-api:
	@echo ">>> Professor API の healthz を確認中..."
	@PROFESSOR_API_URL=$$(grep -E '^PROFESSOR_API_URL=' .env 2>/dev/null | cut -d= -f2 || echo "http://localhost:8080"); \
	curl -sf "$${PROFESSOR_API_URL}/healthz" | python3 -m json.tool || \
	(echo "❌ API に接続できません: $${PROFESSOR_API_URL}" && exit 1)
	@echo "✅ API は正常に動作しています"

# ─────────────────────────────────────────────
# 動作確認（疎通テスト）
# ─────────────────────────────────────────────
.PHONY: smoke-test
smoke-test:
	@echo ">>> スモークテスト: 01_academic_papers (3問, Judge スキップ)..."
	$(EVAL) --domain 01_academic_papers --limit 3 --no-judge --index-wait 10

# ─────────────────────────────────────────────
# 評価実行
# ─────────────────────────────────────────────
.PHONY: eval-academic
eval-academic:
	@echo ">>> 01_academic_papers を評価中..."
	$(EVAL) --domain 01_academic_papers

.PHONY: eval-financial
eval-financial:
	@echo ">>> 02_financial_results を評価中..."
	$(EVAL) --domain 02_financial_results

.PHONY: eval-policy
eval-policy:
	@echo ">>> 03_government_policy を評価中..."
	$(EVAL) --domain 03_government_policy

.PHONY: eval-all
eval-all: eval-academic eval-financial eval-policy
	@echo ""
	@echo "✅ 全ドメイン評価完了"
	@echo "   結果ファイル:"
	@ls -lh */0c_eval_results.csv 2>/dev/null || echo "   (結果ファイルなし)"

# ─────────────────────────────────────────────
# クリーンアップ
# ─────────────────────────────────────────────
.PHONY: clean-results
clean-results:
	@echo ">>> 評価結果 CSV を削除中..."
	@find . -name "0c_eval_results.csv" -delete -print
	@echo "完了"

.PHONY: clean-venv
clean-venv:
	@echo ">>> 仮想環境を削除中..."
	@rm -rf .venv
	@echo "完了"
