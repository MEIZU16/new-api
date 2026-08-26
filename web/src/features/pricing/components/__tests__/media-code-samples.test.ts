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
  buildGeminiImageReferenceSample,
  buildImageEditSample,
  buildImageSample,
  buildMediaSample,
} from '../../lib/media-code-samples'

const GEMINI_IMAGE_CONTEXT = {
  baseUrl: 'https://api.example.com',
  apiKeyEnv: 'NEW_API_KEY',
  modelName: 'gemini-3-pro-image-4k',
  endpointType: 'gemini',
  endpointPath: '/v1beta/models/gemini-3-pro-image-4k:generateContent',
}

const PROMPT_ONLY_IMAGE_CONTEXT = {
  baseUrl: 'https://api.example.com',
  apiKeyEnv: 'NEW_API_KEY',
  modelName: 'gpt-image-2',
  endpointType: 'image-generation',
  endpointPath: '/v1/images/generations',
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
    'uses native Gemini image generation in the %s sample',
    (language) => {
      const sample = buildMediaSample(language, 'gemini', GEMINI_IMAGE_CONTEXT)

      expect(sample).not.toBeNull()
      expect(sample).toContain(
        '/v1beta/models/gemini-3-pro-image-4k:generateContent'
      )
      expect(sample).toContain('generationConfig')
      expect(sample).toContain('responseModalities')
      expect(sample).toContain('imageConfig')
      expect(sample).toContain('aspectRatio')
      expect(sample).toContain('imageSize')
      expect(sample).toContain('4K')
      expect(sample).toContain('generated-image.jpg')
      expect(sample).toContain('inlineData')
      expect(sample).not.toContain('/v1/images/')
      expect(sample).not.toContain('extra_fields')
      expect(sample).not.toContain('response_format')
      expect(sample).not.toContain('b64_json')
      expect(sample).not.toContain('REFERENCE_IMAGE_BASE64')
    }
  )

  it.each(['curl', 'python', 'typescript', 'javascript'] as const)(
    'uses Gemini inlineData for reference images in the %s sample',
    (language) => {
      const sample = buildGeminiImageReferenceSample(
        language,
        GEMINI_IMAGE_CONTEXT
      )

      expect(sample).toContain(':generateContent')
      expect(sample).toContain('inlineData')
      expect(sample).toContain('mimeType')
      expect(sample).toContain('image/png')
      expect(sample).toContain('reference.png')
      expect(sample).toContain('edited-image.jpg')
      expect(sample).toContain('4K')
      expect(sample).not.toContain('/v1/images/edits')
      expect(sample).not.toContain('multipart/form-data')
      expect(sample).not.toContain('response_format')
      expect(sample).not.toContain('b64_json')
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
    'uses the provider-native image edit endpoint in the %s gpt-image-2 sample',
    (language) => {
      const sample = buildImageEditSample(language, PROMPT_ONLY_IMAGE_CONTEXT)

      expect(sample).toContain('/v1/images/edits')
      expect(sample).toContain('gpt-image-2')
      expect(sample).toContain('reference.png')
      expect(sample).not.toContain('generationConfig')
      expect(sample).not.toContain('inlineData')
      expect(sample).not.toContain('response_format')
      expect(sample).not.toContain('quality')
      expect(sample).not.toContain('size')
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
