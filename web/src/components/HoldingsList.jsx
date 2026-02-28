import React from 'react'

export default function HoldingsList({ holdings, onDelete }) {
  if (!holdings || holdings.length === 0) {
    return <div className="no-data">No holdings yet. Add one above.</div>
  }

  return (
    <table className="holdings-table">
      <thead>
        <tr>
          <th>Ticker</th>
          <th>Type</th>
          <th>Qty</th>
          <th>Avg Cost</th>
          <th>Price</th>
          <th>Value</th>
          <th>P&L</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        {holdings.map((h) => (
          <tr key={h.id}>
            <td className="ticker-cell">{h.ticker}</td>
            <td>
              <span className={`type-badge ${h.assetType}`}>{h.assetType}</span>
            </td>
            <td>{h.quantity}</td>
            <td>${h.avgCost.toFixed(2)}</td>
            <td>${h.price.toFixed(2)}</td>
            <td>${h.marketValue.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}</td>
            <td className={h.pnl >= 0 ? 'positive' : 'negative'}>
              {h.pnl >= 0 ? '+' : ''}${h.pnl.toFixed(2)} ({h.pnlPct.toFixed(1)}%)
            </td>
            <td>
              <button className="delete-btn" onClick={() => onDelete(h.id)}>Remove</button>
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}
