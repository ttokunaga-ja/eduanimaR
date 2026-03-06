'use client';

import type { ReactNode } from 'react';

import { I18nProvider } from '@/shared/lib';

interface AppProvidersProps {
  children: ReactNode;
}

export function AppProviders({ children }: AppProvidersProps) {
  return <I18nProvider>{children}</I18nProvider>;
}
