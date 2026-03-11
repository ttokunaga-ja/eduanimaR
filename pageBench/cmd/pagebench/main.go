// pageBench — RAG システム性能評価ツール
//
// Usage:
//
//	pagebench generate [--domain <path>]   # QA テストセット生成
//	pagebench eval     [--domain <path>]   # 評価実行
//	pagebench report   [--domain <path>]   # レポート再生成
//	pagebench upload   [--domain <path>]   # ドキュメントアップロードのみ
//	pagebench check                        # 設定確認
//
// --domain を省略した場合は PAGEBENCH_TARGET_DOMAINS の値を使用する。
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"

	"github.com/ttokunaga-ja/pagebench/internal/config"
	"github.com/ttokunaga-ja/pagebench/internal/domain"
	"github.com/ttokunaga-ja/pagebench/internal/evaluator"
	"github.com/ttokunaga-ja/pagebench/internal/generate"
	"github.com/ttokunaga-ja/pagebench/internal/prepare"
	"github.com/ttokunaga-ja/pagebench/internal/reporter"
)

var envFile string

func main() {
	root := &cobra.Command{
		Use:   "pagebench",
		Short: "pageBench — RAG システム性能評価ツール",
		Long: `pageBench は OpenAI 互換 API に対して PDF ドキュメントを投入し、
Gemini LLM-as-Judge で RAG システムの性能を評価する OSS ツールです。`,
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// .env ファイルをロード（存在しなければ無視）
			if envFile != "" {
				if err := godotenv.Load(envFile); err != nil {
					return fmt.Errorf(".env ファイルの読み込み失敗: %w", err)
				}
			} else {
				_ = godotenv.Load() // デフォルト .env（なくてもOK）
			}
			return nil
		},
	}

	root.PersistentFlags().StringVar(&envFile, "env", "", ".env ファイルのパス（省略時はカレントディレクトリの .env を使用）")

	root.AddCommand(
		generateCmd(),
		prepareCmd(),
		evalCmd(),
		reportCmd(),
		uploadCmd(),
		checkCmd(),
	)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

// resolveDomains は --domain フラグと config.TargetDomains からドメインリストを解決する。
// --domain が指定されていれば単一ドメインを返す。
// 未指定の場合は TargetDomains を返す。どちらも空なら error を返す。
func resolveDomains(flagValue string, cfgDomains []string) ([]string, error) {
	if flagValue != "" {
		return []string{flagValue}, nil
	}
	if len(cfgDomains) > 0 {
		return cfgDomains, nil
	}
	return nil, fmt.Errorf("ドメインを指定してください: --domain フラグ または PAGEBENCH_TARGET_DOMAINS 環境変数を設定してください")
}

// ─── generate ─────────────────────────────────────────────

func generateCmd() *cobra.Command {
	var (
		domainDir     string
		qaPerDoc      int
		thinkingLevel string
		model         string
		force         bool
	)

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Gemini File API を使って QA テストセットを生成する",
		Example: `  pagebench generate --domain ./domains/00_sample
  pagebench generate --domain ./domains/00_sample --qa-per-doc 20 --thinking deep
  pagebench generate   # PAGEBENCH_TARGET_DOMAINS の全ドメインを対象に実行`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			if model == "" {
				model = cfg.Gemini.GenerateModel
			}
			if thinkingLevel == "" {
				thinkingLevel = cfg.Gemini.ThinkingLevel
			}

			domains, err := resolveDomains(domainDir, cfg.TargetDomains)
			if err != nil {
				return err
			}

			var lastErr error
			for _, d := range domains {
				if len(domains) > 1 {
					fmt.Printf("\n━━━ generate: %s ━━━\n", d)
				}
				if runErr := generate.Run(ctx, generate.Options{
					DomainDir:     d,
					Model:         model,
					ThinkingLevel: thinkingLevel,
					QAPerDoc:      qaPerDoc,
					Cfg:           cfg,
					Force:         force,
				}); runErr != nil {
					fmt.Fprintf(os.Stderr, "[ERROR] %s: %v\n", d, runErr)
					lastErr = runErr
				}
			}
			return lastErr
		},
	}

	cmd.Flags().StringVarP(&domainDir, "domain", "d", "", "ドメインディレクトリパス（省略時は PAGEBENCH_TARGET_DOMAINS を使用）")
	cmd.Flags().IntVar(&qaPerDoc, "qa-per-doc", 10, "1 ドキュメントあたりの QA 生成件数")
	cmd.Flags().StringVar(&thinkingLevel, "thinking", "", "thinking レベル: minimal|low|medium|high (デフォルト: GEMINI_THINKING_LEVEL)")
	cmd.Flags().StringVar(&model, "model", "", "Gemini モデル名 (デフォルト: GEMINI_GENERATE_MODEL)")
	cmd.Flags().BoolVar(&force, "force", false, "既存の 0a/0b 出力を上書きして再生成する")

	return cmd
}

