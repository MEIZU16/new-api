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
export const MEDIA_IMAGE_ASPECT_RATIOS = [
  '1:1',
  '16:9',
  '9:16',
  '4:3',
  '3:4',
  '3:2',
  '2:3',
] as const

export type MediaImageResolution = '1K' | '2K' | '4K'

const MEDIA_IMAGE_MODEL_PATTERN =
  /^(?:gemini-3-pro-image|gemini-3\.1-flash-image)(?:-(?:2k|4k))?$/i

/** Return the resolution tier encoded in one of this service's image SKUs. */
export function mediaImageResolution(
  modelName: string
): MediaImageResolution | null {
  if (!MEDIA_IMAGE_MODEL_PATTERN.test(modelName)) return null
  if (/-4k$/i.test(modelName)) return '4K'
  if (/-2k$/i.test(modelName)) return '2K'
  return '1K'
}
