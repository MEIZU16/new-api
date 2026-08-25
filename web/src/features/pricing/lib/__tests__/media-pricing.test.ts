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
import { describe, expect, it } from 'vitest'

import { QUOTA_TYPES } from '../../constants'
import type { PricingModel } from '../../types'
import { filterByQuotaType } from '../filters'
import { isPerSecondMediaModel } from '../media-docs'

function model(modelName: string, quotaType: number): PricingModel {
  return {
    id: 1,
    model_name: modelName,
    quota_type: quotaType,
    model_ratio: 1,
    completion_ratio: 1,
    enable_groups: ['default'],
  }
}

const models = [
  model('chat-model', 0),
  model('gemini-3-pro-image', 1),
  model('flow-omni', 1),
]

describe('media pricing cadence', () => {
  it('marks only the public flow-omni SKU as per-second', () => {
    expect(isPerSecondMediaModel('flow-omni')).toBe(true)
    expect(isPerSecondMediaModel('FLOW-OMNI')).toBe(true)
    expect(isPerSecondMediaModel('gemini-3-pro-image')).toBe(false)
  })

  it('separates per-second media from ordinary fixed-price models', () => {
    expect(
      filterByQuotaType(models, QUOTA_TYPES.REQUEST).map(
        (item) => item.model_name
      )
    ).toEqual(['gemini-3-pro-image'])
    expect(
      filterByQuotaType(models, QUOTA_TYPES.SECOND).map(
        (item) => item.model_name
      )
    ).toEqual(['flow-omni'])
  })
})
