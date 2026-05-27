import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'

import App from './App'
import './styles/app.css'

// Vite 入口：挂载 React 应用并启用 StrictMode，帮助开发期发现副作用问题。
createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>
)
