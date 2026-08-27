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
import { describe, expect, it, vi } from 'vitest'

import type { PricingModel } from '../../types'
import { ModelDetailsApi } from '../model-details-api'

vi.mock('@/hooks/use-status', () => ({
  useStatus: () => ({ status: { server_address: 'https://api.example.com' } }),
}))

const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
  interpolation: { escapeValue: false },
})

const ENDPOINT_MAP = {
  'openai-video': { path: '/v1/videos', method: 'POST' },
}

function videoModel(modelName: string): PricingModel {
  return {
    id: 1,
    model_name: modelName,
    quota_type: 1,
    model_ratio: 1,
    completion_ratio: 1,
    enable_groups: ['default'],
    supported_endpoint_types: ['openai-video'],
  }
}

function renderApiTab(modelName: string) {
  return render(
    <I18nextProvider i18n={i18n}>
      <ModelDetailsApi
        model={videoModel(modelName)}
        endpointMap={ENDPOINT_MAP}
      />
    </I18nextProvider>
  )
}

describe('video model API documentation', () => {
  it('labels all three generation modes for a model that accepts reference images', () => {
    renderApiTab('omni-flash')

    expect(screen.getByText('Text-to-video request')).toBeInTheDocument()
    expect(screen.getByText('Image-to-video request')).toBeInTheDocument()
    expect(screen.getByText('Multi-reference video request')).toBeInTheDocument()
  })

  it('resolves the reference-image ceiling in the multi-reference description', () => {
    renderApiTab('omni-flash')

    expect(
      screen.getByText(/Repeat input_reference for up to 7 images/)
    ).toBeInTheDocument()
  })

  it('shows no mode sections for a video model without reference support', () => {
    renderApiTab('sora-2')

    expect(screen.queryByText('Text-to-video request')).not.toBeInTheDocument()
    expect(screen.queryByText('Image-to-video request')).not.toBeInTheDocument()
    expect(
      screen.queryByText('Multi-reference video request')
    ).not.toBeInTheDocument()
  })

  it('lists the reference-mode request fields only for the supporting model', () => {
    const view = renderApiTab('omni-flash')
    expect(screen.getByText('input_reference')).toBeInTheDocument()

    view.rerender(
      <I18nextProvider i18n={i18n}>
        <ModelDetailsApi model={videoModel('sora-2')} endpointMap={ENDPOINT_MAP} />
      </I18nextProvider>
    )
    expect(screen.queryByText('input_reference')).not.toBeInTheDocument()
  })
})
