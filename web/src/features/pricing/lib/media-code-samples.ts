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
import { isPromptOnlyImageModel } from './media-docs'

export type MediaSampleLanguage =
  | 'curl'
  | 'python'
  | 'typescript'
  | 'javascript'

export type MediaSampleContext = {
  baseUrl: string
  apiKeyEnv: string
  modelName: string
  endpointPath: string
}

export function buildImageSample(
  lang: MediaSampleLanguage,
  ctx: MediaSampleContext
): string {
  const url = `${ctx.baseUrl}${ctx.endpointPath}`
  const prompt = 'A serene koi pond at sunset, rendered as a woodblock print.'
  const promptOnly = isPromptOnlyImageModel(ctx.modelName)
  const outputFilename = promptOnly
    ? 'generated-image.png'
    : 'generated-image.jpg'
  const body = JSON.stringify(
    {
      model: ctx.modelName,
      prompt,
      ...(promptOnly ? {} : { extra_fields: { aspect_ratio: '16:9' } }),
    },
    null,
    2
  )

  if (lang === 'curl') {
    return [
      `curl ${url} \\`,
      `  -H "Authorization: Bearer $${ctx.apiKeyEnv}" \\`,
      `  -H "Content-Type: application/json" \\`,
      `  -d '${body.replaceAll('\n', '\n     ')}'`,
    ].join('\n')
  }
  if (lang === 'python') {
    return [
      'from base64 import b64decode',
      'from pathlib import Path',
      '',
      'from openai import OpenAI',
      '',
      `client = OpenAI(base_url="${ctx.baseUrl}/v1", api_key="<YOUR_API_KEY>")`,
      '',
      'response = client.images.generate(',
      `    model="${ctx.modelName}",`,
      `    prompt="${prompt}",`,
      ...(promptOnly
        ? []
        : [
            '    extra_body={',
            '        "extra_fields": {"aspect_ratio": "16:9"},',
            '    },',
          ]),
      ')',
      '',
      `Path("${outputFilename}").write_bytes(`,
      '    b64decode(response.data[0].b64_json)',
      ')',
    ].join('\n')
  }
  if (lang === 'typescript') {
    return [
      `import { writeFile } from 'node:fs/promises'`,
      '',
      `const apiKey = process.env.${ctx.apiKeyEnv}`,
      `if (!apiKey) throw new Error('${ctx.apiKeyEnv} is not set')`,
      '',
      `const response = await fetch('${url}', {`,
      `  method: 'POST',`,
      `  headers: {`,
      `    Authorization: \`Bearer \${apiKey}\`,`,
      `    'Content-Type': 'application/json',`,
      `  },`,
      `  body: JSON.stringify(${body}),`,
      `})`,
      `if (!response.ok) throw new Error(await response.text())`,
      '',
      `const data = (await response.json()) as {`,
      `  data: Array<{ b64_json: string }>`,
      `}`,
      `await writeFile(`,
      `  '${outputFilename}',`,
      `  Buffer.from(data.data[0].b64_json, 'base64')`,
      `)`,
    ].join('\n')
  }
  return [
    `import { writeFile } from 'node:fs/promises'`,
    '',
    `const response = await fetch('${url}', {`,
    `  method: 'POST',`,
    `  headers: {`,
    `    Authorization: \`Bearer \${process.env.${ctx.apiKeyEnv}}\`,`,
    `    'Content-Type': 'application/json',`,
    `  },`,
    `  body: JSON.stringify(${body}),`,
    `})`,
    `if (!response.ok) throw new Error(await response.text())`,
    '',
    `const data = await response.json()`,
    `await writeFile(`,
    `  '${outputFilename}',`,
    `  Buffer.from(data.data[0].b64_json, 'base64')`,
    `)`,
  ].join('\n')
}

