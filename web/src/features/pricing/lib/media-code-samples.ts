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
import {
  isGeminiImageModel,
  isPromptOnlyImageModel,
  mediaImageResolution,
} from './media-docs'

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

function geminiImageBody(
  ctx: MediaSampleContext,
  prompt: string,
  referenceBase64?: string
) {
  return {
    contents: [
      {
        role: 'user',
        parts: [
          ...(referenceBase64
            ? [
                {
                  inlineData: {
                    mimeType: 'image/png',
                    data: referenceBase64,
                  },
                },
              ]
            : []),
          { text: prompt },
        ],
      },
    ],
    generationConfig: {
      responseModalities: ['IMAGE'],
      imageConfig: {
        aspectRatio: '16:9',
        imageSize: mediaImageResolution(ctx.modelName) ?? '1K',
      },
    },
  }
}

function buildGeminiImageRequestSample(
  lang: MediaSampleLanguage,
  ctx: MediaSampleContext,
  withReference: boolean
): string {
  const url = `${ctx.baseUrl}${ctx.endpointPath}`
  const prompt = withReference
    ? 'Use the reference image composition and render it as a watercolor illustration.'
    : 'A serene koi pond at sunset, rendered as a woodblock print.'
  const outputFilename = withReference
    ? 'edited-image.jpg'
    : 'generated-image.jpg'
  const body = geminiImageBody(
    ctx,
    prompt,
    withReference ? '$REFERENCE_IMAGE_BASE64' : undefined
  )
  const bodyJSON = JSON.stringify(body, null, 2)

  if (lang === 'curl') {
    return [
      ...(withReference
        ? [`REFERENCE_IMAGE_BASE64=$(base64 < reference.png | tr -d '\\n')`, '']
        : []),
      `curl ${url} \\`,
      `  -H "Authorization: Bearer $${ctx.apiKeyEnv}" \\`,
      `  -H "Content-Type: application/json" \\`,
      `  --data-binary @- > response.json <<JSON`,
      bodyJSON,
      'JSON',
      '',
      `jq -r '.candidates[0].content.parts[] | select(.inlineData) | .inlineData.data' response.json \\`,
      `  | base64 --decode > ${outputFilename}`,
    ].join('\n')
  }
  if (lang === 'python') {
    return [
      'from base64 import b64decode, b64encode',
      'from pathlib import Path',
      '',
      'import requests',
      '',
      ...(withReference
        ? [
            'reference_base64 = b64encode(',
            '    Path("reference.png").read_bytes()',
            ').decode("ascii")',
            '',
          ]
        : []),
      'body = {',
      '    "contents": [',
      '        {',
      '            "role": "user",',
      '            "parts": [',
      ...(withReference
        ? [
            '                {',
            '                    "inlineData": {',
            '                        "mimeType": "image/png",',
            '                        "data": reference_base64,',
            '                    }',
            '                },',
          ]
        : []),
      `                {"text": "${prompt}"},`,
      '            ],',
      '        }',
      '    ],',
      '    "generationConfig": {',
      '        "responseModalities": ["IMAGE"],',
      '        "imageConfig": {',
      '            "aspectRatio": "16:9",',
      `            "imageSize": "${mediaImageResolution(ctx.modelName) ?? '1K'}",`,
      '        },',
      '    },',
      '}',
      '',
      'response = requests.post(',
      `    "${url}",`,
      '    headers={',
      '        "Authorization": "Bearer <YOUR_API_KEY>",',
      '        "Content-Type": "application/json",',
      '    },',
      '    json=body,',
      ')',
      'response.raise_for_status()',
      '',
      'parts = response.json()["candidates"][0]["content"]["parts"]',
      'image = next(part["inlineData"] for part in parts if "inlineData" in part)',
      `Path("${outputFilename}").write_bytes(b64decode(image["data"]))`,
    ].join('\n')
  }
  if (lang === 'typescript') {
    return [
      `import { readFile, writeFile } from 'node:fs/promises'`,
      '',
      `const apiKey = process.env.${ctx.apiKeyEnv}`,
      `if (!apiKey) throw new Error('${ctx.apiKeyEnv} is not set')`,
      ...(withReference
        ? [
            `const referenceBase64 = (`,
            `  await readFile('reference.png')`,
            `).toString('base64')`,
          ]
        : []),
      '',
      `const response = await fetch('${url}', {`,
      `  method: 'POST',`,
      `  headers: {`,
      `    Authorization: \`Bearer \${apiKey}\`,`,
      `    'Content-Type': 'application/json',`,
      `  },`,
      `  body: JSON.stringify({`,
      `    contents: [`,
      `      {`,
      `        role: 'user',`,
      `        parts: [`,
      ...(withReference
        ? [
            `          {`,
            `            inlineData: {`,
            `              mimeType: 'image/png',`,
            `              data: referenceBase64,`,
            `            },`,
            `          },`,
          ]
        : []),
      `          { text: '${prompt}' },`,
      `        ],`,
      `      },`,
      `    ],`,
      `    generationConfig: {`,
      `      responseModalities: ['IMAGE'],`,
      `      imageConfig: {`,
      `        aspectRatio: '16:9',`,
      `        imageSize: '${mediaImageResolution(ctx.modelName) ?? '1K'}',`,
      `      },`,
      `    },`,
      `  }),`,
      `})`,
      `if (!response.ok) throw new Error(await response.text())`,
      '',
      `type GeminiPart = { inlineData?: { data: string; mimeType: string } }`,
      `const data = (await response.json()) as {`,
      `  candidates: Array<{ content: { parts: GeminiPart[] } }>`,
      `}`,
      `const image = data.candidates[0].content.parts.find(`,
      `  (part) => part.inlineData`,
      `)?.inlineData`,
      `if (!image) throw new Error('Gemini returned no image')`,
      `await writeFile('${outputFilename}', Buffer.from(image.data, 'base64'))`,
    ].join('\n')
  }
  return [
    `import { readFile, writeFile } from 'node:fs/promises'`,
    ...(withReference
      ? [
          '',
          `const referenceBase64 = (`,
          `  await readFile('reference.png')`,
          `).toString('base64')`,
        ]
      : []),
    '',
    `const response = await fetch('${url}', {`,
    `  method: 'POST',`,
    `  headers: {`,
    `    Authorization: \`Bearer \${process.env.${ctx.apiKeyEnv}}\`,`,
    `    'Content-Type': 'application/json',`,
    `  },`,
    `  body: JSON.stringify({`,
    `    contents: [`,
    `      {`,
    `        role: 'user',`,
    `        parts: [`,
    ...(withReference
      ? [
          `          {`,
          `            inlineData: {`,
          `              mimeType: 'image/png',`,
          `              data: referenceBase64,`,
          `            },`,
          `          },`,
        ]
      : []),
    `          { text: '${prompt}' },`,
    `        ],`,
    `      },`,
    `    ],`,
    `    generationConfig: {`,
    `      responseModalities: ['IMAGE'],`,
    `      imageConfig: {`,
    `        aspectRatio: '16:9',`,
    `        imageSize: '${mediaImageResolution(ctx.modelName) ?? '1K'}',`,
    `      },`,
    `    },`,
    `  }),`,
    `})`,
    `if (!response.ok) throw new Error(await response.text())`,
    '',
    `const data = await response.json()`,
    `const image = data.candidates[0].content.parts.find(`,
    `  (part) => part.inlineData`,
    `)?.inlineData`,
    `if (!image) throw new Error('Gemini returned no image')`,
    `await writeFile('${outputFilename}', Buffer.from(image.data, 'base64'))`,
  ].join('\n')
}

