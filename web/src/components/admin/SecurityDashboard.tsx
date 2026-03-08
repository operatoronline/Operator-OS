// ============================================================================
// Operator OS — Security Dashboard
// Visual security audit UI: risk gauge, findings by category, remediation.
// ============================================================================

import { useMemo, memo, useCallback } from 'react'
import {
  ShieldWarning,
  ShieldCheck,

  ArrowClockwise,
  CaretDown,
  CaretRight,
  Warning,
  CheckCircle,
  XCircle,
  Info,
  Link as LinkIcon,
  MapPin,
  Wrench,
  Timer,
  Funnel,
} from '@phosphor-icons/react'
import { useSecurityAuditStore } from '../../stores/securityAuditStore'
import { Badge } from '../shared/Badge'
import { Button } from '../shared/Button'
import type {
  SecurityAuditReport,
  SecurityFinding,
  AuditCategory,
  AuditSeverity,
  CategoryStats,
} from '../../types/api'

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const SEVERITY_ORDER: AuditSeverity[] = ['critical', 'high', 'medium', 'low', 'info']

const SEVERITY_CONFIG: Record<AuditSeverity, {
  label: string
  color: string
  bgColor: string
  borderColor: string
  badgeVariant: 'error' | 'warning' | 'accent' | 'default' | 'success'
}> = {
  critical: {
    label: 'Critical',
    color: 'var(--error)',
    bgColor: 'var(--error-subtle)',
    borderColor: 'var(--error)',
    badgeVariant: 'error',
  },
  high: {
    label: 'High',
    color: 'oklch(0.65 0.2 25)',
    bgColor: 'oklch(0.65 0.2 25 / 0.1)',
    borderColor: 'oklch(0.65 0.2 25 / 0.3)',
    badgeVariant: 'warning',
  },
  medium: {
    label: 'Medium',
    color: 'var(--warning)',
    bgColor: 'var(--warning-subtle)',
    borderColor: 'var(--warning)',
    badgeVariant: 'warning',
  },
  low: {
    label: 'Low',
    color: 'var(--accent-text)',
    bgColor: 'var(--accent-subtle)',
    borderColor: 'var(--accent)',
    badgeVariant: 'accent',
  },
  info: {
    label: 'Info',
    color: 'var(--text-dim)',
    bgColor: 'var(--surface-2)',
    borderColor: 'var(--border-subtle)',
    badgeVariant: 'default',
  },
}

const CATEGORY_LABELS: Record<AuditCategory, string> = {
  authentication: 'Authentication',
  authorization: 'Authorization',
  input_validation: 'Input Validation',
  cryptography: 'Cryptography',
  session_management: 'Session Mgmt',
  api_security: 'API Security',
  configuration: 'Configuration',
  data_protection: 'Data Protection',
  rate_limiting: 'Rate Limiting',
  security_headers: 'Security Headers',
  injection: 'Injection',
  compliance: 'Compliance',
}

// ---------------------------------------------------------------------------
// Risk Gauge
// ---------------------------------------------------------------------------

