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
  buildImageSample,
  buildMediaSample,
} from '../../lib/media-code-samples'

const IMAGE_CONTEXT = {
  baseUrl: 'https://api.example.com',
  apiKeyEnv: 'NEW_API_KEY',
  modelName: 'gemini-3-pro-image-4k',
  endpointType: 'image-generation',
  endpointPath: '/v1/images/generations',
}

const VIDEO_CONTEXT = {
  baseUrl: 'https://api.example.com',
  apiKeyEnv: 'NEW_API_KEY',
  modelName: 'flow-omni',
  endpointType: 'openai-video',
  endpointPath: '/v1/videos',
}

describe('media API code samples', () => {
  it.each(['curl', 'python', 'typescript', 'javascript'] as const)(
    'uses the independent aspect-ratio extension in the %s image sample',
    (language) => {
      const sample = buildImageSample(language, IMAGE_CONTEXT)

      expect(sample).toContain('extra_fields')
      expect(sample).toContain('aspect_ratio')
      expect(sample).toContain('16:9')
      expect(sample).toContain('b64_json')
      expect(sample).toContain('n')
      expect(sample).not.toContain('1024x1024')
      expect(sample).not.toContain('style')
    }
  )

  it.each(['curl', 'python', 'typescript', 'javascript'] as const)(
    'documents create, poll, and download in the %s video sample',
    (language) => {
      const sample = buildMediaSample(language, 'openai-video', VIDEO_CONTEXT)

      expect(sample).not.toBeNull()
      expect(sample).toContain('/v1/videos')
      expect(sample).toContain('seconds')
      expect(sample).toContain('1280x720')
      expect(sample).toContain('completed')
      expect(sample).toContain('/content')
      expect(sample).not.toContain('/v1/chat/completions')
    }
  )
})
