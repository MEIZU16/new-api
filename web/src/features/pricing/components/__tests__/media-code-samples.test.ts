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
  buildImageEditSample,
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

const PROMPT_ONLY_IMAGE_CONTEXT = {
  ...IMAGE_CONTEXT,
  modelName: 'gpt-image-2',
}

const VIDEO_CONTEXT = {
  baseUrl: 'https://api.example.com',
  apiKeyEnv: 'NEW_API_KEY',
  modelName: 'omni-flash',
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
      expect(sample).not.toContain('"n":')
      expect(sample).not.toContain('n=1')
      expect(sample).not.toContain('response_format')
      expect(sample).not.toContain('1024x1024')
      expect(sample).not.toContain('style')
      if (language !== 'curl') {
        expect(sample).toContain('b64_json')
      }
    }
  )

  it.each(['curl', 'python', 'typescript', 'javascript'] as const)(
    'shows only model and prompt in the %s gpt-image-2 request',
    (language) => {
      const sample = buildImageSample(language, PROMPT_ONLY_IMAGE_CONTEXT)

      expect(sample).toContain('gpt-image-2')
      expect(sample).toContain('prompt')
      if (language !== 'curl') {
        expect(sample).toContain('generated-image.png')
      }
      expect(sample).not.toContain('extra_fields')
      expect(sample).not.toContain('aspect_ratio')
      expect(sample).not.toContain('response_format')
      expect(sample).not.toContain('background')
      expect(sample).not.toContain('output_format')
      expect(sample).not.toContain('quality')
      expect(sample).not.toContain('size')
      expect(sample).not.toContain('n=')
      expect(sample).not.toContain('"n"')
    }
  )

  it.each(['curl', 'python', 'typescript', 'javascript'] as const)(
    'documents multipart reference-image uploads in the %s image sample',
    (language) => {
      const sample = buildImageEditSample(language, IMAGE_CONTEXT)

      expect(sample).toContain('/v1/images/edits')
      expect(sample).toContain('gemini-3-pro-image-4k')
      expect(sample).toContain('prompt')
      expect(sample).toContain('image')
      expect(sample).toContain('reference.png')
      if (language !== 'curl') {
        expect(sample).toContain('b64_json')
      }
      expect(sample).not.toContain('response_format')
      expect(sample).not.toContain('extra_fields')
      expect(sample).not.toContain('aspect_ratio')
      expect(sample).not.toContain('"n"')
      expect(sample).not.toContain('n=')
      expect(sample).not.toContain('quality')
      expect(sample).not.toContain('size')
      if (language === 'curl') {
        expect(sample).toContain('-F "image=@reference.png"')
        expect(sample).not.toContain('Content-Type')
      } else if (language === 'python') {
        expect(sample).toContain('files={')
      } else {
        expect(sample).toContain('FormData')
        expect(sample).toContain('Blob')
      }
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

  it('uses the OpenAI video status name in the TypeScript task type', () => {
    const sample = buildMediaSample('typescript', 'openai-video', VIDEO_CONTEXT)

    expect(sample).toContain("'in_progress'")
    expect(sample).not.toContain("'processing'")
  })
})
