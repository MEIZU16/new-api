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

import type { PricingModel } from '../../types'
import { buildSupportedParameters } from '../mock-stats'

function model(
  modelName: string,
  supportedEndpointTypes: string[]
): PricingModel {
  return {
    id: 1,
    model_name: modelName,
    quota_type: 1,
    model_ratio: 1,
    completion_ratio: 1,
    enable_groups: ['default'],
    supported_endpoint_types: supportedEndpointTypes,
  }
}

const proAspectRatios = [
  '1:1',
  '9:16',
  '16:9',
  '3:4',
  '4:3',
  '3:2',
  '2:3',
  '5:4',
  '4:5',
  '21:9',
]
const flashAspectRatios = [...proAspectRatios, '4:1', '1:4', '8:1', '1:8']

describe('media API supported parameters', () => {
  it.each([
    ['gemini-3-pro-image', '1K', proAspectRatios],
    ['gemini-3-pro-image-2k', '2K', proAspectRatios],
    ['gemini-3.1-flash-image-4k', '4K', flashAspectRatios],
  ])(
    'shows the locked resolution and model-specific aspect ratios for %s',
    (modelName, resolution, aspectRatios) => {
      const parameters = buildSupportedParameters(
        model(modelName, ['image-generation'])
      )

      expect(parameters.map((parameter) => parameter.name)).toEqual([
        'prompt',
        'extra_fields.aspect_ratio',
      ])
      expect(parameters[1]?.type).toBe('enum')
      expect(parameters[1]?.enumValues).toEqual(aspectRatios)
      expect(parameters[1]?.descriptionKey).toContain(resolution)
      expect(parameters.some((parameter) => parameter.name === 'style')).toBe(
        false
      )
      expect(parameters.some((parameter) => parameter.name === 'quality')).toBe(
        false
      )
    }
  )

  it('uses the actual flow-omni video fields instead of generic video fields', () => {
    const parameters = buildSupportedParameters(
      model('flow-omni', ['openai-video'])
    )

    expect(parameters.map((parameter) => parameter.name)).toEqual([
      'prompt',
      'seconds',
      'size',
    ])
    expect(parameters[1]?.enumValues).toEqual(['4', '6', '8', '10'])
    expect(parameters[2]?.enumValues).toEqual(['1280x720', '720x1280'])
  })
})
