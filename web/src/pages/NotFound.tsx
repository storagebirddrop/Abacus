import { Link } from 'react-router-dom'

export default function NotFound() {
  return (
    <div className="flex flex-col items-center justify-center h-full p-8 text-center">
      <p className="text-5xl font-bold text-muted-foreground">404</p>
      <h1 className="text-xl font-semibold text-foreground mt-4">Page not found</h1>
      <p className="text-sm text-muted-foreground mt-1">
        The page you're looking for doesn't exist.
      </p>
      <Link
        to="/wallets"
        className="mt-6 text-sm font-medium text-foreground underline hover:no-underline"
      >
        Back to Wallets
      </Link>
    </div>
  )
}