export function buildImageEditSample(
  lang: MediaSampleLanguage,
  ctx: MediaSampleContext
): string {
  const url = `${ctx.baseUrl}${ctx.endpointPath.replace(
    /\/images\/generations\/?$/,
    '/images/edits'
  )}`
  const prompt =
    'Use the reference image composition and render it as a watercolor illustration.'
  const outputFilename = isPromptOnlyImageModel(ctx.modelName)
    ? 'edited-image.png'
    : 'edited-image.jpg'

  if (lang === 'curl') {
    return [
      `curl ${url} \\`,
      `  -H "Authorization: Bearer $${ctx.apiKeyEnv}" \\`,
      `  -F "model=${ctx.modelName}" \\`,
      `  -F "prompt=${prompt}" \\`,
      `  -F "image=@reference.png"`,
    ].join('\n')
  }
  if (lang === 'python') {
    return [
      'from base64 import b64decode',
      'from pathlib import Path',
      '',
      'import requests',
      '',
      'with Path("reference.png").open("rb") as reference_image:',
      '    response = requests.post(',
      `        "${url}",`,
      '        headers={"Authorization": "Bearer <YOUR_API_KEY>"},',
      '        data={',
      `            "model": "${ctx.modelName}",`,
      `            "prompt": "${prompt}",`,
      '        },',
      '        files={',
      '            "image": ("reference.png", reference_image, "image/png"),',
      '        },',
      '    )',
      '',
      'response.raise_for_status()',
      `Path("${outputFilename}").write_bytes(`,
      '    b64decode(response.json()["data"][0]["b64_json"])',
      ')',
    ].join('\n')
  }
  if (lang === 'typescript') {
    return [
      `import { readFile, writeFile } from 'node:fs/promises'`,
      '',
      `const apiKey = process.env.${ctx.apiKeyEnv}`,
      `if (!apiKey) throw new Error('${ctx.apiKeyEnv} is not set')`,
      `const referenceImage = await readFile('reference.png')`,
      `const form = new FormData()`,
      `form.set('model', '${ctx.modelName}')`,
      `form.set('prompt', '${prompt}')`,
      `form.set(`,
      `  'image',`,
      `  new Blob([referenceImage], { type: 'image/png' }),`,
      `  'reference.png'`,
      `)`,
      '',
      `const response = await fetch('${url}', {`,
      `  method: 'POST',`,
      `  headers: { Authorization: \`Bearer \${apiKey}\` },`,
      `  body: form,`,
      `})`,
      `if (!response.ok) throw new Error(await response.text())`,
      `const data = (await response.json()) as {`,
      `  data: Array<{ b64_json: string }>`,
      `}`,
      `await writeFile(`,
      `  '${outputFilename}',`,
      `  Buffer.from(data.data[0].b64_json, 'base64')`,
      `)`,
    ].join('\n')
  }
  return [
    `import { readFile, writeFile } from 'node:fs/promises'`,
    '',
    `const referenceImage = await readFile('reference.png')`,
    `const form = new FormData()`,
    `form.set('model', '${ctx.modelName}')`,
    `form.set('prompt', '${prompt}')`,
    `form.set(`,
    `  'image',`,
    `  new Blob([referenceImage], { type: 'image/png' }),`,
    `  'reference.png'`,
    `)`,
    '',
    `const response = await fetch('${url}', {`,
    `  method: 'POST',`,
    `  headers: {`,
    `    Authorization: \`Bearer \${process.env.${ctx.apiKeyEnv}}\``,
    `  },`,
    `  body: form,`,
    `})`,
    `if (!response.ok) throw new Error(await response.text())`,
    `const data = await response.json()`,
    `await writeFile(`,
    `  '${outputFilename}',`,
    `  Buffer.from(data.data[0].b64_json, 'base64')`,
    `)`,
  ].join('\n')
}

/**
 * The three video generation modes the public documentation covers. The
 * upstream mode is otherwise inferred from how many reference images arrive,
 * so each sample names it explicitly instead of relying on that inference.
 */
export type VideoSampleMode = 't2v' | 'i2v' | 'r2v'

/**
 * Prompt and reference filenames per mode. Only the create request differs
 * between modes; polling and downloading are identical, so the samples share
 * that half rather than repeating it three times per language.
 */
const VIDEO_SAMPLE_SHAPES: Record<
  VideoSampleMode,
  { prompt: string; references: string[] }
> = {
  t2v: {
    prompt: 'A paper boat sailing through a neon city in the rain.',
    references: [],
  },
  i2v: {
    prompt: 'Animate the reference image with a slow push-in as rain starts.',
    references: ['reference.png'],
  },
  r2v: {
    prompt: 'Bring every referenced subject together in one neon street scene.',
    references: ['reference-1.png', 'reference-2.png', 'reference-3.png'],
  },
}

const VIDEO_SAMPLE_SECONDS = '6'
const VIDEO_SAMPLE_SIZE = '1280x720'

