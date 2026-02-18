import type { NextConfig } from 'next';

const nextConfig: NextConfig = {
  // Docker イメージをスリム化するための standalone 出力
  // `.next/standalone` に自己完結型サーバーが生成される
  output: 'standalone',
};

export default nextConfig;
