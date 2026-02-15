type ApiFetchInit = Omit<RequestInit, 'body'> & {
  body?: unknown;
};

function resolveBaseUrl(): string {
  const baseUrl = process.env.API_BASE_URL ?? process.env.NEXT_PUBLIC_API_BASE_URL;
  if (!baseUrl) {
    throw new Error(
      'API base URL is not configured. Set API_BASE_URL (server) and/or NEXT_PUBLIC_API_BASE_URL (client).',
    );
  }
  return baseUrl.replace(/\/$/, '');
}

export async function apiFetch<T>(
  url: string,
  init: ApiFetchInit = {},
): Promise<T> {
  const baseUrl = resolveBaseUrl();
  const targetUrl = url.startsWith('http') ? url : `${baseUrl}${url.startsWith('/') ? '' : '/'}${url}`;

  const headers = new Headers(init.headers);
  if (init.body != null && !headers.has('content-type')) {
    headers.set('content-type', 'application/json');
  }

  const res = await fetch(targetUrl, {
    ...init,
    headers,
    body: init.body == null ? undefined : JSON.stringify(init.body),
    credentials: 'include',
  });

  if (!res.ok) {
    const text = await res.text().catch(() => '');
    throw new Error(`API request failed: ${res.status} ${res.statusText}${text ? ` - ${text}` : ''}`);
  }

  // Orval fetch client expects JSON by default.
  return (await res.json()) as T;
}