export function buildVideoSample(
  lang: MediaSampleLanguage,
  ctx: MediaSampleContext,
  mode: VideoSampleMode = 't2v'
): string {
  const url = `${ctx.baseUrl}${ctx.endpointPath}`
  const shape = VIDEO_SAMPLE_SHAPES[mode]
  const references = shape.references
  const isMultipart = references.length > 0
  const body = JSON.stringify(
    {
      model: ctx.modelName,
      prompt: shape.prompt,
      seconds: VIDEO_SAMPLE_SECONDS,
      size: VIDEO_SAMPLE_SIZE,
    },
    null,
    2
  )

  if (lang === 'curl') {
    const jsonArgs = [
      `  -H "Authorization: Bearer $${ctx.apiKeyEnv}"`,
      `  -H "Content-Type: application/json"`,
      `  -d '${body.replaceAll('\n', '\n     ')}'`,
    ]
    const multipartArgs = [
      `  -H "Authorization: Bearer $${ctx.apiKeyEnv}"`,
      `  -F "model=${ctx.modelName}"`,
      `  -F "prompt=${shape.prompt}"`,
      `  -F "seconds=${VIDEO_SAMPLE_SECONDS}"`,
      `  -F "size=${VIDEO_SAMPLE_SIZE}"`,
      `  -F "mode=${mode}"`,
      // Repeating the field is what carries an ordered reference set; an
      // indexed or comma-joined field would collapse to a single reference.
      ...references.map((name) => `  -F "input_reference=@${name}"`),
    ]
    const createArgs = isMultipart ? multipartArgs : jsonArgs
    const createCommand = [
      `VIDEO=$(curl --fail-with-body --silent --show-error ${url}`,
      ...createArgs,
    ].join(' \\\n')
    return [
      `${createCommand})`,
      `printf '%s\\n' "$VIDEO" | jq`,
      `VIDEO_ID=$(printf '%s' "$VIDEO" | jq -r '.id')`,
      '',
      'while true; do',
      `  VIDEO=$(curl --fail-with-body --silent --show-error \\`,
      `    -H "Authorization: Bearer $${ctx.apiKeyEnv}" \\`,
      `    "${url}/$VIDEO_ID")`,
      `  STATUS=$(printf '%s' "$VIDEO" | jq -r '.status')`,
      `  printf '%s\\n' "$VIDEO" | jq`,
      '  case "$STATUS" in',
      '    completed) break ;;',
      '    failed) exit 1 ;;',
      '  esac',
      '  sleep 5',
      'done',
      '',
      `curl --fail-with-body --silent --show-error \\`,
      `  -H "Authorization: Bearer $${ctx.apiKeyEnv}" \\`,
      `  "${url}/$VIDEO_ID/content" \\`,
      `  -o generated-video.mp4`,
    ].join('\n')
  }
  if (lang === 'python') {
    const jsonCreate = [
      'response = requests.post(',
      '    videos_url,',
      '    headers=headers,',
      '    json={',
      `        "model": "${ctx.modelName}",`,
      `        "prompt": "${shape.prompt}",`,
      `        "seconds": "${VIDEO_SAMPLE_SECONDS}",`,
      `        "size": "${VIDEO_SAMPLE_SIZE}",`,
      '    },',
      ')',
    ]
    const multipartCreate = [
      'references = [',
      ...references.map(
        (name) =>
          `    ("input_reference", ("${name}", Path("${name}").read_bytes(), "image/png")),`
      ),
      ']',
      '',
      'response = requests.post(',
      '    videos_url,',
      '    headers=headers,',
      '    data={',
      `        "model": "${ctx.modelName}",`,
      `        "prompt": "${shape.prompt}",`,
      `        "seconds": "${VIDEO_SAMPLE_SECONDS}",`,
      `        "size": "${VIDEO_SAMPLE_SIZE}",`,
      `        "mode": "${mode}",`,
      '    },',
      '    files=references,',
      ')',
    ]
    return [
      'import time',
      'from pathlib import Path',
      '',
      'import requests',
      '',
      `videos_url = "${url}"`,
      'headers = {"Authorization": "Bearer <YOUR_API_KEY>"}',
      '',
      ...(isMultipart ? multipartCreate : jsonCreate),
      'response.raise_for_status()',
      'video = response.json()',
      '',
      'while video["status"] not in {"completed", "failed"}:',
      '    time.sleep(5)',
      '    response = requests.get(',
      '        f"{videos_url}/{video[\'id\']}", headers=headers',
      '    )',
      '    response.raise_for_status()',
      '    video = response.json()',
      '',
      'if video["status"] == "failed":',
      '    raise RuntimeError(video.get("error", "Video generation failed"))',
      '',
      'response = requests.get(',
      '    f"{videos_url}/{video[\'id\']}/content", headers=headers',
      ')',
      'response.raise_for_status()',
      'Path("generated-video.mp4").write_bytes(response.content)',
    ].join('\n')
  }

  const fsImport = isMultipart
    ? `import { readFile, writeFile } from 'node:fs/promises'`
    : `import { writeFile } from 'node:fs/promises'`
  const jsonCreate = [
    `let response = await fetch(videosUrl, {`,
    `  method: 'POST',`,
    `  headers: { ...headers, 'Content-Type': 'application/json' },`,
    `  body: JSON.stringify(${body}),`,
    `})`,
  ]
  const multipartCreate = [
    `const form = new FormData()`,
    `form.set('model', '${ctx.modelName}')`,
    `form.set('prompt', '${shape.prompt}')`,
    `form.set('seconds', '${VIDEO_SAMPLE_SECONDS}')`,
    `form.set('size', '${VIDEO_SAMPLE_SIZE}')`,
    `form.set('mode', '${mode}')`,
    // append, not set: repeated parts are what make an ordered reference set.
    `for (const name of [${references.map((n) => `'${n}'`).join(', ')}]) {`,
    `  form.append(`,
    `    'input_reference',`,
    `    new Blob([await readFile(name)], { type: 'image/png' }),`,
    `    name`,
    `  )`,
    `}`,
    '',
    `let response = await fetch(videosUrl, {`,
    `  method: 'POST',`,
    `  headers,`,
    `  body: form,`,
    `})`,
  ]
  const createBlock = isMultipart ? multipartCreate : jsonCreate

  if (lang === 'typescript') {
    return [
      fsImport,
      '',
      `type VideoTask = {`,
      `  id: string`,
      `  status: 'queued' | 'in_progress' | 'completed' | 'failed'`,
      `  error?: { message?: string }`,
      `}`,
      '',
      `const apiKey = process.env.${ctx.apiKeyEnv}`,
      `if (!apiKey) throw new Error('${ctx.apiKeyEnv} is not set')`,
      `const videosUrl = '${url}'`,
      `const headers = { Authorization: \`Bearer \${apiKey}\` }`,
      '',
      ...createBlock,
      `if (!response.ok) throw new Error(await response.text())`,
      `let video = (await response.json()) as VideoTask`,
      '',
      `while (video.status !== 'completed' && video.status !== 'failed') {`,
      `  await new Promise((resolve) => setTimeout(resolve, 5_000))`,
      `  response = await fetch(\`${url}/\${video.id}\`, { headers })`,
      `  if (!response.ok) throw new Error(await response.text())`,
      `  video = (await response.json()) as VideoTask`,
      `}`,
      '',
      `if (video.status === 'failed') {`,
      `  throw new Error(video.error?.message ?? 'Video generation failed')`,
      `}`,
      '',
      `response = await fetch(\`${url}/\${video.id}/content\`, { headers })`,
      `if (!response.ok) throw new Error(await response.text())`,
      `await writeFile(`,
      `  'generated-video.mp4',`,
      `  Buffer.from(await response.arrayBuffer())`,
      `)`,
    ].join('\n')
  }
  return [
    fsImport,
    '',
    `const videosUrl = '${url}'`,
    `const headers = {`,
    `  Authorization: \`Bearer \${process.env.${ctx.apiKeyEnv}}\`,`,
    `}`,
    '',
    ...createBlock,
    `if (!response.ok) throw new Error(await response.text())`,
    `let video = await response.json()`,
    '',
    `while (video.status !== 'completed' && video.status !== 'failed') {`,
    `  await new Promise((resolve) => setTimeout(resolve, 5_000))`,
    `  response = await fetch(\`${url}/\${video.id}\`, { headers })`,
    `  if (!response.ok) throw new Error(await response.text())`,
    `  video = await response.json()`,
    `}`,
    '',
    `if (video.status === 'failed') {`,
    `  throw new Error(video.error?.message ?? 'Video generation failed')`,
    `}`,
    '',
    `response = await fetch(\`${url}/\${video.id}/content\`, { headers })`,
    `if (!response.ok) throw new Error(await response.text())`,
    `await writeFile(`,
    `  'generated-video.mp4',`,
    `  Buffer.from(await response.arrayBuffer())`,
    `)`,
  ].join('\n')
}

export function buildMediaSample(
  lang: MediaSampleLanguage,
  endpointType: string,
  ctx: MediaSampleContext
): string | null {
  if (endpointType === 'image-generation') return buildImageSample(lang, ctx)
  if (endpointType === 'openai-video') return buildVideoSample(lang, ctx)
  return null
}
