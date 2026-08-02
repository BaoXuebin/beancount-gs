const navItems = [
  { label: '仪表盘', href: '#dashboard' },
  { label: '交易', href: '#transactions' },
  { label: '账户', href: '#accounts' },
  { label: '统计', href: '#stats' },
  { label: 'AI 助手', href: '#ai' },
]

function App() {
  return (
    <div className="min-h-screen">
      <header className="border-b border-zinc-200 bg-white/80 backdrop-blur dark:border-zinc-800 dark:bg-zinc-950/80">
        <div className="mx-auto flex max-w-5xl items-center justify-between px-4 py-3">
          <span className="font-medium">beancount-gs</span>
          <nav className="hidden gap-6 text-sm text-zinc-500 sm:flex dark:text-zinc-400">
            {navItems.map((item) => (
              <a key={item.label} href={item.href} className="hover:text-zinc-900 dark:hover:text-zinc-100">
                {item.label}
              </a>
            ))}
          </nav>
          <button
            type="button"
            className="rounded-lg bg-zinc-900 px-3 py-1.5 text-sm text-white hover:bg-zinc-700 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-300"
          >
            使用 GitHub 登录
          </button>
        </div>
      </header>

      <main className="mx-auto max-w-5xl px-4 py-12">
        <h1 className="text-2xl font-semibold">V2 前端脚手架已就绪</h1>
        <p className="mt-2 text-sm text-zinc-500 dark:text-zinc-400">
          Vite + React + TypeScript + Tailwind CSS v4，shadcn/ui 与 API 类型已预留接入点。
          页面结构参考
          <a className="mx-1 underline" href="../../prototype/index.html">线框原型</a>
          ，接口契约见
          <a className="underline" href="../../packages/contracts/openapi.yaml">openapi.yaml</a>。
        </p>

        <div className="mt-8 grid gap-4 sm:grid-cols-3">
          {[
            { title: '账本', desc: '多账本管理，修订号并发控制' },
            { title: '交易', desc: 'narration / postings / cost 规范建模' },
            { title: 'AI 与 MCP', desc: '自然语言记账与 Agent 接入' },
          ].map((card) => (
            <div
              key={card.title}
              className="rounded-xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900"
            >
              <h2 className="font-medium">{card.title}</h2>
              <p className="mt-1 text-sm text-zinc-500 dark:text-zinc-400">{card.desc}</p>
            </div>
          ))}
        </div>
      </main>
    </div>
  )
}

export default App
