import { Link } from 'react-router-dom'

export default function NotFound() {
  return (
    <div className="flex flex-col items-center justify-center h-full p-8 text-center">
      <p className="text-5xl font-bold text-slate-300">404</p>
      <h1 className="text-xl font-semibold text-slate-800 dark:text-slate-100 mt-4">Page not found</h1>
      <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
        The page you're looking for doesn't exist.
      </p>
      <Link
        to="/wallets"
        className="mt-6 text-sm font-medium text-slate-900 dark:text-slate-100 underline hover:no-underline"
      >
        Back to Wallets
      </Link>
    </div>
  )
}
