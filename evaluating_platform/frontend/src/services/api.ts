import axios from 'axios'

const api = axios.create({
  baseURL: (import.meta as unknown as { env: { VITE_API_BASE?: string } }).env.VITE_API_BASE || '/api/v1',
  timeout: 30000,
})

// 请求拦截：自动注入 JWT
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// 响应拦截：统一错误处理，401 自动清除 token
api.interceptors.response.use(
  (res) => res,
  (error) => {
    if (error.response?.status === 401) {
      // 登录接口返回 401 是"密码错误"，不应清 token 或跳转
      const url = error.config?.url || ''
      if (!url.includes('/auth/login')) {
        localStorage.removeItem('token')
        localStorage.removeItem('user')
        window.location.href = '/login'
      }
    }
    const msg = error.response?.data?.error || error.message || '请求失败'
    return Promise.reject(new Error(msg))
  },
)

export default api