// ─── prepare ──────────────────────────────────────────────

func prepareCmd() *cobra.Command {
	var (
		domainDir string
		force     bool
	)

	cmd := &cobra.Command{
		Use:   "prepare",
		Short: "source/ の PDF を RAG システムにアップロードしてインデックスを構築する",
		Long: `pagebench prepare は OpenAI Files API 互換の POST /files エンドポイントに
source/ 内の PDF をアップロードし、インデックス完了を待機します。
完了後に {domainDir}/.pagebench_prep.json に file_ids を保存します。
次回の pagebench eval 実行時にアップロード/インデックス待機をスキップします。`,
		Example: `  pagebench prepare --domain ./domains/00_sample
  pagebench prepare --domain ./domains/04_test --force
  pagebench prepare   # PAGEBENCH_TARGET_DOMAINS の全ドメインを対象に実行`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			domains, err := resolveDomains(domainDir, cfg.TargetDomains)
			if err != nil {
				return err
			}

			var lastErr error
			for _, d := range domains {
				if len(domains) > 1 {
					fmt.Printf("\n━━━ prepare: %s ━━━\n", d)
				}
				if runErr := prepare.Run(ctx, prepare.Options{
					DomainDir: d,
					Cfg:       cfg,
					Force:     force,
				}); runErr != nil {
					fmt.Fprintf(os.Stderr, "[ERROR] %s: %v\n", d, runErr)
					lastErr = runErr
				}
			}
			return lastErr
		},
	}

	cmd.Flags().StringVarP(&domainDir, "domain", "d", "", "ドメインディレクトリパス（省略時は PAGEBENCH_TARGET_DOMAINS を使用）")
	cmd.Flags().BoolVar(&force, "force", false, "既存の .pagebench_prep.json を上書きして再アップロードする")

	return cmd
}

// ─── eval ─────────────────────────────────────────────────

func evalCmd() *cobra.Command {
	var (
		domainDir string
		limit     int
		skipJudge bool
		resume    bool
		noCleanup bool
		force     bool
	)

	cmd := &cobra.Command{
		Use:   "eval",
		Short: "RAG システムに対して評価を実行する",
		Example: `  pagebench eval --domain ./domains/00_sample
  pagebench eval --domain ./domains/00_sample --limit 10 --skip-judge
  pagebench eval --domain ./domains/00_sample --resume
  pagebench eval   # PAGEBENCH_TARGET_DOMAINS の全ドメインを対象に実行`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			domains, err := resolveDomains(domainDir, cfg.TargetDomains)
			if err != nil {
				return err
			}

			var lastErr error
			for _, d := range domains {
				if len(domains) > 1 {
					fmt.Printf("\n━━━ eval: %s ━━━\n", d)
				}
				if _, runErr := evaluator.Run(ctx, evaluator.Options{
					DomainDir: d,
					Cfg:       cfg,
					Limit:     limit,
					SkipJudge: skipJudge,
					Resume:    resume,
					NoCleanup: noCleanup,
					Force:     force,
				}); runErr != nil {
					fmt.Fprintf(os.Stderr, "[ERROR] %s: %v\n", d, runErr)
					lastErr = runErr
				}
			}
			return lastErr
		},
	}

	cmd.Flags().StringVarP(&domainDir, "domain", "d", "", "ドメインディレクトリパス（省略時は PAGEBENCH_TARGET_DOMAINS を使用）")
	cmd.Flags().IntVar(&limit, "limit", 0, "評価する最大 QA 件数（0 = 全件）")
	cmd.Flags().BoolVar(&skipJudge, "skip-judge", false, "LLM-as-Judge をスキップする")
	cmd.Flags().BoolVar(&resume, "resume", false, "チェックポイントから再開する")
	cmd.Flags().BoolVar(&noCleanup, "no-cleanup", false, "評価後にコレクションを削除しない")
	cmd.Flags().BoolVar(&force, "force", false, "既存の 0c/0d 出力を上書きして評価を実行する")

	return cmd
}

// ─── report ───────────────────────────────────────────────

func reportCmd() *cobra.Command {
	var domainDir string

	cmd := &cobra.Command{
		Use:   "report",
		Short: "results.csv から Markdown レポートを再生成する",
		Example: `  pagebench report --domain ./domains/00_sample
  pagebench report   # PAGEBENCH_TARGET_DOMAINS の全ドメインを対象に実行`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			domains, err := resolveDomains(domainDir, cfg.TargetDomains)
			if err != nil {
				return err
			}

			var lastErr error
			for _, d := range domains {
				if len(domains) > 1 {
					fmt.Printf("\n━━━ report: %s ━━━\n", d)
				}
				if runErr := runReport(d, cfg); runErr != nil {
					fmt.Fprintf(os.Stderr, "[ERROR] %s: %v\n", d, runErr)
					lastErr = runErr
				}
			}
			return lastErr
		},
	}

	cmd.Flags().StringVarP(&domainDir, "domain", "d", "", "ドメインディレクトリパス（省略時は PAGEBENCH_TARGET_DOMAINS を使用）")
	return cmd
}

