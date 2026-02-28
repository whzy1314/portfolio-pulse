import React, { useState } from 'react'

export default function AddAlertForm({ onAdd }) {
  const [ticker, setTicker] = useState('')
  const [assetType, setAssetType] = useState('stock')
  const [direction, setDirection] = useState('above')
  const [threshold, setThreshold] = useState('')

  const handleSubmit = async (e) => {
    e.preventDefault()
    if (!ticker.trim() || !threshold) return
    try {
      await onAdd({
        ticker: ticker.trim().toUpperCase(),
        assetType,
        direction,
        threshold: parseFloat(threshold),
      })
      setTicker('')
      setThreshold('')
    } catch { /* handled by parent */ }
  }

  return (
    <form className="add-form" onSubmit={handleSubmit}>
      <div className="form-group">
        <label>Ticker</label>
        <input
          type="text"
          placeholder="BTC"
          value={ticker}
          onChange={(e) => setTicker(e.target.value)}
          required
        />
      </div>
      <div className="form-group">
        <label>Type</label>
        <select value={assetType} onChange={(e) => setAssetType(e.target.value)}>
          <option value="stock">Stock</option>
          <option value="crypto">Crypto</option>
        </select>
      </div>
      <div className="form-group">
        <label>Direction</label>
        <select value={direction} onChange={(e) => setDirection(e.target.value)}>
          <option value="above">Above</option>
          <option value="below">Below</option>
        </select>
      </div>
      <div className="form-group">
        <label>Price</label>
        <input
          type="number"
          step="any"
          min="0.01"
          placeholder="100000"
          value={threshold}
          onChange={(e) => setThreshold(e.target.value)}
          required
        />
      </div>
      <button className="submit-btn" type="submit">Add</button>
    </form>
  )
}
