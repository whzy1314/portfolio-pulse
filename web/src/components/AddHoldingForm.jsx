import React, { useState } from 'react'

export default function AddHoldingForm({ onAdd }) {
  const [ticker, setTicker] = useState('')
  const [assetType, setAssetType] = useState('stock')
  const [quantity, setQuantity] = useState('')
  const [avgCost, setAvgCost] = useState('')

  const handleSubmit = async (e) => {
    e.preventDefault()
    if (!ticker.trim() || !quantity || !avgCost) return
    try {
      await onAdd({
        ticker: ticker.trim().toUpperCase(),
        assetType,
        quantity: parseFloat(quantity),
        avgCost: parseFloat(avgCost),
      })
      setTicker('')
      setQuantity('')
      setAvgCost('')
    } catch { /* handled by parent */ }
  }

  return (
    <form className="add-form" onSubmit={handleSubmit}>
      <div className="form-group">
        <label>Ticker</label>
        <input
          type="text"
          placeholder="AAPL"
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
        <label>Quantity</label>
        <input
          type="number"
          step="any"
          min="0.0001"
          placeholder="10"
          value={quantity}
          onChange={(e) => setQuantity(e.target.value)}
          required
        />
      </div>
      <div className="form-group">
        <label>Avg Cost</label>
        <input
          type="number"
          step="any"
          min="0"
          placeholder="150.00"
          value={avgCost}
          onChange={(e) => setAvgCost(e.target.value)}
          required
        />
      </div>
      <button className="submit-btn" type="submit">Add</button>
    </form>
  )
}
