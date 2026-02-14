'use server';

export type RequestLoginLinkState =
  | { status: 'idle' }
  | { status: 'success'; message: string }
  | { status: 'error'; message: string };

export async function requestLoginLink(
  _prevState: RequestLoginLinkState,
  formData: FormData,
): Promise<RequestLoginLinkState> {
  const emailRaw = formData.get('email');
  const email = typeof emailRaw === 'string' ? emailRaw.trim() : '';

  if (!email) {
    return { status: 'error', message: 'Email is required.' };
  }

  // NOTE: This is a minimal template sample.
  // In a real app, call a server-only DAL function here and return expected errors as values.
  await Promise.resolve();

  return { status: 'success', message: 'Login link requested (demo).' };
}