func runReport(domainDir string, cfg *config.Config) error {
	records, err := domain.LoadEvaluate(domainDir)
	if err != nil {
		return fmt.Errorf("%s 読み込み失敗: %w", domain.FileEvaluate, err)
	}
	if len(records) == 0 {
		return fmt.Errorf("%s が空です。先に pagebench eval を実行してください", domain.FileEvaluate)
	}

	domainName := domainDir
	if i := len(domainDir) - 1; i > 0 {
		for i > 0 && domainDir[i] != '/' && domainDir[i] != '\\' {
			i--
		}
		domainName = domainDir[i+1:]
	}

	summary := reporter.ComputeSummary(records)
	if err := reporter.WriteMarkdownReport(records, domainDir, domainName, cfg.BackendDisplay(), summary); err != nil {
		return err
	}
	reporter.PrintSummary(summary, domainName)
	fmt.Printf("  レポート  : %s/%s\n", domainDir, domain.FileReport)
	return nil
}

// ─── upload ───────────────────────────────────────────────

func uploadCmd() *cobra.Command {
	var domainDir string

	cmd := &cobra.Command{
		Use:   "upload",
		Short: "ドキュメントをアップロードしてコレクション ID を表示する（assistants モード用）",
		Example: `  pagebench upload --domain ./domains/00_sample
  pagebench upload   # PAGEBENCH_TARGET_DOMAINS の全ドメインを対象に実行`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			domains, err := resolveDomains(domainDir, cfg.TargetDomains)
			if err != nil {
				return err
			}

			var lastErr error
			for _, d := range domains {
				if len(domains) > 1 {
					fmt.Printf("\n━━━ upload: %s ━━━\n", d)
				}
				if _, runErr := evaluator.Run(ctx, evaluator.Options{
					DomainDir:  d,
					Cfg:        cfg,
					Limit:      0,
					SkipJudge:  true,
					Resume:     false,
					NoCleanup:  true,
					UploadOnly: true,
				}); runErr != nil {
					fmt.Fprintf(os.Stderr, "[ERROR] %s: %v\n", d, runErr)
					lastErr = runErr
				}
			}
			return lastErr
		},
	}

	cmd.Flags().StringVarP(&domainDir, "domain", "d", "", "ドメインディレクトリパス（省略時は PAGEBENCH_TARGET_DOMAINS を使用）")
	return cmd
}

// ─── check ────────────────────────────────────────────────

func checkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "設定（.env）と接続を確認する",
		Example: `  pagebench check
  pagebench --env .env.prod check`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("設定読み込み失敗: %w", err)
			}

			fmt.Println("=== pageBench 設定確認 ===")
			fmt.Printf("バックエンド       : %s\n", cfg.BackendDisplay())
			fmt.Printf("Agent APIBase      : %s\n", cfg.Agent.APIBase)
			fmt.Printf("Agent APIKey       : %s\n", maskKey(cfg.Agent.APIKey))
			fmt.Printf("Gemini JudgeModel  : %s\n", cfg.Gemini.JudgeModel)
			fmt.Printf("Gemini GenModel    : %s\n", cfg.Gemini.GenerateModel)
			fmt.Printf("Gemini Thinking    : %s\n", cfg.Gemini.ThinkingLevel)
			fmt.Printf("Gemini APIKey      : %s\n", maskKey(cfg.Gemini.APIKey))
			if len(cfg.TargetDomains) > 0 {
				fmt.Printf("TargetDomains      : %v\n", cfg.TargetDomains)
			} else {
				fmt.Printf("TargetDomains      : (未設定 — --domain フラグで指定してください)\n")
			}
			fmt.Printf("ExecuteRegistry    : %v\n", cfg.ExecuteRegistry)
			fmt.Printf("ExecuteQA          : %v\n", cfg.ExecuteQA)
			fmt.Printf("ExecuteEvalPrep    : %v\n", cfg.ExecuteEvaluationPreparation)
			fmt.Printf("ExecuteEvaluation  : %v\n", cfg.ExecuteEvaluation)
			fmt.Printf("UploadPurpose      : %s\n", cfg.Upload.Purpose)
			fmt.Printf("FileStatusReady    : %s\n", cfg.Upload.FileStatusReady)
			fmt.Printf("IndexTimeout       : %ds\n", cfg.Upload.TimeoutSecs)
			fmt.Printf("IndexPollInterval  : %ds\n", cfg.Upload.PollIntervalSecs)

			// Agent 接続確認
			fmt.Print("\nAgent 接続テスト... ")
			if cfg.Agent.APIBase == "" {
				fmt.Println("SKIP (AGENT_API_BASE未設定)")
			} else {
				fmt.Println("OK（接続テストは eval 実行時に確認されます）")
			}

			return nil
		},
	}
}

func maskKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "..." + key[len(key)-4:]
}
