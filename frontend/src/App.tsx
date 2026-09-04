import { BrowserRouter, Route, Routes } from 'react-router-dom'
import AccountManageScreen from './pages/AccountManageScreen.tsx'
import AdminIndex from './pages/AdminIndex.tsx'
import CreateReport from './pages/CreateReport.tsx'
import Dashboard from './pages/Dashboard.tsx'
import Menu from './pages/Menu.tsx'
import UserRecord from './pages/UserRecord.tsx'

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Menu />} />
        <Route path="/create-report" element={<CreateReport />} />
        <Route path="/history" element={<UserRecord />} />
        <Route path="/admin" element={<AdminIndex />} />
        <Route path="/admin/users/:userId" element={<AccountManageScreen />} />
        <Route path="/dashboard" element={<Dashboard />} />
      </Routes>
    </BrowserRouter>
  )
}
