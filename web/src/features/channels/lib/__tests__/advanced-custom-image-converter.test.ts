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

import {
  getAdvancedCustomConverterDefaults,
  getAdvancedCustomConverterOptions,
  getAdvancedCustomIncomingPathOptions,
  validateAdvancedCustomConfig,
} from '../advanced-custom'

const CONVERTER = 'openai_images_to_gemini_generate_content' as const

describe('Advanced Custom OpenAI Images to Gemini converter', () => {
  it('is offered only for OpenAI image generation and edit routes', () => {
    expect(
      getAdvancedCustomIncomingPathOptions(CONVERTER).map(
        (option) => option.value
      )
    ).toEqual(['/v1/images/generations', '/v1/images/edits'])

    expect(
      getAdvancedCustomConverterOptions('/v1/images/edits').map(
        (option) => option.value
      )
    ).toContain(CONVERTER)
    expect(
      getAdvancedCustomConverterOptions('/v1/chat/completions').map(
        (option) => option.value
      )
    ).not.toContain(CONVERTER)
  })

  it('targets Gemini generateContent by default', () => {
    expect(
      getAdvancedCustomConverterDefaults(CONVERTER, '/v1/images/generations')
    ).toEqual({
      upstream_path: '/v1beta/models/{model}:generateContent',
      auth: {
        type: 'query',
        name: 'key',
        value: '{api_key}',
      },
    })
  })

  it('validates both public OpenAI Images routes', () => {
    for (const incomingPath of ['/v1/images/generations', '/v1/images/edits']) {
      expect(
        validateAdvancedCustomConfig({
          advanced_routes: [
            {
              incoming_path: incomingPath,
              upstream_path: '/v1beta/models/{model}:generateContent',
              converter: CONVERTER,
            },
          ],
        })
      ).toBeNull()
    }
  })
})
