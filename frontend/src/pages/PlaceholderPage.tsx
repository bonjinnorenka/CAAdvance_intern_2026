import PageLayout from '../components/PageLayout.tsx'

type PlaceholderPageProps = {
  title: string
  description: string
}

export default function PlaceholderPage({ title, description }: PlaceholderPageProps) {
  return (
    <PageLayout title={title} description={description}>
      <p className="text-sm text-slate-600">この画面はこれから実装します。</p>
    </PageLayout>
  )
}
