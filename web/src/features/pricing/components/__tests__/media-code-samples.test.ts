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
  buildVideoSample,
} from '../../lib/media-code-samples'

const IMAGE_CONTEXT = {
  baseUrl: 'https://api.example.com',
  apiKeyEnv: 'NEW_API_KEY',
  modelName: 'gemini-3-pro-image-4k',
  endpointType: 'image-generation',
  endpointPath: '/v1/images/generations',
}

const GPT_IMAGE_CONTEXT = {
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
      expect(sample).not.toContain('generationConfig')
      expect(sample).not.toContain('inlineData')
      expect(sample).not.toContain('imageSize')
      if (language !== 'curl') {
        expect(sample).toContain('b64_json')
      }
    }
  )

  it.each(['curl', 'python', 'typescript', 'javascript'] as const)(
    'uses OpenAI multipart edits for Gemini reference images in the %s sample',
    (language) => {
      const sample = buildImageEditSample(language, IMAGE_CONTEXT)

      expect(sample).toContain('/v1/images/edits')
      expect(sample).toContain('gemini-3-pro-image-4k')
      expect(sample).toContain('reference.png')
      expect(sample).toContain('image')
      expect(sample).not.toContain('generationConfig')
      expect(sample).not.toContain('inlineData')
      expect(sample).not.toContain('imageSize')
      expect(sample).not.toContain('response_format')
      expect(sample).not.toContain('quality')
      expect(sample).not.toContain('size')
      if (language !== 'curl') {
        expect(sample).toContain('b64_json')
      }
    }
  )

  it.each(['curl', 'python', 'typescript', 'javascript'] as const)(
    'shows only model and prompt in the %s gpt-image-2 request',
    (language) => {
      const sample = buildImageSample(language, GPT_IMAGE_CONTEXT)

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
    'uses OpenAI multipart edits for gpt-image-2 in the %s sample',
    (language) => {
      const sample = buildImageEditSample(language, GPT_IMAGE_CONTEXT)

      expect(sample).toContain('/v1/images/edits')
      expect(sample).toContain('gpt-image-2')
      expect(sample).toContain('reference.png')
      expect(sample).toContain('image')
      expect(sample).not.toContain('extra_fields')
      expect(sample).not.toContain('aspect_ratio')
      expect(sample).not.toContain('response_format')
      if (language !== 'curl') {
        expect(sample).toContain('edited-image.png')
        expect(sample).toContain('b64_json')
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

  it.each(['curl', 'python', 'typescript', 'javascript'] as const)(
    'keeps the default %s video sample free of reference fields',
    (language) => {
      const sample = buildMediaSample(language, 'openai-video', VIDEO_CONTEXT)

      expect(sample).not.toContain('input_reference')
      // Word-bounded so the `model` field does not count as a `mode` field.
      expect(sample).not.toMatch(/\bmode\b/)
    }
  )

  it('uses the OpenAI video status name in the TypeScript task type', () => {
    const sample = buildMediaSample('typescript', 'openai-video', VIDEO_CONTEXT)

    expect(sample).toContain("'in_progress'")
    expect(sample).not.toContain("'processing'")
  })
})

describe('video reference-mode samples', () => {
  const LANGUAGES = ['curl', 'python', 'typescript', 'javascript'] as const

  it.each(LANGUAGES)(
    'sends a single named reference in the %s image-to-video sample',
    (language) => {
      const sample = buildVideoSample(language, VIDEO_CONTEXT, 'i2v')

      expect(sample).toContain('input_reference')
      expect(sample).toContain('reference.png')
      expect(sample).toContain('i2v')
      expect(sample).not.toContain('reference-2.png')
    }
  )

  it.each(LANGUAGES)(
    'carries every reference image of the %s multi-reference sample',
    (language) => {
      const sample = buildVideoSample(language, VIDEO_CONTEXT, 'r2v')

      expect(sample).toContain('reference-1.png')
      expect(sample).toContain('reference-2.png')
      expect(sample).toContain('reference-3.png')
      expect(sample).toContain('r2v')
    }
  )

  // The create call switches to multipart once a reference is attached; a
  // leftover JSON content type would make the copied sample fail upstream.
  it.each([
    ['i2v', 'curl'],
    ['i2v', 'python'],
    ['i2v', 'typescript'],
    ['i2v', 'javascript'],
    ['r2v', 'curl'],
    ['r2v', 'python'],
    ['r2v', 'typescript'],
    ['r2v', 'javascript'],
  ] as const)(
    'drops the JSON content type from the %s %s sample',
    (mode, language) => {
      const sample = buildVideoSample(language, VIDEO_CONTEXT, mode)

      expect(sample).not.toContain('application/json')
    }
  )

  it.each(LANGUAGES)(
    'keeps create, poll, and download in the %s multi-reference sample',
    (language) => {
      const sample = buildVideoSample(language, VIDEO_CONTEXT, 'r2v')

      expect(sample).toContain('/v1/videos')
      expect(sample).toContain('completed')
      expect(sample).toContain('/content')
    }
  )

  // Reference order and count survive only when the part is repeated, so each
  // dialect is checked against the mechanism that actually repeats it.
  it('repeats the curl reference part once per image', () => {
    const sample = buildVideoSample('curl', VIDEO_CONTEXT, 'r2v')

    expect(sample.match(/-F "input_reference=@/g)).toHaveLength(3)
  })

  it('repeats the python reference tuple once per image', () => {
    const sample = buildVideoSample('python', VIDEO_CONTEXT, 'r2v')

    expect(sample.match(/\("input_reference", \(/g)).toHaveLength(3)
  })

  it.each(['typescript', 'javascript'] as const)(
    'appends instead of overwriting the %s form reference part',
    (language) => {
      const sample = buildVideoSample(language, VIDEO_CONTEXT, 'r2v')

      expect(sample).toContain(`form.append(`)
      expect(sample).not.toContain(`form.set('input_reference'`)
    }
  )
})
