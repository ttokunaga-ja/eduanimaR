import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ChatInput } from './ChatInput'

describe('ChatInput', () => {
  // ─── 表示 ──────────────────────────────────────────────────────

  it('テキストエリアと送信ボタンが表示される', () => {
    render(<ChatInput onSubmit={vi.fn()} />)
    expect(screen.getByRole('textbox', { name: '質問入力' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '送信' })).toBeInTheDocument()
  })

  it('初期状態では送信ボタンが無効', () => {
    render(<ChatInput onSubmit={vi.fn()} />)
    expect(screen.getByRole('button', { name: '送信' })).toBeDisabled()
  })

  it('disabled=true のとき「処理中…」と表示され操作不可', () => {
    render(<ChatInput onSubmit={vi.fn()} disabled />)
    // aria-label="送信" が accessible name のため name: '送信' で取得
    const btn = screen.getByRole('button', { name: '送信' })
    expect(btn).toBeDisabled()
    // 表示テキストが「処理中…」に変わること
    expect(btn).toHaveTextContent('処理中…')
    expect(screen.getByRole('textbox', { name: '質問入力' })).toBeDisabled()
  })

  // ─── 入力操作 ──────────────────────────────────────────────────

  it('テキスト入力後に送信ボタンが有効になる', async () => {
    const user = userEvent.setup()
    render(<ChatInput onSubmit={vi.fn()} />)

    await user.type(screen.getByRole('textbox', { name: '質問入力' }), '質問です')
    expect(screen.getByRole('button', { name: '送信' })).toBeEnabled()
  })

  it('空白のみの入力では送信ボタンが無効のまま', async () => {
    const user = userEvent.setup()
    render(<ChatInput onSubmit={vi.fn()} />)

    await user.type(screen.getByRole('textbox', { name: '質問入力' }), '   ')
    expect(screen.getByRole('button', { name: '送信' })).toBeDisabled()
  })

  // ─── 送信動作 ──────────────────────────────────────────────────

  it('送信ボタンをクリックすると onSubmit が trimmed テキストで呼ばれる', async () => {
    const user = userEvent.setup()
    const handleSubmit = vi.fn()
    render(<ChatInput onSubmit={handleSubmit} />)

    await user.type(screen.getByRole('textbox', { name: '質問入力' }), '  テスト質問  ')
    await user.click(screen.getByRole('button', { name: '送信' }))

    expect(handleSubmit).toHaveBeenCalledOnce()
    expect(handleSubmit).toHaveBeenCalledWith('テスト質問')
  })

  it('送信後にテキストエリアがクリアされる', async () => {
    const user = userEvent.setup()
    render(<ChatInput onSubmit={vi.fn()} />)
    const textarea = screen.getByRole('textbox', { name: '質問入力' })

    await user.type(textarea, '質問テキスト')
    await user.click(screen.getByRole('button', { name: '送信' }))

    expect(textarea).toHaveValue('')
  })

  it('Enter キーで送信される', async () => {
    const user = userEvent.setup()
    const handleSubmit = vi.fn()
    render(<ChatInput onSubmit={handleSubmit} />)

    await user.type(screen.getByRole('textbox', { name: '質問入力' }), 'Enter送信テスト{Enter}')

    expect(handleSubmit).toHaveBeenCalledOnce()
    expect(handleSubmit).toHaveBeenCalledWith('Enter送信テスト')
  })

  it('Shift+Enter では送信されず改行される', async () => {
    const user = userEvent.setup()
    const handleSubmit = vi.fn()
    render(<ChatInput onSubmit={handleSubmit} />)

    await user.type(screen.getByRole('textbox', { name: '質問入力' }), '1行目{Shift>}{Enter}{/Shift}2行目')

    expect(handleSubmit).not.toHaveBeenCalled()
    expect(screen.getByRole('textbox', { name: '質問入力' })).toHaveValue('1行目\n2行目')
  })

  it('disabled 状態では Enter キーで送信されない', async () => {
    const user = userEvent.setup()
    const handleSubmit = vi.fn()
    render(<ChatInput onSubmit={handleSubmit} disabled />)

    // disabled なのでタイプ自体が効かない
    await user.type(screen.getByRole('textbox', { name: '質問入力' }), '質問{Enter}')

    expect(handleSubmit).not.toHaveBeenCalled()
  })
})
