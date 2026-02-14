import { LoginForm } from '@/features/auth-by-email';

export function AppHeader() {
  return (
    <header>
      <strong>App</strong>
      <LoginForm />
    </header>
  );
}
