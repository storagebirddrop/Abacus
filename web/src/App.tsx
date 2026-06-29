import { createBrowserRouter, RouterProvider, Navigate } from 'react-router-dom'
import Layout from './components/Layout'
import WalletsPage from './pages/WalletsPage'
import WalletPage from './pages/WalletPage'
import PricesPage from './pages/PricesPage'

const router = createBrowserRouter([
  {
    path: '/',
    element: <Layout />,
    children: [
      { index: true, element: <Navigate to="/wallets" replace /> },
      { path: 'wallets', element: <WalletsPage /> },
      { path: 'wallets/:id', element: <WalletPage /> },
      { path: 'prices', element: <PricesPage /> },
    ],
  },
])

export default function App() {
  return <RouterProvider router={router} />
}
