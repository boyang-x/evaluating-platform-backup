import React from 'react'
import ReactDOM from 'react-dom/client'
import { ConfigProvider, App as AntApp, theme } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import App from './App.tsx'
import './styles/theme.css'

const cobaltBlue = '#1a6dff'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <ConfigProvider
      locale={zhCN}
      theme={{
        algorithm: theme.darkAlgorithm,
        token: {
          colorPrimary: cobaltBlue,
          colorBgBase: '#0a0c10',
          colorBgContainer: '#141820',
          colorBgElevated: '#1a2030',
          colorBorder: '#1e2535',
          colorText: '#e8eaf0',
          colorTextSecondary: '#8890a4',
          borderRadius: 6,
          fontFamily: "'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif",
        },
        components: {
          Button: {
            colorPrimary: cobaltBlue,
            colorPrimaryHover: '#3d85ff',
            colorPrimaryActive: '#0052d9',
          },
          Layout: {
            siderBg: '#0f1117',
            headerBg: '#0f1117',
            bodyBg: '#0a0c10',
          },
          Menu: {
            darkItemBg: '#0f1117',
            darkSubMenuItemBg: '#0a0c10',
            darkItemSelectedBg: 'rgba(26, 109, 255, 0.2)',
            darkItemSelectedColor: '#4d96ff',
          },
          Table: {
            headerBg: '#141820',
            rowHoverBg: '#1a2030',
          },
          Card: {
            colorBgContainer: '#141820',
          },
          Input: {
            colorBgContainer: '#0f1117',
          },
          Select: {
            colorBgContainer: '#0f1117',
          },
        },
      }}
    >
      <AntApp>
        <App />
      </AntApp>
    </ConfigProvider>
  </React.StrictMode>,
)
