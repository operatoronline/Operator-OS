import { useState } from 'react'
import { useNavigate } from 'react-router-dom'

export function LoginPage() {
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    // Auth flow will be implemented in C3
    navigate('/chat')
  }

  return (
    <div className="h-full flex items-center justify-center bg-bg">
      <div className="w-full max-w-sm mx-4">
        <div className="text-center mb-8">
          <h1 className="text-2xl font-bold text-text tracking-tight">Operator OS</h1>
          <p className="text-sm text-text-secondary mt-1">Sign in to continue</p>
        </div>

        <form onSubmit={handleSubmit} className="flex flex-col gap-3">
          <input
            type="email"
            placeholder="Email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className="w-full px-4 py-3 bg-surface border border-border rounded-[var(--radius-sm)] text-text text-sm placeholder:text-text-dim outline-none focus:border-accent transition-colors"
          />
          <input
            type="password"
            placeholder="Password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="w-full px-4 py-3 bg-surface border border-border rounded-[var(--radius-sm)] text-text text-sm placeholder:text-text-dim outline-none focus:border-accent transition-colors"
          />
          <button
            type="submit"
            className="w-full py-3 bg-accent text-white text-sm font-semibold rounded-[var(--radius-sm)] hover:opacity-90 transition-opacity mt-2"
          >
            Sign In
          </button>
        </form>

        <p className="text-center text-xs text-text-dim mt-6">
          Don't have an account?{' '}
          <span className="text-accent-text cursor-pointer hover:underline">Register</span>
        </p>
      </div>
    </div>
  )
}
