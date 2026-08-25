/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { render, screen } from '@testing-library/react'
import type { ReactNode } from 'react'
import { describe, expect, it, vi } from 'vitest'

import type { PricingModel } from '../../types'
import { usePricingColumns } from '../pricing-columns'

vi.mock('@/lib/lobe-icon', () => ({ getLobeIcon: () => null }))

const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
})

function model(modelName: string): PricingModel {
  return {
    id: 1,
    model_name: modelName,
    quota_type: 1,
    model_ratio: 1,
    completion_ratio: 1,
    model_price: 0.05,
    enable_groups: ['default'],
  }
}

function PriceCell(props: { model: PricingModel }) {
  const columns = usePricingColumns()
  const priceColumn = columns.find(
    (column) => 'accessorKey' in column && column.accessorKey === 'price'
  )
  if (!priceColumn || typeof priceColumn.cell !== 'function') {
    throw new Error('Price column cell is unavailable')
  }
  const renderCell = priceColumn.cell as (context: {
    row: { original: PricingModel }
  }) => ReactNode
  return <>{renderCell({ row: { original: props.model } })}</>
}

function renderPrice(modelName: string) {
  return render(
    <I18nextProvider i18n={i18n}>
      <PriceCell model={model(modelName)} />
    </I18nextProvider>
  )
}

describe('pricing table fixed-price units', () => {
  it('shows omni-flash per second without changing image-model units', () => {
    const view = renderPrice('omni-flash')
    expect(screen.getByText('/ second')).toBeInTheDocument()

    view.rerender(
      <I18nextProvider i18n={i18n}>
        <PriceCell model={model('gemini-3-pro-image')} />
      </I18nextProvider>
    )
    expect(screen.getByText('/ request')).toBeInTheDocument()
  })
})