function RiskGauge({ score, passRate }: { score: number; passRate: number }) {
  const circumference = 2 * Math.PI * 54 // r=54
  const progress = (score / 100) * circumference

  // Color based on score: 0-25 green, 25-50 accent, 50-75 warning, 75-100 critical
  const gaugeColor =
    score <= 25
      ? 'var(--success)'
      : score <= 50
        ? 'var(--accent)'
        : score <= 75
          ? 'var(--warning)'
          : 'var(--error)'

  const riskLabel =
    score <= 25 ? 'Low Risk' : score <= 50 ? 'Moderate' : score <= 75 ? 'High Risk' : 'Critical'

  return (
    <div className="flex flex-col items-center gap-3">
      {/* SVG Gauge */}
      <div className="relative w-32 h-32">
        <svg className="w-full h-full -rotate-90" viewBox="0 0 120 120">
          {/* Background track */}
          <circle
            cx="60"
            cy="60"
            r="54"
            fill="none"
            stroke="var(--surface-2)"
            strokeWidth="8"
            strokeLinecap="round"
          />
          {/* Progress arc */}
          <circle
            cx="60"
            cy="60"
            r="54"
            fill="none"
            stroke={gaugeColor}
            strokeWidth="8"
            strokeLinecap="round"
            strokeDasharray={circumference}
            strokeDashoffset={circumference - progress}
            className="transition-all duration-700 ease-out"
          />
        </svg>
        {/* Center text */}
        <div className="absolute inset-0 flex flex-col items-center justify-center">
          <span
            className="text-2xl font-bold tabular-nums"
            style={{ color: gaugeColor }}
          >
            {score.toFixed(0)}
          </span>
          <span className="text-[10px] text-[var(--text-dim)] font-medium">/ 100</span>
        </div>
      </div>

      <div className="text-center">
        <p className="text-sm font-semibold" style={{ color: gaugeColor }}>
          {riskLabel}
        </p>
        <p className="text-xs text-[var(--text-dim)] mt-0.5">
          {passRate.toFixed(1)}% checks passed
        </p>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Summary Cards
// ---------------------------------------------------------------------------

function SeveritySummary({ summary }: { summary: SecurityAuditReport['summary'] }) {
  const counts: { severity: AuditSeverity; count: number }[] = [
    { severity: 'critical', count: summary.critical },
    { severity: 'high', count: summary.high },
    { severity: 'medium', count: summary.medium },
    { severity: 'low', count: summary.low },
    { severity: 'info', count: summary.info },
  ]

  return (
    <div className="grid grid-cols-5 gap-2">
      {counts.map(({ severity, count }) => {
        const cfg = SEVERITY_CONFIG[severity]
        return (
          <div
            key={severity}
            className="flex flex-col items-center gap-1 px-2 py-2.5
              rounded-[var(--radius-md)] border transition-colors"
            style={{
              borderColor: count > 0 ? cfg.borderColor : 'var(--border-subtle)',
              backgroundColor: count > 0 ? cfg.bgColor : 'transparent',
            }}
          >
            <span
              className="text-lg font-bold tabular-nums"
              style={{ color: count > 0 ? cfg.color : 'var(--text-dim)' }}
            >
              {count}
            </span>
            <span className="text-[10px] font-medium text-[var(--text-dim)] uppercase tracking-wider">
              {cfg.label}
            </span>
          </div>
        )
      })}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Category Breakdown
// ---------------------------------------------------------------------------

function CategoryBreakdown({
  categories,
  activeCategory,
  onSelect,
}: {
  categories: Record<string, CategoryStats>
  activeCategory: AuditCategory | 'all'
  onSelect: (cat: AuditCategory | 'all') => void
}) {
  const sortedCats = useMemo(() => {
    return Object.entries(categories)
      .sort(([, a], [, b]) => b.failed - a.failed || b.total - a.total)
  }, [categories])

  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between mb-2">
        <h3 className="text-xs font-semibold text-[var(--text-secondary)] uppercase tracking-wider">
          By Category
        </h3>
        {activeCategory !== 'all' && (
          <button
            onClick={() => onSelect('all')}
            className="text-[10px] text-[var(--accent-text)] hover:underline cursor-pointer"
          >
            Clear filter
          </button>
        )}
      </div>

      {sortedCats.map(([cat, stats]) => {
        const category = cat as AuditCategory
        const passRate = stats.total > 0 ? (stats.passed / stats.total) * 100 : 100
        const isActive = activeCategory === category

        return (
          <button
            key={cat}
            onClick={() => onSelect(isActive ? 'all' : category)}
            className={`
              w-full flex items-center gap-3 px-3 py-2
              rounded-[var(--radius-md)] text-left
              transition-all duration-150 cursor-pointer
              ${isActive
                ? 'bg-[var(--accent-subtle)] border border-[var(--accent)]/30'
                : 'hover:bg-[var(--surface-2)] border border-transparent'
              }
            `}
          >
            <div className="flex-1 min-w-0">
              <p className="text-xs font-medium text-[var(--text)] truncate">
                {CATEGORY_LABELS[category] || cat}
              </p>
              <div className="mt-1 h-1.5 rounded-full bg-[var(--surface-2)] overflow-hidden">
                <div
                  className="h-full rounded-full transition-all duration-500 ease-out"
                  style={{
                    width: `${passRate}%`,
                    backgroundColor:
                      stats.failed === 0
                        ? 'var(--success)'
                        : passRate >= 75
                          ? 'var(--warning)'
                          : 'var(--error)',
                  }}
                />
              </div>
            </div>
            <div className="flex items-center gap-2 text-[10px] tabular-nums shrink-0">
              {stats.failed > 0 && (
                <span className="text-[var(--error)] font-semibold">{stats.failed} ✗</span>
              )}
              <span className="text-[var(--text-dim)]">
                {stats.passed}/{stats.total}
              </span>
            </div>
          </button>
        )
      })}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Finding Row
// ---------------------------------------------------------------------------

const FindingRow = memo(function FindingRow({
  finding,
  expanded,
  onToggle,
}: {
  finding: SecurityFinding
  expanded: boolean
  onToggle: () => void
}) {
  const cfg = SEVERITY_CONFIG[finding.severity]

  return (
    <div
      className={`
        border rounded-[var(--radius-md)] overflow-hidden
        transition-colors duration-150
        ${finding.passed
          ? 'border-[var(--border-subtle)]'
          : `border-l-[3px]`
        }
      `}
      style={{
        borderLeftColor: finding.passed ? undefined : cfg.color,
      }}
    >
      {/* Header */}
      <button
        onClick={onToggle}
        className="w-full flex items-center gap-3 px-3.5 py-2.5
          hover:bg-[var(--surface-2)] transition-colors cursor-pointer text-left"
      >
        {/* Status icon */}
        {finding.passed ? (
          <CheckCircle
            size={16}
            weight="fill"
            className="shrink-0 text-[var(--success)]"
          />
        ) : (
          <XCircle
            size={16}
            weight="fill"
            className="shrink-0"
            style={{ color: cfg.color }}
          />
        )}

        {/* Title + ID */}
        <div className="flex-1 min-w-0">
          <p className="text-xs font-medium text-[var(--text)] truncate">
            {finding.title}
          </p>
          <p className="text-[10px] text-[var(--text-dim)] mt-0.5 font-mono">
            {finding.id}
          </p>
        </div>

        {/* Category badge */}
        <Badge variant="default" className="hidden sm:inline-flex shrink-0">
          {CATEGORY_LABELS[finding.category] || finding.category}
        </Badge>

        {/* Severity badge */}
        <Badge variant={cfg.badgeVariant} dot className="shrink-0">
          {cfg.label}
        </Badge>

        {/* Expand chevron */}
        {expanded ? (
          <CaretDown size={14} className="shrink-0 text-[var(--text-dim)]" />
        ) : (
          <CaretRight size={14} className="shrink-0 text-[var(--text-dim)]" />
        )}
      </button>

      {/* Expanded detail */}
      {expanded && (
        <div
          className="px-4 py-3 border-t border-[var(--border-subtle)]
            bg-[var(--surface-2)]/50 space-y-3 animate-fadeSlideDown"
        >
          {/* Description */}
          {finding.description && (
            <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
              {finding.description}
            </p>
          )}

          {/* Location */}
          {finding.location && (
            <div className="flex items-start gap-2">
              <MapPin size={13} className="shrink-0 mt-0.5 text-[var(--text-dim)]" />
              <div>
                <span className="text-[10px] font-semibold text-[var(--text-dim)] uppercase tracking-wider">
                  Location
                </span>
                <p className="text-xs text-[var(--text-secondary)] font-mono mt-0.5">
                  {finding.location}
                </p>
              </div>
            </div>
          )}

          {/* Evidence */}
          {finding.evidence && (
            <div className="flex items-start gap-2">
              <Info size={13} className="shrink-0 mt-0.5 text-[var(--text-dim)]" />
              <div>
                <span className="text-[10px] font-semibold text-[var(--text-dim)] uppercase tracking-wider">
                  Evidence
                </span>
                <pre className="text-xs text-[var(--text-secondary)] font-mono mt-0.5
                  bg-[var(--surface)] px-2.5 py-1.5 rounded-[var(--radius-sm)]
                  overflow-x-auto whitespace-pre-wrap break-all">
                  {finding.evidence}
                </pre>
              </div>
            </div>
          )}

          {/* Remediation */}
          {finding.remediation && (
            <div className="flex items-start gap-2">
              <Wrench size={13} className="shrink-0 mt-0.5 text-[var(--accent-text)]" />
              <div>
                <span className="text-[10px] font-semibold text-[var(--accent-text)] uppercase tracking-wider">
                  Remediation
                </span>
                <p className="text-xs text-[var(--text)] leading-relaxed mt-0.5">
                  {finding.remediation}
                </p>
              </div>
            </div>
          )}

          {/* References */}
          {finding.references && finding.references.length > 0 && (
            <div className="flex items-start gap-2">
              <LinkIcon size={13} className="shrink-0 mt-0.5 text-[var(--text-dim)]" />
              <div>
                <span className="text-[10px] font-semibold text-[var(--text-dim)] uppercase tracking-wider">
                  References
                </span>
                <div className="flex flex-wrap gap-1.5 mt-1">
                  {finding.references.map((ref, i) => (
                    <a
                      key={i}
                      href={ref.startsWith('http') ? ref : `https://cwe.mitre.org/data/definitions/${ref.replace(/\D/g, '')}.html`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-[10px] font-mono px-2 py-0.5
                        rounded-full bg-[var(--surface)] border border-[var(--border-subtle)]
                        text-[var(--accent-text)] hover:border-[var(--accent)]
                        transition-colors"
                    >
                      {ref}
                    </a>
                  ))}
                </div>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
})

// ---------------------------------------------------------------------------
// Loading Skeleton
// ---------------------------------------------------------------------------

function AuditSkeleton() {
  return (
    <div className="space-y-6 animate-pulse">
      {/* Gauge skeleton */}
      <div className="flex justify-center">
        <div className="w-32 h-32 rounded-full bg-[var(--surface-2)]" />
      </div>
      {/* Severity cards */}
      <div className="grid grid-cols-5 gap-2">
        {Array.from({ length: 5 }).map((_, i) => (
          <div key={i} className="h-16 rounded-[var(--radius-md)] bg-[var(--surface-2)]" />
        ))}
      </div>
      {/* Finding rows */}
      {Array.from({ length: 6 }).map((_, i) => (
        <div key={i} className="h-12 rounded-[var(--radius-md)] bg-[var(--surface-2)]" />
      ))}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Empty State
// ---------------------------------------------------------------------------

function EmptyState({ onRun, loading }: { onRun: () => void; loading: boolean }) {
  return (
    <div className="h-full flex flex-col items-center justify-center text-center px-4 py-12">
      <ShieldCheck
        size={56}
        weight="thin"
        className="text-[var(--text-dim)] mb-4"
      />
      <h3 className="text-base font-semibold text-[var(--text)] mb-1">
        Security Audit
      </h3>
      <p className="text-sm text-[var(--text-dim)] max-w-sm mb-5">
        Run a comprehensive security audit to check authentication, authorization,
        cryptography, API security, and more against OWASP standards.
      </p>
      <Button onClick={onRun} loading={loading} icon={<ShieldWarning size={16} />}>
        Run Security Audit
      </Button>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Filter Bar
// ---------------------------------------------------------------------------

function FilterBar({
  severityFilter,
  statusFilter,
  onSeverityChange,
  onStatusChange,
  totalFindings,
  filteredCount,
}: {
  severityFilter: AuditSeverity | 'all'
  statusFilter: 'all' | 'passed' | 'failed'
  onSeverityChange: (s: AuditSeverity | 'all') => void
  onStatusChange: (s: 'all' | 'passed' | 'failed') => void
  totalFindings: number
  filteredCount: number
}) {
  return (
    <div className="flex items-center gap-3 flex-wrap">
      {/* Status pills */}
      <div className="flex items-center gap-1 bg-[var(--surface-2)] rounded-full p-0.5">
        {(['all', 'failed', 'passed'] as const).map((status) => (
          <button
            key={status}
            onClick={() => onStatusChange(status)}
            className={`
              px-3 py-1.5 rounded-full text-xs font-medium
              transition-all duration-150 cursor-pointer capitalize
              ${statusFilter === status
                ? 'bg-[var(--surface)] text-[var(--text)] shadow-sm'
                : 'text-[var(--text-dim)] hover:text-[var(--text-secondary)]'
              }
            `}
          >
            {status === 'all' ? 'All' : status === 'failed' ? '✗ Failed' : '✓ Passed'}
          </button>
        ))}
      </div>

      {/* Severity pills */}
      <div className="flex items-center gap-1 bg-[var(--surface-2)] rounded-full p-0.5">
        <button
          onClick={() => onSeverityChange('all')}
          className={`
            px-2.5 py-1.5 rounded-full text-xs font-medium
            transition-all duration-150 cursor-pointer
            ${severityFilter === 'all'
              ? 'bg-[var(--surface)] text-[var(--text)] shadow-sm'
              : 'text-[var(--text-dim)] hover:text-[var(--text-secondary)]'
            }
          `}
        >
          All
        </button>
        {SEVERITY_ORDER.map((sev) => {
          const cfg = SEVERITY_CONFIG[sev]
          return (
            <button
              key={sev}
              onClick={() => onSeverityChange(sev)}
              className={`
                px-2.5 py-1.5 rounded-full text-xs font-medium
                transition-all duration-150 cursor-pointer
                ${severityFilter === sev
                  ? 'bg-[var(--surface)] text-[var(--text)] shadow-sm'
                  : 'text-[var(--text-dim)] hover:text-[var(--text-secondary)]'
                }
              `}
            >
              {cfg.label}
            </button>
          )
        })}
      </div>

      {/* Count */}
      {filteredCount !== totalFindings && (
        <span className="text-[10px] text-[var(--text-dim)] flex items-center gap-1">
          <Funnel size={11} />
          {filteredCount} / {totalFindings}
        </span>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Main Dashboard
// ---------------------------------------------------------------------------

export function SecurityDashboard() {
  const store = useSecurityAuditStore()
  const filteredFindings = store.filteredFindings()

  const handleRun = useCallback(() => {
    store.runAudit()
  }, [store])

  const handleToggle = useCallback(
    (id: string) => store.toggleFinding(id),
    [store],
  )

  // No report yet — show empty state
  if (!store.report && !store.loading) {
    return <EmptyState onRun={handleRun} loading={false} />
  }

  // Loading
  if (store.loading && !store.report) {
    return (
      <div className="px-1 py-2">
        <AuditSkeleton />
      </div>
    )
  }

  const report = store.report!
  const durationMs = report.duration / 1_000_000 // nanoseconds → ms

  return (
    <div className="space-y-5">
      {/* Error banner */}
      {store.error && (
        <div
          className="flex items-center gap-3 px-4 py-3
            bg-[var(--error-subtle)] border border-[var(--error)]/20
            rounded-[var(--radius-md)] text-sm text-[var(--error)]"
        >
          <Warning size={16} />
          <span className="flex-1">{store.error}</span>
          <Button variant="ghost" size="sm" onClick={store.clearError}>
            Dismiss
          </Button>
        </div>
      )}

      {/* Top section: Gauge + Summary + Meta */}
      <div className="grid grid-cols-1 md:grid-cols-[auto_1fr] gap-6 items-start">
        {/* Gauge */}
        <div className="flex justify-center md:justify-start">
          <RiskGauge score={report.risk_score} passRate={report.pass_rate} />
        </div>

        {/* Right: severity cards + run info */}
        <div className="space-y-4">
          <SeveritySummary summary={report.summary} />

          {/* Run meta */}
          <div className="flex items-center gap-4 flex-wrap text-[10px] text-[var(--text-dim)]">
            <span className="flex items-center gap-1">
              <Timer size={11} />
              {durationMs.toFixed(0)}ms · {report.checks_run} checks
            </span>
            <span>
              {report.summary.passed} passed · {report.summary.failed} failed
            </span>
            {store.lastRunAt && (
              <span>
                Last run: {new Date(store.lastRunAt).toLocaleTimeString()}
              </span>
            )}
            <Button
              variant="ghost"
              size="sm"
              onClick={handleRun}
              loading={store.loading}
              icon={<ArrowClockwise size={13} />}
            >
              Re-run
            </Button>
          </div>
        </div>
      </div>

      {/* Main content: Categories sidebar + Findings list */}
      <div className="grid grid-cols-1 lg:grid-cols-[240px_1fr] gap-5">
        {/* Category sidebar */}
        <div className="lg:sticky lg:top-0">
          <CategoryBreakdown
            categories={report.categories}
            activeCategory={store.categoryFilter}
            onSelect={store.setCategoryFilter}
          />
        </div>

        {/* Findings */}
        <div className="space-y-3">
          {/* Filter bar */}
          <FilterBar
            severityFilter={store.severityFilter}
            statusFilter={store.statusFilter}
            onSeverityChange={store.setSeverityFilter}
            onStatusChange={store.setStatusFilter}
            totalFindings={report.findings.length}
            filteredCount={filteredFindings.length}
          />

          {/* Finding rows */}
          {filteredFindings.length === 0 ? (
            <div className="text-center py-8 text-sm text-[var(--text-dim)]">
              No findings match the current filters.
            </div>
          ) : (
            <div className="space-y-2">
              {filteredFindings.map((f) => (
                <FindingRow
                  key={f.id}
                  finding={f}
                  expanded={store.expandedFindingId === f.id}
                  onToggle={() => handleToggle(f.id)}
                />
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
