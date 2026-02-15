'use client';

import { useActionState } from 'react';

import { UserCard } from '@/entities/user';
import { Button } from '@/shared/ui';

import { requestLoginLink, type RequestLoginLinkState } from '../model/actions';

const initialState: RequestLoginLinkState = { status: 'idle' };

export function LoginForm() {
  const [state, formAction, isPending] = useActionState(requestLoginLink, initialState);

  return (
    <section>
      <UserCard user={{ id: 'demo', name: 'Demo User' }} />

      <form action={formAction}>
        <label>
          Email
          <input name="email" type="email" autoComplete="email" required />
        </label>

        <Button type="submit" disabled={isPending}>
          {isPending ? 'Sending…' : 'Send login link'}
        </Button>

        {state.status !== 'idle' ? <p aria-live="polite">{state.message}</p> : null}
      </form>
    </section>
  );
}
