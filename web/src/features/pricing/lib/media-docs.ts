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
export { isPerSecondMediaModel } from '@/lib/media-billing'

export type MediaImageResolution = '1K' | '2K' | '4K'

const MEDIA_IMAGE_MODEL_PATTERN =
  /^(?:gemini-3-pro-image|gemini-3\.1-flash-image)(?:-(?:2k|4k))?$/i
const PROMPT_ONLY_IMAGE_MODELS = new Set(['gpt-image-2'])
const REFERENCE_IMAGE_EDIT_MODELS = new Set(['gpt-image-2'])
const REFERENCE_VIDEO_MODELS = new Set(['omni-flash'])
/** Reference images a single multi-reference video request may carry. */
export const MAX_VIDEO_REFERENCE_IMAGES = 7
const PRO_IMAGE_ASPECT_RATIOS = [
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
] as const

const FLASH_IMAGE_ASPECT_RATIOS = [
  ...PRO_IMAGE_ASPECT_RATIOS,
  '4:1',
  '1:4',
  '8:1',
  '1:8',
] as const

/** Return whether the model is one of this service's Gemini image SKUs. */
export function isGeminiImageModel(modelName: string): boolean {
  return MEDIA_IMAGE_MODEL_PATTERN.test(modelName.trim())
}

/** Return whether generation documentation should expose only the image prompt. */
export function isPromptOnlyImageModel(modelName: string): boolean {
  return PROMPT_ONLY_IMAGE_MODELS.has(modelName.trim().toLowerCase())
}

/** Return whether public documentation should expose multipart image edits. */
export function supportsReferenceImageEditing(modelName: string): boolean {
  return (
    isGeminiImageModel(modelName) ||
    REFERENCE_IMAGE_EDIT_MODELS.has(modelName.trim().toLowerCase())
  )
}

/**
 * Return whether the video SKU also accepts reference images, so its
 * documentation covers image-to-video and multi-reference video next to the
 * plain text-to-video request.
 */
export function supportsVideoReferenceModes(modelName: string): boolean {
  return REFERENCE_VIDEO_MODELS.has(modelName.trim().toLowerCase())
}

/** Return the resolution tier encoded in one of this service's image SKUs. */
export function mediaImageResolution(
  modelName: string
): MediaImageResolution | null {
  if (!MEDIA_IMAGE_MODEL_PATTERN.test(modelName)) return null
  if (/-4k$/i.test(modelName)) return '4K'
  if (/-2k$/i.test(modelName)) return '2K'
  return '1K'
}

/** Ratios currently offered by each model's AI Studio web control. */
export function mediaImageAspectRatios(
  modelName: string
): readonly string[] | null {
  const baseModel = modelName.toLowerCase().replace(/-(?:2k|4k)$/, '')
  if (baseModel === 'gemini-3-pro-image') return PRO_IMAGE_ASPECT_RATIOS
  if (baseModel === 'gemini-3.1-flash-image') {
    return FLASH_IMAGE_ASPECT_RATIOS
  }
  return null
}
