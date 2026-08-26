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
import { isPerSecondMediaModel } from '@/lib/media-billing'

/** Keep fixed-price log labels aligned with the public SKU's billing unit. */
export function getFixedPriceBillingPresentation(
  modelName: string,
  formattedPrice: string,
  t: (key: string) => string
): { mode: string; price: string; summary: string } {
  const isPerSecond = isPerSecondMediaModel(modelName)
  const mode = t(isPerSecond ? 'Per Second' : 'Per-call')
  const price = isPerSecond
    ? `${formattedPrice} / ${t('second')}`
    : formattedPrice

  return {
    mode,
    price,
    summary: `${mode} · ${price}`,
  }
}
