import { randomUUID } from 'node:crypto';
import { headers } from 'next/headers';
import { NextResponse } from 'next/server';

export async function GET() {
  const reqHeaders = await headers();
  const requestId = reqHeaders.get('x-request-id') ?? randomUUID();
  const traceId = reqHeaders.get('x-trace-id') ?? requestId;

  return NextResponse.json(
    {
      status: 'ok',
      service: 'web',
      request_id: requestId,
      trace_id: traceId,
      user_id: 'anonymous',
    },
    {
      headers: {
        'x-request-id': requestId,
        'x-trace-id': traceId,
      },
    },
  );
}
