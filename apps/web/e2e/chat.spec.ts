import { test, expect } from '@playwright/test'

test.describe('チャットページ', () => {
  test('subject_id を表示し入力UIが有効', async ({ page }) => {
    await page.goto('/subjects/test-subject-001/chat')

    await expect(page.getByRole('heading', { name: /資料に質問する/i })).toBeVisible()
    await expect(page.getByText('科目 ID: test-subject-001')).toBeVisible()

    const input = page.getByLabel('質問入力')
    await expect(input).toBeVisible()
    await expect(input).toHaveAttribute('placeholder', '質問を入力してください（Enter で送信、Shift+Enter で改行）')

    const submit = page.getByRole('button', { name: '送信' })
    await expect(submit).toBeDisabled()

    await input.fill('線形代数の基底とは？')
    await expect(submit).toBeEnabled()
  })

  test('バックエンドエラー時に error 表示される', async ({ page }) => {
    await page.route('**/v1/subjects/**/chats', async route => {
      await route.fulfill({
        status: 500,
        contentType: 'text/plain',
        body: 'mock server error',
      })
    })

    await page.goto('/subjects/test-subject-002/chat')
    await page.getByLabel('質問入力').fill('テスト質問')
    await page.getByRole('button', { name: '送信' }).click()

    const errorAlert = page.getByRole('alert').filter({ hasText: 'HTTP 500' })
    await expect(errorAlert).toContainText('エラー:')
    await expect(errorAlert).toContainText('HTTP 500')
  })
})
