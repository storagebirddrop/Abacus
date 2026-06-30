import { createBrowserRouter, RouterProvider, Navigate } from 'react-router-dom'
import Layout from './components/Layout'
import { ToastProvider } from './components/Toast'
import { ConfirmProvider } from './components/ConfirmDialog'
import WalletsPage from './pages/WalletsPage'
import WalletPage from './pages/WalletPage'
import PricesPage from './pages/PricesPage'
import SettingsPage from './pages/SettingsPage'
import NotFound from './pages/NotFound'

const router = createBrowserRouter([
  {
    path: '/',
    element: <Layout />,
    children: [
      { index: true, element: <Navigate to="/wallets" replace /> },
      { path: 'wallets', element: <WalletsPage /> },
      { path: 'wallets/:id', element: <WalletPage /> },
      { path: 'prices', element: <PricesPage /> },
      { path: 'settings', element: <SettingsPage /> },
      { path: '*', element: <NotFound /> },
    ],
  },
])

export default function App() {
  return (
    <ToastProvider>
      <ConfirmProvider>
        <RouterProvider router={router} />
      </ConfirmProvider>
    </ToastProvider>
  )
}
