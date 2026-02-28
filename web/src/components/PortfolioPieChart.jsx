import React from 'react'
import { PieChart, Pie, Cell, Tooltip, ResponsiveContainer } from 'recharts'

const COLORS = ['#6366f1', '#22c55e', '#eab308', '#ef4444', '#06b6d4', '#f97316', '#a78bfa', '#ec4899', '#14b8a6', '#f43f5e']

export default function PortfolioPieChart({ holdings }) {
  const data = (holdings || [])
    .filter((h) => h.marketValue > 0)
    .map((h) => ({ name: h.ticker, value: h.marketValue }))

  if (data.length === 0) {
    return <div className="no-data">Add holdings to see allocation</div>
  }

  return (
    <div className="chart-container">
      <ResponsiveContainer width="100%" height={250}>
        <PieChart>
          <Pie
            data={data}
            cx="50%"
            cy="50%"
            innerRadius={60}
            outerRadius={100}
            paddingAngle={2}
            dataKey="value"
            label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
          >
            {data.map((_, i) => (
              <Cell key={i} fill={COLORS[i % COLORS.length]} />
            ))}
          </Pie>
          <Tooltip
            formatter={(value) => `$${value.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`}
          />
        </PieChart>
      </ResponsiveContainer>
    </div>
  )
}
