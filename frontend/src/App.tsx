import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { useToast } from './hooks/useToast'
import HomePage from './pages/HomePage'
import RoomPage from './pages/RoomPage'
import HistoryPage from './pages/HistoryPage'

export default function App() {
  const { toast, Toast } = useToast()

  return (
    <BrowserRouter>
      <div id="app">
        <Routes>
          <Route path="/" element={<HomePage toast={toast} />} />
          <Route path="/room/:code" element={<RoomPage toast={toast} />} />
          <Route path="/room/:code/history" element={<HistoryPage toast={toast} />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </div>
      <Toast />
    </BrowserRouter>
  )
}
