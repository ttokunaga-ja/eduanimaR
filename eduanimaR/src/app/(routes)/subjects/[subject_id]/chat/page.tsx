import type { Metadata } from 'next';

import { QAPage } from '@/views/qa';

interface PageProps {
  params: Promise<{ subject_id: string }>;
}

export const metadata: Metadata = {
  title: '資料に質問する | eduanimaR',
  description: 'アップロードした資料に対して AI に質問できます。',
};

/**
 * App Router ルート: /subjects/[subject_id]/chat
 *
 * Next.js 15 では params は Promise のため await が必要。
 */
export default async function SubjectChatPage({ params }: PageProps) {
  const { subject_id } = await params;

  return <QAPage subjectId={subject_id} />;
}
