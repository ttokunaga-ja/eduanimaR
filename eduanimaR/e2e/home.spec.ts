import { test, expect } from '@playwright/test'

/**
 * ホームページ E2E テスト（最小導線）
 * インフラ: docker compose up postgres kafka minio kafka-init minio-init
 * バックエンド: professor が起動していること
 */
test.describe('ホームページ', () => {
  test('ページタイトルが表示される', async ({ page }) => {
    await page.goto('/')
    await expect(page).toHaveTitle(/EduAnima/i)
  })

  test('ページが 200 で返る', async ({ page }) => {
    const response = await page.goto('/')
    expect(response?.status()).toBe(200)
  })
})