export function buildGeminiImageSample(
  lang: MediaSampleLanguage,
  ctx: MediaSampleContext
): string {
  return buildGeminiImageRequestSample(lang, ctx, false)
}

export function buildGeminiImageReferenceSample(
  lang: MediaSampleLanguage,
  ctx: MediaSampleContext
): string {
  return buildGeminiImageRequestSample(lang, ctx, true)
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
      'Path("edited-image.png").write_bytes(',
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
      `  'edited-image.png',`,
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
    `    Authorization: \`Bearer \${process.env.${ctx.apiKeyEnv}}\`,`,
    `  },`,
    `  body: form,`,
    `})`,
    `if (!response.ok) throw new Error(await response.text())`,
    `const data = await response.json()`,
    `await writeFile(`,
    `  'edited-image.png',`,
    `  Buffer.from(data.data[0].b64_json, 'base64')`,
    `)`,
  ].join('\n')
}

export function buildVideoSample(
  lang: MediaSampleLanguage,
  ctx: MediaSampleContext
): string {
  const url = `${ctx.baseUrl}${ctx.endpointPath}`
  const prompt = 'A paper boat sailing through a neon city in the rain.'
  const body = JSON.stringify(
    {
      model: ctx.modelName,
      prompt,
      seconds: '6',
      size: '1280x720',
    },
    null,
    2
  )

  if (lang === 'curl') {
    return [
      `VIDEO=$(curl --fail-with-body --silent --show-error ${url} \\`,
      `  -H "Authorization: Bearer $${ctx.apiKeyEnv}" \\`,
      `  -H "Content-Type: application/json" \\`,
      `  -d '${body.replaceAll('\n', '\n     ')}')`,
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
    return [
      'import time',
      'from pathlib import Path',
      '',
      'import requests',
      '',
      `videos_url = "${url}"`,
      'headers = {"Authorization": "Bearer <YOUR_API_KEY>"}',
      '',
      'response = requests.post(',
      '    videos_url,',
      '    headers=headers,',
      '    json={',
      `        "model": "${ctx.modelName}",`,
      `        "prompt": "${prompt}",`,
      '        "seconds": "6",',
      '        "size": "1280x720",',
      '    },',
      ')',
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
  if (lang === 'typescript') {
    return [
      `import { writeFile } from 'node:fs/promises'`,
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
      `const headers = {`,
      `  Authorization: \`Bearer \${apiKey}\`,`,
      `  'Content-Type': 'application/json',`,
      `}`,
      '',
      `let response = await fetch(videosUrl, {`,
      `  method: 'POST',`,
      `  headers,`,
      `  body: JSON.stringify(${body}),`,
      `})`,
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
    `import { writeFile } from 'node:fs/promises'`,
    '',
    `const videosUrl = '${url}'`,
    `const headers = {`,
    `  Authorization: \`Bearer \${process.env.${ctx.apiKeyEnv}}\`,`,
    `  'Content-Type': 'application/json',`,
    `}`,
    '',
    `let response = await fetch(videosUrl, {`,
    `  method: 'POST',`,
    `  headers,`,
    `  body: JSON.stringify(${body}),`,
    `})`,
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
  if (endpointType === 'gemini' && isGeminiImageModel(ctx.modelName)) {
    return buildGeminiImageSample(lang, ctx)
  }
  if (endpointType === 'image-generation') return buildImageSample(lang, ctx)
  if (endpointType === 'openai-video') return buildVideoSample(lang, ctx)
  return null
}
