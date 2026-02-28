import React from 'react'

export default function AlertsList({ alerts, onDelete }) {
  if (!alerts || alerts.length === 0) {
    return <div className="no-data">No alerts configured</div>
  }

  return (
    <div>
      {alerts.map((a) => (
        <div key={a.id} className="alert-item">
          <div className="alert-info">
            <span className="alert-ticker">
              {a.ticker} <span className={`type-badge ${a.assetType}`}>{a.assetType}</span>
            </span>
            <span className="alert-condition">
              {a.direction === 'above' ? '>' : '<'} ${a.threshold.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
            </span>
            {a.triggered && (
              <span className="alert-triggered">
                TRIGGERED {a.triggeredAt ? new Date(a.triggeredAt).toLocaleString() : ''}
              </span>
            )}
          </div>
          <button className="delete-btn" onClick={() => onDelete(a.id)}>Remove</button>
        </div>
      ))}
    </div>
  )
}
