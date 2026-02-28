import React, { useState, useEffect, useRef, useCallback } from 'react'
import HoldingsList from './components/HoldingsList'
import AddHoldingForm from './components/AddHoldingForm'
import PortfolioPieChart from './components/PortfolioPieChart'
import AlertsList from './components/AlertsList'
import AddAlertForm from './components/AddAlertForm'

const API = '/api'

export default function App() {
  const [portfolio, setPortfolio] = useState(null)
  const [alerts, setAlerts] = useState([])
  const [wsConnected, setWsConnected] = useState(false)
  const wsRef = useRef(null)
  const reconnectTimer = useRef(null)

  const connectWs = useCallback(() => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) return

    const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws'
    const ws = new WebSocket(`${protocol}://${window.location.host}/ws`)
    wsRef.current = ws

    ws.onopen = () => setWsConnected(true)

    ws.onmessage = (e) => {
      try {
        const snapshot = JSON.parse(e.data)
        setPortfolio(snapshot)
        if (snapshot.alertsFired && snapshot.alertsFired.length > 0) {
          refreshAlerts()
        }
      } catch { /* ignore malformed */ }
    }

    ws.onclose = () => {
      setWsConnected(false)
      reconnectTimer.current = setTimeout(connectWs, 3000)
    }

    ws.onerror = () => ws.close()
  }, [])

  const refreshAlerts = async () => {
    try {
      const resp = await fetch(`${API}/alerts`)
      if (resp.ok) setAlerts(await resp.json())
    } catch { /* ignore */ }
  }

  const refreshPortfolio = async () => {
    try {
      const resp = await fetch(`${API}/portfolio`)
      if (resp.ok) setPortfolio(await resp.json())
    } catch { /* ignore */ }
  }

  useEffect(() => {
    refreshPortfolio()
    refreshAlerts()
    connectWs()
    return () => {
      clearTimeout(reconnectTimer.current)
      if (wsRef.current) wsRef.current.close()
    }
  }, [connectWs])

  const addHolding = async (holding) => {
    const resp = await fetch(`${API}/holdings`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(holding),
    })
    if (!resp.ok) throw new Error('Failed to add holding')
    refreshPortfolio()
  }

  const deleteHolding = async (id) => {
    await fetch(`${API}/holdings/${id}`, { method: 'DELETE' })
    refreshPortfolio()
  }

  const addAlert = async (alert) => {
    const resp = await fetch(`${API}/alerts`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(alert),
    })
    if (!resp.ok) throw new Error('Failed to add alert')
    refreshAlerts()
  }

  const deleteAlert = async (id) => {
    await fetch(`${API}/alerts/${id}`, { method: 'DELETE' })
    refreshAlerts()
  }

  const holdings = portfolio?.holdings || []
  const totalValue = portfolio?.totalValue || 0
  const totalCost = portfolio?.totalCost || 0
  const totalPnl = portfolio?.totalPnl || 0

  return (
    <>
      <div className="header">
        <div className="header-left">
          <h1>PortfolioPulse</h1>
          <h2>Real-time Portfolio Tracker</h2>
        </div>
        <div className="ws-status">
          <span className={`ws-dot ${wsConnected ? 'connected' : ''}`} />
          {wsConnected ? 'Live' : 'Disconnected'}
        </div>
      </div>

      <div className="summary-cards">
        <div className="summary-card">
          <div className="label">Portfolio Value</div>
          <div className="value">${totalValue.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}</div>
        </div>
        <div className="summary-card">
          <div className="label">Total Cost</div>
          <div className="value">${totalCost.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}</div>
        </div>
        <div className="summary-card">
          <div className="label">Total P&L</div>
          <div className={`value ${totalPnl >= 0 ? 'positive' : 'negative'}`}>
            {totalPnl >= 0 ? '+' : ''}${totalPnl.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
          </div>
        </div>
        <div className="summary-card">
          <div className="label">Holdings</div>
          <div className="value">{holdings.length}</div>
        </div>
      </div>

      <div className="main-grid">
        <div>
          <div className="panel">
            <h3>Add Holding</h3>
            <AddHoldingForm onAdd={addHolding} />
          </div>

          <div className="panel">
            <h3>Holdings</h3>
            <HoldingsList holdings={holdings} onDelete={deleteHolding} />
          </div>
        </div>

        <div>
          <div className="panel">
            <h3>Allocation</h3>
            <PortfolioPieChart holdings={holdings} />
          </div>

          <div className="panel">
            <h3>Add Alert</h3>
            <AddAlertForm onAdd={addAlert} />
          </div>

          <div className="panel">
            <h3>Price Alerts</h3>
            <AlertsList alerts={alerts} onDelete={deleteAlert} />
          </div>
        </div>
      </div>
    </>
  )
}
