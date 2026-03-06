package main

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// initTracer は OTEL グローバルトレーサープロバイダーを初期化する。
//
// OTEL_EXPORTER_OTLP_ENDPOINT が空の場合は noop 相当（exporterなし）のプロバイダーを使う。
// 本番環境では OTEL_EXPORTER_OTLP_ENDPOINT を設定することでトレースが Cloud Trace/Jaeger 等に送信される。
// OTLP exporter 実装は Phase 3 で追加予定。
//
// 戻り値の shutdown 関数は必ず defer で呼ぶこと。
func initTracer(serviceName, otelEndpoint string) func(context.Context) {
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(semconv.ServiceName(serviceName)),
	)
	if err != nil {
		slog.Warn("OTEL resource creation failed; using default", "error", err)
		res = resource.Default()
	}

	var opts []sdktrace.TracerProviderOption
	opts = append(opts, sdktrace.WithResource(res))

	if otelEndpoint != "" {
		// TODO(Phase 3): OTLP gRPC exporter を追加
		// exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint(otelEndpoint), ...)
		// opts = append(opts, sdktrace.WithBatcher(exporter))
		slog.Info("OTEL endpoint configured (exporter hookup: Phase 3)", "endpoint", otelEndpoint)
	}

	tp := sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(tp)

	if otelEndpoint != "" {
		slog.Info("OTEL tracer provider initialized", "service", serviceName)
	}

	return func(ctx context.Context) {
		if err := tp.Shutdown(ctx); err != nil {
			slog.Warn("OTEL tracer provider shutdown error", "error", err)
		}
	}
}
