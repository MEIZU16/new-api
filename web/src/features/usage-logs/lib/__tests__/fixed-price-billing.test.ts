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

import { getFixedPriceBillingPresentation } from '../billing'

const translations: Record<string, string> = {
  'Per Second': '按秒计费',
  'Per-call': '每次调用',
  second: '秒',
}
const t = (key: string) => translations[key] ?? key

describe('fixed-price billing presentation', () => {
  it('shows omni-flash as per-second in usage logs', () => {
    expect(getFixedPriceBillingPresentation('omni-flash', '$0.05', t)).toEqual({
      mode: '按秒计费',
      price: '$0.05 / 秒',
      summary: '按秒计费 · $0.05 / 秒',
    })
  })

  it('keeps image models labeled per call', () => {
    expect(
      getFixedPriceBillingPresentation('gemini-3.1-flash-image', '$0.10', t)
    ).toEqual({
      mode: '每次调用',
      price: '$0.10',
      summary: '每次调用 · $0.10',
    })
  })
})
